package transcriptindex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/vrooli/cli-core/agenttranscript"
)

// Source replacement: a transcript whose bytes no longer extend its
// checkpoint (rewritten in place, replaced under its path, or re-derived by a
// newer parser) is staged from an immutable snapshot and promoted atomically.
// A source that moves while it is staged defers the replacement; only a
// refusal that needs an owner's judgment locks the checkpoint.

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
	// CommitReplacement promotes file over previous, the checkpoint it
	// replaces (the same file rewritten, moved or recompressed).
	CommitReplacement(ctx context.Context, store Store, previous FileRow, fileToken, eventToken string, file FileRow, usage []UsageRow, events []*domain.RunEvent, retain bool, update RunUpdate) (bool, error)
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
		return ix.refuseReplacement(ctx, previous, stagingLimitReason)
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
	snapshot, err := snapshotTranscript(candidate)
	if err != nil {
		var limit *stagingLimitError
		if errors.As(err, &limit) {
			return ix.refuseReplacement(ctx, previous, stagingLimitReason)
		}
		return deferReplacement(err)
	}
	data := snapshot.data
	if len(data) == 0 {
		return deferReplacement(errors.New("replacement source has no complete record yet"))
	}
	if previous.RootKind != RootGeminiCLI && previous.RootKind != RootCursor {
		for _, line := range bytes.Split(data, []byte{'\n'}) {
			line = bytes.TrimSpace(line)
			if len(line) != 0 && !json.Valid(line) {
				return ix.refuseReplacement(ctx, previous, "replacement contains malformed JSON")
			}
		}
	}
	file := *previous
	// The checkpoint covers the snapshot: a live source that grew past it, or
	// ends in a line still being written, is tailed from here next tick.
	file.Key, file.Path, file.Size, file.MtimeUnixNano = candidate.Key, candidate.Path, snapshot.extent, snapshot.mtime
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
	result, err := ix.readSource(ctx, &file, runID, state, 0, int64(len(data)), true, true, staged, bytes.NewReader(data))
	if err != nil {
		return ix.refuseReplacement(ctx, previous, "replacement parse: "+err.Error())
	}
	if result.cursor != int64(len(data)) {
		return ix.refuseReplacement(ctx, previous, "replacement has an incomplete final record")
	}
	if err := snapshot.verify(); err != nil {
		return deferReplacement(fmt.Errorf("replacement source changed during staging: %w", err))
	}
	if result.sessionID != "" && file.HeaderSessionID != "" && result.sessionID != file.HeaderSessionID {
		return ix.refuseReplacement(ctx, previous, "replacement session identity changed")
	}
	file.Offset, file.Turns, file.UsageReports = result.cursor, result.turns, int64(len(result.usage))
	file.PendingFacts, file.HeadHash = state.facts.encode(), agenttranscript.SHA256BytesPrefix(data, headHashBytes)
	if file.RootKind == RootGeminiCLI || file.RootKind == RootCursor {
		file.HeadHash = agenttranscript.SHA256Bytes(data)
	}
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
	for _, other := range t.byKey {
		if other.Key != previous.Key && other.RunID == file.RunID && other.Path != "" {
			update.LineagePaths = append(update.LineagePaths, other.Path)
		}
	}
	retain := ix.recalls(&file, t.now)
	if !retain {
		update.Recorded = 0
	}
	committed, err := host.CommitReplacement(ctx, ix.store, *previous, fileToken, eventToken, file, result.usage, staged.events, retain, update)
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
	// The staged parser continues the stream, so the next tail does not
	// resume a fresh parser mid-file.
	ix.tails[file.Key] = &tailState{parser: parser, offset: file.Offset, facts: state.facts}
	if err == nil {
		file.Status, file.Error = StatusIndexed, ""
	}
	if retainedBytes, measureErr := ix.store.RecallBytes(ctx, file.RunID); measureErr == nil {
		file.RecallBytes = retainedBytes
	}
	*previous = file
	t.byKey[file.Key], t.byPath[file.Path] = previous, previous
	t.budget -= file.Offset
	t.summary.Read++
	t.summary.BytesRead += file.Offset
	if reimport {
		t.summary.Reimported++
		t.reimport -= file.Offset
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
	t := &tick{now: ix.cfg.Now(), byKey: map[string]*FileRow{}, byPath: map[string]*FileRow{}, candidates: map[string]Candidate{candidate.Key: candidate}, owned: owned, budget: rebuildLimit, reimport: rebuildLimit, versions: map[string]int{}, summary: &summary, inFlight: map[string]bool{}, done: map[string]bool{}, reconcile: true}
	for i := range rows {
		row := &rows[i]
		t.byKey[row.Key], t.byPath[row.Path] = row, row
	}
	return ix.stageReplacement(ctx, t, target, candidate, false)
}

const rebuildLimit = 32 << 20

// stagingLimitError reports a source too large to stage in memory.
type stagingLimitError struct{ size int64 }

func (e *stagingLimitError) Error() string {
	return fmt.Sprintf("source of %d bytes exceeds the %d byte staging limit", e.size, rebuildLimit)
}

// transcriptSnapshot is an immutable extent of a source, staged in memory.
type transcriptSnapshot struct {
	// data is the decoded bytes up to the last complete line.
	data []byte
	// extent is every decoded byte observed, including a trailing line still
	// being written; the checkpoint covers it and the next tail rereads that
	// line once it is complete.
	extent int64
	mtime  int64
	// verify reports any change to the staged bytes since they were read.
	verify func() error
}

// snapshotTranscript stages the source a candidate names. An appending writer
// is expected: a live rollout that grew since the walk is staged at its
// current length, and verification checks only that the staged bytes are
// still the file's prefix. A sealed (compressed) source is immutable, so its
// verification is that the raw file is unchanged.
func snapshotTranscript(candidate Candidate) (*transcriptSnapshot, error) {
	path := candidate.Path
	if candidate.Root.Kind == RootCursor {
		data, err := ReadCursorSnapshot(context.Background(), path, candidate.SessionID)
		if err != nil {
			return nil, fmt.Errorf("read Cursor snapshot: %w", err)
		}
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			if err != nil {
				return nil, fmt.Errorf("stat Cursor state database: %w", err)
			}
			return nil, errors.New("Cursor state database is not a regular file")
		}
		snapshot := &transcriptSnapshot{data: data, extent: int64(len(data)), mtime: info.ModTime().UnixNano()}
		snapshot.verify = func() error {
			current, err := ReadCursorSnapshot(context.Background(), path, candidate.SessionID)
			if err != nil || agenttranscript.SHA256Bytes(current) != agenttranscript.SHA256Bytes(data) {
				return errors.New("Cursor composer source changed")
			}
			return nil
		}
		return snapshot, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open replacement: %w", err)
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat replacement: %w", err)
	}
	if !before.Mode().IsRegular() || fileKey(path, before) != candidate.Key {
		return nil, errors.New("replacement source changed before snapshot: the path names another file")
	}
	snapshot := &transcriptSnapshot{mtime: before.ModTime().UnixNano()}
	if Sealed(path) {
		decoded, err := OpenTranscript(path)
		if err != nil {
			return nil, fmt.Errorf("read replacement: %w", err)
		}
		data, err := io.ReadAll(io.LimitReader(decoded, rebuildLimit+1))
		if closeErr := decoded.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			return nil, fmt.Errorf("read replacement: %w", err)
		}
		if int64(len(data)) > rebuildLimit {
			return nil, &stagingLimitError{size: candidate.Size}
		}
		snapshot.data = data
		snapshot.verify = func() error {
			now, err := os.Stat(path)
			if err != nil || !os.SameFile(before, now) || now.Size() != before.Size() || !now.ModTime().Equal(before.ModTime()) {
				return errors.New("sealed source changed")
			}
			return nil
		}
	} else {
		if before.Size() < candidate.Size {
			return nil, errors.New("replacement source changed before snapshot: it shrank")
		}
		if before.Size() > rebuildLimit {
			return nil, &stagingLimitError{size: before.Size()}
		}
		data, err := io.ReadAll(io.LimitReader(file, before.Size()))
		if err == nil && int64(len(data)) != before.Size() {
			err = io.ErrUnexpectedEOF
		}
		if err != nil {
			return nil, fmt.Errorf("read replacement: %w", err)
		}
		snapshot.data = data
		snapshot.verify = func() error { return verifyPrefix(path, before, data) }
	}
	snapshot.extent = int64(len(snapshot.data))
	if candidate.Root.Kind == RootGeminiCLI || candidate.Root.Kind == RootCursor {
		// Gemini CLI archives are one complete JSON document and commonly have
		// no trailing newline. Treat the entire exact snapshot as its record.
		snapshot.verify = func() error {
			now, err := os.Stat(path)
			if err != nil || !os.SameFile(before, now) || now.Size() != before.Size() || !now.ModTime().Equal(before.ModTime()) {
				return errors.New("Gemini CLI source changed")
			}
			return nil
		}
	} else {
		snapshot.data = snapshot.data[:bytes.LastIndexByte(snapshot.data, '\n')+1]
	}
	return snapshot, nil
}

// verifyPrefix detects a change to data, read from the start of the file
// expected describes: the file may have grown, but the path must still name
// it and it must still begin with data. A matching reread cannot prove
// atomicity against a cooperating concurrent writer.
func verifyPrefix(path string, expected os.FileInfo, data []byte) error {
	return verifyPrefixWithHook(path, expected, data, nil)
}

func verifyPrefixWithHook(path string, expected os.FileInfo, data []byte, afterRead func()) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(expected, before) || before.Size() < int64(len(data)) {
		return errors.New("source identity changed or it shrank")
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
	if !os.SameFile(before, after) || after.Size() < int64(len(data)) {
		return errors.New("source changed during verification")
	}
	pathInfo, err := os.Stat(path)
	if err != nil || !os.SameFile(after, pathInfo) {
		return errors.New("source path changed during verification")
	}
	return nil
}

// replacementDeferred is a replacement that met a source still moving under
// it. Nothing was written: the prior checkpoint and evidence stand and the next
// tick stages it again.
type replacementDeferred struct{ cause error }

func (e *replacementDeferred) Error() string {
	return "replacement deferred until its source is stable: " + e.cause.Error()
}
func (e *replacementDeferred) Unwrap() error { return e.cause }

func deferReplacement(cause error) error { return &replacementDeferred{cause: cause} }

// transientReplacementReasons are refusals that said only that the source
// moved while it was staged. Checkpoints locked with one before replacements
// learned to defer are released and retried; every other refusal still needs
// an explicit reconciliation.
var transientReplacementReasons = []string{
	"replacement source changed before snapshot",
	"replacement source changed during snapshot",
	"replacement source changed during staging",
	"open replacement",
	"stat replacement",
	"read replacement",
}

// stagingLimitReason refuses a source too large to stage. A parser-version
// reimport no longer stages such a file (it keeps its older events), so a
// checkpoint locked with it whose source still extends the checkpoint was
// locked by a reimport and is released; a rewritten one stays locked.
const stagingLimitReason = "source exceeds replacement staging limit"

// releasableReplacement reports a locked replacement the current rules would
// not have locked: the source only moved while staged, or an oversized
// reimport refused a source that still extends its checkpoint.
func releasableReplacement(row *FileRow, path string, size int64) bool {
	if row.Status != StatusUnreadable {
		return false
	}
	if transientReplacement(row.Error) {
		return true
	}
	if row.Error != sourceReconcilePrefix+stagingLimitReason {
		return false
	}
	rewritten, err := sourceRewritten(path, row, size)
	return err == nil && !rewritten
}

func transientReplacement(errText string) bool {
	if !strings.HasPrefix(errText, sourceReconcilePrefix) {
		return false
	}
	reason := strings.TrimPrefix(errText, sourceReconcilePrefix)
	for _, transient := range transientReplacementReasons {
		if strings.HasPrefix(reason, transient) {
			return true
		}
	}
	return false
}

// sourceRewritten reports whether a source no longer begins with the bytes
// its checkpoint read. A head that cannot be read is an error to retry, never
// evidence of a rewrite.
func sourceRewritten(path string, row *FileRow, size int64) (bool, error) {
	if row.Offset > size {
		return true, nil
	}
	if row.Offset == 0 {
		return false, nil
	}
	if row.RootKind == RootGeminiCLI {
		full, err := FullHash(path)
		if err != nil {
			return false, fmt.Errorf("read full transcript document hash: %w", err)
		}
		return full != row.HeadHash, nil
	}
	if row.RootKind == RootCursor {
		data, err := ReadCursorSnapshot(context.Background(), path, row.SessionID)
		if err != nil {
			return false, fmt.Errorf("read Cursor composer snapshot: %w", err)
		}
		return agenttranscript.SHA256Bytes(data) != row.HeadHash, nil
	}
	if row.RootKind == RootCursor {
		data, err := ReadCursorSnapshot(context.Background(), path, row.SessionID)
		if err != nil {
			return false, fmt.Errorf("read Cursor composer snapshot: %w", err)
		}
		return agenttranscript.SHA256Bytes(data) != row.HeadHash, nil
	}
	head, err := HeadHash(path, row.Offset)
	if err != nil {
		return false, fmt.Errorf("read transcript head: %w", err)
	}
	return head != row.HeadHash, nil
}
