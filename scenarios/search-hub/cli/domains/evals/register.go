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
	bindings := map[string]cliapp.PrimitiveHandler{
		"EvalService.RegisterSuite":    cliapp.ProtoMutation(h.registerCall, h.registerReport),
		"EvalService.ListSuites":       cliapp.ProtoList(h.listCall, h.listReport),
		"EvalService.GetSuite":         cliapp.ProtoList(h.showCall, h.showReport),
		"EvalService.RunSuite":         cliapp.ProtoMutationOutcome(h.runCall, h.runReport, h.runOutcome),
		"EvalService.ValidateCorpus":   cliapp.ProtoList(h.validateCall, h.validateReport),
		"EvalService.ListRuns":         cliapp.ProtoList(h.runsCall, h.runsReport),
		"EvalService.GetRun":           cliapp.ProtoList(h.showRunCall, h.showRunReport),
		"EvalService.CompareRuns":      cliapp.ProtoList(h.compareCall, h.compareReport),
		"EvalService.Sweep":            cliapp.ProtoList(h.sweepCall, h.sweepReport),
		"EvalService.Generate":         cliapp.ProtoList(h.generateCall, h.generateReport),
		"EvalService.PromoteCases":     cliapp.ProtoList(h.promoteCall, h.promoteReport),
		"EvalService.ReapOrphanSuites": cliapp.ProtoList(h.reapCall, h.reapReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("evals: load from manifest: %w", err)
	}
	return group, nil
}
