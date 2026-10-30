package transcriptindex_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	runnercore "agent-manager/internal/adapters/runner/core"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/pricing"
	"agent-manager/internal/transcriptindex"

	"github.com/google/uuid"
)

// An owner can commit some events before returning an error. Its event IDs
// allow a retry of the same source bytes without duplicate evidence.
type failingRecordHost struct {
	dropFailingHost
	fail      bool
	parseFail bool
	events    map[uuid.UUID]string
}

type failingCheckpointStore struct {
	transcriptindex.Store
	fail bool
}

type failingIntentStore struct{ *failingCheckpointStore }

type nonAtomicQuotaStore struct{}

func (nonAtomicQuotaStore) RecordQuotaObservation(context.Context, *pricing.QuotaObservation) error {
	return nil
}
func (nonAtomicQuotaStore) ListQuotaObservations(context.Context, string, string, string, int) ([]pricing.QuotaObservation, error) {
	return nil, nil
}

type advancingRebuildStore struct{ transcriptindex.Store }

func (s *advancingRebuildStore) CommitRecallRebuild(ctx context.Context, expected, file transcriptindex.FileRow, appendEvents func(*sql.Tx) error) error {
	owner := s.Store.(interface {
		BeginTranscriptCapture(context.Context, *transcriptindex.FileRow, transcriptindex.FileRow) error
		CommitRecallRebuild(context.Context, transcriptindex.FileRow, transcriptindex.FileRow, func(*sql.Tx) error) error
	})
	other := expected
	other.Status, other.Error = transcriptindex.StatusPending, "event recording outcome uncertain; cursor preserved: other-owner"
	if err := owner.BeginTranscriptCapture(ctx, &expected, other); err != nil {
		return err
	}
	return owner.CommitRecallRebuild(ctx, expected, file, appendEvents)
}

func TestRecallRebuildDoesNotOverwriteNewerOwnerIntent(t *testing.T) {
	r := newRig(t, 1)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	r.tick()
	row := r.row(path)
	r.now = r.now.Add(time.Hour)
	r.tick()
	store := &advancingRebuildStore{Store: r.repos.TranscriptIndex}
	index := orchestration.NewTranscriptIndex(r.svc, store, transcriptindex.Config{Roots: transcriptindex.RootConfig{Home: r.home}, Now: func() time.Time { return r.now }})
	err := index.RebuildRecall(context.Background(), row.RunID)
	if err == nil {
		t.Fatal("expected checkpoint conflict")
	}
	after := r.row(path)
	if after.Status != transcriptindex.StatusPending || !strings.Contains(after.Error, "other-owner") {
		t.Fatalf("newer intent overwritten: %+v", after)
	}
	if count, err := r.events.Count(context.Background(), uuid.MustParse(row.RunID)); err != nil || count != 0 {
		t.Fatalf("stale recall was published: %d %v", count, err)
	}
}

func (s *failingIntentStore) BeginTranscriptCapture(context.Context, *transcriptindex.FileRow, transcriptindex.FileRow) error {
	return errors.New("simulated intent failure")
}

func (s *failingIntentStore) SaveTranscriptFile(ctx context.Context, file transcriptindex.FileRow, usage []transcriptindex.UsageRow) error {
	if file.Status == transcriptindex.StatusPending && strings.Contains(file.Error, "capture interrupted") {
		return errors.New("simulated intent failure")
	}
	return s.failingCheckpointStore.SaveTranscriptFile(ctx, file, usage)
}

func TestCaptureIntentFailurePreventsEventWrites(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	r.tick()
	row := r.row(path)
	id := uuid.MustParse(row.RunID)
	before, err := r.events.Count(context.Background(), id)
	if err != nil || before == 0 {
		t.Fatalf("fixture events: %d %v", before, err)
	}
	appendLines(t, path, tokenCount(3000, 0, 300, 2000, 0, 200, 3))
	store := &failingIntentStore{&failingCheckpointStore{Store: r.repos.TranscriptIndex}}
	index := orchestration.NewTranscriptIndex(r.svc, store, transcriptindex.Config{Roots: transcriptindex.RootConfig{Home: r.home}, RecallMaxBytes: r.recall, Now: func() time.Time { return r.now }})
	if summary, err := index.RunOnce(context.Background()); err != nil || len(summary.Failures) == 0 {
		t.Fatalf("expected intent failure: %+v %v", summary, err)
	}
	after, err := r.events.Count(context.Background(), id)
	if err != nil || before != after {
		t.Fatalf("events written despite failed intent: before=%d after=%d %v", before, after, err)
	}
	if after := r.row(path); after.Offset != row.Offset {
		t.Fatalf("cursor advanced after failed intent: %+v", after)
	}
}

type rebuildFaultHost struct {
	*failingRecordHost
	promoted bool
}

type ownedFailureHost struct {
	*failingRecordHost
	path  string
	runID string
}

func (h *ownedFailureHost) OwnedTranscripts(context.Context) (map[string]string, error) {
	return map[string]string{h.path: h.runID}, nil
}

func TestRunOwnedReadFailureUsesFencedIntent(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), `{"type":"REPLAY_TEST_FAULT"}`)
	host := &ownedFailureHost{failingRecordHost: &failingRecordHost{parseFail: true}, path: path, runID: uuid.NewString()}
	index := transcriptindex.New(r.repos.TranscriptIndex, host, transcriptindex.Config{Roots: transcriptindex.RootConfig{Home: r.home}})
	if summary, err := index.RunOnce(context.Background()); err != nil || len(summary.Failures) == 0 {
		t.Fatalf("missing run-owned read failure: %+v %v", summary, err)
	}
	row := r.row(path)
	if row.Owner != transcriptindex.OwnerRun || row.Offset != 0 || !strings.Contains(row.Error, "outcome uncertain") {
		t.Fatalf("run-owned failure not retained: %+v", row)
	}
}

func TestEvictionBudgetPreservesUncertainEvidence(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	r.tick()
	row := r.row(path)
	id := uuid.MustParse(row.RunID)
	before, err := r.events.Count(context.Background(), id)
	if err != nil || before == 0 || row.RecallBytes == 0 {
		t.Fatalf("fixture has no recall: %d %+v %v", before, row, err)
	}
	row.Active, row.Status, row.Error = false, transcriptindex.StatusPending, "event recording outcome uncertain; cursor preserved: pending"
	if err := r.repos.TranscriptIndex.SaveTranscriptFile(context.Background(), row, nil); err != nil {
		t.Fatal(err)
	}
	r.now = r.now.Add(time.Hour)
	index := orchestration.NewTranscriptIndex(r.svc, r.repos.TranscriptIndex, transcriptindex.Config{Roots: transcriptindex.RootConfig{Home: r.home}, RecallMaxBytes: 1, Now: func() time.Time { return r.now }})
	summary, _ := index.RunOnce(context.Background())
	if summary.Evicted != 0 {
		t.Fatalf("uncertain evidence evicted: %+v", summary)
	}
	after, err := r.events.Count(context.Background(), id)
	if err != nil || after != before {
		t.Fatalf("uncertain events lost: before=%d after=%d %v", before, after, err)
	}
}

func TestLegacyRelinkPreservesPriorEvidenceWhenStagingFails(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	ctx := context.Background()
	taskID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("agent-manager/imported-transcripts"))
	if _, err := r.svc.CreateTask(ctx, &domain.Task{ID: taskID, Title: "Imported external transcripts", ScopePath: "."}); err != nil {
		t.Fatal(err)
	}
	legacy := &domain.Run{ID: uuid.New(), TaskID: taskID, Tag: "agent-manager-imported", RunMode: domain.RunModeInPlace, ExecutionMode: domain.ExecutionModeImported, Status: domain.RunStatusComplete, ImportSourceHarness: "codex", ImportSourceSessionID: parentID}
	legacy.TranscriptPath = filepath.Join(r.runRoot, legacy.ID.String(), "transcript.ndjson")
	write(t, legacy.TranscriptPath, sessionMeta(parentID))
	if err := r.repos.Runs.Create(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	old := domain.NewLogEvent(legacy.ID, "info", "retained evidence")
	if err := r.events.Append(ctx, legacy.ID, old); err != nil {
		t.Fatal(err)
	}
	source := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, source, sessionMeta(parentID), `{bad json`)
	if summary, err := r.index.RunOnce(ctx); err != nil || len(summary.Failures) == 0 {
		t.Fatalf("legacy staging failure not reported: %+v %v", summary, err)
	}
	run := r.run(legacy.ID.String())
	if run.TranscriptPath != legacy.TranscriptPath || run.Status != domain.RunStatusComplete {
		t.Fatalf("legacy run changed: %+v", run)
	}
	if _, err := os.Stat(legacy.TranscriptPath); err != nil {
		t.Fatalf("legacy private transcript lost: %v", err)
	}
	got, err := r.events.Get(ctx, legacy.ID, event.GetOptions{AfterSequence: -1})
	if err != nil || len(got) != 1 || got[0].ID != old.ID {
		t.Fatalf("legacy event lost: %+v %v", got, err)
	}
}

// A legacy relink may retire the private transcript, but must not remove
// unrelated artifacts that happen to share its run-state directory.
func TestLegacyRelinkPreservesUnrelatedRunArtifacts(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	ctx := context.Background()
	taskID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("agent-manager/imported-transcripts"))
	if _, err := r.svc.CreateTask(ctx, &domain.Task{ID: taskID, Title: "Imported external transcripts", ScopePath: "."}); err != nil {
		t.Fatal(err)
	}
	legacy := &domain.Run{ID: uuid.New(), TaskID: taskID, Tag: "agent-manager-imported", RunMode: domain.RunModeInPlace, ExecutionMode: domain.ExecutionModeImported, Status: domain.RunStatusComplete, ImportSourceHarness: "codex", ImportSourceSessionID: parentID}
	legacy.TranscriptPath = filepath.Join(r.runRoot, legacy.ID.String(), "transcript.ndjson")
	write(t, legacy.TranscriptPath, sessionMeta(parentID))
	artifact := filepath.Join(filepath.Dir(legacy.TranscriptPath), "unrelated.txt")
	if err := os.WriteFile(artifact, []byte("unrelated artifact"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := r.repos.Runs.Create(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	source := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, source, sessionMeta(parentID), tokenCount(100, 0, 10, 100, 0, 10, 1))
	if _, err := r.index.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if run := r.run(legacy.ID.String()); run.TranscriptPath != source {
		t.Fatalf("relinked transcript path = %s", run.TranscriptPath)
	}
	if _, err := os.Stat(legacy.TranscriptPath); err != nil {
		t.Fatalf("legacy copy must be retained until identity-safe cleanup: %v", err)
	}
	issues, err := r.index.ReplacementIssues(ctx, legacy.ID.String())
	if err != nil || len(issues.Items) != 1 || issues.Items[0].Path != legacy.TranscriptPath {
		t.Fatalf("legacy cleanup obligation not visible: %+v %v", issues, err)
	}
	r.restart()
	issues, err = r.index.ReplacementIssues(ctx, legacy.ID.String())
	if err != nil || len(issues.Items) != 1 || issues.Items[0].Path != legacy.TranscriptPath {
		t.Fatalf("legacy cleanup obligation lost on restart: %+v %v", issues, err)
	}
	data, err := os.ReadFile(artifact)
	if err != nil || string(data) != "unrelated artifact" {
		t.Fatalf("unrelated run artifact changed: %q %v", data, err)
	}
}

func TestLegacyRelinkRefusesOtherTranscriptUsage(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	ctx := context.Background()
	taskID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("agent-manager/imported-transcripts"))
	if _, err := r.svc.CreateTask(ctx, &domain.Task{ID: taskID, Title: "Imported external transcripts", ScopePath: "."}); err != nil {
		t.Fatal(err)
	}
	legacy := &domain.Run{ID: uuid.New(), TaskID: taskID, Tag: "agent-manager-imported", RunMode: domain.RunModeInPlace, ExecutionMode: domain.ExecutionModeImported, Status: domain.RunStatusComplete, ImportSourceHarness: "codex", ImportSourceSessionID: parentID}
	legacy.TranscriptPath = filepath.Join(r.runRoot, legacy.ID.String(), "transcript.ndjson")
	write(t, legacy.TranscriptPath, sessionMeta(parentID))
	if err := r.repos.Runs.Create(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	other := transcriptindex.FileRow{Key: "other-source", RunID: legacy.ID.String(), Owner: transcriptindex.OwnerRun, Status: transcriptindex.StatusOwned, FirstSeenAt: r.now}
	usage := transcriptindex.UsageRow{FileKey: other.Key, RunID: other.RunID, ObservedAt: r.now, Usage: domain.UsageEventData{InputTokens: 37}}
	if err := r.repos.TranscriptIndex.SaveTranscriptFile(ctx, other, []transcriptindex.UsageRow{usage}); err != nil {
		t.Fatal(err)
	}
	source := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, source, sessionMeta(parentID), tokenCount(100, 0, 10, 100, 0, 10, 1))
	summary, err := r.index.RunOnce(ctx)
	if err != nil || len(summary.Failures) == 0 {
		t.Fatalf("expected conflict: %+v %v", summary, err)
	}
	if got := r.run(legacy.ID.String()).TranscriptPath; got != legacy.TranscriptPath {
		t.Fatalf("legacy path changed: %s", got)
	}
	rows, err := r.repos.TranscriptIndex.TranscriptUsageForRun(ctx, legacy.ID.String())
	if err != nil || len(rows) != 1 || rows[0].Usage.InputTokens != 37 {
		t.Fatalf("usage evidence changed: %+v %v", rows, err)
	}
	if _, err := os.Stat(legacy.TranscriptPath); err != nil {
		t.Fatalf("legacy copy lost: %v", err)
	}
}

func TestLegacyRelinkPreservesUncoveredExistingEvent(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	ctx := context.Background()
	taskID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("agent-manager/imported-transcripts"))
	if _, err := r.svc.CreateTask(ctx, &domain.Task{ID: taskID, Title: "Imported external transcripts", ScopePath: "."}); err != nil {
		t.Fatal(err)
	}
	legacy := &domain.Run{ID: uuid.New(), TaskID: taskID, Tag: "agent-manager-imported", RunMode: domain.RunModeInPlace, ExecutionMode: domain.ExecutionModeImported, Status: domain.RunStatusComplete, ImportSourceHarness: "codex", ImportSourceSessionID: parentID}
	legacy.TranscriptPath = filepath.Join(r.runRoot, legacy.ID.String(), "transcript.ndjson")
	write(t, legacy.TranscriptPath, sessionMeta(parentID))
	if err := r.repos.Runs.Create(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	old := domain.NewLogEvent(legacy.ID, "info", "evidence absent from source")
	if err := r.events.Append(ctx, legacy.ID, old); err != nil {
		t.Fatal(err)
	}
	source := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, source, sessionMeta(parentID), tokenCount(100, 0, 10, 100, 0, 10, 1))
	summary, err := r.index.RunOnce(ctx)
	if err != nil || len(summary.Failures) == 0 {
		t.Fatalf("expected uncovered evidence failure: %+v %v", summary, err)
	}
	if got := r.run(legacy.ID.String()).TranscriptPath; got != legacy.TranscriptPath {
		t.Fatalf("legacy path changed: %s", got)
	}
	events, err := r.events.Get(ctx, legacy.ID, event.GetOptions{AfterSequence: -1})
	if err != nil || len(events) != 1 || events[0].ID != old.ID {
		t.Fatalf("legacy event evidence changed: %+v %v", events, err)
	}
	if _, err := os.Stat(legacy.TranscriptPath); err != nil {
		t.Fatalf("legacy copy lost: %v", err)
	}
}

func TestLegacyRelinkNeverDeletesSymlinkOrTarget(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	ctx := context.Background()
	taskID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("agent-manager/imported-transcripts"))
	if _, err := r.svc.CreateTask(ctx, &domain.Task{ID: taskID, Title: "Imported external transcripts", ScopePath: "."}); err != nil {
		t.Fatal(err)
	}
	legacy := &domain.Run{ID: uuid.New(), TaskID: taskID, Tag: "agent-manager-imported", RunMode: domain.RunModeInPlace, ExecutionMode: domain.ExecutionModeImported, Status: domain.RunStatusComplete, ImportSourceHarness: "codex", ImportSourceSessionID: parentID}
	legacy.TranscriptPath = filepath.Join(r.runRoot, legacy.ID.String(), "transcript.ndjson")
	target := filepath.Join(t.TempDir(), "other")
	if err := os.WriteFile(target, []byte("untouched"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(legacy.TranscriptPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, legacy.TranscriptPath); err != nil {
		t.Fatal(err)
	}
	if err := r.repos.Runs.Create(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	source := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, source, sessionMeta(parentID), tokenCount(100, 0, 10, 100, 0, 10, 1))
	if _, err := r.index.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if got := r.run(legacy.ID.String()).TranscriptPath; got != source {
		t.Fatalf("source was not relinked: %s", got)
	}
	info, err := os.Lstat(legacy.TranscriptPath)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("legacy symlink changed: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "untouched" {
		t.Fatalf("symlink target changed: %q %v", data, err)
	}
}

type mutationReplacementHost struct {
	dropFailingHost
	path     string
	promoted bool
}

func (h *mutationReplacementHost) Parser(context.Context, transcriptindex.FileRow) (runner.TranscriptParser, error) {
	return &mutationReplacementParser{TranscriptParser: codecs.NewCodexForTest().NewTranscriptParser(), path: h.path}, nil
}

func (h *mutationReplacementHost) ReplacementTokens(context.Context, transcriptindex.Store, transcriptindex.FileRow) (string, string, error) {
	return "file", "events", nil
}
func (h *mutationReplacementHost) CommitReplacement(context.Context, transcriptindex.Store, string, string, string, transcriptindex.FileRow, []transcriptindex.UsageRow, []*domain.RunEvent, bool, transcriptindex.RunUpdate) (bool, error) {
	h.promoted = true
	return true, nil
}

type mutationReplacementParser struct {
	runner.TranscriptParser
	path string
	done bool
}

func (p *mutationReplacementParser) ParseTranscriptLine(id uuid.UUID, line string) runner.TranscriptParseResult {
	if !p.done {
		p.done = true
		info, err := os.Stat(p.path)
		if err != nil {
			return runner.TranscriptParseResult{Err: err}
		}
		f, err := os.OpenFile(p.path, os.O_WRONLY, 0)
		if err != nil {
			return runner.TranscriptParseResult{Err: err}
		}
		_, err = f.WriteAt([]byte("X"), 1)
		if closeErr := f.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			return runner.TranscriptParseResult{Err: err}
		}
		if err := os.Chtimes(p.path, info.ModTime(), info.ModTime()); err != nil {
			return runner.TranscriptParseResult{Err: err}
		}
	}
	return p.TranscriptParser.ParseTranscriptLine(id, line)
}

func TestReplacementSourceMutationDuringStagingPreservesEvidence(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	r.tick()
	before := r.row(path)
	oldCount, err := r.events.Count(context.Background(), uuid.MustParse(before.RunID))
	if err != nil || oldCount == 0 {
		t.Fatalf("fixture events: %d %v", oldCount, err)
	}
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(2000, 0, 200, 2000, 0, 200, 2))
	host := &mutationReplacementHost{path: path}
	index := transcriptindex.New(r.repos.TranscriptIndex, host, transcriptindex.Config{Roots: transcriptindex.RootConfig{Home: r.home}, RecallMaxBytes: r.recall, Now: func() time.Time { return r.now }})
	if summary, err := index.RunOnce(context.Background()); err != nil || len(summary.Failures) == 0 {
		t.Fatalf("mutation not reported: %+v %v", summary, err)
	}
	if host.promoted {
		t.Fatal("changed source was promoted")
	}
	if after := r.row(path); after.Offset != before.Offset || after.Status != transcriptindex.StatusUnreadable {
		t.Fatalf("checkpoint lost: %+v", after)
	}
	if count, err := r.events.Count(context.Background(), uuid.MustParse(before.RunID)); err != nil || count != oldCount {
		t.Fatalf("events lost: %d %v", count, err)
	}
}

func TestReplacementSearchFailureIsDurablyRetriedAfterRestart(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	r.tick()
	before := r.row(path)
	r.svc.SetConversationSearchNotifier(func(context.Context, string, string, string) error { return errors.New("simulated search outage") })
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(2000, 0, 200, 2000, 0, 200, 2))
	summary, err := r.index.RunOnce(context.Background())
	if err != nil || len(summary.Failures) == 0 {
		t.Fatalf("missing committed-effect failure: %+v %v", summary, err)
	}
	pending := r.row(path)
	if pending.Error != transcriptindex.ReplacementEffectsPending || pending.Offset == before.Offset && pending.HeadHash == before.HeadHash {
		t.Fatalf("pending effect not durable: %+v", pending)
	}
	count, err := r.events.Count(context.Background(), uuid.MustParse(before.RunID))
	if err != nil || count == 0 {
		t.Fatalf("committed events missing: %d %v", count, err)
	}
	r.svc.SetConversationSearchNotifier(func(context.Context, string, string, string) error { return nil })
	r.restart()
	if summary, err := r.index.RunOnce(context.Background()); err != nil || len(summary.Failures) != 0 {
		t.Fatalf("effect retry failed: %+v %v", summary, err)
	}
	after := r.row(path)
	if after.Error != "" || after.Status != transcriptindex.StatusIndexed {
		t.Fatalf("effect marker not cleared: %+v", after)
	}
	if got, err := r.events.Count(context.Background(), uuid.MustParse(before.RunID)); err != nil || got != count {
		t.Fatalf("effect retry changed events: %d %v", got, err)
	}
}

func TestReplacementMissingSearchOwnerRemainsPending(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	r.tick()
	r.svc.SetConversationSearchNotifier(nil)
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(2000, 0, 200, 2000, 0, 200, 2))
	if summary, err := r.index.RunOnce(context.Background()); err != nil || len(summary.Failures) == 0 {
		t.Fatalf("missing owner not reported: %+v %v", summary, err)
	}
	if row := r.row(path); row.Error != transcriptindex.ReplacementEffectsPending {
		t.Fatalf("missing owner cleared marker: %+v", row)
	}
	r.restart()
	if summary, err := r.index.RunOnce(context.Background()); err != nil || len(summary.Failures) == 0 {
		t.Fatalf("missing owner lost after restart: %+v %v", summary, err)
	}
	if row := r.row(path); row.Error != transcriptindex.ReplacementEffectsPending {
		t.Fatalf("restart cleared unresolved marker: %+v", row)
	}
	r.svc.SetConversationSearchNotifier(func(context.Context, string, string, string) error { return nil })
	if summary, err := r.index.RunOnce(context.Background()); err != nil || len(summary.Failures) != 0 {
		t.Fatalf("owner recovery failed: %+v %v", summary, err)
	}
	if row := r.row(path); row.Error != "" {
		t.Fatalf("recovered owner left marker: %+v", row)
	}
}

func TestReplacementMissingProjectionOwnerRemainsPending(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	r.tick()
	registry := runner.NewRegistry()
	if err := registry.Register(runnercore.NewRunner(codecs.NewCodexForTest(), nil, nil)); err != nil {
		t.Fatal(err)
	}
	newService := func(withProjection bool) *orchestration.Orchestrator {
		opts := []orchestration.Option{orchestration.WithEvents(r.events), orchestration.WithRunners(registry), orchestration.WithoutQuotaObservations()}
		if withProjection {
			opts = append(opts, orchestration.WithInvocationReadModel(r.repos.InvocationReadModel))
		}
		svc := orchestration.New(r.repos.Profiles, r.repos.Tasks, r.repos.Runs, opts...)
		svc.SetConversationSearchNotifier(func(context.Context, string, string, string) error { return nil })
		return svc
	}
	r.svc = newService(false)
	r.restart()
	r.now = r.now.Add(time.Hour)
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(2000, 0, 200, 2000, 0, 200, 2))
	if summary, err := r.index.RunOnce(context.Background()); err != nil || len(summary.Failures) == 0 {
		t.Fatalf("missing projection owner not reported: %+v %v", summary, err)
	}
	if row := r.row(path); row.Error != transcriptindex.ReplacementEffectsPending {
		t.Fatalf("missing projection cleared marker: %+v", row)
	}
	r.svc = newService(true)
	r.restart()
	if summary, err := r.index.RunOnce(context.Background()); err != nil || len(summary.Failures) != 0 {
		t.Fatalf("projection recovery failed: %+v %v", summary, err)
	}
	if row := r.row(path); row.Error != "" {
		t.Fatalf("projection recovery left marker: %+v", row)
	}
}

func TestNonretainedReplacementRequiresQuotaOwnerOrExplicitDisable(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	r.tick()
	before := r.row(path)
	id := uuid.MustParse(before.RunID)
	count, err := r.events.Count(context.Background(), id)
	if err != nil || count == 0 {
		t.Fatalf("fixture events: %d %v", count, err)
	}
	registry := runner.NewRegistry()
	if err := registry.Register(runnercore.NewRunner(codecs.NewCodexForTest(), nil, nil)); err != nil {
		t.Fatal(err)
	}
	r.svc = orchestration.New(r.repos.Profiles, r.repos.Tasks, r.repos.Runs, orchestration.WithEvents(r.events), orchestration.WithRunners(registry), orchestration.WithInvocationReadModel(r.repos.InvocationReadModel))
	r.svc.SetConversationSearchNotifier(func(context.Context, string, string, string) error { return nil })
	r.restart()
	r.now = r.now.Add(31 * 24 * time.Hour)
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(2000, 0, 200, 2000, 0, 200, 2))
	summary, err := r.index.RunOnce(context.Background())
	if err != nil || len(summary.Failures) == 0 || !strings.Contains(strings.Join(summary.Failures, " "), "quota observation owner unavailable") {
		t.Fatalf("missing quota owner not reported: %+v %v", summary, err)
	}
	if got, err := r.events.Count(context.Background(), id); err != nil || got != count {
		t.Fatalf("unavailable quota owner lost old events: %d %v", got, err)
	}
	if row := r.row(path); row.Offset != before.Offset || row.HeadHash != before.HeadHash {
		t.Fatalf("unavailable quota owner changed checkpoint: %+v", row)
	}
	r.svc = orchestration.New(r.repos.Profiles, r.repos.Tasks, r.repos.Runs, orchestration.WithEvents(r.events), orchestration.WithRunners(registry), orchestration.WithInvocationReadModel(r.repos.InvocationReadModel), orchestration.WithoutQuotaObservations())
	r.svc.SetConversationSearchNotifier(func(context.Context, string, string, string) error { return nil })
	r.restart()
	if err := r.index.ReconcileReplacement(context.Background(), before.RunID, before.Key); err != nil {
		t.Fatalf("explicitly disabled quota capture did not proceed: %v", err)
	}
	if got := r.tokens(before.RunID); got != 2200 {
		t.Fatalf("replacement usage not promoted: %d", got)
	}
}

func TestReplacementRejectsNontransactionalQuotaOwner(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	r.tick()
	before := r.row(path)
	id := uuid.MustParse(before.RunID)
	count, err := r.events.Count(context.Background(), id)
	if err != nil || count == 0 {
		t.Fatalf("fixture events: %d %v", count, err)
	}
	registry := runner.NewRegistry()
	if err := registry.Register(runnercore.NewRunner(codecs.NewCodexForTest(), nil, nil)); err != nil {
		t.Fatal(err)
	}
	r.svc = orchestration.New(r.repos.Profiles, r.repos.Tasks, r.repos.Runs, orchestration.WithEvents(r.events), orchestration.WithRunners(registry), orchestration.WithInvocationReadModel(r.repos.InvocationReadModel), orchestration.WithQuotaObservationStore(nonAtomicQuotaStore{}))
	r.svc.SetConversationSearchNotifier(func(context.Context, string, string, string) error { return nil })
	r.restart()
	r.now = r.now.Add(31 * 24 * time.Hour)
	quota := strings.Replace(tokenCount(2000, 0, 200, 2000, 0, 200, 2), "1793000000", "1893000000", 1)
	write(t, path, sessionMeta(parentID), turnContext(), quota)
	summary, err := r.index.RunOnce(context.Background())
	if err != nil || len(summary.Failures) == 0 || !strings.Contains(strings.Join(summary.Failures, " "), "atomic quota observation owner unavailable") {
		t.Fatalf("nontransactional quota owner accepted: %+v %v", summary, err)
	}
	if got, err := r.events.Count(context.Background(), id); err != nil || got != count {
		t.Fatalf("quota failure lost evidence: %d %v", got, err)
	}
}

type nonAtomicHost struct{ transcriptindex.Host }

func TestMissingAtomicOwnerPreventsEventRecording(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	host := &failingRecordHost{events: map[uuid.UUID]string{}}
	index := transcriptindex.New(r.repos.TranscriptIndex, &nonAtomicHost{Host: host}, transcriptindex.Config{Roots: transcriptindex.RootConfig{Home: r.home}, Now: func() time.Time { return r.now }})
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	summary, err := index.RunOnce(context.Background())
	if err != nil || len(summary.Failures) == 0 || len(host.events) != 0 {
		t.Fatalf("missing atomic owner did not fail before recording: %+v events=%d err=%v", summary, len(host.events), err)
	}
}

func (h *rebuildFaultHost) RecallToken(context.Context, string) (string, error) { return "token", nil }

func (h *rebuildFaultHost) ReplaceRecall(context.Context, transcriptindex.Store, transcriptindex.FileRow, transcriptindex.FileRow, string, []*domain.RunEvent) (bool, error) {
	h.promoted = true
	return true, nil
}

func (s *failingCheckpointStore) CommitRecallRebuild(ctx context.Context, expected, file transcriptindex.FileRow, appendEvents func(*sql.Tx) error) error {
	if s.fail {
		s.fail = false
		return errors.New("simulated checkpoint write failure")
	}
	return s.Store.(interface {
		CommitRecallRebuild(context.Context, transcriptindex.FileRow, transcriptindex.FileRow, func(*sql.Tx) error) error
	}).CommitRecallRebuild(ctx, expected, file, appendEvents)
}

func (s *failingCheckpointStore) SaveTranscriptFile(ctx context.Context, file transcriptindex.FileRow, usage []transcriptindex.UsageRow) error {
	if s.fail && file.IndexedAt != nil && file.Status == transcriptindex.StatusIndexed {
		s.fail = false
		return errors.New("simulated checkpoint write failure")
	}
	return s.Store.SaveTranscriptFile(ctx, file, usage)
}

func (s *failingCheckpointStore) BeginTranscriptCapture(ctx context.Context, expected *transcriptindex.FileRow, intent transcriptindex.FileRow) error {
	return s.Store.(interface {
		BeginTranscriptCapture(context.Context, *transcriptindex.FileRow, transcriptindex.FileRow) error
	}).BeginTranscriptCapture(ctx, expected, intent)
}

func (s *failingCheckpointStore) SaveTranscriptIfUnchanged(ctx context.Context, expected *transcriptindex.FileRow, file transcriptindex.FileRow) error {
	if s.fail && file.IndexedAt != nil && file.Status == transcriptindex.StatusIndexed {
		s.fail = false
		return errors.New("simulated checkpoint write failure")
	}
	return s.Store.(interface {
		SaveTranscriptIfUnchanged(context.Context, *transcriptindex.FileRow, transcriptindex.FileRow) error
	}).SaveTranscriptIfUnchanged(ctx, expected, file)
}

func (s *failingCheckpointStore) FailTranscriptCapture(ctx context.Context, key, intent, status, reason string) error {
	return s.Store.(interface {
		FailTranscriptCapture(context.Context, string, string, string, string) error
	}).FailTranscriptCapture(ctx, key, intent, status, reason)
}

func (s *failingCheckpointStore) CommitTranscriptAndRun(ctx context.Context, file transcriptindex.FileRow, usage []transcriptindex.UsageRow, expected *transcriptindex.FileRow, intent string, apply func(*domain.Run) error) error {
	if s.fail && file.IndexedAt != nil && file.Status == transcriptindex.StatusIndexed {
		s.fail = false
		return errors.New("simulated checkpoint write failure")
	}
	owner, ok := s.Store.(interface {
		CommitTranscriptAndRun(context.Context, transcriptindex.FileRow, []transcriptindex.UsageRow, *transcriptindex.FileRow, string, func(*domain.Run) error) error
	})
	if !ok {
		return errors.New("atomic owner unavailable")
	}
	return owner.CommitTranscriptAndRun(ctx, file, usage, expected, intent, apply)
}

func TestCheckpointSaveFailureCannotMarkImportedRunComplete(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	r.tick()
	before := r.row(path)
	appendLines(t, path, tokenCount(3000, 0, 300, 2000, 0, 200, 3))
	r.now = r.now.Add(time.Hour)
	store := &failingCheckpointStore{Store: r.repos.TranscriptIndex, fail: true}
	index := orchestration.NewTranscriptIndex(r.svc, store, transcriptindex.Config{Roots: transcriptindex.RootConfig{Home: r.home}, RecallMaxBytes: r.recall, Now: func() time.Time { return r.now }})
	summary, err := index.RunOnce(context.Background())
	if err != nil || len(summary.Failures) == 0 {
		t.Fatalf("expected a reported checkpoint failure: %+v, %v", summary, err)
	}
	if after := r.row(path); after.Offset != before.Offset {
		t.Fatalf("failed checkpoint advanced cursor: before=%d after=%d", before.Offset, after.Offset)
	}
	if run := r.run(before.RunID); run.Status == domain.RunStatusComplete {
		t.Fatalf("checkpoint failure falsely completed import: %+v", run)
	}
	if after := r.row(path); after.Status != transcriptindex.StatusPending || !strings.Contains(after.Error, "outcome uncertain") {
		t.Fatalf("missing durable capture intent: %+v", after)
	}
	index = orchestration.NewTranscriptIndex(r.svc, store, transcriptindex.Config{Roots: transcriptindex.RootConfig{Home: r.home}, RecallMaxBytes: r.recall, Now: func() time.Time { return r.now }})
	if summary, err := index.RunOnce(context.Background()); err != nil || len(summary.Failures) == 0 {
		t.Fatalf("uncertain checkpoint was retried: %+v %v", summary, err)
	}
	if after := r.row(path); after.Status != transcriptindex.StatusUnreadable || after.Offset != before.Offset {
		t.Fatalf("uncertain checkpoint not retained for reconciliation: %+v", after)
	}
}

func TestReplacementDropFailurePreservesPriorUsageEvidence(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	r.tick()
	before := r.row(path)
	usage, err := r.repos.TranscriptIndex.TranscriptUsageForRun(context.Background(), before.RunID)
	if err != nil || len(usage) == 0 {
		t.Fatalf("fixture has no prior usage: %v, %v", usage, err)
	}
	replacement := path + ".tmp"
	write(t, replacement, sessionMeta(parentID), tokenCount(2000, 0, 200, 2000, 0, 200, 2))
	if err := os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
	host := &dropFailingHost{failDrop: true}
	index := transcriptindex.New(r.repos.TranscriptIndex, host, transcriptindex.Config{
		Roots: transcriptindex.RootConfig{Home: r.home}, RecallMaxBytes: r.recall,
		Now: func() time.Time { return r.now },
	})
	summary, err := index.RunOnce(context.Background())
	if err != nil || len(summary.Failures) == 0 {
		t.Fatalf("expected replacement to fail: %+v, %v", summary, err)
	}
	after, err := r.repos.TranscriptIndex.TranscriptUsageForRun(context.Background(), before.RunID)
	if err != nil || len(after) != len(usage) {
		t.Fatalf("failed replacement lost prior usage: before=%d after=%d err=%v", len(usage), len(after), err)
	}
	coverage, err := index.Coverage(context.Background(), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range coverage.Roots {
		for _, issue := range root.Issues {
			if issue.Path == path && strings.Contains(issue.Reason, "replacement requires reconciliation") {
				return
			}
		}
	}
	t.Fatalf("replacement reconciliation missing from coverage: %+v", coverage.Roots)
}

func TestChangedTranscriptPreservesPriorEvidenceForReconciliation(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	r.tick()
	before := r.row(path)
	id := uuid.MustParse(before.RunID)
	count, err := r.events.Count(context.Background(), id)
	if err != nil || count == 0 {
		t.Fatalf("fixture events: %d %v", count, err)
	}
	usage, err := r.repos.TranscriptIndex.TranscriptUsageForRun(context.Background(), before.RunID)
	if err != nil || len(usage) == 0 {
		t.Fatalf("fixture usage: %d %v", len(usage), err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteAt([]byte("X"), 0); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if summary, err := r.index.RunOnce(context.Background()); err != nil || len(summary.Failures) == 0 {
		t.Fatalf("expected reconciliation: %+v %v", summary, err)
	}
	after := r.row(path)
	if after.Offset != before.Offset || after.Size != before.Size || after.MtimeUnixNano != before.MtimeUnixNano || after.Status != transcriptindex.StatusUnreadable || !strings.Contains(after.Error, "replacement requires reconciliation") {
		t.Fatalf("prior checkpoint not preserved: %+v", after)
	}
	gotEvents, err := r.events.Count(context.Background(), id)
	gotUsage, usageErr := r.repos.TranscriptIndex.TranscriptUsageForRun(context.Background(), before.RunID)
	if err != nil || usageErr != nil || gotEvents != count || len(gotUsage) != len(usage) {
		t.Fatalf("prior evidence lost: events=%d usage=%d errors=%v %v", gotEvents, len(gotUsage), err, usageErr)
	}
}

func TestExplicitReplacementReconciliationFencesExactTarget(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	r.tick()
	before := r.row(path)
	if err := os.WriteFile(path, []byte("not json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if summary, err := r.index.RunOnce(context.Background()); err != nil || len(summary.Failures) == 0 {
		t.Fatalf("expected guarded replacement: %+v %v", summary, err)
	}
	issues, err := r.index.ReplacementIssues(context.Background(), before.RunID)
	if err != nil || len(issues.Items) != 1 || issues.Items[0].FileKey != before.Key || issues.Truncated {
		t.Fatalf("guard not inspectable: %+v %v", issues, err)
	}
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(2000, 0, 200, 2000, 0, 200, 2))
	if err := r.index.ReconcileReplacement(context.Background(), uuid.NewString(), before.Key); err == nil {
		t.Fatal("wrong run accepted")
	}
	if err := r.index.ReconcileReplacement(context.Background(), before.RunID, "wrong-key"); err == nil {
		t.Fatal("wrong file key accepted")
	}
	if err := r.index.ReconcileReplacement(context.Background(), before.RunID, before.Key); err != nil {
		t.Fatalf("explicit reconciliation failed: %v", err)
	}
	after := r.row(path)
	if after.Status != transcriptindex.StatusIndexed || after.RunID != before.RunID || after.Error != "" {
		t.Fatalf("replacement not reconciled: %+v", after)
	}
	if err := r.index.ReconcileReplacement(context.Background(), before.RunID, before.Key); err == nil {
		t.Fatal("unguarded checkpoint accepted")
	}
	after.Status, after.Error = transcriptindex.StatusUnreadable, "event recording outcome uncertain; cursor preserved: test"
	if err := r.repos.TranscriptIndex.SaveTranscriptFile(context.Background(), after, nil); err != nil {
		t.Fatal(err)
	}
	if err := r.index.ReconcileReplacement(context.Background(), before.RunID, after.Key); err == nil {
		t.Fatal("uncertain recording was cleared by replacement reconciliation")
	}
}

func TestRecallRebuildReadFailurePreservesExistingEvents(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	r.tick()
	row := r.row(path)
	id := uuid.MustParse(row.RunID)
	before, err := r.events.Count(context.Background(), id)
	if err != nil || before == 0 {
		t.Fatalf("fixture has no retained events: count=%d err=%v", before, err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := r.index.RebuildRecall(context.Background(), row.RunID); err == nil {
		t.Fatal("expected rebuild to report missing source")
	}
	after, err := r.events.Count(context.Background(), id)
	if err != nil || after != before {
		t.Fatalf("failed rebuild lost retained events: before=%d after=%d err=%v", before, after, err)
	}
}

func TestRecallRebuildCheckpointFailurePublishesNoEvents(t *testing.T) {
	r := newRig(t, 1)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	r.tick()
	row := r.row(path)
	r.now = r.now.Add(time.Hour)
	r.tick()
	before := r.row(path)
	if !before.RecallEvicted {
		t.Fatal("fixture did not evict recall")
	}
	store := &failingCheckpointStore{Store: r.repos.TranscriptIndex, fail: true}
	index := orchestration.NewTranscriptIndex(r.svc, store, transcriptindex.Config{Roots: transcriptindex.RootConfig{Home: r.home}, Now: func() time.Time { return r.now }})
	err := index.RebuildRecall(context.Background(), row.RunID)
	if err == nil {
		t.Fatal("expected checkpoint failure")
	}
	id := uuid.MustParse(row.RunID)
	count, err := r.events.Count(context.Background(), id)
	if err != nil || count != 0 {
		t.Fatalf("failed checkpoint published events: %d %v", count, err)
	}
	if after := r.row(path); !after.RecallEvicted || after.Offset != before.Offset {
		t.Fatalf("failed save changed checkpoint: %+v", after)
	}
}

func TestRecallRebuildEffectsRetryAfterRestart(t *testing.T) {
	r := newRig(t, 1)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	r.tick()
	row := r.row(path)
	r.now = r.now.Add(time.Hour)
	r.tick()
	r.svc.SetConversationSearchNotifier(func(context.Context, string, string, string) error { return errors.New("search unavailable") })
	err := r.index.RebuildRecall(context.Background(), row.RunID)
	if err == nil || !strings.Contains(err.Error(), "committed") {
		t.Fatalf("missing committed-effect report: %v", err)
	}
	if pending := r.row(path); pending.Error != transcriptindex.ReplacementEffectsPending {
		t.Fatalf("missing durable effect marker: %+v", pending)
	}
	id := uuid.MustParse(row.RunID)
	before, err := r.events.Count(context.Background(), id)
	if err != nil || before == 0 {
		t.Fatalf("committed recall missing: %d %v", before, err)
	}
	r.svc.SetConversationSearchNotifier(func(context.Context, string, string, string) error { return nil })
	// Avoid immediately evicting the rebuilt recall again during effect retry.
	r.recall = transcriptindex.DefaultRecallMaxBytes
	r.restart()
	if summary, err := r.index.RunOnce(context.Background()); err != nil || len(summary.Failures) != 0 {
		t.Fatalf("effect retry failed: %+v %v", summary, err)
	}
	if after := r.row(path); after.Error != "" || after.RecallEvicted {
		t.Fatalf("effect marker not cleared: %+v", after)
	}
	if after, err := r.events.Count(context.Background(), id); err != nil || after != before {
		t.Fatalf("effect retry changed recall: %d %v", after, err)
	}
}

func TestRecallRebuildRejectsChangedSource(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	r.tick()
	row := r.row(path)
	appendLines(t, path, tokenCount(2000, 0, 200, 1000, 0, 100, 2))
	if err := r.index.RebuildRecall(context.Background(), row.RunID); err == nil {
		t.Fatal("changed source accepted")
	}
	count, err := r.events.Count(context.Background(), uuid.MustParse(row.RunID))
	if err != nil || count == 0 {
		t.Fatalf("changed source lost evidence: %d %v", count, err)
	}
}

func TestRecallRebuildRejectsHeadRewriteWithUnchangedMetadata(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	r.tick()
	row := r.row(path)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteAt([]byte("X"), 0); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	err = r.index.RebuildRecall(context.Background(), row.RunID)
	if err == nil || !strings.Contains(err.Error(), "head fingerprint") {
		t.Fatalf("expected head mismatch: %v", err)
	}
	count, err := r.events.Count(context.Background(), uuid.MustParse(row.RunID))
	if err != nil || count == 0 {
		t.Fatalf("rewrite lost evidence: %d %v", count, err)
	}
}

func TestRecallRebuildRequiresValidHeadFingerprint(t *testing.T) {
	for _, hash := range []string{"", "invalid"} {
		t.Run(fmt.Sprintf("hash_%q", hash), func(t *testing.T) {
			r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
			path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
			write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
			r.tick()
			row := r.row(path)
			row.HeadHash = hash
			if err := r.repos.TranscriptIndex.SaveTranscriptFile(context.Background(), row, nil); err != nil {
				t.Fatal(err)
			}
			if err := r.index.RebuildRecall(context.Background(), row.RunID); err == nil || !strings.Contains(err.Error(), "unavailable or invalid") {
				t.Fatalf("expected unknown fingerprint: %v", err)
			}
		})
	}
}

func TestRecallRebuildAcceptsMatchingShortAndLongHeads(t *testing.T) {
	for _, padding := range []int{0, 5000} {
		t.Run(fmt.Sprintf("padding_%d", padding), func(t *testing.T) {
			r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
			path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
			write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1), fmt.Sprintf(`{"padding":"%s"}`, strings.Repeat("x", padding)))
			r.tick()
			row := r.row(path)
			if err := r.events.Delete(context.Background(), uuid.MustParse(row.RunID)); err != nil {
				t.Fatal(err)
			}
			if err := r.index.RebuildRecall(context.Background(), row.RunID); err != nil {
				t.Fatalf("matching fingerprint rejected: %v", err)
			}
			count, err := r.events.Count(context.Background(), uuid.MustParse(row.RunID))
			if err != nil || count == 0 {
				t.Fatalf("recall not rebuilt: %d %v", count, err)
			}
		})
	}
}

func TestRecallRebuildParseFailureDoesNotPromote(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1), `{"type":"REPLAY_TEST_FAULT"}`)
	r.tick()
	row := r.row(path)
	host := &rebuildFaultHost{failingRecordHost: &failingRecordHost{parseFail: true}}
	index := transcriptindex.New(r.repos.TranscriptIndex, host, transcriptindex.Config{Roots: transcriptindex.RootConfig{Home: r.home}})
	if err := index.RebuildRecall(context.Background(), row.RunID); err == nil {
		t.Fatal("expected parse failure")
	}
	if host.promoted {
		t.Fatal("parse failure promoted partial events")
	}
	count, err := r.events.Count(context.Background(), uuid.MustParse(row.RunID))
	if err != nil || count == 0 {
		t.Fatalf("parse failure lost prior events: %d %v", count, err)
	}
}

func TestUncertainReplayOfChangedSourceIsVisibleAsUnreadable(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	host := &failingRecordHost{events: map[uuid.UUID]string{}}
	index := transcriptindex.New(r.repos.TranscriptIndex, host, transcriptindex.Config{
		Roots: transcriptindex.RootConfig{Home: r.home}, RecallMaxBytes: r.recall,
		Now: func() time.Time { return r.now },
	})
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	if summary, err := index.RunOnce(context.Background()); err != nil || len(summary.Failures) != 0 {
		t.Fatalf("initial index: %+v %v", summary, err)
	}
	appendLines(t, path, tokenCount(3000, 0, 300, 2000, 0, 200, 3))
	host.fail = true
	index.RunOnce(context.Background())
	before := r.row(path)
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteAt([]byte("x"), 0); err != nil {
		t.Fatal(err)
	}
	file.Close()
	host.fail = false
	summary, _ := index.RunOnce(context.Background())
	if len(summary.Failures) == 0 {
		t.Fatal("changed source was replayed")
	}
	row := r.row(path)
	if row.Offset != before.Offset || row.Status != transcriptindex.StatusUnreadable || !strings.Contains(row.Error, "source changed") {
		t.Fatalf("missing unresolved source evidence: %+v", row)
	}
	appendLines(t, path, imageToolOutput())
	if summary, err := index.RunOnce(context.Background()); err != nil || summary.Read != 0 {
		t.Fatalf("changed source retried without reconciliation: %+v %v", summary, err)
	}
	if after := r.row(path); after.Offset != before.Offset || after.Status != transcriptindex.StatusUnreadable {
		t.Fatalf("growth cleared unresolved capture: %+v", after)
	}
	coverage, err := index.Coverage(context.Background(), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range coverage.Roots {
		for _, issue := range root.Issues {
			if issue.Path == path && strings.Contains(issue.Reason, "source changed") {
				return
			}
		}
	}
	t.Fatalf("unreadable source missing from coverage: %+v", coverage.Roots)
}

func TestUncertainRecordingDoesNotReplayRewriteBeyondHead(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	host := &failingRecordHost{events: map[uuid.UUID]string{}}
	index := transcriptindex.New(r.repos.TranscriptIndex, host, transcriptindex.Config{Roots: transcriptindex.RootConfig{Home: r.home}, Now: func() time.Time { return r.now }})
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), fmt.Sprintf(`{"padding":"%s"}`, strings.Repeat("x", 5000)), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	if summary, err := index.RunOnce(context.Background()); err != nil || len(summary.Failures) != 0 {
		t.Fatalf("initial index: %+v %v", summary, err)
	}
	before := r.row(path)
	appendLines(t, path, tokenCount(3000, 0, 300, 2000, 0, 200, 3))
	host.fail = true
	if summary, _ := index.RunOnce(context.Background()); len(summary.Failures) == 0 {
		t.Fatal("missing failed recording")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteAt([]byte("y"), 4500); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	if got := transcriptindex.HeadHash(path, before.Offset); got != before.HeadHash {
		t.Fatal("fixture unexpectedly changed the head fingerprint")
	}
	host.fail = false
	count := len(host.events)
	if summary, _ := index.RunOnce(context.Background()); len(summary.Failures) == 0 {
		t.Fatal("unverified replay was not reported")
	}
	if row := r.row(path); row.Status != transcriptindex.StatusUnreadable || row.Offset != before.Offset || len(host.events) != count {
		t.Fatalf("unverified rewrite replayed: %+v", row)
	}
}

func (h *failingRecordHost) Parser(context.Context, transcriptindex.FileRow) (runner.TranscriptParser, error) {
	return &faultParser{TranscriptParser: codecs.NewCodexForTest().NewTranscriptParser(), host: h}, nil
}

type faultParser struct {
	runner.TranscriptParser
	host *failingRecordHost
}

func (p *faultParser) SetTranscriptSession(id string, fromStart bool) {
	if setter, ok := p.TranscriptParser.(runner.TranscriptSessionSetter); ok {
		setter.SetTranscriptSession(id, fromStart)
	}
}

func (p *faultParser) ParseTranscriptLine(id uuid.UUID, line string) runner.TranscriptParseResult {
	if p.host.parseFail && strings.Contains(line, "REPLAY_TEST_FAULT") {
		return runner.TranscriptParseResult{Err: errors.New("database is locked")}
	}
	return p.TranscriptParser.ParseTranscriptLine(id, line)
}

func (h *failingRecordHost) Record(_ context.Context, _ string, events []*domain.RunEvent, _ bool) error {
	for _, event := range events {
		body, err := json.Marshal(event.Data)
		if err != nil {
			return err
		}
		if prior, exists := h.events[event.ID]; exists && prior != string(body) {
			return fmt.Errorf("event replay changed immutable payload: id=%s", event.ID)
		}
		h.events[event.ID] = string(body)
	}
	if h.fail {
		return errors.New("database is locked")
	}
	return nil
}

func TestIndexerRetainsCursorWhenEventRecordingFails(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	host := &failingRecordHost{events: map[uuid.UUID]string{}}
	index := transcriptindex.New(r.repos.TranscriptIndex, host, transcriptindex.Config{
		Roots: transcriptindex.RootConfig{Home: r.home}, RecallMaxBytes: r.recall,
		Now: func() time.Time { return r.now },
	})
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	if summary, err := index.RunOnce(context.Background()); err != nil || len(summary.Failures) != 0 {
		t.Fatalf("initial index: %+v %v", summary, err)
	}
	before := r.row(path)
	appendLines(t, path, imageToolOutput(), tokenCount(3000, 0, 300, 2000, 0, 200, 3))
	host.fail = true
	if summary, _ := index.RunOnce(context.Background()); len(summary.Failures) == 0 {
		t.Fatal("missing failed recording")
	}
	failed := r.row(path)
	if failed.Offset != before.Offset || failed.Status != transcriptindex.StatusPending || failed.UsageReports != before.UsageReports {
		t.Fatalf("failed write advanced capture: before=%+v after=%+v", before, failed)
	}
	host.fail = false
	previousEvents := len(host.events)
	if summary, err := index.RunOnce(context.Background()); err != nil || len(summary.Failures) == 0 {
		t.Fatalf("uncertain write not reported: %+v %v", summary, err)
	}
	result := r.row(path)
	if result.Offset != before.Offset || result.Status != transcriptindex.StatusUnreadable || len(host.events) != previousEvents {
		t.Fatalf("uncertain write replayed: %+v events=%d before=%d", result, len(host.events), previousEvents)
	}
}

func TestInitialRecordingFailureRequiresReconciliation(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	host := &failingRecordHost{events: map[uuid.UUID]string{}, fail: true}
	index := transcriptindex.New(r.repos.TranscriptIndex, host, transcriptindex.Config{
		Roots: transcriptindex.RootConfig{Home: r.home}, RecallMaxBytes: r.recall,
		Now: func() time.Time { return r.now },
	})
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	if summary, _ := index.RunOnce(context.Background()); len(summary.Failures) == 0 {
		t.Fatal("missing initial failure")
	}
	if row := r.row(path); row.Offset != 0 || row.Status != transcriptindex.StatusPending {
		t.Fatalf("initial cursor: %+v", row)
	}
	host.fail = false
	if summary, err := index.RunOnce(context.Background()); err != nil || len(summary.Failures) == 0 {
		t.Fatalf("initial uncertainty not reported: %+v %v", summary, err)
	}
	if row := r.row(path); row.Offset != 0 || row.Status != transcriptindex.StatusUnreadable {
		t.Fatalf("initial uncertainty replayed: %+v", row)
	}
}

func TestParseFailureAfterEventFlushRequiresReconciliation(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	host := &failingRecordHost{events: map[uuid.UUID]string{}}
	index := transcriptindex.New(r.repos.TranscriptIndex, host, transcriptindex.Config{
		Roots: transcriptindex.RootConfig{Home: r.home}, RecallMaxBytes: r.recall,
		Now: func() time.Time { return r.now },
	})
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	if summary, err := index.RunOnce(context.Background()); err != nil || len(summary.Failures) != 0 {
		t.Fatalf("initial: %+v %v", summary, err)
	}
	before := r.row(path)
	appendLines(t, path, imageToolOutput(), tokenCount(3000, 0, 300, 2000, 0, 200, 3), `{"type":"REPLAY_TEST_FAULT"}`)
	host.parseFail = true
	if summary, _ := index.RunOnce(context.Background()); len(summary.Failures) == 0 {
		t.Fatal("missing parse failure")
	}
	if row := r.row(path); row.Offset != before.Offset || row.Status != transcriptindex.StatusPending || !strings.Contains(row.Error, "outcome uncertain") {
		t.Fatalf("missing replay checkpoint: %+v", row)
	}
	host.parseFail = false
	if summary, err := index.RunOnce(context.Background()); err != nil || len(summary.Failures) == 0 {
		t.Fatalf("uncertain parse not reported: %+v %v", summary, err)
	}
	if row := r.row(path); row.Offset != before.Offset || row.Status != transcriptindex.StatusUnreadable {
		t.Fatalf("uncertain parse replayed: %+v", row)
	}
}

func TestUncertainReplayBeyondTickBudgetStaysVisible(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	host := &failingRecordHost{events: map[uuid.UUID]string{}, fail: true}
	cfg := transcriptindex.Config{Roots: transcriptindex.RootConfig{Home: r.home}, RecallMaxBytes: r.recall, Now: func() time.Time { return r.now }}
	index := transcriptindex.New(r.repos.TranscriptIndex, host, cfg)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	if summary, _ := index.RunOnce(context.Background()); len(summary.Failures) == 0 {
		t.Fatal("missing initial failure")
	}
	host.fail = false
	cfg.TickBytes = 32
	index = transcriptindex.New(r.repos.TranscriptIndex, host, cfg)
	if summary, _ := index.RunOnce(context.Background()); len(summary.Failures) == 0 {
		t.Fatal("oversize replay was not reported")
	}
	row := r.row(path)
	if row.Offset != 0 || row.Status != transcriptindex.StatusUnreadable || !strings.Contains(row.Error, "reconciliation required") {
		t.Fatalf("oversize replay: %+v", row)
	}
	appendLines(t, path, imageToolOutput())
	if summary, err := index.RunOnce(context.Background()); err != nil || summary.Read != 0 {
		t.Fatalf("unresolved replay retried: %+v %v", summary, err)
	}
}

func TestCoverageNeverTreatsUnresolvedOrChangedSourceAsCaptured(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	r.tick()
	base := r.row(path)
	root := transcriptindex.Root{Path: base.RootPath, Kind: base.RootKind, Harness: base.Harness}
	for _, tc := range []struct {
		name                        string
		mutate                      func(*transcriptindex.FileRow, *transcriptindex.Root)
		wantPending, wantUnreadable int
	}{
		{"ambiguous pending at end", func(row *transcriptindex.FileRow, _ *transcriptindex.Root) {
			row.Status, row.Error = transcriptindex.StatusPending, "event outcome uncertain"
		}, 1, 0},
		{"owned unreadable", func(row *transcriptindex.FileRow, root *transcriptindex.Root) {
			row.Status, row.Error, root.RunID = transcriptindex.StatusUnreadable, "read failed", uuid.NewString()
		}, 0, 1},
		{"changed same size", func(row *transcriptindex.FileRow, _ *transcriptindex.Root) { row.MtimeUnixNano-- }, 1, 0},
		{"owned cursor behind", func(row *transcriptindex.FileRow, root *transcriptindex.Root) {
			row.Status, row.Offset, root.RunID = transcriptindex.StatusOwned, 0, uuid.NewString()
		}, 1, 0},
		{"owned never indexed", func(row *transcriptindex.FileRow, root *transcriptindex.Root) {
			row.Status, row.IndexedAt, root.RunID = transcriptindex.StatusOwned, nil, uuid.NewString()
		}, 1, 0},
		{"unknown status", func(row *transcriptindex.FileRow, _ *transcriptindex.Root) {
			row.Status = "future-status"
		}, 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row, testRoot := base, root
			tc.mutate(&row, &testRoot)
			coverage := transcriptindex.BuildCoverage([]transcriptindex.Root{testRoot}, []transcriptindex.FileRow{row}, time.Time{}).Totals
			if coverage.Pending != tc.wantPending || coverage.Unreadable != tc.wantUnreadable || coverage.Indexed != 0 || coverage.Owned != 0 || coverage.IndexedBytes != 0 {
				t.Fatalf("unresolved source counted as captured: %+v", coverage)
			}
		})
	}
	t.Run("partial checkpoint remains visible", func(t *testing.T) {
		row := base
		row.Status, row.Offset = transcriptindex.StatusPending, base.Size/2
		coverage := transcriptindex.BuildCoverage([]transcriptindex.Root{root}, []transcriptindex.FileRow{row}, time.Time{}).Totals
		if coverage.Pending != 1 || coverage.IndexedBytes != row.Offset {
			t.Fatalf("partial checkpoint lost: %+v", coverage)
		}
	})
}
