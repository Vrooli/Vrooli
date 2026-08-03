// Invocation read-model orchestration owns replay, refresh, and corpus access
// without coupling projection consumers to the concrete database adapter.
package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/repository"
	"agent-manager/internal/runreport"
	"agent-manager/internal/runsignal"
	"github.com/google/uuid"
)

const invocationEvidenceLimit = 1_000_000

func (o *Orchestrator) EpisodeCohort(ctx context.Context, filter invocationreadmodel.Filter, limit int) (runreport.EpisodeCohort, error) {
	if o.invocationReadModel == nil {
		return runreport.EpisodeCohort{}, fmt.Errorf("invocation read model is not configured")
	}
	selected := map[string][]runsignal.FrictionEpisode{}
	if projection, ok := o.invocationReadModel.(invocationreadmodel.ProjectionStore); ok {
		episodes, err := projection.Episodes(ctx, filter, invocationEvidenceLimit)
		if err != nil {
			return runreport.EpisodeCohort{}, err
		}
		for _, episode := range episodes {
			if limit > 0 && len(selected) >= limit {
				if _, exists := selected[episode.RunID]; !exists {
					continue
				}
			}
			selected[episode.RunID] = append(selected[episode.RunID], episode.FrictionEpisode)
		}
	} else {
		cohort, err := o.invocationReadModel.Cohort(ctx, filter, limit)
		if err != nil {
			return runreport.EpisodeCohort{}, err
		}
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
	}
	return runreport.BuildEpisodeCohort(selected), nil
}

func (o *Orchestrator) CompareEpisodeCohorts(ctx context.Context, left, right invocationreadmodel.Filter, limit int) (runreport.CohortComparison, error) {
	projection, ok := o.invocationReadModel.(invocationreadmodel.ProjectionStore)
	if !ok {
		return runreport.CohortComparison{}, fmt.Errorf("episode comparison requires a projection store")
	}
	leftCohort, err := o.invocationReadModel.Cohort(ctx, left, invocationEvidenceLimit)
	if err != nil {
		return runreport.CohortComparison{}, err
	}
	rightCohort, err := o.invocationReadModel.Cohort(ctx, right, invocationEvidenceLimit)
	if err != nil {
		return runreport.CohortComparison{}, err
	}
	leftEpisodes, err := projection.Episodes(ctx, left, invocationEvidenceLimit)
	if err != nil {
		return runreport.CohortComparison{}, err
	}
	rightEpisodes, err := projection.Episodes(ctx, right, invocationEvidenceLimit)
	if err != nil {
		return runreport.CohortComparison{}, err
	}
	leftFacts := make([]runsignal.FrictionEpisode, 0, len(leftEpisodes))
	for _, episode := range leftEpisodes {
		fact := episode.FrictionEpisode
		fact.RunID = episode.RunID
		leftFacts = append(leftFacts, fact)
	}
	rightFacts := make([]runsignal.FrictionEpisode, 0, len(rightEpisodes))
	for _, episode := range rightEpisodes {
		fact := episode.FrictionEpisode
		fact.RunID = episode.RunID
		rightFacts = append(rightFacts, fact)
	}
	return runreport.BuildCohortComparison(leftFacts, rightFacts, leftCohort.MatchedRuns, rightCohort.MatchedRuns, limit), nil
}

func (o *Orchestrator) EpisodeTrend(ctx context.Context, filter invocationreadmodel.Filter, bucket time.Duration, limit int) ([]runreport.EpisodeTrendBucket, error) {
	projection, ok := o.invocationReadModel.(invocationreadmodel.ProjectionStore)
	if !ok {
		return nil, fmt.Errorf("episode trend requires a projection store")
	}
	episodes, err := projection.Episodes(ctx, filter, invocationEvidenceLimit)
	if err != nil {
		return nil, err
	}
	timed := make([]runreport.TimedEpisode, 0, len(episodes))
	for _, episode := range episodes {
		fact := episode.FrictionEpisode
		fact.RunID = episode.RunID
		timed = append(timed, runreport.TimedEpisode{Episode: fact, OccurredAt: episode.OccurredAt})
	}
	result := runreport.BuildEpisodeTrend(timed, bucket)
	if limit > 0 && len(result) > limit {
		result = result[len(result)-limit:]
	}
	return result, nil
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
	Replayed          int             `json:"replayed"`
	Refreshed         int             `json:"refreshed"`
	Skipped           int             `json:"skipped"`
	Unreplayable      int             `json:"unreplayable"`
	EpisodesReDerived int             `json:"episodesReDerived"`
	Truncated         bool            `json:"truncated"`
	Failures          []ReplayFailure `json:"failures,omitempty"`
}

// ReplayFailure makes a per-run maintenance failure inspectable without
// turning an otherwise useful corpus rebuild into an all-or-nothing request.
// It deliberately holds only the run identity and bounded error class.
type ReplayFailure struct {
	RunID string `json:"runId"`
	Error string `json:"error"`
}

const replayFailureLimit = 20

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

func (o *Orchestrator) cohortDefinitions() (invocationreadmodel.CohortDefinitionStore, error) {
	store, ok := o.invocationReadModel.(invocationreadmodel.CohortDefinitionStore)
	if !ok {
		return nil, fmt.Errorf("cohort definitions are not configured")
	}
	return store, nil
}

func (o *Orchestrator) DefineCohort(ctx context.Context, name, filterJSON, changeBinding string) (invocationreadmodel.CohortDefinition, error) {
	store, err := o.cohortDefinitions()
	if err != nil {
		return invocationreadmodel.CohortDefinition{}, err
	}
	var filter invocationreadmodel.Filter
	if err := json.Unmarshal([]byte(filterJSON), &filter); err != nil {
		return invocationreadmodel.CohortDefinition{}, fmt.Errorf("invalid cohort filter: %w", err)
	}
	_ = filter
	definition := invocationreadmodel.CohortDefinition{Name: name, FilterJSON: filterJSON, ClassifierVersion: runsignal.InvocationFactVersion, CreatedAt: o.now(), ChangeBinding: changeBinding}
	if err := store.DefineCohort(ctx, definition); err != nil {
		return invocationreadmodel.CohortDefinition{}, err
	}
	return definition, nil
}

func (o *Orchestrator) ListCohorts(ctx context.Context) ([]invocationreadmodel.CohortDefinition, error) {
	store, err := o.cohortDefinitions()
	if err != nil {
		return nil, err
	}
	return store.ListCohorts(ctx)
}

func (o *Orchestrator) ShowCohort(ctx context.Context, name string, limit int) (invocationreadmodel.CohortDefinition, invocationreadmodel.Cohort, error) {
	store, err := o.cohortDefinitions()
	if err != nil {
		return invocationreadmodel.CohortDefinition{}, invocationreadmodel.Cohort{}, err
	}
	definition, err := store.GetCohortDefinition(ctx, name)
	if err != nil {
		return invocationreadmodel.CohortDefinition{}, invocationreadmodel.Cohort{}, err
	}
	if definition == nil {
		return invocationreadmodel.CohortDefinition{}, invocationreadmodel.Cohort{}, fmt.Errorf("cohort %q not found", name)
	}
	var filter invocationreadmodel.Filter
	if err := json.Unmarshal([]byte(definition.FilterJSON), &filter); err != nil {
		return *definition, invocationreadmodel.Cohort{}, err
	}
	cohort, err := o.invocationReadModel.Cohort(ctx, filter, limit)
	if err != nil {
		return *definition, invocationreadmodel.Cohort{}, err
	}
	versions := map[string]bool{}
	for _, id := range cohort.RunIDs {
		facts, factErr := o.invocationReadModel.Facts(ctx, id)
		if factErr != nil {
			return *definition, invocationreadmodel.Cohort{}, factErr
		}
		for _, fact := range facts {
			if fact.Version != "" {
				versions[fact.Version] = true
			}
		}
	}
	if len(versions) > 1 || (len(versions) == 1 && !versions[definition.ClassifierVersion]) {
		return *definition, invocationreadmodel.Cohort{}, fmt.Errorf("cohort %q spans classifier versions %v; expected %s", name, versions, definition.ClassifierVersion)
	}
	return *definition, cohort, nil
}

func (o *Orchestrator) DeleteCohort(ctx context.Context, name string) error {
	store, err := o.cohortDefinitions()
	if err != nil {
		return err
	}
	return store.DeleteCohort(ctx, name)
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
	if len(events) == 0 {
		if run != nil && run.ExecutionMode.Normalized() == domain.ExecutionModeImported && strings.TrimSpace(run.TranscriptPath) != "" {
			if err := o.rehydrateImportedTranscript(ctx, run); err == nil {
				events, err = o.GetRunEvents(ctx, runID, event.GetOptions{AfterSequence: -1, Limit: 10000})
				if err != nil {
					return nil, fmt.Errorf("load rehydrated run events: %w", err)
				}
			}
		}
	}
	if len(events) == 0 {
		// A projection watermark requires a retained source event timestamp. A
		// transcript can legitimately carry only session metadata, so represent
		// that evidence as unreplayable rather than attempting an invalid empty
		// watermark write. Existing facts remain untouched when one exists.
		version := "unprojected"
		if watermark != nil {
			version = watermark.ClassifierVersion
		}
		return &ReplayResult{RunID: runID.String(), Status: "unreplayable", ClassifierVersion: version}, nil
	}
	projectedAt := o.now()
	facts, next := invocationreadmodel.ProjectWithResolvers(run, events, projectedAt, o.capabilityResolver(run), o.commandResolver(run))
	if projection, ok := o.invocationReadModel.(invocationreadmodel.ProjectionStore); ok {
		episodes := invocationreadmodel.ProjectEpisodes(run, facts, events, projectedAt)
		spans := invocationreadmodel.ProjectSelfReportSpans(run, events, projectedAt)
		if err := projection.ReplaceProjection(ctx, facts, invocationreadmodel.ProjectErrors(run, events, projectedAt), episodes, spans, next, invocationreadmodel.ProjectRun(run, events, projectedAt)); err != nil {
			return nil, err
		}
		return &ReplayResult{RunID: runID.String(), Status: "replayed", FactCount: len(facts), EpisodeCount: len(episodes), ClassifierVersion: next.ClassifierVersion}, nil
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
	// A refresh also owns classifier-version migration. A run with no newer
	// source event still needs a full projection when its durable watermark is
	// older than the current classifier; otherwise mixed-version cohorts can
	// never converge without an ad hoc database operation.
	if watermark.ClassifierVersion != runsignal.InvocationFactVersion ||
		watermark.EpisodeClassifierVersion != runsignal.EpisodeClassifierVersion ||
		watermark.SelfReportClassifierVersion != runsignal.SelfReportClassifierVersion {
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
	runs, err := o.runs.List(ctx, repository.RunListFilter{ListFilter: repository.ListFilter{Limit: limit + 1}, AgentProfileID: filter.ProfileID, Status: filter.Status, TagPrefix: filter.TagPrefix, EndedFrom: filter.From, EndedTo: filter.To})
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
		if workload, ok := domain.WorkloadFromHistoricalTag(run.Tag); ok {
			if backfill, supportsBackfill := o.invocationReadModel.(invocationreadmodel.WorkloadStore); supportsBackfill {
				if err := backfill.BackfillWorkload(ctx, run.ID.String(), string(workload.Kind), workload.Key, workload.Instance); err != nil {
					return nil, err
				}
			}
		}
		var result *ReplayResult
		if refresh {
			result, err = o.RefreshInvocationFacts(ctx, run.ID)
		} else {
			result, err = o.ReplayInvocationFacts(ctx, run.ID)
		}
		if err != nil {
			summary.Unreplayable++
			if len(summary.Failures) < replayFailureLimit {
				summary.Failures = append(summary.Failures, ReplayFailure{RunID: run.ID.String(), Error: "projection_failed"})
			}
			continue
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
	capabilityResolver := o.capabilityResolver(run)
	facts, watermark := invocationreadmodel.ProjectWithResolvers(run, events, projectedAt, capabilityResolver, o.commandResolver(run))
	if projection, ok := o.invocationReadModel.(invocationreadmodel.ProjectionStore); ok {
		return projection.ReplaceProjection(ctx, facts, invocationreadmodel.ProjectErrors(run, events, projectedAt), invocationreadmodel.ProjectEpisodes(run, facts, events, projectedAt), invocationreadmodel.ProjectSelfReportSpans(run, events, projectedAt), watermark, invocationreadmodel.ProjectRun(run, events, projectedAt, capabilityResolver))
	}
	return o.invocationReadModel.Replace(ctx, facts, watermark)
}

func (o *Orchestrator) capabilityResolver(run *domain.Run) runsignal.CapabilityResolver {
	if o == nil || o.runners == nil || run == nil || run.ResolvedConfig == nil {
		return nil
	}
	owned, err := o.runners.Get(run.ResolvedConfig.RunnerType)
	if err != nil {
		return nil
	}
	provider, ok := owned.(interface{ ToolCapabilityMap() map[string]string })
	if !ok {
		return nil
	}
	declared := provider.ToolCapabilityMap()
	return func(tool string) string {
		if value := declared[tool]; value != "" {
			return value
		}
		return "other"
	}
}

func (o *Orchestrator) commandResolver(run *domain.Run) runsignal.CommandResolver {
	if o == nil || o.runners == nil || run == nil || run.ResolvedConfig == nil {
		return nil
	}
	owned, err := o.runners.Get(run.ResolvedConfig.RunnerType)
	if err != nil {
		return nil
	}
	provider, ok := owned.(runner.CommandExtractor)
	if !ok {
		return nil
	}
	return func(_ string, input map[string]any) runner.CommandExtraction {
		return provider.ExtractCommand(input)
	}
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
