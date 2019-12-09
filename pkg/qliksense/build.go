package qliksense

import (
	"bufio"
	"strconv"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/gofrs/flock"
	"github.com/pkg/errors"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/releaseutil"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/strvals"
)

// This is an example. Replace thie following with whatever steps are needed to
// install required components into

const (
	dockerfileLines = `
RUN echo "deb http://deb.debian.org/debian stretch-backports main" >> /etc/apt/sources.list && \
    apt-get update && \
    apt-get install libgpgme11-dev libassuan-dev libbtrfs-dev libdevmapper-dev -y && \
    rm -rf /var/lib/apt/lists/*
COPY --from=qlik/qliksense-cloud-tools:latest /usr/local/bin /usr/local/bin
COPY --from=qlik/qliksense-cloud-tools:latest /root/.config/kustomize /root/.config/kustomize
COPY --from=qlik/qliksense-cloud-tools:latest /usr/local/bin/skopeo /usr/local/bin
COPY --from=qlik/qliksense-operator:latest /usr/local/bin/qliksense-operator /usr/local/bin

`
	stableRepoName = "stable"
	stableRepoURL  = "https://kubernetes-charts.storage.googleapis.com"
	qlikRepoName   = "qlik"
	qlikRepoURL    = "https://qlik.bintray.com/edge"
	chartName      = "qliksense"
	releaseName    = "release-name"
	namespace      = "qliksense"
	sets           = "devMode.enabled=true,engine.acceptEULA=\"yes\""
	helmHomePrefix = "helmHome"
	chartCache     = ".chartcache"
	qlikRegsitry   = "QLIK_REGISTRY"
	qseokVersion   = "QSEOK_VERSION"
	airGapped      = "AIR_GAPPED"
	porterFile     = "porter.yaml"
)

var (
	settings       *cli.EnvSettings
	publicRegistry = "qlik-docker-qsefe.bintray.io"
	helmDir        = filepath.Join("helm", "repository")
	versionFile    = filepath.Join("transformers", "qseokversion.yaml")
)

type porterYaml struct {
	Dockerfile string `yaml:"dockerfile"`
}

type patch struct {
	Target struct {
		Kind          string `yaml:"kind"`
		LabelSelector string `yaml:"labelSelector"`
	} `yaml:"target"`
	Patch string `yaml:"patch"`
}
type selectivePatch struct {
	APIVersion string `yaml:"apiVersion"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Enabled bool    `yaml:"enabled"`
	Patches []patch `yaml:"patches"`
}

type helmChart struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	ReleaseNamespace string `yaml:"releaseNamespace"`
	ChartHome        string `yaml:"chartHome"`
	ChartRepo        string `yaml:"chartRepo"`
	ChartName        string `yaml:"chartName"`
	ChartVersion     string `yaml:"chartVersion"`
}

// Build will generate the necessary Dockerfile lines
// for an invocation image using this mixin
func (m *Mixin) Build() error {
	var (
		version, imagesFile, chartFile, image, scannedText string
		err                                   error
		createFile                            bool
		file,porterDockerFile                 *os.File
		scanner                               *bufio.Scanner
		parts                                 []string
		porterBytes                           []byte
		porterFileYaml                        porterYaml
		pullImages							  = true
	)
	if  porterBytes, err = ioutil.ReadFile(porterFile); err != nil {
		return err
	}


	if err = yaml.Unmarshal(porterBytes, &porterFileYaml); err != nil {
		return err
	}

	if len(porterFileYaml.Dockerfile) > 0 {
		if porterDockerFile, err = os.Open(porterFileYaml.Dockerfile); err != nil {
			return err
		}
		defer porterDockerFile.Close()

		scanner = bufio.NewScanner(porterDockerFile)

		for scanner.Scan() {
			scannedText = scanner.Text()
			if strings.Contains(scannedText, qlikRegsitry) {
				parts = strings.Split(scannedText, "=")
				if len(parts) > 1 {
					publicRegistry = parts[len(parts)-1]
				}
			}
			if strings.Contains(scannedText, qseokVersion) {
				parts = strings.Split(scannedText, "=")
				if len(parts) > 1 {
					version = parts[len(parts)-1]
				}
			}
			if strings.Contains(scanner.Text(), airGapped) {
				parts = strings.Split(scannedText, "=")
				if len(parts) > 1 {
					pullImages,_= strconv.ParseBool(parts[len(parts)-1])
				}
			}
		}
		if err = scanner.Err(); err != nil {
			return err
		}
	}

	fmt.Fprintf(m.Out, dockerfileLines)
	if len(version) == 0 {
		version, _ = getTransformerVersion()
	}
	if len(version) > 0 {
		imagesFile = filepath.Join(chartCache, "images-"+version+".txt")
	} else {
		imagesFile = filepath.Join(chartCache, "images-latest.txt")
	}
	if _, err = os.Stat(imagesFile); err != nil {
		if os.IsNotExist(err) {
			createFile = true
		} else {
			return errors.Errorf("Unable to determine version file %v exists", imagesFile)
		}
	}
	if _, err = os.Stat(chartFile); err != nil {
		if os.IsNotExist(err) {
			createFile = true
		} else {
			return errors.Errorf("Unable to determine chart file %v exists", chartFile)
		}
	}
	if createFile {
		if err = createImagesFile(version, imagesFile); err != nil {
			return err
		}
	}
	if pullImages {
		if file, err = os.Open(imagesFile); err != nil {
			return err
		}
		defer file.Close()

		scanner = bufio.NewScanner(file)

		for scanner.Scan() {
			parts = strings.Split(scanner.Text(), "/")
			image = parts[len(parts)-1]
			fmt.Fprintln(m.Out, "COPY --from="+publicRegistry+"/qlikop-"+image+" /cache/"+image+" /cache/"+image)
		}

		if err = scanner.Err(); err != nil {
			return err
		}
	}
	if len(version) > 0 {
		chartFile = filepath.Join(chartCache, helmDir, chartName+"-"+version+".tgz")
		fmt.Fprintln(m.Out, strings.ReplaceAll("ADD "+chartFile+" /tmp/.chartcache/", "\\", "/"))
	} else {
		chartFile = filepath.Join(chartCache, helmDir, chartName+"-latest.tgz")
		fmt.Fprintln(m.Out, strings.ReplaceAll("ADD "+filepath.Join(chartCache, helmDir, chartName+"-*.tgz")+" /tmp/.chartcache/", "\\", "/"))
	}
	return nil
}

func getTransformerVersion() (string, error) {
	var patchInst patch
	var bytes []byte
	var err error
	var selPatch selectivePatch
	var chart helmChart

	if _, err = os.Stat(versionFile); err != nil {
		if os.IsNotExist(err) {
			return "", err
		}
		return "", errors.Errorf("Unable to determine version file %v exists", versionFile)
	}
	if bytes, err = ioutil.ReadFile(versionFile); err != nil {
		return "", err
	}
	if err = yaml.Unmarshal(bytes, &selPatch); err != nil {
		return "", err
	}
	for _, patchInst = range selPatch.Patches {
		err = yaml.Unmarshal([]byte(patchInst.Patch), &chart)
		if err != nil {
			if chart.ChartName == chartName {
				return chart.ChartVersion, nil
			}
		}
	}
	return "", nil
}

func createImagesFile(version string, imagesFile string) error {
	var image, helmHome string
	var err error
	var images []string
	var file *os.File
	var w *bufio.Writer

	if helmHome, err = ioutil.TempDir("", helmHomePrefix); err != nil {
		return err
	}
	defer os.RemoveAll(helmHome)

	os.Mkdir(chartCache, os.ModePerm)
	os.Setenv("HELM_NAMESPACE", namespace)
	os.Setenv("XDG_CONFIG_HOME", helmHome)
	os.Setenv("XDG_CACHE_HOME", chartCache)
	settings = cli.New()

	if err = repoAdd(stableRepoName, stableRepoURL); err != nil {
		return err
	}

	if err = repoAdd(qlikRepoName, qlikRepoURL); err != nil {
		return err
	}

	if err = repoUpdate(); err != nil {
		return err
	}
	if file, err = os.Create(imagesFile); err != nil {
		return err
	}
	defer file.Close()

	w = bufio.NewWriter(file)
	if images, err = getImages(releaseName, qlikRepoName, chartName, version, sets); err != nil {
		return err
	}
	for _, image = range images {
		fmt.Fprintln(w, image)
	}
	return w.Flush()
}

// RepoAdd adds repo with given name and url
func repoAdd(name, url string) error {
	var (
		repoFile    = settings.RepositoryConfig
		fileLock    *flock.Flock
		lockContext context.Context
		cancel      context.CancelFunc
		locked      bool
		err         error
		b           []byte
		f           repo.File
		r           *repo.ChartRepository
		c           = repo.Entry{
			Name: name,
			URL:  url,
		}
	)

	if err = os.MkdirAll(filepath.Dir(repoFile), os.ModePerm); err != nil && !os.IsExist(err) {
		return err
	}

	// Acquire a file lock for process synchronization
	fileLock = flock.New(strings.Replace(repoFile, filepath.Ext(repoFile), ".lock", 1))

	lockContext, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	locked, err = fileLock.TryLockContext(lockContext, time.Second)
	if err == nil && locked {
		defer fileLock.Unlock()
	}
	if err != nil {
		return err
	}

	if b, err = ioutil.ReadFile(repoFile); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err = yaml.Unmarshal(b, &f); err != nil {
		return err
	}

	if f.Has(name) {
		//fmt.Printf("repository name (%s) already exists\n", name)
		return nil
	}

	if r, err = repo.NewChartRepository(&c, getter.All(settings)); err != nil {
		return err
	}

	if _, err = r.DownloadIndexFile(); err != nil {
		return errors.Wrapf(err, "looks like %q is not a valid chart repository or cannot be reached", url)
	}

	f.Update(&c)

	if err = f.WriteFile(repoFile, 0644); err != nil {
		return err
	}
	return nil
}

// RepoUpdate updates charts for all helm repos
func repoUpdate() error {
	var (
		repoFile = settings.RepositoryConfig
		err      error
		f        *repo.File
		r        *repo.ChartRepository
		repos    []*repo.ChartRepository
		cfg      *repo.Entry
		wg       sync.WaitGroup
	)

	f, err = repo.LoadFile(repoFile)
	if os.IsNotExist(errors.Cause(err)) || len(f.Repositories) == 0 {
		return errors.New("no repositories found. You must add one before updating")
	}

	for _, cfg = range f.Repositories {
		r, err = repo.NewChartRepository(cfg, getter.All(settings))
		if err != nil {
			return err
		}
		repos = append(repos, r)
	}

	// fmt.Printf("Downloading helm chart index ...\n")
	for _, r = range repos {
		wg.Add(1)
		go func(re *repo.ChartRepository) {
			defer wg.Done()
			if _, err = re.DownloadIndexFile(); err != nil {
				// fmt.Printf("...Unable to get an update from the %q chart repository (%s):\n\t%s\n", re.Config.Name, re.Config.URL, err)
			}
		}(r)
	}
	wg.Wait()
	return nil
}

// TemplateChart ...
func getImages(name, repo, chart, version, args string) ([]string, error) {

	var (
		actionConfig   = new(action.Configuration)
		client         = action.NewInstall(actionConfig)
		err            error
		validate       bool
		rel            *release.Release
		m              *release.Hook
		manifests      bytes.Buffer
		splitManifests map[string]string
		manifest       string
		any            = map[string]interface{}{}
		images         = make([]string, 0)
		k              string
		v              interface{}
	)

	if err = actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), debug); err != nil {
		return images, err
	}

	client.DryRun = true
	client.ReleaseName = releaseName
	client.Replace = true // Skip the name check
	client.ClientOnly = !validate
	if len(version) > 0 {
		client.Version = version
	}
	//	client.APIVersions = chartutil.VersionSet(extraAPIs)
	if rel, err = runInstall(name, repo, chart, args, client); err != nil {
		return images, err
	}

	fmt.Fprintln(&manifests, strings.TrimSpace(rel.Manifest))
	for _, m = range rel.Hooks {
		fmt.Fprintf(&manifests, "---\n# Source: %s\n%s\n", m.Path, m.Manifest)
	}
	// fmt.Printf("Building Image List ...\n")
	splitManifests = releaseutil.SplitManifests(manifests.String())
	for _, manifest = range splitManifests {
		if err = yaml.Unmarshal([]byte(manifest), &any); err != nil {
			return images, err
		}
		for k, v = range any {
			images = searchImages(k, v, images)
		}
	}
	// fmt.Printf("Done Image List\n")
	return uniqueNonEmptyElementsOf(images), nil
}

func runInstall(name, repo, chartName, sets string, client *action.Install) (*release.Release, error) {
	var (
		valueOpts             = &values.Options{}
		vals                  map[string]interface{}
		p                     getter.Providers
		cp                    string
		validInstallableChart bool
		err                   error
		chartRequested        *chart.Chart
		req                   []*chart.Dependency
		man                   *downloader.Manager
	)

	debug("Original chart version: %q", client.Version)
	if client.Version == "" && client.Devel {
		debug("setting version to >0.0.0-0")
		client.Version = ">0.0.0-0"
	}

	if cp, err = client.ChartPathOptions.LocateChart(fmt.Sprintf("%s/%s", repo, chartName), settings); err != nil {
		return nil, err
	}

	debug("CHART PATH: %s\n", cp)

	p = getter.All(settings)
	if vals, err = valueOpts.MergeValues(p); err != nil {
		return nil, err
	}

	// Add args
	if err = strvals.ParseInto(sets, vals); err != nil {
		return nil, errors.Wrap(err, "failed parsing --set data")
	}
	// Check chart dependencies to make sure all are present in /charts
	// fmt.Printf("Downloading helm chart ...\n")
	if chartRequested, err = loader.Load(cp); err != nil {
		return nil, err
	}

	validInstallableChart, err = isChartInstallable(chartRequested)
	if !validInstallableChart {
		return nil, err
	}

	if req = chartRequested.Metadata.Dependencies; req != nil {
		// If CheckDependencies returns an error, we have unfulfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		// https://github.com/helm/helm/issues/2209
		if err = action.CheckDependencies(chartRequested, req); err != nil {
			if client.DependencyUpdate {
				man = &downloader.Manager{
					Out:              os.Stdout,
					ChartPath:        cp,
					Keyring:          client.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          p,
					RepositoryConfig: settings.RepositoryConfig,
					RepositoryCache:  settings.RepositoryCache,
				}
				if err = man.Update(); err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
	}

	client.Namespace = settings.Namespace()
	return client.Run(chartRequested, vals)
}

func isChartInstallable(ch *chart.Chart) (bool, error) {
	switch ch.Metadata.Type {
	case "", "application":
		return true, nil
	}
	return false, errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}

func debug(format string, v ...interface{}) {
	//format = fmt.Sprintf("[debug] %s\n", format)
	//log.Output(2, fmt.Sprintf(format, v...))
}

func searchImages(key string, value interface{}, images []string) []string {
	var (
		submap     map[interface{}]interface{}
		stringlist []interface{}
		k, v       interface{}
		ok         bool
		i          int
	)

	submap, ok = value.(map[interface{}]interface{})
	if ok {
		for k, v = range submap {
			images = searchImages(k.(string), v, images)
		}
		return images
	}
	stringlist, ok = value.([]interface{})
	if ok {
		images = searchImages("size", len(stringlist), images)
		for i, v = range stringlist {
			images = searchImages(fmt.Sprintf("%d", i), v, images)
		}
		return images
	}

	if key == "image" {
		images = append(images, fmt.Sprintf("%v", value))
	}
	return images
}

func uniqueNonEmptyElementsOf(s []string) []string {
	var (
		unique = make(map[string]bool, len(s))
		us     = make([]string, len(unique))
		elem   string
	)
	for _, elem = range s {
		if len(elem) != 0 {
			if !unique[elem] {
				us = append(us, elem)
				unique[elem] = true
			}
		}
	}
	return us
}
