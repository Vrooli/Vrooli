// Package agents provides CLI commands for agent management.
//
// DOC: docs/reference/cli-commands.md#agents
package agents

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

// Agent represents an agent from the API
type Agent struct {
	ID          string      `json:"id"`
	DisplayName string      `json:"displayName"`
	Description string      `json:"description,omitempty"`
	Status      string      `json:"status"`
	Appearance  *Appearance `json:"appearance,omitempty"`
	Skills      []string    `json:"skills,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	CreatedAt   string      `json:"createdAt"`
	UpdatedAt   string      `json:"updatedAt"`
}

// Appearance represents agent visual appearance
type Appearance struct {
	Body   string `json:"body"`
	Head   string `json:"head"`
	Accent string `json:"accent"`
}

// CreateAgentRequest is the request body for creating an agent
type CreateAgentRequest struct {
	ID          string      `json:"id,omitempty"`
	DisplayName string      `json:"displayName"`
	Description string      `json:"description,omitempty"`
	Appearance  *Appearance `json:"appearance,omitempty"`
	Skills      []string    `json:"skills,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
}

// UpdateAgentRequest is the request body for updating an agent
type UpdateAgentRequest struct {
	DisplayName *string     `json:"displayName,omitempty"`
	Description *string     `json:"description,omitempty"`
	Status      *string     `json:"status,omitempty"`
	Appearance  *Appearance `json:"appearance,omitempty"`
	Skills      []string    `json:"skills,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
}

// SoulRequest is the request body for setting SOUL.md content
type SoulRequest struct {
	Content string `json:"content"`
}

// SoulResponse is the response for SOUL.md operations
type SoulResponse struct {
	AgentID string `json:"agentId"`
	Content string `json:"content"`
}

// Commands returns the agent command groups using noun-verb pattern.
func Commands(ctx appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Agents",
		Commands: []cliapp.Command{
			{
				Name:        "agent",
				Aliases:     []string{"agents", "a"},
				NeedsAPI:    true,
				Description: "Manage agents (list|show|create|update|delete|soul)",
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
	case "soul":
		return cmdSoul(ctx, subArgs)
	default:
		return fmt.Errorf("unknown subcommand: %s\n\n%s", subcommand, usageText())
	}
}

func printUsage() error {
	fmt.Println(usageText())
	return nil
}

func usageText() string {
	return `Usage: prompt-manager agent <subcommand> [args]

Subcommands:
  list, ls              List all agents
  show, get <id>        Show agent details
  create, add <name>    Create a new agent
  update, edit <id>     Update an agent
  delete, rm <id>       Delete an agent
  soul <id>             Get/set agent SOUL.md content`
}

func cmdList(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var agents []Agent
	if err := ctx.Get("/agents", &agents); err != nil {
		return fmt.Errorf("failed to list agents: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(agents)
	}

	if len(agents) == 0 {
		fmt.Println("No agents found")
		return nil
	}

	fmt.Println("Agents:")
	for _, a := range agents {
		skillCount := len(a.Skills)
		fmt.Printf("  %s - %d skills [%s] (%s)\n", a.DisplayName, skillCount, a.ID, a.Status)
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
		return fmt.Errorf("usage: agent show <id>")
	}
	agentID := fs.Arg(0)

	var agent Agent
	if err := ctx.Get(fmt.Sprintf("/agents/%s", agentID), &agent); err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(agent)
	}

	fmt.Printf("Name: %s\n", agent.DisplayName)
	fmt.Printf("ID: %s\n", agent.ID)
	fmt.Printf("Status: %s\n", agent.Status)
	if agent.Description != "" {
		fmt.Printf("Description: %s\n", agent.Description)
	}
	if agent.Appearance != nil {
		fmt.Printf("Appearance:\n")
		fmt.Printf("  Body: %s\n", agent.Appearance.Body)
		fmt.Printf("  Head: %s\n", agent.Appearance.Head)
		fmt.Printf("  Accent: %s\n", agent.Appearance.Accent)
	}
	fmt.Printf("Skills: %d assigned\n", len(agent.Skills))
	if len(agent.Skills) > 0 {
		fmt.Printf("  %s\n", strings.Join(agent.Skills, ", "))
	}
	if len(agent.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(agent.Tags, ", "))
	}
	fmt.Printf("Created: %s\n", agent.CreatedAt)
	fmt.Printf("Updated: %s\n", agent.UpdatedAt)
	return nil
}

func cmdCreate(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	description := fs.String("description", "", "Agent description")
	bodyColor := fs.String("body-color", "#3B82F6", "Body color (hex)")
	headColor := fs.String("head-color", "#F59E0B", "Head color (hex)")
	accentColor := fs.String("accent-color", "#10B981", "Accent color (hex)")
	skills := fs.String("skills", "", "Comma-separated skill IDs")
	tags := fs.String("tags", "", "Comma-separated tags")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: agent create <name> [--description=...] [--body-color=#RRGGBB] [--head-color=#RRGGBB] [--accent-color=#RRGGBB] [--skills=id1,id2] [--tags=tag1,tag2]")
	}
	name := fs.Arg(0)

	var skillList []string
	if *skills != "" {
		skillList = strings.Split(*skills, ",")
		for i, s := range skillList {
			skillList[i] = strings.TrimSpace(s)
		}
	}

	var tagList []string
	if *tags != "" {
		tagList = strings.Split(*tags, ",")
		for i, t := range tagList {
			tagList[i] = strings.TrimSpace(t)
		}
	}

	req := CreateAgentRequest{
		DisplayName: name,
		Description: *description,
		Appearance: &Appearance{
			Body:   *bodyColor,
			Head:   *headColor,
			Accent: *accentColor,
		},
		Skills: skillList,
		Tags:   tagList,
	}

	var agent Agent
	if err := ctx.Post("/agents", req, &agent); err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(agent)
	}

	fmt.Printf("Created agent: %s [%s]\n", agent.DisplayName, agent.ID)
	return nil
}

func cmdUpdate(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	name := fs.String("name", "", "New display name")
	description := fs.String("description", "", "New description")
	status := fs.String("status", "", "New status (active|inactive|suspended)")
	bodyColor := fs.String("body-color", "", "Body color (hex)")
	headColor := fs.String("head-color", "", "Head color (hex)")
	accentColor := fs.String("accent-color", "", "Accent color (hex)")
	skills := fs.String("skills", "", "Comma-separated skill IDs (replaces existing)")
	tags := fs.String("tags", "", "Comma-separated tags (replaces existing)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: agent update <id> [--name=...] [--description=...] [--status=...] [--body-color=#RRGGBB] [--head-color=#RRGGBB] [--accent-color=#RRGGBB] [--skills=id1,id2] [--tags=tag1,tag2]")
	}
	agentID := fs.Arg(0)

	req := UpdateAgentRequest{}
	if *name != "" {
		req.DisplayName = name
	}
	if *description != "" {
		req.Description = description
	}
	if *status != "" {
		req.Status = status
	}
	if *bodyColor != "" || *headColor != "" || *accentColor != "" {
		// Get current appearance first
		var agent Agent
		if err := ctx.Get(fmt.Sprintf("/agents/%s", agentID), &agent); err != nil {
			return fmt.Errorf("failed to get agent: %w", err)
		}
		app := &Appearance{}
		if agent.Appearance != nil {
			*app = *agent.Appearance
		}
		if *bodyColor != "" {
			app.Body = *bodyColor
		}
		if *headColor != "" {
			app.Head = *headColor
		}
		if *accentColor != "" {
			app.Accent = *accentColor
		}
		req.Appearance = app
	}
	if *skills != "" {
		skillList := strings.Split(*skills, ",")
		for i, s := range skillList {
			skillList[i] = strings.TrimSpace(s)
		}
		req.Skills = skillList
	}
	if *tags != "" {
		tagList := strings.Split(*tags, ",")
		for i, t := range tagList {
			tagList[i] = strings.TrimSpace(t)
		}
		req.Tags = tagList
	}

	var agent Agent
	if err := ctx.Put(fmt.Sprintf("/agents/%s", agentID), req, &agent); err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(agent)
	}

	fmt.Printf("Updated agent: %s [%s]\n", agent.DisplayName, agent.ID)
	return nil
}

func cmdDelete(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	force := fs.Bool("force", false, "Skip confirmation prompt")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: agent delete <id> [--force]")
	}
	agentID := fs.Arg(0)

	// Get agent info first for confirmation
	var agent Agent
	if err := ctx.Get(fmt.Sprintf("/agents/%s", agentID), &agent); err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	if !*force {
		fmt.Printf("Delete agent %q (%s)? [y/N]: ", agent.DisplayName, agentID)
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	if err := ctx.Delete(fmt.Sprintf("/agents/%s", agentID)); err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	fmt.Printf("Deleted agent: %s\n", agent.DisplayName)
	return nil
}

func cmdSoul(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("soul", flag.ContinueOnError)
	setContent := fs.String("set", "", "Set SOUL.md content from string")
	setFile := fs.String("file", "", "Set SOUL.md content from file")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: agent soul <id> [--set='content'] [--file=path]")
	}
	agentID := fs.Arg(0)

	// If setting content
	if *setContent != "" || *setFile != "" {
		var content string
		if *setFile != "" {
			data, err := os.ReadFile(*setFile)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			content = string(data)
		} else {
			content = *setContent
		}

		req := SoulRequest{Content: content}
		var resp SoulResponse
		if err := ctx.Put(fmt.Sprintf("/agents/%s/soul", agentID), req, &resp); err != nil {
			return fmt.Errorf("failed to set soul: %w", err)
		}

		if *jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(resp)
		}

		fmt.Printf("Updated SOUL.md for agent %s (%d bytes)\n", agentID, len(content))
		return nil
	}

	// Get SOUL.md content
	var resp SoulResponse
	if err := ctx.Get(fmt.Sprintf("/agents/%s/soul", agentID), &resp); err != nil {
		return fmt.Errorf("failed to get soul: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if resp.Content == "" {
		fmt.Println("No SOUL.md content defined for this agent")
		return nil
	}

	fmt.Println(resp.Content)
	return nil
}
