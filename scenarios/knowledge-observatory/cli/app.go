package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
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
		ExtraAPIEnvVars:     []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		ExtraAPIPortEnvVars: []string{"API_PORT"},
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
			{Name: "graph", NeedsAPI: true, Description: "Generate a knowledge graph", Run: a.cmdGraph},
		},
	}

	docs := cliapp.CommandGroup{
		Title: "Documentation",
		Commands: []cliapp.Command{
			{Name: "docs", NeedsAPI: true, Description: "Documentation explorer commands (search-files, search-text, search-deep, scenarios, tree, health, view, reset, heal, heal-status)", Run: a.cmdDocs},
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
	if err := fs.Parse(args); err != nil {
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
	if err := fs.Parse(args); err != nil {
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
	if err := fs.Parse(args); err != nil {
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

func (a *App) cmdDocs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: docs <search-files|search-text|search-deep|scenarios|tree|health|view|reset|heal|heal-status> [options]")
	}
	subcommand := strings.TrimSpace(args[0])
	switch subcommand {
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
	default:
		return fmt.Errorf("unknown docs subcommand: %s", subcommand)
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
	if err := fs.Parse(args); err != nil {
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
	if err := fs.Parse(args); err != nil {
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
	if err := fs.Parse(args); err != nil {
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
	if err := fs.Parse(args); err != nil {
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
	if err := fs.Parse(args); err != nil {
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
	fs := flag.NewFlagSet("docs health", flag.ContinueOnError)
	scenario := fs.String("scenario", "", "Scenario name")
	if err := fs.Parse(args); err != nil {
		return err
	}

	scenarioValue := strings.TrimSpace(*scenario)
	if scenarioValue == "" {
		scenarioValue = strings.TrimSpace(strings.Join(fs.Args(), " "))
	}
	if scenarioValue == "" {
		return fmt.Errorf("usage: docs health <scenario> [--scenario=name]")
	}

	body, err := a.doRequest("GET", fmt.Sprintf("/scenarios/%s/docs/health", scenarioValue), nil, nil)
	if err != nil {
		return err
	}
	return a.printJSON(body)
}

func (a *App) cmdDocsView(args []string) error {
	fs := flag.NewFlagSet("docs view", flag.ContinueOnError)
	path := fs.String("path", "", "Document path")
	format := fs.String("format", "raw", "Format: raw, highlighted, or preview")
	if err := fs.Parse(args); err != nil {
		return err
	}

	pathValue := strings.TrimSpace(*path)
	if pathValue == "" {
		pathValue = strings.TrimSpace(strings.Join(fs.Args(), " "))
	}
	if pathValue == "" {
		return fmt.Errorf("usage: docs view <path> [--format=raw|highlighted|preview]")
	}

	query := url.Values{}
	query.Set("path", pathValue)
	formatValue := strings.TrimSpace(*format)
	if formatValue != "" {
		query.Set("format", formatValue)
	}

	body, err := a.doRequest("GET", "/docs/content", query, nil)
	if err != nil {
		return err
	}
	return a.printJSON(body)
}

func (a *App) cmdDocsReset(args []string) error {
	fs := flag.NewFlagSet("docs reset", flag.ContinueOnError)
	path := fs.String("path", "", "Document path")
	maxAgeDays := fs.Int("max-age-days", 0, "Remove entries older than N days")
	keepMin := fs.Int("keep-min-entries", 0, "Always keep at least N entries")
	preview := fs.Bool("preview", false, "Preview changes without writing")
	if err := fs.Parse(args); err != nil {
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
	if err := fs.Parse(args); err != nil {
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
	if err := fs.Parse(args); err != nil {
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
	if err := fs.Parse(args); err != nil {
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
	if err := fs.Parse(args); err != nil {
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

func (a *App) cmdHealth(args []string) error {
	fs := flag.NewFlagSet("health", flag.ContinueOnError)
	watch := fs.Bool("watch", false, "Poll health metrics continuously")
	intervalRaw := fs.String("interval", "5s", "Polling interval when --watch is set")
	if err := fs.Parse(args); err != nil {
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
