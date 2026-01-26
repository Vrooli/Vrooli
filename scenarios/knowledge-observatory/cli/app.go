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

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, knowledge, config}
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
