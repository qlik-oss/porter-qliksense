package qliksense

import "fmt"

// This is an example. Replace thie following with whatever steps are needed to
// install required components into

const dockerfileLines = `
COPY --from=qlik/cnab-qliksense-base:latest /usr/local/bin /usr/local/bin
COPY --from=qlik/cnab-qliksense-base:latest /root/.config/kustomize /root/.config/kustomize
COPY --from=qlik/qliksense-operator:latest /usr/local/bin/qliksense-operator /usr/local/bin
`

// Build will generate the necessary Dockerfile lines
// for an invocation image using this mixin
func (m *Mixin) Build() error {
	fmt.Fprintf(m.Out, dockerfileLines)
	return nil
}
