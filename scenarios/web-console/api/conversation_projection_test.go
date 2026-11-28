package main

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/agenttranscript"
	"web-console/internal/sessionstore"
)

// The rows below are real shapes captured by Web Console (texts shortened):
// a Codex turn with an injected AGENTS.md preamble, tool calls and results,
// commentary and a final answer; a Claude task notification delivered on the
// user channel; and a Codex subagent thread.
const (
	liveAgentsPreamble = "# AGENTS.md instructions for /home/matthalloran8/Vrooli\n\n<INSTRUCTIONS>\n# Working in Vrooli\n\nDeliver the operator’s requested outcome at the stated scale."
	livePrompt         = "/goal please fully implement and validate the plan at /home/matthalloran8/.vrooli/plans/desktop-release"
	liveToolCall       = "[Tool call: exec]\n\"await tools.exec_command({ cmd: \\\"sed -n '1,220p' README.md\\\" })\"\nCall ID: call_KZ0x"
	liveToolResult     = "[Tool result: tool]\n[\n  {\n    \"type\": \"input_text\",\n    \"text\": \"Script completed\"\n  }\n]\nCall ID: call_KZ0x"
	liveCommentary     = "I’m continuing from the current plan state. I’ll read the checkpoint first."
	liveFinal          = "Implemented the release flow and validated it end to end."
	liveTaskNotice     = "<task-notification>\n<task-id>bmpimwtih</task-id>\n<summary>Goal check-in: background work still running</summary>\n</task-notification>"
)

func appendLive(t *testing.T, store *ConversationStore, sessionID, source string, role ConversationRole, text string, at time.Time, class *agenttranscript.Classification) ConversationEvent {
	t.Helper()
	event, result := store.AppendClassifiedEvent(context.Background(), sessionID, source, role, text, at, class)
	if !result.Appended || result.Duplicate {
		t.Fatalf("append %q: %+v", text[:min(len(text), 30)], result)
	}
	return event
}

func TestAppendClassifiesRowsAtIngestion(t *testing.T) {
	store := NewConversationStoreWithRepository(NewSQLConversationRepository(setupTestDB(t)))
	now := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
	preamble := appendLive(t, store, "s1", codexTailerSource, ConversationRoleUser, liveAgentsPreamble, now, nil)
	call := appendLive(t, store, "s1", codexTailerSource, ConversationRoleAssistant, liveToolCall, now, nil)
	notice := appendLive(t, store, "s1", "prompt_submit_hook", ConversationRoleUser, liveTaskNotice, now, nil)
	reply := appendLive(t, store, "s1", codexTailerSource, ConversationRoleAssistant, liveFinal, now, nil)
	for _, check := range []struct {
		event        ConversationEvent
		kind, author string
	}{
		{preamble, "context", "harness"},
		{call, "tool_call", "agent"},
		{notice, "notice", "harness"},
		{reply, "message", "agent"},
	} {
		stored, ok := store.GetEvent(context.Background(), "s1", check.event.ID)
		if !ok || stored.Kind != check.kind || stored.Author != check.author {
			t.Errorf("%q stored as %s/%s, want %s/%s", check.event.Text[:20], stored.Kind, stored.Author, check.kind, check.author)
		}
	}
	if len(call.SpeechParagraphs) != 0 {
		t.Errorf("a tool call must never be prepared for playback: %v", call.SpeechParagraphs)
	}
	if len(reply.SpeechParagraphs) == 0 {
		t.Errorf("the agent's reply must be prepared for playback")
	}
	if call.Label != "exec" {
		t.Errorf("tool rows carry the tool name, got %q", call.Label)
	}
}

func TestMessagesProjectionInterleavesWorkAndPagesContiguously(t *testing.T) {
	ctx := context.Background()
	repo := NewSQLConversationRepository(setupTestDB(t))
	store := NewConversationStoreWithRepository(repo)
	start := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
	at := func(seconds int) time.Time { return start.Add(time.Duration(seconds) * time.Second) }
	subagent := &agenttranscript.Classification{
		Kind: agenttranscript.KindMessage, Author: agenttranscript.AuthorPeer,
		Thread: agenttranscript.ThreadRef{ID: "01a217a4-8fcd", Name: "Hilbert", Description: "/root/sc9_owner_fresh2", Kind: agenttranscript.ThreadSubagent},
	}
	subagentTool := *subagent
	subagentTool.Kind, subagentTool.Author = agenttranscript.KindToolCall, agenttranscript.AuthorAgent

	appendLive(t, store, "s1", codexTailerSource, ConversationRoleUser, liveAgentsPreamble, at(0), nil)                                          // 1 context
	prompt := appendLive(t, store, "s1", codexTailerSource, ConversationRoleUser, livePrompt, at(1), nil)                                        // 2 message
	appendLive(t, store, "s1", codexTailerSource, ConversationRoleAssistant, liveToolCall, at(3), nil)                                           // 3 tool_call
	appendLive(t, store, "s1", codexTailerSource, ConversationRoleUser, "Read the brief at work/SC9/BRIEF.md and report back.", at(4), subagent) // 4 thread
	appendLive(t, store, "s1", codexTailerSource, ConversationRoleAssistant, liveToolResult, at(8), nil)                                         // 5 tool_result
	appendLive(t, store, "s1", codexTailerSource, ConversationRoleAssistant, "[Tool call: exec]\n\"ls\"", at(9), &subagentTool)                  // 6 thread work
	commentary := appendLive(t, store, "s1", codexTailerSource, ConversationRoleAssistant, liveCommentary, at(10), nil)                          // 7 message
	appendLive(t, store, "s1", codexTailerSource, ConversationRoleAssistant, "[Tool call: exec]\n\"make test\"", at(12), nil)                    // 8 tool_call
	final := appendLive(t, store, "s1", codexTailerSource, ConversationRoleAssistant, liveFinal, at(30), nil)                                    // 9 message
	appendLive(t, store, "s1", codexTailerSource, ConversationRoleAssistant, "[Tool call: exec]\n\"git status\"", at(31), nil)                   // 10 trailing work

	newest, err := repo.ListProjectionPage(ctx, "s1", ConversationProjectionQuery{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	kinds := projectionShape(newest.State.Events)
	if got, want := kinds, "work[3-6 c1 r1]|message#7|work[8-8 c1 r0]|message#9|work[10-10 c1 r0]"; got != want {
		t.Fatalf("newest page = %s, want %s", got, want)
	}
	if !newest.HasMore || newest.OldestSequence != 3 {
		t.Fatalf("newest page hasMore=%v oldest=%d", newest.HasMore, newest.OldestSequence)
	}
	if newest.State.Events[1].ID != commentary.ID || newest.State.Events[3].ID != final.ID {
		t.Fatalf("conversation rows out of order")
	}
	if len(newest.Threads) != 1 || newest.Threads[0].Label != "Hilbert" || newest.Threads[0].MessageCount != 1 || newest.Threads[0].WorkCount != 1 || newest.Threads[0].FirstSequence != 4 {
		t.Fatalf("threads = %+v", newest.Threads)
	}

	older, err := repo.ListProjectionPage(ctx, "s1", ConversationProjectionQuery{Limit: 2, BeforeSequence: commentary.Sequence})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := projectionShape(older.State.Events), "work[1-1 c0 r0 x1]|message#2"; got != want {
		t.Fatalf("older page = %s, want %s", got, want)
	}
	if older.HasMore || older.OldestSequence != 1 || older.State.Events[1].ID != prompt.ID {
		t.Fatalf("older page hasMore=%v oldest=%d", older.HasMore, older.OldestSequence)
	}
	// The span between the prompt and the commentary covers 3..6 (the
	// subagent's rows included) but counts and times only this thread's work.
	span := newest.State.Events[0].Work
	if span.StartedAt != at(3) || span.EndedAt != at(8) {
		t.Fatalf("span time %s..%s", span.StartedAt, span.EndedAt)
	}
	// Contiguity: every sequence from 1 to the newest is covered exactly once.
	assertContiguous(t, append(older.State.Events, newest.State.Events...), 10)

	thread, err := repo.ListProjectionPage(ctx, "s1", ConversationProjectionQuery{ThreadID: "01a217a4-8fcd", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := projectionShape(thread.State.Events), "work[1-3 c0 r0]|message#4|work[5-10 c1 r0]"; got != want {
		t.Fatalf("thread page = %s, want %s", got, want)
	}
	if thread.State.Events[1].Author != "peer" {
		t.Fatalf("a subagent's brief is the parent's, got %q", thread.State.Events[1].Author)
	}

	work, err := repo.ListWorkRange(ctx, "s1", "", 3, 6, 40)
	if err != nil {
		t.Fatal(err)
	}
	if len(work) != 2 || work[0].Kind != "tool_call" || work[1].Kind != "tool_result" || !work[1].TextTruncated || len([]rune(work[1].Text)) != 40 {
		t.Fatalf("work range = %+v", work)
	}

	since, err := repo.ListProjectionPage(ctx, "s1", ConversationProjectionQuery{SinceSequence: 7})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := projectionShape(since.State.Events), "work[8-8 c1 r0]|message#9|work[10-10 c1 r0]"; got != want {
		t.Fatalf("since page = %s, want %s", got, want)
	}
	// A since read cut by its limit stops at its last row: what it did not
	// return is not work, so it carries no tail span.
	cut, err := repo.ListProjectionPage(ctx, "s1", ConversationProjectionQuery{SinceSequence: 2, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := projectionShape(cut.State.Events), "work[3-6 c1 r1]|message#7"; got != want || !cut.HasMore || len(cut.Threads) != 0 {
		t.Fatalf("cut since page = %s (hasMore=%v), want %s", got, cut.HasMore, want)
	}
}

func TestMemoryProjectionMatchesSQL(t *testing.T) {
	ctx := context.Background()
	sqlRepo := NewSQLConversationRepository(setupTestDB(t))
	memRepo := NewInMemoryConversationRepository()
	for _, repo := range []ConversationRepository{sqlRepo, memRepo} {
		store := NewConversationStoreWithRepository(repo)
		now := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
		appendLive(t, store, "s1", codexTailerSource, ConversationRoleUser, livePrompt, now, nil)
		appendLive(t, store, "s1", codexTailerSource, ConversationRoleAssistant, liveToolCall, now.Add(time.Second), nil)
		appendLive(t, store, "s1", codexTailerSource, ConversationRoleAssistant, liveFinal, now.Add(2*time.Second), nil)
	}
	sqlPage, err := sqlRepo.ListProjectionPage(ctx, "s1", ConversationProjectionQuery{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	memPage, err := memRepo.ListProjectionPage(ctx, "s1", ConversationProjectionQuery{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if projectionShape(sqlPage.State.Events) != projectionShape(memPage.State.Events) {
		t.Fatalf("sql %s != memory %s", projectionShape(sqlPage.State.Events), projectionShape(memPage.State.Events))
	}
	sqlCut, _ := sqlRepo.ListProjectionPage(ctx, "s1", ConversationProjectionQuery{SinceSequence: 1, Limit: 1})
	memCut, _ := memRepo.ListProjectionPage(ctx, "s1", ConversationProjectionQuery{SinceSequence: 1, Limit: 1})
	if projectionShape(sqlCut.State.Events) != projectionShape(memCut.State.Events) || sqlCut.HasMore != memCut.HasMore {
		t.Fatalf("cut: sql %s != memory %s", projectionShape(sqlCut.State.Events), projectionShape(memCut.State.Events))
	}
}

func projectionShape(events []ConversationEvent) string {
	parts := make([]string, 0, len(events))
	for _, event := range events {
		if event.Kind == conversationKindWork {
			work := event.Work
			shape := "work[" + itoa(work.FromSequence) + "-" + itoa(work.ToSequence) + " c" + itoa(int64(work.ToolCalls)) + " r" + itoa(int64(work.ToolResults))
			if work.Context > 0 {
				shape += " x" + itoa(int64(work.Context))
			}
			parts = append(parts, shape+"]")
			continue
		}
		parts = append(parts, event.Kind+"#"+itoa(event.Sequence))
	}
	return strings.Join(parts, "|")
}

func itoa(value int64) string { return strconv.FormatInt(value, 10) }

func assertContiguous(t *testing.T, events []ConversationEvent, through int64) {
	t.Helper()
	covered := map[int64]int{}
	for _, event := range events {
		from := event.Sequence
		if event.Work != nil {
			from = event.Work.FromSequence
		}
		for sequence := from; sequence <= event.Sequence; sequence++ {
			covered[sequence]++
		}
	}
	for sequence := int64(1); sequence <= through; sequence++ {
		if covered[sequence] != 1 {
			t.Fatalf("sequence %d covered %d times", sequence, covered[sequence])
		}
	}
}

func TestClassificationBackfillIsIdempotentAndAttributesLaunchPrompts(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	handle := db
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS web_console_migrations (name TEXT PRIMARY KEY, last_rowid INTEGER NOT NULL DEFAULT 0, completed_at TEXT NOT NULL DEFAULT '')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sessions (id, origin, owner) VALUES ('launched', 'programmatic', 'research-desk'), ('typed', 'ui', '')`); err != nil {
		t.Fatal(err)
	}
	// Rows as they were stored before classification: every row a message
	// with no author, exactly what ALTER TABLE ... DEFAULT 'message' leaves.
	repo := NewSQLConversationRepository(handle)
	legacy := []struct{ session, source, role, text string }{
		{"launched", codexTailerSource, "user", liveAgentsPreamble},
		{"launched", codexTailerSource, "user", "You are the portfolio steward. Read /home/matthalloran8/.vrooli/portfolio/steward/CHARTER.md and follow it."},
		{"launched", codexTailerSource, "assistant", liveToolCall},
		{"launched", codexTailerSource, "assistant", liveToolResult},
		{"launched", codexTailerSource, "user", "Thanks, keep going."},
		{"typed", "claude_archive", "user", "Another Claude session sent a message:\n<teammate-message teammate_id=\"serving-fabric\" color=\"blue\" summary=\"Ask lead\">\nWS-F question"},
		{"typed", "prompt_submit_hook", "user", liveTaskNotice},
		{"typed", "claude_hook", "assistant", liveFinal},
		{"typed", "prompt_submit_hook", "user", livePrompt},
	}
	for index, row := range legacy {
		event := ConversationEvent{ID: "legacy-" + itoa(int64(index)), SessionID: row.session, Source: row.source, Role: ConversationRole(row.role), Text: row.text,
			CreatedAt: time.Date(2026, 11, 20, 10, 0, index, 0, time.UTC), DeliveryState: ConversationDeliveryPending, TTSState: ConversationTTSIdle, ConsumptionState: ConversationConsumptionUnseen}
		if _, err := repo.AppendEvent(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	backfill := newConversationClassificationBackfill(handle)
	backfill.batch, backfill.pause = 3, 0
	report, err := backfill.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if report.Scanned != int64(len(legacy)) || report.ByKind["tool_call"] != 1 || report.ByKind["tool_result"] != 1 || report.ByKind["context"] != 1 || report.ByKind["notice"] != 1 || report.LaunchPrompts != 1 {
		t.Fatalf("report = %+v", report)
	}
	expect := map[string]string{
		"legacy-0": "context/harness/", "legacy-1": "message/automation/research-desk", "legacy-2": "tool_call/agent/",
		"legacy-3": "tool_result/agent/", "legacy-4": "message/human/", "legacy-5": "message/peer/serving-fabric",
		"legacy-6": "notice/harness/", "legacy-7": "message/agent/", "legacy-8": "message/human/",
	}
	for id, want := range expect {
		session := "launched"
		if id >= "legacy-5" {
			session = "typed"
		}
		event, ok, err := repo.GetEvent(ctx, session, id)
		if err != nil || !ok {
			t.Fatalf("%s: %v", id, err)
		}
		if got := event.Kind + "/" + event.Author + "/" + event.AuthorName; got != want {
			t.Errorf("%s = %s, want %s", id, got, want)
		}
	}
	teammate, _, _ := repo.GetEvent(ctx, "typed", "legacy-5")
	if teammate.ThreadID != "teammate:serving-fabric" {
		t.Errorf("a teammate's message is filed under its thread, got %q", teammate.ThreadID)
	}
	again, err := backfill.Run(ctx)
	if err != nil || again.Scanned != 0 || again.Changed != 0 {
		t.Fatalf("second run must be a no-op: %+v %v", again, err)
	}
	var indexCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_conversation_events_projection'`).Scan(&indexCount); err != nil || indexCount != 1 {
		t.Fatalf("projection index missing: %d %v", indexCount, err)
	}
}

func TestLaunchPromptAttributionAtIngestion(t *testing.T) {
	db := setupTestDB(t)
	handle := db
	sessions := sessionstore.NewSQL(handle)
	if _, err := db.Exec(`INSERT INTO sessions (id, origin, owner) VALUES ('launched', 'programmatic', 'agent-manager')`); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLConversationRepository(handle)
	store := NewConversationStoreWithRepository(repo)
	store.launchPrompts = newSessionLaunchPrompts(sessions, repo)
	now := time.Now()
	appendLive(t, store, "launched", codexTailerSource, ConversationRoleUser, liveAgentsPreamble, now, nil)
	first := appendLive(t, store, "launched", codexTailerSource, ConversationRoleUser, "Read and execute .vrooli/work/operator-steering/briefs/G.md completely.", now, nil)
	second := appendLive(t, store, "launched", codexTailerSource, ConversationRoleUser, "Continue please", now, nil)
	if first.Author != "automation" || first.AuthorName != "agent-manager" || first.Label != "launch_prompt" {
		t.Fatalf("launch prompt = %+v", first)
	}
	if second.Author != "human" {
		t.Fatalf("later prompts stay the human's, got %q", second.Author)
	}
}

func TestCodexSubagentRolloutRowsLandInTheirThread(t *testing.T) {
	meta, err := os.ReadFile(filepath.Join(findRepoRoot(t), "packages", "cli-core", "agenttranscript", "testdata", "message-classes", "codex-session-meta.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(meta)), "\n")
	rollout := filepath.Join(t.TempDir(), "rollout-2026-11-26T06-36-30-01a217a4-8fcd-7b60-b228-fc99db6fc8d8.jsonl")
	if err := os.WriteFile(rollout, []byte(lines[1]+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	source := codexTranscriptSource{tailer: &CodexTailer{threads: &transcriptThreads{}}}
	brief := `{"timestamp":"2026-11-26T11:36:31.000Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"Own SC9 end to end. Read work/SC9/BRIEF.md first."}]}}`
	events, err := source.DecodeLine(rollout, []byte(brief))
	if err != nil || len(events) != 1 {
		t.Fatalf("decode: %v %v", events, err)
	}
	class := events[0].Class
	if class == nil || class.Thread.Key() != "01a217a4-8fcd-7b60-b228-fc99db6fc8d8" || class.Thread.Name != "Hilbert" || class.Author != agenttranscript.AuthorPeer {
		t.Fatalf("subagent brief classification = %+v", class)
	}
}
