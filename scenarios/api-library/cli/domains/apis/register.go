package apis

import (
	"fmt"
	"os"
	"strings"

	"api-library/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `apis` subcommand group covering the /apis CRUD surface
// plus per-API adjuncts (notes, configure, versions, tags, status, snippets,
// recipes, usage, analytics, endpoints).
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "apis",
		Description: "Manage API entries and their metadata",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List APIs", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one API with its notes", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "create", Description: "Create a new API (requires --body-file)", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "update", Description: "Update an API (requires --body-file)", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", Description: "Delete an API", Run: func(args []string) error { return runDelete(core, args) }},
			{Name: "endpoints", Description: "List endpoints for an API", Run: func(args []string) error { return runEndpoints(core, args) }},
			{Name: "notes", Description: "List notes for an API", Run: func(args []string) error { return runNotes(core, args) }},
			{Name: "add-note", Description: "Add a note to an API", Run: func(args []string) error { return runAddNote(core, args) }},
			{Name: "configure", Description: "Mark an API as configured", Run: func(args []string) error { return runConfigure(core, args) }},
			{Name: "versions", Description: "List version history for an API", Run: func(args []string) error { return runVersions(core, args) }},
			{Name: "add-version", Description: "Record a new version for an API (requires --body-file)", Run: func(args []string) error { return runAddVersion(core, args) }},
			{Name: "update-status", Description: "Update API lifecycle status", Run: func(args []string) error { return runUpdateStatus(core, args) }},
			{Name: "update-tags", Description: "Replace tags on an API", Run: func(args []string) error { return runUpdateTags(core, args) }},
			{Name: "snippets", Description: "List integration snippets for an API", Run: func(args []string) error { return runSnippets(core, args) }},
			{Name: "add-snippet", Description: "Create a new integration snippet (requires --body-file)", Run: func(args []string) error { return runAddSnippet(core, args) }},
			{Name: "recipes", Description: "List integration recipes for an API", Run: func(args []string) error { return runRecipes(core, args) }},
			{Name: "add-recipe", Description: "Create a new integration recipe (requires --body-file)", Run: func(args []string) error { return runAddRecipe(core, args) }},
			{Name: "usage", Description: "Record usage for an API (requires --body-file)", Run: func(args []string) error { return runUsage(core, args) }},
			{Name: "analytics", Description: "Show analytics for an API", Run: func(args []string) error { return runAnalytics(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apis list")
	category := fs.String("category", "", "Filter by category")
	status := fs.String("status", "", "Filter by status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"category": *category,
		"status":   *status,
	})
	raw, err := core.Get("/apis", query)
	if err != nil {
		return err
	}
	var list []support.API
	if err := support.Decode(raw, &list); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("APIs: %d", len(list))},
		ResultsHeading: "APIs",
		Results:        apiRows(list),
		RetrievalHints: []string{
			fmt.Sprintf("%s apis get <id>", support.CLIName),
			fmt.Sprintf("%s search <query>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apis get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: apis get <api-id>")
	}
	id := fs.Arg(0)

	raw, err := core.Get("/apis/"+id, nil)
	if err != nil {
		return err
	}
	var resp support.APIDetailResponse
	if err := support.Decode(raw, &resp); err != nil {
		return err
	}

	results := detailRows(resp.API)
	if len(resp.Notes) > 0 {
		results = append(results, "", "Notes:")
		for _, n := range resp.Notes {
			results = append(results, fmt.Sprintf("  [%s] %s -- by %s", strings.ToUpper(n.Type), n.Content, orDash(n.CreatedBy)))
		}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("API: %s (%s)", resp.API.Name, orDash(resp.API.Provider))},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s apis notes %s", support.CLIName, resp.API.ID),
			fmt.Sprintf("%s apis snippets %s", support.CLIName, resp.API.ID),
			fmt.Sprintf("%s apis recipes %s", support.CLIName, resp.API.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apis create")
	bodyFile := fs.String("body-file", "", "Path to JSON file describing the new API")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	raw, err := core.Request("POST", "/apis", nil, body)
	if err != nil {
		return err
	}
	var created support.API
	if err := support.Decode(raw, &created); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Created API %s (%s)", created.Name, created.ID)},
		Changes:     []string{"POST /apis"},
		NextCommand: []string{fmt.Sprintf("%s apis get %s", support.CLIName, created.ID)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apis update")
	bodyFile := fs.String("body-file", "", "Path to JSON file with the updated fields")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: apis update <api-id> --body-file PATH")
	}
	id := fs.Arg(0)
	body, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	if _, err := core.Request("PUT", "/apis/"+id, nil, body); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated API %s", id)},
		Changes:     []string{fmt.Sprintf("PUT /apis/%s", id)},
		NextCommand: []string{fmt.Sprintf("%s apis get %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apis delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: apis delete <api-id>")
	}
	id := fs.Arg(0)
	if _, err := core.Request("DELETE", "/apis/"+id, nil, nil); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Deleted API %s", id)},
		Changes: []string{fmt.Sprintf("DELETE /apis/%s", id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runEndpoints(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apis endpoints")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: apis endpoints <api-id>")
	}
	id := fs.Arg(0)
	raw, err := core.Get("/apis/"+id+"/endpoints", nil)
	if err != nil {
		return err
	}
	var endpoints []map[string]interface{}
	if err := support.Decode(raw, &endpoints); err != nil {
		return err
	}

	rows := make([]string, 0, len(endpoints))
	for _, e := range endpoints {
		method := support.RenderValue(e["method"])
		path := support.RenderValue(e["path"])
		rows = append(rows, fmt.Sprintf("%s %s", method, path))
	}
	if len(rows) == 0 {
		rows = []string{"(no endpoints)"}
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Endpoints for %s: %d", id, len(endpoints))},
		ResultsHeading: "Endpoints",
		Results:        rows,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runNotes(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apis notes")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: apis notes <api-id>")
	}
	id := fs.Arg(0)
	raw, err := core.Get("/apis/"+id+"/notes", nil)
	if err != nil {
		return err
	}
	var notes []support.Note
	if err := support.Decode(raw, &notes); err != nil {
		return err
	}
	rows := make([]string, 0, len(notes))
	for _, n := range notes {
		rows = append(rows, fmt.Sprintf("[%s] %s -- by %s", strings.ToUpper(n.Type), n.Content, orDash(n.CreatedBy)))
	}
	if len(rows) == 0 {
		rows = []string{"(no notes)"}
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Notes for %s: %d", id, len(notes))},
		ResultsHeading: "Notes",
		Results:        rows,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runAddNote(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apis add-note")
	noteType := fs.String("type", "tip", "Note type: gotcha|tip|warning|example|success|failure")
	createdBy := fs.String("created-by", "", "Override the created_by field")
	bodyFile := fs.String("body-file", "", "Path to JSON file with a full note body (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: apis add-note <api-id> <content...> | apis add-note <api-id> --body-file PATH")
	}
	id := fs.Arg(0)

	var body interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		body = raw
	} else {
		if fs.NArg() < 2 {
			return fmt.Errorf("note content is required unless --body-file is provided")
		}
		content := strings.Join(fs.Args()[1:], " ")
		payload := map[string]interface{}{
			"content": content,
			"type":    *noteType,
		}
		if strings.TrimSpace(*createdBy) != "" {
			payload["created_by"] = *createdBy
		}
		body = payload
	}

	raw, err := core.Request("POST", "/apis/"+id+"/notes", nil, body)
	if err != nil {
		return err
	}
	var note support.Note
	if err := support.Decode(raw, &note); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Added note %s to API %s", note.ID, id)},
		Changes: []string{fmt.Sprintf("POST /apis/%s/notes", id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runConfigure(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apis configure")
	environment := fs.String("environment", "development", "Environment name")
	notes := fs.String("notes", "", "Configuration notes")
	bodyFile := fs.String("body-file", "", "Path to JSON file with full request body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: apis configure <api-id> [--environment NAME] [--notes TEXT]")
	}
	id := fs.Arg(0)
	var body interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		body = raw
	} else {
		body = map[string]interface{}{
			"environment": *environment,
			"notes":       *notes,
		}
	}
	if _, err := core.Request("POST", "/apis/"+id+"/configure", nil, body); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Marked %s as configured in %s", id, *environment)},
		Changes: []string{fmt.Sprintf("POST /apis/%s/configure", id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runVersions(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apis versions")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: apis versions <api-id>")
	}
	id := fs.Arg(0)
	raw, err := core.Get("/apis/"+id+"/versions", nil)
	if err != nil {
		return err
	}
	var versions []support.APIVersion
	if err := support.Decode(raw, &versions); err != nil {
		return err
	}
	rows := make([]string, 0, len(versions))
	for _, v := range versions {
		breaking := ""
		if v.BreakingChanges {
			breaking = " (breaking)"
		}
		ts := ""
		if v.CreatedAt != nil {
			ts = " @ " + support.FormatTimeValue(*v.CreatedAt)
		}
		rows = append(rows, fmt.Sprintf("%s%s%s -- %s", v.Version, breaking, ts, v.ChangeSummary))
	}
	if len(rows) == 0 {
		rows = []string{"(no version history)"}
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Versions for %s: %d", id, len(versions))},
		ResultsHeading: "Versions",
		Results:        rows,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runAddVersion(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apis add-version")
	bodyFile := fs.String("body-file", "", "Path to JSON file with the new version payload")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: apis add-version <api-id> --body-file PATH")
	}
	id := fs.Arg(0)
	body, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	if _, err := core.Request("POST", "/apis/"+id+"/versions", nil, body); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Recorded new version for %s", id)},
		Changes: []string{fmt.Sprintf("POST /apis/%s/versions", id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdateStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apis update-status")
	status := fs.String("status", "", "New status: active|deprecated|sunset|beta")
	sunsetDate := fs.String("sunset-date", "", "ISO8601 sunset date (optional)")
	bodyFile := fs.String("body-file", "", "Path to JSON file with the status request body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: apis update-status <api-id> --status STATUS [--sunset-date DATE]")
	}
	id := fs.Arg(0)
	var body interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		body = raw
	} else {
		if strings.TrimSpace(*status) == "" {
			return fmt.Errorf("--status is required unless --body-file is used")
		}
		payload := map[string]interface{}{"status": *status}
		if strings.TrimSpace(*sunsetDate) != "" {
			payload["sunset_date"] = *sunsetDate
		}
		body = payload
	}
	raw, err := core.Request("PUT", "/apis/"+id+"/status", nil, body)
	if err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Updated status for %s", id), string(support.DecodeRaw(raw))},
		Changes: []string{fmt.Sprintf("PUT /apis/%s/status", id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdateTags(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apis update-tags")
	tags := fs.String("tags", "", "Comma-separated tag list (replaces all tags)")
	bodyFile := fs.String("body-file", "", "Path to JSON file with the tags request body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: apis update-tags <api-id> --tags a,b,c")
	}
	id := fs.Arg(0)
	var body interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		body = raw
	} else {
		body = map[string]interface{}{"tags": splitCSV(*tags)}
	}
	if _, err := core.Request("PUT", "/apis/"+id+"/tags", nil, body); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Updated tags for %s", id)},
		Changes: []string{fmt.Sprintf("PUT /apis/%s/tags", id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runSnippets(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apis snippets")
	language := fs.String("language", "", "Filter by language")
	framework := fs.String("framework", "", "Filter by framework")
	snippetType := fs.String("type", "", "Filter by snippet type")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: apis snippets <api-id> [--language L] [--framework F] [--type T]")
	}
	id := fs.Arg(0)
	query := support.BuildQuery(map[string]string{
		"language":  *language,
		"framework": *framework,
		"type":      *snippetType,
	})
	raw, err := core.Get("/apis/"+id+"/snippets", query)
	if err != nil {
		return err
	}
	var env support.SnippetsEnvelope
	if err := support.Decode(raw, &env); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Snippets for %s: %d", id, env.Count)},
		ResultsHeading: "Snippets",
		Results:        snippetRows(env.Snippets),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runAddSnippet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apis add-snippet")
	bodyFile := fs.String("body-file", "", "Path to JSON file describing the new snippet")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: apis add-snippet <api-id> --body-file PATH")
	}
	id := fs.Arg(0)
	body, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	raw, err := core.Request("POST", "/apis/"+id+"/snippets", nil, body)
	if err != nil {
		return err
	}
	var snippet support.Snippet
	if err := support.Decode(raw, &snippet); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Created snippet %s on API %s", snippet.ID, id)},
		Changes:     []string{fmt.Sprintf("POST /apis/%s/snippets", id)},
		NextCommand: []string{fmt.Sprintf("%s snippets get %s", support.CLIName, snippet.ID)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runRecipes(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apis recipes")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: apis recipes <api-id>")
	}
	id := fs.Arg(0)
	raw, err := core.Get("/apis/"+id+"/recipes", nil)
	if err != nil {
		return err
	}
	var env support.RecipesEnvelope
	if err := support.Decode(raw, &env); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Recipes for %s: %d", id, env.Count)},
		ResultsHeading: "Recipes",
		Results:        recipeRows(env.Recipes),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runAddRecipe(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apis add-recipe")
	bodyFile := fs.String("body-file", "", "Path to JSON file describing the new recipe")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: apis add-recipe <api-id> --body-file PATH")
	}
	id := fs.Arg(0)
	body, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	raw, err := core.Request("POST", "/apis/"+id+"/recipes", nil, body)
	if err != nil {
		return err
	}
	var recipe support.Recipe
	if err := support.Decode(raw, &recipe); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Created recipe %s on API %s", recipe.ID, id)},
		Changes:     []string{fmt.Sprintf("POST /apis/%s/recipes", id)},
		NextCommand: []string{fmt.Sprintf("%s recipes get %s", support.CLIName, recipe.ID)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUsage(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apis usage")
	bodyFile := fs.String("body-file", "", "Path to JSON file with {requests, data_mb, errors, user_id}")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: apis usage <api-id> --body-file PATH")
	}
	id := fs.Arg(0)
	body, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	if _, err := core.Request("POST", "/apis/"+id+"/usage", nil, body); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Tracked usage for %s", id)},
		Changes: []string{fmt.Sprintf("POST /apis/%s/usage", id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runAnalytics(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apis analytics")
	rangeFlag := fs.String("range", "30d", "Time range: 24h|7d|30d")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: apis analytics <api-id> [--range 24h|7d|30d]")
	}
	id := fs.Arg(0)
	query := support.BuildQuery(map[string]string{"range": *rangeFlag})
	raw, err := core.Get("/apis/"+id+"/analytics", query)
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	if err := support.Decode(raw, &resp); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Analytics for %s over %s", id, *rangeFlag)},
		ResultsHeading: "Metrics",
		Results:        support.MapRows(resp),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func apiRows(list []support.API) []string {
	if len(list) == 0 {
		return []string{"(no APIs)"}
	}
	rows := make([]string, 0, len(list))
	for _, a := range list {
		rows = append(rows, fmt.Sprintf("%s (%s) | provider=%s | category=%s | status=%s",
			a.Name, support.ShortID(a.ID), orDash(a.Provider), orDash(a.Category), orDash(a.Status)))
	}
	return rows
}

func detailRows(a support.API) []string {
	rows := []string{
		fmt.Sprintf("ID: %s", a.ID),
		fmt.Sprintf("Name: %s", a.Name),
		fmt.Sprintf("Provider: %s", orDash(a.Provider)),
		fmt.Sprintf("Category: %s", orDash(a.Category)),
		fmt.Sprintf("Status: %s", orDash(a.Status)),
		fmt.Sprintf("Auth: %s", orDash(a.AuthType)),
	}
	if a.Description != "" {
		rows = append(rows, fmt.Sprintf("Description: %s", a.Description))
	}
	if a.BaseURL != "" {
		rows = append(rows, fmt.Sprintf("Base URL: %s", a.BaseURL))
	}
	if a.DocumentationURL != "" {
		rows = append(rows, fmt.Sprintf("Docs: %s", a.DocumentationURL))
	}
	if a.PricingURL != "" {
		rows = append(rows, fmt.Sprintf("Pricing: %s", a.PricingURL))
	}
	if len(a.Tags) > 0 {
		rows = append(rows, fmt.Sprintf("Tags: %s", strings.Join(a.Tags, ", ")))
	}
	if len(a.Capabilities) > 0 {
		rows = append(rows, fmt.Sprintf("Capabilities: %s", strings.Join(a.Capabilities, ", ")))
	}
	return rows
}

func snippetRows(list []support.Snippet) []string {
	if len(list) == 0 {
		return []string{"(no snippets)"}
	}
	rows := make([]string, 0, len(list))
	for _, s := range list {
		rows = append(rows, fmt.Sprintf("%s (%s) | language=%s | framework=%s | helpful=%d | uses=%d",
			s.Title, support.ShortID(s.ID), orDash(s.Language), orDash(s.Framework), s.HelpfulCount, s.UsageCount))
	}
	return rows
}

func recipeRows(list []support.Recipe) []string {
	if len(list) == 0 {
		return []string{"(no recipes)"}
	}
	rows := make([]string, 0, len(list))
	for _, r := range list {
		rows = append(rows, fmt.Sprintf("%s (%s) | difficulty=%s | rating=%.2f | used=%d",
			r.Name, support.ShortID(r.ID), orDash(r.DifficultyLevel), r.Rating, r.TimesUsed))
	}
	return rows
}

func splitCSV(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func orDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}
