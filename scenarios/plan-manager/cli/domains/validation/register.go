// Package validation is the CLI's validation-domain command surface. It owns the
// `validate` manifest group — resolve references, compute staleness, derive +
// run baselines, and verify the Definition of Done — backed by the API's
// ValidationService. handlers.go builds each typed request and renders the
// response.
package validation

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "validate"

// Register builds the validate subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"ValidationService.ResolveReferences":      h.references,
		"ValidationService.ComputeStaleness":       h.staleness,
		"ValidationService.DeriveBaselineScope":    h.baselineScope,
		"ValidationService.RunValidation":          h.run,
		"ValidationService.VerifyDefinitionOfDone": h.verifyDoD,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("validation: load validate group: %w", err)
	}
	return group, nil
}
