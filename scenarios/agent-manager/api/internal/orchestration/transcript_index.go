// Transcript index wiring connects the Transcript Indexer (USAGE-INTELLIGENCE
// T-1) to runs, events and conversation search. The indexer owns discovery,
// file identity, tailing and durable usage summaries; this file owns the run
// row each external transcript file gets and the optional recall layer.
package orchestration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/conversationpolicy"
	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/pricing"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/transcriptindex"
	"agent-manager/internal/transcriptredact"

	"github.com/google/uuid"
)

// TranscriptIndex runs the Transcript Indexer for an orchestrator. Its
// background loop replaces the snapshot import sweep; RunOnce is the same tick
// on demand.
type TranscriptIndex struct {
	*transcriptindex.Indexer
	store transcriptindex.Store
}

// NewTranscriptIndex builds the indexer over store and installs it on the
// orchestrator, which needs it for manual import, recall rebuilds and
// usage summaries of runs whose events left the recall layer.
func NewTranscriptIndex(o *Orchestrator, store transcriptindex.Store, cfg transcriptindex.Config) *TranscriptIndex {
	host := &transcriptIndexHost{o: o, redact: map[string]bool{}}
	index := &TranscriptIndex{Indexer: transcriptindex.New(store, host, cfg), store: store}
	o.transcriptIndex = index
	return index
}

// RunOnce satisfies the operator-triggered sweep seam.
func (t *TranscriptIndex) RunOnce(ctx context.Context) (transcriptindex.Summary, error) {
	if t == nil {
		return transcriptindex.Summary{}, nil
	}
	return t.Indexer.RunOnce(ctx)
}

type transcriptIndexHost struct {
	o      *Orchestrator
	mu     sync.Mutex
	redact map[string]bool
}

func (h *transcriptIndexHost) OwnedTranscripts(ctx context.Context) (map[string]string, error) {
	paths, err := h.o.runs.ListRunTranscriptPaths(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(paths))
	for path, id := range paths {
		out[path] = id.String()
	}
	return out, nil
}

func (h *transcriptIndexHost) runnerType(file transcriptindex.FileRow) (domain.RunnerType, float64, error) {
	if runnerType, ok := familyRunner(file.Harness); ok {
		return runnerType, 1, nil
	}
	source, err := os.Open(file.Path)
	if err != nil {
		return "", 0, err
	}
	defer source.Close()
	detection, err := h.o.detectTranscriptRunner(source, "")
	return detection.RunnerType, detection.Score, err
}

// familyRunner names the runner of a harness known without reading the file.
func familyRunner(harness string) (domain.RunnerType, bool) {
	switch transcriptindex.Family(harness) {
	case transcriptindex.HarnessCodex:
		return domain.RunnerTypeCodex, true
	case transcriptindex.HarnessClaude:
		return domain.RunnerTypeClaudeCode, true
	}
	return "", false
}

// ParserVersion is the version the harness's codec declares for the events
// it derives (runner.TranscriptVersioner); a configured harness, whose runner
// is detected per file, is version 0 and never re-imported.
func (h *transcriptIndexHost) ParserVersion(harness string) int {
	runnerType, ok := familyRunner(harness)
	if !ok {
		return 0
	}
	owned, err := h.o.runners.Get(runnerType)
	if err != nil {
		return 0
	}
	factory, ok := owned.(runner.TranscriptParserFactory)
	if !ok {
		return 0
	}
	if versioned, ok := factory.NewTranscriptParser().(runner.TranscriptVersioner); ok {
		return versioned.TranscriptVersion()
	}
	return 0
}

// TranscriptParserVersion is the current parser version of a harness's
// transcripts, for readers that judge which runs an older parser read.
func (o *Orchestrator) TranscriptParserVersion(harness string) int {
	return (&transcriptIndexHost{o: o}).ParserVersion(harness)
}

// Adopt applies the import visibility policy to a file and returns its run:
// the run it already has, or a new one. A top-level file whose session
// was adopted by the retired snapshot importer relinks that run (its copy is
// deleted) instead of duplicating it; every other file gets a new run.
func (h *transcriptIndexHost) Adopt(ctx context.Context, file transcriptindex.FileRow, parentRunID string) (transcriptindex.Adoption, error) {
	o := h.o
	runnerType, score, err := h.runnerType(file)
	if err != nil {
		return transcriptindex.Adoption{}, err
	}
	redact := false
	if o.conversationPolicy.Enabled() {
		fields := conversationpolicy.Fields{Harness: file.Harness, Tag: importedRunTag}
		// The import counters record each file at adoption and when a later
		// rule excludes it; an excluded file is never re-admitted.
		decision := o.conversationPolicy.Evaluate(conversationpolicy.EnforceImport, fields)
		if file.RunID == "" || decision.Hidden() {
			decision = o.conversationPolicy.RecordImport(fields)
		}
		if decision.Hidden() {
			obs.Component("transcript-index").Info("transcript excluded at import by policy", "rule", conversationpolicy.DescribeRule(decision.Rule), "path", file.Path)
			// A rule added after adoption makes the exclusion real: the run's
			// derived search projection is dropped and no further events land.
			if file.RunID != "" {
				o.notifyConversationSearch(ctx, "delete_run", file.RunID, "")
			}
			return transcriptindex.Adoption{Excluded: true}, nil
		}
		redact = decision.Matched && decision.Action == conversationpolicy.ActionRedact && o.piiMasker != nil
	}
	if file.RunID != "" {
		h.rememberRedact(file.RunID, redact)
		return transcriptindex.Adoption{RunID: file.RunID}, nil
	}
	var parent *uuid.UUID
	if id, err := uuid.Parse(parentRunID); err == nil {
		parent = &id
	}
	provenance := file.SessionID
	if existing, err := o.runs.GetByImportProvenance(ctx, file.Harness, provenance); err != nil {
		return transcriptindex.Adoption{}, err
	} else if existing != nil {
		switch {
		case existing.TranscriptPath == file.Path:
			h.rememberRedact(existing.ID.String(), redact)
			return transcriptindex.Adoption{RunID: existing.ID.String()}, nil
		case !file.Child() && h.isLegacyCopy(ctx, existing):
			h.rememberRedact(existing.ID.String(), redact)
			return transcriptindex.Adoption{RunID: existing.ID.String(), LegacyRelink: true, LegacyPath: existing.TranscriptPath}, nil
		default:
			// Another file already carries this session id (a resumed
			// rollout): provenance stays unique per file.
			provenance = file.SessionID + "#" + file.Key
		}
	}
	task, err := o.ensureImportedTranscriptTask(ctx)
	if err != nil {
		return transcriptindex.Adoption{}, err
	}
	now := o.now()
	run := &domain.Run{
		ID: uuid.New(), TaskID: task.ID, Tag: importedRunTag, RunMode: domain.RunModeInPlace, ExecutionMode: domain.ExecutionModeImported,
		Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting,
		ResolvedConfig:      &domain.RunConfig{RunnerType: runnerType, ManifestIndexSnapshot: runsignal.CurrentCatalogSnapshot(), TranscriptCodec: string(runnerType), TranscriptCodecScore: score},
		ImportSourceHarness: file.Harness, ImportSourceSessionID: provenance, ImportedAt: &now,
		TranscriptPath: file.Path, SessionID: file.HeaderSessionID, WebConsoleSessionID: file.WebConsoleSessionID, ParentRunID: parent,
	}
	var markers runner.GoalMarkerProvider
	if owned, err := o.runners.Get(runnerType); err == nil {
		markers, _ = owned.(runner.GoalMarkerProvider)
	}
	evidence := readTranscriptHead(file.Path, markers)
	run.Label, run.LabelSource = o.resolveImportedLabel(ctx, ImportTranscriptRequest{SourceSessionID: file.SessionID}, runnerType, evidence.label)
	run.GoalID = evidence.goalID
	if err := o.runs.Create(ctx, run); err != nil {
		return transcriptindex.Adoption{}, err
	}
	h.rememberRedact(run.ID.String(), redact)
	return transcriptindex.Adoption{RunID: run.ID.String()}, nil
}

const importedRunTag = "agent-manager-imported"

// isLegacyCopy reports whether a run was adopted by the snapshot importer,
// which pointed transcript_path at its private copy under the run-state root.
func (h *transcriptIndexHost) isLegacyCopy(ctx context.Context, run *domain.Run) bool {
	if run.ExecutionMode.Normalized() != domain.ExecutionModeImported || filepath.Base(run.TranscriptPath) != "transcript.ndjson" {
		return false
	}
	root, err := h.o.resolveRunStateRoot(ctx)
	return err == nil && filepath.Dir(run.TranscriptPath) == filepath.Join(root, run.ID.String())
}

func (h *transcriptIndexHost) rememberRedact(runID string, redact bool) {
	h.mu.Lock()
	h.redact[runID] = redact
	h.mu.Unlock()
}

func (h *transcriptIndexHost) redacts(ctx context.Context, runID uuid.UUID) bool {
	h.mu.Lock()
	redact, known := h.redact[runID.String()]
	h.mu.Unlock()
	if known {
		return redact
	}
	run, err := h.o.runs.Get(ctx, runID)
	redact = err == nil && h.o.redactMatchesExistingRun(run)
	h.rememberRedact(runID.String(), redact)
	return redact
}

func (h *transcriptIndexHost) Parser(ctx context.Context, file transcriptindex.FileRow) (runner.TranscriptParser, error) {
	runnerType, _, err := h.runnerType(file)
	if err != nil {
		return nil, err
	}
	owned, err := h.o.runners.Get(runnerType)
	if err != nil {
		return nil, fmt.Errorf("unsupported transcript format: %w", err)
	}
	factory, ok := owned.(runner.TranscriptParserFactory)
	if !ok {
		return nil, fmt.Errorf("runner %q has no transcript parser", runnerType)
	}
	parser := factory.NewTranscriptParser()
	if id, err := uuid.Parse(file.RunID); err == nil {
		if run, err := h.o.runs.Get(ctx, id); err == nil && run != nil {
			if setter, ok := parser.(runner.TranscriptModelSetter); ok {
				setter.SetTranscriptModel(runTranscriptModel(run))
			}
			if setter, ok := parser.(runner.TranscriptBillingSetter); ok {
				setter.SetTranscriptBilling(run.Billing)
			}
		}
	}
	if setter, ok := parser.(runner.TranscriptRetentionSetter); ok {
		setter.SetTranscriptRetention(true)
	}
	return parser, nil
}

// Record records provider quota frames from every batch; it appends the full
// events (with user text redacted and PII masked when policy says so) only
// for recall. Imported runs are read-only history, so token caps and workflow
// nudges never act on them.
func (h *transcriptIndexHost) Record(ctx context.Context, runID string, events []*domain.RunEvent, recall bool) error {
	id, err := uuid.Parse(runID)
	if err != nil {
		return err
	}
	if !recall {
		for _, event := range events {
			h.o.observeQuotaEvent(event, false)
		}
		return nil
	}
	masked := make([]*domain.RunEvent, 0, len(events))
	redact := h.redacts(ctx, id)
	for _, event := range events {
		if message, ok := event.Data.(*domain.MessageEventData); ok && (strings.EqualFold(message.Role, "user") || strings.EqualFold(message.Role, "operator")) {
			copied := *message
			copied.Content = transcriptredact.Redact(copied.Content)
			event = &domain.RunEvent{ID: event.ID, RunID: event.RunID, EventType: event.EventType, Timestamp: event.Timestamp, Data: &copied}
		}
		if redact {
			event, _ = h.o.piiMasker.MaskEvent(ctx, event)
		}
		masked = append(masked, event)
	}
	if err := h.o.events.Append(ctx, id, masked...); err != nil {
		return err
	}
	for _, event := range masked {
		h.o.observeQuotaEvent(event, true)
	}
	return nil
}

// UpdateRun applies a tick's activity: the event window, the harness title,
// and the status rule for external runs — running while the file grew within
// the active window, complete once idle. Going idle, or gaining events while
// idle (a re-import), marks the run for the durable projection sweep.
func (h *transcriptIndexHost) UpdateRun(ctx context.Context, runID string, update transcriptindex.RunUpdate) error {
	o := h.o
	id, err := uuid.Parse(runID)
	if err != nil {
		return err
	}
	run, err := o.runs.Get(ctx, id)
	if err != nil {
		return err
	}
	if run == nil {
		return domain.NewNotFoundError("Run", id)
	}
	wasActive := h.applyRunUpdate(run, update)
	if err := o.runs.Update(ctx, run); err != nil {
		return err
	}
	h.afterRunUpdate(ctx, run, update, wasActive)
	return nil
}

func (h *transcriptIndexHost) applyRunUpdate(run *domain.Run, update transcriptindex.RunUpdate) bool {
	o := h.o
	if update.SessionID != "" && run.SessionID == "" {
		run.SessionID = update.SessionID
	}
	if update.Label != "" && run.LabelSource != domain.RunLabelSourceManual {
		run.Label, run.LabelSource = shortenTranscriptLabel(update.Label), domain.RunLabelSourceHarness
	}
	if !update.StartedAt.IsZero() && (run.StartedAt == nil || update.StartedAt.Before(*run.StartedAt)) {
		started := update.StartedAt
		run.StartedAt = &started
	}
	if !update.Active && !update.EndedAt.IsZero() && (run.EndedAt == nil || update.EndedAt.After(*run.EndedAt)) {
		ended := update.EndedAt
		run.EndedAt = &ended
	}
	wasActive := !run.Status.IsTerminal()
	if update.Active {
		run.Status, run.Phase = domain.RunStatusRunning, domain.RunPhaseExecuting
		run.EndedAt = nil
	} else {
		run.Status, run.Phase = domain.RunStatusComplete, domain.RunPhaseCompleted
		if run.StartedAt == nil {
			now := o.now()
			run.StartedAt, run.EndedAt = &now, &now
		}
	}
	run.UpdatedAt = o.now()
	return wasActive
}

func (h *transcriptIndexHost) afterRunUpdate(ctx context.Context, run *domain.Run, update transcriptindex.RunUpdate, wasActive bool) {
	if !update.Active && (wasActive || update.Recorded > 0) {
		h.o.projectTerminalInvocationReadModel(run)
		h.o.notifyConversationSearch(ctx, "upsert_run", run.ID.String(), "")
	}
}

func (h *transcriptIndexHost) CommitTranscript(ctx context.Context, store transcriptindex.Store, file transcriptindex.FileRow, usage []transcriptindex.UsageRow, expected *transcriptindex.FileRow, intent string, update transcriptindex.RunUpdate) error {
	owner, ok := store.(interface {
		CommitTranscriptAndRun(context.Context, transcriptindex.FileRow, []transcriptindex.UsageRow, *transcriptindex.FileRow, string, func(*domain.Run) error) error
	})
	if !ok {
		return fmt.Errorf("atomic transcript/run update unavailable")
	}
	var run *domain.Run
	var wasActive bool
	if err := owner.CommitTranscriptAndRun(ctx, file, usage, expected, intent, func(current *domain.Run) error {
		run = current
		wasActive = h.applyRunUpdate(current, update)
		return nil
	}); err != nil {
		return err
	}
	h.afterRunUpdate(ctx, run, update, wasActive)
	return nil
}

type replacementStore interface {
	TranscriptReplacementToken(context.Context, string) (string, error)
	MarkReplacementIntent(context.Context, string, string) (string, error)
	CommitTranscriptReplacement(context.Context, string, string, transcriptindex.FileRow, []transcriptindex.UsageRow, func(*domain.Run) error, func(*sql.Tx) error) error
	FinishReplacementEffects(context.Context, transcriptindex.FileRow) error
}

func (h *transcriptIndexHost) ReplacementTokens(ctx context.Context, store transcriptindex.Store, file transcriptindex.FileRow) (string, string, error) {
	owner, ok := store.(replacementStore)
	if !ok {
		return "", "", fmt.Errorf("atomic transcript replacement unavailable")
	}
	eventOwner, ok := h.o.events.(event.StreamReplacer)
	if !ok {
		return "", "", fmt.Errorf("event stream replacement unavailable")
	}
	id, err := uuid.Parse(file.RunID)
	if err != nil {
		return "", "", err
	}
	fileToken, err := owner.TranscriptReplacementToken(ctx, file.Key)
	if err != nil {
		return "", "", err
	}
	eventToken, err := eventOwner.StreamToken(ctx, id)
	return fileToken, eventToken, err
}

func (h *transcriptIndexHost) CommitReplacement(ctx context.Context, store transcriptindex.Store, oldKey, fileToken, eventToken string, file transcriptindex.FileRow, usage []transcriptindex.UsageRow, events []*domain.RunEvent, retain bool, update transcriptindex.RunUpdate) (bool, error) {
	reject := func(err error) (bool, error) { return false, &transcriptindex.ReplacementPrecommitError{Err: err} }
	owner, ok := store.(replacementStore)
	if !ok {
		return reject(fmt.Errorf("atomic transcript replacement unavailable"))
	}
	eventOwner, ok := h.o.events.(event.StreamReplacer)
	if !ok {
		return reject(fmt.Errorf("event stream replacement unavailable"))
	}
	id, err := uuid.Parse(file.RunID)
	if err != nil {
		return reject(err)
	}
	masked := make([]*domain.RunEvent, 0, len(events))
	var observations []*pricing.QuotaObservation
	redact := h.redacts(ctx, id)
	for _, item := range events {
		if item.EventType == domain.EventTypeQuota && !h.o.quotaObservationsDisabled {
			if h.o.quotaObservations == nil {
				return reject(fmt.Errorf("quota observation owner unavailable"))
			}
			observation, err := pricing.ObservationFromEvent(item, h.o.now().UTC())
			if err != nil {
				return reject(fmt.Errorf("replacement quota observation: %w", err))
			}
			if observation != nil {
				if !retain {
					observation.EvidenceRef = "agent-manager://runs/" + file.RunID
				}
				observations = append(observations, observation)
			}
		}
		if !retain {
			continue
		}
		if message, ok := item.Data.(*domain.MessageEventData); ok && (strings.EqualFold(message.Role, "user") || strings.EqualFold(message.Role, "operator")) {
			copied := *message
			copied.Content = transcriptredact.Redact(copied.Content)
			item = &domain.RunEvent{ID: item.ID, RunID: item.RunID, EventType: item.EventType, Timestamp: item.Timestamp, Data: &copied}
		}
		if redact {
			var failed bool
			item, failed = h.o.piiMasker.MaskEvent(ctx, item)
			if failed {
				return reject(fmt.Errorf("replacement PII masking unavailable; prior evidence retained"))
			}
		}
		data, err := json.Marshal(item.Data)
		if err != nil {
			return reject(fmt.Errorf("measure replacement event: %w", err))
		}
		if item.Data == nil {
			data = []byte("{}")
		}
		file.RecallBytes += int64(len(data))
		masked = append(masked, item)
	}
	quotaOwner, quotaAtomic := h.o.quotaObservations.(interface {
		RecordQuotaObservationInTx(context.Context, *sql.Tx, *pricing.QuotaObservation) error
	})
	if len(observations) > 0 && !quotaAtomic {
		return reject(fmt.Errorf("atomic quota observation owner unavailable"))
	}
	fileToken, err = owner.MarkReplacementIntent(ctx, oldKey, fileToken)
	if err != nil {
		return false, fmt.Errorf("replacement intent failed or outcome uncertain: %w", err)
	}
	guard := eventOwner.LockReplacement()
	var legacyPath string
	err = owner.CommitTranscriptReplacement(ctx, oldKey, fileToken, file, usage, func(current *domain.Run) error {
		if current.TranscriptPath != file.Path {
			if file.Child() || !h.isLegacyCopy(ctx, current) || current.ImportSourceHarness != file.Harness || current.ImportSourceSessionID != file.SessionID {
				return fmt.Errorf("replacement run source changed during staging")
			}
			legacyPath = current.TranscriptPath
			current.TranscriptPath, current.TranscriptCursor, current.TranscriptLastSeq = file.Path, 0, 0
			current.WebConsoleSessionID = file.WebConsoleSessionID
		}
		h.applyRunUpdate(current, update)
		return nil
	}, func(tx *sql.Tx) error {
		if legacyPath != "" {
			cleanupOwner, ok := store.(interface {
				RecordLegacyCleanupInTx(context.Context, *sql.Tx, transcriptindex.FileRow, string) error
			})
			if !ok {
				return fmt.Errorf("legacy cleanup owner unavailable")
			}
			if err := cleanupOwner.RecordLegacyCleanupInTx(ctx, tx, file, legacyPath); err != nil {
				return err
			}
		}
		if err := guard.ReplaceInTx(ctx, tx, id, eventToken, masked); err != nil {
			return err
		}
		for _, observation := range observations {
			if err := quotaOwner.RecordQuotaObservationInTx(ctx, tx, observation); err != nil {
				return err
			}
		}
		return nil
	})
	// Even an error may be a commit with an unknown result. Closing existing
	// readers makes them reconnect instead of retaining an invalid sequence.
	guard.Finish(id, true)
	if err != nil {
		return false, fmt.Errorf("replacement failed or outcome uncertain; inspect owner before retry: %w", err)
	}
	if err := h.CompleteReplacementEffects(ctx, store, file); err != nil {
		return true, fmt.Errorf("replacement committed but derived effects remain pending: %w", err)
	}
	return true, nil
}

func (h *transcriptIndexHost) CompleteReplacementEffects(ctx context.Context, store transcriptindex.Store, file transcriptindex.FileRow) error {
	owner, ok := store.(replacementStore)
	if !ok {
		return fmt.Errorf("replacement effects owner unavailable")
	}
	id, err := uuid.Parse(file.RunID)
	if err != nil {
		return err
	}
	run, err := h.o.runs.Get(ctx, id)
	if err != nil {
		return err
	}
	if run == nil {
		return domain.NewNotFoundError("Run", id)
	}
	if run.Status.IsTerminal() {
		queue, ok := h.o.invocationReadModel.(invocationreadmodel.ProjectionQueue)
		if !ok {
			return fmt.Errorf("invocation projection owner unavailable")
		}
		if err := queue.MarkProjectionPending(ctx, run.ID.String(), h.o.now()); err != nil {
			return err
		}
	}
	if h.o.conversationSearchNotify == nil {
		return fmt.Errorf("conversation search owner unavailable")
	}
	if err := h.o.conversationSearchNotify(ctx, "delete_run", file.RunID, ""); err != nil {
		return err
	}
	if err := h.o.conversationSearchNotify(ctx, "upsert_run", file.RunID, ""); err != nil {
		return err
	}
	return owner.FinishReplacementEffects(ctx, file)
}

func (h *transcriptIndexHost) EvictRecall(ctx context.Context, store transcriptindex.Store, expected transcriptindex.FileRow) (bool, error) {
	owner, ok := store.(interface {
		CommitRecallEviction(context.Context, transcriptindex.FileRow, transcriptindex.FileRow, func(*sql.Tx) error) error
	})
	if !ok {
		return false, fmt.Errorf("atomic recall eviction unavailable")
	}
	eventOwner, ok := h.o.events.(event.StreamReplacer)
	if !ok {
		return false, fmt.Errorf("event stream eviction unavailable")
	}
	id, err := uuid.Parse(expected.RunID)
	if err != nil {
		return false, err
	}
	token, err := eventOwner.StreamToken(ctx, id)
	if err != nil {
		return false, err
	}
	file := expected
	file.RecallBytes, file.RecallEvicted = 0, true
	file.Status, file.Error = transcriptindex.StatusPending, transcriptindex.ReplacementEffectsPending
	guard := eventOwner.LockReplacement()
	err = owner.CommitRecallEviction(ctx, expected, file, func(tx *sql.Tx) error { return guard.ReplaceInTx(ctx, tx, id, token, nil) })
	guard.Finish(id, true)
	if err != nil {
		return false, fmt.Errorf("recall eviction failed or outcome uncertain: %w", err)
	}
	if err := h.CompleteReplacementEffects(ctx, store, file); err != nil {
		return true, fmt.Errorf("recall eviction committed but derived effects remain pending: %w", err)
	}
	return true, nil
}

func (h *transcriptIndexHost) DropRecall(ctx context.Context, runID string) error {
	id, err := uuid.Parse(runID)
	if err != nil {
		return err
	}
	if err := h.o.events.Delete(ctx, id); err != nil {
		return err
	}
	return h.o.DeleteRunFromConversationSearch(ctx, runID)
}

func (h *transcriptIndexHost) RecallToken(ctx context.Context, runID string) (string, error) {
	id, err := uuid.Parse(runID)
	if err != nil {
		return "", err
	}
	owner, ok := h.o.events.(event.StreamReplacer)
	if !ok {
		return "", fmt.Errorf("atomic recall event owner unavailable")
	}
	return owner.StreamToken(ctx, id)
}

func (h *transcriptIndexHost) ReplaceRecall(ctx context.Context, store transcriptindex.Store, expected, file transcriptindex.FileRow, token string, events []*domain.RunEvent) (bool, error) {
	id, err := uuid.Parse(expected.RunID)
	if err != nil {
		return false, err
	}
	owner, ok := store.(interface {
		CommitRecallRebuild(context.Context, transcriptindex.FileRow, transcriptindex.FileRow, func(*sql.Tx) error) error
	})
	if !ok {
		return false, fmt.Errorf("atomic recall checkpoint owner unavailable")
	}
	eventOwner, ok := h.o.events.(event.StreamReplacer)
	if !ok {
		return false, fmt.Errorf("atomic recall event owner unavailable")
	}
	masked := make([]*domain.RunEvent, 0, len(events))
	redact := h.redacts(ctx, id)
	file.RecallBytes = 0
	for _, item := range events {
		if message, ok := item.Data.(*domain.MessageEventData); ok && (strings.EqualFold(message.Role, "user") || strings.EqualFold(message.Role, "operator")) {
			copied := *message
			copied.Content = transcriptredact.Redact(copied.Content)
			item = &domain.RunEvent{ID: item.ID, RunID: item.RunID, EventType: item.EventType, Timestamp: item.Timestamp, Data: &copied}
		}
		if redact {
			var failed bool
			item, failed = h.o.piiMasker.MaskEvent(ctx, item)
			if failed {
				return false, fmt.Errorf("recall PII masking unavailable; prior evidence retained")
			}
		}
		data, err := json.Marshal(item.Data)
		if err != nil {
			return false, err
		}
		if item.Data == nil {
			data = []byte("{}")
		}
		file.RecallBytes += int64(len(data))
		masked = append(masked, item)
	}
	guard := eventOwner.LockReplacement()
	err = owner.CommitRecallRebuild(ctx, expected, file, func(tx *sql.Tx) error { return guard.AppendEmptyInTx(ctx, tx, id, token, masked) })
	if err != nil {
		guard.Finish(id, true)
		return false, fmt.Errorf("recall promotion failed or outcome uncertain; inspect owner before retry: %w", err)
	}
	guard.FinishAppend(id, true, masked)
	if err := h.CompleteReplacementEffects(ctx, store, file); err != nil {
		return true, fmt.Errorf("recall events committed but derived effects remain pending: %w", err)
	}
	return true, nil
}

// summaryUsageEvents returns an imported run's usage rebuilt from durable
// summaries when its retained events carry none (recall evicted or never
// written), so projection and accounting keep its tokens.
func (o *Orchestrator) summaryUsageEvents(ctx context.Context, run *domain.Run, events []*domain.RunEvent) []*domain.RunEvent {
	if o.transcriptIndex == nil || run == nil || run.ExecutionMode.Normalized() != domain.ExecutionModeImported {
		return events
	}
	for _, event := range events {
		if _, ok := event.Data.(*domain.UsageEventData); ok {
			return events
		}
	}
	rows, err := o.transcriptIndex.store.TranscriptUsageForRun(ctx, run.ID.String())
	if err != nil || len(rows) == 0 {
		return events
	}
	return append(events, transcriptindex.UsageEvents(rows)...)
}

// transcriptHead is what a file's first bytes say about its label and goal.
type transcriptHead struct {
	label  transcriptLabelEvidence
	goalID string
}

// transcriptHeadBytes bounds the label and goal scan; a harness title that
// appears later reaches the run through the parser's label callback.
const transcriptHeadBytes = 4 << 20

func readTranscriptHead(path string, markers runner.GoalMarkerProvider) transcriptHead {
	var head transcriptHead
	file, err := os.Open(path)
	if err != nil {
		return head
	}
	defer file.Close()
	if evidence, err := readTranscriptLabelEvidence(io.LimitReader(file, transcriptHeadBytes)); err == nil {
		head.label = evidence
	}
	if _, err := file.Seek(0, 0); err == nil {
		head.goalID, _ = transcriptGoalMetadata(io.LimitReader(file, transcriptHeadBytes), markers)
	}
	return head
}
