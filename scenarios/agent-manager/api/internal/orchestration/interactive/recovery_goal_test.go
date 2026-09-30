package interactive

import (
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
	for _, variant := range []string{"consumed", "continuation", "empty", "wrong-session", "wrong-objective", "old-goal", "new-user", "new-active-goal", "partial-tail", "absent", "oversized"} {
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
			terminal, err := coord.InspectRetainedGoal(t.Context(), run)
			if variant == "empty" && err == nil {
				t.Fatal("empty extent was accepted as an unbounded replay")
			}
			if variant == "consumed" {
				if err != nil || terminal == nil || terminal.TerminalReason != "goal_complete" {
					t.Fatal("consumed accepted goal not recovered", terminal, err)
				}
			} else if terminal != nil {
				t.Fatal("unproven or superseded goal accepted", terminal)
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
