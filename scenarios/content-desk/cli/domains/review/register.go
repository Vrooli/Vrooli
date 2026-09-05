package review

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/review"
	reviewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/review/review_v1connect"
)

const GroupName = "review"

type handlers struct {
	client reviewconnect.ReviewServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: reviewconnect.NewReviewServiceClient(httpClient, baseURL)}
}

func (h *handlers) listCall(_ cliapp.OperationContext) (*reviewv1.ListReviewRunsResponse, error) {
	response, err := h.client.ListReviewRuns(context.Background(), connect.NewRequest(&reviewv1.ListReviewRunsRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list review runs", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no review runs response")
	}
	return response.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, message *reviewv1.ListReviewRunsResponse) cliapp.ListReport {
	results := make([]string, 0, len(message.ReviewRuns))
	for _, run := range message.ReviewRuns {
		results = append(results, fmt.Sprintf("%s — draft=%s outcome=%s", run.Id, run.DraftId, run.Outcome))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d review run(s).", len(message.ReviewRuns))}, ResultsHeading: "Review runs", Results: results}
}

func (h *handlers) recordCall(ctx cliapp.OperationContext) (*reviewv1.RecordReviewRunResponse, error) {
	response, err := h.client.RecordReviewRun(context.Background(), connect.NewRequest(&reviewv1.RecordReviewRunRequest{DraftId: ctx.Flag("draft"), Verdicts: []*reviewv1.Verdict{{Mode: ctx.Flag("mode"), Passed: ctx.BoolFlag("passed"), Evidence: ctx.Flag("evidence"), Finding: ctx.Flag("finding")}}}))
	if err != nil {
		return nil, cliapp.WrapAPIError("record review run", err, nil)
	}
	if response == nil || response.Msg == nil || response.Msg.ReviewRun == nil {
		return nil, fmt.Errorf("server returned no review run")
	}
	return response.Msg, nil
}
func (h *handlers) recordReport(_ cliapp.OperationContext, message *reviewv1.RecordReviewRunResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Recorded review %s: %s.", message.ReviewRun.Id, message.ReviewRun.Outcome)}}
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{"ReviewService.ListReviewRuns": cliapp.ProtoList(h.listCall, h.listReport), "ReviewService.RecordReviewRun": cliapp.ProtoMutation(h.recordCall, h.recordReport)})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("review: load from manifest: %w", err)
	}
	return group, nil
}
