package measures

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/runreport"
	"agent-manager/internal/runsignal"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	sharedmeasurepb "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

const (
	// maxCohortRunIDs bounds explicit selections and run-report folds.
	maxCohortRunIDs = 100
	// episodeEvidenceLimit bounds one episode read; it is a safety ceiling, not
	// a sample size.
	episodeEvidenceLimit = 1_000_000
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

// episodes returns the selection's persisted friction episodes, each carrying
// its run id, and the number of runs in the population.
func (h *Handler) episodes(ctx context.Context, selection cohortSelection) ([]runsignal.FrictionEpisode, int, error) {
	projection, ok := h.store.(invocationreadmodel.ProjectionStore)
	if !ok {
		return nil, 0, fmt.Errorf("measure store does not provide persisted episodes")
	}
	var records []invocationreadmodel.Episode
	population := len(selection.runIDs)
	if selection.explicit() {
		for _, id := range selection.runIDs {
			rows, err := projection.EpisodesForRun(ctx, id)
			if err != nil {
				return nil, 0, err
			}
			records = append(records, rows...)
		}
	} else {
		rows, err := projection.Episodes(ctx, selection.filter, episodeEvidenceLimit)
		if err != nil {
			return nil, 0, err
		}
		records = rows
		cohort, err := h.runs(ctx, selection, 1)
		if err != nil {
			return nil, 0, err
		}
		population = cohort.MatchedRuns
	}
	out := make([]runsignal.FrictionEpisode, 0, len(records))
	for _, record := range records {
		episode := record.FrictionEpisode
		episode.RunID = record.RunID
		out = append(out, episode)
	}
	return out, population, nil
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

// EpisodeCohort ranks the recurring friction-episode fingerprints of one
// selected cohort. A predicate selection folds at most limit distinct runs.
func (h *Handler) EpisodeCohort(ctx context.Context, req *connect.Request[measurepb.EpisodeCohortRequest]) (*connect.Response[measurepb.EpisodeCohortResponse], error) {
	selection, err := h.resolveCohort(ctx, req.Msg.GetWindow(), req.Msg.GetFilter(), req.Msg.GetCohort(), req.Msg.GetRunIds())
	if err != nil {
		return nil, err
	}
	limit := boundedLimit(req.Msg.GetLimit(), 25, 0)
	episodes, _, err := h.episodes(ctx, selection)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	byRun := map[string][]runsignal.FrictionEpisode{}
	for _, id := range selection.runIDs {
		byRun[id] = nil
	}
	for _, episode := range episodes {
		if _, kept := byRun[episode.RunID]; !kept && len(byRun) >= limit {
			continue
		}
		byRun[episode.RunID] = append(byRun[episode.RunID], episode)
	}
	cohort := runreport.BuildEpisodeCohort(byRun)
	response := &measurepb.EpisodeCohortResponse{AvailabilityState: string(cohort.Availability.State), AvailabilityReason: cohort.Availability.Reason}
	for _, signal := range cohort.Signals {
		response.Signals = append(response.Signals, &measurepb.EpisodeCohortSignal{Fingerprint: signal.Fingerprint, Occurrences: int64(signal.Occurrences), DistinctRuns: int64(signal.DistinctRuns), SummedCostMs: signal.SummedCostMS, Confidence: signal.Confidence, RepresentativeRunIds: append([]string(nil), signal.RepresentativeRunIDs...)})
	}
	response.Validity = h.validityForSample(int64(len(cohort.Signals)))
	response.Provenance = h.cohortProvenance(ctx, selection, "episode_cohort", int64(len(cohort.Signals)), fmt.Sprintf("SELECT persisted friction episodes FROM selected cohort LIMIT %d runs", limit))
	response.DefinitionId = definitionID("episode_cohort")
	return connect.NewResponse(response), nil
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
			AverageTimeToFirstMetMs: goal.AverageTimeToFirstMetMS, AverageTokensToFirstMet: goal.AverageTokensToFirstMet, InterventionsBeforeMet: int64(goal.InterventionsBeforeMet), HandoffsBeforeMet: int64(goal.HandoffsBeforeMet), Regressions: int64(goal.Regressions), Restatements: int64(goal.Restatements),
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

// CompareEpisodeCohorts compares episode fingerprints between two selected
// cohorts; the right cohort is the "after" side of every delta.
func (h *Handler) CompareEpisodeCohorts(ctx context.Context, req *connect.Request[measurepb.CompareEpisodeCohortsRequest]) (*connect.Response[measurepb.CompareEpisodeCohortsResponse], error) {
	sides := [2]*measurepb.CohortSelector{req.Msg.GetLeft(), req.Msg.GetRight()}
	var episodes [2][]runsignal.FrictionEpisode
	var populations [2]int
	var selections [2]cohortSelection
	for i, side := range sides {
		selection, err := h.resolveCohort(ctx, side.GetWindow(), side.GetFilter(), side.GetCohort(), side.GetRunIds())
		if err != nil {
			return nil, err
		}
		selections[i] = selection
		episodes[i], populations[i], err = h.episodes(ctx, selection)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	comparison := runreport.BuildCohortComparison(episodes[0], episodes[1], populations[0], populations[1], boundedLimit(req.Msg.GetLimit(), 100, 0))
	response := &measurepb.CompareEpisodeCohortsResponse{
		ClassifierVersion: comparison.ClassifierVersion, ChangeBinding: req.Msg.GetChangeBinding(), LeftPopulation: int64(comparison.LeftPopulation), RightPopulation: int64(comparison.RightPopulation),
		AvailabilityState: string(comparison.Availability.State), AvailabilityReason: comparison.Availability.Reason,
	}
	for _, signal := range comparison.Signals {
		response.Signals = append(response.Signals, &measurepb.EpisodeComparisonSignal{
			Fingerprint: signal.Fingerprint, LeftOccurrences: int64(signal.LeftOccurrences), RightOccurrences: int64(signal.RightOccurrences), LeftDistinctRuns: int64(signal.LeftDistinctRuns), RightDistinctRuns: int64(signal.RightDistinctRuns),
			LeftCostMs: signal.LeftCostMS, RightCostMs: signal.RightCostMS, OccurrenceDelta: int64(signal.OccurrenceDelta), CostDeltaMs: signal.CostDeltaMS,
		})
	}
	response.Validity = h.validityForSample(int64(min(populations[0], populations[1])))
	response.Provenance = h.cohortProvenance(ctx, selections[1], "episode_comparison", int64(len(episodes[0])+len(episodes[1])), "SELECT persisted friction episodes FROM left and right cohorts")
	response.DefinitionId = definitionID("episode_comparison")
	return connect.NewResponse(response), nil
}
