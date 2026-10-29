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
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	runnercore "agent-manager/internal/adapters/runner/core"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/transcriptindex"

	"github.com/google/uuid"
)

// rig is a host with temp roots for every harness, real SQLite repositories
// and the real codex/claude transcript parsers.
type rig struct {
	t       *testing.T
	home    string
	runRoot string
	repos   *database.Repositories
	events  event.Store
	svc     *orchestration.Orchestrator
	index   *orchestration.TranscriptIndex
	now     time.Time
	recall  int64
}

func newRig(t *testing.T, recall int64) *rig {
	t.Helper()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	registry := runner.NewRegistry()
	for _, codec := range []runner.Runner{runnercore.NewRunner(codecs.NewCodexForTest(), nil, nil), runnercore.NewRunner(codecs.NewClaudeForTest(), nil, nil)} {
		if err := registry.Register(codec); err != nil {
			t.Fatal(err)
		}
	}
	r := &rig{t: t, home: t.TempDir(), runRoot: t.TempDir(), repos: repos, events: events, now: time.Now(), recall: recall}
	r.svc = orchestration.New(repos.Profiles, repos.Tasks, repos.Runs, orchestration.WithEvents(events), orchestration.WithRunners(registry),
		orchestration.WithRunStateRoot(r.runRoot), orchestration.WithInvocationReadModel(repos.InvocationReadModel),
		orchestration.WithConfig(orchestration.OrchestratorConfig{DefaultTimeout: time.Minute, MaxConcurrentRuns: 1}))
	r.restart()
	return r
}

// restart builds a fresh indexer over the same store, as an API restart does.
func (r *rig) restart() {
	r.index = orchestration.NewTranscriptIndex(r.svc, r.repos.TranscriptIndex, transcriptindex.Config{
		Roots:          transcriptindex.RootConfig{Home: r.home, RunStateRoot: r.runRoot},
		RecallMaxBytes: r.recall,
		Now:            func() time.Time { return r.now },
	})
}

func (r *rig) tick() transcriptindex.Summary {
	r.t.Helper()
	summary, err := r.index.RunOnce(context.Background())
	if err != nil || len(summary.Failures) > 0 {
		r.t.Fatalf("tick: %v %v", err, summary.Failures)
	}
	return summary
}

func (r *rig) files() map[string]transcriptindex.FileRow {
	r.t.Helper()
	rows, err := r.repos.TranscriptIndex.ListTranscriptFiles(context.Background())
	if err != nil {
		r.t.Fatal(err)
	}
	out := map[string]transcriptindex.FileRow{}
	for _, row := range rows {
		out[row.Path] = row
	}
	return out
}

func (r *rig) row(path string) transcriptindex.FileRow {
	r.t.Helper()
	row, ok := r.files()[path]
	if !ok {
		r.t.Fatalf("no transcript_files row for %s", path)
	}
	return row
}

func (r *rig) run(id string) *domain.Run {
	r.t.Helper()
	run, err := r.repos.Runs.Get(context.Background(), uuid.MustParse(id))
	if err != nil || run == nil {
		r.t.Fatalf("run %s: %v", id, err)
	}
	return run
}

// tokens is the one accounting function over a run's durable usage rows.
func (r *rig) tokens(runID string) int64 {
	r.t.Helper()
	rows, err := r.repos.TranscriptIndex.TranscriptUsageForRun(context.Background(), runID)
	if err != nil {
		r.t.Fatal(err)
	}
	return runsignal.AccountUsage(transcriptindex.UsageEvents(rows)).TotalTokens()
}

func (r *rig) usageRows(runID string) []transcriptindex.UsageRow {
	r.t.Helper()
	rows, err := r.repos.TranscriptIndex.TranscriptUsageForRun(context.Background(), runID)
	if err != nil {
		r.t.Fatal(err)
	}
	return rows
}

func write(t *testing.T, path string, lines ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func appendLines(t *testing.T, path string, lines ...string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.WriteString(strings.Join(lines, "\n") + "\n"); err != nil {
		t.Fatal(err)
	}
}

// Codex rollout fixture lines.
func sessionMeta(id string) string {
	return fmt.Sprintf(`{"timestamp":"2026-10-28T10:00:00Z","type":"session_meta","payload":{"id":%q}}`, id)
}

func turnContext() string {
	return `{"timestamp":"2026-10-28T10:00:01Z","type":"turn_context","payload":{"model":"gpt-6-terra"}}`
}

// tokenCount reports a cumulative counter (input incl. cached) and the
// report's own amount, with a weekly quota frame.
func tokenCount(cumInput, cumCached, cumOutput, lastInput, lastCached, lastOutput int, minute int) string {
	usage := func(input, cached, output int) string {
		return fmt.Sprintf(`{"input_tokens":%d,"cached_input_tokens":%d,"output_tokens":%d,"total_tokens":%d}`, input, cached, output, input+output)
	}
	return fmt.Sprintf(`{"timestamp":"2026-10-28T10:%02d:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":%s,"last_token_usage":%s},"rate_limits":{"primary":{"used_percent":%d,"window_minutes":10080,"resets_at":1793000000},"credits":{"has_credits":true,"balance":"62500"}}}}`,
		minute, usage(cumInput, cumCached, cumOutput), usage(lastInput, lastCached, lastOutput), 10+minute)
}

func imageToolOutput() string {
	return `{"timestamp":"2026-10-28T10:02:30Z","type":"response_item","payload":{"type":"custom_tool_call_output","call_id":"c1","output":[{"type":"input_image","detail":"high","image_url":"data:image/png;base64,AAAAAAAA"}]}}`
}

// Claude project-file fixture lines.
func claudeAssistant(session, messageID string, input, output int, sidechain bool) string {
	return fmt.Sprintf(`{"type":"assistant","sessionId":%q,"isSidechain":%t,"timestamp":"2026-10-28T11:00:00Z","message":{"id":%q,"role":"assistant","model":"claude-opus-5-5","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":%d,"output_tokens":%d}}}`, session, sidechain, messageID, input, output)
}

const (
	parentID = "01a15ea4-d9bc-7f21-b539-2e73fb96fedc"
	childID  = "01a1661a-d91f-7ca2-bb28-63ad13405a4c"
)

func codexPath(r *rig, name string) string {
	return filepath.Join(r.home, ".codex", "sessions", "2026", "10", "28", name)
}

// TestIndexerTailsGrowingFilesIncrementallyByFileKey proves the T-1 capture
// contract: a growing rollout is read from its offset each tick, its usage
// equals the provider's final cumulative counter after re-tails and a
// restart, and a Codex fork and a Claude sidechain are separate files linked
// to their parent runs instead of colliding on the parent session id.
func TestIndexerTailsGrowingFilesIncrementallyByFileKey(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	parent := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, parent, sessionMeta(parentID), turnContext(), tokenCount(1000, 600, 100, 1000, 600, 100, 1))
	r.tick()
	row := r.row(parent)
	if row.Status != transcriptindex.StatusIndexed || row.Offset != row.Size || row.UsageReports != 1 || row.SessionID != parentID {
		t.Fatalf("first tick row = %+v", row)
	}
	run := r.run(row.RunID)
	if run.ExecutionMode != domain.ExecutionModeImported || run.Status != domain.RunStatusRunning || run.TranscriptPath != parent {
		t.Fatalf("growing external file run = %+v", run)
	}
	if got := r.tokens(row.RunID); got != 1100 {
		t.Fatalf("tokens after first tick = %d, want 1100", got)
	}

	// The file grows: only the new report is read.
	appendLines(t, parent, imageToolOutput(), tokenCount(3000, 2000, 300, 2000, 1400, 200, 3))
	r.tick()
	grown := r.row(parent)
	if grown.Key != row.Key || grown.RunID != row.RunID || grown.UsageReports != 2 {
		t.Fatalf("grown row = %+v", grown)
	}
	if got := r.tokens(row.RunID); got != 3300 {
		t.Fatalf("tokens after growth = %d, want final cumulative 3300", got)
	}
	rows := r.usageRows(row.RunID)
	if facts := rows[len(rows)-1].Facts; facts.ImagesIn != 1 || facts.ImagesInHighDetail != 1 || facts.ImageBytesIn != 6 || facts.CreditsBalance != "62500" {
		t.Fatalf("non-text facts on the report that read the image = %+v", facts)
	}

	// An unchanged tick reads nothing; a restart resumes mid-file from the
	// stored offset with a fresh parser and still matches the counter.
	if summary := r.tick(); summary.Changed != 0 || summary.Read != 0 {
		t.Fatalf("unchanged tick = %+v", summary)
	}
	r.restart()
	appendLines(t, parent, tokenCount(4000, 2800, 400, 1000, 800, 100, 5))
	r.tick()
	if got := r.tokens(row.RunID); got != 4400 {
		t.Fatalf("tokens after restart re-tail = %d, want final cumulative 4400", got)
	}

	// A fork rollout names its parent thread in its header; it is its own file
	// and run, a child of the parent run.
	fork := codexPath(r, "rollout-2026-10-28T10-10-00-"+parentID+"_"+childID+".jsonl")
	write(t, fork, sessionMeta(parentID), turnContext(), tokenCount(5000, 4000, 50, 500, 300, 50, 6))
	// A Claude sub-agent sidechain links to its main session file.
	project := filepath.Join(r.home, ".claude", "projects", "-home-op-Vrooli")
	mainClaude := filepath.Join(project, "c0ffee00-0000-4000-8000-000000000001.jsonl")
	sidechain := filepath.Join(project, "c0ffee00-0000-4000-8000-000000000001", "subagents", "agent-a1b2c3.jsonl")
	write(t, sidechain, claudeAssistant("c0ffee00-0000-4000-8000-000000000001", "msg-side", 40, 4, true))
	write(t, mainClaude, claudeAssistant("c0ffee00-0000-4000-8000-000000000001", "msg-main", 70, 7, false))
	r.tick()

	forkRow := r.row(fork)
	if forkRow.Key == row.Key || forkRow.RunID == row.RunID || forkRow.SessionID != childID || forkRow.ParentSessionID != parentID || forkRow.ParentFileKey != row.Key {
		t.Fatalf("fork row = %+v (parent %+v)", forkRow, row)
	}
	if forkRun := r.run(forkRow.RunID); forkRun.ParentRunID == nil || forkRun.ParentRunID.String() != row.RunID {
		t.Fatalf("fork run parent = %v, want %s", forkRun.ParentRunID, row.RunID)
	}
	if got := r.tokens(forkRow.RunID); got != 550 {
		t.Fatalf("fork tokens = %d, want its own report 550 (not the inherited counter)", got)
	}
	if got := r.tokens(row.RunID); got != 4400 {
		t.Fatalf("parent tokens changed by the fork: %d", got)
	}
	sideRow, mainRow := r.row(sidechain), r.row(mainClaude)
	if !sideRow.Sidechain || sideRow.ParentFileKey != mainRow.Key || sideRow.RunID == mainRow.RunID {
		t.Fatalf("sidechain row = %+v main = %+v", sideRow, mainRow)
	}
	if sideRun := r.run(sideRow.RunID); sideRun.ParentRunID == nil || sideRun.ParentRunID.String() != mainRow.RunID {
		t.Fatalf("sidechain run parent = %v, want %s", sideRun.ParentRunID, mainRow.RunID)
	}
	if r.tokens(sideRow.RunID) != 44 || r.tokens(mainRow.RunID) != 77 {
		t.Fatalf("claude tokens side=%d main=%d", r.tokens(sideRow.RunID), r.tokens(mainRow.RunID))
	}
}

// A file that shrinks or is rewritten in place re-indexes from zero, and a
// file replaced under the same path (new inode) keeps its run.
func TestIndexerReindexesTruncatedRewrittenAndReplacedFiles(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1), tokenCount(2000, 0, 200, 1000, 0, 100, 2))
	r.tick()
	first := r.row(path)
	if r.tokens(first.RunID) != 2200 {
		t.Fatalf("initial tokens = %d", r.tokens(first.RunID))
	}

	// Truncated to a shorter session: the old reports are gone.
	if err := os.Truncate(path, 0); err != nil {
		t.Fatal(err)
	}
	appendLines(t, path, sessionMeta(parentID), tokenCount(700, 0, 70, 700, 0, 70, 1))
	r.tick()
	truncated := r.row(path)
	if truncated.RunID != first.RunID || truncated.UsageReports != 1 || r.tokens(first.RunID) != 770 {
		t.Fatalf("after truncation row=%+v tokens=%d", truncated, r.tokens(first.RunID))
	}

	// Replaced by a new inode at the same path (atomic rewrite).
	replacement := path + ".tmp"
	write(t, replacement, sessionMeta(parentID), tokenCount(900, 0, 90, 900, 0, 90, 1))
	if err := os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
	r.tick()
	replaced := r.row(path)
	if replaced.Key == first.Key || replaced.RunID != first.RunID || r.tokens(first.RunID) != 990 {
		t.Fatalf("after replacement row=%+v tokens=%d", replaced, r.tokens(first.RunID))
	}
	if len(r.files()) != 1 {
		t.Fatalf("replacement left %d rows, want 1", len(r.files()))
	}
}

// Agent Manager's own run files belong to their run, which records the
// events; the indexer records only their usage summaries, keyed to that run,
// so usage has one source (D-16). It never records events, quota or activity
// for them, and it reads a run-owned file registered before 2026-10-29
// (offset at its size, never indexed) once from the start.
func TestIndexerRecordsUsageOnlyForRunOwnedFiles(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	runID := uuid.New()
	homeFile := filepath.Join(r.runRoot, runID.String(), "runtime", "codex", "sessions", "2026", "10", "28", "rollout-x-"+parentID+".jsonl")
	write(t, homeFile, sessionMeta(parentID), tokenCount(1000, 0, 100, 1000, 0, 100, 1))

	// An interactive Claude run tails a file in the operator's Claude home.
	task := &domain.Task{ID: uuid.New(), Title: "interactive", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := r.repos.Tasks.Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	tailed := filepath.Join(r.home, ".claude", "projects", "p", "c0ffee00-0000-4000-8000-000000000002.jsonl")
	write(t, tailed, claudeAssistant("c0ffee00-0000-4000-8000-000000000002", "m1", 10, 1, false))
	interactive := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: "interactive-owned", RunMode: domain.RunModeInPlace, ExecutionMode: domain.ExecutionModeInteractive, Status: domain.RunStatusRunning, TranscriptPath: tailed}
	if err := r.repos.Runs.Create(context.Background(), interactive); err != nil {
		t.Fatal(err)
	}
	r.tick()
	for path, want := range map[string]struct {
		owner  string
		tokens int64
	}{homeFile: {runID.String(), 1100}, tailed: {interactive.ID.String(), 11}} {
		row := r.row(path)
		if row.Owner != transcriptindex.OwnerRun || row.Status != transcriptindex.StatusOwned || row.RunID != want.owner || row.UsageReports != 1 {
			t.Fatalf("owned row %s = %+v", path, row)
		}
		if got := r.tokens(want.owner); got != want.tokens {
			t.Fatalf("owned file %s tokens=%d want %d", path, got, want.tokens)
		}
		if count, _ := r.events.Count(context.Background(), uuid.MustParse(want.owner)); count != 0 {
			t.Fatalf("indexer recorded %d events for run-owned file %s", count, path)
		}
	}
	if run := r.run(interactive.ID.String()); run.Status != domain.RunStatusRunning {
		t.Fatalf("indexer changed the owning run: status=%s", run.Status)
	}

	r.now = r.now.Add(time.Minute)
	appendLines(t, homeFile, tokenCount(3000, 0, 300, 2000, 0, 200, 2))
	r.tick()
	if got := r.tokens(runID.String()); got != 3300 {
		t.Fatalf("growth of a run-owned file: tokens=%d want 3300", got)
	}

	// A run-owned row registered before D-16 is re-read from the start.
	legacy := r.row(homeFile)
	legacy.Offset, legacy.UsageReports, legacy.Turns, legacy.IndexedAt, legacy.HeadHash = legacy.Size, 0, 0, nil, ""
	if err := r.repos.TranscriptIndex.ResetTranscriptFile(context.Background(), legacy.Key); err != nil {
		t.Fatal(err)
	}
	if err := r.repos.TranscriptIndex.SaveTranscriptFile(context.Background(), legacy, nil); err != nil {
		t.Fatal(err)
	}
	r.restart()
	r.tick()
	if row := r.row(homeFile); row.UsageReports != 2 || r.tokens(runID.String()) != 3300 {
		t.Fatalf("legacy run-owned row after back-fill = %+v tokens=%d", row, r.tokens(runID.String()))
	}
}

// Web-console homes attribute their files and runs to the web-console session,
// which is how an attached (identity-only) run links to its transcript.
func TestIndexerLinksWebConsoleHomesToTheirSession(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	wcSession := "1e5904ed-8378-458a-86d7-8dbb039cff19"
	path := filepath.Join(transcriptindex.WebConsoleCodexBase(r.home), wcSession, "sessions", "2026", "10", "28", "rollout-x-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), tokenCount(10, 0, 1, 10, 0, 1, 1))
	write(t, filepath.Join(transcriptindex.WebConsoleCodexBase(r.home), wcSession, "session_index.jsonl"), `{"id":"x"}`)
	r.tick()
	row := r.row(path)
	if row.RootKind != transcriptindex.RootWebConsoleCodex || row.WebConsoleSessionID != wcSession || r.run(row.RunID).WebConsoleSessionID != wcSession {
		t.Fatalf("web-console row = %+v", row)
	}
	if len(r.files()) != 1 {
		t.Fatalf("session_index.jsonl outside sessions/ was indexed: %d rows", len(r.files()))
	}
}

// Recall is bounded by its own budget: evicting an idle file drops its events
// and keeps its usage summaries, the run completes once idle, and a replay
// rebuilds the events from the source file.
func TestIndexerEvictsRecallByItsOwnBudgetAndKeepsSummaries(t *testing.T) {
	r := newRig(t, 1)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	r.tick()
	row := r.row(path)
	id := uuid.MustParse(row.RunID)
	if count, _ := r.events.Count(context.Background(), id); count == 0 || row.RecallBytes == 0 {
		t.Fatalf("active file has no recall: events=%d row=%+v", count, row)
	}

	r.now = r.now.Add(time.Hour) // the file has been idle past the active window
	summary := r.tick()
	if summary.Completed != 1 || summary.Evicted != 1 {
		t.Fatalf("idle tick = %+v", summary)
	}
	if run := r.run(row.RunID); run.Status != domain.RunStatusComplete {
		t.Fatalf("idle run status = %s, want complete", run.Status)
	}
	if count, _ := r.events.Count(context.Background(), id); count != 0 {
		t.Fatalf("evicted run kept %d events", count)
	}
	if evicted := r.row(path); !evicted.RecallEvicted || evicted.RecallBytes != 0 || r.tokens(row.RunID) != 1100 {
		t.Fatalf("evicted row = %+v tokens = %d", evicted, r.tokens(row.RunID))
	}
	if _, err := r.svc.ReplayInvocationFacts(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if count, _ := r.events.Count(context.Background(), id); count == 0 {
		t.Fatal("replay did not rebuild recall from the source file")
	}
	if rows := r.usageRows(row.RunID); len(rows) != 1 {
		t.Fatalf("rebuild duplicated usage rows: %d", len(rows))
	}
}

// Coverage joins the disk with the rows: an indexed file, a run-owned file and
// a file the budget has not reached are each counted where they belong.
func TestCoverageReportsCapturedAndPendingFilesPerRoot(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	indexed := codexPath(r, "rollout-a-"+parentID+".jsonl")
	write(t, indexed, sessionMeta(parentID), tokenCount(10, 0, 1, 10, 0, 1, 1))
	owned := filepath.Join(r.runRoot, uuid.NewString(), "codex", "sessions", "rollout-b-"+childID+".jsonl")
	write(t, owned, sessionMeta(childID))
	r.tick()
	pending := filepath.Join(r.home, ".claude", "projects", "p", "c0ffee00-0000-4000-8000-000000000003.jsonl")
	write(t, pending, claudeAssistant("c0ffee00-0000-4000-8000-000000000003", "m", 1, 1, false))

	coverage, err := r.index.Coverage(context.Background(), time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	byKind := map[string]transcriptindex.RootCoverage{}
	for _, root := range coverage.Roots {
		byKind[root.Kind] = root
	}
	if c := byKind[transcriptindex.RootCodexHome]; c.Files != 1 || c.Indexed != 1 || c.WithUsage != 1 || c.IndexedBytes != c.Bytes {
		t.Fatalf("codex home coverage = %+v", c)
	}
	if c := byKind[transcriptindex.RootRunHome]; c.Files != 1 || c.Owned != 1 {
		t.Fatalf("run home coverage = %+v", c)
	}
	if c := byKind[transcriptindex.RootClaudeHome]; c.Files != 1 || c.Pending != 1 || len(c.Issues) != 1 || c.Issues[0].Path != pending {
		t.Fatalf("claude home coverage = %+v", c)
	}
	if coverage.Totals.Files != 3 {
		t.Fatalf("totals = %+v", coverage.Totals)
	}
}

// A session the retired snapshot importer adopted is relinked to its source
// file (no duplicate run) and its private transcript copy is deleted.
func TestIndexerRelinksLegacySnapshotImportsAndDeletesTheirCopy(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	ctx := context.Background()
	if _, err := r.svc.CreateTask(ctx, &domain.Task{ID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("agent-manager/imported-transcripts")), Title: "Imported external transcripts", ScopePath: "."}); err != nil {
		t.Fatal(err)
	}
	legacy := &domain.Run{
		ID: uuid.New(), TaskID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("agent-manager/imported-transcripts")), Tag: "agent-manager-imported", RunMode: domain.RunModeInPlace,
		ExecutionMode: domain.ExecutionModeImported, Status: domain.RunStatusComplete, ImportSourceHarness: "codex", ImportSourceSessionID: parentID,
	}
	legacy.TranscriptPath = filepath.Join(r.runRoot, legacy.ID.String(), "transcript.ndjson")
	write(t, legacy.TranscriptPath, sessionMeta(parentID))
	if err := r.repos.Runs.Create(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	source := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, source, sessionMeta(parentID), tokenCount(100, 0, 10, 100, 0, 10, 1))
	r.tick()
	row := r.row(source)
	if row.RunID != legacy.ID.String() {
		t.Fatalf("source file adopted run %s, want legacy run %s", row.RunID, legacy.ID)
	}
	if run := r.run(row.RunID); run.TranscriptPath != source {
		t.Fatalf("relinked transcript path = %s", run.TranscriptPath)
	}
	if _, err := os.Stat(filepath.Dir(legacy.TranscriptPath)); !os.IsNotExist(err) {
		t.Fatalf("legacy copy still on disk: %v", err)
	}
	if r.tokens(row.RunID) != 110 {
		t.Fatalf("relinked tokens = %d", r.tokens(row.RunID))
	}
}

// Configured extra roots add directories but never re-walk a built-in root.
func TestRootsDropConfiguredRootsInsideBuiltins(t *testing.T) {
	home := t.TempDir()
	extra := t.TempDir()
	roots := transcriptindex.Roots(transcriptindex.RootConfig{Home: home, ExtraRoots: []string{"~/.codex/sessions", "~/.claude/projects/p", extra, extra + "/nested", " "}})
	configured := 0
	for _, root := range roots {
		if root.Kind == transcriptindex.RootConfigured {
			configured++
			if root.Path != extra {
				t.Fatalf("unexpected configured root %s", root.Path)
			}
		}
	}
	if configured != 1 {
		t.Fatalf("configured roots = %d, want 1 (%+v)", configured, roots)
	}
}

// A back-filled usage row records when the work happened and on which model:
// observed_at is the transcript line's time (not the indexing time) and the
// model is the turn's own, for both harnesses.
func TestIndexedUsageRowsCarryLineTimeAndTurnModel(t *testing.T) {
	r := newRig(t, 0)
	rollout := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, rollout, sessionMeta(parentID), turnContext(), tokenCount(1000, 600, 10, 1000, 600, 10, 7))
	claude := filepath.Join(r.home, ".claude", "projects", "p", "c0ffee00-0000-4000-8000-000000000009.jsonl")
	write(t, claude, claudeAssistant("c0ffee00-0000-4000-8000-000000000009", "msg-1", 70, 7, false))
	r.tick()
	for path, want := range map[string]struct {
		at    string
		model string
	}{rollout: {"2026-10-28T10:07:00Z", "gpt-6-terra"}, claude: {"2026-10-28T11:00:00Z", "claude-opus-5-5"}} {
		rows := r.usageRows(r.row(path).RunID)
		if len(rows) == 0 {
			t.Fatalf("%s: no usage rows", path)
		}
		for _, row := range rows {
			if got := row.ObservedAt.UTC().Format(time.RFC3339); got != want.at || row.Usage.Model != want.model {
				t.Errorf("%s: row at %s model %q, want %s %q", filepath.Base(path), got, row.Usage.Model, want.at, want.model)
			}
		}
	}
}

// F-W2-2: non-text work read after a file's newest usage report is durable:
// a restart that resumes at the stored offset still attaches it to the next
// report instead of losing it with the in-memory parser state.
func TestIndexerKeepsWorkAfterTheNewestReportAcrossARestart(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 600, 100, 1000, 600, 100, 1), imageToolOutput())
	r.tick()
	row := r.row(path)
	if row.Offset != row.Size || row.UsageReports != 1 {
		t.Fatalf("first tick row = %+v", row)
	}
	r.restart()
	appendLines(t, path, tokenCount(3000, 2000, 300, 2000, 1400, 200, 3))
	r.tick()
	rows := r.usageRows(row.RunID)
	if len(rows) != 2 {
		t.Fatalf("usage rows = %d, want 2", len(rows))
	}
	if facts := rows[1].Facts; facts.ImagesIn != 1 || facts.ImagesInHighDetail != 1 {
		t.Fatalf("the image read before the restart is missing from the next report: %+v", facts)
	}
	if facts := rows[0].Facts; facts.ImagesIn != 0 {
		t.Fatalf("the image was attached to the earlier report: %+v", facts)
	}
}
