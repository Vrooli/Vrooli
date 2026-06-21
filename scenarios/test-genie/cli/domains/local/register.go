package local

import (
	"test-genie/cli/eligibility"
	"test-genie/cli/fix"
	"test-genie/cli/fleet"
	"test-genie/cli/health"
	"test-genie/cli/internal/deps"
	"test-genie/cli/playbooksseed"
	"test-genie/cli/providercontract"
	"test-genie/cli/registry"
	"test-genie/cli/requirements"
	"test-genie/cli/runlocal"
	"test-genie/cli/runs"
	"test-genie/cli/storage"
	"test-genie/cli/uismoke"

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
				Name:        "ui-smoke",
				NeedsAPI:    true,
				Description: "Run UI smoke test for a scenario",
				Run:         func(args []string) error { return uismoke.Run(runtime.UISmoke, args) },
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
				Name:        "eligibility",
				NeedsAPI:    true,
				Description: "Check whether a scenario qualifies for the routed test-db path",
				Run:         func(args []string) error { return eligibility.Run(runtime.APIClient, args) },
			},
			{
				Name:        "runs",
				NeedsAPI:    true,
				Description: "Inspect, pin, compare, delete, and check freshness of recorded test runs",
				Run:         func(args []string) error { return runs.Run(runtime.APIClient, args) },
			},
			{
				Name:        "health",
				NeedsAPI:    true,
				Description: "Show Test Genie self-health: catalog, provider conformance, reliability ledger",
				Run:         func(args []string) error { return health.Run(runtime.APIClient, args) },
			},
			{
				Name:        "fix",
				NeedsAPI:    true,
				Description: "Remediate a scenario: --deterministic aggregates provider autofixers (dry-run; --apply to write), --fleet walks the priority-ordered fleet, else spawns a fix agent",
				Run:         func(args []string) error { return fix.Run(runtime.APIClient, args) },
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
			{
				Name:        "storage",
				NeedsAPI:    false,
				Description: "Run one-time storage maintenance tasks",
				Run:         func(args []string) error { return storage.Run(args) },
			},
		},
	}
}
