package ai

import (
	"fmt"
	"sort"
	"strings"
)

// Markers bounding the generated host-tool matrix inside
// docs/reference/backends.md. The content between them is rendered from
// providerSpecs() (RenderHostToolMatrix) — never hand-edited — and a freshness
// test (TestBackendsDocHostToolMatrixUpToDate) fails the build on drift.
const (
	HostToolMatrixBeginMarker = "<!-- BEGIN GENERATED: host-tool-matrix (regenerate with `make backends-doc`) -->"
	HostToolMatrixEndMarker   = "<!-- END GENERATED: host-tool-matrix -->"
)

// RenderHostToolMatrix renders the provider→host-tool provisioning matrix as a
// Markdown table, derived from HostToolBindings(). This is the single source of
// truth: the docs table, the runtime provision message, and the conformance
// tests all flow from providerSpecs().
func RenderHostToolMatrix() string {
	bindings := HostToolBindings()
	sort.Slice(bindings, func(i, j int) bool {
		if bindings[i].Provider != bindings[j].Provider {
			return bindings[i].Provider < bindings[j].Provider
		}
		return strings.Join(bindings[i].Ops, ",") < strings.Join(bindings[j].Ops, ",")
	})

	var b strings.Builder
	b.WriteString("| Backend (provider) | Host tool | Operations | Install / remediation |\n")
	b.WriteString("|---|---|---|---|\n")
	for _, bd := range bindings {
		ops := make([]string, 0, len(bd.Ops))
		for _, op := range bd.Ops {
			ops = append(ops, "`"+op+"`")
		}
		install := fmt.Sprintf("`vrooli host install %s`", bd.HostTool)
		fmt.Fprintf(&b, "| `%s` | `%s` | %s | %s |\n", bd.Provider, bd.HostTool, strings.Join(ops, ", "), install)
	}
	return b.String()
}
