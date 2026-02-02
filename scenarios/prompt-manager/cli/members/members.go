// Package members provides CLI commands for member management.
//
// DOC: docs/reference/cli-commands.md#members
package members

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"

	"prompt-manager/cli/internal/appctx"
)

// Member represents a member from the API
type Member struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	BodyColor   string   `json:"bodyColor"`
	HeadColor   string   `json:"headColor"`
	AccentColor string   `json:"accentColor"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

// CreateMemberRequest is the request body for creating a member
type CreateMemberRequest struct {
	ID          string   `json:"id,omitempty"`
	Name        string   `json:"name"`
	BodyColor   string   `json:"bodyColor"`
	HeadColor   string   `json:"headColor"`
	AccentColor string   `json:"accentColor"`
}

// UpdateMemberRequest is the request body for updating a member
type UpdateMemberRequest struct {
	Name        *string  `json:"name,omitempty"`
	BodyColor   *string  `json:"bodyColor,omitempty"`
	HeadColor   *string  `json:"headColor,omitempty"`
	AccentColor *string  `json:"accentColor,omitempty"`
}

// Commands returns the member command groups using noun-verb pattern.
func Commands(ctx appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Members",
		Commands: []cliapp.Command{
			{
				Name:        "member",
				Aliases:     []string{"members", "m"},
				NeedsAPI:    true,
				Description: "Manage members (list|show|create|update|delete)",
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
	case "show", "get":
		return cmdShow(ctx, subArgs)
	case "create", "add":
		return cmdCreate(ctx, subArgs)
	case "update", "edit":
		return cmdUpdate(ctx, subArgs)
	case "delete", "rm":
		return cmdDelete(ctx, subArgs)
	default:
		return fmt.Errorf("unknown subcommand: %s\n\n%s", subcommand, usageText())
	}
}

func printUsage() error {
	fmt.Println(usageText())
	return nil
}

func usageText() string {
	return `Usage: prompt-manager member <subcommand> [args]

Subcommands:
  list, ls              List all members
  show, get <id>        Show member details
  create, add <name>    Create a new member
  update, edit <id>     Update a member
  delete, rm <id>       Delete a member`
}

func cmdList(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var members []Member
	if err := ctx.Get("/members", &members); err != nil {
		return fmt.Errorf("failed to list members: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(members)
	}

	if len(members) == 0 {
		fmt.Println("No members found")
		return nil
	}

	fmt.Println("Members:")
	for _, m := range members {
		fmt.Printf("  %s [%s]\n", m.Name, m.ID)
	}
	return nil
}

func cmdShow(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: member show <id>")
	}
	memberID := fs.Arg(0)

	var member Member
	if err := ctx.Get(fmt.Sprintf("/members/%s", memberID), &member); err != nil {
		return fmt.Errorf("failed to get member: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(member)
	}

	fmt.Printf("Name: %s\n", member.Name)
	fmt.Printf("ID: %s\n", member.ID)
	fmt.Printf("Body Color: %s\n", member.BodyColor)
	fmt.Printf("Head Color: %s\n", member.HeadColor)
	fmt.Printf("Accent Color: %s\n", member.AccentColor)
	fmt.Printf("Created: %s\n", member.CreatedAt)
	fmt.Printf("Updated: %s\n", member.UpdatedAt)
	return nil
}

func cmdCreate(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	bodyColor := fs.String("body-color", "#3B82F6", "Body color (hex)")
	headColor := fs.String("head-color", "#F59E0B", "Head color (hex)")
	accentColor := fs.String("accent-color", "#10B981", "Accent color (hex)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: member create <name> [--body-color=#RRGGBB] [--head-color=#RRGGBB] [--accent-color=#RRGGBB]")
	}
	name := fs.Arg(0)

	req := CreateMemberRequest{
		Name:        name,
		BodyColor:   *bodyColor,
		HeadColor:   *headColor,
		AccentColor: *accentColor,
	}

	var member Member
	if err := ctx.Post("/members", req, &member); err != nil {
		return fmt.Errorf("failed to create member: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(member)
	}

	fmt.Printf("Created member: %s [%s]\n", member.Name, member.ID)
	return nil
}

func cmdUpdate(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	name := fs.String("name", "", "New name")
	bodyColor := fs.String("body-color", "", "Body color (hex)")
	headColor := fs.String("head-color", "", "Head color (hex)")
	accentColor := fs.String("accent-color", "", "Accent color (hex)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: member update <id> [--name=...] [--body-color=#RRGGBB] [--head-color=#RRGGBB] [--accent-color=#RRGGBB]")
	}
	memberID := fs.Arg(0)

	req := UpdateMemberRequest{}
	if *name != "" {
		req.Name = name
	}
	if *bodyColor != "" {
		req.BodyColor = bodyColor
	}
	if *headColor != "" {
		req.HeadColor = headColor
	}
	if *accentColor != "" {
		req.AccentColor = accentColor
	}

	var member Member
	if err := ctx.Put(fmt.Sprintf("/members/%s", memberID), req, &member); err != nil {
		return fmt.Errorf("failed to update member: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(member)
	}

	fmt.Printf("Updated member: %s [%s]\n", member.Name, member.ID)
	return nil
}

func cmdDelete(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	force := fs.Bool("force", false, "Skip confirmation prompt")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: member delete <id> [--force]")
	}
	memberID := fs.Arg(0)

	// Get member info first for confirmation
	var member Member
	if err := ctx.Get(fmt.Sprintf("/members/%s", memberID), &member); err != nil {
		return fmt.Errorf("failed to get member: %w", err)
	}

	if !*force {
		fmt.Printf("Delete member %q (%s)? [y/N]: ", member.Name, memberID)
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	if err := ctx.Delete(fmt.Sprintf("/members/%s", memberID)); err != nil {
		return fmt.Errorf("failed to delete member: %w", err)
	}

	fmt.Printf("Deleted member: %s\n", member.Name)
	return nil
}
