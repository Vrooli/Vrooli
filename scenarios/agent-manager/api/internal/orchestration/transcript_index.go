// Transcript index wiring connects the Transcript Indexer (USAGE-INTELLIGENCE
// T-1) to runs, events and conversation search. The indexer owns discovery,
// file identity, tailing and durable usage summaries; this file owns the run
// row each external transcript file gets and the optional recall layer.
package orchestration

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/conversationpolicy"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
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
	switch transcriptindex.Family(file.Harness) {
	case transcriptindex.HarnessCodex:
		return domain.RunnerTypeCodex, 1, nil
	case transcriptindex.HarnessClaude:
		return domain.RunnerTypeClaudeCode, 1, nil
	}
	source, err := os.Open(file.Path)
	if err != nil {
		return "", 0, err
	}
	defer source.Close()
	detection, err := h.o.detectTranscriptRunner(source, "")
	return detection.RunnerType, detection.Score, err
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
			if err := h.relinkLegacy(ctx, existing, file, parent); err != nil {
				return transcriptindex.Adoption{}, err
			}
			h.rememberRedact(existing.ID.String(), redact)
			return transcriptindex.Adoption{RunID: existing.ID.String()}, nil
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

// relinkLegacy points a snapshot-imported run at its source file and deletes
// its stale copy and events; the indexer then re-reads the source from zero
// and the projection sweep corrects its read model.
func (h *transcriptIndexHost) relinkLegacy(ctx context.Context, run *domain.Run, file transcriptindex.FileRow, parent *uuid.UUID) error {
	o := h.o
	if err := o.events.Delete(ctx, run.ID); err != nil {
		return fmt.Errorf("drop legacy imported events: %w", err)
	}
	o.notifyConversationSearch(ctx, "delete_run", run.ID.String(), "")
	if err := os.RemoveAll(filepath.Dir(run.TranscriptPath)); err != nil {
		obs.Component("transcript-index").Warn("legacy transcript copy removal failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
	} else {
		obs.Component("transcript-index").Info("removed legacy transcript copy", obs.KeyRunID, run.ID.String(), "path", filepath.Dir(run.TranscriptPath))
	}
	run.TranscriptPath, run.TranscriptCursor, run.TranscriptLastSeq = file.Path, 0, 0
	run.WebConsoleSessionID, run.ParentRunID = file.WebConsoleSessionID, parent
	run.StartedAt, run.EndedAt = nil, nil
	run.Status, run.Phase, run.UpdatedAt = domain.RunStatusRunning, domain.RunPhaseExecuting, o.now()
	return o.runs.Update(ctx, run)
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
// the active window, complete once idle. Going idle marks the run for the
// durable projection sweep.
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
	if !update.EndedAt.IsZero() && (run.EndedAt == nil || update.EndedAt.After(*run.EndedAt)) {
		ended := update.EndedAt
		run.EndedAt = &ended
	}
	wasActive := !run.Status.IsTerminal()
	if update.Active {
		run.Status, run.Phase = domain.RunStatusRunning, domain.RunPhaseExecuting
	} else {
		run.Status, run.Phase = domain.RunStatusComplete, domain.RunPhaseCompleted
		if run.StartedAt == nil {
			now := o.now()
			run.StartedAt, run.EndedAt = &now, &now
		}
	}
	run.UpdatedAt = o.now()
	if err := o.runs.Update(ctx, run); err != nil {
		return err
	}
	if wasActive && !update.Active {
		o.projectTerminalInvocationReadModel(run)
		o.notifyConversationSearch(ctx, "upsert_run", run.ID.String(), "")
	}
	return nil
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
