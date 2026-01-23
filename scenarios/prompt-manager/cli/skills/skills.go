// Package skills provides CLI commands for skill management.
package skills

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

// SkillResponse matches the API response for skills
type SkillResponse struct {
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

// CreateSkillRequest matches the API request for creating skills
type CreateSkillRequest struct {
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

// Commands returns the skill command groups.
func Commands(ctx appctx.Context) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		{
			Title: "Skills",
			Commands: []cliapp.Command{
				{
					Name:        "add",
					Aliases:     []string{"create"},
					NeedsAPI:    true,
					Description: "Add a new skill",
					Run: func(args []string) error {
						return cmdAdd(ctx, args)
					},
				},
				{
					Name:        "list",
					Aliases:     []string{"ls"},
					NeedsAPI:    true,
					Description: "List skills (optionally filter by folder: core, local, drafts)",
					Run: func(args []string) error {
						return cmdList(ctx, args)
					},
				},
				{
					Name:        "show",
					Aliases:     []string{"get", "details"},
					NeedsAPI:    true,
					Description: "Show detailed skill information",
					Run: func(args []string) error {
						return cmdShow(ctx, args)
					},
				},
				{
					Name:        "use",
					Aliases:     []string{"copy"},
					NeedsAPI:    true,
					Description: "Record usage and display skill content",
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
	fmt.Println("Enter skill content (end with Ctrl+D on a new line):")
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
		return fmt.Errorf("skill content is required")
	}

	req := CreateSkillRequest{
		Name:    name,
		Content: content,
		Folder:  folder,
		Modes:   []string{},
		Tags:    []string{},
	}

	var skill SkillResponse
	if err := ctx.Post("/skills", req, &skill); err != nil {
		return fmt.Errorf("failed to create skill: %w", err)
	}

	fmt.Printf("Created skill: %s [%s] in %s/\n", skill.Name, skill.ID, skill.Folder)
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

	var skills []SkillResponse
	if err := ctx.GetWithQuery("/skills", query, &skills); err != nil {
		return fmt.Errorf("failed to list skills: %w", err)
	}

	if len(skills) == 0 {
		fmt.Println("No skills found")
		return nil
	}

	fmt.Println("Skills:")
	for _, p := range skills {
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
		return fmt.Errorf("usage: show <skill-id>")
	}

	skillID := args[0]

	var skill SkillResponse
	if err := ctx.Get(fmt.Sprintf("/skills/%s", skillID), &skill); err != nil {
		return fmt.Errorf("failed to get skill: %w", err)
	}

	fmt.Printf("Name: %s\n", skill.Name)
	fmt.Printf("Folder: %s\n", skill.Folder)
	if skill.Description != "" {
		fmt.Printf("Description: %s\n", skill.Description)
	}
	fmt.Printf("Usage Count: %d\n", skill.UsageCount)
	fmt.Printf("Draft: %v\n", skill.Draft)
	fmt.Printf("Created: %s\n", skill.CreatedAt)
	fmt.Printf("Updated: %s\n", skill.UpdatedAt)
	fmt.Printf("ID: %s\n", skill.ID)

	if len(skill.Modes) > 0 {
		fmt.Printf("Modes: %s\n", strings.Join(skill.Modes, ", "))
	}
	if len(skill.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(skill.Tags, ", "))
	}

	fmt.Printf("\nContent:\n%s\n", skill.Content)

	return nil
}

func cmdUse(ctx appctx.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: use <skill-id>")
	}

	skillID := args[0]

	// Record usage
	if err := ctx.Post(fmt.Sprintf("/skills/%s/use", skillID), struct{}{}, nil); err != nil {
		return fmt.Errorf("failed to record usage: %w", err)
	}

	// Get and display the skill
	var skill SkillResponse
	if err := ctx.Get(fmt.Sprintf("/skills/%s", skillID), &skill); err != nil {
		return fmt.Errorf("failed to get skill: %w", err)
	}

	fmt.Println("Usage recorded!")
	fmt.Printf("\nSkill Content:\n%s\n", skill.Content)

	// Copy to clipboard if available
	if clipboard.IsAvailable() {
		if errMsg := clipboard.Copy(skill.Content); errMsg == "" {
			fmt.Printf("\n(Copied to clipboard via %s)\n", clipboard.ToolName())
		} else {
			fmt.Printf("\n(%s)\n", errMsg)
		}
	}

	return nil
}
