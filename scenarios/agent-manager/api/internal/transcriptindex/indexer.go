package transcriptindex

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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
	// ReimportBytes bounds the bytes of files one tick starts re-importing
	// because an older parser version derived their recall (newest change
	// first; at least one file per tick). Default 64 MiB.
	ReimportBytes int64
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
	defaultReimportBytes  = 64 << 20
	recordBatch           = 500
	uncertainRecordPrefix = "event recording outcome uncertain; cursor preserved: "
	sourceReconcilePrefix = ReplacementReconciliationRequired
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
	if cfg.ReimportBytes <= 0 {
		cfg.ReimportBytes = defaultReimportBytes
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
	Reimported int      `json:"reimported"`
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

// tick is the state one RunOnce shares across its files.
type tick struct {
	now        time.Time
	byKey      map[string]*FileRow
	byPath     map[string]*FileRow
	candidates map[string]Candidate
	codexByID  map[string]string
	owned      map[string]string
	budget     int64
	reimport   int64
	versions   map[string]int
	summary    *Summary
	inFlight   map[string]bool
	reconcile  bool
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
	t := &tick{now: ix.cfg.Now(), byKey: map[string]*FileRow{}, byPath: map[string]*FileRow{}, candidates: map[string]Candidate{}, codexByID: map[string]string{}, owned: owned, budget: ix.cfg.TickBytes, reimport: ix.cfg.ReimportBytes, versions: map[string]int{}, summary: &summary, inFlight: map[string]bool{}}
	for index := range rows {
		row := &rows[index]
		t.byKey[row.Key], t.byPath[row.Path] = row, row
		if row.Error == ReplacementEffectsPending {
			owner, ok := ix.host.(interface {
				CompleteReplacementEffects(context.Context, Store, FileRow) error
			})
			if !ok {
				summary.Failures = append(summary.Failures, row.Path+": replacement effects owner unavailable")
			} else if err := owner.CompleteReplacementEffects(ctx, ix.store, *row); err != nil {
				summary.Failures = append(summary.Failures, row.Path+": replacement effects pending: "+err.Error())
			} else {
				row.Status, row.Error = StatusIndexed, ""
			}
		}
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
		return retriesInterruptedCapture(row) // anything else requires explicit reconciliation
	}
	if row.Status == StatusUnreadable && strings.HasPrefix(row.Error, sourceReconcilePrefix) {
		return false
	}
	if row.Owner == OwnerIndexer && t.owned[candidate.Path] != "" {
		return true // an Agent Manager run now reads this file itself
	}
	if row.Owner == OwnerRun && row.IndexedAt == nil {
		return true // registered before run-owned files were read for usage (D-16)
	}
	return row.Size != candidate.Size || row.MtimeUnixNano != candidate.MtimeUnixNano || row.Path != candidate.Path || row.Status == StatusPending ||
		(row.Status == StatusUnreadable && TransientStoreFailure(row.Error)) || // marked unreadable by database contention before 2026-10-29
		ix.stale(row, t)
}

// legacyInterruptedCapture is the state an interrupted capture was left in until
// 2026-10-30: unreadable until an operator acted. Such a row retries once.
const legacyInterruptedCapture = uncertainRecordPrefix + "source changed or cannot be verified; manual reconciliation required"

// retriesInterruptedCapture reports an unreadable row left by those older rules:
// also one whose retry met an event store that refused a payload a resumed parser
// derived differently, which the store now resolves for transcript events.
func retriesInterruptedCapture(row *FileRow) bool {
	return row.Error == legacyInterruptedCapture || (strings.HasPrefix(row.Error, uncertainRecordPrefix) && strings.Contains(row.Error, "replay_event_conflict"))
}

// interruptedCapture reports a capture that did not reach its commit.
func interruptedCapture(row *FileRow) bool {
	return (row.Status == StatusPending && strings.HasPrefix(row.Error, uncertainRecordPrefix)) || retriesInterruptedCapture(row)
}

// recalls reports whether a file's full events belong in the recall layer.
func (ix *Indexer) recalls(row *FileRow, now time.Time) bool {
	return row.Owner != OwnerRun && ix.cfg.RecallMaxBytes > 0 && !row.RecallEvicted && now.Sub(row.Mtime()) <= ix.cfg.RecallWindow
}

// stale reports an adopted recall file whose events an older parser version
// derived. Only recall is re-imported: a file outside it keeps summaries and
// no events, and is rebuilt from source on demand.
func (ix *Indexer) stale(row *FileRow, t *tick) bool {
	return row.RunID != "" && row.Offset > 0 && row.Error == "" && (row.Status == StatusIndexed || row.Status == StatusPending) &&
		ix.recalls(row, t.now) && row.ParserVersion < ix.parserVersion(t, row.Harness)
}

// parserVersion asks the host for a harness's parser version once a tick.
func (ix *Indexer) parserVersion(t *tick, harness string) int {
	version, ok := t.versions[harness]
	if !ok {
		version = ix.host.ParserVersion(harness)
		t.versions[harness] = version
	}
	return version
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
	existed := row != nil
	if row == nil {
		if prior := t.byPath[candidate.Path]; prior != nil && prior.Key != candidate.Key && prior.Owner == OwnerIndexer {
			if _, stillPresent := t.candidates[prior.Key]; !stillPresent {
				return ix.stageReplacement(ctx, t, prior, candidate, false)
			}
		}
		var err error
		if row, err = ix.newRow(ctx, t, candidate); err != nil {
			return err
		}
		t.byKey[row.Key] = row
	}
	if row.Status == StatusUnreadable && !retriesInterruptedCapture(row) && strings.HasPrefix(row.Error, uncertainRecordPrefix) {
		return errors.New(row.Error)
	}
	if row.Status == StatusUnreadable && strings.HasPrefix(row.Error, sourceReconcilePrefix) {
		return errors.New(row.Error)
	}
	if row.Error == ReplacementEffectsPending {
		return errors.New(row.Error)
	}
	checkpoint := *row
	var expected *FileRow
	if existed {
		expected = &checkpoint
	}
	retrying := false
	if interruptedCapture(row) {
		// The checkpoint did not move: its commit is atomic and consumes the
		// intent. A prefix of events may be stored, but events have stable ids
		// and the event store refuses changed evidence, so reading again from
		// the cursor cannot double count or rewrite anything; a parser resumed
		// mid-file is the normal restart path. An API restart during any tick
		// lands here, so live sessions must not need an operator. Only a source
		// that no longer matches the cursor stops.
		intent := row.Error
		if row.Offset > candidate.Size || (row.Offset > 0 && HeadHash(candidate.Path, row.Offset) != row.HeadHash) {
			row.Status = StatusUnreadable
			row.Error = uncertainRecordPrefix + "source no longer matches the checkpoint; manual reconciliation required"
			delete(ix.tails, row.Key)
			owner, ok := ix.store.(interface {
				FailTranscriptCapture(context.Context, string, string, string, string) error
			})
			if !ok {
				return errors.New("atomic transcript failure marker unavailable")
			}
			if err := owner.FailTranscriptCapture(ctx, row.Key, intent, row.Status, row.Error); err != nil {
				return err
			}
			return errors.New(row.Error)
		}
		delete(ix.tails, row.Key)
		row.Error, row.Status = "", StatusPending
		retrying = true
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
		return ix.saveIfUnchanged(ctx, expected, *row)
	}
	if row.Offset > candidate.Size || (row.Offset > 0 && HeadHash(row.Path, row.Offset) != row.HeadHash) || (row.Owner == OwnerRun && row.IndexedAt == nil) {
		if checkpoint.Owner == OwnerIndexer && checkpoint.Offset > 0 {
			*row = checkpoint
			if ownerRun(candidate, t.owned) != "" {
				return ix.refuseReplacement(ctx, row, "source is now owned by a managed run")
			}
			return ix.stageReplacement(ctx, t, row, candidate, false)
		}
		if checkpoint.Owner == OwnerRun && checkpoint.IndexedAt == nil && checkpoint.Error == "" {
			owner, ok := ix.store.(interface {
				ResetEmptyOwnedTranscript(context.Context, FileRow, FileRow) error
			})
			if !ok {
				return errors.New("run-owned backfill owner unavailable")
			}
			row.Offset, row.Turns, row.UsageReports, row.RecallBytes, row.RecallEvicted, row.HeadHash, row.PendingFacts = 0, 0, 0, 0, false, "", ""
			row.LastObservedEventAt = nil
			if err := owner.ResetEmptyOwnedTranscript(ctx, checkpoint, *row); err != nil {
				return err
			}
			checkpoint = *row
			delete(ix.tails, row.Key)
		} else if checkpoint.Owner == OwnerRun && checkpoint.Offset > 0 {
			*row = checkpoint
			return ix.refuseReplacement(ctx, row, "run-owned source changed; usage reconciliation required")
		} else if !(row.Owner == OwnerRun && row.IndexedAt == nil && row.Offset == 0) {
			if err := ix.reset(ctx, row); err != nil {
				return err
			}
		}
	} else if !retrying && ix.stale(row, t) {
		if checkpoint.Owner == OwnerIndexer && checkpoint.Offset > 0 {
			*row = checkpoint
			if t.reimport <= 0 {
				t.summary.Deferred++
				return nil
			}
			return ix.stageReplacement(ctx, t, row, candidate, true)
		}
		// An older parser derived this file's events: drop them and re-read
		// the file from zero into the same run, within the re-import budget.
		if t.reimport <= 0 {
			t.summary.Deferred++
			return nil
		}
		t.reimport -= candidate.Size
		if err := ix.reset(ctx, row); err != nil {
			return err
		}
		t.summary.Reimported++
	}
	if row.Owner == OwnerRun {
		return ix.tail(ctx, t, row, expected)
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
		return ix.fail(ctx, row, expected, fmt.Errorf("adopt: %w", err))
	}
	if adoption.Excluded {
		row.Status, row.Offset = StatusExcluded, candidate.Size
		delete(ix.tails, row.Key)
		return ix.saveIfUnchanged(ctx, expected, *row)
	}
	if adoption.LegacyRelink {
		owner, ok := ix.store.(interface {
			AdmitLegacyRelink(context.Context, *FileRow, FileRow, string) error
		})
		if !ok {
			return errors.New("atomic legacy relink admission unavailable")
		}
		row.RunID, row.Status, row.Error = adoption.RunID, StatusUnreadable, sourceReconcilePrefix+"legacy relink awaiting staged promotion"
		if err := owner.AdmitLegacyRelink(ctx, expected, *row, adoption.LegacyPath); err != nil {
			return err
		}
		priorReconcile := t.reconcile
		t.reconcile = true
		err := ix.stageReplacement(ctx, t, row, candidate, false)
		t.reconcile = priorReconcile
		return err
	}
	if row.RunID == "" {
		t.summary.Adopted++
	}
	row.RunID = adoption.RunID
	return ix.tail(ctx, t, row, expected)
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
			return nil, ix.refuseReplacement(ctx, previous, "source inode changed before replacement capture")
		}
	}
	return row, nil
}

func (ix *Indexer) refuseReplacement(ctx context.Context, row *FileRow, reason string) error {
	if row.Status == StatusUnreadable && strings.HasPrefix(row.Error, sourceReconcilePrefix) {
		return errors.New(row.Error)
	}
	expected := *row
	row.Status, row.Error = StatusUnreadable, sourceReconcilePrefix+reason
	delete(ix.tails, row.Key)
	if err := ix.saveIfUnchanged(ctx, &expected, *row); err != nil {
		return err
	}
	return errors.New(row.Error)
}

type replacementHost interface {
	ReplacementTokens(context.Context, Store, FileRow) (string, string, error)
	CommitReplacement(context.Context, Store, string, string, string, FileRow, []UsageRow, []*domain.RunEvent, bool, RunUpdate) (bool, error)
}

// stageReplacement reads a bounded immutable snapshot and promotes its event,
// usage, checkpoint and run state together only if both old owner tokens still
// match. Any pre-promotion failure preserves the old evidence.
func (ix *Indexer) stageReplacement(ctx context.Context, t *tick, previous *FileRow, candidate Candidate, reimport bool) error {
	if strings.HasPrefix(previous.Error, uncertainRecordPrefix) {
		return errors.New(previous.Error)
	}
	if previous.Status == StatusUnreadable && strings.HasPrefix(previous.Error, sourceReconcilePrefix) && !t.reconcile {
		return errors.New(previous.Error)
	}
	if candidate.Size > rebuildLimit {
		return ix.refuseReplacement(ctx, previous, "source exceeds replacement staging limit")
	}
	if candidate.Size > t.budget {
		t.summary.Deferred++
		return nil
	}
	host, ok := ix.host.(replacementHost)
	if !ok {
		return ix.refuseReplacement(ctx, previous, "atomic replacement owner unavailable")
	}
	adoption, err := ix.host.Adopt(ctx, *previous, "")
	if err != nil {
		return ix.refuseReplacement(ctx, previous, "visibility check failed: "+err.Error())
	}
	if adoption.Excluded {
		return ErrExcluded
	}
	fileToken, eventToken, err := host.ReplacementTokens(ctx, ix.store, *previous)
	if err != nil {
		return ix.refuseReplacement(ctx, previous, "evidence token unavailable: "+err.Error())
	}
	source, err := os.Open(candidate.Path)
	if err != nil {
		return ix.refuseReplacement(ctx, previous, "open replacement: "+err.Error())
	}
	before, err := source.Stat()
	if err != nil {
		source.Close()
		return ix.refuseReplacement(ctx, previous, "stat replacement: "+err.Error())
	}
	if !before.Mode().IsRegular() || before.Size() != candidate.Size || before.ModTime().UnixNano() != candidate.MtimeUnixNano {
		source.Close()
		return ix.refuseReplacement(ctx, previous, "replacement source changed before snapshot")
	}
	data, readErr := io.ReadAll(io.LimitReader(source, candidate.Size))
	after, statErr := source.Stat()
	closeErr := source.Close()
	if int64(len(data)) != candidate.Size && readErr == nil {
		readErr = io.ErrUnexpectedEOF
	}
	if err := errors.Join(readErr, statErr, closeErr); err != nil {
		return ix.refuseReplacement(ctx, previous, "read replacement: "+err.Error())
	}
	pathInfo, err := os.Stat(candidate.Path)
	if err != nil || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) || !os.SameFile(before, after) || !os.SameFile(before, pathInfo) {
		return ix.refuseReplacement(ctx, previous, "replacement source changed during snapshot")
	}
	for _, line := range bytes.Split(data, []byte{'\n'}) {
		line = bytes.TrimSpace(line)
		if len(line) != 0 && !json.Valid(line) {
			return ix.refuseReplacement(ctx, previous, "replacement contains malformed JSON")
		}
	}
	file := *previous
	file.Key, file.Path, file.Size, file.MtimeUnixNano = candidate.Key, candidate.Path, candidate.Size, candidate.MtimeUnixNano
	file.Offset, file.Turns, file.UsageReports, file.RecallBytes = 0, 0, 0, 0
	file.HeadHash, file.PendingFacts, file.Error = "", "", ""
	file.LastObservedEventAt = nil
	parser, err := ix.host.Parser(ctx, file)
	if err != nil {
		return ix.refuseReplacement(ctx, previous, "replacement parser: "+err.Error())
	}
	if setter, ok := parser.(runner.TranscriptSessionSetter); ok && file.HeaderSessionID != "" {
		setter.SetTranscriptSession(file.HeaderSessionID, true)
	}
	runID, err := uuid.Parse(file.RunID)
	if err != nil {
		return ix.refuseReplacement(ctx, previous, "replacement run identity: "+err.Error())
	}
	state := &tailState{parser: parser}
	staged := &stagedRecall{Host: ix.host}
	result, err := ix.readSource(ctx, &file, runID, state, 0, candidate.Size, true, true, staged, bytes.NewReader(data))
	if err != nil {
		return ix.refuseReplacement(ctx, previous, "replacement parse: "+err.Error())
	}
	if result.cursor != candidate.Size {
		return ix.refuseReplacement(ctx, previous, "replacement has an incomplete final record")
	}
	if err := verifySnapshot(candidate.Path, before, data); err != nil {
		return ix.refuseReplacement(ctx, previous, "replacement source changed during staging: "+err.Error())
	}
	if result.sessionID != "" && file.HeaderSessionID != "" && result.sessionID != file.HeaderSessionID {
		return ix.refuseReplacement(ctx, previous, "replacement session identity changed")
	}
	file.Offset, file.Turns, file.UsageReports = result.cursor, result.turns, int64(len(result.usage))
	file.PendingFacts, file.HeadHash = state.facts.encode(), hashHead(data)
	file.ParserVersion = ix.parserVersion(t, file.Harness)
	file.Status, file.Error = StatusPending, ReplacementEffectsPending
	file.Active = t.now.Sub(file.Mtime()) < ix.cfg.ActiveWindow
	if !result.ended.IsZero() {
		observed := result.ended
		file.LastObservedEventAt = &observed
	}
	now := ix.cfg.Now()
	file.IndexedAt = &now
	update := RunUpdate{SessionID: result.sessionID, Label: result.label, StartedAt: result.started, EndedAt: result.ended, Active: file.Active, Recorded: len(staged.events)}
	retain := ix.recalls(&file, t.now)
	if !retain {
		update.Recorded = 0
	}
	committed, err := host.CommitReplacement(ctx, ix.store, previous.Key, fileToken, eventToken, file, result.usage, staged.events, retain, update)
	if !committed {
		var precommit *ReplacementPrecommitError
		if errors.As(err, &precommit) {
			return ix.refuseReplacement(ctx, previous, precommit.Error())
		}
		// A commit error may be ambiguous. Never overwrite either checkpoint
		// while determining whether promotion took effect.
		previous.Status, previous.Error = StatusPending, sourceReconcilePrefix+"replacement outcome uncertain; inspect owner before retry"
		return err
	}
	delete(t.byKey, previous.Key)
	delete(ix.tails, previous.Key)
	if err == nil {
		file.Status, file.Error = StatusIndexed, ""
	}
	if retainedBytes, measureErr := ix.store.RecallBytes(ctx, file.RunID); measureErr == nil {
		file.RecallBytes = retainedBytes
	}
	*previous = file
	t.byKey[file.Key], t.byPath[file.Path] = previous, previous
	t.budget -= candidate.Size
	t.summary.Read++
	t.summary.BytesRead += candidate.Size
	if reimport {
		t.summary.Reimported++
		t.reimport -= candidate.Size
	}
	return err
}

type ReplacementIssue struct {
	FileKey string `json:"fileKey"`
	Path    string `json:"path"`
	Reason  string `json:"reason"`
}

type ReplacementIssues struct {
	Items     []ReplacementIssue `json:"items"`
	Truncated bool               `json:"truncated"`
}

// ReplacementIssues reports exact guarded file keys for a single run.
func (ix *Indexer) ReplacementIssues(ctx context.Context, runID string) (ReplacementIssues, error) {
	if _, err := uuid.Parse(runID); err != nil {
		return ReplacementIssues{}, errors.New("valid run ID is required")
	}
	rows, err := ix.store.ListTranscriptFiles(ctx)
	if err != nil {
		return ReplacementIssues{}, err
	}
	out := ReplacementIssues{Items: []ReplacementIssue{}}
	if owner, ok := ix.store.(interface {
		LegacyCleanupIssue(context.Context, string) (*ReplacementIssue, error)
	}); ok {
		issue, err := owner.LegacyCleanupIssue(ctx, runID)
		if err != nil {
			return out, err
		}
		if issue != nil {
			out.Items = append(out.Items, *issue)
		}
	}
	for _, row := range rows {
		if row.RunID != runID || row.Status != StatusUnreadable || !strings.HasPrefix(row.Error, sourceReconcilePrefix) {
			continue
		}
		if len(out.Items) == 50 {
			out.Truncated = true
			break
		}
		out.Items = append(out.Items, ReplacementIssue{FileKey: row.Key, Path: row.Path, Reason: row.Error})
	}
	return out, nil
}

// ReconcileReplacement retries only a specific, explicitly guarded source
// replacement. It never clears uncertain event-recording outcomes.
func (ix *Indexer) ReconcileReplacement(ctx context.Context, runID, fileKey string) error {
	if _, err := uuid.Parse(runID); err != nil || strings.TrimSpace(fileKey) == "" {
		return errors.New("exact run ID and file key are required")
	}
	ix.mu.Lock()
	defer ix.mu.Unlock()
	rows, err := ix.store.ListTranscriptFiles(ctx)
	if err != nil {
		return err
	}
	var target *FileRow
	for i := range rows {
		row := &rows[i]
		if row.Key == fileKey && row.RunID == runID {
			target = row
			break
		}
	}
	if target == nil {
		return errors.New("replacement checkpoint not found for exact run and file key")
	}
	if target.Status != StatusUnreadable || !strings.HasPrefix(target.Error, sourceReconcilePrefix) {
		return errors.New("checkpoint is not awaiting replacement reconciliation")
	}
	root := Root{Path: target.RootPath, Kind: target.RootKind, Harness: target.Harness, WebConsoleSessionID: target.WebConsoleSessionID}
	candidate, ok := Identify(root, target.Path)
	if !ok {
		return errors.New("replacement source unavailable")
	}
	owned, err := ix.host.OwnedTranscripts(ctx)
	if err != nil {
		return err
	}
	if ownerRun(candidate, owned) != "" {
		return errors.New("replacement source is now owned by a managed run")
	}
	var summary Summary
	t := &tick{now: ix.cfg.Now(), byKey: map[string]*FileRow{}, byPath: map[string]*FileRow{}, candidates: map[string]Candidate{candidate.Key: candidate}, owned: owned, budget: rebuildLimit, reimport: rebuildLimit, versions: map[string]int{}, summary: &summary, inFlight: map[string]bool{}, reconcile: true}
	for i := range rows {
		row := &rows[i]
		t.byKey[row.Key], t.byPath[row.Path] = row, row
	}
	return ix.stageReplacement(ctx, t, target, candidate, false)
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
	row.LastObservedEventAt = nil
	delete(ix.tails, row.Key)
	return nil
}

func (ix *Indexer) saveIfUnchanged(ctx context.Context, expected *FileRow, file FileRow) error {
	owner, ok := ix.store.(interface {
		SaveTranscriptIfUnchanged(context.Context, *FileRow, FileRow) error
	})
	if !ok {
		return errors.New("atomic transcript checkpoint update unavailable")
	}
	return owner.SaveTranscriptIfUnchanged(ctx, expected, file)
}

func (ix *Indexer) fail(ctx context.Context, row *FileRow, expected *FileRow, cause error) error {
	row.Status, row.Error = failedStatus(cause), cause.Error()
	if err := ix.saveIfUnchanged(ctx, expected, *row); err != nil {
		return errors.Join(cause, err)
	}
	return cause
}

// tail reads a file from its offset with its cached parser, writing usage
// summaries always. For an indexer-owned file it also records quota
// observations, full events while the file is inside the recall window and
// the run's activity; a run-owned file's run records those itself.
func (ix *Indexer) tail(ctx context.Context, t *tick, row *FileRow, expected *FileRow) error {
	previous := *row
	runID, err := uuid.Parse(row.RunID)
	if err != nil {
		return ix.fail(ctx, row, expected, fmt.Errorf("invalid run id %q", row.RunID))
	}
	state := ix.tails[row.Key]
	if state == nil || state.offset != row.Offset {
		parser, err := ix.host.Parser(ctx, *row)
		if err != nil {
			return ix.fail(ctx, row, expected, err)
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
	intentToken := ""
	if !owned {
		if _, ok := ix.host.(interface {
			CommitTranscript(context.Context, Store, FileRow, []UsageRow, *FileRow, string, RunUpdate) error
		}); !ok {
			return errors.New("atomic transcript/run update unavailable")
		}
	} else if _, ok := ix.store.(interface {
		CommitOwnedTranscript(context.Context, FileRow, []UsageRow, string) error
	}); !ok {
		return errors.New("atomic run-owned transcript commit unavailable")
	}
	owner, ok := ix.store.(interface {
		BeginTranscriptCapture(context.Context, *FileRow, FileRow) error
	})
	if !ok {
		return errors.New("atomic transcript capture intent unavailable")
	}
	if _, ok := ix.store.(interface {
		FailTranscriptCapture(context.Context, string, string, string, string) error
	}); !ok {
		return errors.New("atomic transcript failure marker unavailable")
	}
	intent := previous
	intentToken = uncertainRecordPrefix + "capture interrupted; reconciliation required; token=" + uuid.NewString()
	intent.Status, intent.Error = StatusPending, intentToken
	if err := owner.BeginTranscriptCapture(ctx, expected, intent); err != nil {
		return fmt.Errorf("save capture intent: %w", err)
	}
	recall := ix.recalls(row, t.now)
	read, err := ix.read(ctx, row, runID, state, row.Offset, end, recall, true)
	if err == nil && read.cursor == row.Offset && end < row.Size {
		// One line is longer than the remaining budget; read it whole.
		read, err = ix.read(ctx, row, runID, state, row.Offset, row.Size, recall, true)
	}
	if err != nil {
		// A failed recording may have committed a prefix, and restarting a
		// stateful parser at this offset can change an event's immutable payload.
		// Keep the old cursor and surface uncertainty for reconciliation.
		delete(ix.tails, row.Key)
		row.Status, row.Error = failedStatus(err), uncertainRecordPrefix+err.Error()
		owner := ix.store.(interface {
			FailTranscriptCapture(context.Context, string, string, string, string) error
		})
		if saveErr := owner.FailTranscriptCapture(ctx, row.Key, intentToken, row.Status, row.Error); saveErr != nil {
			return errors.Join(err, saveErr)
		}
		return errors.New(row.Error)
	}
	t.budget -= read.cursor - row.Offset
	t.summary.Read++
	t.summary.BytesRead += read.cursor - row.Offset
	row.Offset, state.offset = read.cursor, read.cursor
	row.Turns += read.turns
	row.UsageReports += int64(len(read.usage))
	row.PendingFacts = state.facts.encode()
	row.ParserVersion = ix.parserVersion(t, row.Harness)
	if !read.ended.IsZero() && (row.LastObservedEventAt == nil || read.ended.After(*row.LastObservedEventAt)) {
		observed := read.ended
		row.LastObservedEventAt = &observed
	}
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
	var update RunUpdate
	if !owned {
		// A stale mtime does not make an incompletely captured file idle.
		// Keep the import nonterminal until its entire observed source has
		// been consumed, even when the tick's byte budget ended mid-file.
		row.Active = row.Offset < row.Size || t.now.Sub(row.Mtime()) < ix.cfg.ActiveWindow
		update = RunUpdate{SessionID: read.sessionID, Label: read.label, StartedAt: read.started, EndedAt: read.ended, Active: row.Active, Recorded: read.events}
		if row.LastObservedEventAt != nil {
			update.EndedAt = *row.LastObservedEventAt
		}
	}
	if recall && read.events > 0 {
		if bytes, measureErr := ix.store.RecallBytes(ctx, row.RunID); measureErr == nil {
			row.RecallBytes = bytes
		}
	}
	if !owned {
		if owner, ok := ix.host.(interface {
			CommitTranscript(context.Context, Store, FileRow, []UsageRow, *FileRow, string, RunUpdate) error
		}); ok {
			if commitErr := owner.CommitTranscript(ctx, ix.store, *row, read.usage, nil, intentToken, update); commitErr != nil {
				delete(ix.tails, row.Key)
				*row = previous
				row.Status, row.Error = StatusPending, uncertainRecordPrefix+"commit outcome uncertain; reconciliation required"
				// A commit error can be ambiguous; writing the old checkpoint
				// here could overwrite a transaction that actually committed.
				return fmt.Errorf("transcript/run commit failed or outcome is uncertain; inspect owner before retry: %w", commitErr)
			}
			return nil
		}
		return errors.New("atomic transcript/run update unavailable")
	}
	ownedStore, ok := ix.store.(interface {
		CommitOwnedTranscript(context.Context, FileRow, []UsageRow, string) error
	})
	if !ok {
		return errors.New("atomic run-owned transcript commit unavailable")
	}
	if saveErr := ownedStore.CommitOwnedTranscript(ctx, *row, read.usage, intentToken); saveErr != nil {
		return errors.Join(err, saveErr)
	}
	return nil
}

type readResult struct {
	cursor         int64
	usage          []UsageRow
	turns          int64
	events         int
	sessionID      string
	label          string
	started, ended time.Time
}

// read parses [start, end) of a file. Only complete lines are consumed, so a
// line still being written is read on the next tick.
func (ix *Indexer) read(ctx context.Context, row *FileRow, runID uuid.UUID, state *tailState, start, end int64, recall, keepUsage bool) (readResult, error) {
	return ix.readSource(ctx, row, runID, state, start, end, recall, keepUsage, ix.host, nil)
}

func (ix *Indexer) readSource(ctx context.Context, row *FileRow, runID uuid.UUID, state *tailState, start, end int64, recall, keepUsage bool, host Host, snapshot io.Reader) (readResult, error) {
	result := readResult{cursor: start}
	sink := &recordSink{ctx: ctx, host: host, runID: row.RunID, recall: recall, discard: row.Owner == OwnerRun}
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
		RunID: runID, Transcript: row.Path, Snapshot: snapshot, StartAt: start, EndAt: end, ParseFn: parse, EventSink: sink,
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
	return err
}

// completeIdle moves runs whose files stopped growing to complete.
func (ix *Indexer) completeIdle(ctx context.Context, t *tick) {
	for _, row := range t.byKey {
		if !row.Active || row.Owner != OwnerIndexer || row.RunID == "" || t.now.Sub(row.Mtime()) < ix.cfg.ActiveWindow {
			continue
		}
		// An old checkpoint does not establish that a missing, changed, or
		// incompletely captured source is idle. In particular, uncertain event
		// writes and deferred reads must not become an imported completion.
		candidate, observed := t.candidates[row.Key]
		if !observed || row.Status != StatusIndexed || row.Error != "" || row.Offset < candidate.Size ||
			row.Path != candidate.Path || row.Size != candidate.Size || row.MtimeUnixNano != candidate.MtimeUnixNano || ix.stale(row, t) {
			continue
		}
		update := RunUpdate{Active: false}
		if row.LastObservedEventAt != nil {
			update.EndedAt = *row.LastObservedEventAt
		}
		if owner, ok := ix.host.(interface {
			CommitTranscript(context.Context, Store, FileRow, []UsageRow, *FileRow, string, RunUpdate) error
		}); ok {
			checkpoint := *row
			checkpoint.Active = false
			if err := owner.CommitTranscript(ctx, ix.store, checkpoint, nil, row, "", update); err != nil {
				t.summary.Failures = append(t.summary.Failures, row.Path+": complete idle run: "+err.Error())
				continue
			}
			row.Active = false
			t.summary.Completed++
			continue
		}
		t.summary.Failures = append(t.summary.Failures, row.Path+": complete idle run: atomic transcript/run update unavailable")
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
		if row.Active || row.Status != StatusIndexed || row.Error != "" {
			continue
		}
		owner, ok := ix.host.(interface {
			EvictRecall(context.Context, Store, FileRow) (bool, error)
		})
		if !ok {
			t.summary.Failures = append(t.summary.Failures, row.Path+": atomic recall eviction unavailable")
			continue
		}
		committed, err := owner.EvictRecall(ctx, ix.store, *row)
		if committed {
			total -= row.RecallBytes
			row.RecallBytes, row.RecallEvicted = 0, true
			if err != nil {
				row.Status, row.Error = StatusPending, ReplacementEffectsPending
			}
			t.summary.Evicted++
		}
		if err != nil {
			t.summary.Failures = append(t.summary.Failures, row.Path+": evict recall: "+err.Error())
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
	t := &tick{now: ix.cfg.Now(), byKey: map[string]*FileRow{}, byPath: map[string]*FileRow{}, candidates: map[string]Candidate{candidate.Key: candidate}, codexByID: map[string]string{}, owned: owned, reimport: 1 << 62, versions: map[string]int{}, summary: &Summary{}, inFlight: map[string]bool{}}
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

const rebuildLimit = 32 << 20

// verifySnapshot detects additional source changes before promotion. A matching
// reread still cannot prove atomicity against a cooperating concurrent writer.
func verifySnapshot(path string, expected os.FileInfo, data []byte) error {
	return verifySnapshotWithHook(path, expected, data, nil)
}

func verifySnapshotWithHook(path string, expected os.FileInfo, data []byte, afterRead func()) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(expected, before) || before.Size() != expected.Size() || !before.ModTime().Equal(expected.ModTime()) {
		return errors.New("source metadata changed")
	}
	buf := make([]byte, 64<<10)
	for at := 0; at < len(data); {
		n := min(len(buf), len(data)-at)
		if _, err := io.ReadFull(f, buf[:n]); err != nil {
			return err
		}
		if !bytes.Equal(buf[:n], data[at:at+n]) {
			return errors.New("source bytes changed")
		}
		at += n
	}
	if afterRead != nil {
		afterRead()
	}
	after, err := f.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return errors.New("source metadata changed during verification")
	}
	pathInfo, err := os.Stat(path)
	if err != nil || !os.SameFile(after, pathInfo) {
		return errors.New("source path changed during verification")
	}
	return nil
}

type stagedRecall struct {
	Host
	events []*domain.RunEvent
	bytes  int
}

func (s *stagedRecall) Record(_ context.Context, _ string, events []*domain.RunEvent, _ bool) error {
	for _, event := range events {
		encoded, err := json.Marshal(event)
		if err != nil {
			return err
		}
		s.bytes += len(encoded)
		if s.bytes > rebuildLimit {
			return errors.New("recall rebuild exceeds event staging limit")
		}
		s.events = append(s.events, event)
	}
	return nil
}

type recallReplacer interface {
	RecallToken(context.Context, string) (string, error)
	ReplaceRecall(context.Context, Store, FileRow, FileRow, string, []*domain.RunEvent) (bool, error)
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
		checkpoint := row
		parsedRunID, err := uuid.Parse(runID)
		if err != nil {
			return err
		}
		parser, err := ix.host.Parser(ctx, row)
		if err != nil {
			return err
		}
		replacer, ok := ix.host.(recallReplacer)
		if !ok {
			return errors.New("recall replacement is unavailable")
		}
		eventToken, err := replacer.RecallToken(ctx, runID)
		if err != nil {
			return err
		}
		if row.Offset <= 0 || row.Offset > rebuildLimit {
			return errors.New("recall rebuild exceeds source staging limit or has no captured bytes")
		}
		source, err := os.Open(row.Path)
		if err != nil {
			return err
		}
		before, err := source.Stat()
		if err != nil {
			source.Close()
			return err
		}
		if !before.Mode().IsRegular() || before.Size() != row.Size || before.ModTime().UnixNano() != row.MtimeUnixNano {
			source.Close()
			return errors.New("recall source differs from checkpoint or is not a regular file")
		}
		data, copyErr := io.ReadAll(io.LimitReader(source, row.Offset))
		if int64(len(data)) != row.Offset && copyErr == nil {
			copyErr = io.ErrUnexpectedEOF
		}
		after, statErr := source.Stat()
		closeErr := source.Close()
		if err := errors.Join(copyErr, statErr, closeErr); err != nil {
			return err
		}
		if after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) || !os.SameFile(before, after) {
			return errors.New("recall source changed during snapshot")
		}
		fingerprint, decodeErr := hex.DecodeString(row.HeadHash)
		if decodeErr != nil || len(fingerprint) != 32 {
			return errors.New("recall checkpoint head fingerprint unavailable or invalid; reconciliation required")
		}
		if hashHead(data) != row.HeadHash {
			return errors.New("recall snapshot differs from checkpoint head fingerprint; reconciliation required")
		}
		state := &tailState{parser: parser}
		staged := &stagedRecall{Host: ix.host}
		result, err := ix.readSource(ctx, &row, parsedRunID, state, 0, row.Offset, true, false, staged, bytes.NewReader(data))
		if err != nil {
			return err
		}
		if result.cursor != row.Offset {
			return errors.New("recall snapshot was not fully parsed")
		}
		if err := verifySnapshot(row.Path, before, data); err != nil {
			return fmt.Errorf("recall source changed during staging: %w", err)
		}
		row.RecallEvicted, row.ParserVersion = false, ix.host.ParserVersion(row.Harness)
		row.Status, row.Error = StatusPending, ReplacementEffectsPending
		_, err = replacer.ReplaceRecall(ctx, ix.store, checkpoint, row, eventToken, staged.events)
		return err
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
