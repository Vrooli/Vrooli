// Invocation read-model orchestration owns replay, refresh, and corpus access
// without coupling projection consumers to the concrete database adapter.
package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// projectionEventPage bounds one database round trip while a run's retained
// event stream is loaded. It is a page size, never a cap on the run.
const projectionEventPage = 10_000

// projectionEventCeiling stops an unbounded read if a run's retained stream is
// corrupt or pathological. Reaching it is an error rather than a truncation:
// the failure this guards against is a projection that silently describes part
// of a run as though it were the whole run.
const projectionEventCeiling = 5_000_000

// allRunEvents loads every retained event for a run, in sequence order.
//
// Projection must see the whole stream. A single capped Get drops the tail of
// a long run, and because the run total is derived from the same truncated
// slice, the conservation ledger still reconciles — against a partial universe.
// That makes the longest, most expensive runs the least accurately measured,
// which is precisely backwards for a prioritization signal. Paging by sequence
// keeps each round trip bounded without bounding the run.
func (o *Orchestrator) allRunEvents(ctx context.Context, runID uuid.UUID, opts event.GetOptions) ([]*domain.RunEvent, error) {
	opts.Limit = projectionEventPage
	opts.Offset = 0
	after := opts.AfterSequence
	var all []*domain.RunEvent
	for {
		opts.AfterSequence = after
		batch, err := o.GetRunEvents(ctx, runID, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < projectionEventPage {
			return all, nil
		}
		next := batch[len(batch)-1].Sequence
		if next <= after {
			// Retained sequences are monotonic per run, so a cursor that fails
			// to advance is a corrupt stream rather than a long one. Returning
			// here prevents an infinite read.
			return nil, fmt.Errorf("run %s event sequence did not advance past %d", runID, after)
		}
		after = next
		if len(all) >= projectionEventCeiling {
			return nil, fmt.Errorf("run %s exceeded the %d retained event ceiling", runID, projectionEventCeiling)
		}
	}
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
	// RunID selects one run; the other fields are then ignored.
	RunID     *uuid.UUID
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
	events, err := o.allRunEvents(ctx, runID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		return nil, fmt.Errorf("load retained run events: %w", err)
	}
	if len(events) == 0 && run != nil && run.ExecutionMode.Normalized() == domain.ExecutionModeImported && o.transcriptIndex != nil {
		// An explicit replay is a recall request: rebuild the evicted events
		// from the source file the indexer read.
		if err := o.transcriptIndex.RebuildRecall(ctx, runID.String()); err == nil {
			events, err = o.allRunEvents(ctx, runID, event.GetOptions{AfterSequence: -1})
			if err != nil {
				return nil, fmt.Errorf("load rebuilt run events: %w", err)
			}
		}
	}
	events = o.summaryUsageEvents(ctx, run, events)
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
	if err := o.persistRunSubject(ctx, run, facts, events); err != nil {
		return nil, err
	}
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

func (o *Orchestrator) persistRunSubject(ctx context.Context, run *domain.Run, facts []invocationreadmodel.Fact, events []*domain.RunEvent) error {
	if o == nil || o.runs == nil || run == nil {
		return nil
	}
	updater, ok := o.runs.(repository.RunSubjectUpdater)
	if !ok {
		return nil
	}
	subject := invocationreadmodel.DeriveRunSubject(facts, events)
	if len(subject) == 0 {
		return nil
	}
	if err := updater.UpdateRunSubject(ctx, run.ID, subject); err != nil {
		return fmt.Errorf("persist run subject: %w", err)
	}
	run.Subject = subject
	return nil
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
	var runs []*domain.Run
	var err error
	if filter.RunID != nil {
		run, getErr := o.runs.Get(ctx, *filter.RunID)
		if getErr != nil {
			return nil, getErr
		}
		runs, filter.From, filter.To = []*domain.Run{run}, nil, nil
	} else if runs, err = o.runs.List(ctx, repository.RunListFilter{ListFilter: repository.ListFilter{Limit: limit + 1}, AgentProfileID: filter.ProfileID, Status: filter.Status, TagPrefix: filter.TagPrefix, EndedFrom: filter.From, EndedTo: filter.To}); err != nil {
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
	events, err := o.allRunEvents(ctx, run.ID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		return fmt.Errorf("load run events: %w", err)
	}
	events = o.summaryUsageEvents(ctx, run, events)
	projectedAt := o.now()
	capabilityResolver := o.capabilityResolver(run)
	facts, watermark := invocationreadmodel.ProjectWithResolvers(run, events, projectedAt, capabilityResolver, o.commandResolver(run))
	if err := o.persistRunSubject(ctx, run, facts, events); err != nil {
		return err
	}
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

// projectTerminalInvocationReadModel durably marks a run as needing
// projection. Every terminal-state writer calls it (run lifecycle, executor,
// Reconciler recovery, terminal-accounting receipts); the reconcile sweep
// ProjectPendingInvocationReadModels projects marked runs idempotently, so a
// projection that fails or is never attempted is retried, not lost.
func (o *Orchestrator) projectTerminalInvocationReadModel(run *domain.Run) {
	if run == nil {
		return
	}
	queue, ok := o.invocationReadModel.(invocationreadmodel.ProjectionQueue)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), projectionMarkTimeout)
	defer cancel()
	if err := queue.MarkProjectionPending(ctx, run.ID.String(), o.now()); err != nil {
		obs.Component("invocation-read-model").Warn("needs-projection mark failed; the stale sweep will enqueue the run", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
	}
}

const (
	projectionMarkTimeout   = 5 * time.Second
	projectionSweepBudget   = 10 * time.Second
	projectionSweepBatch    = 200
	projectionRetryInterval = 15 * time.Minute
)

// ProjectPendingInvocationReadModels is the reconcile sweep for durable
// projection marks. It first enqueues ended runs whose watermark is missing or
// from an older classifier (so a classifier change re-projects history after a
// restart), then projects due marks within a time budget. A mark is cleared
// only for the generation it projected; a failure defers it with backoff.
func (o *Orchestrator) ProjectPendingInvocationReadModels(ctx context.Context) error {
	queue, ok := o.invocationReadModel.(invocationreadmodel.ProjectionQueue)
	if !ok || o.events == nil || o.runs == nil {
		return nil
	}
	now := o.now()
	if _, err := queue.EnqueueStaleProjections(ctx, runsignal.InvocationFactVersion, runsignal.EpisodeClassifierVersion, now, projectionSweepBatch); err != nil {
		return fmt.Errorf("enqueue stale projections: %w", err)
	}
	due, err := queue.DueProjections(ctx, now, projectionSweepBatch)
	if err != nil {
		return fmt.Errorf("list due projections: %w", err)
	}
	deadline := time.Now().Add(projectionSweepBudget)
	var errs []error
	for _, pending := range due {
		if ctx.Err() != nil || time.Now().After(deadline) {
			break
		}
		id, parseErr := uuid.Parse(pending.RunID)
		var run *domain.Run
		if parseErr == nil {
			run, err = o.runs.Get(ctx, id)
		}
		var notFound *domain.NotFoundError
		if parseErr != nil || errors.As(err, &notFound) || (err == nil && run == nil) {
			// A deleted run has nothing to project.
			errs = append(errs, queue.CompleteProjection(ctx, pending.RunID, pending.Generation))
			continue
		}
		if err == nil {
			err = o.projectInvocationReadModel(ctx, run)
		}
		if err != nil {
			backoff := projectionRetryInterval * time.Duration(1<<min(pending.Attempts, 6))
			errs = append(errs, queue.DeferProjection(ctx, pending.RunID, now.Add(backoff)))
			obs.Component("invocation-read-model").Warn("durable invocation projection failed; deferred", obs.KeyRunID, pending.RunID, obs.KeyError, err.Error())
			continue
		}
		errs = append(errs, queue.CompleteProjection(ctx, pending.RunID, pending.Generation))
	}
	return errors.Join(errs...)
}
