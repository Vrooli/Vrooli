package search

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/business-health/v1/search"
	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/business-health/v1/search/search_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client searchconnect.SearchServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: searchconnect.NewSearchServiceClient(httpClient, baseURL),
	}
}

// query calls the generated Connect SearchService.Search method. Phase 1
// surfaces the server's Unimplemented response; Phase 3 wires real
// rendering.
func (h *handlers) query(ctx cliapp.RunContext) error {
	text := ctx.Positional("text")
	limit := int32(10)
	if v := strings.TrimSpace(ctx.Flag("limit")); v != "" {
		// ParseInt with an explicit 32-bit size keeps the conversion
		// overflow-safe (gosec G109).
		if parsed, err := strconv.ParseInt(v, 10, 32); err == nil && parsed > 0 {
			limit = int32(parsed)
		}
	}
	mode := parseMode(ctx.Flag("mode"))

	resp, err := h.client.Search(context.Background(), connect.NewRequest(&searchv1.SearchRequest{
		Query: text,
		Limit: limit,
		Mode:  mode,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("search %q", text), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no search response")
	}
	results := make([]string, 0, len(resp.Msg.Results))
	for i, r := range resp.Msg.Results {
		desc := truncate(r.Snippet, 80)
		// The server computes the regime-aware weak flag once; the CLI just
		// renders it (no client-side threshold).
		weak := ""
		if r.Weak {
			weak = " (weak)"
		}
		results = append(results, fmt.Sprintf("%d. [%s] %s — %s [score=%.3f %s]%s",
			i+1, r.Type, r.Title, desc, r.Score, r.Anchor, weak))
	}
	if len(results) == 0 {
		results = append(results, "(no matches)")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d result(s) (mode=%s).", len(resp.Msg.Results), resp.Msg.Mode.String())},
		ResultsHeading: "Results",
		Results:        results,
	})
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// status calls the generated Connect SearchService.Status method.
func (h *handlers) status(ctx cliapp.RunContext) error {
	resp, err := h.client.Status(context.Background(), connect.NewRequest(&searchv1.StatusRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("search status", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no status response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Available: %v", resp.Msg.Available),
			fmt.Sprintf("Ollama: %v  Qdrant: %v  Reranker: %v  Indexed: %d", resp.Msg.OllamaUp, resp.Msg.QdrantUp, resp.Msg.RerankerUp, resp.Msg.Indexed),
			fmt.Sprintf("Collection: %s", resp.Msg.Collection),
		},
		ResultsHeading: "Backend status",
		Results:        []string{},
	})
}

// parseMode maps the CLI --mode flag to the proto enum. Empty / unknown
// values fall back to UNSPECIFIED (let the server choose).
func parseMode(s string) searchv1.Mode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "ai":
		return searchv1.Mode_MODE_AI
	case "text":
		return searchv1.Mode_MODE_TEXT
	default:
		return searchv1.Mode_MODE_UNSPECIFIED
	}
}
