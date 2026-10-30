package transcriptindex_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/domain"
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
	previousEvents := len(host.events)
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
	if summary, err := index.RunOnce(context.Background()); err != nil || len(summary.Failures) != 0 {
		t.Fatalf("replay from start failed: %+v %v", summary, err)
	}
	result := r.row(path)
	if result.Offset != result.Size || result.UsageReports != before.UsageReports+1 || len(host.events) <= previousEvents {
		t.Fatalf("replay lost evidence: %+v events=%d before=%d", result, len(host.events), previousEvents)
	}
}

func TestInitialRecordingFailureReplaysFromZero(t *testing.T) {
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
	if summary, err := index.RunOnce(context.Background()); err != nil || len(summary.Failures) != 0 {
		t.Fatalf("initial retry: %+v %v", summary, err)
	}
	if row := r.row(path); row.Offset != row.Size || row.UsageReports != 1 {
		t.Fatalf("initial replay: %+v", row)
	}
}

func TestParseFailureAfterEventFlushReplaysFromStart(t *testing.T) {
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
	if summary, err := index.RunOnce(context.Background()); err != nil || len(summary.Failures) != 0 {
		t.Fatalf("replay: %+v %v", summary, err)
	}
	if row := r.row(path); row.Offset != row.Size || row.UsageReports != 2 {
		t.Fatalf("replay not complete: %+v", row)
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
	if row.Offset != 0 || row.Status != transcriptindex.StatusUnreadable || !strings.Contains(row.Error, "tick byte budget") {
		t.Fatalf("oversize replay: %+v", row)
	}
	appendLines(t, path, imageToolOutput())
	if summary, err := index.RunOnce(context.Background()); err != nil || summary.Read != 0 {
		t.Fatalf("unresolved replay retried: %+v %v", summary, err)
	}
}
