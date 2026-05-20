package search

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/search"
	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/search/search_v1connect"
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
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
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
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d result(s) (mode=%s).", len(resp.Msg.Results), resp.Msg.ModeUsed)},
		ResultsHeading: "Results",
		Results:        []string{},
	})
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
			fmt.Sprintf("Ollama: %v  Qdrant: %v  Indexed: %d", resp.Msg.Ollama, resp.Msg.Qdrant, resp.Msg.IndexedCount),
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
