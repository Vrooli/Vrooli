package search

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/search"
	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/search/search_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/shared"
)

type handlers struct {
	client searchconnect.SearchServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: searchconnect.NewSearchServiceClient(httpClient, baseURL)}
}

func (h *handlers) suggest(ctx cliapp.RunContext) error {
	limit, err := parseLimit(ctx.Flag("limit"))
	if err != nil {
		return err
	}
	resp, err := h.client.Suggest(context.Background(), connect.NewRequest(&searchv1.SuggestRequest{
		Query: ctx.Positional("query"),
		Types: cliutil.ParseCSV(ctx.Flag("types")),
		Limit: limit,
		Group: ctx.Flag("group"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("search suggestions", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no search suggestions")
	}
	results := make([]string, 0, len(resp.Msg.GetHits()))
	for _, hit := range resp.Msg.GetHits() {
		results = append(results, formatHit(hit))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Found %d suggestion(s).", len(resp.Msg.GetHits())),
			fmt.Sprintf("degraded=%t latency_ms=%d reason=%s", resp.Msg.GetDegraded(), resp.Msg.GetLatencyMs(), resp.Msg.GetReason()),
		},
		ResultsHeading: "Suggestions",
		Results:        results,
	})
}

func parseLimit(value string) (int32, error) {
	if value == "" {
		return 8, nil
	}
	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("--limit must be an integer: %w", err)
	}
	return int32(n), nil
}

func formatHit(hit *sharedv1.SearchHit) string {
	if hit == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s/%s score=%.3f rerank=%.3f path=%s title=%s",
		hit.GetProviderId(), hit.GetType(), hit.GetScore(), hit.GetRerankScore(), hit.GetPath(), hit.GetTitle())
}
