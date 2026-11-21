package validate

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "validate"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"ScenarioValidationService.ValidateScenario": h.validateScenario,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("validate: load from manifest: %w", err)
	}
	// LoadFromManifest marks the whole group as API-backed because its
	// declarative commands use Connect-RPC. This group also owns the local
	// prove-isolation command, so keep the group neutral and mark the actual
	// API-backed commands explicitly below. Otherwise cli-core combines
	// group.NeedsAPI with the command flag and rejects prove-isolation while
	// the target API is intentionally stopped.
	group.NeedsAPI = false
	for i := range group.Subcommands {
		if group.Subcommands[i].Name == "scenario" {
			group.Subcommands[i].NeedsAPI = true
			group.Subcommands[i].RunCtx = h.validateScenario
		}
	}
	for _, kind := range []string{"resource", "tool", "safeguard"} {
		group.Subcommands = append(group.Subcommands, ownerCommand(kind, h))
	}
	group.Subcommands = append(group.Subcommands, fleetCommand(h))
	for i := range group.Subcommands {
		switch group.Subcommands[i].Name {
		case "resource", "tool", "safeguard":
			group.Subcommands[i] = ownerCommand(group.Subcommands[i].Name, h)
		case "fleet":
			group.Subcommands[i].Args = cliapp.ArgSchema{Flags: []cliapp.Flag{
				{Name: "kind", Description: "filter by owner kind; repeatable", Values: []string{"scenario", "resource", "tool", "safeguard"}},
				{Name: "platform", Description: "linux, macos, or windows"},
			}}
			group.Subcommands[i].NeedsAPI = true
			group.Subcommands[i].RunCtx = h.validateFleet
		}
	}
	group.Subcommands = append(group.Subcommands, cliapp.Command{
		Name:        "prove-isolation",
		Description: "Prove routed storage isolation without starting the target API",
		NeedsAPI:    false,
		Args:        cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "name", Required: true, Description: "Scenario id"}}},
		RunCtx:      h.proveIsolation,
	})
	return group, nil
}

func ownerCommand(kind string, h *handlers) cliapp.Command {
	return cliapp.Command{
		Name:        kind,
		Description: fmt.Sprintf("Validate a %s storage owner", kind),
		NeedsAPI:    true,
		Args: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{{Name: "name", Required: true, Description: kind + " id"}},
			Flags:       []cliapp.Flag{{Name: "platform", Description: "linux, macos, or windows"}},
		},
		RunCtx: func(ctx cliapp.RunContext) error { return h.validateOwner(ctx, kind, ctx.Positional("name")) },
	}
}

func fleetCommand(h *handlers) cliapp.Command {
	return cliapp.Command{
		Name:        "fleet",
		Description: "Validate every storage owner in the repository",
		NeedsAPI:    true,
		Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "kind", Description: "filter by owner kind; repeatable", Values: []string{"scenario", "resource", "tool", "safeguard"}},
			{Name: "platform", Description: "linux, macos, or windows"},
		}},
		RunCtx: h.validateFleet,
	}
}
