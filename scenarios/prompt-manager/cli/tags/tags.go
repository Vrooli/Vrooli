// Package tags provides CLI commands for tag management.
//
// DOC: docs/reference/cli-commands.md#tags
package tags

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"prompt-manager/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Tag represents a tag from the API
type Tag struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Color       *string `json:"color,omitempty"`
	Description *string `json:"description,omitempty"`
}

// CreateTagRequest is the request body for creating a tag
type CreateTagRequest struct {
	Name        string  `json:"name"`
	Color       *string `json:"color,omitempty"`
	Description *string `json:"description,omitempty"`
}

// Commands returns the tag command groups using noun-verb pattern.
func Commands(ctx appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Tags",
		Commands: []cliapp.Command{
			{
				Name:        "tag",
				Aliases:     []string{"tags", "t"},
				NeedsAPI:    true,
				Description: "Manage tags (list|create)",
				Run: func(args []string) error {
					return route(ctx, args)
				},
			},
		},
	}
}

// route dispatches to the appropriate subcommand.
func route(ctx appctx.Context, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "list", "ls":
		return cmdList(ctx, subArgs)
	case "create", "add":
		return cmdCreate(ctx, subArgs)
	default:
		return fmt.Errorf("unknown subcommand: %s\n\n%s", subcommand, usageText())
	}
}

func printUsage() error {
	fmt.Println(usageText())
	return nil
}

func usageText() string {
	return `Usage: prompt-manager tag <subcommand> [args]

Subcommands:
  list, ls              List all tags
  create, add <name>    Create a new tag`
}

func cmdList(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var tags []Tag
	if err := ctx.Get("/tags", &tags); err != nil {
		return fmt.Errorf("failed to list tags: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(tags)
	}

	if len(tags) == 0 {
		fmt.Println("No tags found")
		return nil
	}

	fmt.Println("Tags:")
	for _, t := range tags {
		color := ""
		if t.Color != nil {
			color = fmt.Sprintf(" (%s)", *t.Color)
		}
		desc := ""
		if t.Description != nil && *t.Description != "" {
			desc = fmt.Sprintf(" - %s", *t.Description)
		}
		fmt.Printf("  %s%s%s [%s]\n", t.Name, color, desc, t.ID)
	}
	return nil
}

func cmdCreate(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	color := fs.String("color", "", "Tag color (hex, e.g., #FF5733)")
	description := fs.String("description", "", "Tag description")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: tag create <name> [--color=#RRGGBB] [--description=...]")
	}
	name := fs.Arg(0)

	req := CreateTagRequest{
		Name: name,
	}
	if *color != "" {
		req.Color = color
	}
	if *description != "" {
		req.Description = description
	}

	var tag Tag
	if err := ctx.Post("/tags", req, &tag); err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(tag)
	}

	fmt.Printf("Created tag: %s [%s]\n", tag.Name, tag.ID)
	return nil
}
