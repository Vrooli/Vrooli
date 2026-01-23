// Package versions provides CLI commands for skill version history.
package versions

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"

	"prompt-manager/cli/internal/appctx"
	"prompt-manager/cli/internal/types"
)

// Commands returns the versions command group.
func Commands(ctx appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Version History",
		Commands: []cliapp.Command{
			{
				Name:        "versions",
				Aliases:     []string{"history"},
				NeedsAPI:    true,
				Description: "Show version history for a skill",
				Run: func(args []string) error {
					return cmdVersions(ctx, args)
				},
			},
			{
				Name:        "revert",
				Aliases:     []string{"restore"},
				NeedsAPI:    true,
				Description: "Revert a skill to a previous version",
				Run: func(args []string) error {
					return cmdRevert(ctx, args)
				},
			},
		},
	}
}

func cmdVersions(ctx appctx.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: versions <skill-id>")
	}

	skillID := args[0]

	var versions []types.SkillVersion
	if err := ctx.Get(fmt.Sprintf("/skills/%s/versions", skillID), &versions); err != nil {
		return fmt.Errorf("failed to get version history: %w", err)
	}

	if len(versions) == 0 {
		fmt.Println("No version history found for this skill")
		return nil
	}

	fmt.Println("Version History:")
	for _, v := range versions {
		summary := "No summary"
		if v.ChangeSummary != nil && *v.ChangeSummary != "" {
			summary = *v.ChangeSummary
		}
		fmt.Printf("  v%d - %s - %s\n", v.VersionNumber, v.CreatedAt.Format("2006-01-02 15:04"), summary)
	}
	return nil
}

func cmdRevert(ctx appctx.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: revert <skill-id> <version-number>")
	}

	skillID := args[0]
	versionNum := args[1]

	var skill types.Skill
	if err := ctx.Post(fmt.Sprintf("/skills/%s/revert/%s", skillID, versionNum), struct{}{}, &skill); err != nil {
		return fmt.Errorf("failed to revert: %w", err)
	}

	fmt.Printf("Successfully reverted to version %s\n", versionNum)
	fmt.Printf("Current version: %d\n", skill.Version)
	return nil
}
