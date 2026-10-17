package knowledge

// docsearch_cmd.go is Phase 6 of the KO documentation search cutover: the CLI
// verbs for the hybrid documentation engine, mirroring cli-health's semantics —
// `search query` / `search status` and `reindex run` / `reindex status` /
// `reindex cancel`. They call the converged REST surface (/api/v1/search/*,
// /api/v1/reindex/*) the api package exposes.

import (
	"flag"
	"fmt"
	"strings"

	"knowledge-observatory/cli/internal/support"

	"github.com/vrooli/cli-core/cliutil"
)

// docSearchQueryRequest mirrors the api package's request shape.
type docSearchQueryRequest struct {
	Query  string          `json:"query"`
	Mode   string          `json:"mode,omitempty"`
	Scope  string          `json:"scope,omitempty"`
	Target string          `json:"target,omitempty"`
	Limit  int             `json:"limit,omitempty"`
	Facets docSearchFacets `json:"facets,omitempty"`
}

type docSearchFacets struct {
	DocType      []string `json:"doc_type,omitempty"`
	Audience     []string `json:"audience,omitempty"`
	Maturity     []string `json:"maturity,omitempty"`
	CanonicalFor []string `json:"canonical_for,omitempty"`
}

// searchDispatch routes `search query|status`. A bare `search <text>` is treated
// as `search query <text>` for convenience.
func searchDispatch(deps support.Dependencies, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: search <query|status> [...]")
	}
	switch args[0] {
	case "query":
		return searchQuery(deps, args[1:])
	case "status":
		return searchStatus(deps, args[1:])
	default:
		// Backward-compatible shorthand: `search <text>` == `search query <text>`.
		return searchQuery(deps, args)
	}
}

func searchQuery(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("search query", flag.ContinueOnError)
	mode := fs.String("mode", "", "Retrieval mode: auto|hybrid|dense|text")
	scope := fs.String("scope", "", "Scope: global|scenario|path")
	target := fs.String("target", "", "Scenario name (scope=scenario) or path prefix (scope=path)")
	limit := fs.Int("limit", 0, "Maximum number of results")
	docType := fs.String("doc-type", "", "Comma-separated docType facet filter")
	audience := fs.String("audience", "", "Comma-separated audience facet filter")
	maturity := fs.String("maturity", "", "Comma-separated maturity facet filter")
	canonicalFor := fs.String("canonical-for", "", "Comma-separated canonicalFor facet filter")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if query == "" {
		return fmt.Errorf("usage: search query <text> [--scope=global|scenario|path] [--target=...] [--mode=auto|hybrid|dense|text] [--limit=N] [--doc-type=...] [--audience=...] [--maturity=...] [--canonical-for=...]")
	}

	req := docSearchQueryRequest{
		Query:  query,
		Mode:   strings.TrimSpace(*mode),
		Scope:  strings.TrimSpace(*scope),
		Target: strings.TrimSpace(*target),
		Facets: docSearchFacets{
			DocType:      support.SplitCSV(*docType),
			Audience:     support.SplitCSV(*audience),
			Maturity:     support.SplitCSV(*maturity),
			CanonicalFor: support.SplitCSV(*canonicalFor),
		},
	}
	if *limit > 0 {
		req.Limit = *limit
	}

	body, err := deps.ScenarioApp().Request("POST", "/search/query", nil, req)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func searchStatus(deps support.Dependencies, _ []string) error {
	body, err := deps.ScenarioApp().Request("GET", "/search/status", nil, nil)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

// reindexDispatch routes `reindex run|status|cancel`.
func reindexDispatch(deps support.Dependencies, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: reindex <run|status|cancel> [...]")
	}
	switch args[0] {
	case "run":
		return reindexRun(deps, args[1:])
	case "status":
		return reindexStatus(deps, args[1:])
	case "cancel":
		return reindexCancel(deps, args[1:])
	default:
		return fmt.Errorf("usage: reindex <run|status|cancel> [...]")
	}
}

func reindexRun(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("reindex run", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "Plan only; report planned upserts/deletes without writing")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	req := map[string]bool{"dry_run": *dryRun}
	body, err := deps.ScenarioApp().Request("POST", "/reindex/run", nil, req)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func reindexStatus(deps support.Dependencies, _ []string) error {
	body, err := deps.ScenarioApp().Request("GET", "/reindex/status", nil, nil)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func reindexCancel(deps support.Dependencies, _ []string) error {
	body, err := deps.ScenarioApp().Request("POST", "/reindex/cancel", nil, nil)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}
