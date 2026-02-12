// Package graph provides CLI commands for the relationship graph.
package graph

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"

	"prompt-manager/cli/internal/appctx"
)

// --- API response types (mirroring api/graph/models.go) ---

type node struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	Label       string   `json:"label"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type edge struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Kind       string `json:"kind"`
	SourceFile string `json:"sourceFile,omitempty"`
	LineNumber int    `json:"lineNumber,omitempty"`
}

type healthScore struct {
	NodeID  string             `json:"nodeId"`
	Score   float64            `json:"score"`
	Factors map[string]float64 `json:"factors"`
}

type graph struct {
	Nodes        []node        `json:"nodes"`
	Edges        []edge        `json:"edges"`
	HealthScores []healthScore `json:"healthScores,omitempty"`
}

type graphIndex struct {
	GeneratedAt string `json:"generatedAt"`
	Graph       graph  `json:"graph"`
}

// Commands returns the graph command group.
func Commands(ctx appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Graph",
		Commands: []cliapp.Command{
			{
				Name:        "graph",
				Aliases:     []string{"g"},
				NeedsAPI:    true,
				Description: "Relationship graph (show|dump|node|regenerate|orphaned-skills|skillless-agents|empty-teams|unaffiliated-agents|cliless-skills|popular|circular-refs|health)",
				Run: func(args []string) error {
					return route(ctx, args)
				},
			},
		},
	}
}

func route(ctx appctx.Context, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "show":
		return cmdShow(ctx, subArgs)
	case "dump":
		return cmdDump(ctx, subArgs)
	case "node":
		return cmdNode(ctx, subArgs)
	case "regenerate", "regen":
		return cmdRegenerate(ctx, subArgs)
	case "orphaned-skills", "orphans":
		return cmdOrphanedSkills(ctx, subArgs)
	case "skillless-agents", "skillless":
		return cmdSkilllessAgents(ctx, subArgs)
	case "empty-teams":
		return cmdEmptyTeams(ctx, subArgs)
	case "unaffiliated-agents", "unaffiliated":
		return cmdUnaffiliatedAgents(ctx, subArgs)
	case "cliless-skills", "cliless":
		return cmdCLIlessSkills(ctx, subArgs)
	case "popular":
		return cmdPopular(ctx, subArgs)
	case "circular-refs", "cycles":
		return cmdCircularRefs(ctx, subArgs)
	case "health":
		return cmdHealth(ctx, subArgs)
	default:
		return fmt.Errorf("unknown subcommand: %s\n\n%s", sub, usageText())
	}
}

func printUsage() error {
	fmt.Println(usageText())
	return nil
}

func usageText() string {
	return `Usage: prompt-manager graph <subcommand> [args]

Subcommands:
  show                                Summary (counts by type)
  dump [--json]                       Full graph data
  node <id> [--json]                  Node with connections
  regenerate                          Force rebuild
  orphaned-skills [--limit N]         Skills not referenced by agents
  skillless-agents [--limit N]        Agents not referencing skills
  empty-teams                         Teams with no members
  unaffiliated-agents                 Agents not in any teams
  cliless-skills [--limit N]          Skills not referencing CLIs
  popular [--limit 10] [--type X]     Most referenced nodes
  circular-refs                       Circular reference detection
  health [--type X | <id>]            Health scores`
}

// cmdShow prints a summary of graph counts by type.
func cmdShow(ctx appctx.Context, _ []string) error {
	var idx graphIndex
	if err := ctx.Get("/graph", &idx); err != nil {
		return fmt.Errorf("failed to fetch graph: %w", err)
	}

	g := idx.Graph
	counts := map[string]int{
		"team":  0,
		"agent": 0,
		"skill": 0,
		"cli":   0,
	}
	for _, n := range g.Nodes {
		counts[n.Type]++
	}

	edgeKinds := make(map[string]int)
	for _, e := range g.Edges {
		edgeKinds[e.Kind]++
	}

	fmt.Printf("Graph Summary (generated %s)\n", idx.GeneratedAt)
	fmt.Println()
	fmt.Println("Nodes:")
	fmt.Printf("  Teams:  %d\n", counts["team"])
	fmt.Printf("  Agents: %d\n", counts["agent"])
	fmt.Printf("  Skills: %d\n", counts["skill"])
	fmt.Printf("  CLIs:   %d\n", counts["cli"])
	fmt.Printf("  Total:  %d\n", len(g.Nodes))
	fmt.Println()
	fmt.Println("Edges:")
	for kind, count := range edgeKinds {
		fmt.Printf("  %s: %d\n", kind, count)
	}
	fmt.Printf("  Total: %d\n", len(g.Edges))

	if len(g.HealthScores) > 0 {
		var sum float64
		for _, hs := range g.HealthScores {
			sum += hs.Score
		}
		avg := sum / float64(len(g.HealthScores))
		fmt.Println()
		fmt.Printf("Health: %.2f avg across %d scored nodes\n", avg, len(g.HealthScores))
	}

	return nil
}

// cmdDump prints the full graph data.
func cmdDump(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("dump", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON (default: human-readable)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var idx graphIndex
	if err := ctx.Get("/graph", &idx); err != nil {
		return fmt.Errorf("failed to fetch graph: %w", err)
	}

	if *jsonOut {
		return encodeJSON(idx)
	}

	// Human-readable output
	fmt.Printf("Generated: %s\n\n", idx.GeneratedAt)

	fmt.Println("=== Nodes ===")
	for _, n := range idx.Graph.Nodes {
		tags := ""
		if len(n.Tags) > 0 {
			tags = " [" + strings.Join(n.Tags, ", ") + "]"
		}
		fmt.Printf("  %-8s %-30s %s%s\n", n.Type, n.ID, n.Label, tags)
	}

	fmt.Println()
	fmt.Println("=== Edges ===")
	for _, e := range idx.Graph.Edges {
		loc := ""
		if e.SourceFile != "" {
			loc = fmt.Sprintf(" (%s", e.SourceFile)
			if e.LineNumber > 0 {
				loc += fmt.Sprintf(":%d", e.LineNumber)
			}
			loc += ")"
		}
		fmt.Printf("  %s -[%s]-> %s%s\n", e.From, e.Kind, e.To, loc)
	}

	if len(idx.Graph.HealthScores) > 0 {
		fmt.Println()
		fmt.Println("=== Health Scores ===")
		for _, hs := range idx.Graph.HealthScores {
			fmt.Printf("  %-30s %.2f\n", hs.NodeID, hs.Score)
		}
	}

	return nil
}

// cmdNode shows a single node with its adjacent edges.
func cmdNode(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("node", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: graph node <id> [--json]")
	}
	id := fs.Arg(0)

	// Fetch edges for this node
	var edges []edge
	if err := ctx.Get(fmt.Sprintf("/graph/nodes/%s/edges", id), &edges); err != nil {
		return fmt.Errorf("failed to fetch node edges: %w", err)
	}

	// Also fetch the full graph to find the node details
	var idx graphIndex
	if err := ctx.Get("/graph", &idx); err != nil {
		return fmt.Errorf("failed to fetch graph: %w", err)
	}

	// Find the node
	var found *node
	for i := range idx.Graph.Nodes {
		if idx.Graph.Nodes[i].ID == id {
			found = &idx.Graph.Nodes[i]
			break
		}
	}

	if found == nil {
		return fmt.Errorf("node not found: %s", id)
	}

	type nodeDetail struct {
		Node          node         `json:"node"`
		AdjacentEdges []edge       `json:"adjacentEdges"`
		HealthScore   *healthScore `json:"healthScore,omitempty"`
	}

	detail := nodeDetail{
		Node:          *found,
		AdjacentEdges: edges,
	}

	// Find health score
	for i := range idx.Graph.HealthScores {
		if idx.Graph.HealthScores[i].NodeID == id {
			detail.HealthScore = &idx.Graph.HealthScores[i]
			break
		}
	}

	if *jsonOut {
		return encodeJSON(detail)
	}

	fmt.Printf("ID:     %s\n", found.ID)
	fmt.Printf("Type:   %s\n", found.Type)
	fmt.Printf("Label:  %s\n", found.Label)
	if found.Description != "" {
		fmt.Printf("Desc:   %s\n", found.Description)
	}
	if found.Status != "" {
		fmt.Printf("Status: %s\n", found.Status)
	}
	if len(found.Tags) > 0 {
		fmt.Printf("Tags:   %s\n", strings.Join(found.Tags, ", "))
	}

	if detail.HealthScore != nil {
		fmt.Printf("Health: %.2f\n", detail.HealthScore.Score)
		if len(detail.HealthScore.Factors) > 0 {
			fmt.Println("  Factors:")
			for k, v := range detail.HealthScore.Factors {
				fmt.Printf("    %s: %.2f\n", k, v)
			}
		}
	}

	// Show inbound and outbound edges
	var inbound, outbound []edge
	for _, e := range edges {
		if e.To == id {
			inbound = append(inbound, e)
		} else {
			outbound = append(outbound, e)
		}
	}

	if len(inbound) > 0 {
		fmt.Println()
		fmt.Printf("Inbound edges (%d):\n", len(inbound))
		for _, e := range inbound {
			fmt.Printf("  %s -[%s]-> (this)\n", e.From, e.Kind)
		}
	}

	if len(outbound) > 0 {
		fmt.Println()
		fmt.Printf("Outbound edges (%d):\n", len(outbound))
		for _, e := range outbound {
			fmt.Printf("  (this) -[%s]-> %s\n", e.Kind, e.To)
		}
	}

	if len(edges) == 0 {
		fmt.Println()
		fmt.Println("No adjacent edges")
	}

	return nil
}

// cmdRegenerate forces a graph rebuild.
func cmdRegenerate(ctx appctx.Context, _ []string) error {
	var idx graphIndex
	if err := ctx.Post("/graph/regenerate", nil, &idx); err != nil {
		return fmt.Errorf("failed to regenerate graph: %w", err)
	}

	fmt.Printf("Graph regenerated at %s\n", idx.GeneratedAt)
	fmt.Printf("  %d nodes, %d edges\n", len(idx.Graph.Nodes), len(idx.Graph.Edges))
	return nil
}

// cmdOrphanedSkills lists skills not referenced by any agent.
func cmdOrphanedSkills(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("orphaned-skills", flag.ContinueOnError)
	limit := fs.Int("limit", 0, "Limit results (0 = all)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var nodes []node
	if err := ctx.Get("/graph/orphans", &nodes); err != nil {
		return fmt.Errorf("failed to fetch orphaned skills: %w", err)
	}

	if *limit > 0 && len(nodes) > *limit {
		nodes = nodes[:*limit]
	}

	if *jsonOut {
		return encodeJSON(nodes)
	}

	return printNodeList("Orphaned skills", nodes)
}

// cmdSkilllessAgents lists agents without skill references.
func cmdSkilllessAgents(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("skillless-agents", flag.ContinueOnError)
	limit := fs.Int("limit", 0, "Limit results (0 = all)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var nodes []node
	if err := ctx.Get("/graph/skillless", &nodes); err != nil {
		return fmt.Errorf("failed to fetch skillless agents: %w", err)
	}

	if *limit > 0 && len(nodes) > *limit {
		nodes = nodes[:*limit]
	}

	if *jsonOut {
		return encodeJSON(nodes)
	}

	return printNodeList("Skillless agents", nodes)
}

// cmdEmptyTeams lists teams with no members.
func cmdEmptyTeams(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("empty-teams", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var nodes []node
	if err := ctx.Get("/graph/empty-teams", &nodes); err != nil {
		return fmt.Errorf("failed to fetch empty teams: %w", err)
	}

	if *jsonOut {
		return encodeJSON(nodes)
	}

	return printNodeList("Empty teams", nodes)
}

// cmdUnaffiliatedAgents lists agents not in any team.
func cmdUnaffiliatedAgents(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("unaffiliated-agents", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var nodes []node
	if err := ctx.Get("/graph/unaffiliated", &nodes); err != nil {
		return fmt.Errorf("failed to fetch unaffiliated agents: %w", err)
	}

	if *jsonOut {
		return encodeJSON(nodes)
	}

	return printNodeList("Unaffiliated agents", nodes)
}

// cmdCLIlessSkills lists skills that don't reference any CLIs.
// Note: The API doesn't have a dedicated endpoint for this, so we fetch the
// full graph and compute locally using the same logic as queries.go.
func cmdCLIlessSkills(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("cliless-skills", flag.ContinueOnError)
	limit := fs.Int("limit", 0, "Limit results (0 = all)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var idx graphIndex
	if err := ctx.Get("/graph", &idx); err != nil {
		return fmt.Errorf("failed to fetch graph: %w", err)
	}

	// Replicate CLIlessSkills logic from queries.go
	hasCLI := make(map[string]bool)
	for _, e := range idx.Graph.Edges {
		if e.Kind == "code-usage" {
			hasCLI[e.From] = true
		}
	}

	var nodes []node
	for _, n := range idx.Graph.Nodes {
		if n.Type == "skill" && !hasCLI[n.ID] {
			nodes = append(nodes, n)
		}
	}

	if *limit > 0 && len(nodes) > *limit {
		nodes = nodes[:*limit]
	}

	if *jsonOut {
		return encodeJSON(nodes)
	}

	return printNodeList("CLI-less skills", nodes)
}

// cmdPopular lists the most referenced nodes.
func cmdPopular(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("popular", flag.ContinueOnError)
	limit := fs.Int("limit", 10, "Number of results")
	nodeType := fs.String("type", "", "Filter by node type (team|agent|skill|cli)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	q := url.Values{}
	q.Set("limit", fmt.Sprintf("%d", *limit))

	var nodes []node
	if err := ctx.GetWithQuery("/graph/popular", q, &nodes); err != nil {
		return fmt.Errorf("failed to fetch popular nodes: %w", err)
	}

	// Client-side type filter (API doesn't support it)
	if *nodeType != "" {
		var filtered []node
		for _, n := range nodes {
			if n.Type == *nodeType {
				filtered = append(filtered, n)
			}
		}
		nodes = filtered
	}

	if *jsonOut {
		return encodeJSON(nodes)
	}

	return printNodeList("Popular nodes", nodes)
}

// cmdCircularRefs detects circular references.
func cmdCircularRefs(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("circular-refs", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var cycles [][]string
	if err := ctx.Get("/graph/cycles", &cycles); err != nil {
		return fmt.Errorf("failed to fetch circular refs: %w", err)
	}

	if *jsonOut {
		return encodeJSON(cycles)
	}

	if len(cycles) == 0 {
		fmt.Println("No circular references detected")
		return nil
	}

	fmt.Printf("Found %d circular reference(s):\n", len(cycles))
	for i, cycle := range cycles {
		fmt.Printf("  %d. %s\n", i+1, strings.Join(cycle, " -> "))
	}
	return nil
}

// cmdHealth shows health scores.
func cmdHealth(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("health", flag.ContinueOnError)
	nodeType := fs.String("type", "", "Filter by node type (team|agent|skill|cli)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// If a positional argument is provided, treat it as a node ID
	if fs.NArg() > 0 {
		return cmdHealthNode(ctx, fs.Arg(0), *jsonOut)
	}

	var scores []healthScore
	if err := ctx.Get("/graph/health", &scores); err != nil {
		return fmt.Errorf("failed to fetch health scores: %w", err)
	}

	// For type filtering we need the graph to know node types
	if *nodeType != "" {
		var idx graphIndex
		if err := ctx.Get("/graph", &idx); err != nil {
			return fmt.Errorf("failed to fetch graph: %w", err)
		}
		nodeTypes := make(map[string]string)
		for _, n := range idx.Graph.Nodes {
			nodeTypes[n.ID] = n.Type
		}
		var filtered []healthScore
		for _, hs := range scores {
			if nodeTypes[hs.NodeID] == *nodeType {
				filtered = append(filtered, hs)
			}
		}
		scores = filtered
	}

	if *jsonOut {
		return encodeJSON(scores)
	}

	if len(scores) == 0 {
		fmt.Println("No health scores available")
		return nil
	}

	fmt.Printf("Health Scores (%d nodes):\n", len(scores))
	for _, hs := range scores {
		fmt.Printf("  %-30s %.2f\n", hs.NodeID, hs.Score)
	}
	return nil
}

// cmdHealthNode shows health details for a single node.
func cmdHealthNode(ctx appctx.Context, id string, jsonOut bool) error {
	var scores []healthScore
	if err := ctx.Get("/graph/health", &scores); err != nil {
		return fmt.Errorf("failed to fetch health scores: %w", err)
	}

	for _, hs := range scores {
		if hs.NodeID == id {
			if jsonOut {
				return encodeJSON(hs)
			}
			fmt.Printf("Node:   %s\n", hs.NodeID)
			fmt.Printf("Score:  %.2f\n", hs.Score)
			if len(hs.Factors) > 0 {
				fmt.Println("Factors:")
				for k, v := range hs.Factors {
					fmt.Printf("  %-25s %.2f\n", k, v)
				}
			}
			return nil
		}
	}

	return fmt.Errorf("no health score found for node: %s", id)
}

// --- Helpers ---

func encodeJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func printNodeList(title string, nodes []node) error {
	if len(nodes) == 0 {
		fmt.Printf("No %s found\n", strings.ToLower(title))
		return nil
	}

	fmt.Printf("%s (%d):\n", title, len(nodes))
	for _, n := range nodes {
		desc := ""
		if n.Description != "" {
			desc = " - " + truncate(n.Description, 60)
		}
		fmt.Printf("  %-8s %-30s %s%s\n", n.Type, n.ID, n.Label, desc)
	}
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
