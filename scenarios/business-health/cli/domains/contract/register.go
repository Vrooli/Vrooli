// Package contract is the CLI's business-contract command surface: the
// validate / matrix / drift / manual-log groups, mirroring the API's
// ContractService. The manifest carries the declarative surface
// (governance, flags, positionals, RPC bindings) and is the single
// source of truth for the command-line shape.
package contract

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// Group names this package owns in cli/manifest.json.
const (
	ValidateGroup  = "validate"
	MatrixGroup    = "matrix"
	DriftGroup     = "drift"
	ManualLogGroup = "manual-log"
)

// Register builds the four contract-facing subcommand groups from the
// embedded manifest and wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) ([]cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	groups := make([]cliapp.SubcommandGroup, 0, 4)
	for _, spec := range []struct {
		name     string
		bindings map[string]func(cliapp.RunContext) error
	}{
		{ValidateGroup, map[string]func(cliapp.RunContext) error{"ContractService.ValidateScenario": h.validateScenario}},
		{MatrixGroup, map[string]func(cliapp.RunContext) error{"ContractService.GetMatrix": h.getMatrix}},
		{DriftGroup, map[string]func(cliapp.RunContext) error{"ContractService.GetDrift": h.getDrift}},
		{ManualLogGroup, map[string]func(cliapp.RunContext) error{"ContractService.LogManualValidation": h.logManual}},
	} {
		group, err := cliapp.LoadFromManifest(manifest, spec.name, spec.bindings)
		if err != nil {
			return nil, fmt.Errorf("contract: load group %q from manifest: %w", spec.name, err)
		}
		groups = append(groups, group)
	}
	return groups, nil
}
