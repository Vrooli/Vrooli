package local

import (
	"test-genie/cli/internal/deps"
	"test-genie/cli/playbooksseed"
	"test-genie/cli/registry"
	"test-genie/cli/requirements"
	"test-genie/cli/runlocal"
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
