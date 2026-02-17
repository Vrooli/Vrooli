// Package agents provides CLI commands for agent management.
//
// DOC: docs/reference/cli-commands.md#agents
package agents

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"prompt-manager/cli/internal/appctx"
)

// Agent represents an agent from the API
type Agent struct {
	ID          string      `json:"id"`
	DisplayName string      `json:"displayName"`
	Description string      `json:"description,omitempty"`
	Status      string      `json:"status"`
	Appearance  *Appearance `json:"appearance,omitempty"`
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
	Tags        []string    `json:"tags,omitempty"`
}

// UpdateAgentRequest is the request body for updating an agent
type UpdateAgentRequest struct {
	DisplayName *string     `json:"displayName,omitempty"`
	Description *string     `json:"description,omitempty"`
	Status      *string     `json:"status,omitempty"`
	Appearance  *Appearance `json:"appearance,omitempty"`
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
	case "search", "find":
		return cmdSearch(ctx, subArgs)
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
  soul <id>             Get/set agent SOUL.md content
  search, find <query>  Search agents (AI-powered, --text for text-only, --content for file content)`
}

func cmdList(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
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
		fmt.Printf("  %s [%s] (%s)\n", a.DisplayName, a.ID, a.Status)
	}
	return nil
}

func cmdShow(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
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
	tags := fs.String("tags", "", "Comma-separated tags")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: agent create <name> [--description=...] [--body-color=#RRGGBB] [--head-color=#RRGGBB] [--accent-color=#RRGGBB] [--tags=tag1,tag2]")
	}
	name := fs.Arg(0)

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
	tags := fs.String("tags", "", "Comma-separated tags (replaces existing)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: agent update <id> [--name=...] [--description=...] [--status=...] [--body-color=#RRGGBB] [--head-color=#RRGGBB] [--accent-color=#RRGGBB] [--tags=tag1,tag2]")
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
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
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
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
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

// --- Agent search types ---

// AgentSearchResult represents a text search result for agents.
type AgentSearchResult struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"displayName"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status"`
	Tags        []string `json:"tags,omitempty"`
	Score       float64  `json:"score,omitempty"`
	Highlight   string   `json:"highlight,omitempty"`
}

// AgentSearchResponse wraps agent text search results.
type AgentSearchResponse struct {
	Results []AgentSearchResult `json:"results"`
	Total   int                 `json:"total"`
	Query   string              `json:"query"`
}

// AIAgentSearchResult represents an AI search result for agents.
type AIAgentSearchResult struct {
	ID           string   `json:"id"`
	DisplayName  string   `json:"displayName"`
	Description  string   `json:"description,omitempty"`
	Status       string   `json:"status"`
	Tags         []string `json:"tags,omitempty"`
	Score        float64  `json:"score"`
	ScorePercent int      `json:"scorePercent"`
}

// AIAgentSearchResponse wraps agent AI search results.
type AIAgentSearchResponse struct {
	Results []AIAgentSearchResult `json:"results,omitempty"`
	Total   int                   `json:"total"`
	Query   string                `json:"query"`
	Method  string                `json:"method"`
}

// AIAgentSearchRequest is the request body for agent AI search.
type AIAgentSearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

// AgentContentSearchMatch represents a content search match in agent files.
type AgentContentSearchMatch struct {
	AgentID    string `json:"agentId"`
	AgentName  string `json:"agentName"`
	File       string `json:"file"`
	LineNumber int    `json:"lineNumber"`
	Line       string `json:"line"`
}

// AgentContentSearchResponse wraps agent content search results.
type AgentContentSearchResponse struct {
	Matches []AgentContentSearchMatch `json:"matches"`
	Total   int                       `json:"total"`
	Query   string                    `json:"query"`
}

func cmdSearch(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("agent search", flag.ContinueOnError)
	textOnly := fs.Bool("text", false, "Force text-only search (skip AI)")
	contentOnly := fs.Bool("content", false, "Search within agent file contents")
	caseSensitive := fs.Bool("case-sensitive", false, "Case-sensitive content search")
	wholeWord := fs.Bool("whole-word", false, "Whole word matching for content search")
	regex := fs.Bool("regex", false, "Treat query as regex for content search")
	limit := fs.Int("limit", 5, "Maximum number of results")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: agent search <query> [--text] [--content] [--case-sensitive] [--whole-word] [--regex] [--limit=N] [--json]")
	}

	query := strings.Join(fs.Args(), " ")

	if *contentOnly {
		return agentContentSearch(ctx, query, *limit, *caseSensitive, *wholeWord, *regex, *jsonOut)
	}

	if *textOnly {
		return agentTextSearch(ctx, query, *jsonOut)
	}

	return agentAISearch(ctx, query, *limit, *jsonOut)
}

func agentAISearch(ctx appctx.Context, query string, limit int, jsonOut bool) error {
	req := AIAgentSearchRequest{
		Query: query,
		Limit: limit,
	}

	var resp AIAgentSearchResponse
	if err := ctx.Post("/search/agents/ai", req, &resp); err != nil {
		fmt.Fprintln(os.Stderr, "(AI search unavailable, using text search)")
		return agentTextSearch(ctx, query, jsonOut)
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	methodLabel := "AI"
	if resp.Method == "text" {
		methodLabel = "text (AI unavailable)"
	}

	if resp.Total == 0 {
		fmt.Printf("No agents found matching: %s (%s search)\n", query, methodLabel)
		return nil
	}

	fmt.Printf("Agent Search Results (%d found, %s search):\n", resp.Total, methodLabel)
	for _, r := range resp.Results {
		tags := ""
		if len(r.Tags) > 0 {
			tags = " [" + strings.Join(r.Tags, ", ") + "]"
		}
		score := fmt.Sprintf(" (%d%%)", r.ScorePercent)
		fmt.Printf("  %s%s%s (%s) [%s]\n", r.DisplayName, score, tags, r.Status, r.ID)
		if r.Description != "" {
			desc := r.Description
			if len(desc) > 80 {
				desc = desc[:77] + "..."
			}
			fmt.Printf("    → %s\n", desc)
		}
	}
	return nil
}

func agentTextSearch(ctx appctx.Context, query string, jsonOut bool) error {
	params := url.Values{}
	if query != "" {
		params.Set("q", query)
	}

	var resp AgentSearchResponse
	if err := ctx.GetWithQuery("/search/agents", params, &resp); err != nil {
		return fmt.Errorf("agent search failed: %w", err)
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if resp.Total == 0 {
		fmt.Printf("No agents found matching: %s\n", query)
		return nil
	}

	fmt.Printf("Agent Search Results (%d found, text search):\n", resp.Total)
	for _, r := range resp.Results {
		tags := ""
		if len(r.Tags) > 0 {
			tags = " [" + strings.Join(r.Tags, ", ") + "]"
		}
		score := ""
		if r.Score > 0 {
			score = fmt.Sprintf(" (%.1f)", r.Score)
		}
		fmt.Printf("  %s%s%s (%s) [%s]\n", r.DisplayName, score, tags, r.Status, r.ID)
		if r.Highlight != "" {
			highlight := r.Highlight
			if len(highlight) > 80 {
				highlight = highlight[:77] + "..."
			}
			fmt.Printf("    → %s\n", highlight)
		}
	}
	return nil
}

func agentContentSearch(ctx appctx.Context, query string, limit int, caseSensitive, wholeWord, regex, jsonOut bool) error {
	params := url.Values{}
	params.Set("q", query)
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if caseSensitive {
		params.Set("caseSensitive", "true")
	}
	if wholeWord {
		params.Set("wholeWord", "true")
	}
	if regex {
		params.Set("regex", "true")
	}

	var resp AgentContentSearchResponse
	if err := ctx.GetWithQuery("/search/agents/content", params, &resp); err != nil {
		return fmt.Errorf("agent content search failed: %w", err)
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if resp.Total == 0 {
		fmt.Printf("No content matches found for: %s\n", query)
		return nil
	}

	fmt.Printf("Agent Content Matches (%d found):\n", resp.Total)
	for _, m := range resp.Matches {
		line := m.Line
		if len(line) > 120 {
			line = line[:117] + "..."
		}
		fmt.Printf("  %s/%s:%d: %s\n", m.AgentName, m.File, m.LineNumber, line)
	}
	return nil
}
