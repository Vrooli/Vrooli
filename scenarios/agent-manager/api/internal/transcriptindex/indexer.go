package transcriptindex

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// Config tunes the indexer. Zero values take the defaults below.
type Config struct {
	Roots RootConfig
	// Interval between ticks. Change detection is a stat walk, so a short
	// interval is cheap; reading is bounded by TickBytes. Default 1m.
	Interval time.Duration
	// TickBytes bounds how many transcript bytes one tick reads, so a first
	// run's back-fill drains across ticks, newest change first. Default 512 MiB.
	TickBytes int64
	// ActiveWindow is how recently a file must have grown for its run to be
	// `running`; after it the run is `complete`. Default 10m.
	ActiveWindow time.Duration
	// RecallWindow limits the optional full-event recall layer to files
	// modified this recently. Older files get durable summaries only and are
	// rebuilt from source on demand. Default 30 days.
	RecallWindow time.Duration
	// RecallMaxBytes is the recall layer's own budget, measured on its own
	// rows (run events plus search documents of indexed runs). Over budget,
	// the least recently active files lose their recall (never their usage
	// summaries). 0 disables recall entirely. Default 4 GiB.
	RecallMaxBytes int64
	// AfterEviction runs once after a pass evicted recall (FTS reclaim).
	AfterEviction func(context.Context) error
	Now           func() time.Time
}

const (
	defaultInterval       = time.Minute
	defaultTickBytes      = 512 << 20
	defaultActiveWindow   = 10 * time.Minute
	defaultRecallWindow   = 30 * 24 * time.Hour
	DefaultRecallMaxBytes = 4 << 30
	recordBatch           = 500
	uncertainRecordPrefix = "event recording outcome uncertain; cursor preserved: "
)

// ErrExcluded reports that a visibility policy keeps a file out of capture.
var ErrExcluded = errors.New("transcript excluded by conversation policy")

// Indexer is the Transcript Indexer.
type Indexer struct {
	cfg   Config
	store Store
	host  Host

	mu    sync.Mutex // one tick at a time
	tails map[string]*tailState

	lifecycle sync.Mutex
	cancel    context.CancelFunc
	done      chan struct{}
	ticks     atomic.Int64
}

// tailState keeps a file's parser across ticks so a continuously growing
// file is parsed as one stream.
type tailState struct {
	parser runner.TranscriptParser
	offset int64
	facts  factAccumulator
}

// New builds an indexer.
func New(store Store, host Host, cfg Config) *Indexer {
	if cfg.Interval <= 0 {
		cfg.Interval = defaultInterval
	}
	if cfg.TickBytes <= 0 {
		cfg.TickBytes = defaultTickBytes
	}
	if cfg.ActiveWindow <= 0 {
		cfg.ActiveWindow = defaultActiveWindow
	}
	if cfg.RecallWindow <= 0 {
		cfg.RecallWindow = defaultRecallWindow
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Indexer{cfg: cfg, store: store, host: host, tails: map[string]*tailState{}}
}

// Summary describes one tick.
type Summary struct {
	Discovered int      `json:"discovered"`
	Changed    int      `json:"changed"`
	Read       int      `json:"read"`
	BytesRead  int64    `json:"bytes_read"`
	Adopted    int      `json:"adopted"`
	Deferred   int      `json:"deferred"`
	Evicted    int      `json:"evicted"`
	Completed  int      `json:"completed"`
	Failures   []string `json:"failures,omitempty"`
}

// Start runs a tick now and then every Interval until Stop.
func (ix *Indexer) Start(ctx context.Context) {
	ix.lifecycle.Lock()
	defer ix.lifecycle.Unlock()
	if ix.cancel != nil {
		return
	}
	workerCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	ix.cancel, ix.done = cancel, done
	go func() {
		defer close(done)
		ticker := time.NewTicker(ix.cfg.Interval)
		defer ticker.Stop()
		for {
			_, _ = ix.RunOnce(workerCtx)
			ix.ticks.Add(1)
			select {
			case <-ticker.C:
			case <-workerCtx.Done():
				return
			}
		}
	}()
}

// Stop ends the background loop and waits for the current tick.
func (ix *Indexer) Stop() {
	ix.lifecycle.Lock()
	cancel, done := ix.cancel, ix.done
	ix.cancel, ix.done = nil, nil
	ix.lifecycle.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

// Ticks counts completed background ticks.
func (ix *Indexer) Ticks() int64 { return ix.ticks.Load() }

// tick is the state one RunOnce shares across its files.
type tick struct {
	now        time.Time
	byKey      map[string]*FileRow
	byPath     map[string]*FileRow
	candidates map[string]Candidate // key → candidate on disk now
	codexByID  map[string]string    // top-level Codex session id → candidate key
	owned      map[string]string
	budget     int64
	summary    *Summary
	inFlight   map[string]bool
}

// RunOnce discovers every root, reads changed files newest change first
// within the tick budget, completes idle runs and enforces the recall budget.
func (ix *Indexer) RunOnce(ctx context.Context) (Summary, error) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	var summary Summary
	rows, err := ix.store.ListTranscriptFiles(ctx)
	if err != nil {
		return summary, fmt.Errorf("list transcript files: %w", err)
	}
	owned, err := ix.host.OwnedTranscripts(ctx)
	if err != nil {
		return summary, fmt.Errorf("list run-owned transcripts: %w", err)
	}
	t := &tick{now: ix.cfg.Now(), byKey: map[string]*FileRow{}, byPath: map[string]*FileRow{}, candidates: map[string]Candidate{}, codexByID: map[string]string{}, owned: owned, budget: ix.cfg.TickBytes, summary: &summary, inFlight: map[string]bool{}}
	for index := range rows {
		row := &rows[index]
		t.byKey[row.Key], t.byPath[row.Path] = row, row
	}
	for _, root := range Roots(ix.cfg.Roots) {
		found, walkErr := Walk(root)
		if walkErr != nil {
			summary.Failures = append(summary.Failures, root.Path+": "+walkErr.Error())
		}
		for _, candidate := range found {
			t.candidates[candidate.Key] = candidate
			if id, ok := topLevelCodexID(candidate); ok {
				t.codexByID[id] = candidate.Key
			}
		}
	}
	summary.Discovered = len(t.candidates)
	var work []Candidate
	for _, candidate := range t.candidates {
		if ix.changed(candidate, t) {
			work = append(work, candidate)
		}
	}
	summary.Changed = len(work)
	sort.Slice(work, func(i, j int) bool { return work[i].MtimeUnixNano > work[j].MtimeUnixNano })
	for _, candidate := range work {
		if ctx.Err() != nil || t.budget <= 0 {
			summary.Deferred++
			continue
		}
		if err := ix.process(ctx, t, candidate); err != nil {
			summary.Failures = append(summary.Failures, candidate.Path+": "+err.Error())
		}
	}
	ix.completeIdle(ctx, t)
	ix.evictRecall(ctx, t)
	return summary, ctx.Err()
}

// changed is the cheap stat-based change test.
func (ix *Indexer) changed(candidate Candidate, t *tick) bool {
	row := t.byKey[candidate.Key]
	if row == nil {
		return true
	}
	if row.Status == StatusUnreadable && strings.HasPrefix(row.Error, uncertainRecordPrefix) {
		return false // a source or budget conflict requires explicit reconciliation
	}
	if row.Owner == OwnerIndexer && t.owned[candidate.Path] != "" {
		return true // an Agent Manager run now reads this file itself
	}
	if row.Owner == OwnerRun && row.IndexedAt == nil {
		return true // registered before run-owned files were read for usage (D-16)
	}
	return row.Size != candidate.Size || row.MtimeUnixNano != candidate.MtimeUnixNano || row.Path != candidate.Path || row.Status == StatusPending ||
		(row.Status == StatusUnreadable && TransientStoreFailure(row.Error)) // marked unreadable by database contention before 2026-10-29
}

// failedStatus keeps a file pending when the failure was database contention,
// so the next tick retries it from its offset; only a file that cannot be
// read or parsed is unreadable.
func failedStatus(err error) string {
	if TransientStoreFailure(err.Error()) {
		return StatusPending
	}
	return StatusUnreadable
}

func topLevelCodexID(candidate Candidate) (string, bool) {
	if Family(candidate.Root.Harness) != HarnessCodex {
		return "", false
	}
	stem := strings.TrimSuffix(filepath.Base(candidate.Path), ".jsonl")
	ids := uuidPattern.FindAllString(stem, -1)
	if strings.Contains(stem, "_") || len(ids) == 0 {
		return "", false
	}
	return ids[len(ids)-1], true
}

// process brings one file's row up to date and reads its new bytes.
func (ix *Indexer) process(ctx context.Context, t *tick, candidate Candidate) error {
	if t.inFlight[candidate.Key] {
		return nil
	}
	t.inFlight[candidate.Key] = true
	defer delete(t.inFlight, candidate.Key)
	row := t.byKey[candidate.Key]
	if row == nil {
		var err error
		if row, err = ix.newRow(ctx, t, candidate); err != nil {
			return err
		}
		t.byKey[row.Key] = row
	}
	if row.Status == StatusUnreadable && strings.HasPrefix(row.Error, uncertainRecordPrefix) {
		return errors.New(row.Error)
	}
	if row.Status == StatusPending && strings.HasPrefix(row.Error, uncertainRecordPrefix) {
		// Reconstruct parser state from the start. Re-reading only from the
		// previous cursor can generate different payloads for the same event
		// IDs (for example cumulative provider usage). The normal replay path
		// deduplicates both events and usage rows; no retained evidence is
		// deleted. A file changed since the failed read is not safe to replay:
		// the head hash alone does not cover bytes elsewhere in the file.
		if candidate.Path != row.Path || candidate.Size != row.Size || candidate.MtimeUnixNano != row.MtimeUnixNano || (row.Offset > 0 && HeadHash(candidate.Path, row.Offset) != row.HeadHash) {
			row.Status = StatusUnreadable
			row.Error = uncertainRecordPrefix + "transcript source changed; manual reconciliation required"
			if err := ix.store.SaveTranscriptFile(ctx, *row, nil); err != nil {
				return err
			}
			return errors.New(row.Error)
		}
		if candidate.Size > ix.cfg.TickBytes {
			// Recovery must reach the ambiguous batch in the same pass; losing
			// parser state between partial passes can change replayed payloads.
			row.Status = StatusUnreadable
			row.Error = uncertainRecordPrefix + "transcript exceeds tick byte budget; manual reconciliation required"
			if err := ix.store.SaveTranscriptFile(ctx, *row, nil); err != nil {
				return err
			}
			return errors.New(row.Error)
		}
		if candidate.Size > t.budget {
			t.summary.Deferred++
			return nil
		}
		row.Offset, row.Turns, row.UsageReports, row.HeadHash, row.PendingFacts = 0, 0, 0, "", ""
		delete(ix.tails, row.Key)
	}
	t.byPath[candidate.Path] = row
	row.Path, row.Size, row.MtimeUnixNano = candidate.Path, candidate.Size, candidate.MtimeUnixNano
	if runID := ownerRun(candidate, t.owned); runID != "" {
		// Agent Manager run homes and interactive tails belong to their run,
		// which records the events; the indexer records only the file's usage
		// summaries, keyed to that run, so usage has one source (D-16).
		row.Owner, row.RunID = OwnerRun, runID
	}
	if row.Status == StatusExcluded {
		row.Offset = candidate.Size
		return ix.store.SaveTranscriptFile(ctx, *row, nil)
	}
	if row.Offset > candidate.Size || (row.Offset > 0 && HeadHash(row.Path, row.Offset) != row.HeadHash) || (row.Owner == OwnerRun && row.IndexedAt == nil) {
		if err := ix.reset(ctx, row); err != nil {
			return err
		}
	}
	if row.Owner == OwnerRun {
		return ix.tail(ctx, t, row)
	}
	// Every read is admitted by the host's visibility policy, so a rule added
	// after a file was adopted stops further capture.
	parentRunID := ""
	if row.RunID == "" {
		var ok bool
		if parentRunID, ok = ix.parentRun(ctx, t, row); !ok {
			t.summary.Deferred++
			return nil
		}
	}
	adoption, err := ix.host.Adopt(ctx, *row, parentRunID)
	if err != nil {
		return ix.fail(ctx, row, fmt.Errorf("adopt: %w", err))
	}
	if adoption.Excluded {
		row.Status, row.Offset = StatusExcluded, candidate.Size
		delete(ix.tails, row.Key)
		return ix.store.SaveTranscriptFile(ctx, *row, nil)
	}
	if row.RunID == "" {
		t.summary.Adopted++
	}
	row.RunID = adoption.RunID
	return ix.tail(ctx, t, row)
}

// newRow registers a file seen for the first time. A file that took the path
// of a known row whose inode is gone is the same logical transcript replaced
// on disk (an atomic rewrite): it inherits the run and re-indexes from zero.
func (ix *Indexer) newRow(ctx context.Context, t *tick, candidate Candidate) (*FileRow, error) {
	identity := ReadIdentity(candidate.Path, candidate.Root.Harness)
	row := &FileRow{
		Key: candidate.Key, Path: candidate.Path, RootKind: candidate.Root.Kind, RootPath: candidate.Root.Path,
		Harness: identity.Harness, SessionID: identity.SessionID, HeaderSessionID: identity.HeaderSessionID,
		ParentSessionID: identity.ParentSessionID, Sidechain: identity.Sidechain,
		WebConsoleSessionID: candidate.Root.WebConsoleSessionID, Owner: OwnerIndexer, Status: StatusPending, FirstSeenAt: t.now,
	}
	if previous := t.byPath[candidate.Path]; previous != nil && previous.Key != candidate.Key {
		if _, stillThere := t.candidates[previous.Key]; !stillThere && previous.Owner == OwnerIndexer {
			// The old events must be gone before the replacement re-reads from
			// zero; if they cannot be dropped the old row stays and the pass
			// fails, so the next tick retries instead of appending a second copy.
			if err := ix.store.ResetTranscriptFile(ctx, previous.Key); err != nil {
				return nil, err
			}
			if previous.RunID != "" {
				if err := ix.host.DropRecall(ctx, previous.RunID); err != nil {
					return nil, fmt.Errorf("drop replaced transcript's events: %w", err)
				}
			}
			if err := ix.store.DeleteTranscriptFile(ctx, previous.Key); err != nil {
				return nil, fmt.Errorf("forget replaced transcript: %w", err)
			}
			row.RunID, row.FirstSeenAt = previous.RunID, previous.FirstSeenAt
			delete(t.byKey, previous.Key)
			delete(ix.tails, previous.Key)
		}
	}
	return row, nil
}

func ownerRun(candidate Candidate, owned map[string]string) string {
	if candidate.Root.RunID != "" {
		return candidate.Root.RunID
	}
	return owned[candidate.Path]
}

// parentRun resolves the run a sidechain or fork belongs to, indexing the
// parent first when it is new in this tick. ok is false when the parent
// exists but has no run yet (it is deferred by the budget): the child waits
// so it is never created unlinked.
func (ix *Indexer) parentRun(ctx context.Context, t *tick, row *FileRow) (string, bool) {
	if !row.Child() {
		return "", true
	}
	var parentKey string
	if row.Sidechain {
		identity := ReadIdentity(row.Path, row.Harness)
		if parent := t.byPath[identity.ParentPath]; parent != nil {
			parentKey = parent.Key
		} else {
			for key, candidate := range t.candidates {
				if candidate.Path == identity.ParentPath {
					parentKey = key
					break
				}
			}
		}
	} else {
		parentKey = t.codexByID[row.ParentSessionID]
		if parentKey == "" {
			for key, candidate := range t.byKey {
				if candidate.SessionID == row.ParentSessionID && !candidate.Child() {
					parentKey = key
					break
				}
			}
		}
	}
	if parentKey == "" {
		return "", true // the parent is not on this host
	}
	row.ParentFileKey = parentKey
	if parent := t.byKey[parentKey]; parent != nil && parent.RunID != "" {
		return parent.RunID, true
	}
	if candidate, onDisk := t.candidates[parentKey]; onDisk {
		if err := ix.process(ctx, t, candidate); err != nil {
			return "", false
		}
		if parent := t.byKey[parentKey]; parent != nil && parent.RunID != "" {
			return parent.RunID, true
		}
		if parent := t.byKey[parentKey]; parent != nil && parent.Status == StatusExcluded {
			return "", true
		}
	}
	return "", false
}

// reset forgets everything read from a file so it is re-read from zero.
func (ix *Indexer) reset(ctx context.Context, row *FileRow) error {
	if err := ix.store.ResetTranscriptFile(ctx, row.Key); err != nil {
		return err
	}
	if row.RunID != "" && row.Owner == OwnerIndexer { // a run's own events are never the indexer's to drop
		if err := ix.host.DropRecall(ctx, row.RunID); err != nil {
			return err
		}
	}
	row.Offset, row.Turns, row.UsageReports, row.RecallBytes, row.RecallEvicted, row.HeadHash, row.PendingFacts = 0, 0, 0, 0, false, "", ""
	delete(ix.tails, row.Key)
	return nil
}

func (ix *Indexer) fail(ctx context.Context, row *FileRow, cause error) error {
	row.Status, row.Error = failedStatus(cause), cause.Error()
	if err := ix.store.SaveTranscriptFile(ctx, *row, nil); err != nil {
		return errors.Join(cause, err)
	}
	return cause
}

// tail reads a file from its offset with its cached parser, writing usage
// summaries always. For an indexer-owned file it also records quota
// observations, full events while the file is inside the recall window and
// the run's activity; a run-owned file's run records those itself.
func (ix *Indexer) tail(ctx context.Context, t *tick, row *FileRow) error {
	runID, err := uuid.Parse(row.RunID)
	if err != nil {
		return ix.fail(ctx, row, fmt.Errorf("invalid run id %q", row.RunID))
	}
	state := ix.tails[row.Key]
	if state == nil || state.offset != row.Offset {
		parser, err := ix.host.Parser(ctx, *row)
		if err != nil {
			return ix.fail(ctx, row, err)
		}
		if setter, ok := parser.(runner.TranscriptSessionSetter); ok && row.HeaderSessionID != "" {
			// A parser resumed mid-file names the session it reads and claims
			// no amount for a counter it cannot difference (T-2).
			setter.SetTranscriptSession(row.HeaderSessionID, row.Offset == 0)
		}
		state = &tailState{parser: parser, offset: row.Offset, facts: decodeFacts(row.PendingFacts)}
		ix.tails[row.Key] = state
	}
	end := min(row.Size, row.Offset+t.budget)
	owned := row.Owner == OwnerRun
	recall := !owned && ix.cfg.RecallMaxBytes > 0 && !row.RecallEvicted && t.now.Sub(row.Mtime()) <= ix.cfg.RecallWindow
	read, err := ix.read(ctx, row, runID, state, row.Offset, end, recall, true)
	if err == nil && read.cursor == row.Offset && end < row.Size {
		// One line is longer than the remaining budget; read it whole.
		read, err = ix.read(ctx, row, runID, state, row.Offset, row.Size, recall, true)
	}
	if err != nil {
		// A failed recording may have committed a prefix, and restarting a
		// stateful parser at this offset can change an event's immutable payload.
		// Keep the old cursor; on the next tick replay the source from the start
		// under the ordinary byte budget so the parser state is reconstructed.
		delete(ix.tails, row.Key)
		if read.recordFailed || read.events > 0 {
			row.Status, row.Error = failedStatus(err), uncertainRecordPrefix+err.Error()
			if saveErr := ix.store.SaveTranscriptFile(ctx, *row, nil); saveErr != nil {
				return errors.Join(err, saveErr)
			}
			return errors.New(row.Error)
		}
		return ix.fail(ctx, row, err)
	}
	t.budget -= read.cursor - row.Offset
	t.summary.Read++
	t.summary.BytesRead += read.cursor - row.Offset
	row.Offset, state.offset = read.cursor, read.cursor
	row.Turns += read.turns
	row.UsageReports += int64(len(read.usage))
	row.PendingFacts = state.facts.encode()
	row.HeadHash = HeadHash(row.Path, row.Offset)
	row.Status, row.Error = StatusIndexed, ""
	if owned {
		row.Status = StatusOwned
	}
	if row.Offset < row.Size && end < row.Size {
		row.Status = StatusPending
	}
	now := ix.cfg.Now()
	row.IndexedAt = &now
	if !owned {
		row.Active = t.now.Sub(row.Mtime()) < ix.cfg.ActiveWindow
		update := RunUpdate{SessionID: read.sessionID, Label: read.label, StartedAt: read.started, EndedAt: read.ended, Active: row.Active}
		if updateErr := ix.host.UpdateRun(ctx, row.RunID, update); updateErr != nil {
			err = errors.Join(err, updateErr)
		}
	}
	if recall && read.events > 0 {
		if bytes, measureErr := ix.store.RecallBytes(ctx, row.RunID); measureErr == nil {
			row.RecallBytes = bytes
		}
	}
	if saveErr := ix.store.SaveTranscriptFile(ctx, *row, read.usage); saveErr != nil {
		return errors.Join(err, saveErr)
	}
	return err
}

type readResult struct {
	cursor         int64
	usage          []UsageRow
	turns          int64
	events         int
	recordFailed   bool
	sessionID      string
	label          string
	started, ended time.Time
}

// read parses [start, end) of a file. Only complete lines are consumed, so a
// line still being written is read on the next tick.
func (ix *Indexer) read(ctx context.Context, row *FileRow, runID uuid.UUID, state *tailState, start, end int64, recall, keepUsage bool) (readResult, error) {
	result := readResult{cursor: start}
	sink := &recordSink{ctx: ctx, host: ix.host, runID: row.RunID, recall: recall, discard: row.Owner == OwnerRun}
	lineOffset := start
	parse := func(id uuid.UUID, line string) runner.TranscriptParseResult {
		offset := lineOffset
		lineOffset += int64(len(line))
		parsed := state.parser.ParseTranscriptLine(id, line)
		state.facts.observeLine(line)
		state.facts.observeEvents(parsed.Events)
		if parsed.ProviderTurnStarted {
			result.turns++
		}
		ordinal := 0
		for _, event := range parsed.Events {
			if event == nil {
				continue
			}
			at := event.Timestamp
			if at.IsZero() {
				at = parsed.Timestamp
			}
			if !at.IsZero() {
				if result.started.IsZero() || at.Before(result.started) {
					result.started = at
				}
				if at.After(result.ended) {
					result.ended = at
				}
			}
			usage, ok := event.Data.(*domain.UsageEventData)
			if !ok || usage.PayloadKind == domain.PayloadKindCharge {
				continue
			}
			if keepUsage {
				result.usage = append(result.usage, UsageRow{FileKey: row.Key, LineOffset: offset, Ordinal: ordinal, RunID: row.RunID, ObservedAt: at, Usage: *usage, Facts: state.facts.take()})
			}
			ordinal++
		}
		return parsed
	}
	cursor, _, err := runner.Consume(ctx, runner.ConsumeArgs{
		RunID: runID, Transcript: row.Path, StartAt: start, EndAt: end, ParseFn: parse, EventSink: sink,
		OnAdvance:   func(cursor, _ int64) error { result.cursor = cursor; return nil },
		OnSessionID: func(id string) error { result.sessionID = id; return nil },
		OnLabel: func(label string, source domain.RunLabelSource) error {
			if source == domain.RunLabelSourceHarness {
				result.label = strings.TrimSpace(label)
			}
			return nil
		},
	})
	if cursor > result.cursor {
		result.cursor = cursor
	}
	if flushErr := sink.flush(); flushErr != nil {
		err = errors.Join(err, flushErr)
	}
	result.events = sink.recorded
	result.recordFailed = sink.failed
	return result, err
}

// recordSink batches parsed events to the host.
type recordSink struct {
	ctx      context.Context
	host     Host
	runID    string
	recall   bool
	discard  bool // the file's run records its own events
	batch    []*domain.RunEvent
	recorded int
	failed   bool
}

func (s *recordSink) Emit(event *domain.RunEvent) error {
	if event == nil {
		return nil
	}
	s.batch = append(s.batch, event)
	if len(s.batch) >= recordBatch {
		return s.flush()
	}
	return nil
}

func (s *recordSink) Close() error { return nil }

func (s *recordSink) flush() error {
	if len(s.batch) == 0 {
		return nil
	}
	batch := s.batch
	s.batch = nil
	if s.discard {
		return nil
	}
	if s.recall {
		s.recorded += len(batch)
	}
	err := s.host.Record(s.ctx, s.runID, batch, s.recall)
	if err != nil {
		s.failed = true
	}
	return err
}

// completeIdle moves runs whose files stopped growing to complete.
func (ix *Indexer) completeIdle(ctx context.Context, t *tick) {
	for _, row := range t.byKey {
		if !row.Active || row.Owner != OwnerIndexer || row.RunID == "" || t.now.Sub(row.Mtime()) < ix.cfg.ActiveWindow {
			continue
		}
		if err := ix.host.UpdateRun(ctx, row.RunID, RunUpdate{Active: false}); err != nil {
			t.summary.Failures = append(t.summary.Failures, row.Path+": complete idle run: "+err.Error())
			continue
		}
		row.Active = false
		if err := ix.store.SaveTranscriptFile(ctx, *row, nil); err == nil {
			t.summary.Completed++
		}
	}
}

// evictRecall enforces the recall budget on the recall layer's own bytes,
// dropping the full events of the least recently active files first. Their
// usage summaries stay.
func (ix *Indexer) evictRecall(ctx context.Context, t *tick) {
	if ix.cfg.RecallMaxBytes <= 0 {
		return
	}
	var total int64
	var held []*FileRow
	for _, row := range t.byKey {
		if row.RecallBytes > 0 && row.Owner == OwnerIndexer {
			total += row.RecallBytes
			held = append(held, row)
		}
	}
	if total <= ix.cfg.RecallMaxBytes {
		return
	}
	sort.Slice(held, func(i, j int) bool { return held[i].MtimeUnixNano < held[j].MtimeUnixNano })
	for _, row := range held {
		if total <= ix.cfg.RecallMaxBytes || ctx.Err() != nil {
			break
		}
		if row.Active {
			continue
		}
		if err := ix.host.DropRecall(ctx, row.RunID); err != nil {
			t.summary.Failures = append(t.summary.Failures, row.Path+": evict recall: "+err.Error())
			continue
		}
		total -= row.RecallBytes
		row.RecallBytes, row.RecallEvicted = 0, true
		if err := ix.store.SaveTranscriptFile(ctx, *row, nil); err == nil {
			t.summary.Evicted++
		}
	}
	if t.summary.Evicted > 0 && ix.cfg.AfterEviction != nil {
		if err := ix.cfg.AfterEviction(ctx); err != nil {
			t.summary.Failures = append(t.summary.Failures, "after eviction: "+err.Error())
		}
	}
}

// IndexPath captures one file on demand (manual import of a file outside the
// roots, or inside them ahead of the next tick) and returns its run.
func (ix *Indexer) IndexPath(ctx context.Context, path, harness string) (string, error) {
	path, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return "", err
	}
	if harness == "" {
		harness = HarnessConfigured
	}
	candidate, ok := Identify(Root{Path: filepath.Dir(path), Kind: RootManual, Harness: harness}, path)
	if !ok {
		return "", fmt.Errorf("transcript %s is not a readable file", path)
	}
	ix.mu.Lock()
	defer ix.mu.Unlock()
	rows, err := ix.store.ListTranscriptFiles(ctx)
	if err != nil {
		return "", err
	}
	owned, err := ix.host.OwnedTranscripts(ctx)
	if err != nil {
		return "", err
	}
	t := &tick{now: ix.cfg.Now(), byKey: map[string]*FileRow{}, byPath: map[string]*FileRow{}, candidates: map[string]Candidate{candidate.Key: candidate}, codexByID: map[string]string{}, owned: owned, summary: &Summary{}, inFlight: map[string]bool{}}
	for index := range rows {
		row := &rows[index]
		t.byKey[row.Key], t.byPath[row.Path] = row, row
	}
	if existing := t.byKey[candidate.Key]; existing != nil {
		candidate.Root = Root{Path: existing.RootPath, Kind: existing.RootKind, Harness: existing.Harness, WebConsoleSessionID: existing.WebConsoleSessionID}
	}
	for previous := int64(-1); ; {
		t.budget = 1 << 62
		if err := ix.process(ctx, t, candidate); err != nil {
			return "", err
		}
		row := t.byKey[candidate.Key]
		switch {
		case row.Status == StatusExcluded:
			return "", ErrExcluded
		case row.RunID == "":
			return "", fmt.Errorf("transcript %s could not be linked to its parent session", path)
		case row.Status != StatusPending || row.Offset == previous:
			return row.RunID, nil
		}
		previous = row.Offset
	}
}

// RebuildRecall restores a run's evicted full events from its source file.
// Usage summaries and quota observations are not rewritten: they are already
// durable and deduplicated.
func (ix *Indexer) RebuildRecall(ctx context.Context, runID string) error {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	rows, err := ix.store.ListTranscriptFiles(ctx)
	if err != nil {
		return err
	}
	for index := range rows {
		row := rows[index]
		if row.RunID != runID || row.Owner != OwnerIndexer {
			continue
		}
		parsedRunID, err := uuid.Parse(runID)
		if err != nil {
			return err
		}
		parser, err := ix.host.Parser(ctx, row)
		if err != nil {
			return err
		}
		if err := ix.host.DropRecall(ctx, runID); err != nil {
			return err
		}
		state := &tailState{parser: parser}
		if _, err := ix.read(ctx, &row, parsedRunID, state, 0, row.Offset, true, false); err != nil {
			return err
		}
		row.RecallEvicted = false
		if row.RecallBytes, err = ix.store.RecallBytes(ctx, runID); err != nil {
			return err
		}
		return ix.store.SaveTranscriptFile(ctx, row, nil)
	}
	return fmt.Errorf("run %s has no indexed transcript file", runID)
}

// SetAfterEviction installs the hook run after a pass evicted recall; wiring
// sets it once the conversation-search repository exists.
func (ix *Indexer) SetAfterEviction(fn func(context.Context) error) {
	ix.mu.Lock()
	ix.cfg.AfterEviction = fn
	ix.mu.Unlock()
}

// TransientStoreFailure reports whether a store error is SQLite contention
// (busy or locked) that a later attempt can succeed past, as opposed to a
// file the indexer cannot read.
func TransientStoreFailure(message string) bool {
	for _, marker := range []string{"SQLITE_BUSY", "SQLITE_LOCKED", "database is locked", "database table is locked"} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}
