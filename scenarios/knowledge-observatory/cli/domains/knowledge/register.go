package knowledge

import (
	"flag"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"knowledge-observatory/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

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

type CollectionMaintenanceRequest struct {
	DryRun     bool `json:"dry_run"`
	MaxDeletes int  `json:"max_deletes,omitempty"`
}

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Knowledge",
		Commands: []cliapp.Command{
			{Name: "search", NeedsAPI: true, Description: "Documentation hybrid search: search query <text> | search status", Run: func(args []string) error { return searchDispatch(deps, args) }},
			{Name: "reindex", NeedsAPI: true, Description: "Documentation index control: reindex run [--dry-run] | reindex status | reindex cancel", Run: func(args []string) error { return reindexDispatch(deps, args) }},
			{Name: "collection-diagnostics", NeedsAPI: false, Description: "Inspect collection embedding/chunk diagnostics", Run: func(args []string) error { return collectionDiagnostics(deps, args) }},
			{Name: "collection-prune-stale", NeedsAPI: false, Description: "Prune stale chunk versions (dry-run by default)", Run: func(args []string) error { return collectionPruneStale(deps, args) }},
			{Name: "collection-dedupe", NeedsAPI: false, Description: "Delete duplicate content chunks (dry-run by default)", Run: func(args []string) error { return collectionDedupe(deps, args) }},
			{Name: "graph", NeedsAPI: true, Description: "Generate a knowledge graph", Run: func(args []string) error { return graph(deps, args) }},
		},
	}
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
