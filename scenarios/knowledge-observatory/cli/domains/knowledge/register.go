package knowledge

import (
	"flag"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"knowledge-observatory/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type SearchRequest struct {
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

type GraphRequest struct {
	Center     string   `json:"center_concept"`
	Collection string   `json:"collection,omitempty"`
	Namespaces []string `json:"namespaces,omitempty"`
	Visibility []string `json:"visibility,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Depth      int      `json:"depth,omitempty"`
	Limit      int      `json:"limit,omitempty"`
	Threshold  float64  `json:"threshold,omitempty"`
}

type IngestRequest struct {
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

type IngestJobRequest struct {
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

type CollectionMaintenanceRequest struct {
	DryRun     bool `json:"dry_run"`
	MaxDeletes int  `json:"max_deletes,omitempty"`
}

type DocumentDeleteRequest struct {
	Namespace  string `json:"namespace"`
	Collection string `json:"collection,omitempty"`
	DocumentID string `json:"document_id,omitempty"`
	ExternalID string `json:"external_id,omitempty"`
	DryRun     bool   `json:"dry_run,omitempty"`
}

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Knowledge",
		Commands: []cliapp.Command{
			{Name: "search", NeedsAPI: true, Description: "Semantic search over knowledge", Run: func(args []string) error { return search(deps, args) }},
			{Name: "ingest", NeedsAPI: true, Description: "Ingest a single knowledge record", Run: func(args []string) error { return ingest(deps, args) }},
			{Name: "ingest-job", NeedsAPI: true, Description: "Enqueue an async document ingest job", Run: func(args []string) error { return ingestJob(deps, args) }},
			{Name: "job-status", NeedsAPI: true, Description: "Check async ingest job status", Run: func(args []string) error { return jobStatus(deps, args) }},
			{Name: "ingest-health", NeedsAPI: true, Description: "Inspect ingest queue and runner health", Run: func(args []string) error { return ingestHealth(deps, args) }},
			{Name: "collection-diagnostics", NeedsAPI: true, Description: "Inspect collection embedding/chunk diagnostics", Run: func(args []string) error { return collectionDiagnostics(deps, args) }},
			{Name: "collection-prune-stale", NeedsAPI: true, Description: "Prune stale chunk versions (dry-run by default)", Run: func(args []string) error { return collectionPruneStale(deps, args) }},
			{Name: "collection-dedupe", NeedsAPI: true, Description: "Delete duplicate content chunks (dry-run by default)", Run: func(args []string) error { return collectionDedupe(deps, args) }},
			{Name: "document-delete", NeedsAPI: true, Description: "Delete all chunks for a document (dry-run by default)", Run: func(args []string) error { return documentDelete(deps, args) }},
			{Name: "graph", NeedsAPI: true, Description: "Generate a knowledge graph", Run: func(args []string) error { return graph(deps, args) }},
		},
	}
}

func search(deps support.Dependencies, args []string) error {
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

	req := SearchRequest{
		Query:          query,
		Collection:     strings.TrimSpace(*collection),
		Namespaces:     support.SplitCSV(*namespaces),
		Visibility:     support.SplitCSV(*visibility),
		Tags:           support.SplitCSV(*tags),
		IngestedAfter:  strings.TrimSpace(*ingestedAfter),
		IngestedBefore: strings.TrimSpace(*ingestedBefore),
	}
	if *limit > 0 {
		req.Limit = *limit
	}
	if *threshold > 0 {
		req.Threshold = *threshold
	}

	body, err := deps.ScenarioApp().Request("POST", "/knowledge/search", nil, req)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func graph(deps support.Dependencies, args []string) error {
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

	req := GraphRequest{
		Center:     centerValue,
		Collection: strings.TrimSpace(*collection),
		Namespaces: support.SplitCSV(*namespaces),
		Visibility: support.SplitCSV(*visibility),
		Tags:       support.SplitCSV(*tags),
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

	body, err := deps.ScenarioApp().Request("POST", "/knowledge/graph", nil, req)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func ingest(deps support.Dependencies, args []string) error {
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
	contentValue, err := support.ReadContent(*content, fs.Args())
	if err != nil {
		return err
	}
	if contentValue == "" {
		return fmt.Errorf("content is required (use --content, positional args, or stdin)")
	}
	metadata, err := support.ParseMetadata(*metadataRaw)
	if err != nil {
		return err
	}

	req := IngestRequest{
		Namespace:  strings.TrimSpace(*namespace),
		Collection: strings.TrimSpace(*collection),
		RecordID:   strings.TrimSpace(*recordID),
		ExternalID: strings.TrimSpace(*externalID),
		Content:    contentValue,
		Tags:       support.SplitCSV(*tags),
		Metadata:   metadata,
		Visibility: strings.TrimSpace(*visibility),
		Source:     strings.TrimSpace(*source),
		SourceType: strings.TrimSpace(*sourceType),
	}
	body, err := deps.ScenarioApp().Request("POST", "/knowledge/records/upsert", nil, req)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func ingestJob(deps support.Dependencies, args []string) error {
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
	contentValue, err := support.ReadContent(*content, fs.Args())
	if err != nil {
		return err
	}
	if contentValue == "" {
		return fmt.Errorf("content is required (use --content, positional args, or stdin)")
	}
	metadata, err := support.ParseMetadata(*metadataRaw)
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

	req := IngestJobRequest{
		Namespace:    strings.TrimSpace(*namespace),
		Collection:   strings.TrimSpace(*collection),
		DocumentID:   strings.TrimSpace(*documentID),
		ExternalID:   strings.TrimSpace(*externalID),
		Content:      contentValue,
		Tags:         support.SplitCSV(*tags),
		Metadata:     metadata,
		Visibility:   strings.TrimSpace(*visibility),
		Source:       strings.TrimSpace(*source),
		SourceType:   strings.TrimSpace(*sourceType),
		ChunkSize:    chunkSizeValue,
		ChunkOverlap: chunkOverlapValue,
	}
	body, err := deps.ScenarioApp().Request("POST", "/ingest/jobs", nil, req)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func jobStatus(deps support.Dependencies, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: job-status <job_id>")
	}
	jobID := strings.TrimSpace(args[0])
	if jobID == "" {
		return fmt.Errorf("job_id is required")
	}
	body, err := deps.ScenarioApp().Request("GET", fmt.Sprintf("/ingest/jobs/%s", jobID), nil, nil)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func ingestHealth(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("ingest-health", flag.ContinueOnError)
	watch := fs.Bool("watch", false, "Poll ingest health continuously")
	intervalRaw := fs.String("interval", "5s", "Polling interval when --watch is set")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if !*watch {
		body, err := deps.ScenarioApp().Request("GET", "/ingest/health", nil, nil)
		if err != nil {
			return err
		}
		cliutil.PrintJSON(body)
		return nil
	}

	interval, err := support.ParseDuration(*intervalRaw)
	if err != nil {
		return err
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		body, err := deps.ScenarioApp().Request("GET", "/ingest/health", nil, nil)
		if err != nil {
			return err
		}
		fmt.Printf("== %s ==\n", time.Now().UTC().Format(time.RFC3339))
		cliutil.PrintJSON(body)
		<-ticker.C
	}
}

func collectionDiagnostics(deps support.Dependencies, args []string) error {
	leadingCollection, parseArgs := support.SplitLeadingPositional(args)
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
	if modeValue := strings.TrimSpace(*mode); modeValue != "" {
		query.Set("mode", modeValue)
	}
	if *limit > 0 {
		query.Set("limit", strconv.Itoa(*limit))
	}

	path := fmt.Sprintf("/knowledge/collections/%s/diagnostics", url.PathEscape(collectionValue))
	body, err := deps.ScenarioApp().Request("GET", path, query, nil)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func collectionPruneStale(deps support.Dependencies, args []string) error {
	leadingCollection, parseArgs := support.SplitLeadingPositional(args)
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

	req := CollectionMaintenanceRequest{DryRun: *dryRun, MaxDeletes: *maxDeletes}
	if *apply {
		req.DryRun = false
	}

	path := fmt.Sprintf("/knowledge/collections/%s/maintenance/prune-stale-chunks", url.PathEscape(collectionValue))
	body, err := deps.ScenarioApp().Request("POST", path, nil, req)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func collectionDedupe(deps support.Dependencies, args []string) error {
	leadingCollection, parseArgs := support.SplitLeadingPositional(args)
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

	req := CollectionMaintenanceRequest{DryRun: *dryRun, MaxDeletes: *maxDeletes}
	if *apply {
		req.DryRun = false
	}

	path := fmt.Sprintf("/knowledge/collections/%s/maintenance/dedupe-content", url.PathEscape(collectionValue))
	body, err := deps.ScenarioApp().Request("POST", path, nil, req)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func documentDelete(deps support.Dependencies, args []string) error {
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

	req := DocumentDeleteRequest{
		Namespace:  namespaceValue,
		Collection: strings.TrimSpace(*collection),
		DocumentID: documentIDValue,
		ExternalID: externalIDValue,
		DryRun:     *dryRun,
	}
	if *apply {
		req.DryRun = false
	}
	body, err := deps.ScenarioApp().Request("POST", "/knowledge/documents/delete", nil, req)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}
