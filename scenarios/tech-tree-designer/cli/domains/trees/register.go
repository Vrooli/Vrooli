package trees

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"tech-tree-designer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "trees",
		Description: "List and manage tech trees",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List tech trees", Run: func(args []string) error { return runList(deps, args) }},
			{Name: "create", Description: "Create a new tech tree", Run: func(args []string) error { return runCreate(deps, args) }},
			{Name: "update", Description: "Update a tech tree", Run: func(args []string) error { return runUpdate(deps, args) }},
			{Name: "clone", Description: "Clone an existing tech tree", Run: func(args []string) error { return runClone(deps, args) }},
		},
	}
}

func runList(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("trees list")
	treeType := fs.String("type", "", "Filter by tree type")
	status := fs.String("status", "", "Filter by status")
	includeArchived := fs.Bool("include-archived", false, "Include archived trees")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if strings.TrimSpace(*treeType) != "" {
		query.Set("type", *treeType)
	}
	if strings.TrimSpace(*status) != "" {
		query.Set("status", *status)
	}
	if *includeArchived {
		query.Set("include_archived", "true")
	}
	query = deps.Selector.Append(query)

	body, err := deps.Core.GetRoot("/tech-trees", query)
	if err != nil {
		return err
	}
	var response support.TreeListResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			support.TreeScopeLine(deps.Selector),
			fmt.Sprintf("Trees returned: %d", len(response.Trees)),
		},
		ResultsHeading: "Tech trees",
		Results:        listRows(response.Trees),
		RetrievalHints: []string{
			"tech-tree-designer trees create --name \"New Tree\" --slug new-tree --type experimental",
			"tech-tree-designer --tree <tree-id> overview --verbose",
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("trees create")
	name := fs.String("name", "", "Tree name")
	slug := fs.String("slug", "", "Tree slug")
	description := fs.String("description", "", "Tree description")
	treeType := fs.String("type", "experimental", "Tree type")
	status := fs.String("status", "active", "Tree status")
	version := fs.String("version", "1.0.0", "Tree version")
	parentID := fs.String("parent-id", "", "Parent tree ID")
	active := fs.Bool("active", false, "Mark the created tree as active")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" || strings.TrimSpace(*slug) == "" {
		return fmt.Errorf("name and slug are required")
	}

	body, err := deps.Core.RequestRoot("POST", "/tech-trees", nil, map[string]interface{}{
		"name":           *name,
		"slug":           *slug,
		"description":    *description,
		"tree_type":      *treeType,
		"status":         *status,
		"version":        *version,
		"parent_tree_id": *parentID,
		"is_active":      *active,
	})
	if err != nil {
		return err
	}
	var response support.TreeEnvelope
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Created tree %s (%s).", response.Tree.Name, response.Tree.ID),
		},
		Changes: []string{
			fmt.Sprintf("Slug: %s", response.Tree.Slug),
			fmt.Sprintf("Type: %s", response.Tree.TreeType),
			fmt.Sprintf("Sectors: %d | stages: %d", response.Stats.Sectors, response.Stats.Stages),
		},
		NextCommand: []string{
			fmt.Sprintf("tech-tree-designer --tree %s overview --verbose", response.Tree.ID),
			fmt.Sprintf("tech-tree-designer --tree %s sectors list", response.Tree.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("trees update")
	name := fs.String("name", "", "New tree name")
	slug := fs.String("slug", "", "New tree slug")
	description := fs.String("description", "", "New description")
	treeType := fs.String("type", "", "New tree type")
	status := fs.String("status", "", "New tree status")
	version := fs.String("version", "", "New tree version")
	isActive := fs.String("active", "", "Set active to true or false")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: trees update <tree-id> [flags]")
	}

	payload := map[string]interface{}{}
	setString(payload, "name", *name)
	setString(payload, "slug", *slug)
	setString(payload, "description", *description)
	setString(payload, "tree_type", *treeType)
	setString(payload, "status", *status)
	setString(payload, "version", *version)
	if strings.TrimSpace(*isActive) != "" {
		active, err := strconv.ParseBool(strings.TrimSpace(*isActive))
		if err != nil {
			return fmt.Errorf("--active must be true or false")
		}
		payload["is_active"] = active
	}
	if len(payload) == 0 {
		return fmt.Errorf("at least one field must be updated")
	}

	treeID := fs.Arg(0)
	body, err := deps.Core.RequestRoot("PATCH", "/tech-trees/"+treeID, nil, payload)
	if err != nil {
		return err
	}
	var response support.TreeEnvelope
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Updated tree %s (%s).", response.Tree.Name, response.Tree.ID),
		},
		Changes: []string{
			fmt.Sprintf("Slug: %s", response.Tree.Slug),
			fmt.Sprintf("Status: %s", response.Tree.Status),
			fmt.Sprintf("Active: %t", response.Tree.IsActive),
		},
		NextCommand: []string{
			fmt.Sprintf("tech-tree-designer --tree %s overview --verbose", response.Tree.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runClone(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("trees clone")
	name := fs.String("name", "", "New tree name")
	slug := fs.String("slug", "", "New tree slug")
	description := fs.String("description", "", "Override description")
	treeType := fs.String("type", "", "Override tree type")
	status := fs.String("status", "", "Override status")
	active := fs.String("active", "", "Set active to true or false")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: trees clone <tree-id> [flags]")
	}
	payload := map[string]interface{}{}
	setString(payload, "name", *name)
	setString(payload, "slug", *slug)
	setString(payload, "description", *description)
	setString(payload, "tree_type", *treeType)
	setString(payload, "status", *status)
	if strings.TrimSpace(*active) != "" {
		value, err := strconv.ParseBool(strings.TrimSpace(*active))
		if err != nil {
			return fmt.Errorf("--active must be true or false")
		}
		payload["is_active"] = value
	}
	body, err := deps.Core.RequestRoot("POST", "/tech-trees/"+fs.Arg(0)+"/clone", nil, payload)
	if err != nil {
		return err
	}
	var response support.TreeEnvelope
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Cloned tree to %s (%s).", response.Tree.Name, response.Tree.ID),
		},
		Changes: []string{
			fmt.Sprintf("Slug: %s", response.Tree.Slug),
			fmt.Sprintf("Sectors: %d | stages: %d", response.Stats.Sectors, response.Stats.Stages),
		},
		NextCommand: []string{
			fmt.Sprintf("tech-tree-designer --tree %s overview --verbose", response.Tree.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func listRows(trees []support.TreeSummary) []string {
	if len(trees) == 0 {
		return []string{"No tech trees found."}
	}
	rows := make([]string, 0, len(trees))
	for _, tree := range trees {
		rows = append(rows, fmt.Sprintf("%s | %s | type=%s | status=%s | sectors=%d | stages=%d", tree.Tree.Name, tree.Tree.ID, tree.Tree.TreeType, tree.Tree.Status, tree.SectorCount, tree.StageCount))
	}
	return rows
}

func setString(payload map[string]interface{}, key, value string) {
	if strings.TrimSpace(value) != "" {
		payload[key] = value
	}
}
