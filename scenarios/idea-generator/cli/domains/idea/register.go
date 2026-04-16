package idea

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"idea-generator/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `idea` subcommand group covering listing, AI-backed
// generation, refinement, and semantic search against /api/ideas and /api/search.
// The API owns all LLM orchestration, DB persistence, and vector storage; this
// package is a thin wrapper that shapes request bodies from CLI flags.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "idea",
		Description: "List, generate, refine, and search ideas",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List ideas (optionally filtered by campaign)", Run: func(args []string) error { return runList(core, args) }},
			{Name: "generate", Aliases: []string{"create"}, Description: "Generate ideas for a campaign via the API's LLM pipeline", Run: func(args []string) error { return runGenerate(core, args) }},
			{Name: "refine", Description: "Refine an existing idea with feedback", Run: func(args []string) error { return runRefine(core, args) }},
			{Name: "search", Description: "Semantic search across stored ideas", Run: func(args []string) error { return runSearch(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("idea list")
	campaign := fs.String("campaign", "", "Filter ideas by campaign ID")
	limit := fs.Int("limit", 0, "Maximum number of ideas to return (server-side cap applies)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	params := map[string]string{
		"campaign_id": *campaign,
	}
	if *limit > 0 {
		params["limit"] = strconv.Itoa(*limit)
	}
	query := support.BuildQuery(params)

	body, err := core.Get("/ideas", query)
	if err != nil {
		return err
	}
	var ideas []support.Idea
	if err := support.Decode(body, &ideas); err != nil {
		return err
	}

	summary := []string{fmt.Sprintf("Ideas: %d", len(ideas))}
	if strings.TrimSpace(*campaign) != "" {
		summary = append(summary, fmt.Sprintf("Campaign: %s", *campaign))
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Ideas",
		Results:        ideaRows(ideas),
		RetrievalHints: []string{
			fmt.Sprintf("%s idea refine --id <idea-id> --feedback \"...\"", support.CLIName),
			fmt.Sprintf("%s idea search --query \"...\"", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGenerate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("idea generate")
	campaign := fs.String("campaign", "", "Campaign ID the generated ideas belong to (required)")
	prompt := fs.String("prompt", "", "User prompt guiding generation")
	count := fs.Int("count", 1, "Number of ideas to generate (1-10)")
	user := fs.String("user", "", "Optional user ID to attribute the generation to")
	bodyFile := fs.String("body-file", "", "Path to JSON request body, or '-' for stdin. Overrides other flags if set.")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if strings.TrimSpace(*campaign) == "" {
			return fmt.Errorf("--campaign is required (or supply --body-file)")
		}
		built := map[string]interface{}{
			"campaign_id": strings.TrimSpace(*campaign),
			"prompt":      strings.TrimSpace(*prompt),
			"count":       *count,
		}
		if strings.TrimSpace(*user) != "" {
			built["user_id"] = strings.TrimSpace(*user)
		}
		payload = built
	}

	body, err := core.Request("POST", "/ideas", nil, payload)
	if err != nil {
		return err
	}

	var resp struct {
		Success bool `json:"success"`
		Ideas   []struct {
			Title               string   `json:"title"`
			Description         string   `json:"description"`
			Category            string   `json:"category,omitempty"`
			Tags                []string `json:"tags,omitempty"`
			ImplementationNotes string   `json:"implementation_notes,omitempty"`
		} `json:"ideas"`
		Message string `json:"message,omitempty"`
		Error   string `json:"error,omitempty"`
	}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	if !resp.Success {
		msg := resp.Error
		if msg == "" {
			msg = "idea generation failed"
		}
		return fmt.Errorf("%s", msg)
	}

	results := []string{}
	if resp.Message != "" {
		results = append(results, resp.Message)
	}
	for i, idea := range resp.Ideas {
		results = append(results, fmt.Sprintf("--- Idea %d ---", i+1))
		results = append(results, fmt.Sprintf("Title: %s", idea.Title))
		if idea.Category != "" {
			results = append(results, fmt.Sprintf("Category: %s", idea.Category))
		}
		if len(idea.Tags) > 0 {
			results = append(results, fmt.Sprintf("Tags: %s", strings.Join(idea.Tags, ", ")))
		}
		if idea.Description != "" {
			results = append(results, fmt.Sprintf("Description: %s", idea.Description))
		}
		if idea.ImplementationNotes != "" {
			results = append(results, fmt.Sprintf("Implementation: %s", idea.ImplementationNotes))
		}
	}
	if len(resp.Ideas) == 0 {
		results = append(results, "(no ideas returned)")
	}

	report := cliapp.MutationReport{
		Result:  results,
		Changes: []string{fmt.Sprintf("Generated %d idea(s)", len(resp.Ideas))},
		NextCommand: []string{
			fmt.Sprintf("%s idea list --campaign %s", support.CLIName, strings.TrimSpace(*campaign)),
			fmt.Sprintf("%s idea search --query \"...\"", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runRefine(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("idea refine")
	id := fs.String("id", "", "Idea ID to refine (required)")
	feedback := fs.String("feedback", "", "Refinement feedback text (required)")
	user := fs.String("user", "", "Optional user ID")
	bodyFile := fs.String("body-file", "", "Path to JSON request body, or '-' for stdin. Overrides other flags if set.")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if strings.TrimSpace(*id) == "" || strings.TrimSpace(*feedback) == "" {
			return fmt.Errorf("--id and --feedback are required (or supply --body-file)")
		}
		built := map[string]interface{}{
			"idea_id":    strings.TrimSpace(*id),
			"refinement": strings.TrimSpace(*feedback),
		}
		if strings.TrimSpace(*user) != "" {
			built["user_id"] = strings.TrimSpace(*user)
		}
		payload = built
	}

	body, err := core.Request("POST", "/ideas/refine", nil, payload)
	if err != nil {
		return err
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		var resp map[string]interface{}
		if err := support.Decode(body, &resp); err == nil {
			if msg, ok := resp["message"].(string); ok {
				message = msg
			}
		}
	}
	if message == "" {
		message = "Idea refinement submitted"
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     []string{fmt.Sprintf("Idea %s refined", support.ShortID(strings.TrimSpace(*id)))},
		NextCommand: []string{fmt.Sprintf("%s idea list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runSearch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("idea search")
	query := fs.String("query", "", "Natural-language search query (required)")
	campaign := fs.String("campaign", "", "Optional campaign ID filter")
	limit := fs.Int("limit", 0, "Maximum number of results (server-side cap applies)")
	bodyFile := fs.String("body-file", "", "Path to JSON request body, or '-' for stdin. Overrides other flags if set.")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if strings.TrimSpace(*query) == "" {
			return fmt.Errorf("--query is required (or supply --body-file)")
		}
		built := map[string]interface{}{
			"query": strings.TrimSpace(*query),
		}
		if strings.TrimSpace(*campaign) != "" {
			built["campaign_id"] = strings.TrimSpace(*campaign)
		}
		if *limit > 0 {
			built["limit"] = *limit
		}
		payload = built
	}

	body, err := core.Request("POST", "/search", nil, payload)
	if err != nil {
		return err
	}

	var results []struct {
		ID       string   `json:"id"`
		Title    string   `json:"title"`
		Content  string   `json:"content"`
		Score    float64  `json:"score"`
		Category string   `json:"category,omitempty"`
		Tags     []string `json:"tags,omitempty"`
	}
	if err := support.Decode(body, &results); err != nil {
		return err
	}

	rows := make([]string, 0, len(results))
	for _, r := range results {
		line := fmt.Sprintf("%s (%s) | score=%.3f", r.Title, support.ShortID(r.ID), r.Score)
		if r.Category != "" {
			line += fmt.Sprintf(" | %s", r.Category)
		}
		if len(r.Tags) > 0 {
			line += fmt.Sprintf(" | tags=%s", strings.Join(r.Tags, ","))
		}
		rows = append(rows, line)
	}
	if len(rows) == 0 {
		rows = []string{"(no matches)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Matches: %d", len(results))},
		ResultsHeading: "Results",
		Results:        rows,
		RetrievalHints: []string{
			fmt.Sprintf("%s idea refine --id <id> --feedback \"...\"", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func ideaRows(ideas []support.Idea) []string {
	if len(ideas) == 0 {
		return []string{"No ideas"}
	}
	rows := make([]string, 0, len(ideas))
	for _, i := range ideas {
		line := fmt.Sprintf("%s (%s) | status=%s", i.Title, support.ShortID(i.ID), i.Status)
		if i.CampaignID != "" {
			line += fmt.Sprintf(" | campaign=%s", support.ShortID(i.CampaignID))
		}
		line += fmt.Sprintf(" | updated=%s", support.FormatTimeValue(i.UpdatedAt))
		rows = append(rows, line)
	}
	return rows
}
