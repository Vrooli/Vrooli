package codecs

import (
	"strings"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// forkedRollout is a forked rollout in the shape Codex writes (0.166): the
// fork's session_meta, then a copy of the parent's history re-stamped at the
// fork (the parent's session_meta, its compacted record, its turns with a
// call, a user message and their terminals), then the fork's own lines.
// boundary is the fork's first line: thread_settings_applied naming the fork,
// or, when absent, a turn begun after the fork; forkTurn is the id of the
// fork's first turn.
func forkedRollout(boundary, forkTurn string) []string {
	const fork, parent = "01a18b01-6939-7482-a405-898d1a33c930", "01a18a49-0feb-7ed3-b314-1d586a775e2b"
	return []string{
		`{"timestamp":"2026-10-30T04:11:28.128Z","type":"session_meta","payload":{"id":"` + fork + `","session_id":"` + parent + `","forked_from_id":"` + parent + `","parent_thread_id":"` + parent + `","thread_source":"subagent","source":{"subagent":{"thread_spawn":{"parent_thread_id":"` + parent + `","depth":1}}}}}`,
		`{"timestamp":"2026-10-30T04:11:28.130Z","type":"session_meta","payload":{"id":"` + parent + `","session_id":"` + parent + `","thread_source":"user"}}`,
		`{"timestamp":"2026-10-30T04:11:28.130Z","type":"compacted","payload":{"message":"","replacement_history":[]}}`,
		`{"timestamp":"2026-10-30T04:11:28.137Z","type":"turn_context","payload":{"turn_id":"01a18ae0-da1f-7011-a37d-9eaacb06b4aa","model":"gpt-parent"}}`,
		`{"timestamp":"2026-10-30T04:11:28.138Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"parent prompt"}]}}`,
		`{"timestamp":"2026-10-30T04:11:28.138Z","type":"response_item","payload":{"type":"custom_tool_call","call_id":"call_parent","name":"exec","input":"text(await tools.exec_command({cmd:\"prompt-manager skill read research-coordination\"}))"}}`,
		`{"timestamp":"2026-10-30T04:11:28.138Z","type":"response_item","payload":{"type":"custom_tool_call_output","call_id":"call_parent","output":[{"type":"input_text","text":"skill text"}]}}`,
		`{"timestamp":"2026-10-30T04:11:28.150Z","type":"event_msg","payload":{"type":"agent_message","message":"parent reply"}}`,
		`{"timestamp":"2026-10-30T04:11:28.150Z","type":"event_msg","payload":{"type":"task_complete","turn_id":"01a18ae0-da1f-7011-a37d-9eaacb06b4aa"}}`,
		`{"timestamp":"2026-10-30T04:11:28.150Z","type":"event_msg","payload":{"type":"task_started","turn_id":"01a18aec-30b8-7062-84c8-d95c58257ee6"}}`,
		`{"timestamp":"2026-10-30T04:11:28.171Z","type":"event_msg","payload":{"type":"turn_aborted","turn_id":"01a18aec-30b8-7062-84c8-d95c58257ee6","reason":"interrupted"}}`,
		boundary,
		`{"timestamp":"2026-10-30T04:11:28.871Z","type":"event_msg","payload":{"type":"task_started","turn_id":"` + forkTurn + `"}}`,
		`{"timestamp":"2026-10-30T04:11:30.616Z","type":"turn_context","payload":{"turn_id":"` + forkTurn + `","model":"gpt-fork"}}`,
		`{"timestamp":"2026-10-30T04:11:37.060Z","type":"response_item","payload":{"type":"custom_tool_call","call_id":"call_fork","name":"exec","input":"text(await tools.exec_command({cmd:\"git status --short\"}))"}}`,
		`{"timestamp":"2026-10-30T04:11:37.223Z","type":"event_msg","payload":{"type":"item_completed","thread_id":"` + fork + `","turn_id":"` + forkTurn + `","item":{"type":"CommandExecution","id":"exec-1","command":["/bin/bash","-lc","git status --short"],"status":"completed","aggregated_output":" M a.go\n","exit_code":0,"duration":{"secs":0,"nanos":1000000}}}}`,
		`{"timestamp":"2026-10-30T04:11:37.300Z","type":"response_item","payload":{"type":"custom_tool_call_output","call_id":"call_fork","output":[{"type":"input_text","text":" M a.go\n"}]}}`,
		`{"timestamp":"2026-10-30T04:11:38.000Z","type":"event_msg","payload":{"type":"agent_message","message":"fork reply"}}`,
		`{"timestamp":"2026-10-30T04:11:38.100Z","type":"event_msg","payload":{"type":"task_complete","turn_id":"` + forkTurn + `"}}`,
	}
}

// A forked rollout's copy of its parent's history is the parent's run: the
// fork yields only the calls, messages, compactions and terminals of its own
// lines, and keeps its own session.
func TestCodexForkedRolloutYieldsOnlyTheForksOwnHistory(t *testing.T) {
	const settings = `{"timestamp":"2026-10-30T04:11:28.735Z","type":"event_msg","payload":{"type":"thread_settings_applied","thread_id":"01a18b01-6939-7482-a405-898d1a33c930","thread_settings":{"model":"gpt-fork"}}}`
	const timeOrdered = "01a18b01-6ca4-7b53-8c0a-300bddb4bbd4"
	for name, tc := range map[string]struct{ boundary, forkTurn string }{
		"thread settings": {settings, timeOrdered},
		"first fork turn": {`{"timestamp":"2026-10-30T04:11:28.735Z","type":"world_state","payload":{"full":true}}`, timeOrdered},
		// A turn id that does not sort after the fork still follows the
		// line naming the fork's thread.
		"thread settings, unordered turn id": {settings, "00000000-6ca4-7b53-8c0a-300bddb4bbd4"},
	} {
		t.Run(name, func(t *testing.T) {
			parser := NewCodexForTest().NewTranscriptParser()
			runID := uuid.New()
			var calls, messages []string
			var compactions, terminals int
			for _, line := range forkedRollout(tc.boundary, tc.forkTurn) {
				got := parser.ParseTranscriptLine(runID, line)
				if got.SessionID != "" && got.SessionID != "01a18b01-6939-7482-a405-898d1a33c930" {
					t.Fatalf("session switched to %q: the parent's copied session_meta is not this rollout's", got.SessionID)
				}
				if got.Terminal != nil {
					terminals++
				}
				for _, event := range got.Events {
					switch data := event.Data.(type) {
					case *domain.ToolCallEventData:
						calls = append(calls, data.ToolCallID)
					case *domain.MessageEventData:
						if data.Content != "" {
							messages = append(messages, data.Content)
						}
					case *domain.CompactionEventData:
						compactions++
					}
				}
			}
			if strings.Join(calls, ",") != "call_fork" || strings.Join(messages, ",") != "fork reply" || compactions != 0 || terminals != 1 {
				t.Fatalf("calls=%v messages=%v compactions=%d terminals=%d, want only the fork's own call, reply and terminal", calls, messages, compactions, terminals)
			}
		})
	}
}
