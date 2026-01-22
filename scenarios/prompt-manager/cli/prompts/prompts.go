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
)

// PromptResponse matches the API response for prompts
type PromptResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Content     string   `json:"content"`
	Modes       []string `json:"modes"`
	Tags        []string `json:"tags"`
	Icon        string   `json:"icon,omitempty"`
	Draft       bool     `json:"draft"`
	Folder      string   `json:"folder"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
	UsageCount  int      `json:"usageCount"`
}

// CreatePromptRequest matches the API request for creating prompts
type CreatePromptRequest struct {
	ID          string   `json:"id,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Content     string   `json:"content"`
	Modes       []string `json:"modes,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Icon        string   `json:"icon,omitempty"`
	Draft       bool     `json:"draft"`
	Folder      string   `json:"folder"`
}

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
					Description: "List prompts (optionally filter by folder: core, local, drafts)",
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
			},
		},
	}
}

func cmdAdd(ctx appctx.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: add <name> [folder]\n  folder: 'local' (default) or 'drafts'")
	}

	name := args[0]

	// Get folder (default to "local")
	folder := "local"
	if len(args) > 1 {
		folder = strings.ToLower(args[1])
		if folder != "local" && folder != "drafts" {
			return fmt.Errorf("folder must be 'local' or 'drafts'")
		}
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

	req := CreatePromptRequest{
		Name:    name,
		Content: content,
		Folder:  folder,
		Modes:   []string{},
		Tags:    []string{},
	}

	var prompt PromptResponse
	if err := ctx.Post("/prompts", req, &prompt); err != nil {
		return fmt.Errorf("failed to create prompt: %w", err)
	}

	fmt.Printf("Created prompt: %s [%s] in %s/\n", prompt.Name, prompt.ID, prompt.Folder)
	return nil
}

func cmdList(ctx appctx.Context, args []string) error {
	query := url.Values{}

	// Parse arguments
	var folderFilter string
	var tagFilter string
	for _, arg := range args {
		switch {
		case arg == "core" || arg == "local" || arg == "drafts":
			folderFilter = arg
		default:
			tagFilter = arg
		}
	}

	if folderFilter != "" {
		query.Set("folder", folderFilter)
	}
	if tagFilter != "" {
		query.Set("tag", tagFilter)
	}

	var prompts []PromptResponse
	if err := ctx.GetWithQuery("/prompts", query, &prompts); err != nil {
		return fmt.Errorf("failed to list prompts: %w", err)
	}

	if len(prompts) == 0 {
		fmt.Println("No prompts found")
		return nil
	}

	fmt.Println("Prompts:")
	for _, p := range prompts {
		tags := ""
		if len(p.Tags) > 0 {
			tags = " [" + strings.Join(p.Tags, ", ") + "]"
		}
		fmt.Printf("  %s - %s (used %d times)%s [%s]\n", p.Name, p.Folder, p.UsageCount, tags, p.ID)
	}
	return nil
}

func cmdShow(ctx appctx.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: show <prompt-id>")
	}

	promptID := args[0]

	var prompt PromptResponse
	if err := ctx.Get(fmt.Sprintf("/prompts/%s", promptID), &prompt); err != nil {
		return fmt.Errorf("failed to get prompt: %w", err)
	}

	fmt.Printf("Name: %s\n", prompt.Name)
	fmt.Printf("Folder: %s\n", prompt.Folder)
	if prompt.Description != "" {
		fmt.Printf("Description: %s\n", prompt.Description)
	}
	fmt.Printf("Usage Count: %d\n", prompt.UsageCount)
	fmt.Printf("Draft: %v\n", prompt.Draft)
	fmt.Printf("Created: %s\n", prompt.CreatedAt)
	fmt.Printf("Updated: %s\n", prompt.UpdatedAt)
	fmt.Printf("ID: %s\n", prompt.ID)

	if len(prompt.Modes) > 0 {
		fmt.Printf("Modes: %s\n", strings.Join(prompt.Modes, ", "))
	}
	if len(prompt.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(prompt.Tags, ", "))
	}

	fmt.Printf("\nContent:\n%s\n", prompt.Content)

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
	var prompt PromptResponse
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
