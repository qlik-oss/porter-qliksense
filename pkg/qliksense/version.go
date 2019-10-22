package qliksense

import (
	"github.com/qlik-oss/porter-qliksense/pkg"
	"github.com/deislabs/porter/pkg/mixin"
	"github.com/deislabs/porter/pkg/porter/version"
)

func (m *Mixin) PrintVersion(opts version.Options) error {
	metadata := mixin.Metadata{
		Name: "qliksense",
		VersionInfo: mixin.VersionInfo{
			Version: pkg.Version,
			Commit:  pkg.Commit,
			Author:  "Boris Kuschel",
		},
	}
	return version.PrintVersion(m.Context, opts, metadata)
}
