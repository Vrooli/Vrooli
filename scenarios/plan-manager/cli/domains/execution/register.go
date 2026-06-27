// Package execution is the CLI's execution-domain command surface. It owns the
// `exec` manifest group — start a guided run, get the just-in-time context,
// advance + transition phases, capture decisions/findings in-flow, complete +
// fetch the canonical handoff, list/triage candidate findings, and read the
// per-plan velocity series — all backed by the API's ExecutionService. The
// manifest (cli/manifest.json) carries the declarative command shape; handlers.go
// builds each typed request and renders the response.
package execution

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "exec"

// Register builds the exec subcommand group from the embedded manifest and wires
// Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"ExecutionService.Start":                 h.start,
		"ExecutionService.GetStatus":             h.status,
		"ExecutionService.GetContext":            h.context,
		"ExecutionService.Resume":                h.resume,
		"ExecutionService.GetNext":               h.next,
		"ExecutionService.TransitionPhase":       h.transition,
		"ExecutionService.RecordDecision":        h.decisionAdd,
		"ExecutionService.RecordFinding":         h.findingAdd,
		"ExecutionService.Complete":              h.complete,
		"ExecutionService.GetHandoff":            h.handoff,
		"ExecutionService.ListCandidateFindings": h.findings,
		"ExecutionService.TriageFinding":         h.triage,
		"ExecutionService.GetVelocity":           h.velocity,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("execution: load exec group: %w", err)
	}
	return group, nil
}
