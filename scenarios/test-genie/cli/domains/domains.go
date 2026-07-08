package domains

import (
	"test-genie/cli/domains/local"
	"test-genie/cli/domains/suites"
	"test-genie/cli/eligibility"
	"test-genie/cli/internal/deps"
	"test-genie/cli/runs"

	"github.com/vrooli/api-core/spacecli"
	"github.com/vrooli/api-core/spacedoc"
	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates the scenario's domain registrations. The root
// /health probe is served by cli-core's built-in `status` command, so no
// status/system/health domain is registered here.
func CommandGroups(runtime deps.Runtime) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		suites.Register(runtime),
		local.Register(runtime),
		// test-genie owns the Validate projection denominator
		// (docs/spaces/validate-space.md); `space` is the cross-scenario read
		// contract meta-optimization-manager consumes.
		spacecli.CommandGroup(spacecli.Config{Owner: "test-genie", Projection: spacedoc.ProjectionValidate}),
	}
}

// SubcommandGroups returns manifest-backed hierarchical command groups.
func SubcommandGroups(manifest []byte, runtime deps.Runtime) ([]cliapp.SubcommandGroup, error) {
	runsGroup, err := runs.Register(manifest, runtime.APIClient)
	if err != nil {
		return nil, err
	}
	eligibilityGroup, err := eligibility.Register(manifest, runtime.APIClient)
	if err != nil {
		return nil, err
	}
	return []cliapp.SubcommandGroup{runsGroup, eligibilityGroup}, nil
}
