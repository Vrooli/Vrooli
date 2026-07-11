package local

import (
	"test-genie/cli/fleet"
	"test-genie/cli/health"
	"test-genie/cli/internal/deps"
	phasecmd "test-genie/cli/phases"
	"test-genie/cli/playbooksseed"
	"test-genie/cli/providercontract"
	"test-genie/cli/registry"
	"test-genie/cli/requirements"
	"test-genie/cli/runlocal"

	"github.com/vrooli/cli-core/cliapp"
)

// Register returns the local-operations command group.
func Register(runtime deps.Runtime) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Local",
		Commands: []cliapp.Command{
			{
				Name:        "run-tests",
				NeedsAPI:    true,
				Description: "Trigger scenario-local test runner",
				Run:         func(args []string) error { return runlocal.Run(runtime.RunLocal, args) },
			},
			{
				Name:        "registry",
				NeedsAPI:    false,
				Description: "Manage playbook registries",
				Run:         func(args []string) error { return registry.Run(args) },
			},
			{
				Name:        "requirements",
				NeedsAPI:    false,
				Description: "Inspect and sync scenario requirements",
				Run:         func(args []string) error { return requirements.Run(args) },
			},
			{
				Name:        "provider-contract",
				NeedsAPI:    false,
				Description: "Validate provider maturity assessment contracts",
				Run:         func(args []string) error { return providercontract.Run(args) },
			},
			{
				Name:        "health",
				NeedsAPI:    true,
				Description: "Show Test Genie self-health: catalog, provider conformance, reliability ledger",
				Run:         func(args []string) error { return health.Run(runtime.APIClient, args) },
			},
			{
				Name:        "phases",
				NeedsAPI:    true,
				Description: "Inspect phase catalog, applicability, and execution plans",
				Run:         func(args []string) error { return phasecmd.Run(runtime.APIClient, args) },
			},
			{
				Name:        "fleet",
				NeedsAPI:    true,
				Description: "Fleet-wide health over stored runs: fleet status [--json] [--roster] (as-of stamped, most-errored first)",
				Run:         func(args []string) error { return fleet.Run(runtime.APIClient, args) },
			},
			{
				Name:        "playbooks-seed",
				NeedsAPI:    true,
				Description: "Manage playbooks seed lifecycle (apply/cleanup)",
				Run:         func(args []string) error { return playbooksseed.Run(runtime.Seed, args) },
			},
		},
	}
}
