package optimize

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	optimizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/optimization"
	optimizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/optimization/optimization_v1connect"
	"google.golang.org/protobuf/proto"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client optimizationconnect.OptimizationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: optimizationconnect.NewOptimizationServiceClient(httpClient, baseURL),
	}
}

func (h handlers) run(ctx cliapp.RunContext) error {
	resp, err := h.client.CreateOptimizationRun(context.Background(), connect.NewRequest(&optimizationv1.CreateOptimizationRunRequest{ScoringProfile: ctx.Flag("scoring-profile"), DryRun: ctx.BoolFlag("dry-run")}))
	if err != nil {
		return cliapp.WrapAPIError("create optimization run", err, nil)
	}
	return renderRun(ctx, resp.Msg, resp.Msg.GetRun())
}

func (h handlers) candidate(ctx cliapp.RunContext) error {
	resp, err := h.client.RunCandidate(context.Background(), connect.NewRequest(&optimizationv1.RunCandidateRequest{RunId: ctx.Positional("run_id"), CandidateId: ctx.Flag("candidate-id")}))
	if err != nil {
		return cliapp.WrapAPIError("run candidate", err, nil)
	}
	return renderRun(ctx, resp.Msg, resp.Msg.GetRun())
}

func (h handlers) score(ctx cliapp.RunContext) error {
	resp, err := h.client.ScoreCandidates(context.Background(), connect.NewRequest(&optimizationv1.ScoreCandidatesRequest{RunId: ctx.Positional("run_id")}))
	if err != nil {
		return cliapp.WrapAPIError("score candidates", err, nil)
	}
	return renderRun(ctx, resp.Msg, resp.Msg.GetRun())
}

func (h handlers) approve(ctx cliapp.RunContext) error {
	resp, err := h.client.ApproveCandidate(context.Background(), connect.NewRequest(&optimizationv1.ApproveCandidateRequest{RunId: ctx.Positional("run_id"), CandidateId: ctx.Flag("candidate-id"), Approved: ctx.BoolFlag("approved")}))
	if err != nil {
		return cliapp.WrapAPIError("approve candidate", err, nil)
	}
	return renderRunMutation(ctx, resp.Msg, resp.Msg.GetRun())
}

func (h handlers) rollback(ctx cliapp.RunContext) error {
	resp, err := h.client.RollbackOptimization(context.Background(), connect.NewRequest(&optimizationv1.RollbackOptimizationRequest{RunId: ctx.Positional("run_id")}))
	if err != nil {
		return cliapp.WrapAPIError("rollback optimization", err, nil)
	}
	return renderRunMutation(ctx, resp.Msg, resp.Msg.GetRun())
}

func renderRun(ctx cliapp.RunContext, payload proto.Message, run *optimizationv1.OptimizationRun) error {
	return cliapp.RenderProtoList(ctx, payload, cliapp.ListReport{Summary: []string{formatRun(run)}, ResultsHeading: "Candidates", Results: formatCandidates(run.GetCandidates()), RetrievalHints: []string{run.GetRecommendation()}})
}

func renderRunMutation(ctx cliapp.RunContext, payload proto.Message, run *optimizationv1.OptimizationRun) error {
	return cliapp.RenderProtoMutation(ctx, payload, cliapp.MutationReport{Result: []string{formatRun(run)}, Changes: formatCandidates(run.GetCandidates())})
}

func formatRun(r *optimizationv1.OptimizationRun) string {
	if r == nil {
		return "Optimization run unavailable."
	}
	return fmt.Sprintf("%s status=%s scoring=%s", r.GetId(), r.GetStatus(), r.GetScoringProfile())
}

func formatCandidates(candidates []*optimizationv1.Candidate) []string {
	out := make([]string, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, fmt.Sprintf("%s status=%s score=%.2f approval_required=%t %s", c.GetId(), c.GetStatus(), c.GetScore(), c.GetApprovalRequired(), c.GetDescription()))
	}
	return out
}
