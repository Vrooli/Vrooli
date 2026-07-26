package measures

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/proto"

	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
	smmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/measures"
	smmeasuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/measures/measures_v1connect"
)

var windowValues = []string{"this_week", "last_7d", "last_30d", "this_month", "last_month", "this_quarter"}

// Register exposes typed, parameterized analytical questions. The commands use
// renderer-separated cli-core primitives, so their request work cannot vary by
// output mode and their manifest architecture declarations are verified at
// runtime.
func Register() cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "measures",
		Description: "Typed analytical measures over Swarm Manager history",
		Subcommands: []cliapp.Command{
			windowMeasure("backlog-completed", "Count completed backlog items", func(c smmeasuresconnect.MeasuresServiceClient, ctx context.Context, window *measuresv1.TimeWindow) (*smmeasuresv1.CountBacklogCompletedResponse, error) {
				response, err := c.CountBacklogCompleted(ctx, connect.NewRequest(&smmeasuresv1.CountBacklogCompletedRequest{Window: window}))
				if err != nil {
					return nil, err
				}
				return response.Msg, nil
			}),
			windowMeasure("backlog-created", "Count created backlog items", func(c smmeasuresconnect.MeasuresServiceClient, ctx context.Context, window *measuresv1.TimeWindow) (*smmeasuresv1.CountBacklogCreatedResponse, error) {
				response, err := c.CountBacklogCreated(ctx, connect.NewRequest(&smmeasuresv1.CountBacklogCreatedRequest{Window: window}))
				if err != nil {
					return nil, err
				}
				return response.Msg, nil
			}),
			windowMeasure("backlog-net-delta", "Calculate created minus completed backlog items", func(c smmeasuresconnect.MeasuresServiceClient, ctx context.Context, window *measuresv1.TimeWindow) (*smmeasuresv1.CountBacklogNetDeltaResponse, error) {
				response, err := c.CountBacklogNetDelta(ctx, connect.NewRequest(&smmeasuresv1.CountBacklogNetDeltaRequest{Window: window}))
				if err != nil {
					return nil, err
				}
				return response.Msg, nil
			}),
			windowMeasure("execution-completed", "Count completed executions", func(c smmeasuresconnect.MeasuresServiceClient, ctx context.Context, window *measuresv1.TimeWindow) (*smmeasuresv1.CountExecutionsCompletedResponse, error) {
				response, err := c.CountExecutionsCompleted(ctx, connect.NewRequest(&smmeasuresv1.CountExecutionsCompletedRequest{Window: window}))
				if err != nil {
					return nil, err
				}
				return response.Msg, nil
			}),
			windowMeasure("goal-created", "Count created goals", func(c smmeasuresconnect.MeasuresServiceClient, ctx context.Context, window *measuresv1.TimeWindow) (*smmeasuresv1.CountGoalsCreatedResponse, error) {
				response, err := c.CountGoalsCreated(ctx, connect.NewRequest(&smmeasuresv1.CountGoalsCreatedRequest{Window: window}))
				if err != nil {
					return nil, err
				}
				return response.Msg, nil
			}),
			windowMeasure("agent-session-created", "Count created agent sessions", func(c smmeasuresconnect.MeasuresServiceClient, ctx context.Context, window *measuresv1.TimeWindow) (*smmeasuresv1.CountAgentSessionsCreatedResponse, error) {
				response, err := c.CountAgentSessionsCreated(ctx, connect.NewRequest(&smmeasuresv1.CountAgentSessionsCreatedRequest{Window: window}))
				if err != nil {
					return nil, err
				}
				return response.Msg, nil
			}),
			planRefMeasure(),
			goalMilestoneHealthMeasure(),
			backlogOpenMeasure(),
			backlogBlockedMeasure(),
			backlogLeadTimeMeasure(),
			agentSessionProposalRateMeasure(),
			executionSuccessRateMeasure(),
			executionDurationMeasure(),
			executionReviewRateMeasure(),
		},
	}
}

type countResponse interface {
	proto.Message
	GetCount() int64
}

type windowCall[Resp countResponse] func(smmeasuresconnect.MeasuresServiceClient, context.Context, *measuresv1.TimeWindow) (Resp, error)

func windowMeasure[Resp countResponse](name, description string, call windowCall[Resp]) cliapp.Command {
	cmd := cliapp.Command{
		Name: name, NeedsAPI: true, Description: description + " [--window TOKEN] [--json]",
		Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{
			Name: "window", Description: "Analytical time window", Values: windowValues,
		}}},
	}
	return cmd.WithPrimitive(cliapp.ProtoList(
		func(op cliapp.OperationContext) (Resp, error) {
			window, err := parseWindow(op.Flag("window"))
			if err != nil {
				var zero Resp
				return zero, err
			}
			h, base := cliapp.NewConnectHTTPClient(op.Core())
			return call(smmeasuresconnect.NewMeasuresServiceClient(h, base), context.Background(), window)
		},
		func(_ cliapp.OperationContext, response Resp) cliapp.ListReport {
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Count: %d", response.GetCount())}}
		},
	))
}

func planRefMeasure() cliapp.Command {
	cmd := cliapp.Command{Name: "plan-ref-count", NeedsAPI: true, Description: "Count backlog items with a canonical plan reference [--json]"}
	return cmd.WithPrimitive(cliapp.ProtoList(
		func(op cliapp.OperationContext) (*smmeasuresv1.CountPlanRefsResponse, error) {
			h, base := cliapp.NewConnectHTTPClient(op.Core())
			response, err := smmeasuresconnect.NewMeasuresServiceClient(h, base).CountPlanRefs(context.Background(), connect.NewRequest(&smmeasuresv1.CountPlanRefsRequest{}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *smmeasuresv1.CountPlanRefsResponse) cliapp.ListReport {
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Count: %d", response.GetCount())}}
		},
	))
}

func goalMilestoneHealthMeasure() cliapp.Command {
	cmd := cliapp.Command{Name: "goal-milestone-health", NeedsAPI: true, Description: "Show current completion and blocking health by milestone [--json]"}
	return cmd.WithPrimitive(cliapp.ProtoList(
		func(op cliapp.OperationContext) (*smmeasuresv1.GoalMilestoneHealthResponse, error) {
			h, base := cliapp.NewConnectHTTPClient(op.Core())
			response, err := smmeasuresconnect.NewMeasuresServiceClient(h, base).GoalMilestoneHealth(context.Background(), connect.NewRequest(&smmeasuresv1.GoalMilestoneHealthRequest{}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *smmeasuresv1.GoalMilestoneHealthResponse) cliapp.ListReport {
			rows := make([]string, 0, len(response.GetMilestones()))
			for _, milestone := range response.GetMilestones() {
				rows = append(rows, fmt.Sprintf("%s: total=%d completed=%d in_progress=%d blocked=%d", milestone.GetMilestone(), milestone.GetTotal(), milestone.GetCompleted(), milestone.GetInProgress(), milestone.GetBlocked()))
			}
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Milestones: %d", len(rows))}, ResultsHeading: "Milestone health", Results: rows}
		},
	))
}

func backlogOpenMeasure() cliapp.Command {
	cmd := cliapp.Command{Name: "backlog-open", NeedsAPI: true, Description: "Count actionable backlog items currently open [--json]"}
	return cmd.WithPrimitive(cliapp.ProtoList(
		func(op cliapp.OperationContext) (*smmeasuresv1.CountBacklogOpenResponse, error) {
			h, base := cliapp.NewConnectHTTPClient(op.Core())
			response, err := smmeasuresconnect.NewMeasuresServiceClient(h, base).CountBacklogOpen(context.Background(), connect.NewRequest(&smmeasuresv1.CountBacklogOpenRequest{}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *smmeasuresv1.CountBacklogOpenResponse) cliapp.ListReport {
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Count: %d", response.GetCount())}}
		},
	))
}

func backlogBlockedMeasure() cliapp.Command {
	cmd := cliapp.Command{Name: "backlog-blocked", NeedsAPI: true, Description: "Count backlog items blocked by unresolved dependencies [--json]"}
	return cmd.WithPrimitive(cliapp.ProtoList(
		func(op cliapp.OperationContext) (*smmeasuresv1.CountBacklogBlockedResponse, error) {
			h, base := cliapp.NewConnectHTTPClient(op.Core())
			response, err := smmeasuresconnect.NewMeasuresServiceClient(h, base).CountBacklogBlocked(context.Background(), connect.NewRequest(&smmeasuresv1.CountBacklogBlockedRequest{}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *smmeasuresv1.CountBacklogBlockedResponse) cliapp.ListReport {
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Count: %d", response.GetCount())}}
		},
	))
}

func agentSessionProposalRateMeasure() cliapp.Command {
	cmd := cliapp.Command{Name: "agent-session-proposal-rate", NeedsAPI: true, Description: "Calculate agent-session proposal apply rate [--window TOKEN] [--json]", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "window", Description: "Analytical time window", Values: windowValues}}}}
	return cmd.WithPrimitive(cliapp.ProtoList(
		func(op cliapp.OperationContext) (*smmeasuresv1.AgentSessionProposalRateResponse, error) {
			window, err := parseWindow(op.Flag("window"))
			if err != nil {
				return nil, err
			}
			h, base := cliapp.NewConnectHTTPClient(op.Core())
			response, err := smmeasuresconnect.NewMeasuresServiceClient(h, base).AgentSessionProposalRate(context.Background(), connect.NewRequest(&smmeasuresv1.AgentSessionProposalRateRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *smmeasuresv1.AgentSessionProposalRateResponse) cliapp.ListReport {
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Proposal apply rate: %.1f%% (%d proposals)", response.GetRate()*100, response.GetSampleSize())}}
		},
	))
}

func executionSuccessRateMeasure() cliapp.Command {
	cmd := cliapp.Command{Name: "execution-success-rate", NeedsAPI: true, Description: "Calculate execution success rate [--window TOKEN] [--json]", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "window", Description: "Analytical time window", Values: windowValues}}}}
	return cmd.WithPrimitive(cliapp.ProtoList(
		func(op cliapp.OperationContext) (*smmeasuresv1.ExecutionSuccessRateResponse, error) {
			window, err := parseWindow(op.Flag("window"))
			if err != nil {
				return nil, err
			}
			h, base := cliapp.NewConnectHTTPClient(op.Core())
			response, err := smmeasuresconnect.NewMeasuresServiceClient(h, base).ExecutionSuccessRate(context.Background(), connect.NewRequest(&smmeasuresv1.ExecutionSuccessRateRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *smmeasuresv1.ExecutionSuccessRateResponse) cliapp.ListReport {
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Execution success rate: %.1f%% (%d terminal executions)", response.GetRate()*100, response.GetSampleSize())}}
		},
	))
}

func backlogLeadTimeMeasure() cliapp.Command {
	cmd := cliapp.Command{Name: "backlog-lead-time", NeedsAPI: true, Description: "Calculate backlog creation-to-completion lead time [--window TOKEN] [--json]", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "window", Description: "Analytical time window", Values: windowValues}}}}
	return cmd.WithPrimitive(cliapp.ProtoList(
		func(op cliapp.OperationContext) (*smmeasuresv1.BacklogLeadTimeResponse, error) {
			window, err := parseWindow(op.Flag("window"))
			if err != nil {
				return nil, err
			}
			h, base := cliapp.NewConnectHTTPClient(op.Core())
			response, err := smmeasuresconnect.NewMeasuresServiceClient(h, base).BacklogLeadTime(context.Background(), connect.NewRequest(&smmeasuresv1.BacklogLeadTimeRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *smmeasuresv1.BacklogLeadTimeResponse) cliapp.ListReport {
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Backlog lead time: %.1f average hours, %.1f median (%d completed items)", response.GetAverageHours(), response.GetMedianHours(), response.GetSampleSize())}}
		},
	))
}

func executionDurationMeasure() cliapp.Command {
	cmd := cliapp.Command{Name: "execution-duration", NeedsAPI: true, Description: "Calculate completed execution duration [--window TOKEN] [--json]", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "window", Description: "Analytical time window", Values: windowValues}}}}
	return cmd.WithPrimitive(cliapp.ProtoList(
		func(op cliapp.OperationContext) (*smmeasuresv1.ExecutionDurationResponse, error) {
			window, err := parseWindow(op.Flag("window"))
			if err != nil {
				return nil, err
			}
			h, base := cliapp.NewConnectHTTPClient(op.Core())
			response, err := smmeasuresconnect.NewMeasuresServiceClient(h, base).ExecutionDuration(context.Background(), connect.NewRequest(&smmeasuresv1.ExecutionDurationRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *smmeasuresv1.ExecutionDurationResponse) cliapp.ListReport {
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Execution duration: %.1f average minutes, %.1f median (%d completed runs)", response.GetAverageMinutes(), response.GetMedianMinutes(), response.GetSampleSize())}}
		},
	))
}

func executionReviewRateMeasure() cliapp.Command {
	cmd := cliapp.Command{Name: "execution-review-rate", NeedsAPI: true, Description: "Calculate terminal execution review rate [--window TOKEN] [--json]", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "window", Description: "Analytical time window", Values: windowValues}}}}
	return cmd.WithPrimitive(cliapp.ProtoList(
		func(op cliapp.OperationContext) (*smmeasuresv1.ExecutionReviewRateResponse, error) {
			window, err := parseWindow(op.Flag("window"))
			if err != nil {
				return nil, err
			}
			h, base := cliapp.NewConnectHTTPClient(op.Core())
			response, err := smmeasuresconnect.NewMeasuresServiceClient(h, base).ExecutionReviewRate(context.Background(), connect.NewRequest(&smmeasuresv1.ExecutionReviewRateRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *smmeasuresv1.ExecutionReviewRateResponse) cliapp.ListReport {
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Execution review rate: %.1f%% (%d terminal executions)", response.GetRate()*100, response.GetSampleSize())}}
		},
	))
}

func parseWindow(value string) (*measuresv1.TimeWindow, error) {
	if value == "" {
		return nil, nil
	}
	key := "TIME_WINDOW_TOKEN_" + strings.ReplaceAll(strings.ToUpper(value), "-", "_")
	number, ok := measuresv1.TimeWindowToken_value[key]
	if !ok || number == int32(measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_UNSPECIFIED) {
		return nil, fmt.Errorf("unsupported time window %q", value)
	}
	return &measuresv1.TimeWindow{Window: &measuresv1.TimeWindow_Token{Token: measuresv1.TimeWindowToken(number)}}, nil
}
