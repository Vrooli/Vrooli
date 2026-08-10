// Package topics provides CLI commands for topic management.
//
// DOC: docs/reference/cli-commands.md#topics
package topics

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"prompt-manager/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Topic represents a topic from the API.
type Topic struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	ParentTopicID *string  `json:"parentTopicId,omitempty"`
	Skills        []string `json:"skills,omitempty"`
	Icon          string   `json:"icon,omitempty"`
	Status        string   `json:"status"`
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt"`
}

// AccumulatedSkillsResponse is the response for accumulated skills.
type AccumulatedSkillsResponse struct {
	TopicID  string   `json:"topicId"`
	Ancestry []string `json:"ancestry"`
	Skills   []string `json:"skills"`
}

// MatchRequest is the request for topic matching.
type MatchRequest struct {
	Queries []string `json:"queries"`
	Limit   int      `json:"limit,omitempty"`
}

// MatchResponse is the response for topic matching.
type MatchResponse struct {
	Topics []MatchedTopic `json:"topics"`
	Skills []string       `json:"skills"`
	Method string         `json:"method"`
}

// MatchedTopic represents a matched topic.
type MatchedTopic struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Description  string  `json:"description,omitempty"`
	Score        float64 `json:"score"`
	ScorePercent int     `json:"scorePercent"`
}

// Commands returns the topic command group.
func Commands(ctx appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Topics",
		Commands: []cliapp.Command{
			{
				Name:        "topic",
				Aliases:     []string{"topics"},
				NeedsAPI:    true,
				Description: "Manage topics (list|show|create|update|delete|skills|search|tree)",
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
	case "skills":
		return cmdSkills(ctx, subArgs)
	case "search":
		return cmdSearch(ctx, subArgs)
	case "tree":
		return cmdTree(ctx, subArgs)
	default:
		return fmt.Errorf("unknown subcommand: %s\n\n%s", subcommand, usageText())
	}
}

func printUsage() error {
	fmt.Println(usageText())
	return nil
}

func usageText() string {
	return `Usage: prompt-manager topic <command> [options]

Commands:
  list, ls           List all topics
  show, get <id>     Show topic details
  create, add        Create a new topic
  update, edit <id>  Update a topic
  delete, rm <id>    Delete a topic
  skills <id>        Show accumulated skills (topic + ancestors)
  search "t1" "t2"   Search topics and return accumulated skills
  tree               Show topic hierarchy tree`
}

func cmdList(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("topic list", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var topics []Topic
	if err := ctx.Get("/topics", &topics); err != nil {
		return fmt.Errorf("listing topics: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(topics)
	}

	if len(topics) == 0 {
		fmt.Println("No topics found")
		return nil
	}

	fmt.Printf("Topics (%d):\n", len(topics))
	for _, t := range topics {
		parent := ""
		if t.ParentTopicID != nil && *t.ParentTopicID != "" {
			parent = fmt.Sprintf(" (parent: %s)", *t.ParentTopicID)
		}
		skills := ""
		if len(t.Skills) > 0 {
			skills = fmt.Sprintf(" [%d skills]", len(t.Skills))
		}
		fmt.Printf("  %s - %s%s%s\n", t.ID, t.Name, parent, skills)
	}
	return nil
}

func cmdShow(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("topic show", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: topic show <id>")
	}

	id := fs.Arg(0)
	var topic Topic
	if err := ctx.Get("/topics/"+url.PathEscape(id), &topic); err != nil {
		return fmt.Errorf("topic not found: %s", id)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(topic)
	}

	fmt.Printf("ID:          %s\n", topic.ID)
	fmt.Printf("Name:        %s\n", topic.Name)
	if topic.Description != "" {
		fmt.Printf("Description: %s\n", topic.Description)
	}
	if topic.ParentTopicID != nil && *topic.ParentTopicID != "" {
		fmt.Printf("Parent:      %s\n", *topic.ParentTopicID)
	}
	if len(topic.Skills) > 0 {
		fmt.Printf("Skills:      %s\n", strings.Join(topic.Skills, ", "))
	}
	fmt.Printf("Status:      %s\n", topic.Status)
	return nil
}

func cmdCreate(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("topic create", flag.ContinueOnError)
	id := fs.String("id", "", "Topic ID (auto-generated from name if empty)")
	name := fs.String("name", "", "Topic name (required)")
	description := fs.String("description", "", "Topic description")
	parent := fs.String("parent", "", "Parent topic ID")
	skillsStr := fs.String("skills", "", "Comma-separated skill IDs")
	icon := fs.String("icon", "", "Icon identifier")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if *name == "" {
		return fmt.Errorf("--name is required")
	}

	req := map[string]interface{}{
		"name": *name,
	}
	if *id != "" {
		req["id"] = *id
	}
	if *description != "" {
		req["description"] = *description
	}
	if *parent != "" {
		req["parentTopicId"] = *parent
	}
	if *skillsStr != "" {
		req["skills"] = strings.Split(*skillsStr, ",")
	}
	if *icon != "" {
		req["icon"] = *icon
	}

	var topic Topic
	if err := ctx.Post("/topics", req, &topic); err != nil {
		return fmt.Errorf("creating topic: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(topic)
	}

	fmt.Printf("Created topic: %s (%s)\n", topic.Name, topic.ID)
	return nil
}

func cmdUpdate(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("topic update", flag.ContinueOnError)
	name := fs.String("name", "", "New name")
	description := fs.String("description", "", "New description")
	parent := fs.String("parent", "", "New parent topic ID (empty to clear)")
	skillsStr := fs.String("skills", "", "Comma-separated skill IDs (replaces all)")
	icon := fs.String("icon", "", "New icon")
	status := fs.String("status", "", "New status")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: topic update <id> [--name ...] [--skills ...]")
	}

	id := fs.Arg(0)
	req := make(map[string]interface{})
	if *name != "" {
		req["name"] = *name
	}
	if *description != "" {
		req["description"] = *description
	}
	if fs.Lookup("parent").Value.String() != "" || *parent != "" {
		req["parentTopicId"] = *parent
	}
	if *skillsStr != "" {
		req["skills"] = strings.Split(*skillsStr, ",")
	}
	if *icon != "" {
		req["icon"] = *icon
	}
	if *status != "" {
		req["status"] = *status
	}

	var topic Topic
	if err := ctx.Put("/topics/"+url.PathEscape(id), req, &topic); err != nil {
		return fmt.Errorf("updating topic: %w", err)
	}

	fmt.Printf("Updated topic: %s (%s)\n", topic.Name, topic.ID)
	return nil
}

func cmdDelete(ctx appctx.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: topic delete <id>")
	}
	id := args[0]

	if err := ctx.Delete("/topics/" + url.PathEscape(id)); err != nil {
		return fmt.Errorf("deleting topic: %w", err)
	}

	fmt.Printf("Deleted topic: %s\n", id)
	return nil
}

func cmdSkills(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("topic skills", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: topic skills <id>")
	}

	id := fs.Arg(0)
	var resp AccumulatedSkillsResponse
	if err := ctx.Get("/topics/"+url.PathEscape(id)+"/skills", &resp); err != nil {
		return fmt.Errorf("getting accumulated skills: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	fmt.Printf("Topic: %s\n", resp.TopicID)
	if len(resp.Ancestry) > 0 {
		fmt.Printf("Ancestry: %s\n", strings.Join(resp.Ancestry, " → "))
	}
	if len(resp.Skills) == 0 {
		fmt.Println("No accumulated skills")
	} else {
		fmt.Printf("Accumulated Skills (%d):\n", len(resp.Skills))
		for _, s := range resp.Skills {
			fmt.Printf("  %s\n", s)
		}
	}
	return nil
}

func cmdSearch(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("topic search", flag.ContinueOnError)
	limit := fs.Int("limit", 5, "Maximum number of results")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return errors.New(`usage: topic search "term1" "term2" [...]`)
	}

	// Each positional arg is a separate search query
	queries := fs.Args()

	req := MatchRequest{
		Queries: queries,
		Limit:   *limit,
	}

	var resp MatchResponse
	if err := ctx.Post("/topics/match", req, &resp); err != nil {
		return fmt.Errorf("searching topics: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if len(resp.Topics) == 0 {
		fmt.Printf("No topics found for: %s\n", strings.Join(queries, ", "))
		return nil
	}

	fmt.Printf("Matched Topics (%d, %s search):\n", len(resp.Topics), resp.Method)
	for _, t := range resp.Topics {
		fmt.Printf("  %s - %s (%d%%)\n", t.ID, t.Name, t.ScorePercent)
		if t.Description != "" {
			fmt.Printf("    → %s\n", truncate(t.Description, 80))
		}
	}

	if len(resp.Skills) > 0 {
		fmt.Printf("\nAccumulated Skills (%d):\n", len(resp.Skills))
		for _, s := range resp.Skills {
			fmt.Printf("  %s\n", s)
		}
	}

	return nil
}

func cmdTree(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("topic tree", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var topics []Topic
	if err := ctx.Get("/topics", &topics); err != nil {
		return fmt.Errorf("listing topics: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(topics)
	}

	if len(topics) == 0 {
		fmt.Println("No topics found")
		return nil
	}

	// Build parent-children map
	children := make(map[string][]Topic)
	var roots []Topic
	for _, t := range topics {
		if t.ParentTopicID == nil || *t.ParentTopicID == "" {
			roots = append(roots, t)
		} else {
			children[*t.ParentTopicID] = append(children[*t.ParentTopicID], t)
		}
	}

	fmt.Println("Topic Tree:")
	for _, root := range roots {
		printTreeNode(root, children, "", true)
	}

	return nil
}

func printTreeNode(topic Topic, children map[string][]Topic, prefix string, isLast bool) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	skills := ""
	if len(topic.Skills) > 0 {
		skills = fmt.Sprintf(" [%d skills]", len(topic.Skills))
	}

	fmt.Printf("%s%s%s%s\n", prefix, connector, topic.Name, skills)

	childPrefix := prefix + "│   "
	if isLast {
		childPrefix = prefix + "    "
	}

	kids := children[topic.ID]
	for i, child := range kids {
		printTreeNode(child, children, childPrefix, i == len(kids)-1)
	}
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
