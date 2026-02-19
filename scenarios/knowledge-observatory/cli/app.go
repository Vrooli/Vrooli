package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	appName        = "knowledge-observatory"
	appVersion     = "2.0.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

// App wraps the cli-core ScenarioApp with knowledge-observatory commands.
type App struct {
	core *cliapp.ScenarioApp
}

// NewApp constructs the CLI wiring.
func NewApp() (*App, error) {
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{
		ExtraAPIEnvVars: []string{"API_BASE_URL", "VITE_API_BASE_URL"},
	})
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:              appName,
		Version:           appVersion,
		Description:       "Knowledge Observatory CLI - search, ingest, and graph knowledge",
		DefaultAPIBase:    defaultAPIBase,
		APIEnvVars:        env.APIEnvVars,
		APIPortEnvVars:    env.APIPortEnvVars,
		APIPortDetector:   cliutil.DetectPortFromVrooli(appName, "API_PORT"),
		ConfigDirEnvVars:  env.ConfigDirEnvVars,
		SourceRootEnvVars: env.SourceRootEnvVars,
		TokenEnvVars:      env.TokenEnvVars,
		BuildFingerprint:  buildFingerprint,
		BuildTimestamp:    buildTimestamp,
		BuildSourceRoot:   buildSourceRoot,
		AllowAnonymous:    true,
	})
	if err != nil {
		return nil, err
	}
	app := &App{core: core}
	app.core.SetCommands(app.registerCommands())
	return app, nil
}

// Run executes the CLI with provided args.
func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	health := cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{Name: "status", NeedsAPI: true, Description: "Check API health", Run: a.cmdStatus},
			{Name: "health", NeedsAPI: true, Description: "Get knowledge health metrics", Run: a.cmdHealth},
			{Name: "metrics", NeedsAPI: true, Description: "Alias for health", Run: a.cmdHealth},
		},
	}

	knowledge := cliapp.CommandGroup{
		Title: "Knowledge",
		Commands: []cliapp.Command{
			{Name: "search", NeedsAPI: true, Description: "Semantic search over knowledge", Run: a.cmdSearch},
			{Name: "ingest", NeedsAPI: true, Description: "Ingest a single knowledge record", Run: a.cmdIngest},
			{Name: "ingest-job", NeedsAPI: true, Description: "Enqueue an async document ingest job", Run: a.cmdIngestJob},
			{Name: "job-status", NeedsAPI: true, Description: "Check async ingest job status", Run: a.cmdJobStatus},
			{Name: "ingest-health", NeedsAPI: true, Description: "Inspect ingest queue and runner health", Run: a.cmdIngestHealth},
			{Name: "collection-diagnostics", NeedsAPI: true, Description: "Inspect collection embedding/chunk diagnostics", Run: a.cmdCollectionDiagnostics},
			{Name: "collection-prune-stale", NeedsAPI: true, Description: "Prune stale chunk versions (dry-run by default)", Run: a.cmdCollectionPruneStale},
			{Name: "collection-dedupe", NeedsAPI: true, Description: "Delete duplicate content chunks (dry-run by default)", Run: a.cmdCollectionDedupe},
			{Name: "document-delete", NeedsAPI: true, Description: "Delete all chunks for a document (dry-run by default)", Run: a.cmdDocumentDelete},
			{Name: "graph", NeedsAPI: true, Description: "Generate a knowledge graph", Run: a.cmdGraph},
		},
	}

	docs := cliapp.CommandGroup{
		Title: "Documentation",
		Commands: []cliapp.Command{
			{Name: "docs", NeedsAPI: true, Description: "Documentation explorer commands (search-files, search-text, search-deep, scenarios, tree, health, view, reset, heal, heal-status, read, add, stats)", Run: a.cmdDocs},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, knowledge, docs, config}
}

func (a *App) apiPath(v1Path string) string {
	v1Path = strings.TrimSpace(v1Path)
	if v1Path == "" {
		return ""
	}
	if !strings.HasPrefix(v1Path, "/") {
		v1Path = "/" + v1Path
	}
	base := strings.TrimRight(strings.TrimSpace(a.core.APIClient.BaseURL()), "/")
	if strings.HasSuffix(base, "/api/v1") {
		return v1Path
	}
	return "/api/v1" + v1Path
}

func (a *App) apiRoot() (string, error) {
	base := strings.TrimRight(strings.TrimSpace(a.core.APIClient.BaseURL()), "/")
	if base == "" {
		return "", fmt.Errorf("api base URL is empty")
	}
	if strings.HasSuffix(base, "/api/v1") {
		return strings.TrimSuffix(base, "/api/v1"), nil
	}
	return base, nil
}

func (a *App) doRequest(method, path string, query url.Values, payload interface{}) ([]byte, error) {
	return a.core.APIClient.Request(method, a.apiPath(path), query, payload)
}

func (a *App) printJSON(body []byte) error {
	cliutil.PrintJSON(body)
	return nil
}

// printDocContent extracts and prints just the "content" field from a docs JSON response.
func (a *App) printDocContent(body []byte) error {
	var doc struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return a.printJSON(body)
	}
	fmt.Print(doc.Content)
	return nil
}

type healthResponse struct {
	Status       string            `json:"status"`
	Service      string            `json:"service"`
	Version      string            `json:"version"`
	Timestamp    string            `json:"timestamp"`
	Dependencies map[string]string `json:"dependencies"`
}

func (a *App) cmdStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output raw JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	root, err := a.apiRoot()
	if err != nil {
		return err
	}

	client := cliutil.NewHTTPClient(cliutil.HTTPClientOptions{
		BaseOptions: cliutil.APIBaseOptions{DefaultBase: root},
		Timeout:     a.core.HTTPClient.Timeout(),
	})
	body, err := client.Do("GET", "/health", nil, nil)
	if err != nil {
		return err
	}

	if *jsonOut {
		return a.printJSON(body)
	}

	var parsed healthResponse
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Status != "" {
		fmt.Printf("Status: %s\n", parsed.Status)
		if parsed.Service != "" {
			fmt.Printf("Service: %s\n", parsed.Service)
		}
		if parsed.Version != "" {
			fmt.Printf("Version: %s\n", parsed.Version)
		}
		if parsed.Timestamp != "" {
			fmt.Printf("Timestamp: %s\n", parsed.Timestamp)
		}
		if len(parsed.Dependencies) > 0 {
			fmt.Println("Dependencies:")
			for key, value := range parsed.Dependencies {
				fmt.Printf("  %s: %s\n", key, value)
			}
		}
		return nil
	}

	return a.printJSON(body)
}

func (a *App) cmdSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	collection := fs.String("collection", "", "Collection to search")
	namespaces := fs.String("namespaces", "", "Comma-separated namespaces")
	visibility := fs.String("visibility", "", "Comma-separated visibility values")
	tags := fs.String("tags", "", "Comma-separated tags")
	ingestedAfter := fs.String("ingested-after", "", "Filter records ingested after RFC3339 timestamp")
	ingestedBefore := fs.String("ingested-before", "", "Filter records ingested before RFC3339 timestamp")
	limit := fs.Int("limit", 0, "Maximum number of results")
	threshold := fs.Float64("threshold", 0, "Score threshold")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if query == "" {
		return fmt.Errorf("usage: search <query> [--collection=...] [--namespaces=...] [--visibility=...] [--tags=...] [--ingested-after=...] [--ingested-before=...] [--limit=N] [--threshold=0.35]")
	}

	req := searchRequest{
		Query:          query,
		Collection:     strings.TrimSpace(*collection),
		Namespaces:     splitCSV(*namespaces),
		Visibility:     splitCSV(*visibility),
		Tags:           splitCSV(*tags),
		IngestedAfter:  strings.TrimSpace(*ingestedAfter),
		IngestedBefore: strings.TrimSpace(*ingestedBefore),
	}
	if *limit > 0 {
		req.Limit = *limit
	}
	if *threshold > 0 {
		req.Threshold = *threshold
	}

	body, err := a.doRequest("POST", "/knowledge/search", nil, req)
	if err != nil {
		return err
	}
	return a.printJSON(body)
}

func (a *App) cmdGraph(args []string) error {
	fs := flag.NewFlagSet("graph", flag.ContinueOnError)
	center := fs.String("center", "", "Center concept for the graph")
	collection := fs.String("collection", "", "Collection to search")
	namespaces := fs.String("namespaces", "", "Comma-separated namespaces")
	visibility := fs.String("visibility", "", "Comma-separated visibility values")
	tags := fs.String("tags", "", "Comma-separated tags")
	depth := fs.Int("depth", 0, "Graph traversal depth")
	limit := fs.Int("limit", 0, "Maximum number of nodes")
	threshold := fs.Float64("threshold", 0, "Score threshold")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	centerValue := strings.TrimSpace(*center)
	if centerValue == "" {
		centerValue = strings.TrimSpace(strings.Join(fs.Args(), " "))
	}
	if centerValue == "" {
		return fmt.Errorf("usage: graph --center <concept> [--depth=N] [--limit=N] [--collection=...] [--namespaces=...] [--visibility=...] [--tags=...] [--threshold=0.35]")
	}

	req := graphRequest{
		Center:     centerValue,
		Collection: strings.TrimSpace(*collection),
		Namespaces: splitCSV(*namespaces),
		Visibility: splitCSV(*visibility),
		Tags:       splitCSV(*tags),
	}
	if *depth > 0 {
		req.Depth = *depth
	}
	if *limit > 0 {
		req.Limit = *limit
	}
	if *threshold > 0 {
		req.Threshold = *threshold
	}

	body, err := a.doRequest("POST", "/knowledge/graph", nil, req)
	if err != nil {
		return err
	}
	return a.printJSON(body)
}

func (a *App) docsUsage() string {
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

func (a *App) cmdDocs(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stdout, a.docsUsage())
		return nil
	}
	subcommand := strings.TrimSpace(args[0])
	switch subcommand {
	case "help", "--help", "-help", "-h":
		fmt.Fprintln(os.Stdout, a.docsUsage())
		return nil
	case "search-files":
		return a.cmdDocsSearchFiles(args[1:])
	case "search-text":
		return a.cmdDocsSearchText(args[1:])
	case "search-deep":
		return a.cmdDocsSearchDeep(args[1:])
	case "scenarios":
		return a.cmdDocsScenarios(args[1:])
	case "tree":
		return a.cmdDocsTree(args[1:])
	case "health":
		return a.cmdDocsHealth(args[1:])
	case "view":
		return a.cmdDocsView(args[1:])
	case "reset":
		return a.cmdDocsReset(args[1:])
	case "heal":
		return a.cmdDocsHeal(args[1:])
	case "heal-status":
		return a.cmdDocsHealStatus(args[1:])
	case "autofix":
		return a.cmdDocsAutoFix(args[1:])
	case "read":
		return a.cmdDocsRead(args[1:])
	case "add":
		return a.cmdDocsAdd(args[1:])
	case "stats":
		return a.cmdDocsStats(args[1:])
	case "templates":
		return a.cmdDocsTemplates(args[1:])
	case "template":
		return a.cmdDocsTemplate(args[1:])
	case "audit":
		return a.cmdDocsAudit(args[1:])
	default:
		return fmt.Errorf("unknown docs subcommand: %s\n\n%s", subcommand, a.docsUsage())
	}
}

func (a *App) cmdDocsSearchFiles(args []string) error {
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

	req := docsFileSearchRequest{
		Pattern:        patternValue,
		Scope:          strings.TrimSpace(*scope),
		Scenario:       strings.TrimSpace(*scenario),
		BasePath:       strings.TrimSpace(*basePath),
		IncludeContent: *includeContent,
	}
	if *limit > 0 {
		req.Limit = *limit
	}

	body, err := a.doRequest("POST", "/docs/search/files", nil, req)
	if err != nil {
		return err
	}
	return a.printJSON(body)
}

func (a *App) cmdDocsSearchText(args []string) error {
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

	req := docsTextSearchRequest{
		Query:         queryValue,
		Scope:         strings.TrimSpace(*scope),
		Scenario:      strings.TrimSpace(*scenario),
		BasePath:      strings.TrimSpace(*basePath),
		FileTypes:     splitCSV(*fileTypes),
		CaseSensitive: *caseSensitive,
		ContextLines:  *contextLines,
	}
	if *limit > 0 {
		req.Limit = *limit
	}

	body, err := a.doRequest("POST", "/docs/search/text", nil, req)
	if err != nil {
		return err
	}
	return a.printJSON(body)
}

func (a *App) cmdDocsSearchDeep(args []string) error {
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

	req := docsDeepSearchRequest{
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

	body, err := a.doRequest("POST", "/docs/search/deep", nil, req)
	if err != nil {
		return err
	}
	if !*wait {
		return a.printJSON(body)
	}

	var job docsDeepSearchJob
	if err := json.Unmarshal(body, &job); err != nil {
		return fmt.Errorf("failed to decode job: %w", err)
	}
	if job.JobID == "" {
		return fmt.Errorf("missing job id in response")
	}
	if job.Status == "completed" || job.Status == "failed" {
		return a.printJSON(body)
	}

	timeout := time.Duration(*timeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	deadline := time.Now().Add(timeout + 5*time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		statusBody, err := a.doRequest("GET", fmt.Sprintf("/docs/search/deep/%s", job.JobID), nil, nil)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(statusBody, &job); err != nil {
			return fmt.Errorf("failed to decode job status: %w", err)
		}
		if job.Status == "completed" || job.Status == "failed" {
			return a.printJSON(statusBody)
		}
	}
	return fmt.Errorf("deep search timed out before completion")
}

func (a *App) cmdDocsScenarios(args []string) error {
	fs := flag.NewFlagSet("docs scenarios", flag.ContinueOnError)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.doRequest("GET", "/scenarios", nil, nil)
	if err != nil {
		return err
	}
	return a.printJSON(body)
}

func (a *App) cmdDocsTree(args []string) error {
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

	body, err := a.doRequest("GET", fmt.Sprintf("/scenarios/%s/docs", scenarioValue), nil, nil)
	if err != nil {
		return err
	}
	return a.printJSON(body)
}

func (a *App) cmdDocsHealth(args []string) error {
	var jsonFromPositional bool
	args, jsonFromPositional = stripBoolFlag(args, "--json")

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

	body, err := a.doRequest("GET", fmt.Sprintf("/scenarios/%s/docs/health", scenarioValue), nil, nil)
	if err != nil {
		return err
	}
	if *jsonOut || jsonFromPositional {
		return a.printJSON(body)
	}

	var result docsHealthResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return a.printJSON(body)
	}

	fmt.Print(renderDocsHealthReport(result, scenarioValue))
	return nil
}

func (a *App) cmdDocsView(args []string) error {
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
		formatValue := strings.TrimSpace(*format)
		if formatValue != "" {
			query.Set("format", formatValue)
		}
	}

	body, err := a.doRequest("GET", "/docs/content", query, nil)
	if err != nil {
		return err
	}
	if wantJSON {
		return a.printJSON(body)
	}
	return a.printDocContent(body)
}

func (a *App) cmdDocsReset(args []string) error {
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

	req := docsResetRequest{
		Path:           pathValue,
		MaxAgeDays:     *maxAgeDays,
		KeepMinEntries: *keepMin,
		PreviewOnly:    *preview,
	}

	body, err := a.doRequest("POST", "/docs/reset", nil, req)
	if err != nil {
		return err
	}
	return a.printJSON(body)
}

func (a *App) cmdDocsHeal(args []string) error {
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

	req := docsHealRequest{
		ScenarioName: scenarioValue,
		Issues:       splitCSV(*issues),
		AutoApprove:  *autoApprove,
		DryRun:       *dryRun,
	}

	body, err := a.doRequest("POST", fmt.Sprintf("/scenarios/%s/docs/heal", scenarioValue), nil, req)
	if err != nil {
		return err
	}
	if !*wait {
		return a.printJSON(body)
	}

	var job docsHealJob
	if err := json.Unmarshal(body, &job); err != nil {
		return err
	}
	if job.JobID == "" {
		return fmt.Errorf("healing job id missing")
	}
	for {
		statusBody, err := a.doRequest("GET", fmt.Sprintf("/docs/heal/%s", job.JobID), nil, nil)
		if err != nil {
			return err
		}
		var status docsHealJob
		if err := json.Unmarshal(statusBody, &status); err != nil {
			return err
		}
		if status.Status != "pending" && status.Status != "running" {
			return a.printJSON(statusBody)
		}
		time.Sleep(2 * time.Second)
	}
}

func (a *App) cmdDocsHealStatus(args []string) error {
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
	body, err := a.doRequest("GET", fmt.Sprintf("/docs/heal/%s", jobValue), nil, nil)
	if err != nil {
		return err
	}
	return a.printJSON(body)
}

func (a *App) cmdDocsAutoFix(args []string) error {
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

	req := docsAutoFixRequest{
		DryRun: *dryRun,
	}

	body, err := a.doRequest("POST", fmt.Sprintf("/scenarios/%s/docs/autofix", scenarioValue), nil, req)
	if err != nil {
		return err
	}

	var result docsAutoFixResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return a.printJSON(body)
	}

	if *dryRun {
		fmt.Println("Dry run — no files were moved.")
	}

	if len(result.Moved) > 0 {
		fmt.Printf("Moved %d file(s):\n", len(result.Moved))
		for _, m := range result.Moved {
			fmt.Printf("  %s → %s\n", m.FromPath, m.ToPath)
		}
	}
	if len(result.Skipped) > 0 {
		fmt.Printf("Skipped %d file(s):\n", len(result.Skipped))
		for _, s := range result.Skipped {
			fmt.Printf("  %s → %s (%s)\n", s.FromPath, s.ToPath, s.Reason)
		}
	}
	if len(result.Moved) == 0 && len(result.Skipped) == 0 {
		fmt.Println("No misplaced docs to fix.")
	}

	fmt.Printf("Health: %.0f%% → %.0f%%\n", result.HealthBefore*100, result.HealthAfter*100)
	return nil
}

func (a *App) cmdDocsRead(args []string) error {
	fs := flag.NewFlagSet("docs read", flag.ContinueOnError)
	scenario := fs.String("scenario", "", "Scenario name")
	doc := fs.String("doc", "", "Document type (problems, progress, seams, invariants, assumptions, error-semantics, security-posture, temporal-flows, coherence-notes, experience-audit, quickstart, architecture, glossary, prd, readme, manifest)")
	format := fs.String("format", "raw", "Output format: raw (default, content only) or json (full response)")
	jsonOut := fs.Bool("json", false, "Output full JSON response (shorthand for --format json)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	scenarioValue := strings.TrimSpace(*scenario)
	docValue := strings.TrimSpace(*doc)

	// Support positional: docs read <scenario> <type>
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
		formatValue := strings.TrimSpace(*format)
		if formatValue != "" {
			query.Set("format", formatValue)
		}
	}

	body, err := a.doRequest("GET", fmt.Sprintf("/scenarios/%s/docs/%s/content", scenarioValue, docValue), query, nil)
	if err != nil {
		return err
	}
	if wantJSON {
		return a.printJSON(body)
	}
	return a.printDocContent(body)
}

func (a *App) cmdDocsAdd(args []string) error {
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

	req := docsAddEntryRequest{
		Title:  titleValue,
		Body:   strings.TrimSpace(*body),
		Author: strings.TrimSpace(*author),
		Status: strings.TrimSpace(*status),
	}

	respBody, err := a.doRequest("POST", fmt.Sprintf("/scenarios/%s/docs/%s/entries", scenarioValue, docValue), nil, req)
	if err != nil {
		return err
	}
	return a.printJSON(respBody)
}

func (a *App) cmdDocsStats(args []string) error {
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

	body, err := a.doRequest("GET", "/docs/stats", query, nil)
	if err != nil {
		return err
	}
	return a.printJSON(body)
}

func (a *App) cmdDocsTemplates(args []string) error {
	fs := flag.NewFlagSet("docs templates", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output raw JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.doRequest("GET", "/docs/templates", nil, nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		return a.printJSON(body)
	}

	var items []docsTemplateListItem
	if err := json.Unmarshal(body, &items); err != nil {
		return a.printJSON(body)
	}

	for _, item := range items {
		fmt.Printf("%-20s %s\n", item.DocType, item.Purpose)
	}
	return nil
}

func (a *App) cmdDocsTemplate(args []string) error {
	fs := flag.NewFlagSet("docs template", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output full JSON response")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	docType := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if docType == "" {
		return fmt.Errorf("usage: docs template <type> [--json]")
	}

	body, err := a.doRequest("GET", fmt.Sprintf("/docs/templates/%s", docType), nil, nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		return a.printJSON(body)
	}

	var detail docsTemplateDetailResponse
	if err := json.Unmarshal(body, &detail); err != nil {
		return a.printJSON(body)
	}
	fmt.Print(detail.Content)
	return nil
}

func (a *App) cmdDocsAudit(args []string) error {
	var jsonFromPositional bool
	args, jsonFromPositional = stripBoolFlag(args, "--json")

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

	body, err := a.doRequest("GET", fmt.Sprintf("/scenarios/%s/docs/audit", scenarioValue), nil, nil)
	if err != nil {
		return err
	}
	if *jsonOut || jsonFromPositional {
		return a.printJSON(body)
	}

	var result docsAuditResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return a.printJSON(body)
	}
	fmt.Print(renderDocsAuditReport(result, scenarioValue))

	return nil
}

func (a *App) cmdIngest(args []string) error {
	fs := flag.NewFlagSet("ingest", flag.ContinueOnError)
	namespace := fs.String("namespace", "", "Namespace for the record")
	collection := fs.String("collection", "", "Target collection")
	visibility := fs.String("visibility", "", "Visibility (shared|private|restricted)")
	recordID := fs.String("record-id", "", "Record ID (optional)")
	externalID := fs.String("external-id", "", "External ID (optional)")
	tags := fs.String("tags", "", "Comma-separated tags")
	metadataRaw := fs.String("metadata", "", "Metadata as JSON object")
	source := fs.String("source", "", "Source identifier")
	sourceType := fs.String("source-type", "", "Source type")
	content := fs.String("content", "", "Content string (or pass as arg/stdin)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if strings.TrimSpace(*namespace) == "" {
		return fmt.Errorf("--namespace is required")
	}

	contentValue, err := readContent(*content, fs.Args())
	if err != nil {
		return err
	}
	if contentValue == "" {
		return fmt.Errorf("content is required (use --content, positional args, or stdin)")
	}

	metadata, err := parseMetadata(*metadataRaw)
	if err != nil {
		return err
	}

	req := ingestRequest{
		Namespace:  strings.TrimSpace(*namespace),
		Collection: strings.TrimSpace(*collection),
		RecordID:   strings.TrimSpace(*recordID),
		ExternalID: strings.TrimSpace(*externalID),
		Content:    contentValue,
		Tags:       splitCSV(*tags),
		Metadata:   metadata,
		Visibility: strings.TrimSpace(*visibility),
		Source:     strings.TrimSpace(*source),
		SourceType: strings.TrimSpace(*sourceType),
	}

	body, err := a.doRequest("POST", "/knowledge/records/upsert", nil, req)
	if err != nil {
		return err
	}
	return a.printJSON(body)
}

func (a *App) cmdIngestJob(args []string) error {
	fs := flag.NewFlagSet("ingest-job", flag.ContinueOnError)
	namespace := fs.String("namespace", "", "Namespace for the document")
	collection := fs.String("collection", "", "Target collection")
	visibility := fs.String("visibility", "", "Visibility (shared|private|restricted)")
	documentID := fs.String("document-id", "", "Document ID (optional)")
	externalID := fs.String("external-id", "", "External ID (optional)")
	tags := fs.String("tags", "", "Comma-separated tags")
	metadataRaw := fs.String("metadata", "", "Metadata as JSON object")
	source := fs.String("source", "", "Source identifier")
	sourceType := fs.String("source-type", "", "Source type")
	chunkSize := fs.Int("chunk-size", 0, "Chunk size (default handled by API)")
	chunkOverlap := fs.Int("chunk-overlap", 0, "Chunk overlap (default handled by API)")
	content := fs.String("content", "", "Content string (or pass as arg/stdin)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	chunkSizeSet := false
	chunkOverlapSet := false
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "chunk-size":
			chunkSizeSet = true
		case "chunk-overlap":
			chunkOverlapSet = true
		}
	})

	if strings.TrimSpace(*namespace) == "" {
		return fmt.Errorf("--namespace is required")
	}

	contentValue, err := readContent(*content, fs.Args())
	if err != nil {
		return err
	}
	if contentValue == "" {
		return fmt.Errorf("content is required (use --content, positional args, or stdin)")
	}

	metadata, err := parseMetadata(*metadataRaw)
	if err != nil {
		return err
	}

	var chunkSizeValue *int
	if chunkSizeSet {
		chunkSizeValue = chunkSize
	}
	var chunkOverlapValue *int
	if chunkOverlapSet {
		chunkOverlapValue = chunkOverlap
	}

	req := ingestJobRequest{
		Namespace:    strings.TrimSpace(*namespace),
		Collection:   strings.TrimSpace(*collection),
		DocumentID:   strings.TrimSpace(*documentID),
		ExternalID:   strings.TrimSpace(*externalID),
		Content:      contentValue,
		Tags:         splitCSV(*tags),
		Metadata:     metadata,
		Visibility:   strings.TrimSpace(*visibility),
		Source:       strings.TrimSpace(*source),
		SourceType:   strings.TrimSpace(*sourceType),
		ChunkSize:    chunkSizeValue,
		ChunkOverlap: chunkOverlapValue,
	}

	body, err := a.doRequest("POST", "/ingest/jobs", nil, req)
	if err != nil {
		return err
	}
	return a.printJSON(body)
}

func (a *App) cmdJobStatus(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: job-status <job_id>")
	}
	jobID := strings.TrimSpace(args[0])
	if jobID == "" {
		return fmt.Errorf("job_id is required")
	}

	body, err := a.doRequest("GET", fmt.Sprintf("/ingest/jobs/%s", jobID), nil, nil)
	if err != nil {
		return err
	}
	return a.printJSON(body)
}

func (a *App) cmdIngestHealth(args []string) error {
	fs := flag.NewFlagSet("ingest-health", flag.ContinueOnError)
	watch := fs.Bool("watch", false, "Poll ingest health continuously")
	intervalRaw := fs.String("interval", "5s", "Polling interval when --watch is set")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if !*watch {
		body, err := a.doRequest("GET", "/ingest/health", nil, nil)
		if err != nil {
			return err
		}
		return a.printJSON(body)
	}

	interval, err := parseDuration(*intervalRaw)
	if err != nil {
		return err
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		body, err := a.doRequest("GET", "/ingest/health", nil, nil)
		if err != nil {
			return err
		}
		fmt.Printf("== %s ==\n", time.Now().UTC().Format(time.RFC3339))
		cliutil.PrintJSON(body)
		<-ticker.C
	}
}

func (a *App) cmdCollectionDiagnostics(args []string) error {
	leadingCollection, parseArgs := splitLeadingPositional(args)
	fs := flag.NewFlagSet("collection-diagnostics", flag.ContinueOnError)
	collection := fs.String("collection", "", "Collection to inspect")
	mode := fs.String("mode", "sample", "Diagnostics mode: sample or full")
	limit := fs.Int("limit", 0, "Maximum points to inspect")
	if err := fs.Parse(parseArgs); err != nil {
		return err
	}

	collectionValue := strings.TrimSpace(*collection)
	if collectionValue == "" {
		collectionValue = leadingCollection
	}
	if collectionValue == "" && len(fs.Args()) > 0 {
		collectionValue = strings.TrimSpace(fs.Args()[0])
	}
	if collectionValue == "" {
		return fmt.Errorf("usage: collection-diagnostics <collection> [--mode=sample|full] [--limit=N]")
	}

	query := url.Values{}
	modeValue := strings.TrimSpace(*mode)
	if modeValue != "" {
		query.Set("mode", modeValue)
	}
	if *limit > 0 {
		query.Set("limit", strconv.Itoa(*limit))
	}

	path := fmt.Sprintf("/knowledge/collections/%s/diagnostics", url.PathEscape(collectionValue))
	body, err := a.doRequest("GET", path, query, nil)
	if err != nil {
		return err
	}
	return a.printJSON(body)
}

func (a *App) cmdCollectionPruneStale(args []string) error {
	leadingCollection, parseArgs := splitLeadingPositional(args)
	fs := flag.NewFlagSet("collection-prune-stale", flag.ContinueOnError)
	collection := fs.String("collection", "", "Collection to clean")
	dryRun := fs.Bool("dry-run", true, "Preview candidates without deleting")
	apply := fs.Bool("apply", false, "Execute deletions (overrides --dry-run)")
	maxDeletes := fs.Int("max-deletes", 0, "Maximum points to delete")
	if err := fs.Parse(parseArgs); err != nil {
		return err
	}

	collectionValue := strings.TrimSpace(*collection)
	if collectionValue == "" {
		collectionValue = leadingCollection
	}
	if collectionValue == "" && len(fs.Args()) > 0 {
		collectionValue = strings.TrimSpace(fs.Args()[0])
	}
	if collectionValue == "" {
		return fmt.Errorf("usage: collection-prune-stale <collection> [--dry-run] [--apply] [--max-deletes=N]")
	}

	req := collectionMaintenanceRequest{
		DryRun:     *dryRun,
		MaxDeletes: *maxDeletes,
	}
	if *apply {
		req.DryRun = false
	}
	path := fmt.Sprintf("/knowledge/collections/%s/maintenance/prune-stale-chunks", url.PathEscape(collectionValue))
	body, err := a.doRequest("POST", path, nil, req)
	if err != nil {
		return err
	}
	return a.printJSON(body)
}

func (a *App) cmdCollectionDedupe(args []string) error {
	leadingCollection, parseArgs := splitLeadingPositional(args)
	fs := flag.NewFlagSet("collection-dedupe", flag.ContinueOnError)
	collection := fs.String("collection", "", "Collection to clean")
	dryRun := fs.Bool("dry-run", true, "Preview candidates without deleting")
	apply := fs.Bool("apply", false, "Execute deletions (overrides --dry-run)")
	maxDeletes := fs.Int("max-deletes", 0, "Maximum points to delete")
	if err := fs.Parse(parseArgs); err != nil {
		return err
	}

	collectionValue := strings.TrimSpace(*collection)
	if collectionValue == "" {
		collectionValue = leadingCollection
	}
	if collectionValue == "" && len(fs.Args()) > 0 {
		collectionValue = strings.TrimSpace(fs.Args()[0])
	}
	if collectionValue == "" {
		return fmt.Errorf("usage: collection-dedupe <collection> [--dry-run] [--apply] [--max-deletes=N]")
	}

	req := collectionMaintenanceRequest{
		DryRun:     *dryRun,
		MaxDeletes: *maxDeletes,
	}
	if *apply {
		req.DryRun = false
	}
	path := fmt.Sprintf("/knowledge/collections/%s/maintenance/dedupe-content", url.PathEscape(collectionValue))
	body, err := a.doRequest("POST", path, nil, req)
	if err != nil {
		return err
	}
	return a.printJSON(body)
}

func (a *App) cmdDocumentDelete(args []string) error {
	fs := flag.NewFlagSet("document-delete", flag.ContinueOnError)
	namespace := fs.String("namespace", "", "Namespace for the document")
	collection := fs.String("collection", "", "Collection override")
	documentID := fs.String("document-id", "", "Document ID")
	externalID := fs.String("external-id", "", "External ID mapped to document")
	dryRun := fs.Bool("dry-run", true, "Preview candidates without deleting")
	apply := fs.Bool("apply", false, "Execute deletions (overrides --dry-run)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	namespaceValue := strings.TrimSpace(*namespace)
	if namespaceValue == "" {
		return fmt.Errorf("--namespace is required")
	}
	documentIDValue := strings.TrimSpace(*documentID)
	if documentIDValue == "" && len(fs.Args()) > 0 {
		documentIDValue = strings.TrimSpace(fs.Args()[0])
	}
	externalIDValue := strings.TrimSpace(*externalID)
	if documentIDValue == "" && externalIDValue == "" {
		return fmt.Errorf("usage: document-delete --namespace <namespace> [--collection=name] (--document-id <id>|--external-id <id>) [--dry-run] [--apply]")
	}

	req := documentDeleteRequest{
		Namespace:  namespaceValue,
		Collection: strings.TrimSpace(*collection),
		DocumentID: documentIDValue,
		ExternalID: externalIDValue,
		DryRun:     *dryRun,
	}
	if *apply {
		req.DryRun = false
	}

	body, err := a.doRequest("POST", "/knowledge/documents/delete", nil, req)
	if err != nil {
		return err
	}
	return a.printJSON(body)
}

func (a *App) cmdHealth(args []string) error {
	fs := flag.NewFlagSet("health", flag.ContinueOnError)
	watch := fs.Bool("watch", false, "Poll health metrics continuously")
	intervalRaw := fs.String("interval", "5s", "Polling interval when --watch is set")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if !*watch {
		body, err := a.doRequest("GET", "/knowledge/health", nil, nil)
		if err != nil {
			return err
		}
		return a.printJSON(body)
	}

	interval, err := parseDuration(*intervalRaw)
	if err != nil {
		return err
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		body, err := a.doRequest("GET", "/knowledge/health", nil, nil)
		if err != nil {
			return err
		}
		fmt.Printf("== %s ==\n", time.Now().UTC().Format(time.RFC3339))
		cliutil.PrintJSON(body)
		<-ticker.C
	}
}

type searchRequest struct {
	Query          string   `json:"query"`
	Collection     string   `json:"collection,omitempty"`
	Namespaces     []string `json:"namespaces,omitempty"`
	Visibility     []string `json:"visibility,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	IngestedAfter  string   `json:"ingested_after,omitempty"`
	IngestedBefore string   `json:"ingested_before,omitempty"`
	Limit          int      `json:"limit,omitempty"`
	Threshold      float64  `json:"threshold,omitempty"`
}

type graphRequest struct {
	Center     string   `json:"center_concept"`
	Collection string   `json:"collection,omitempty"`
	Namespaces []string `json:"namespaces,omitempty"`
	Visibility []string `json:"visibility,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Depth      int      `json:"depth,omitempty"`
	Limit      int      `json:"limit,omitempty"`
	Threshold  float64  `json:"threshold,omitempty"`
}

type ingestRequest struct {
	Namespace  string                 `json:"namespace"`
	Collection string                 `json:"collection,omitempty"`
	RecordID   string                 `json:"record_id,omitempty"`
	ExternalID string                 `json:"external_id,omitempty"`
	Content    string                 `json:"content"`
	Tags       []string               `json:"tags,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Visibility string                 `json:"visibility,omitempty"`
	Source     string                 `json:"source,omitempty"`
	SourceType string                 `json:"source_type,omitempty"`
}

type ingestJobRequest struct {
	Namespace    string                 `json:"namespace"`
	Collection   string                 `json:"collection,omitempty"`
	DocumentID   string                 `json:"document_id,omitempty"`
	ExternalID   string                 `json:"external_id,omitempty"`
	Content      string                 `json:"content"`
	Tags         []string               `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Visibility   string                 `json:"visibility,omitempty"`
	Source       string                 `json:"source,omitempty"`
	SourceType   string                 `json:"source_type,omitempty"`
	ChunkSize    *int                   `json:"chunk_size,omitempty"`
	ChunkOverlap *int                   `json:"chunk_overlap,omitempty"`
}

type collectionMaintenanceRequest struct {
	DryRun     bool `json:"dry_run"`
	MaxDeletes int  `json:"max_deletes,omitempty"`
}

type documentDeleteRequest struct {
	Namespace  string `json:"namespace"`
	Collection string `json:"collection,omitempty"`
	DocumentID string `json:"document_id,omitempty"`
	ExternalID string `json:"external_id,omitempty"`
	DryRun     bool   `json:"dry_run,omitempty"`
}

type docsFileSearchRequest struct {
	Pattern        string `json:"pattern"`
	Scope          string `json:"scope,omitempty"`
	Scenario       string `json:"scenario,omitempty"`
	BasePath       string `json:"base_path,omitempty"`
	Limit          int    `json:"limit,omitempty"`
	IncludeContent bool   `json:"include_content,omitempty"`
}

type docsTextSearchRequest struct {
	Query         string   `json:"query"`
	Scope         string   `json:"scope,omitempty"`
	Scenario      string   `json:"scenario,omitempty"`
	BasePath      string   `json:"base_path,omitempty"`
	FileTypes     []string `json:"file_types,omitempty"`
	CaseSensitive bool     `json:"case_sensitive,omitempty"`
	Limit         int      `json:"limit,omitempty"`
	ContextLines  int      `json:"context_lines,omitempty"`
}

type docsDeepSearchRequest struct {
	Query          string `json:"query"`
	Scope          string `json:"scope,omitempty"`
	Scenario       string `json:"scenario,omitempty"`
	BasePath       string `json:"base_path,omitempty"`
	MaxResults     int    `json:"max_results,omitempty"`
	FollowRefs     *bool  `json:"follow_refs,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

type docsDeepSearchJob struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type docsAddEntryRequest struct {
	Title  string `json:"title"`
	Body   string `json:"body,omitempty"`
	Author string `json:"author,omitempty"`
	Status string `json:"status,omitempty"`
}

type docsResetRequest struct {
	Path           string `json:"path"`
	MaxAgeDays     int    `json:"max_age_days,omitempty"`
	KeepMinEntries int    `json:"keep_min_entries,omitempty"`
	PreviewOnly    bool   `json:"preview_only,omitempty"`
}

type docsHealRequest struct {
	ScenarioName string   `json:"scenario_name"`
	Issues       []string `json:"issues,omitempty"`
	AutoApprove  bool     `json:"auto_approve,omitempty"`
	DryRun       bool     `json:"dry_run,omitempty"`
}

type docsHealJob struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type docsAutoFixRequest struct {
	DryRun bool `json:"dry_run,omitempty"`
}

type docsAutoFixResponse struct {
	ScenarioName string               `json:"scenario_name"`
	Moved        []docsAutoFixMoved   `json:"moved"`
	Skipped      []docsAutoFixSkipped `json:"skipped"`
	HealthBefore float64              `json:"health_before"`
	HealthAfter  float64              `json:"health_after"`
	DryRun       bool                 `json:"dry_run"`
}

type docsAutoFixMoved struct {
	FromPath string `json:"from_path"`
	ToPath   string `json:"to_path"`
	DocType  string `json:"doc_type"`
}

type docsAutoFixSkipped struct {
	FromPath string `json:"from_path"`
	ToPath   string `json:"to_path"`
	DocType  string `json:"doc_type"`
	Reason   string `json:"reason"`
}

type docsTemplateListItem struct {
	DocType      string `json:"doc_type"`
	ExpectedPath string `json:"expected_path"`
	Purpose      string `json:"purpose"`
}

type docsTemplateDetailResponse struct {
	DocType      string `json:"doc_type"`
	ExpectedPath string `json:"expected_path"`
	Purpose      string `json:"purpose"`
	Content      string `json:"content"`
}

type docsAuditResponse struct {
	ScenarioName        string                      `json:"scenario_name"`
	HealthScore         float64                     `json:"health_score"`
	TotalDocs           int                         `json:"total_docs"`
	Infrastructure      *docsAuditInfrastructure    `json:"infrastructure"`
	CodeWithoutDocRefs  []docsAuditUndocumentedFile `json:"code_without_doc_refs"`
	BrokenCodeRefs      []docsAuditBrokenRef        `json:"broken_code_refs"`
	OrphanedDocs        []string                    `json:"orphaned_docs"`
	DuplicateTitles     []docsAuditDuplicateTitle   `json:"duplicate_titles"`
	UndocumentedTargets []string                    `json:"undocumented_targets"`
}

type docsAuditInfrastructure struct {
	MisplacedDocs []docsAuditMisplacedDoc `json:"misplaced_docs"`
	MissingDocs   []string                `json:"missing_docs"`
	ExtraDocs     []string                `json:"extra_docs"`
	TemporaryDocs []string                `json:"temporary_docs"`
}

type docsAuditMisplacedDoc struct {
	ActualPath   string `json:"actual_path"`
	ExpectedPath string `json:"expected_path"`
	DocType      string `json:"doc_type"`
	Severity     string `json:"severity"`
}

func (d *docsAuditMisplacedDoc) UnmarshalJSON(data []byte) error {
	type alias docsAuditMisplacedDoc
	var tagged alias
	if err := json.Unmarshal(data, &tagged); err != nil {
		return err
	}
	type legacy struct {
		ActualPath   string `json:"ActualPath"`
		ExpectedPath string `json:"ExpectedPath"`
		DocType      string `json:"DocType"`
		Severity     string `json:"Severity"`
	}
	var legacyValue legacy
	if err := json.Unmarshal(data, &legacyValue); err != nil {
		return err
	}
	if strings.TrimSpace(tagged.ActualPath) == "" {
		tagged.ActualPath = legacyValue.ActualPath
	}
	if strings.TrimSpace(tagged.ExpectedPath) == "" {
		tagged.ExpectedPath = legacyValue.ExpectedPath
	}
	if strings.TrimSpace(tagged.DocType) == "" {
		tagged.DocType = legacyValue.DocType
	}
	if strings.TrimSpace(tagged.Severity) == "" {
		tagged.Severity = legacyValue.Severity
	}
	*d = docsAuditMisplacedDoc(tagged)
	return nil
}

func (d *docsAuditInfrastructure) UnmarshalJSON(data []byte) error {
	type alias docsAuditInfrastructure
	var tagged alias
	if err := json.Unmarshal(data, &tagged); err != nil {
		return err
	}
	type legacy struct {
		MisplacedDocs []docsAuditMisplacedDoc `json:"MisplacedDocs"`
		MissingDocs   []string                `json:"MissingDocs"`
		ExtraDocs     []string                `json:"ExtraDocs"`
		TemporaryDocs []string                `json:"TemporaryDocs"`
	}
	var legacyValue legacy
	if err := json.Unmarshal(data, &legacyValue); err != nil {
		return err
	}
	if len(tagged.MisplacedDocs) == 0 {
		tagged.MisplacedDocs = legacyValue.MisplacedDocs
	}
	if len(tagged.MissingDocs) == 0 {
		tagged.MissingDocs = legacyValue.MissingDocs
	}
	if len(tagged.ExtraDocs) == 0 {
		tagged.ExtraDocs = legacyValue.ExtraDocs
	}
	if len(tagged.TemporaryDocs) == 0 {
		tagged.TemporaryDocs = legacyValue.TemporaryDocs
	}
	*d = docsAuditInfrastructure(tagged)
	return nil
}

type docsAuditUndocumentedFile struct {
	Path            string `json:"path"`
	ExportedSymbols int    `json:"exported_symbols"`
}

type docsAuditBrokenRef struct {
	DocPath string `json:"doc_path"`
	Line    int    `json:"line"`
	Target  string `json:"target"`
}

type docsAuditDuplicateTitle struct {
	Title string   `json:"title"`
	Files []string `json:"files"`
}

type docsHealthResponse struct {
	ScenarioName  string                  `json:"scenario_name"`
	HealthScore   float64                 `json:"health_score"`
	TotalDocs     int                     `json:"total_docs"`
	MisplacedDocs []docsAuditMisplacedDoc `json:"misplaced_docs"`
	MissingDocs   []string                `json:"missing_docs"`
	ExtraDocs     []string                `json:"extra_docs"`
	TemporaryDocs []string                `json:"temporary_docs"`
	CanAutoFix    bool                    `json:"can_auto_fix"`
	FixCategory   string                  `json:"fix_category"`
}

type auditSeverity string

const (
	auditSeverityOK   auditSeverity = "OK"
	auditSeverityWarn auditSeverity = "WARN"
	auditSeverityFail auditSeverity = "FAIL"
)

type triageItem struct {
	priority int
	sortKey  string
	text     string
}

type manualGroup struct {
	name  string
	items []triageItem
}

type nextStep struct {
	description string
	command     string
}

func renderDocsAuditReport(result docsAuditResponse, fallbackScenario string) string {
	scenario := strings.TrimSpace(result.ScenarioName)
	if scenario == "" {
		scenario = strings.TrimSpace(fallbackScenario)
	}
	if scenario == "" {
		scenario = "unknown"
	}

	misplaced, missing, extra, temporary := infraAuditSlices(result.Infrastructure)
	autoItems := buildAutoFixItems(misplaced)
	agentItems := buildAgentItems(missing, extra)
	manualGroups := buildManualGroups(result, temporary)

	autoCount := len(autoItems)
	agentCount := len(agentItems)
	manualCount := countManualGroupItems(manualGroups)
	totalFindings := autoCount + agentCount + manualCount

	status := classifyAuditStatus(totalFindings, result)
	healthPct := int(result.HealthScore*100 + 0.5)
	if healthPct < 0 {
		healthPct = 0
	}
	if healthPct > 100 {
		healthPct = 100
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Documentation Audit: %s\n", scenario)
	fmt.Fprintf(&b, "Status: %s", status)
	drivers := auditStatusDrivers(result)
	if len(drivers) > 0 {
		fmt.Fprintf(&b, " (drivers: %s)", strings.Join(drivers, ", "))
	}
	fmt.Fprintf(&b, "\nHealth: %d%% (%d docs", healthPct, result.TotalDocs)
	if len(misplaced)+len(missing)+len(extra)+len(temporary) > 0 {
		fmt.Fprintf(&b, "; %d misplaced, %d missing, %d extra, %d temporary", len(misplaced), len(missing), len(extra), len(temporary))
	}
	b.WriteString(")\n")
	fmt.Fprintf(&b, "Findings: %d total\n\n", totalFindings)

	b.WriteString("Triage\n")
	wroteAny := false
	if len(autoItems) > 0 {
		writeTriageBucket(&b, "Auto-fix now", autoItems)
		wroteAny = true
	}
	if len(agentItems) > 0 {
		writeTriageBucket(&b, "Agent repair", agentItems)
		wroteAny = true
	}
	if manualCount > 0 {
		writeManualReviewBucket(&b, manualGroups)
		wroteAny = true
	}
	if !wroteAny {
		b.WriteString("- No findings\n")
	}

	b.WriteString("\nNext steps\n")
	steps := nextStepGuidance(scenario, autoCount, agentCount, manualCount, result.HealthScore)
	for i, step := range steps {
		fmt.Fprintf(&b, "%d. %s\n", i+1, step.description)
		fmt.Fprintf(&b, "   %s\n", step.command)
	}

	return b.String()
}

func classifyAuditStatus(totalFindings int, result docsAuditResponse) auditSeverity {
	if totalFindings == 0 {
		return auditSeverityOK
	}
	if len(result.BrokenCodeRefs) > 0 || len(result.UndocumentedTargets) > 0 {
		return auditSeverityFail
	}
	return auditSeverityWarn
}

func infraAuditSlices(infra *docsAuditInfrastructure) ([]docsAuditMisplacedDoc, []string, []string, []string) {
	if infra == nil {
		return nil, nil, nil, nil
	}
	return infra.MisplacedDocs, infra.MissingDocs, infra.ExtraDocs, infra.TemporaryDocs
}

func buildAutoFixItems(misplaced []docsAuditMisplacedDoc) []triageItem {
	items := make([]triageItem, 0, len(misplaced))
	for _, doc := range misplaced {
		actual := strings.TrimSpace(doc.ActualPath)
		expected := strings.TrimSpace(doc.ExpectedPath)
		if actual == "" && expected == "" {
			continue
		}
		items = append(items, triageItem{
			priority: 0,
			sortKey:  strings.ToLower(actual + "|" + expected),
			text:     fmt.Sprintf("%s -> %s", actual, expected),
		})
	}
	sortTriageItems(items)
	return items
}

func buildAgentItems(missing []string, extra []string) []triageItem {
	items := make([]triageItem, 0, len(missing)+len(extra))
	for _, value := range missing {
		docType := strings.TrimSpace(value)
		if docType == "" {
			continue
		}
		items = append(items, triageItem{
			priority: 0,
			sortKey:  strings.ToLower(docType),
			text:     "Missing: " + docType,
		})
	}
	for _, value := range extra {
		path := strings.TrimSpace(value)
		if path == "" {
			continue
		}
		items = append(items, triageItem{
			priority: 1,
			sortKey:  strings.ToLower(path),
			text:     "Extra: " + path,
		})
	}
	sortTriageItems(items)
	return items
}

func sortTriageItems(items []triageItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].priority != items[j].priority {
			return items[i].priority < items[j].priority
		}
		return items[i].sortKey < items[j].sortKey
	})
}

func writeTriageBucket(b *strings.Builder, name string, items []triageItem) {
	fmt.Fprintf(b, "- %s (%d)\n", name, len(items))
	const maxExamples = 10
	limit := len(items)
	if limit > maxExamples {
		limit = maxExamples
	}
	for i := 0; i < limit; i++ {
		fmt.Fprintf(b, "  - %s\n", items[i].text)
	}
	if len(items) > maxExamples {
		fmt.Fprintf(b, "  ... +%d more\n", len(items)-maxExamples)
	}
}

func writeManualReviewBucket(b *strings.Builder, groups []manualGroup) {
	total := countManualGroupItems(groups)
	fmt.Fprintf(b, "- Manual review (%d)\n", total)
	const maxExamples = 10
	for _, group := range groups {
		if len(group.items) == 0 {
			continue
		}
		fmt.Fprintf(b, "  - %s (%d)\n", group.name, len(group.items))
		limit := len(group.items)
		if limit > maxExamples {
			limit = maxExamples
		}
		for i := 0; i < limit; i++ {
			fmt.Fprintf(b, "    - %s\n", group.items[i].text)
		}
		if len(group.items) > maxExamples {
			fmt.Fprintf(b, "    ... +%d more\n", len(group.items)-maxExamples)
		}
	}
}

func nextStepGuidance(scenario string, autoCount int, agentCount int, manualCount int, healthScore float64) []nextStep {
	steps := make([]nextStep, 0, 4)
	if autoCount > 0 {
		steps = append(steps, nextStep{
			description: "To apply deterministic quick fixes for misplaced docs, run:",
			command:     fmt.Sprintf("knowledge-observatory docs autofix %s", scenario),
		})
	}
	if agentCount > 0 {
		steps = append(steps, nextStep{
			description: "To run agent-driven repair for missing/extra docs, run:",
			command:     fmt.Sprintf("knowledge-observatory docs heal %s --wait", scenario),
		})
	}
	if manualCount > 0 {
		steps = append(steps, nextStep{
			description: "To inspect full findings in machine-readable form, run:",
			command:     fmt.Sprintf("knowledge-observatory docs audit %s --json", scenario),
		})
	}
	if healthScore < 0.9995 {
		steps = append(steps, nextStep{
			description: "To see a detailed documentation-health breakdown and penalties, run:",
			command:     fmt.Sprintf("knowledge-observatory docs health %s", scenario),
		})
	}
	if len(steps) == 0 {
		steps = append(steps, nextStep{
			description: "No action required. To verify again later, run:",
			command:     fmt.Sprintf("knowledge-observatory docs audit %s", scenario),
		})
	}
	return steps
}

func auditStatusDrivers(result docsAuditResponse) []string {
	drivers := make([]string, 0, 2)
	if len(result.BrokenCodeRefs) > 0 {
		drivers = append(drivers, fmt.Sprintf("%d broken [CODE:] refs", len(result.BrokenCodeRefs)))
	}
	if len(result.UndocumentedTargets) > 0 {
		drivers = append(drivers, fmt.Sprintf("%d undocumented operational targets", len(result.UndocumentedTargets)))
	}
	return drivers
}

func buildManualGroups(result docsAuditResponse, temporary []string) []manualGroup {
	groups := []manualGroup{
		{name: "No DOC refs", items: make([]triageItem, 0, len(result.CodeWithoutDocRefs))},
		{name: "Broken [CODE:] refs", items: make([]triageItem, 0, len(result.BrokenCodeRefs))},
		{name: "Orphaned docs", items: make([]triageItem, 0, len(result.OrphanedDocs))},
		{name: "Duplicate titles", items: make([]triageItem, 0, len(result.DuplicateTitles))},
		{name: "Undocumented operational targets", items: make([]triageItem, 0, len(result.UndocumentedTargets))},
		{name: "Temporary docs", items: make([]triageItem, 0, len(temporary))},
	}

	for _, file := range result.CodeWithoutDocRefs {
		path := strings.TrimSpace(file.Path)
		groups[0].items = append(groups[0].items, triageItem{
			sortKey: strings.ToLower(path),
			text:    fmt.Sprintf("%s (%d exported)", path, file.ExportedSymbols),
		})
	}
	for _, ref := range result.BrokenCodeRefs {
		path := strings.TrimSpace(ref.DocPath)
		target := strings.TrimSpace(ref.Target)
		groups[1].items = append(groups[1].items, triageItem{
			sortKey: strings.ToLower(path + "|" + strconv.Itoa(ref.Line) + "|" + target),
			text:    fmt.Sprintf("%s:%d -> %s", path, ref.Line, target),
		})
	}
	for _, doc := range result.OrphanedDocs {
		path := strings.TrimSpace(doc)
		groups[2].items = append(groups[2].items, triageItem{
			sortKey: strings.ToLower(path),
			text:    path,
		})
	}
	for _, title := range result.DuplicateTitles {
		name := strings.TrimSpace(title.Title)
		groups[3].items = append(groups[3].items, triageItem{
			sortKey: strings.ToLower(name),
			text:    fmt.Sprintf("%q", name),
		})
	}
	for _, target := range result.UndocumentedTargets {
		value := strings.TrimSpace(target)
		groups[4].items = append(groups[4].items, triageItem{
			sortKey: strings.ToLower(value),
			text:    value,
		})
	}
	for _, path := range temporary {
		value := strings.TrimSpace(path)
		groups[5].items = append(groups[5].items, triageItem{
			sortKey: strings.ToLower(value),
			text:    value,
		})
	}

	out := make([]manualGroup, 0, len(groups))
	for _, group := range groups {
		if len(group.items) == 0 {
			continue
		}
		sort.Slice(group.items, func(i, j int) bool {
			return group.items[i].sortKey < group.items[j].sortKey
		})
		out = append(out, group)
	}
	return out
}

func countManualGroupItems(groups []manualGroup) int {
	total := 0
	for _, group := range groups {
		total += len(group.items)
	}
	return total
}

func renderDocsHealthReport(result docsHealthResponse, fallbackScenario string) string {
	scenario := strings.TrimSpace(result.ScenarioName)
	if scenario == "" {
		scenario = strings.TrimSpace(fallbackScenario)
	}
	if scenario == "" {
		scenario = "unknown"
	}

	requiredDocs := 1
	requiredPresent := requiredDocs
	missingRequired := 0
	for _, missing := range result.MissingDocs {
		if strings.EqualFold(strings.TrimSpace(missing), "readme") {
			missingRequired++
		}
	}
	requiredPresent -= missingRequired
	if requiredPresent < 0 {
		requiredPresent = 0
	}
	requiredCoverage := 1.0
	if requiredDocs > 0 {
		requiredCoverage = float64(requiredPresent) / float64(requiredDocs)
	}

	misplacedPenalty := 0.05 * float64(len(result.MisplacedDocs))
	temporaryPenalty := 0.01 * float64(len(result.TemporaryDocs))
	healthPct := int(result.HealthScore*100 + 0.5)
	if healthPct < 0 {
		healthPct = 0
	}
	if healthPct > 100 {
		healthPct = 100
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Documentation Health: %s\n", scenario)
	fmt.Fprintf(&b, "Score: %d%% (%d docs)\n", healthPct, result.TotalDocs)
	fmt.Fprintf(&b, "Issues: %d misplaced, %d missing, %d extra, %d temporary\n\n", len(result.MisplacedDocs), len(result.MissingDocs), len(result.ExtraDocs), len(result.TemporaryDocs))

	b.WriteString("Score breakdown\n")
	fmt.Fprintf(&b, "- Required docs baseline: %.0f%% (%d/%d present)\n", requiredCoverage*100, requiredPresent, requiredDocs)
	fmt.Fprintf(&b, "- Misplaced penalty: -%.0f%% (%d x 5%%)\n", misplacedPenalty*100, len(result.MisplacedDocs))
	fmt.Fprintf(&b, "- Temporary-docs penalty: -%.0f%% (%d x 1%%)\n", temporaryPenalty*100, len(result.TemporaryDocs))
	fmt.Fprintf(&b, "- Extra docs are informational only (%d)\n", len(result.ExtraDocs))
	b.WriteString("- Final score is clamped to 0-100%\n\n")

	b.WriteString("Fixability\n")
	fmt.Fprintf(&b, "- Fix category: %s\n", strings.TrimSpace(result.FixCategory))
	fmt.Fprintf(&b, "- Quick-fixable files: %d\n", len(result.MisplacedDocs))
	if result.CanAutoFix {
		b.WriteString("- Auto-fix available: yes\n")
	} else {
		b.WriteString("- Auto-fix available: no\n")
	}

	return b.String()
}

func splitCSV(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	return out
}

func splitLeadingPositional(args []string) (string, []string) {
	if len(args) == 0 {
		return "", args
	}
	first := strings.TrimSpace(args[0])
	if first == "" || strings.HasPrefix(first, "-") {
		return "", args
	}
	return first, args[1:]
}

func stripBoolFlag(args []string, flagName string) ([]string, bool) {
	if strings.TrimSpace(flagName) == "" || len(args) == 0 {
		return args, false
	}
	filtered := make([]string, 0, len(args))
	found := false
	for _, arg := range args {
		if strings.TrimSpace(arg) == flagName {
			found = true
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered, found
}

func parseMetadata(raw string) (map[string]interface{}, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("invalid metadata JSON: %w", err)
	}
	if out == nil {
		return nil, fmt.Errorf("metadata must be a JSON object")
	}
	return out, nil
}

func readContent(explicit string, args []string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return explicit, nil
	}
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}
	if stdinHasData() {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}
	return "", nil
}

func stdinHasData() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice == 0
}

func parseDuration(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("interval is required")
	}
	if d, err := time.ParseDuration(raw); err == nil {
		return d, nil
	}
	if seconds, err := time.ParseDuration(raw + "s"); err == nil {
		return seconds, nil
	}
	return 0, fmt.Errorf("invalid interval %q", raw)
}
