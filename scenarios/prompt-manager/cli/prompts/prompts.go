// Package prompts provides CLI commands for prompt management.
package prompts

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"

	"prompt-manager/cli/internal/appctx"
	"prompt-manager/cli/internal/clipboard"
	"prompt-manager/cli/internal/types"
)

// Commands returns the prompt command groups.
func Commands(ctx appctx.Context) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		{
			Title: "Prompts",
			Commands: []cliapp.Command{
				{
					Name:        "add",
					Aliases:     []string{"create"},
					NeedsAPI:    true,
					Description: "Add a new prompt",
					Run: func(args []string) error {
						return cmdAdd(ctx, args)
					},
				},
				{
					Name:        "list",
					Aliases:     []string{"ls"},
					NeedsAPI:    true,
					Description: "List prompts (optionally filter by campaign)",
					Run: func(args []string) error {
						return cmdList(ctx, args)
					},
				},
				{
					Name:        "show",
					Aliases:     []string{"get", "details"},
					NeedsAPI:    true,
					Description: "Show detailed prompt information",
					Run: func(args []string) error {
						return cmdShow(ctx, args)
					},
				},
				{
					Name:        "use",
					Aliases:     []string{"copy"},
					NeedsAPI:    true,
					Description: "Record usage and display prompt content",
					Run: func(args []string) error {
						return cmdUse(ctx, args)
					},
				},
				{
					Name:        "quick",
					Aliases:     []string{"q"},
					NeedsAPI:    true,
					Description: "Get prompt by quick access key",
					Run: func(args []string) error {
						return cmdQuick(ctx, args)
					},
				},
			},
		},
	}
}

func cmdAdd(ctx appctx.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: add <title> [campaign-id-or-name]")
	}

	title := args[0]

	// Get campaign ID
	var campaignID string
	if len(args) > 1 {
		campaignID = args[1]
	} else {
		// Show available campaigns and prompt for selection
		var campaigns []types.Campaign
		if err := ctx.Get("/campaigns", &campaigns); err != nil {
			return fmt.Errorf("failed to list campaigns: %w", err)
		}

		if len(campaigns) == 0 {
			return fmt.Errorf("no campaigns found - create one first with: campaigns create <name>")
		}

		fmt.Println("Available campaigns:")
		for i, c := range campaigns {
			fmt.Printf("  [%d] %s (%s)\n", i+1, c.Name, c.ID)
		}
		fmt.Print("Enter campaign number or ID: ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		// Try as number first
		var idx int
		if _, err := fmt.Sscanf(input, "%d", &idx); err == nil && idx > 0 && idx <= len(campaigns) {
			campaignID = campaigns[idx-1].ID
		} else {
			// Try as ID or name
			campaignID = resolveCampaignID(ctx, input, campaigns)
			if campaignID == "" {
				return fmt.Errorf("campaign not found: %s", input)
			}
		}
	}

	// If campaignID is not a UUID, resolve it
	if !isUUID(campaignID) {
		var campaigns []types.Campaign
		if err := ctx.Get("/campaigns", &campaigns); err != nil {
			return fmt.Errorf("failed to list campaigns: %w", err)
		}
		resolved := resolveCampaignID(ctx, campaignID, campaigns)
		if resolved == "" {
			return fmt.Errorf("campaign not found: %s", campaignID)
		}
		campaignID = resolved
	}

	// Get content from stdin
	fmt.Println("Enter prompt content (end with Ctrl+D on a new line):")
	reader := bufio.NewReader(os.Stdin)
	var lines []string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		lines = append(lines, line)
	}
	content := strings.Join(lines, "")
	content = strings.TrimSpace(content)

	if content == "" {
		return fmt.Errorf("prompt content is required")
	}

	req := types.CreatePromptRequest{
		CampaignID: campaignID,
		Title:      title,
		Content:    content,
		Variables:  []string{},
		Tags:       []string{},
	}

	var prompt types.Prompt
	if err := ctx.Post("/prompts", req, &prompt); err != nil {
		return fmt.Errorf("failed to create prompt: %w", err)
	}

	fmt.Printf("Created prompt: %s [%s]\n", prompt.Title, prompt.ID)
	return nil
}

func cmdList(ctx appctx.Context, args []string) error {
	query := url.Values{}

	// Parse arguments
	var campaignFilter string
	var filterType string
	for _, arg := range args {
		switch arg {
		case "favorites", "fav":
			filterType = "favorites"
		case "recent":
			query.Set("limit", "10")
		default:
			if campaignFilter == "" {
				campaignFilter = arg
			}
		}
	}

	if filterType == "favorites" {
		query.Set("is_favorite", "true")
	}

	// Build endpoint
	endpoint := "/prompts"
	if campaignFilter != "" {
		// Resolve campaign ID
		if !isUUID(campaignFilter) {
			var campaigns []types.Campaign
			if err := ctx.Get("/campaigns", &campaigns); err != nil {
				return fmt.Errorf("failed to list campaigns: %w", err)
			}
			resolved := resolveCampaignID(ctx, campaignFilter, campaigns)
			if resolved == "" {
				return fmt.Errorf("campaign not found: %s", campaignFilter)
			}
			campaignFilter = resolved
		}
		endpoint = fmt.Sprintf("/campaigns/%s/prompts", campaignFilter)
	}

	var prompts []types.Prompt
	if err := ctx.GetWithQuery(endpoint, query, &prompts); err != nil {
		return fmt.Errorf("failed to list prompts: %w", err)
	}

	if len(prompts) == 0 {
		fmt.Println("No prompts found")
		return nil
	}

	fmt.Println("Prompts:")
	for _, p := range prompts {
		campaign := "No Campaign"
		if p.CampaignName != nil {
			campaign = *p.CampaignName
		}
		fav := ""
		if p.IsFavorite {
			fav = " *"
		}
		fmt.Printf("  %s - %s (used %d times)%s [%s]\n", p.Title, campaign, p.UsageCount, fav, p.ID)
	}
	return nil
}

func cmdShow(ctx appctx.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: show <prompt-id>")
	}

	promptID := args[0]

	var prompt types.Prompt
	if err := ctx.Get(fmt.Sprintf("/prompts/%s", promptID), &prompt); err != nil {
		return fmt.Errorf("failed to get prompt: %w", err)
	}

	fmt.Printf("Title: %s\n", prompt.Title)
	if prompt.CampaignName != nil {
		fmt.Printf("Campaign: %s\n", *prompt.CampaignName)
	}
	if prompt.Description != nil {
		fmt.Printf("Description: %s\n", *prompt.Description)
	}
	fmt.Printf("Usage Count: %d\n", prompt.UsageCount)
	if prompt.EffectivenessRating != nil {
		fmt.Printf("Rating: %d/5\n", *prompt.EffectivenessRating)
	}
	if prompt.WordCount != nil {
		fmt.Printf("Word Count: %d\n", *prompt.WordCount)
	}
	if prompt.EstimatedTokens != nil {
		fmt.Printf("Est. Tokens: %d\n", *prompt.EstimatedTokens)
	}
	fmt.Printf("Favorite: %v\n", prompt.IsFavorite)
	fmt.Printf("Created: %s\n", prompt.CreatedAt.Format("2006-01-02 15:04:05"))
	if prompt.LastUsed != nil {
		fmt.Printf("Last Used: %s\n", prompt.LastUsed.Format("2006-01-02 15:04:05"))
	}
	fmt.Printf("ID: %s\n", prompt.ID)

	if len(prompt.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(prompt.Tags, ", "))
	}

	fmt.Printf("\nContent:\n%s\n", prompt.Content)

	if prompt.Notes != nil && *prompt.Notes != "" {
		fmt.Printf("\nNotes:\n%s\n", *prompt.Notes)
	}

	return nil
}

func cmdUse(ctx appctx.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: use <prompt-id>")
	}

	promptID := args[0]

	// Record usage
	if err := ctx.Post(fmt.Sprintf("/prompts/%s/use", promptID), struct{}{}, nil); err != nil {
		return fmt.Errorf("failed to record usage: %w", err)
	}

	// Get and display the prompt
	var prompt types.Prompt
	if err := ctx.Get(fmt.Sprintf("/prompts/%s", promptID), &prompt); err != nil {
		return fmt.Errorf("failed to get prompt: %w", err)
	}

	fmt.Println("Usage recorded!")
	fmt.Printf("\nPrompt Content:\n%s\n", prompt.Content)

	// Copy to clipboard if available
	if clipboard.IsAvailable() {
		if errMsg := clipboard.Copy(prompt.Content); errMsg == "" {
			fmt.Printf("\n(Copied to clipboard via %s)\n", clipboard.ToolName())
		} else {
			fmt.Printf("\n(%s)\n", errMsg)
		}
	}

	return nil
}

func cmdQuick(ctx appctx.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: quick <key>")
	}

	key := args[0]

	var prompt types.Prompt
	if err := ctx.Get(fmt.Sprintf("/prompts/quick/%s", key), &prompt); err != nil {
		return fmt.Errorf("no prompt found with quick key: %s", key)
	}

	fmt.Printf("Quick Prompt:\n%s\n", prompt.Content)
	return nil
}

// Helper functions

func isUUID(s string) bool {
	// Simple UUID check (8-4-4-4-12 format)
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}

func resolveCampaignID(_ appctx.Context, input string, campaigns []types.Campaign) string {
	input = strings.ToLower(input)
	for _, c := range campaigns {
		if c.ID == input || strings.ToLower(c.Name) == input {
			return c.ID
		}
		if strings.Contains(strings.ToLower(c.Name), input) {
			return c.ID
		}
	}
	return ""
}
