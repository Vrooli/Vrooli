package measures

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/runreport"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	sharedmeasurepb "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

const (
	// maxCohortRunIDs bounds explicit selections and run-report folds.
	maxCohortRunIDs = 100
)

// CohortSource supplies what the read model alone cannot: named cohort
// resolution (with its classifier-version guard) and per-run reports.
// Orchestration implements it.
type CohortSource interface {
	ShowCohort(context.Context, string, int) (invocationreadmodel.CohortDefinition, invocationreadmodel.Cohort, error)
	BuildRunReport(context.Context, uuid.UUID) (*runreport.RunReport, error)
}

// SetCohortSource connects named cohorts and run reports.
func (h *Handler) SetCohortSource(source CohortSource) { h.cohorts = source }

// cohortSelection is a resolved CohortSelector: either explicit runs or a
// durable predicate.
type cohortSelection struct {
	filter invocationreadmodel.Filter
	runIDs []string
	name   string
}

func (s cohortSelection) explicit() bool { return s.runIDs != nil }

// resolveCohort applies the one selector rule shared by every cohort request
// (see CohortSelector in measures.proto).
func (h *Handler) resolveCohort(ctx context.Context, window *sharedmeasurepb.TimeWindow, filter *measurepb.InvocationFilter, name string, runIDs []string) (cohortSelection, error) {
	name = strings.TrimSpace(name)
	hasFilter := filter != nil && !proto.Equal(filter, &measurepb.InvocationFilter{})
	set := 0
	for _, present := range []bool{name != "", len(runIDs) > 0, hasFilter} {
		if present {
			set++
		}
	}
	if set > 1 {
		return cohortSelection{}, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("select a cohort with one of cohort, run_ids or filter"))
	}
	switch {
	case len(runIDs) > 0:
		ids, err := normalizeRunIDs(runIDs)
		if err != nil {
			return cohortSelection{}, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return cohortSelection{runIDs: ids}, nil
	case name != "":
		if h.cohorts == nil {
			return cohortSelection{}, connect.NewError(connect.CodeInternal, fmt.Errorf("named cohorts are not configured"))
		}
		definition, _, err := h.cohorts.ShowCohort(ctx, name, 1)
		if err != nil {
			return cohortSelection{}, connect.NewError(connect.CodeInvalidArgument, err)
		}
		selection := cohortSelection{name: name}
		if err := json.Unmarshal([]byte(definition.FilterJSON), &selection.filter); err != nil {
			return cohortSelection{}, connect.NewError(connect.CodeInternal, fmt.Errorf("decode cohort %q: %w", name, err))
		}
		if window != nil {
			bounded, err := filterWithWindow(nil, window, h.now())
			if err != nil {
				return cohortSelection{}, connect.NewError(connect.CodeInvalidArgument, err)
			}
			selection.filter.From, selection.filter.To = bounded.From, bounded.To
		}
		return selection, nil
	default:
		resolved, err := filterWithWindow(filter, window, h.now())
		if err != nil {
			return cohortSelection{}, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return cohortSelection{filter: resolved}, nil
	}
}

func normalizeRunIDs(raw []string) ([]string, error) {
	seen := map[string]bool{}
	ids := make([]string, 0, len(raw))
	for _, value := range raw {
		id, err := uuid.Parse(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("run id %q is not a UUID", value)
		}
		if !seen[id.String()] {
			seen[id.String()] = true
			ids = append(ids, id.String())
		}
	}
	if len(ids) > maxCohortRunIDs {
		return nil, fmt.Errorf("select at most %d run ids", maxCohortRunIDs)
	}
	return ids, nil
}

// runs returns the selection's run ids, bounded by limit for predicates.
func (h *Handler) runs(ctx context.Context, selection cohortSelection, limit int) (invocationreadmodel.Cohort, error) {
	if selection.explicit() {
		return invocationreadmodel.Cohort{RunIDs: selection.runIDs, MatchedRuns: len(selection.runIDs)}, nil
	}
	cohortStore, ok := h.store.(interface {
		Cohort(context.Context, invocationreadmodel.Filter, int) (invocationreadmodel.Cohort, error)
	})
	if !ok {
		return invocationreadmodel.Cohort{}, fmt.Errorf("measure store does not provide durable cohort selection")
	}
	return cohortStore.Cohort(ctx, selection.filter, limit)
}

func (h *Handler) cohortProvenance(ctx context.Context, selection cohortSelection, definition string, rows int64, query string) *measurepb.MeasureProvenance {
	provenance := h.provenanceWithQuery(ctx, selection.filter, definitionFor(definition).SourceTable, rows, query)
	if selection.name != "" {
		provenance.AppliedFilters = append(provenance.AppliedFilters, &measurepb.MeasureFilter{Field: "cohort", Value: selection.name})
	}
	if selection.explicit() {
		provenance.AppliedFilters = append(provenance.AppliedFilters, &measurepb.MeasureFilter{Field: "run_ids", Value: strings.Join(selection.runIDs, ",")})
	}
	return provenance
}

func boundedLimit(value int32, fallback, ceiling int) int {
	limit := int(value)
	if limit <= 0 {
		limit = fallback
	}
	if ceiling > 0 && limit > ceiling {
		limit = ceiling
	}
	return limit
}

// SelectCohort lists the runs behind a measure. With the filter it shares
// every measure's predicate, so the listed runs are the population a value
// describes.
func (h *Handler) SelectCohort(ctx context.Context, req *connect.Request[measurepb.SelectCohortRequest]) (*connect.Response[measurepb.SelectCohortResponse], error) {
	selection, err := h.resolveCohort(ctx, req.Msg.GetWindow(), req.Msg.GetFilter(), req.Msg.GetCohort(), req.Msg.GetRunIds())
	if err != nil {
		return nil, err
	}
	limit := boundedLimit(req.Msg.GetLimit(), 25, 0)
	cohort, err := h.runs(ctx, selection, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	rows := make([]*measurepb.CohortRun, 0, len(cohort.RunIDs))
	if rowStore, ok := h.store.(interface {
		CohortRows(context.Context, []string, string) ([]invocationreadmodel.CohortRun, error)
	}); ok {
		cohortRows, rowErr := rowStore.CohortRows(ctx, cohort.RunIDs, selection.filter.ToolName)
		if rowErr != nil {
			return nil, connect.NewError(connect.CodeInternal, rowErr)
		}
		for _, row := range cohortRows {
			protoRow := &measurepb.CohortRun{
				RunId: row.RunID, TaskTitle: row.TaskTitle, ProfileId: row.ProfileID, ProfileName: row.ProfileName, Status: row.Status, CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339), Model: row.Model, RunnerType: row.RunnerType, WorkloadKey: row.WorkloadKey, TotalTokens: row.TotalTokens,
				TotalChargeMicroUsd: row.TotalChargeMicroUSD, ChargeBasis: row.ChargeBasis, ToolCallCount: row.ToolCallCount,
			}
			rows = append(rows, protoRow)
		}
	}
	query := fmt.Sprintf("SELECT DISTINCT run_id FROM invocation_read_model_facts WHERE <selected cohort> ORDER BY run_id LIMIT %d", limit)
	return connect.NewResponse(&measurepb.SelectCohortResponse{RunIds: cohort.RunIDs, Rows: rows, Truncated: cohort.Truncated, Validity: h.validityForSample(int64(len(cohort.RunIDs))), Provenance: h.cohortProvenance(ctx, selection, "select_cohort", int64(len(cohort.RunIDs)), query), DefinitionId: definitionID("select_cohort")}), nil
}

// CohortReport folds the run reports of one selected cohort into ranked
// friction signals, summed time accounting, evidence validity and goal
// progression.
func (h *Handler) CohortReport(ctx context.Context, req *connect.Request[measurepb.CohortReportRequest]) (*connect.Response[measurepb.CohortReportResponse], error) {
	if h.cohorts == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("run reports are not configured"))
	}
	selection, err := h.resolveCohort(ctx, req.Msg.GetWindow(), req.Msg.GetFilter(), req.Msg.GetCohort(), req.Msg.GetRunIds())
	if err != nil {
		return nil, err
	}
	cohort, err := h.runs(ctx, selection, boundedLimit(req.Msg.GetLimit(), maxCohortRunIDs, maxCohortRunIDs))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	reports := make([]*runreport.RunReport, 0, len(cohort.RunIDs))
	for _, raw := range cohort.RunIDs {
		id, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			return nil, connect.NewError(connect.CodeInternal, parseErr)
		}
		report, reportErr := h.cohorts.BuildRunReport(ctx, id)
		if reportErr != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("run report %s: %w", raw, reportErr))
		}
		reports = append(reports, report)
	}
	folded, goal := runreport.BuildCohort(reports), runreport.BuildGoalCohort(reports)
	spent := folded.TimeAccounting
	response := &measurepb.CohortReportResponse{
		ClassifierVersion: folded.ClassifierVersion, RunIds: folded.RunIDs, Truncated: cohort.Truncated,
		AvailabilityState: string(folded.Availability.State), AvailabilityReason: folded.Availability.Reason,
		TimeAccounting: &measurepb.CohortTimeAccounting{
			ModelGeneratingMs: spent.ModelGeneratingMS, ToolExecutingMs: spent.ToolExecutingMS, IdleWaitingMs: spent.IdleWaitingMS, AwaitingHumanMs: spent.AwaitingHumanMS, UnattributableMs: spent.UnattributableMS,
			ModelTokens: spent.ModelTokens, ToolTokens: spent.ToolTokens, IdleTokens: spent.IdleTokens, HumanTokens: spent.HumanTokens, UnattributableTokens: spent.UnattributableTokens,
		},
		InvocationValidity: &measurepb.CohortInvocationValidity{TotalCalls: int64(folded.InvocationValidity.TotalCalls), ClassifiedBase: int64(folded.InvocationValidity.ClassifiedBase), UnclassifiedCount: int64(folded.InvocationValidity.UnclassifiedCount)},
		Goal: &measurepb.CohortGoalProgression{
			AvailabilityState: string(goal.Availability.State), AvailabilityReason: goal.Availability.Reason, GoalRuns: int64(goal.GoalRuns), MetRuns: int64(goal.MetRuns),
			AverageTimeToFirstMetMs: goal.AverageTimeToFirstMetMS, AverageTokensToFirstMet: goal.AverageTokensToFirstMet, InterventionsBeforeMet: int64(goal.InterventionsBeforeMet), Regressions: int64(goal.Regressions), Restatements: int64(goal.Restatements),
		},
	}
	if folded.RunIDs == nil {
		response.RunIds = []string{}
	}
	for _, signal := range folded.Signals {
		response.Signals = append(response.Signals, &measurepb.CohortReportSignal{Kind: signal.Kind, Count: int64(signal.Count), Impact: int64(signal.Impact), Confidence: signal.Confidence, RepresentativeRunIds: append([]string(nil), signal.RepresentativeRunIDs...)})
	}
	response.Validity = h.validityForSample(int64(len(reports)))
	response.Provenance = h.cohortProvenance(ctx, selection, "cohort_report", int64(len(reports)), "BUILD run reports FOR selected cohort")
	response.DefinitionId = definitionID("cohort_report")
	return connect.NewResponse(response), nil
}
