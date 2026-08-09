package measures

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/proto"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	measureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures/measures_v1connect"
	sharedmeasurepb "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

var windowValues = []string{"this_week", "last_7d", "last_30d", "this_month", "last_month", "this_quarter"}

type (
	rateResponse interface {
		proto.Message
		GetValidity() *measurepb.MeasureValidity
	}
	rateCall[Resp rateResponse] func(measureconnect.MeasuresServiceClient, context.Context, *sharedmeasurepb.TimeWindow) (Resp, error)
	renderer[Resp rateResponse] func(Resp) string
)

// Register exposes every declared friction question through its generated
// Connect client. The API is the sole computation owner.
func Register() cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "measures", Description: "Typed durable friction measures", NeedsAPI: true, Subcommands: []cliapp.Command{
		windowMeasure("external-tool-share", "Show external tool share", func(c measureconnect.MeasuresServiceClient, ctx context.Context, window *sharedmeasurepb.TimeWindow) (*measurepb.ExternalToolShareResponse, error) {
			response, err := c.ExternalToolShare(ctx, connect.NewRequest(&measurepb.ExternalToolShareRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		}, func(r *measurepb.ExternalToolShareResponse) string {
			return fmt.Sprintf("External-tool share: %.1f%% (%d external / %d resolved; %d unknown)", r.GetShare()*100, r.GetExternalCalls(), r.GetResolvedCalls(), r.GetUnknownCalls())
		}),
		windowMeasure("retry-rate", "Show tool invocation retry rate", func(c measureconnect.MeasuresServiceClient, ctx context.Context, window *sharedmeasurepb.TimeWindow) (*measurepb.RetryRateResponse, error) {
			response, err := c.RetryRate(ctx, connect.NewRequest(&measurepb.RetryRateRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		}, func(r *measurepb.RetryRateResponse) string {
			return fmt.Sprintf("Retry rate: %.1f%% (%d / %d calls)", r.GetRate()*100, r.GetRetryCalls(), r.GetTotalCalls())
		}),
		windowMeasure("help-recovery-rate", "Show help recovery rate", func(c measureconnect.MeasuresServiceClient, ctx context.Context, window *sharedmeasurepb.TimeWindow) (*measurepb.HelpRecoveryRateResponse, error) {
			response, err := c.HelpRecoveryRate(ctx, connect.NewRequest(&measurepb.HelpRecoveryRateRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		}, func(r *measurepb.HelpRecoveryRateResponse) string {
			return fmt.Sprintf("Help-recovery rate: %.1f%% (%d / %d calls)", r.GetRate()*100, r.GetHelpRecoveries(), r.GetTotalCalls())
		}),
		windowMeasure("repeated-work-rate", "Show repeated-work rate", func(c measureconnect.MeasuresServiceClient, ctx context.Context, window *sharedmeasurepb.TimeWindow) (*measurepb.RepeatedWorkRateResponse, error) {
			response, err := c.RepeatedWorkRate(ctx, connect.NewRequest(&measurepb.RepeatedWorkRateRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		}, func(r *measurepb.RepeatedWorkRateResponse) string {
			if validity := r.GetValidity(); validity != nil && validity.GetState() == "unreliable" {
				return fmt.Sprintf("Repeated-work rate unavailable: %s (%d / %d calls)", validity.GetReason(), r.GetRepeatedCalls(), r.GetTotalCalls())
			}
			return fmt.Sprintf("Repeated-work rate: %.1f%% (%d / %d calls)", r.GetRate()*100, r.GetRepeatedCalls(), r.GetTotalCalls())
		}),
		windowMeasure("tool-failure-rate", "Show tool failure rate", func(c measureconnect.MeasuresServiceClient, ctx context.Context, window *sharedmeasurepb.TimeWindow) (*measurepb.ToolFailureRateResponse, error) {
			response, err := c.ToolFailureRate(ctx, connect.NewRequest(&measurepb.ToolFailureRateRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		}, func(r *measurepb.ToolFailureRateResponse) string {
			return fmt.Sprintf("Tool failure rate: %.1f%% (%d / %d calls)", r.GetRate()*100, r.GetFailedCalls(), r.GetTotalCalls())
		}),
		windowMeasure("run-success-rate", "Show terminal run success rate", func(c measureconnect.MeasuresServiceClient, ctx context.Context, window *sharedmeasurepb.TimeWindow) (*measurepb.RunSuccessRateResponse, error) {
			response, err := c.RunSuccessRate(ctx, connect.NewRequest(&measurepb.RunSuccessRateRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		}, func(r *measurepb.RunSuccessRateResponse) string {
			return fmt.Sprintf("Run success rate: %.1f%% (%d / %d terminal runs)", r.GetRate()*100, r.GetSuccessfulRuns(), r.GetTerminalRuns())
		}),
		windowMeasure("run-cycle-time", "Show average completed run cycle time", func(c measureconnect.MeasuresServiceClient, ctx context.Context, window *sharedmeasurepb.TimeWindow) (*measurepb.RunCycleTimeResponse, error) {
			response, err := c.RunCycleTime(ctx, connect.NewRequest(&measurepb.RunCycleTimeRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		}, func(r *measurepb.RunCycleTimeResponse) string {
			return fmt.Sprintf("Average run cycle time: %.0f ms (%d completed runs)", r.GetAverageDurationMs(), r.GetCompletedDurationRuns())
		}),
		windowMeasure("run-duration-statistics", "Show complete durable run duration statistics", func(c measureconnect.MeasuresServiceClient, ctx context.Context, window *sharedmeasurepb.TimeWindow) (*measurepb.RunDurationStatisticsResponse, error) {
			response, err := c.RunDurationStatistics(ctx, connect.NewRequest(&measurepb.RunDurationStatisticsRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		}, func(r *measurepb.RunDurationStatisticsResponse) string {
			return fmt.Sprintf("Run durations: avg %.0f ms, p50 %.0f ms, p95 %.0f ms, p99 %.0f ms, %d-%d ms (%d runs)", r.GetAverageDurationMs(), r.GetP50DurationMs(), r.GetP95DurationMs(), r.GetP99DurationMs(), r.GetMinDurationMs(), r.GetMaxDurationMs(), r.GetCount())
		}),
		windowMeasure("run-cost", "Show durable terminal run cost", func(c measureconnect.MeasuresServiceClient, ctx context.Context, window *sharedmeasurepb.TimeWindow) (*measurepb.RunCostResponse, error) {
			response, err := c.RunCost(ctx, connect.NewRequest(&measurepb.RunCostRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		}, func(r *measurepb.RunCostResponse) string {
			return fmt.Sprintf("Run cost: $%.4f across %d runs (%d tokens)", r.GetTotalCostUsd(), r.GetTotalRuns(), r.GetTotalTokens())
		}),
		windowMeasure("run-volume", "Show durable terminal run volume", func(c measureconnect.MeasuresServiceClient, ctx context.Context, window *sharedmeasurepb.TimeWindow) (*measurepb.RunVolumeResponse, error) {
			response, err := c.RunVolume(ctx, connect.NewRequest(&measurepb.RunVolumeRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		}, func(r *measurepb.RunVolumeResponse) string {
			return fmt.Sprintf("Run volume: %d runs (%d terminal)", r.GetTotalRuns(), r.GetTerminalRuns())
		}),
		windowMeasure("run-status-distribution", "Show terminal run status distribution", func(c measureconnect.MeasuresServiceClient, ctx context.Context, window *sharedmeasurepb.TimeWindow) (*measurepb.RunStatusDistributionResponse, error) {
			response, err := c.RunStatusDistribution(ctx, connect.NewRequest(&measurepb.RunStatusDistributionRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		}, func(r *measurepb.RunStatusDistributionResponse) string {
			parts := make([]string, 0, len(r.GetRows()))
			for _, row := range r.GetRows() {
				parts = append(parts, fmt.Sprintf("%s=%d", row.GetStatus(), row.GetCount()))
			}
			return "Run status distribution: " + strings.Join(parts, ", ")
		}),
		windowMeasure("runner-breakdown", "Show terminal run breakdown by runner", func(c measureconnect.MeasuresServiceClient, ctx context.Context, window *sharedmeasurepb.TimeWindow) (*measurepb.RunnerBreakdownResponse, error) {
			response, err := c.RunnerBreakdown(ctx, connect.NewRequest(&measurepb.RunnerBreakdownRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		}, func(r *measurepb.RunnerBreakdownResponse) string { return renderBreakdown("Runner", r.GetRows()) }),
		windowMeasure("model-breakdown", "Show terminal run breakdown by model", func(c measureconnect.MeasuresServiceClient, ctx context.Context, window *sharedmeasurepb.TimeWindow) (*measurepb.ModelBreakdownResponse, error) {
			response, err := c.ModelBreakdown(ctx, connect.NewRequest(&measurepb.ModelBreakdownRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		}, func(r *measurepb.ModelBreakdownResponse) string { return renderBreakdown("Model", r.GetRows()) }),
		windowMeasure("profile-breakdown", "Show terminal run breakdown by profile", func(c measureconnect.MeasuresServiceClient, ctx context.Context, window *sharedmeasurepb.TimeWindow) (*measurepb.ProfileBreakdownResponse, error) {
			response, err := c.ProfileBreakdown(ctx, connect.NewRequest(&measurepb.ProfileBreakdownRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		}, func(r *measurepb.ProfileBreakdownResponse) string { return renderBreakdown("Profile", r.GetRows()) }),
		windowMeasure("terminal-run-trend", "Show hourly terminal run trend", func(c measureconnect.MeasuresServiceClient, ctx context.Context, window *sharedmeasurepb.TimeWindow) (*measurepb.TerminalRunTrendResponse, error) {
			response, err := c.TerminalRunTrend(ctx, connect.NewRequest(&measurepb.TerminalRunTrendRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		}, func(r *measurepb.TerminalRunTrendResponse) string {
			parts := make([]string, 0, len(r.GetRows()))
			for _, row := range r.GetRows() {
				parts = append(parts, fmt.Sprintf("%s: %d terminal (%d complete, %d failed, %d cancelled)", row.GetBucket(), row.GetTerminalRuns(), row.GetCompletedRuns(), row.GetFailedRuns(), row.GetCancelledRuns()))
			}
			return "Terminal run trend: " + strings.Join(parts, "; ")
		}),
		windowMeasure("tool-usage", "Show durable tool usage and outcomes", func(c measureconnect.MeasuresServiceClient, ctx context.Context, window *sharedmeasurepb.TimeWindow) (*measurepb.ToolUsageResponse, error) {
			response, err := c.ToolUsage(ctx, connect.NewRequest(&measurepb.ToolUsageRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		}, func(r *measurepb.ToolUsageResponse) string {
			parts := make([]string, 0, len(r.GetRows()))
			for _, row := range r.GetRows() {
				parts = append(parts, fmt.Sprintf("%s=%d (%d success, %d failed)", row.GetToolName(), row.GetCallCount(), row.GetSuccessCount(), row.GetFailedCount()))
			}
			return "Tool usage: " + strings.Join(parts, ", ")
		}),
		windowMeasure("capability-usage", "Show receipt-backed project capability usage", func(c measureconnect.MeasuresServiceClient, ctx context.Context, window *sharedmeasurepb.TimeWindow) (*measurepb.CapabilityUsageResponse, error) {
			response, err := c.CapabilityUsage(ctx, connect.NewRequest(&measurepb.CapabilityUsageRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		}, func(r *measurepb.CapabilityUsageResponse) string {
			if validity := r.GetValidity(); validity != nil && validity.GetState() != "available" {
				return fmt.Sprintf("Capability usage unavailable: %s", validity.GetReason())
			}
			parts := make([]string, 0, len(r.GetRows()))
			for _, row := range r.GetRows() {
				parts = append(parts, fmt.Sprintf("%s %s=%d (%d success, %d failed, %d ms)", row.GetTargetScenario(), row.GetOperation(), row.GetCallCount(), row.GetSuccessCount(), row.GetFailedCount(), row.GetTotalDurationMs()))
			}
			return "Capability usage: " + strings.Join(parts, ", ")
		}),
		windowMeasure("capability-efficacy", "Show receipt-backed capability efficacy counts", func(c measureconnect.MeasuresServiceClient, ctx context.Context, window *sharedmeasurepb.TimeWindow) (*measurepb.CapabilityEfficacyResponse, error) {
			response, err := c.CapabilityEfficacy(ctx, connect.NewRequest(&measurepb.CapabilityEfficacyRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		}, func(r *measurepb.CapabilityEfficacyResponse) string {
			if validity := r.GetValidity(); validity != nil && validity.GetState() != "available" {
				return fmt.Sprintf("Capability efficacy unavailable: %s", validity.GetReason())
			}
			parts := make([]string, 0, len(r.GetRows()))
			for _, row := range r.GetRows() {
				parts = append(parts, fmt.Sprintf("%s %s=%d (%d success, %d fallback-after, %d abandoned)", row.GetTargetScenario(), row.GetOperation(), row.GetCallCount(), row.GetSuccessCount(), row.GetFallbackAfterCount(), row.GetAbandonedCount()))
			}
			return "Capability efficacy: " + strings.Join(parts, ", ")
		}),
		windowMeasure("error-patterns", "Show durable agent error patterns", func(c measureconnect.MeasuresServiceClient, ctx context.Context, window *sharedmeasurepb.TimeWindow) (*measurepb.ErrorPatternsResponse, error) {
			response, err := c.ErrorPatterns(ctx, connect.NewRequest(&measurepb.ErrorPatternsRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		}, func(r *measurepb.ErrorPatternsResponse) string {
			parts := make([]string, 0, len(r.GetRows()))
			for _, row := range r.GetRows() {
				parts = append(parts, fmt.Sprintf("%s=%d (sample %s)", row.GetErrorCode(), row.GetCount(), row.GetSampleRunId()))
			}
			return "Error patterns: " + strings.Join(parts, ", ")
		}),
		windowMeasure("file-reread-rate", "Show file reread rate", func(c measureconnect.MeasuresServiceClient, ctx context.Context, window *sharedmeasurepb.TimeWindow) (*measurepb.FileRereadRateResponse, error) {
			response, err := c.FileRereadRate(ctx, connect.NewRequest(&measurepb.FileRereadRateRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		}, func(r *measurepb.FileRereadRateResponse) string {
			return fmt.Sprintf("File reread rate: %.1f%% (%d rereads / %d reads)", r.GetRate()*100, r.GetFilesReadMoreThanOnce(), r.GetReadCalls())
		}),
		windowMeasure("finding-recurrence-rate", "Show investigation finding recurrence rate", func(c measureconnect.MeasuresServiceClient, ctx context.Context, window *sharedmeasurepb.TimeWindow) (*measurepb.FindingRecurrenceRateResponse, error) {
			response, err := c.FindingRecurrenceRate(ctx, connect.NewRequest(&measurepb.FindingRecurrenceRateRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		}, func(r *measurepb.FindingRecurrenceRateResponse) string {
			return fmt.Sprintf("Finding recurrence rate: %.1f%% (%d recurring / %d findings; %d fingerprints)", r.GetRate()*100, r.GetRecurringFindings(), r.GetTotalFindings(), r.GetRecurringFingerprints())
		}),
		windowMeasure("select-cohort", "Select the durable run cohort behind a measure", func(c measureconnect.MeasuresServiceClient, ctx context.Context, window *sharedmeasurepb.TimeWindow) (*measurepb.SelectCohortResponse, error) {
			response, err := c.SelectCohort(ctx, connect.NewRequest(&measurepb.SelectCohortRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		}, func(r *measurepb.SelectCohortResponse) string {
			return fmt.Sprintf("Selected run cohort: %s", strings.Join(r.GetRunIds(), ", "))
		}),
		genericMeasure("workload-efficiency", "Show consumption per successful workload completion", "throughput.workload_efficiency", []cliapp.Flag{
			{Name: "window", Description: "Analytical time window", Values: windowValues},
			{Name: "workload-key", Description: "Declared workload key"},
			{Name: "runner-type", Description: "Runner type"},
			{Name: "model", Description: "Model"},
		}),
		genericMeasure("workload-breakdown", "Show terminal run performance grouped by workload", "throughput.workload_breakdown", []cliapp.Flag{
			{Name: "window", Description: "Analytical time window", Values: windowValues},
		}),
		tokenAttributionMeasure(),
	}}
}

var tokenAttributionByValues = []string{"capability", "executable", "command-path", "scenario-operation"}
var tokenAttributionViewValues = []string{"footprint", "residency", "incurred"}

func tokenAttributionMeasure() cliapp.Command {
	flags := []cliapp.Flag{
		{Name: "window", Description: "Analytical time window", Values: windowValues},
		{Name: "by", Description: "Grouping dimension", Values: tokenAttributionByValues, Default: "capability"},
		{Name: "view", Description: "Token view", Values: tokenAttributionViewValues, Default: "footprint"},
		{Name: "runner-type", Description: "Runner type"},
		{Name: "model", Description: "Model"},
		{Name: "workload-key", Description: "Declared workload key"},
	}
	return (cliapp.Command{
		Name: "token-attribution", Description: "Show durable token attribution grouped by dimension and view [--window TOKEN] [--json]", NeedsAPI: true,
		Args: cliapp.ArgSchema{Flags: flags},
	}).WithPrimitive(cliapp.ProtoListEmitUnpopulatedJSON(
		func(op cliapp.OperationContext) (*measurepb.TokenAttributionResponse, error) {
			groupBy, err := normalizeTokenAttributionBy(op.Flag("by"))
			if err != nil {
				return nil, err
			}
			window, err := parseWindow(op.Flag("window"))
			if err != nil {
				return nil, err
			}
			filter := &measurepb.InvocationFilter{
				RunnerType:  op.Flag("runner-type"),
				Model:       op.Flag("model"),
				WorkloadKey: op.Flag("workload-key"),
			}
			if filter.RunnerType == "" && filter.Model == "" && filter.WorkloadKey == "" {
				filter = nil
			}
			h, base := cliapp.NewConnectHTTPClient(op.Core())
			response, err := measureconnect.NewMeasuresServiceClient(h, base).TokenAttribution(context.Background(), connect.NewRequest(&measurepb.TokenAttributionRequest{
				Window:  window,
				Filter:  filter,
				GroupBy: groupBy,
				View:    op.Flag("view"),
			}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *measurepb.TokenAttributionResponse) cliapp.ListReport {
			if validity := response.GetValidity(); validity != nil && validity.GetState() == "unreliable" {
				return cliapp.ListReport{Summary: []string{"Token attribution unavailable: " + validity.GetReason()}}
			}
			parts := make([]string, 0, len(response.GetRows()))
			for _, row := range response.GetRows() {
				parts = append(parts, fmt.Sprintf("%s=%d tokens (estimated share %.2f%%)", row.GetValue(), row.GetTotalTokens(), row.GetEstimatedTokenShare()*100))
			}
			return cliapp.ListReport{Summary: append([]string{
				fmt.Sprintf("Token attribution by %s (%s); estimated token share %.2f%%", response.GetGroupBy(), response.GetView(), response.GetEstimatedTokenShare()*100),
			}, parts...)}
		},
	))
}

func normalizeTokenAttributionBy(value string) (string, error) {
	switch value {
	case "capability", "executable":
		return value, nil
	case "command-path":
		return "command_path", nil
	case "scenario-operation":
		return "target_scenario_operation", nil
	default:
		return "", fmt.Errorf("unsupported token attribution grouping %q; accepted values: %s", value, strings.Join(tokenAttributionByValues, ", "))
	}
}

func genericMeasure(name, description, measure string, flags []cliapp.Flag) cliapp.Command {
	return cliapp.Command{
		Name: name, Description: description + " [--window TOKEN] [--json]", NeedsAPI: true,
		Args: cliapp.ArgSchema{Flags: flags},
		RunCtx: func(ctx cliapp.RunContext) error {
			params := map[string]string{"window": ctx.Flag("window")}
			for _, flag := range flags {
				if flag.Name == "window" {
					continue
				}
				if value := ctx.Flag(flag.Name); value != "" {
					params[strings.ReplaceAll(flag.Name, "-", "_")] = value
				}
			}
			body, err := ctx.Core().Request("POST", "measures/execute", nil, map[string]any{"measure": measure, "params": params})
			if err != nil {
				return err
			}
			if ctx.JSON() {
				_, err = ctx.Stdout().Write(append(body, '\n'))
				return err
			}
			var result struct {
				Value  string              `json:"value"`
				Fields []map[string]string `json:"fields"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				return err
			}
			fmt.Fprintf(ctx.Stdout(), "%s: %s\n", name, result.Value)
			for _, field := range result.Fields {
				encoded, _ := json.Marshal(field)
				fmt.Fprintln(ctx.Stdout(), string(encoded))
			}
			return nil
		},
	}
}

func renderBreakdown(label string, rows []*measurepb.RunBreakdownRow) string {
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		parts = append(parts, fmt.Sprintf("%s=%d (%d success, %d failed)", row.GetValue(), row.GetRunCount(), row.GetSuccessCount(), row.GetFailedCount()))
	}
	return label + " breakdown: " + strings.Join(parts, ", ")
}

func windowMeasure[Resp rateResponse](name, description string, call rateCall[Resp], render renderer[Resp]) cliapp.Command {
	cmd := cliapp.Command{Name: name, NeedsAPI: true, Description: description + " [--window TOKEN] [--json]", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "window", Description: "Analytical time window", Values: windowValues}}}}
	return cmd.WithPrimitive(cliapp.ProtoList(
		func(op cliapp.OperationContext) (Resp, error) {
			window, err := parseWindow(op.Flag("window"))
			if err != nil {
				var zero Resp
				return zero, err
			}
			h, base := cliapp.NewConnectHTTPClient(op.Core())
			return call(measureconnect.NewMeasuresServiceClient(h, base), context.Background(), window)
		},
		func(_ cliapp.OperationContext, response Resp) cliapp.ListReport {
			if validity := response.GetValidity(); validity != nil && validity.GetState() == "unreliable" {
				return cliapp.ListReport{Summary: []string{"Measure unavailable: " + validity.GetReason()}}
			}
			return cliapp.ListReport{Summary: []string{render(response)}}
		},
	))
}

func parseWindow(value string) (*sharedmeasurepb.TimeWindow, error) {
	if value == "" {
		return nil, nil
	}
	key := "TIME_WINDOW_TOKEN_" + strings.ReplaceAll(strings.ToUpper(value), "-", "_")
	number, ok := sharedmeasurepb.TimeWindowToken_value[key]
	if !ok || number == int32(sharedmeasurepb.TimeWindowToken_TIME_WINDOW_TOKEN_UNSPECIFIED) {
		return nil, fmt.Errorf("unsupported time window %q", value)
	}
	return &sharedmeasurepb.TimeWindow{Window: &sharedmeasurepb.TimeWindow_Token{Token: sharedmeasurepb.TimeWindowToken(number)}}, nil
}
