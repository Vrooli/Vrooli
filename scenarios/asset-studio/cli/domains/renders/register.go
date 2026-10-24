package renders

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	studiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/asset-studio/v1/studio"
	studioconnect "github.com/vrooli/vrooli/packages/proto/gen/go/asset-studio/v1/studio/studio_v1connect"
)

// Register exposes the durable render receipt without exposing producer bytes.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := studioconnect.NewStudioServiceClient(httpClient, baseURL)
	return cliapp.SubcommandGroup{Name: "renders", Description: "Inspect durable media-production receipts", NeedsAPI: true, Subcommands: []cliapp.Command{
		(cliapp.Command{Name: "show", Description: "Show a render status, producer receipt, and candidate metadata", Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoList}, Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "render-id", Required: true, Description: "Render identifier"}}}}).WithPrimitive(cliapp.ProtoList(
			func(ctx cliapp.OperationContext) (*studiov1.GetRenderResponse, error) {
				resp, err := client.GetRender(context.Background(), connect.NewRequest(&studiov1.GetRenderRequest{RenderId: ctx.Positional("render-id")}))
				if err != nil {
					return nil, cliapp.WrapAPIError("show render", err, nil)
				}
				return resp.Msg, nil
			},
			func(_ cliapp.OperationContext, msg *studiov1.GetRenderResponse) cliapp.ListReport {
				r := msg.GetRender()
				rows := []string{fmt.Sprintf("status=%s actual_cost_recorded=%t actual_cost=%.4f", r.GetStatus(), r.GetActualCostRecorded(), r.GetActualCost())}
				if p := r.GetProvenance(); p != nil {
					rows = append(rows, fmt.Sprintf("backend=%s model=%s seed=%s", p.GetBackend(), p.GetModel(), p.GetSeed()))
				}
				for _, candidate := range r.GetCandidates() {
					rows = append(rows, fmt.Sprintf("candidate=%s status=%s media_type=%s", candidate.GetId(), candidate.GetStatus(), candidate.GetMediaType()))
				}
				if r.GetFailureCode() != "" {
					rows = append(rows, "failure_code="+r.GetFailureCode())
				}
				return cliapp.ListReport{Summary: []string{fmt.Sprintf("Render %s.", r.GetId())}, ResultsHeading: "Render receipt", Results: rows}
			},
		)),
		(cliapp.Command{Name: "set-campaign-budget", Description: "Set a campaign media spend limit", Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoMutation}, Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "campaign-ref", Required: true, Description: "Campaign reference"}, {Name: "limit-usd", Required: true, Description: "Maximum media spend in USD"}}}}).WithPrimitive(cliapp.ProtoMutation(
			func(ctx cliapp.OperationContext) (*studiov1.SetCampaignBudgetResponse, error) {
				limit, err := strconv.ParseFloat(ctx.Flag("limit-usd"), 64)
				if err != nil {
					return nil, fmt.Errorf("--limit-usd must be a number: %w", err)
				}
				resp, err := client.SetCampaignBudget(context.Background(), connect.NewRequest(&studiov1.SetCampaignBudgetRequest{CampaignRef: ctx.Flag("campaign-ref"), LimitUsd: limit}))
				if err != nil {
					return nil, cliapp.WrapAPIError("set campaign budget", err, nil)
				}
				return resp.Msg, nil
			},
			func(_ cliapp.OperationContext, msg *studiov1.SetCampaignBudgetResponse) cliapp.MutationReport {
				budget := msg.GetBudget()
				return cliapp.MutationReport{Result: []string{fmt.Sprintf("Set campaign budget for %s.", budget.GetCampaignRef())}, Changes: []string{fmt.Sprintf("limit_usd=%.4f spent_usd=%.4f", budget.GetLimitUsd(), budget.GetSpentUsd())}}
			},
		)),
		(cliapp.Command{Name: "regenerate", Description: "Create a fresh render from a successful render's recorded intent", Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoMutation}, Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "source-render-id", Required: true, Description: "Successful source render identifier"}}, Flags: []cliapp.Flag{{Name: "confirm-over-budget", Bool: true, Description: "Record confirmation if the projected campaign spend exceeds its limit"}, {Name: "confirmation-actor", Description: "Operator recording an over-budget confirmation"}}}}).WithPrimitive(cliapp.ProtoMutation(
			func(ctx cliapp.OperationContext) (*studiov1.RegenerateRenderResponse, error) {
				resp, err := client.RegenerateRender(context.Background(), connect.NewRequest(&studiov1.RegenerateRenderRequest{SourceRenderId: ctx.Positional("source-render-id"), ConfirmOverBudget: ctx.BoolFlag("confirm-over-budget"), BudgetConfirmationActorId: ctx.Flag("confirmation-actor")}))
				if err != nil {
					return nil, cliapp.WrapAPIError("regenerate render", err, nil)
				}
				return resp.Msg, nil
			},
			func(_ cliapp.OperationContext, msg *studiov1.RegenerateRenderResponse) cliapp.MutationReport {
				return cliapp.MutationReport{Result: []string{fmt.Sprintf("Queued regenerated render %s from %s.", msg.GetRenderId(), msg.GetSourceRenderId())}}
			},
		)),
		(cliapp.Command{Name: "analyze-conformance", Description: "Record an advisory Image Tools quality signal for an asset", Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoMutation}, Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "asset-id", Required: true, Description: "Asset identifier"}}}}).WithPrimitive(cliapp.ProtoMutation(
			func(ctx cliapp.OperationContext) (*studiov1.AnalyzeConformanceResponse, error) {
				resp, err := client.AnalyzeConformance(context.Background(), connect.NewRequest(&studiov1.AnalyzeConformanceRequest{AssetId: ctx.Positional("asset-id")}))
				if err != nil {
					return nil, cliapp.WrapAPIError("analyze conformance", err, nil)
				}
				return resp.Msg, nil
			},
			func(_ cliapp.OperationContext, msg *studiov1.AnalyzeConformanceResponse) cliapp.MutationReport {
				a := msg.GetAdvisory()
				return cliapp.MutationReport{Result: []string{fmt.Sprintf("Recorded advisory conformance score %.3f for %s.", a.GetScore(), a.GetAssetId())}, Changes: []string{"source=" + a.GetSource()}}
			},
		)),
	}}
}
