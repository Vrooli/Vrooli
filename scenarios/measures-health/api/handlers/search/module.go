package search

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"measures-health/internal/measureindex"
	"measures-health/internal/module"

	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/measures-health/v1/search/search_v1connect"
)

// thresholdEnv is the auto-execute confidence gate lever (theta). Unset/invalid
// falls back to measures.DefaultConfidenceThreshold (resolved inside the
// Provider), so a deployment can tune it without a rebuild.
const thresholdEnv = "MEASURES_HEALTH_CONFIDENCE_THRESHOLD"

// Module returns the index domain's contribution to the API: the generated
// Connect-RPC SearchService handler over a Provider built from the central
// measures corpus (every scenario's manifest `measure` blocks harvested at boot
// against the committed proto descriptor). Harvesting is best-effort: a harvest
// error leaves an empty index (the provider answers honest-empty and reports
// available=false) rather than failing the scenario's boot.
func Module(repoRoot string, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	decls, err := measureindex.NewFilesystemHarvester(repoRoot).Harvest(context.Background())
	if err != nil {
		logger.Printf("[measures-health] measure index harvest degraded (serving empty index): %v", err)
		decls = nil
	}
	provider := measureindex.NewProvider(decls, measureindex.Config{Threshold: thresholdFromEnv()})
	logger.Printf("[measures-health] measure index ready: %d measure(s) across the fleet", provider.Len())

	connectPath, connectHandler := searchconnect.NewSearchServiceHandler(NewConnectHandler(Deps{
		Searcher: provider,
		Logger:   logger,
	}))
	return module.Module{
		Name: "search",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — the measures index is computed on demand from the
// filesystem corpus; it owns no tables.
func Schema() string { return "" }

func thresholdFromEnv() float64 {
	raw := strings.TrimSpace(os.Getenv(thresholdEnv))
	if raw == "" {
		return 0 // Provider applies measures.DefaultConfidenceThreshold.
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 || v > 1 {
		return 0
	}
	return v
}

// Endpoints is the machine-readable description of the search module's public
// surface. Connect-RPC method paths reference the generated *Procedure constants
// so renaming an RPC in search.proto breaks this at compile time; the global
// parity test asserts every rpc has exactly one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "search_search",
		Path:        searchconnect.SearchServiceSearchProcedure,
		Method:      "POST",
		Summary:     "Match an analytical question to a measure and answer it",
		Description: "The federated measures provider (search-hub 'measure' type): matches a natural-language analytical question to the best-fitting measure across the fleet, resolves its parameters (deterministic time_window first, constrained extraction for the rest), and — for a safe read-only measure resolved at high confidence — proxies execution to the owning scenario and returns the answer with provenance. A write/destructive measure is never auto-executed.",
		Category:    "index",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"query": "string (required, natural-language analytical question)",
				"limit": "int (max measure hits; a measure provider returns the best answer)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"results": "array<MeasureResult{score, measure}>",
				"matcher": "string (lexical | ai | none)",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 501, Code: "unimplemented", Description: "Measures index not configured"},
		},
		Examples: []module.Example{
			{Name: "Answer an analytical question", Curl: "curl http://localhost:${API_PORT}/vrooli.measures_health.v1.search.SearchService/Search -H 'Content-Type: application/json' -d '{\"query\":\"how many backlog items did we complete this week\",\"limit\":1}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "measures-health search query",
			Args:    []string{"<question>", "--limit"},
		},
	},
	{
		ID:          "search_status",
		Path:        searchconnect.SearchServiceStatusProcedure,
		Method:      "POST",
		Summary:     "Report measure-index + backend availability",
		Description: "Reports the central measure index size, whether the ollama constrained-extraction backend is reachable, and the active matcher leg — so callers (and the search-hub status_endpoint) can degrade gracefully.",
		Category:    "index",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"available":     "bool",
				"ollama":        "bool",
				"qdrant":        "bool",
				"indexed_count": "int",
				"matcher":       "string",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 501, Code: "unimplemented", Description: "Measures index not configured"},
		},
		Examples: []module.Example{
			{Name: "Index status", Curl: "curl http://localhost:${API_PORT}/vrooli.measures_health.v1.search.SearchService/Status -H 'Content-Type: application/json' -d '{}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "measures-health search status",
		},
	},
}
