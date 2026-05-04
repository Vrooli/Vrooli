package docs

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"knowledge-observatory/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type FileSearchRequest struct {
	Pattern        string `json:"pattern"`
	Scope          string `json:"scope,omitempty"`
	Scenario       string `json:"scenario,omitempty"`
	BasePath       string `json:"base_path,omitempty"`
	Limit          int    `json:"limit,omitempty"`
	IncludeContent bool   `json:"include_content,omitempty"`
}

type TextSearchRequest struct {
	Query         string   `json:"query"`
	Scope         string   `json:"scope,omitempty"`
	Scenario      string   `json:"scenario,omitempty"`
	BasePath      string   `json:"base_path,omitempty"`
	FileTypes     []string `json:"file_types,omitempty"`
	CaseSensitive bool     `json:"case_sensitive,omitempty"`
	Limit         int      `json:"limit,omitempty"`
	ContextLines  int      `json:"context_lines,omitempty"`
}

type DeepSearchRequest struct {
	Query          string `json:"query"`
	Scope          string `json:"scope,omitempty"`
	Scenario       string `json:"scenario,omitempty"`
	BasePath       string `json:"base_path,omitempty"`
	MaxResults     int    `json:"max_results,omitempty"`
	FollowRefs     *bool  `json:"follow_refs,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

type DeepSearchJob struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type AddEntryRequest struct {
	Title  string `json:"title"`
	Body   string `json:"body,omitempty"`
	Author string `json:"author,omitempty"`
	Status string `json:"status,omitempty"`
}

type ResetRequest struct {
	Path           string `json:"path"`
	MaxAgeDays     int    `json:"max_age_days,omitempty"`
	KeepMinEntries int    `json:"keep_min_entries,omitempty"`
	PreviewOnly    bool   `json:"preview_only,omitempty"`
}

type HealRequest struct {
	ScenarioName string   `json:"scenario_name"`
	Issues       []string `json:"issues,omitempty"`
	AutoApprove  bool     `json:"auto_approve,omitempty"`
	DryRun       bool     `json:"dry_run,omitempty"`
}

type HealJob struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type AutoFixRequest struct {
	DryRun bool `json:"dry_run,omitempty"`
}

type AutoFixResponse struct {
	ScenarioName string           `json:"scenario_name"`
	Moved        []AutoFixMoved   `json:"moved"`
	Skipped      []AutoFixSkipped `json:"skipped"`
	HealthBefore float64          `json:"health_before"`
	HealthAfter  float64          `json:"health_after"`
	DryRun       bool             `json:"dry_run"`
}

type AutoFixMoved struct {
	FromPath string `json:"from_path"`
	ToPath   string `json:"to_path"`
	DocType  string `json:"doc_type"`
}

type AutoFixSkipped struct {
	FromPath string `json:"from_path"`
	ToPath   string `json:"to_path"`
	DocType  string `json:"doc_type"`
	Reason   string `json:"reason"`
}

type TemplateListItem struct {
	DocType      string `json:"doc_type"`
	ExpectedPath string `json:"expected_path"`
	Purpose      string `json:"purpose"`
}

type TemplateDetailResponse struct {
	DocType      string `json:"doc_type"`
	ExpectedPath string `json:"expected_path"`
	Purpose      string `json:"purpose"`
	Content      string `json:"content"`
}

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Documentation",
		Commands: []cliapp.Command{
			{Name: "docs", NeedsAPI: true, Description: "Documentation explorer commands (read, add, view, search-files, search-text, search-deep, scenarios, tree, health, audit, heal, heal-status, autofix, reset, stats, templates, template)", Run: func(args []string) error {
				return run(deps, args)
			}},
		},
	}
}

func Usage() string {
	return `Usage: docs <subcommand> [options]

Subcommands:
  read          Read a scenario doc by type
  add           Add a structured entry (problems/progress)
  view          View a doc by file path
  search-files  Find files by glob pattern
  search-text   Full-text search across docs
  search-deep   Semantic deep search
  scenarios     List all scenarios with doc stats
  tree          Show doc tree for a scenario
  health        Check documentation health
  audit         Run comprehensive documentation audit
  heal          Auto-fix documentation health issues (agent)
  heal-status   Check heal job status
  autofix       Quick-fix misplaced docs (deterministic, no agent)
  reset         Clean up stale entries
  stats         Show read/write/reset stats
  templates     List available document templates
  template      Get a document template by type

Run 'docs <subcommand> --help' for subcommand-specific flags.`
}

func run(deps support.Dependencies, args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stdout, Usage())
		return nil
	}
	subcommand := strings.TrimSpace(args[0])
	switch subcommand {
	case "help", "--help", "-help", "-h":
		fmt.Fprintln(os.Stdout, Usage())
		return nil
	case "search-files":
		return searchFiles(deps, args[1:])
	case "search-text":
		return searchText(deps, args[1:])
	case "search-deep":
		return searchDeep(deps, args[1:])
	case "scenarios":
		return scenarios(deps, args[1:])
	case "tree":
		return tree(deps, args[1:])
	case "health":
		return health(deps, args[1:])
	case "view":
		return view(deps, args[1:])
	case "reset":
		return reset(deps, args[1:])
	case "heal":
		return heal(deps, args[1:])
	case "heal-status":
		return healStatus(deps, args[1:])
	case "autofix":
		return autoFix(deps, args[1:])
	case "read":
		return read(deps, args[1:])
	case "add":
		return add(deps, args[1:])
	case "stats":
		return stats(deps, args[1:])
	case "templates":
		return templates(deps, args[1:])
	case "template":
		return template(deps, args[1:])
	case "audit":
		return audit(deps, args[1:])
	default:
		return fmt.Errorf("unknown docs subcommand: %s\n\n%s", subcommand, Usage())
	}
}

func searchFiles(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("docs search-files", flag.ContinueOnError)
	pattern := fs.String("pattern", "", "Glob pattern (e.g. **/README.md)")
	scope := fs.String("scope", "", "Scope: global, scenario, or path")
	scenario := fs.String("scenario", "", "Scenario name (required for scope=scenario)")
	basePath := fs.String("base-path", "", "Base path (required for scope=path)")
	limit := fs.Int("limit", 0, "Maximum number of results")
	includeContent := fs.Bool("include-content", false, "Include content preview")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	patternValue := strings.TrimSpace(*pattern)
	if patternValue == "" {
		patternValue = strings.TrimSpace(strings.Join(fs.Args(), " "))
	}
	if patternValue == "" {
		return fmt.Errorf("usage: docs search-files <pattern> [--scope=global|scenario|path] [--scenario=name] [--base-path=path] [--limit=N] [--include-content]")
	}

	req := FileSearchRequest{
		Pattern:        patternValue,
		Scope:          strings.TrimSpace(*scope),
		Scenario:       strings.TrimSpace(*scenario),
		BasePath:       strings.TrimSpace(*basePath),
		IncludeContent: *includeContent,
	}
	if *limit > 0 {
		req.Limit = *limit
	}
	return printJSON(deps.ScenarioApp().Request("POST", "/docs/search/files", nil, req))
}

func searchText(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("docs search-text", flag.ContinueOnError)
	query := fs.String("query", "", "Text query (regex supported)")
	scope := fs.String("scope", "", "Scope: global, scenario, or path")
	scenario := fs.String("scenario", "", "Scenario name (required for scope=scenario)")
	basePath := fs.String("base-path", "", "Base path (required for scope=path)")
	fileTypes := fs.String("file-types", "", "Comma-separated file extensions")
	caseSensitive := fs.Bool("case-sensitive", false, "Case-sensitive search")
	limit := fs.Int("limit", 0, "Maximum number of results")
	contextLines := fs.Int("context-lines", 0, "Lines of context before/after matches")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	queryValue := strings.TrimSpace(*query)
	if queryValue == "" {
		queryValue = strings.TrimSpace(strings.Join(fs.Args(), " "))
	}
	if queryValue == "" {
		return fmt.Errorf("usage: docs search-text <query> [--scope=global|scenario|path] [--scenario=name] [--base-path=path] [--file-types=md,txt] [--case-sensitive] [--limit=N] [--context-lines=N]")
	}

	req := TextSearchRequest{
		Query:         queryValue,
		Scope:         strings.TrimSpace(*scope),
		Scenario:      strings.TrimSpace(*scenario),
		BasePath:      strings.TrimSpace(*basePath),
		FileTypes:     support.SplitCSV(*fileTypes),
		CaseSensitive: *caseSensitive,
		ContextLines:  *contextLines,
	}
	if *limit > 0 {
		req.Limit = *limit
	}
	return printJSON(deps.ScenarioApp().Request("POST", "/docs/search/text", nil, req))
}

func searchDeep(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("docs search-deep", flag.ContinueOnError)
	query := fs.String("query", "", "Deep search query")
	scope := fs.String("scope", "", "Scope: global, scenario, or path")
	scenario := fs.String("scenario", "", "Scenario name (required for scope=scenario)")
	basePath := fs.String("base-path", "", "Base path (required for scope=path)")
	maxResults := fs.Int("max-results", 0, "Maximum number of results (default 10)")
	followRefs := fs.Bool("follow-refs", true, "Follow documentation references")
	timeoutSeconds := fs.Int("timeout-seconds", 0, "Agent timeout in seconds")
	wait := fs.Bool("wait", true, "Wait for results to complete")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	queryValue := strings.TrimSpace(*query)
	if queryValue == "" {
		queryValue = strings.TrimSpace(strings.Join(fs.Args(), " "))
	}
	if queryValue == "" {
		return fmt.Errorf("usage: docs search-deep <query> [--scope=global|scenario|path] [--scenario=name] [--base-path=path] [--max-results=N] [--follow-refs] [--timeout-seconds=N] [--wait]")
	}

	req := DeepSearchRequest{
		Query:          queryValue,
		Scope:          strings.TrimSpace(*scope),
		Scenario:       strings.TrimSpace(*scenario),
		BasePath:       strings.TrimSpace(*basePath),
		FollowRefs:     followRefs,
		TimeoutSeconds: *timeoutSeconds,
	}
	if *maxResults > 0 {
		req.MaxResults = *maxResults
	}

	body, err := deps.ScenarioApp().Request("POST", "/docs/search/deep", nil, req)
	if err != nil {
		return err
	}
	if !*wait {
		cliutil.PrintJSON(body)
		return nil
	}

	var job DeepSearchJob
	if err := json.Unmarshal(body, &job); err != nil {
		return fmt.Errorf("failed to decode job: %w", err)
	}
	if job.JobID == "" {
		return fmt.Errorf("missing job id in response")
	}
	if job.Status == "completed" || job.Status == "failed" {
		cliutil.PrintJSON(body)
		return nil
	}

	timeout := time.Duration(*timeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	deadline := time.Now().Add(timeout + 5*time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		statusBody, err := deps.ScenarioApp().Request("GET", fmt.Sprintf("/docs/search/deep/%s", job.JobID), nil, nil)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(statusBody, &job); err != nil {
			return fmt.Errorf("failed to decode job status: %w", err)
		}
		if job.Status == "completed" || job.Status == "failed" {
			cliutil.PrintJSON(statusBody)
			return nil
		}
	}
	return fmt.Errorf("deep search timed out before completion")
}

func scenarios(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("docs scenarios", flag.ContinueOnError)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	return printJSON(deps.ScenarioApp().Request("GET", "/scenarios", nil, nil))
}

func tree(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("docs tree", flag.ContinueOnError)
	scenario := fs.String("scenario", "", "Scenario name")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	scenarioValue := strings.TrimSpace(*scenario)
	if scenarioValue == "" {
		scenarioValue = strings.TrimSpace(strings.Join(fs.Args(), " "))
	}
	if scenarioValue == "" {
		return fmt.Errorf("usage: docs tree <scenario> [--scenario=name]")
	}
	return printJSON(deps.ScenarioApp().Request("GET", fmt.Sprintf("/scenarios/%s/docs", scenarioValue), nil, nil))
}

func health(deps support.Dependencies, args []string) error {
	args, jsonFromPositional := support.StripBoolFlag(args, "--json")
	fs := flag.NewFlagSet("docs health", flag.ContinueOnError)
	scenario := fs.String("scenario", "", "Scenario name")
	jsonOut := fs.Bool("json", false, "Output raw JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	scenarioValue := strings.TrimSpace(*scenario)
	if scenarioValue == "" {
		scenarioValue = strings.TrimSpace(strings.Join(fs.Args(), " "))
	}
	if scenarioValue == "" {
		return fmt.Errorf("usage: docs health <scenario> [--scenario=name] [--json]")
	}

	body, err := deps.ScenarioApp().Request("GET", fmt.Sprintf("/scenarios/%s/docs/health", scenarioValue), nil, nil)
	if err != nil {
		return err
	}
	if *jsonOut || jsonFromPositional {
		cliutil.PrintJSON(body)
		return nil
	}
	var result HealthResponse
	if err := json.Unmarshal(body, &result); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Print(RenderHealthReport(result, scenarioValue))
	return nil
}

func view(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("docs view", flag.ContinueOnError)
	path := fs.String("path", "", "Document path")
	format := fs.String("format", "raw", "Output format: raw (default, content only) or json (full response)")
	jsonOut := fs.Bool("json", false, "Output full JSON response (shorthand for --format json)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	pathValue := strings.TrimSpace(*path)
	if pathValue == "" {
		pathValue = strings.TrimSpace(strings.Join(fs.Args(), " "))
	}
	if pathValue == "" {
		return fmt.Errorf("usage: docs view <path> [--format=raw|json] [--json]")
	}
	wantJSON := *jsonOut || strings.TrimSpace(*format) == "json"
	query := url.Values{}
	query.Set("path", pathValue)
	if !wantJSON {
		if formatValue := strings.TrimSpace(*format); formatValue != "" {
			query.Set("format", formatValue)
		}
	}
	body, err := deps.ScenarioApp().Request("GET", "/docs/content", query, nil)
	if err != nil {
		return err
	}
	if wantJSON {
		cliutil.PrintJSON(body)
		return nil
	}
	return printDocContent(body)
}

func reset(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("docs reset", flag.ContinueOnError)
	path := fs.String("path", "", "Document path")
	maxAgeDays := fs.Int("max-age-days", 0, "Remove entries older than N days")
	keepMin := fs.Int("keep-min-entries", 0, "Always keep at least N entries")
	preview := fs.Bool("preview", false, "Preview changes without writing")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	pathValue := strings.TrimSpace(*path)
	if pathValue == "" {
		pathValue = strings.TrimSpace(strings.Join(fs.Args(), " "))
	}
	if pathValue == "" {
		return fmt.Errorf("usage: docs reset <path> [--max-age-days=N] [--keep-min-entries=N] [--preview]")
	}
	req := ResetRequest{Path: pathValue, MaxAgeDays: *maxAgeDays, KeepMinEntries: *keepMin, PreviewOnly: *preview}
	return printJSON(deps.ScenarioApp().Request("POST", "/docs/reset", nil, req))
}

func heal(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("docs heal", flag.ContinueOnError)
	scenario := fs.String("scenario", "", "Scenario name")
	issues := fs.String("issues", "", "Comma-separated issue labels")
	autoApprove := fs.Bool("auto-approve", false, "Auto-approve if health improves")
	dryRun := fs.Bool("dry-run", false, "Preview only (no apply)")
	wait := fs.Bool("wait", false, "Wait for job completion")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	scenarioValue := strings.TrimSpace(*scenario)
	if scenarioValue == "" {
		scenarioValue = strings.TrimSpace(strings.Join(fs.Args(), " "))
	}
	if scenarioValue == "" {
		return fmt.Errorf("usage: docs heal <scenario> [--issues=label1,label2] [--auto-approve] [--dry-run] [--wait]")
	}

	req := HealRequest{ScenarioName: scenarioValue, Issues: support.SplitCSV(*issues), AutoApprove: *autoApprove, DryRun: *dryRun}
	body, err := deps.ScenarioApp().Request("POST", fmt.Sprintf("/scenarios/%s/docs/heal", scenarioValue), nil, req)
	if err != nil {
		return err
	}
	if !*wait {
		cliutil.PrintJSON(body)
		return nil
	}

	var job HealJob
	if err := json.Unmarshal(body, &job); err != nil {
		return err
	}
	if job.JobID == "" {
		return fmt.Errorf("healing job id missing")
	}
	for {
		statusBody, err := deps.ScenarioApp().Request("GET", fmt.Sprintf("/docs/heal/%s", job.JobID), nil, nil)
		if err != nil {
			return err
		}
		var status HealJob
		if err := json.Unmarshal(statusBody, &status); err != nil {
			return err
		}
		if status.Status != "pending" && status.Status != "running" {
			cliutil.PrintJSON(statusBody)
			return nil
		}
		time.Sleep(2 * time.Second)
	}
}

func healStatus(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("docs heal-status", flag.ContinueOnError)
	jobID := fs.String("job-id", "", "Healing job ID")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	jobValue := strings.TrimSpace(*jobID)
	if jobValue == "" {
		jobValue = strings.TrimSpace(strings.Join(fs.Args(), " "))
	}
	if jobValue == "" {
		return fmt.Errorf("usage: docs heal-status <job_id>")
	}
	return printJSON(deps.ScenarioApp().Request("GET", fmt.Sprintf("/docs/heal/%s", jobValue), nil, nil))
}

func autoFix(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("docs autofix", flag.ContinueOnError)
	scenario := fs.String("scenario", "", "Scenario name")
	dryRun := fs.Bool("dry-run", false, "Preview only (no moves)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	scenarioValue := strings.TrimSpace(*scenario)
	if scenarioValue == "" {
		scenarioValue = strings.TrimSpace(strings.Join(fs.Args(), " "))
	}
	if scenarioValue == "" {
		return fmt.Errorf("usage: docs autofix <scenario> [--dry-run]")
	}
	body, err := deps.ScenarioApp().Request("POST", fmt.Sprintf("/scenarios/%s/docs/autofix", scenarioValue), nil, AutoFixRequest{DryRun: *dryRun})
	if err != nil {
		return err
	}

	var result AutoFixResponse
	if err := json.Unmarshal(body, &result); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if *dryRun {
		fmt.Println("Dry run — no files were moved.")
	}
	if len(result.Moved) > 0 {
		fmt.Printf("Moved %d file(s):\n", len(result.Moved))
		for _, move := range result.Moved {
			fmt.Printf("  %s → %s\n", move.FromPath, move.ToPath)
		}
	}
	if len(result.Skipped) > 0 {
		fmt.Printf("Skipped %d file(s):\n", len(result.Skipped))
		for _, skipped := range result.Skipped {
			fmt.Printf("  %s → %s (%s)\n", skipped.FromPath, skipped.ToPath, skipped.Reason)
		}
	}
	if len(result.Moved) == 0 && len(result.Skipped) == 0 {
		fmt.Println("No misplaced docs to fix.")
	}
	fmt.Printf("Health: %.0f%% → %.0f%%\n", result.HealthBefore*100, result.HealthAfter*100)
	return nil
}

func read(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("docs read", flag.ContinueOnError)
	scenario := fs.String("scenario", "", "Scenario name")
	doc := fs.String("doc", "", "Document type")
	format := fs.String("format", "raw", "Output format: raw (default, content only) or json (full response)")
	jsonOut := fs.Bool("json", false, "Output full JSON response (shorthand for --format json)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	scenarioValue := strings.TrimSpace(*scenario)
	docValue := strings.TrimSpace(*doc)
	positional := fs.Args()
	if scenarioValue == "" && len(positional) > 0 {
		scenarioValue = strings.TrimSpace(positional[0])
		positional = positional[1:]
	}
	if docValue == "" && len(positional) > 0 {
		docValue = strings.TrimSpace(positional[0])
	}
	if scenarioValue == "" || docValue == "" {
		return fmt.Errorf("usage: docs read <scenario> <type> [--format=raw|json] [--json]")
	}
	wantJSON := *jsonOut || strings.TrimSpace(*format) == "json"
	query := url.Values{}
	if !wantJSON {
		if formatValue := strings.TrimSpace(*format); formatValue != "" {
			query.Set("format", formatValue)
		}
	}
	body, err := deps.ScenarioApp().Request("GET", fmt.Sprintf("/scenarios/%s/docs/%s/content", scenarioValue, docValue), query, nil)
	if err != nil {
		return err
	}
	if wantJSON {
		cliutil.PrintJSON(body)
		return nil
	}
	return printDocContent(body)
}

func add(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("docs add", flag.ContinueOnError)
	scenario := fs.String("scenario", "", "Scenario name")
	doc := fs.String("doc", "", "Document type (problems, progress)")
	title := fs.String("title", "", "Entry title (required)")
	body := fs.String("body", "", "Entry body/notes")
	author := fs.String("author", "", "Author (for progress entries)")
	status := fs.String("status", "", "Status (for progress entries)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	scenarioValue := strings.TrimSpace(*scenario)
	docValue := strings.TrimSpace(*doc)
	titleValue := strings.TrimSpace(*title)
	if scenarioValue == "" || docValue == "" || titleValue == "" {
		return fmt.Errorf("usage: docs add --scenario <name> --doc <type> --title \"...\" [--body \"...\"] [--author \"...\"] [--status \"...\"]")
	}
	req := AddEntryRequest{Title: titleValue, Body: strings.TrimSpace(*body), Author: strings.TrimSpace(*author), Status: strings.TrimSpace(*status)}
	return printJSON(deps.ScenarioApp().Request("POST", fmt.Sprintf("/scenarios/%s/docs/%s/entries", scenarioValue, docValue), nil, req))
}

func stats(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("docs stats", flag.ContinueOnError)
	scenario := fs.String("scenario", "", "Filter by scenario name")
	doc := fs.String("doc", "", "Filter by document type")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	query := url.Values{}
	if s := strings.TrimSpace(*scenario); s != "" {
		query.Set("scenario", s)
	}
	if d := strings.TrimSpace(*doc); d != "" {
		query.Set("doc_type", d)
	}
	return printJSON(deps.ScenarioApp().Request("GET", "/docs/stats", query, nil))
}

func templates(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("docs templates", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output raw JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := deps.ScenarioApp().Request("GET", "/docs/templates", nil, nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	var items []TemplateListItem
	if err := json.Unmarshal(body, &items); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}
	for _, item := range items {
		fmt.Printf("%-20s %s\n", item.DocType, item.Purpose)
	}
	return nil
}

func template(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("docs template", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output full JSON response")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	docType := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if docType == "" {
		return fmt.Errorf("usage: docs template <type> [--json]")
	}
	body, err := deps.ScenarioApp().Request("GET", fmt.Sprintf("/docs/templates/%s", docType), nil, nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	var detail TemplateDetailResponse
	if err := json.Unmarshal(body, &detail); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Print(detail.Content)
	return nil
}

func audit(deps support.Dependencies, args []string) error {
	args, jsonFromPositional := support.StripBoolFlag(args, "--json")
	fs := flag.NewFlagSet("docs audit", flag.ContinueOnError)
	scenario := fs.String("scenario", "", "Scenario name")
	jsonOut := fs.Bool("json", false, "Output raw JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	scenarioValue := strings.TrimSpace(*scenario)
	if scenarioValue == "" {
		scenarioValue = strings.TrimSpace(strings.Join(fs.Args(), " "))
	}
	if scenarioValue == "" {
		return fmt.Errorf("usage: docs audit <scenario> [--json]")
	}
	body, err := deps.ScenarioApp().Request("GET", fmt.Sprintf("/scenarios/%s/docs/audit", scenarioValue), nil, nil)
	if err != nil {
		return err
	}
	if *jsonOut || jsonFromPositional {
		cliutil.PrintJSON(body)
		return nil
	}
	var result AuditResponse
	if err := json.Unmarshal(body, &result); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Print(RenderAuditReport(result, scenarioValue))
	return nil
}

func printJSON(body []byte, err error) error {
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func printDocContent(body []byte) error {
	var doc struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Print(doc.Content)
	return nil
}
