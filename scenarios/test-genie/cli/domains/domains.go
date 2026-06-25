package domains

import (
	"test-genie/cli/domains/local"
	"test-genie/cli/domains/suites"
	"test-genie/cli/internal/deps"

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
