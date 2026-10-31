package measures

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	measureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures/measures_v1connect"
)

// SelectorFlags is the one cohort selector every cohort command accepts
// (measures and usage compare):
// --cohort NAME | --run-ids ID,ID | --filter-json '{...}', each over --window.
func SelectorFlags(prefix string) []cliapp.Flag {
	return []cliapp.Flag{
		{Name: prefix + "window", Description: "Analytical time window", Values: windowValues},
		{Name: prefix + "cohort", Description: "Named durable cohort"},
		{Name: prefix + "run-ids", Description: "Comma-separated run UUIDs (at most 100)"},
		{Name: prefix + "filter-json", Description: "InvocationFilter JSON, e.g. '{\"tag_prefix\":\"plan-x\"}'"},
	}
}

func ReadSelector(op cliapp.OperationContext, prefix string) (*measurepb.CohortSelector, error) {
	window, err := parseWindow(op.Flag(prefix + "window"))
	if err != nil {
		return nil, err
	}
	selector := &measurepb.CohortSelector{Window: window, Cohort: strings.TrimSpace(op.Flag(prefix + "cohort"))}
	for _, id := range strings.Split(op.Flag(prefix+"run-ids"), ",") {
		if id = strings.TrimSpace(id); id != "" {
			selector.RunIds = append(selector.RunIds, id)
		}
	}
	if raw := strings.TrimSpace(op.Flag(prefix + "filter-json")); raw != "" {
		selector.Filter = &measurepb.InvocationFilter{}
		if err := protojson.Unmarshal([]byte(raw), selector.Filter); err != nil {
			return nil, fmt.Errorf("--%sfilter-json: %w", prefix, err)
		}
	}
	return selector, nil
}

func limitFlag(op cliapp.OperationContext) (int32, error) {
	raw := op.Flag("limit")
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || value < 1 {
		return 0, fmt.Errorf("--limit must be a positive integer no larger than %d", math.MaxInt32)
	}
	return int32(value), nil
}

func client(op cliapp.OperationContext) measureconnect.MeasuresServiceClient {
	h, base := cliapp.NewConnectHTTPClient(op.Core())
	return measureconnect.NewMeasuresServiceClient(h, base)
}

func cohortCommand(name, description string, flags []cliapp.Flag, call func(cliapp.OperationContext) (proto.Message, error), render func(proto.Message) []string) cliapp.Command {
	cmd := cliapp.Command{Name: name, NeedsAPI: true, Description: description + " [--json]", Args: cliapp.ArgSchema{Flags: flags}}
	return cmd.WithPrimitive(cliapp.ProtoList(call, func(_ cliapp.OperationContext, response proto.Message) cliapp.ListReport {
		return cliapp.ListReport{Summary: render(response)}
	}))
}

var limitDef = cliapp.Flag{Name: "limit", Description: "Maximum runs or signals"}

func selectCohortCommand() cliapp.Command {
	return cohortCommand("select-cohort", "Select the durable runs of a cohort (--cohort | --run-ids | --filter-json)", append(SelectorFlags(""), limitDef),
		func(op cliapp.OperationContext) (proto.Message, error) {
			s, err := ReadSelector(op, "")
			if err != nil {
				return nil, err
			}
			limit, err := limitFlag(op)
			if err != nil {
				return nil, err
			}
			r, err := client(op).SelectCohort(context.Background(), connect.NewRequest(&measurepb.SelectCohortRequest{Window: s.Window, Filter: s.Filter, Cohort: s.Cohort, RunIds: s.RunIds, Limit: limit}))
			if err != nil {
				return nil, err
			}
			return r.Msg, nil
		}, func(m proto.Message) []string {
			r := m.(*measurepb.SelectCohortResponse)
			lines := []string{fmt.Sprintf("Selected run cohort: %d runs (truncated=%t)", len(r.GetRunIds()), r.GetTruncated())}
			for _, row := range r.GetRows() {
				lines = append(lines, fmt.Sprintf("%s %s %s %s tokens=%d", row.GetRunId(), row.GetStatus(), row.GetRunnerType(), row.GetModel(), row.GetTotalTokens()))
			}
			return lines
		})
}

func cohortReportCommand() cliapp.Command {
	return cohortCommand("cohort-report", "Fold run reports across a cohort: friction signals, time accounting, goal progression", append(SelectorFlags(""), limitDef),
		func(op cliapp.OperationContext) (proto.Message, error) {
			s, err := ReadSelector(op, "")
			if err != nil {
				return nil, err
			}
			limit, err := limitFlag(op)
			if err != nil {
				return nil, err
			}
			r, err := client(op).CohortReport(context.Background(), connect.NewRequest(&measurepb.CohortReportRequest{Window: s.Window, Filter: s.Filter, Cohort: s.Cohort, RunIds: s.RunIds, Limit: limit}))
			if err != nil {
				return nil, err
			}
			return r.Msg, nil
		}, func(m proto.Message) []string {
			r := m.(*measurepb.CohortReportResponse)
			g := r.GetGoal()
			lines := []string{
				fmt.Sprintf("Cohort: %d runs | classifier %s | evidence %s %s", len(r.GetRunIds()), r.GetClassifierVersion(), r.GetAvailabilityState(), r.GetAvailabilityReason()),
				fmt.Sprintf("Goal: %d goal runs, %d met, avg %.0f ms / %.0f tokens to first met, %d regressions (%s)", g.GetGoalRuns(), g.GetMetRuns(), g.GetAverageTimeToFirstMetMs(), g.GetAverageTokensToFirstMet(), g.GetRegressions(), g.GetAvailabilityState()),
			}
			for _, s := range r.GetSignals() {
				lines = append(lines, fmt.Sprintf("- %s: count=%d impact=%d confidence=%s evidence=%s", s.GetKind(), s.GetCount(), s.GetImpact(), s.GetConfidence(), strings.Join(s.GetRepresentativeRunIds(), ",")))
			}
			return lines
		})
}
