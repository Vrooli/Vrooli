// Package campaigns provides CLI commands for campaign management.
package campaigns

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"

	"prompt-manager/cli/internal/appctx"
	"prompt-manager/cli/internal/types"
)

// Commands returns the campaign command group.
func Commands(ctx appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Campaigns",
		Commands: []cliapp.Command{
			{
				Name:        "campaigns",
				Aliases:     []string{"camp"},
				NeedsAPI:    true,
				Description: "List all campaigns",
				Run: func(args []string) error {
					// If subcommand is provided, dispatch it
					if len(args) > 0 {
						switch args[0] {
						case "list", "ls":
							return cmdList(ctx)
						case "create":
							return cmdCreate(ctx, args[1:])
						default:
							return fmt.Errorf("unknown subcommand: %s (use 'list' or 'create')", args[0])
						}
					}
					// Default to list
					return cmdList(ctx)
				},
			},
		},
	}
}

func cmdList(ctx appctx.Context) error {
	var campaigns []types.Campaign
	if err := ctx.Get("/campaigns", &campaigns); err != nil {
		return err
	}

	if len(campaigns) == 0 {
		fmt.Println("No campaigns found")
		return nil
	}

	fmt.Println("Campaigns:")
	for _, c := range campaigns {
		desc := ""
		if c.Description != nil && *c.Description != "" {
			desc = fmt.Sprintf(" - %s", *c.Description)
		}
		fmt.Printf("  %s (%d prompts)%s [%s]\n", c.Name, c.PromptCount, desc, c.ID)
	}
	return nil
}

func cmdCreate(ctx appctx.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: campaigns create <name> [description] [color] [icon]")
	}

	name := args[0]

	req := types.CreateCampaignRequest{
		Name: name,
	}

	if len(args) > 1 {
		req.Description = types.StringPtr(args[1])
	}
	if len(args) > 2 {
		req.Color = types.StringPtr(args[2])
	}
	if len(args) > 3 {
		req.Icon = types.StringPtr(args[3])
	}

	var campaign types.Campaign
	if err := ctx.Post("/campaigns", req, &campaign); err != nil {
		return fmt.Errorf("failed to create campaign: %w", err)
	}

	fmt.Printf("Created campaign: %s [%s]\n", campaign.Name, campaign.ID)
	return nil
}
