// Package search provides CLI commands for searching prompts.
package search

import (
	"fmt"
	"net/url"

	"github.com/vrooli/cli-core/cliapp"

	"prompt-manager/cli/internal/appctx"
	"prompt-manager/cli/internal/types"
)

// Commands returns the search command group.
func Commands(ctx appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Search",
		Commands: []cliapp.Command{
			{
				Name:        "search",
				Aliases:     []string{"find"},
				NeedsAPI:    true,
				Description: "Search prompts by content or title",
				Run: func(args []string) error {
					return cmdSearch(ctx, args)
				},
			},
		},
	}
}

func cmdSearch(ctx appctx.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: search <query>")
	}

	query := args[0]

	// Build query parameters
	params := url.Values{}
	params.Set("q", query)

	var prompts []types.Prompt
	if err := ctx.GetWithQuery("/search/prompts", params, &prompts); err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(prompts) == 0 {
		fmt.Printf("No prompts found matching: %s\n", query)
		return nil
	}

	fmt.Printf("Search Results (%d found):\n", len(prompts))
	for _, p := range prompts {
		campaign := "No Campaign"
		if p.CampaignName != nil {
			campaign = *p.CampaignName
		}
		fmt.Printf("  %s - %s [%s]\n", p.Title, campaign, p.ID)
	}
	return nil
}
