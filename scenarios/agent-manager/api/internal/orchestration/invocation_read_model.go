// Invocation read-model orchestration owns replay, refresh, and corpus access
// without coupling projection consumers to the concrete database adapter.
package orchestration

import (
	"context"
	"fmt"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/repository"
	"agent-manager/internal/runreport"
	"github.com/google/uuid"
)

func (o *Orchestrator) EpisodeCohort(ctx context.Context, filter invocationreadmodel.Filter, limit int) (runreport.EpisodeCohort, error) {
	if o.invocationReadModel == nil {
		return runreport.EpisodeCohort{}, fmt.Errorf("invocation read model is not configured")
	}
	cohort, err := o.invocationReadModel.Cohort(ctx, filter, limit)
	if err != nil {
		return runreport.EpisodeCohort{}, err
	}
	selected := map[string][]runreport.FrictionEpisode{}
	for _, rawID := range cohort.RunIDs {
		id, err := uuid.Parse(rawID)
		if err != nil {
			return runreport.EpisodeCohort{}, err
		}
		episodes, err := o.Episodes(ctx, id)
		if err != nil {
			return runreport.EpisodeCohort{}, err
		}
		selected[rawID] = episodes
	}
	return runreport.BuildEpisodeCohort(selected), nil
}

type ReplayResult struct {
	RunID             string `json:"runId"`
	Status            string `json:"status"`
	FactCount         int    `json:"factCount"`
	EpisodeCount      int    `json:"episodeCount"`
	ClassifierVersion string `json:"classifierVersion"`
}

// ReplayFilter bounds bulk maintenance. It intentionally shares the existing
// run-list predicates until the corpus-query filter is introduced in phase 5.
type ReplayFilter struct {
	From      *time.Time
	To        *time.Time
	ProfileID *uuid.UUID
	Status    *domain.RunStatus
	TagPrefix string
	Limit     int
}

type ReplaySummary struct {
	Replayed          int  `json:"replayed"`
	Refreshed         int  `json:"refreshed"`
	Skipped           int  `json:"skipped"`
	Unreplayable      int  `json:"unreplayable"`
	EpisodesReDerived int  `json:"episodesReDerived"`
	Truncated         bool `json:"truncated"`
}

func (o *Orchestrator) AggregateInvocationFacts(ctx context.Context, filter invocationreadmodel.Filter, dimension string, limit int) ([]invocationreadmodel.AggregateRow, error) {
	if o.invocationReadModel == nil {
		return nil, fmt.Errorf("invocation read model is not configured")
	}
	return o.invocationReadModel.Aggregate(ctx, filter, dimension, limit)
}

func (o *Orchestrator) SelectInvocationCohort(ctx context.Context, filter invocationreadmodel.Filter, limit int) (invocationreadmodel.Cohort, error) {
	if o.invocationReadModel == nil {
		return invocationreadmodel.Cohort{}, fmt.Errorf("invocation read model is not configured")
	}
	return o.invocationReadModel.Cohort(ctx, filter, limit)
}

func (o *Orchestrator) InvocationMetrics(ctx context.Context, filter invocationreadmodel.Filter) (invocationreadmodel.Metrics, error) {
	if o.invocationReadModel == nil {
		return invocationreadmodel.Metrics{}, fmt.Errorf("invocation read model is not configured")
	}
	return o.invocationReadModel.Metrics(ctx, filter)
}

// ReplayInvocationFacts rebuilds one run from retained source events. A run
// with an existing watermark but no retained events is explicitly
// unreplayable: its existing facts and classifier version are left untouched.
func (o *Orchestrator) ReplayInvocationFacts(ctx context.Context, runID uuid.UUID) (*ReplayResult, error) {
	if o.invocationReadModel == nil {
		return nil, fmt.Errorf("invocation read model is not configured")
	}
	run, err := o.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	watermark, err := o.invocationReadModel.Watermark(ctx, runID.String())
	if err != nil {
		return nil, err
	}
	events, err := o.GetRunEvents(ctx, runID, event.GetOptions{AfterSequence: -1, Limit: 10000})
	if err != nil {
		return nil, fmt.Errorf("load retained run events: %w", err)
	}
	if len(events) == 0 && watermark != nil {
		return &ReplayResult{RunID: runID.String(), Status: "unreplayable", ClassifierVersion: watermark.ClassifierVersion}, nil
	}
	projectedAt := o.now()
	facts, next := invocationreadmodel.Project(run, events, projectedAt)
	if projection, ok := o.invocationReadModel.(invocationreadmodel.ProjectionStore); ok {
		if err := projection.ReplaceProjection(ctx, facts, invocationreadmodel.ProjectErrors(run, events, projectedAt), next, invocationreadmodel.ProjectRun(run, events, projectedAt)); err != nil {
			return nil, err
		}
	} else if err := o.invocationReadModel.Replace(ctx, facts, next); err != nil {
		return nil, err
	}
	report, err := o.BuildRunReport(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("re-derive episodes: %w", err)
	}
	return &ReplayResult{RunID: runID.String(), Status: "replayed", FactCount: len(facts), EpisodeCount: len(report.Episodes), ClassifierVersion: next.ClassifierVersion}, nil
}

// RefreshInvocationFacts avoids a write unless source events advanced beyond
// the watermark. When they did, the fold is rebuilt from retained history so
// call/result pairing and retry linkage remain deterministic at boundaries.
func (o *Orchestrator) RefreshInvocationFacts(ctx context.Context, runID uuid.UUID) (*ReplayResult, error) {
	if o.invocationReadModel == nil {
		return nil, fmt.Errorf("invocation read model is not configured")
	}
	watermark, err := o.invocationReadModel.Watermark(ctx, runID.String())
	if err != nil {
		return nil, err
	}
	if watermark == nil {
		return o.ReplayInvocationFacts(ctx, runID)
	}
	since := watermark.LastEventAt
	newer, err := o.GetRunEvents(ctx, runID, event.GetOptions{AfterSequence: -1, Since: &since, Limit: 1})
	if err != nil {
		return nil, fmt.Errorf("load events after watermark: %w", err)
	}
	if len(newer) == 0 {
		return &ReplayResult{RunID: runID.String(), Status: "skipped", ClassifierVersion: watermark.ClassifierVersion}, nil
	}
	result, err := o.ReplayInvocationFacts(ctx, runID)
	if err == nil && result.Status == "replayed" {
		result.Status = "refreshed"
	}
	return result, err
}

// ReplayInvocationCorpus walks the bounded matching run population. Individual
// unreplayable runs are accounted for, not treated as a failed maintenance job.
func (o *Orchestrator) ReplayInvocationCorpus(ctx context.Context, filter ReplayFilter, refresh bool) (*ReplaySummary, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 1000
	}
	runs, err := o.runs.List(ctx, repository.RunListFilter{ListFilter: repository.ListFilter{Limit: limit + 1}, AgentProfileID: filter.ProfileID, Status: filter.Status, TagPrefix: filter.TagPrefix})
	if err != nil {
		return nil, err
	}
	summary := &ReplaySummary{}
	if len(runs) > limit {
		summary.Truncated = true
		runs = runs[:limit]
	}
	for _, run := range runs {
		if run == nil || (filter.From != nil && (run.EndedAt == nil || run.EndedAt.Before(*filter.From))) || (filter.To != nil && (run.EndedAt == nil || !run.EndedAt.Before(*filter.To))) {
			continue
		}
		var result *ReplayResult
		if refresh {
			result, err = o.RefreshInvocationFacts(ctx, run.ID)
		} else {
			result, err = o.ReplayInvocationFacts(ctx, run.ID)
		}
		if err != nil {
			return nil, fmt.Errorf("%s run %s: %w", map[bool]string{true: "refresh", false: "replay"}[refresh], run.ID, err)
		}
		// Refresh deliberately preserves the no-new-events skip invariant for the
		// invocation-fact projection. Episodes are a separately versioned derived
		// projection, though, so refresh must still re-derive them from the durable
		// corpus even when that fact projection is already current.
		if refresh && result.Status == "skipped" {
			report, reportErr := o.BuildRunReport(ctx, run.ID)
			if reportErr != nil {
				return nil, fmt.Errorf("refresh episodes for run %s: %w", run.ID, reportErr)
			}
			result.EpisodeCount = len(report.Episodes)
		}
		switch result.Status {
		case "replayed":
			summary.Replayed++
		case "refreshed":
			summary.Refreshed++
		case "skipped":
			summary.Skipped++
		case "unreplayable":
			summary.Unreplayable++
		}
		summary.EpisodesReDerived += result.EpisodeCount
	}
	return summary, nil
}

// projectInvocationReadModel is intentionally best-effort: analytics capture
// can never turn a completed run into a failed one. Replace keeps facts and
// watermark in one transaction, making retries safe after a process crash.
func (o *Orchestrator) projectInvocationReadModel(ctx context.Context, run *domain.Run) error {
	if o.invocationReadModel == nil || o.events == nil || run == nil {
		return nil
	}
	events, err := o.GetRunEvents(ctx, run.ID, event.GetOptions{AfterSequence: -1, Limit: 10000})
	if err != nil {
		return fmt.Errorf("load run events: %w", err)
	}
	projectedAt := o.now()
	facts, watermark := invocationreadmodel.Project(run, events, projectedAt)
	if projection, ok := o.invocationReadModel.(invocationreadmodel.ProjectionStore); ok {
		return projection.ReplaceProjection(ctx, facts, invocationreadmodel.ProjectErrors(run, events, projectedAt), watermark, invocationreadmodel.ProjectRun(run, events, projectedAt))
	}
	return o.invocationReadModel.Replace(ctx, facts, watermark)
}

func (o *Orchestrator) projectTerminalInvocationReadModel(run *domain.Run) {
	if run == nil || !run.Status.IsTerminal() || o.invocationReadModel == nil {
		return
	}
	runCopy := *run
	go func(id uuid.UUID) {
		if err := o.projectInvocationReadModel(context.Background(), &runCopy); err != nil {
			obs.Component("invocation-read-model").Warn("durable invocation projection failed", obs.KeyRunID, id.String(), obs.KeyError, err.Error())
		}
	}(run.ID)
}
