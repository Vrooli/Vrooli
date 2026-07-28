package retrieval

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	retrievalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/retrieval"
	retrievalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/retrieval/retrieval_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type handlers struct {
	client retrievalconnect.RetrievalServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	client, base := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: retrievalconnect.NewRetrievalServiceClient(client, base)}
}

func (h *handlers) searchCall(ctx cliapp.OperationContext) (*retrievalv1.SearchResponse, error) {
	filter := &retrievalv1.SearchFilter{Text: ctx.Flag("text"), CategoryId: ctx.Flag("category-id"), Disposition: ctx.Flag("disposition"), SourceKind: ctx.Flag("source-kind"), PageSize: uint32(parseLimit(ctx.Flag("page-size"))), Tags: parseTags(ctx.Flag("tags")), PageAfter: ctx.Flag("page-after")}
	var err error
	if filter.CapturedAfter, err = parseTime(ctx.Flag("captured-after")); err != nil {
		return nil, fmt.Errorf("captured-after: %w", err)
	}
	if filter.CapturedBefore, err = parseTime(ctx.Flag("captured-before")); err != nil {
		return nil, fmt.Errorf("captured-before: %w", err)
	}
	req := &retrievalv1.SearchRequest{Filter: filter}
	response, err := h.client.Search(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("search signals", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) ambientCall(ctx cliapp.OperationContext) (*retrievalv1.AmbientResponse, error) {
	req := &retrievalv1.AmbientRequest{CategoryId: ctx.Flag("category-id"), Budget: uint32(parseLimit(ctx.Flag("budget")))}
	response, err := h.client.Ambient(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("read ambient signals", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) searchReport(_ cliapp.OperationContext, response *retrievalv1.SearchResponse) cliapp.ListReport {
	report := resultsReport(response.Results)
	if response.NextPageAfter != "" {
		report.Summary = append(report.Summary, "Next page cursor: "+response.NextPageAfter)
	}
	return report
}

func (h *handlers) ambientReport(_ cliapp.OperationContext, response *retrievalv1.AmbientResponse) cliapp.ListReport {
	return resultsReport(response.Results)
}

func resultsReport(results []*retrievalv1.RetrievedSignal) cliapp.ListReport {
	rows := make([]string, 0, len(results))
	for _, result := range results {
		rows = append(rows, fmt.Sprintf("%s disposition=%s category=%s %s", result.Signal.GetId(), result.Disposition, result.CategoryId, result.Signal.GetExtractedContent()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d signal(s).", len(rows))}, ResultsHeading: "Signals", Results: rows}
}

func parseLimit(raw string) int { var value int; _, _ = fmt.Sscan(raw, &value); return value }
func parseTags(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if tag := strings.TrimSpace(part); tag != "" {
			result = append(result, tag)
		}
	}
	return result
}

func parseTime(raw string) (*timestamppb.Timestamp, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return timestamppb.New(parsed), nil
}
