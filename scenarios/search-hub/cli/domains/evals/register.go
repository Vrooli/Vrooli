// Package evals is the CLI's eval-domain command surface. It mirrors the API's
// EvalService Connect-RPC (register / list / show / run / runs / show-run /
// compare) — the per-provider search-quality baseline harness.
//
// Like the providers domain, the manifest (cli/manifest.json) is the single
// source of truth for the command-line shape; handlers live in handlers.go and
// are wired via the bindings map. See domains/providers for the canonical shape.
package evals

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "evals"

// Register builds the evals subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"EvalService.RegisterSuite": h.register,
		"EvalService.ListSuites":    h.list,
		"EvalService.GetSuite":      h.show,
		"EvalService.RunSuite":      h.run,
		"EvalService.ListRuns":      h.runs,
		"EvalService.GetRun":        h.showRun,
		"EvalService.CompareRuns":   h.compare,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("evals: load from manifest: %w", err)
	}
	return group, nil
}
