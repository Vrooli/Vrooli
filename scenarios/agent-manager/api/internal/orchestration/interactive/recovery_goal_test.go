package interactive

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

func TestRetainedNativeGoalReplayIsBoundedExactAndDoesNotReemit(t *testing.T) {
	for _, variant := range []string{"consumed", "normalized-multiline", "continuation", "empty", "wrong-session", "wrong-objective", "old-goal", "new-user", "new-active-goal", "partial-tail", "absent", "oversized"} {
		t.Run(variant, func(t *testing.T) {
			session := "native-session-fixture"
			path := filepath.Join(t.TempDir(), "rollout.jsonl")
			body := fmt.Sprintf("{\"timestamp\":\"2026-09-12T10:00:00Z\",\"type\":\"session_meta\",\"payload\":{\"id\":%q}}\n", session) +
				"{\"timestamp\":\"2026-09-12T10:00:02Z\",\"type\":\"event_msg\",\"payload\":{\"type\":\"thread_goal_updated\",\"goal\":{\"objective\":\"finish assignment\",\"status\":\"complete\"}}}\n" +
				"{\"timestamp\":\"2026-09-12T10:00:03Z\",\"type\":\"event_msg\",\"payload\":{\"type\":\"task_complete\",\"last_agent_message\":\"done\"}}\n"
			run := newInteractiveRun(domain.RunnerTypeCodex, path)
			run.SessionID, run.ResolvedConfig.Until = session, "finish assignment"
			started, _ := time.Parse(time.RFC3339, "2026-09-12T10:00:00Z")
			run.StartedAt = &started
			switch variant {
			case "normalized-multiline":
				run.ResolvedConfig.Until = "finish\r\nassignment"
			case "continuation":
				currentAttempt := started.Add(time.Hour)
				run.InteractiveInvocationStartedAt = &currentAttempt
			case "empty":
				body = ""

			case "wrong-session":
				run.SessionID = "other"
			case "wrong-objective":
				run.ResolvedConfig.Until = "different"
			case "old-goal":
				started = started.Add(time.Hour)
			case "new-user":
				body += "{\"timestamp\":\"2026-09-12T10:00:04Z\",\"type\":\"event_msg\",\"payload\":{\"type\":\"user_message\",\"message\":\"more work\"}}\n"
			case "new-active-goal":
				body += "{\"timestamp\":\"2026-09-12T10:00:04Z\",\"type\":\"event_msg\",\"payload\":{\"type\":\"thread_goal_updated\",\"goal\":{\"objective\":\"finish assignment\",\"status\":\"active\"}}}\n"
			case "partial-tail":
				body += "{\"type\":"
			case "oversized":
				body = strings.Repeat("x", (16<<20)+1)
			}
			if variant != "absent" {
				if err := os.WriteFile(path, []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
			}
			run.TranscriptCursor = int64(len(body))
			store, sink := &fakeRunStore{}, &collectSink{}
			coord := NewCoordinator(CoordinatorDeps{Tailer: NewTailer(codecParserResolver), Runs: store, NewSink: func(uuid.UUID) runner.EventSink { return sink }})
			evidence, err := coord.InspectRetainedGoal(t.Context(), run)
			if variant == "empty" && err == nil {
				t.Fatal("empty extent was accepted as an unbounded replay")
			}
			if variant == "consumed" || variant == "normalized-multiline" {
				if err != nil || evidence == nil || evidence.Terminal.TerminalReason != "goal_complete" || evidence.Marker.Status != runner.GoalStatusComplete || evidence.Marker.Objective != "finish assignment" {
					t.Fatal("consumed accepted goal not recovered", evidence, err)
				}
			} else if evidence != nil {
				t.Fatal("unproven or superseded goal accepted", evidence)
			}
			if store.updates != 0 || run.TranscriptCursor != int64(len(body)) {
				t.Fatal("read-only replay altered run state")
			}
			if len(sink.events) != 0 {
				t.Fatal("historical events re-emitted")
			}
		})
	}
}

func TestInspectRetainedGoalAcceptsOnlyCurrentPinnedReadAfterContinuation(t *testing.T) {
	const (
		session   = "a173-native-thread"
		callID    = "a173-get-goal"
		objective = "finish the assigned work"
	)
	for _, tc := range []struct {
		name         string
		read         bool
		request      string
		thread       string
		observed     string
		failed       bool
		wantEvidence bool
	}{
		{name: "current-pinned-read", read: true, request: "const r = await tools.get_goal({}); text(JSON.stringify(r));", thread: session, observed: objective, wantEvidence: true},
		{name: "wrong-thread", read: true, request: "const r = await tools.get_goal({}); text(JSON.stringify(r));", thread: "another-thread", observed: objective},
		{name: "wrong-objective", read: true, request: "const r = await tools.get_goal({}); text(JSON.stringify(r));", thread: session, observed: "different work"},
		{name: "old-only-completion"},
		{name: "other-variable-output", read: true, request: "const r = await tools.get_goal({}); text(JSON.stringify(other));", thread: session, observed: objective},
		{name: "failed-read", read: true, request: "const r = await tools.get_goal({}); text(JSON.stringify(r));", thread: session, observed: objective, failed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			marshal := func(v any) string {
				t.Helper()
				b, err := json.Marshal(v)
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}
			path := filepath.Join(t.TempDir(), "rollout.jsonl")
			started, _ := time.Parse(time.RFC3339, "2026-10-17T10:00:00Z")
			invocation := started.Add(time.Minute)
			lines := []string{
				marshal(map[string]any{"timestamp": started.Format(time.RFC3339), "type": "session_meta", "payload": map[string]any{"id": session}}),
				marshal(map[string]any{"timestamp": started.Add(time.Second).Format(time.RFC3339), "type": "event_msg", "payload": map[string]any{"type": "thread_goal_updated", "goal": map[string]any{"threadId": session, "objective": objective, "status": "active"}}}),
				marshal(map[string]any{"timestamp": started.Add(2 * time.Second).Format(time.RFC3339), "type": "event_msg", "payload": map[string]any{"type": "thread_goal_updated", "goal": map[string]any{"threadId": session, "objective": objective, "status": "complete"}}}),
			}
			if tc.read {
				lines = append(lines,
					marshal(map[string]any{"timestamp": invocation.Add(time.Second).Format(time.RFC3339), "type": "event_msg", "payload": map[string]any{"type": "user_message", "message": "continue the assigned work"}}),
					marshal(map[string]any{"timestamp": invocation.Add(2 * time.Second).Format(time.RFC3339), "type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": callID, "input": tc.request}}),
				)
				receipt, err := json.Marshal(map[string]any{"goal": map[string]any{"threadId": tc.thread, "objective": tc.observed, "status": "complete"}})
				if err != nil {
					t.Fatal(err)
				}
				payload := map[string]any{"type": "custom_tool_call_output", "call_id": callID, "output": []map[string]string{{"type": "input_text", "text": string(receipt)}}}
				if tc.failed {
					payload["isError"] = true
				}
				lines = append(lines, marshal(map[string]any{"timestamp": invocation.Add(3 * time.Second).Format(time.RFC3339), "type": "response_item", "payload": payload}))
			}
			body := strings.Join(lines, "\n") + "\n"
			if err := os.WriteFile(path, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
			run := newInteractiveRun(domain.RunnerTypeCodex, path)
			run.SessionID, run.ResolvedConfig.Until = session, "finish\r\nthe assigned work"
			run.StartedAt = &started
			run.InteractiveInvocationStartedAt = &invocation
			run.TranscriptCursor = int64(len(body))
			store, sink := &fakeRunStore{}, &collectSink{}
			coord := NewCoordinator(CoordinatorDeps{Tailer: NewTailer(codecParserResolver), Runs: store, NewSink: func(uuid.UUID) runner.EventSink { return sink }})
			evidence, err := coord.InspectRetainedGoal(t.Context(), run)
			if err != nil {
				t.Fatalf("InspectRetainedGoal: %v", err)
			}
			if (evidence != nil) != tc.wantEvidence {
				t.Fatalf("evidence=%+v, wantEvidence=%v", evidence, tc.wantEvidence)
			}
			if tc.wantEvidence && (evidence.Marker.Status != runner.GoalStatusComplete || evidence.Marker.Objective != tc.observed || evidence.Terminal.TerminalReason != "goal_complete" || !evidence.At.Equal(invocation.Add(3*time.Second))) {
				t.Fatalf("evidence=%+v, want exact post-invocation read", evidence)
			}
			if store.updates != 0 || len(sink.events) != 0 || run.TranscriptCursor != int64(len(body)) {
				t.Fatal("retained inspection replayed effects or altered run state")
			}
		})
	}
}
