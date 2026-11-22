package transcriptindex_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/repository"
	"agent-manager/internal/transcriptindex"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
)

// Transcript lineage: the files that record one Codex thread (a web-console
// recovery copy, a compressed or archived rollout) share one run, and every
// usage report counts once.

// codexRollout writes rollout lines in the shape Codex writes them (copied
// from a real 0.115 rollout, content redacted): an ordinal on every record, a
// session_meta whose base instructions fill more than the 4 KiB head
// fingerprint, and per turn task_started, turn_context, the assistant
// message, a token_count with cumulative and last usage, and task_complete.
type codexRollout struct {
	thread  string
	ordinal int
	at      time.Time
	cum     [3]int // input (incl. cached), cached, output
}

func newCodexRollout(thread string) *codexRollout {
	return &codexRollout{thread: thread, at: time.Date(2026, 11, 20, 22, 20, 52, 811_000_000, time.UTC)}
}

func (c *codexRollout) record(kind, payload string) string {
	line := fmt.Sprintf(`{"timestamp":%q,"ordinal":%d,"type":%q,"payload":%s}`, c.at.Format("2006-01-02T15:04:05.000Z"), c.ordinal, kind, payload)
	c.ordinal++
	c.at = c.at.Add(37 * time.Millisecond)
	return line
}

func (c *codexRollout) meta() string {
	instructions := strings.Repeat("You are Codex, a coding agent. ", 200)
	return c.record("session_meta", fmt.Sprintf(`{"session_id":%q,"id":%q,"timestamp":%q,"cwd":"/home/operator/Vrooli","originator":"codex-tui","cli_version":"0.167.0","source":"cli","model_provider":"openai","base_instructions":{"text":%q},"history_mode":"paginated"}`,
		c.thread, c.thread, c.at.Format("2006-01-02T15:04:05.000Z"), instructions))
}

// turn is one provider request; it reports input+output tokens.
func (c *codexRollout) turn(input, cached, output int, reply string) []string {
	turnID := fmt.Sprintf("019cfe2d-472e-7d43-a2fa-%012d", c.ordinal)
	c.cum[0], c.cum[1], c.cum[2] = c.cum[0]+input, c.cum[1]+cached, c.cum[2]+output
	usage := func(in, ca, out int) string {
		return fmt.Sprintf(`{"input_tokens":%d,"cached_input_tokens":%d,"cache_write_input_tokens":0,"output_tokens":%d,"reasoning_output_tokens":0,"total_tokens":%d}`, in, ca, out, in+out)
	}
	return []string{
		c.record("event_msg", fmt.Sprintf(`{"type":"task_started","turn_id":%q,"model_context_window":258400,"collaboration_mode_kind":"default"}`, turnID)),
		c.record("turn_context", fmt.Sprintf(`{"turn_id":%q,"cwd":"/home/operator/Vrooli","current_date":"2026-11-20","timezone":"America/New_York","approval_policy":"never","model":"gpt-5.4","effort":"medium","summary":"none"}`, turnID)),
		c.record("response_item", fmt.Sprintf(`{"type":"message","role":"assistant","content":[{"type":"output_text","text":%q}],"phase":"final_answer"}`, reply)),
		c.record("event_msg", fmt.Sprintf(`{"type":"token_count","info":{"total_token_usage":%s,"last_token_usage":%s,"model_context_window":258400},"rate_limits":null}`, usage(c.cum[0], c.cum[1], c.cum[2]), usage(input, cached, output))),
		c.record("event_msg", fmt.Sprintf(`{"type":"task_complete","turn_id":%q,"last_agent_message":%q}`, turnID, reply)),
	}
}

// fork continues a copy of c from its current state, as a second writer
// resuming the same thread would.
func (c *codexRollout) fork() *codexRollout {
	copied := *c
	return &copied
}

const lineageThread = "01a1fc1f-0139-7430-9aa5-efc0cee42d31"

func rolloutName() string { return "rollout-2026-11-20T22-20-52-" + lineageThread + ".jsonl" }

// webConsoleRollout is the rollout's path in one web-console session's home.
func webConsoleRollout(r *rig, session, dir string) string {
	return filepath.Join(r.wcRoot, session, dir, "2026", "11", "20", rolloutName())
}

const (
	orphanSession    = "40a7e56e-10ff-4d62-a382-4d41e04c0553"
	recoveredSession = "8211bc9a-ec1b-46fa-8eb0-c9ca4e1b4553"
)

// copyFile copies src to dst byte for byte (a new identity), as web-console
// recovery copies a session's Codex home.
func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

// importedRuns lists the imported runs whose session is the lineage thread.
func (r *rig) importedRuns() []*domain.Run {
	r.t.Helper()
	runs, err := r.repos.Runs.List(context.Background(), repository.RunListFilter{TagPrefix: "agent-manager-imported"})
	if err != nil {
		r.t.Fatal(err)
	}
	var out []*domain.Run
	for _, run := range runs {
		if run.SessionID == lineageThread {
			out = append(out, run)
		}
	}
	return out
}

func TestRecoveredCopyContinuesItsLineageRunAndCountsUsageOnce(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	rollout := newCodexRollout(lineageThread)
	original := webConsoleRollout(r, orphanSession, "sessions")
	write(t, original, append(append([]string{rollout.meta()}, rollout.turn(13158, 9472, 44, "first")...), rollout.turn(2000, 1500, 60, "second")...)...)
	r.tick()
	first := r.row(original)

	// Recovery copies the home; Codex resumes the thread in the copy.
	copied := webConsoleRollout(r, recoveredSession, "sessions")
	copyFile(t, original, copied)
	appendLines(t, copied, rollout.turn(3000, 2500, 80, "third")...)
	r.now = r.now.Add(time.Minute)
	summary := r.tick()

	row := r.row(copied)
	if row.RunID != first.RunID {
		t.Fatalf("the copy got its own run %s instead of its lineage's %s", row.RunID, first.RunID)
	}
	if summary.Adopted != 0 || len(r.importedRuns()) != 1 {
		t.Fatalf("a copy must not adopt a second run: adopted=%d runs=%d", summary.Adopted, len(r.importedRuns()))
	}
	if want := int64(13158 + 44 + 2000 + 60 + 3000 + 80); r.tokens(first.RunID) != want {
		t.Fatalf("lineage tokens = %d, want %d (each request once)", r.tokens(first.RunID), want)
	}
	if row.Offset != row.Size || row.Turns != 1 {
		t.Fatalf("the copy must read only the turn it added: %+v", row)
	}
	if run := r.run(first.RunID); run.TranscriptPath != copied || run.Status != domain.RunStatusRunning {
		t.Fatalf("run must follow the growing copy and stay running: path=%s status=%s", run.TranscriptPath, run.Status)
	}

	// The frozen original going idle does not complete the run while the
	// copy is active; both idle completes it.
	r.now = r.now.Add(5 * time.Minute)
	if err := os.Chtimes(original, r.now.Add(-time.Hour), r.now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	r.settle(0)
	if run := r.run(first.RunID); run.Status != domain.RunStatusRunning {
		t.Fatalf("idle original completed a run its copy still carries: %s", run.Status)
	}
	r.settle(time.Hour)
	if run := r.run(first.RunID); run.Status != domain.RunStatusComplete {
		t.Fatalf("idle lineage did not complete: %s", run.Status)
	}
}

func TestDivergedCopiesCountTheirSharedTurnsOnce(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	rollout := newCodexRollout(lineageThread)
	shared := append([]string{rollout.meta()}, rollout.turn(1000, 0, 10, "shared")...)
	resumed := rollout.fork()
	original := webConsoleRollout(r, orphanSession, "sessions")
	write(t, original, append(append([]string(nil), shared...), rollout.turn(2000, 0, 20, "original went on")...)...)
	r.tick()
	run := r.row(original).RunID

	copied := webConsoleRollout(r, recoveredSession, "sessions")
	write(t, copied, append(append([]string(nil), shared...), resumed.turn(3000, 0, 30, "copy went another way")...)...)
	r.tick()
	// Every request is one provider report (turn receipts repeat their turn):
	// the shared turn once, and both continuations.
	var requested int64
	for _, row := range r.usageRows(run) {
		if !row.Usage.ReconciliationAuthority {
			requested += int64(row.Usage.InputTokens + row.Usage.CacheReadTokens + row.Usage.OutputTokens)
		}
	}
	if want := int64(1010 + 2020 + 3030); requested != want || r.row(copied).RunID != run {
		t.Fatalf("diverged copies: requested=%d want %d, copy run=%s", requested, want, r.row(copied).RunID)
	}
}

// hidingStore hides one file's row, so a copy is adopted the way it was
// before lineage existed: as its own run, read from zero.
type hidingStore struct {
	*database.TranscriptIndexRepository
	hide string
}

func (s hidingStore) ListTranscriptFiles(ctx context.Context) ([]transcriptindex.FileRow, error) {
	rows, err := s.TranscriptIndexRepository.ListTranscriptFiles(ctx)
	out := rows[:0]
	for _, row := range rows {
		if row.Path != s.hide {
			out = append(out, row)
		}
	}
	return out, err
}

func TestCopyAdoptedBeforeLineageMergesIntoTheOriginalRun(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	rollout := newCodexRollout(lineageThread)
	original := webConsoleRollout(r, orphanSession, "sessions")
	write(t, original, append(append([]string{rollout.meta()}, rollout.turn(1000, 0, 10, "one")...), rollout.turn(2000, 0, 20, "two")...)...)
	r.tick()
	run := r.row(original).RunID

	copied := webConsoleRollout(r, recoveredSession, "sessions")
	copyFile(t, original, copied)
	appendLines(t, copied, rollout.turn(3000, 0, 30, "three")...)
	r.now = r.now.Add(time.Minute)
	legacy := orchestration.NewTranscriptIndex(r.svc, hidingStore{r.repos.TranscriptIndex, original}, transcriptindex.Config{Roots: r.roots(), RecallMaxBytes: r.recall, Now: func() time.Time { return r.now }})
	if _, err := legacy.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	duplicate := r.row(copied).RunID
	if duplicate == run || r.tokens(run)+r.tokens(duplicate) != 1010+2020+1010+2020+3030 {
		t.Fatalf("fixture must reproduce the double count: runs %s/%s tokens %d+%d", run, duplicate, r.tokens(run), r.tokens(duplicate))
	}

	summary := r.tick()
	if summary.Merged != 1 || r.row(copied).RunID != run {
		t.Fatalf("copy was not merged into its lineage's run: %+v row=%+v", summary, r.row(copied))
	}
	if got, want := r.tokens(run), int64(1010+2020+3030); got != want {
		t.Fatalf("merged lineage tokens = %d, want %d", got, want)
	}
	if retired, err := r.repos.Runs.Get(context.Background(), mustUUID(t, duplicate)); err == nil && retired != nil {
		t.Fatalf("the copy's former run must be retired, still %s", retired.Status)
	}
	if len(r.importedRuns()) != 1 {
		t.Fatalf("one session must be one run, got %d", len(r.importedRuns()))
	}
	if summary := r.tick(); summary.Merged != 0 {
		t.Fatalf("merge is not idempotent: %+v", summary)
	}
}

// compress replaces path with path.zst holding the same bytes and mtime, as
// Codex does for a finished rollout.
func compress(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatal(err)
	}
	compressed := path + ".zst"
	if err := os.WriteFile(compressed, encoder.EncodeAll(data, nil), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(compressed, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	return compressed
}

func TestCompressedRolloutFinishesItsRunFromItsRealFinalBytes(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	rollout := newCodexRollout(lineageThread)
	path := filepath.Join(r.home, ".codex", "sessions", "2026", "11", "20", rolloutName())
	write(t, path, append([]string{rollout.meta()}, rollout.turn(1000, 0, 10, "one")...)...)
	r.tick()
	run := r.row(path).RunID
	// The last turn lands after the last tick, then Codex compresses it.
	appendLines(t, path, rollout.turn(2000, 0, 20, "two")...)
	old := r.now.Add(-2 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	compressed := compress(t, path)

	r.settle(time.Hour)
	row := r.row(compressed)
	if row.RunID != run || row.Offset != row.Size {
		t.Fatalf("compressed rollout did not continue its run to its end: %+v", row)
	}
	if got := r.tokens(run); got != 1010+2020 {
		t.Fatalf("tokens = %d, want both turns once", got)
	}
	r.settle(pastGrace)
	if status := r.run(run).Status; status != domain.RunStatusComplete {
		t.Fatalf("compressed rollout's run ended %s, want complete", status)
	}
	if r.row(path).Active {
		t.Fatal("the vanished uncompressed checkpoint must go idle")
	}
}

func TestRunSettledUnknownCompletesWhenItsCompressedRolloutAppears(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	rollout := newCodexRollout(lineageThread)
	path := filepath.Join(r.home, ".codex", "sessions", "2026", "11", "20", rolloutName())
	write(t, path, append([]string{rollout.meta()}, rollout.turn(1000, 0, 10, "one")...)...)
	r.tick()
	run := r.row(path).RunID
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	r.settle(pastGrace)
	if status := r.run(run).Status; status != domain.RunStatusUnknown {
		t.Fatalf("fixture: vanished source must settle unknown, got %s", status)
	}
	write(t, path, strings.TrimSuffix(string(data), "\n"))
	appendLines(t, path, rollout.turn(2000, 0, 20, "two")...)
	old := r.now.Add(-2 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	compress(t, path)
	r.settle(time.Minute)
	if status := r.run(run).Status; status != domain.RunStatusComplete || r.tokens(run) != 1010+2020 {
		t.Fatalf("run ended %s with %d tokens, want complete with both turns", status, r.tokens(run))
	}
}

func TestArchivedRolloutKeepsItsRunAndCompletes(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	rollout := newCodexRollout(lineageThread)
	live := webConsoleRollout(r, orphanSession, "sessions")
	write(t, live, append([]string{rollout.meta()}, rollout.turn(1000, 0, 10, "one")...)...)
	r.tick()
	before := r.row(live)
	appendLines(t, live, rollout.turn(2000, 0, 20, "two")...)
	archived := filepath.Join(r.wcRoot, orphanSession, "archived_sessions", rolloutName())
	if err := os.MkdirAll(filepath.Dir(archived), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(live, archived); err != nil {
		t.Fatal(err)
	}
	old := r.now.Add(-2 * time.Hour)
	if err := os.Chtimes(archived, old, old); err != nil {
		t.Fatal(err)
	}

	r.settle(time.Minute)
	row := r.row(archived)
	if row.Key != before.Key || row.RunID != before.RunID || row.Offset != row.Size {
		t.Fatalf("archived rollout must keep its identity and run and be read to its end: %+v", row)
	}
	run := r.run(before.RunID)
	if run.Status != domain.RunStatusComplete || run.TranscriptPath != archived || r.tokens(before.RunID) != 1010+2020 {
		t.Fatalf("archived run: status=%s path=%s tokens=%d", run.Status, run.TranscriptPath, r.tokens(before.RunID))
	}
}

func mustUUID(t *testing.T, id string) uuid.UUID {
	t.Helper()
	parsed, err := uuid.Parse(id)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
