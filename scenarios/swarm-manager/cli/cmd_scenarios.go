package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdScenariosList(args []string) error {
	fs := flag.NewFlagSet("scenarios list", flag.ContinueOnError)
	search := fs.String("search", "", "Filter by name or description")
	status := fs.String("status", "", "Filter by status (running|stopped|error|unknown)")
	tags := fs.String("tags", "", "Filter by tags (comma-separated)")
	sortField := fs.String("sort", "", "Sort by field (priority|name|displayName)")
	order := fs.String("order", "", "Sort order (asc|desc)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if strings.TrimSpace(*search) != "" {
		query.Set("search", strings.TrimSpace(*search))
	}
	if strings.TrimSpace(*status) != "" {
		query.Set("status", strings.TrimSpace(*status))
	}
	if strings.TrimSpace(*tags) != "" {
		query.Set("tags", strings.TrimSpace(*tags))
	}
	if strings.TrimSpace(*sortField) != "" {
		query.Set("sort", strings.TrimSpace(*sortField))
	}
	if strings.TrimSpace(*order) != "" {
		query.Set("order", strings.TrimSpace(*order))
	}

	body, err := a.core.Get("/scenarios", query)
	if err != nil {
		return err
	}

	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	var response ListScenariosResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Scenarios) == 0 {
		printSection("Summary")
		fmt.Println("  No scenarios found.")
		printCommandListSection("Next Steps", []string{
			cliCommand("status"),
			cliCommand("scenarios", "list", "--status", "running"),
		})
		return nil
	}

	printSection("Summary")
	fmt.Printf("  Found %d scenario(s)\n", len(response.Scenarios))
	printSection("Results")
	for _, scenario := range response.Scenarios {
		display := scenario.DisplayName
		if display == "" {
			display = scenario.Name
		}
		fmt.Printf("  %s (status: %s, priority: %d)\n", scenario.Name, scenario.Status, scenario.Priority)
		fmt.Printf("    Display: %s\n", display)
		if scenario.Description != "" {
			fmt.Printf("    Description: %s\n", scenario.Description)
		}
		if len(scenario.Tags) > 0 {
			fmt.Printf("    Tags: %s\n", strings.Join(scenario.Tags, ", "))
		}
		fmt.Println()
	}
	first := response.Scenarios[0]
	printCommandListSection("Retrieval Hints", []string{
		cliCommand("scenarios", "get", "--name", "<name>"),
		cliCommand("scenarios", "get", "--name", first.Name),
		cliCommand("scenarios", "files", "--name", first.Name),
	})
	return nil
}

func (a *App) cmdScenariosGet(args []string) error {
	fs := flag.NewFlagSet("scenarios get", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Scenario name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: scenarios get --name NAME [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)

	body, err := a.core.Get("/scenarios/"+name, nil)
	if err != nil {
		return err
	}

	// Fetch context for fix-history summary. Failure here must not break the
	// primary `get` — fall back to no fix data.
	var ctxResp ScenarioContextResponse
	if ctxBody, ctxErr := a.core.Get("/scenarios/"+name+"/context", nil); ctxErr == nil {
		_ = json.Unmarshal(ctxBody, &ctxResp)
	}

	if *jsonOut {
		// Greenfield: merge context.fixes into the JSON output so consumers see
		// everything in one shot. Stable key shape: { "scenario": ..., "fixes": ... }.
		merged := map[string]any{}
		var raw map[string]any
		if err := json.Unmarshal(body, &raw); err == nil {
			for k, v := range raw {
				merged[k] = v
			}
		}
		merged["fixes"] = ctxResp.Fixes
		out, _ := json.MarshalIndent(merged, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	response, err := decodeResponse[ScenarioResponse](body)
	if err != nil {
		return err
	}
	scenario := response.Scenario

	printSection("Summary")
	fmt.Printf("  %s (%s)\n", scenario.Name, scenario.Status)

	printSection("Details")
	fmt.Printf("  Name: %s\n", scenario.Name)
	fmt.Printf("  Display Name: %s\n", scenario.DisplayName)
	fmt.Printf("  Description: %s\n", scenario.Description)
	fmt.Printf("  Status: %s\n", scenario.Status)
	fmt.Printf("  Priority: %d\n", scenario.Priority)
	if scenario.CompletenessScore != nil {
		fmt.Printf("  Completeness: %d\n", *scenario.CompletenessScore)
	}
	fmt.Printf("  Greenfield: %v\n", scenario.IsGreenfield)
	if len(scenario.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(scenario.Tags, ", "))
	}

	printFixHistorySummary(ctxResp.Fixes)

	printCommandListSection("Next Steps", []string{
		cliCommand("scenarios", "files", "--name", scenario.Name),
		cliCommand("scenarios", "fixes", "--name", scenario.Name),
		cliCommand("scenarios", "update", "--name", scenario.Name, "--data", "'{\"is_greenfield\":true}'"),
		cliCommand("scenarios", "start", "--name", scenario.Name),
	})
	return nil
}

// cmdScenariosFixes lists fix backlog items targeting a scenario, partitioned
// by active/archived. Default scope is --all. Search is a substring match
// over title and name (case-insensitive).
func (a *App) cmdScenariosFixes(args []string) error {
	fs := flag.NewFlagSet("scenarios fixes", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Scenario name")
	activeOnly := fs.Bool("active", false, "Show only active fixes")
	archivedOnly := fs.Bool("archived", false, "Show only archived fixes")
	allFlag := fs.Bool("all", false, "Show both active and archived (default)")
	search := fs.String("search", "", "Substring filter on title or name (case-insensitive)")
	limit := fs.Int("limit", 50, "Maximum fixes to print per partition")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: scenarios fixes --name NAME [--active|--archived|--all] [--search Q] [--limit N] [--json]\n\n%s", err)
	}
	if boolCount(*activeOnly, *archivedOnly, *allFlag) > 1 {
		return fmt.Errorf("--active, --archived, and --all are mutually exclusive")
	}
	scope := "all"
	switch {
	case *activeOnly:
		scope = "active"
	case *archivedOnly:
		scope = "archived"
	}

	name := strings.TrimSpace(*nameFlag)
	body, err := a.core.Get("/scenarios/"+name+"/context", nil)
	if err != nil {
		return err
	}

	var ctxResp ScenarioContextResponse
	if err := json.Unmarshal(body, &ctxResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	active := filterFixes(ctxResp.Fixes.Active, *search)
	archived := filterFixes(ctxResp.Fixes.Archived, *search)

	if *limit > 0 {
		if len(active) > *limit {
			active = active[:*limit]
		}
		if len(archived) > *limit {
			archived = archived[:*limit]
		}
	}

	if *jsonOut {
		out := map[string]any{"scenario_name": name, "scope": scope, "search": *search}
		switch scope {
		case "active":
			out["fixes"] = ScenarioFixHistory{Active: active, Archived: []ScenarioFix{}}
		case "archived":
			out["fixes"] = ScenarioFixHistory{Active: []ScenarioFix{}, Archived: archived}
		default:
			out["fixes"] = ScenarioFixHistory{Active: active, Archived: archived}
		}
		buf, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(buf))
		return nil
	}

	printSection("Summary")
	fmt.Printf("  Scenario: %s\n", name)
	fmt.Printf("  Scope: %s", scope)
	if strings.TrimSpace(*search) != "" {
		fmt.Printf(" · search=%q", *search)
	}
	fmt.Println()

	if scope == "active" || scope == "all" {
		printSection("Active Fixes")
		if len(active) == 0 {
			fmt.Println("  (none)")
		} else {
			printFixTable(active)
		}
	}
	if scope == "archived" || scope == "all" {
		printSection("Archived Fixes")
		if len(archived) == 0 {
			fmt.Println("  (none)")
		} else {
			printFixTable(archived)
		}
	}

	printCommandListSection("Next Steps", []string{
		cliCommand("scenarios", "fixes", "--name", name, "--archived"),
		cliCommand("aisearch", "search", "--query", "<symptom>", "--kind", "fix", "--include-archived"),
	})
	return nil
}

func boolCount(bs ...bool) int {
	n := 0
	for _, b := range bs {
		if b {
			n++
		}
	}
	return n
}

func filterFixes(in []ScenarioFix, search string) []ScenarioFix {
	q := strings.ToLower(strings.TrimSpace(search))
	if q == "" {
		out := make([]ScenarioFix, len(in))
		copy(out, in)
		return out
	}
	out := make([]ScenarioFix, 0, len(in))
	for _, f := range in {
		if strings.Contains(strings.ToLower(f.Title), q) || strings.Contains(strings.ToLower(f.Name), q) {
			out = append(out, f)
		}
	}
	return out
}

func printFixTable(fixes []ScenarioFix) {
	for _, f := range fixes {
		title := f.Title
		if title == "" {
			title = f.Name
		}
		archived := ""
		if f.ArchivedAt != nil {
			archived = " · archived " + *f.ArchivedAt
		}
		init := ""
		if f.Initiative != "" {
			init = " · initiative=" + f.Initiative
		}
		fmt.Printf("  [P%d %s] %s\n", f.Priority, f.Status, title)
		fmt.Printf("    %s%s%s\n", f.Path, init, archived)
	}
}

// printFixHistorySummary renders a compact summary block under `scenarios get`.
// Shows totals and the top 5 most-recent archived fixes for fast triage.
func printFixHistorySummary(h ScenarioFixHistory) {
	printSection("Fix History")
	fmt.Printf("  Active: %d · Archived: %d\n", len(h.Active), len(h.Archived))
	if len(h.Archived) == 0 {
		return
	}
	fmt.Println("  Recent archived:")
	max := 5
	if len(h.Archived) < max {
		max = len(h.Archived)
	}
	for _, f := range h.Archived[:max] {
		title := f.Title
		if title == "" {
			title = f.Name
		}
		when := ""
		if f.ArchivedAt != nil {
			when = " (archived " + *f.ArchivedAt + ")"
		}
		fmt.Printf("    - %s%s\n", title, when)
	}
}

func (a *App) cmdScenariosUpdate(args []string) error {
	fs := flag.NewFlagSet("scenarios update", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Scenario name")
	data := fs.String("data", "", "JSON payload (inline or @file)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("name", *nameFlag, "data", *data); err != nil {
		return fmt.Errorf("usage: scenarios update --name NAME --data JSON [--json]\n\nExample:\n  scenarios update --name my-scenario --data '{\"is_greenfield\":true}'\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)
	payload, err := parseJSONString(*data)
	if err != nil {
		return err
	}

	var patch map[string]any
	if err := json.Unmarshal(payload, &patch); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	body, err := a.core.Request("PATCH", "/scenarios/"+name, nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[ScenarioResponse](body)
	if err != nil {
		return err
	}
	scenario := response.Scenario

	printSection("Result")
	fmt.Printf("  Updated scenario: %s\n", scenario.Name)
	printSection("What Changed")
	fmt.Printf("  Greenfield: %v\n", scenario.IsGreenfield)
	printCommandListSection("Next Steps", []string{
		cliCommand("scenarios", "get", "--name", scenario.Name),
		cliCommand("scenarios", "files", "--name", scenario.Name),
	})
	return nil
}

func (a *App) cmdScenariosDelete(args []string) error {
	fs := flag.NewFlagSet("scenarios delete", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Scenario name")
	archive := fs.Bool("archive", false, "Archive scenario to backlog (idea) before deletion")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: scenarios delete --name NAME [--archive] [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)

	query := url.Values{}
	if *archive {
		query.Set("archive", "true")
	}

	body, err := a.core.Request("DELETE", "/scenarios/"+name, query, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[DeleteScenarioResponse](body)
	if err != nil {
		return err
	}

	printSection("Result")
	fmt.Printf("  %s\n", response.Message)
	printCommandListSection("Next Steps", []string{
		cliCommand("scenarios", "list"),
		cliCommand("backlog", "list"),
	})
	return nil
}

func (a *App) cmdScenariosFiles(args []string) error {
	fs := flag.NewFlagSet("scenarios files", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Scenario name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: scenarios files --name NAME [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)

	body, err := a.core.Get("/scenarios/"+name+"/files", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[ScenarioFilesResponse](body)
	if err != nil {
		return err
	}
	if len(response.Files) == 0 {
		printSection("Summary")
		fmt.Printf("  No files found for scenario %s.\n", name)
		printCommandListSection("Next Steps", []string{
			cliCommand("scenarios", "get", "--name", name),
		})
		return nil
	}

	printSection("Summary")
	fmt.Printf("  Found %d file node(s) for scenario %s\n", len(response.Files), name)
	printSection("Results")
	printTree(response.Files,
		func(item ScenarioFile) []ScenarioFile { return item.Children },
		func(item ScenarioFile) string {
			if item.Type == "directory" {
				return item.Name + "/"
			}
			return fmt.Sprintf("%s (%d bytes)", item.Name, item.Size)
		},
		0,
	)
	printCommandListSection("Retrieval Hints", []string{
		cliCommand("scenarios", "get", "--name", name),
		cliCommand("scenarios", "update", "--name", name, "--data", "<json-or-@file>"),
	})
	return nil
}

func (a *App) cmdScenariosStart(args []string) error {
	return a.runScenarioLifecycle(args, "start")
}

func (a *App) cmdScenariosSpecSyncArchive(args []string) error {
	fs := flag.NewFlagSet("scenarios spec-sync-archive", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Scenario name")
	preset := fs.String("preset", "", "Preserve-files preset for archive (for example: planning, all-planning)")
	pathsCSV := fs.String("paths", "", "Comma-separated preserve path globs")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: scenarios spec-sync-archive --name NAME [--preset PRESET] [--paths path1,path2] [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)

	var payload any
	trimmedPreset := strings.TrimSpace(*preset)
	paths := make([]string, 0)
	for _, value := range cliutil.ParseCSV(*pathsCSV) {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			paths = append(paths, trimmed)
		}
	}
	if trimmedPreset != "" || len(paths) > 0 {
		preserveFiles := map[string]any{}
		if trimmedPreset != "" {
			preserveFiles["preset"] = trimmedPreset
		}
		if len(paths) > 0 {
			preserveFiles["paths"] = paths
		}
		payload = map[string]any{
			"preserve_files": preserveFiles,
		}
	}

	body, err := a.core.Request("POST", "/scenarios/"+name+"/spec-sync-archive", nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[SpecSyncArchiveResponse](body)
	if err != nil {
		return err
	}
	printSection("Result")
	fmt.Printf("  Queued spec-sync-archive for %s\n", name)
	printSection("What Changed")
	fmt.Printf("  Execution ID: %s\n", response.ExecutionID)
	fmt.Printf("  Status: %s\n", response.Status)
	if strings.TrimSpace(response.Message) != "" {
		fmt.Printf("  Message: %s\n", response.Message)
	}
	printCommandListSection("Next Steps", []string{
		cliCommand("execution", "get", "--id", response.ExecutionID),
		cliCommand("execution", "prompt-trace", "--id", response.ExecutionID),
		cliCommand("execution", "list", "--status", "queued"),
	})
	return nil
}

func (a *App) cmdScenariosStop(args []string) error {
	return a.runScenarioLifecycle(args, "stop")
}

func (a *App) cmdScenariosRestart(args []string) error {
	return a.runScenarioLifecycle(args, "restart")
}

func (a *App) cmdScenariosReviewQueue(args []string) error {
	fs := flag.NewFlagSet("scenarios review-queue", flag.ContinueOnError)
	limit := fs.Int("limit", 5, "Max scenarios to return (1-50)")
	excludeTag := fs.String("exclude-tag", "", "Tag to exclude scenarios with pending QA fixes")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if *limit != 5 {
		query.Set("limit", fmt.Sprintf("%d", *limit))
	}
	if strings.TrimSpace(*excludeTag) != "" {
		query.Set("exclude_tag", strings.TrimSpace(*excludeTag))
	}

	body, err := a.core.Get("/scenarios/review-queue", query)
	if err != nil {
		return err
	}

	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	var response ScenarioReviewQueueResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	printSection("Review Queue")
	fmt.Printf("  Total scenarios: %d | Excluded (pending fixes): %d\n\n", response.TotalScenarios, response.ExcludedCount)

	if len(response.Items) == 0 {
		fmt.Println("  No scenarios require preemptive review.")
		return nil
	}

	// Table header.
	fmt.Printf("  %-30s %7s %12s %8s  %s\n", "Scenario", "Pending", "Last Review", "Score", "Signal")
	fmt.Printf("  %-30s %7s %12s %8s  %s\n", "--------", "-------", "-----------", "-----", "------")
	for _, item := range response.Items {
		lastReview := "-"
		if item.LastReviewClassification != "" {
			lastReview = item.LastReviewClassification
		}
		cooldown := ""
		if item.CooldownUntil != "" {
			cooldown = " (cooldown)"
		}
		fmt.Printf("  %-30s %7d %12s %8.1f  %s%s\n",
			item.ScenarioName,
			item.PendingBacklogCount,
			lastReview,
			item.CompositeScore,
			item.PrimarySignal,
			cooldown,
		)
	}

	fmt.Println()
	printCommandListSection("Next Steps", []string{
		cliCommand("scenarios", "review-queue", "--limit", "10", "--json"),
	})
	return nil
}

func (a *App) runScenarioLifecycle(args []string, action string) error {
	fs := flag.NewFlagSet("scenarios "+action, flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Scenario name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: scenarios %s --name NAME [--json]\n\n%s", action, err)
	}
	name := strings.TrimSpace(*nameFlag)
	body, err := a.core.Request("POST", "/scenarios/"+name+"/"+action, nil, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[ScenarioResponse](body)
	if err != nil {
		return err
	}
	printSection("Result")
	fmt.Printf("  Scenario %s: %s\n", action, response.Scenario.Name)
	printSection("What Changed")
	fmt.Printf("  Status: %s\n", response.Scenario.Status)
	printCommandListSection("Next Steps", []string{
		cliCommand("scenarios", "get", "--name", response.Scenario.Name),
		cliCommand("scenarios", "list", "--status", response.Scenario.Status),
	})
	return nil
}
