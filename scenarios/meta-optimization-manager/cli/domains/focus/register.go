// Package focus is the CLI's focus-domain command surface. It owns two manifest
// groups — `focus` (the ranked next-best gaps) and `gaps` (the registry list /
// show / note verbs) — both backed by the API's FocusService. The manifest
// (cli/manifest.json) carries the declarative command shape; handlers.go builds
// each typed request and renders the response.
package focus

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// FocusGroup and GapsGroup are the manifest group names this package owns.
const (
	FocusGroup     = "focus"
	GapsGroup      = "gaps"
	ConditionGroup = "condition"
)

// Register builds the focus + gaps subcommand groups from the embedded manifest
// and wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) ([]cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	focusGroup, err := cliapp.LoadFromManifest(manifest, FocusGroup, map[string]func(cliapp.RunContext) error{
		"FocusService.GetFocus": h.next,
	})
	if err != nil {
		return nil, fmt.Errorf("focus: load focus group: %w", err)
	}
	gapsGroup, err := cliapp.LoadFromManifest(manifest, GapsGroup, map[string]func(cliapp.RunContext) error{
		"FocusService.ListGaps":   h.gapsList,
		"FocusService.GetGap":     h.gapsShow,
		"FocusService.AddGapNote": h.gapsNote,
	})
	if err != nil {
		return nil, fmt.Errorf("focus: load gaps group: %w", err)
	}
	conditionGroup, err := cliapp.LoadFromManifest(manifest, ConditionGroup, map[string]func(cliapp.RunContext) error{
		"FocusService.ListCondition":    h.conditionStatus,
		"FocusService.ExplainCondition": h.conditionExplain,
	})
	if err != nil {
		return nil, fmt.Errorf("focus: load condition group: %w", err)
	}
	return []cliapp.SubcommandGroup{focusGroup, gapsGroup, conditionGroup}, nil
}
