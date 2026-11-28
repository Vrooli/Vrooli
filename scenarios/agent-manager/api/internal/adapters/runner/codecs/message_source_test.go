package codecs

import (
	"fmt"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// userSources replays one transcript line through the real transcript parser
// with user-content retention on and returns the source of each user message.
func userSources(t *testing.T, codec Codec, line string) []string {
	t.Helper()
	parser := codec.NewTranscriptParser()
	if retainer, ok := parser.(interface{ SetTranscriptRetention(bool) }); ok {
		retainer.SetTranscriptRetention(true)
	}
	var sources []string
	for _, event := range parser.ParseTranscriptLine(uuid.New(), line).Events {
		if message, ok := event.Data.(*domain.MessageEventData); ok && message.Role == "user" {
			sources = append(sources, domain.MessageSource(event))
		}
	}
	return sources
}

func claudeUserLine(content, extra string) string {
	return fmt.Sprintf(`{"type":"user",%s"message":{"role":"user","content":%q}}`, extra, content)
}

func TestClaudeTranscriptMessageSource(t *testing.T) {
	cases := []struct {
		name, line, want string
	}{
		{"origin human", claudeUserLine("fix the bug", `"origin":{"kind":"human"},"promptSource":"typed",`), domain.MessageSourceOperator},
		{"queued prompt", claudeUserLine("also run tests", `"promptSource":"queued",`), domain.MessageSourceOperator},
		{"origin task-notification", claudeUserLine("job finished", `"origin":{"kind":"task-notification","producer":"session-task"},`), domain.MessageSourceNotification},
		{"origin peer", claudeUserLine("thanks, done", `"origin":{"kind":"peer","from":"uds:/x"},`), domain.MessageSourceTeammate},
		{"origin auto-continuation", claudeUserLine("continue", `"origin":{"kind":"auto-continuation"},`), domain.MessageSourceGoalContinuation},
		{"turnOrigin only", claudeUserLine("job finished", `"turnOrigin":"task_notification",`), domain.MessageSourceNotification},
		{"system prompt source", claudeUserLine("do the thing", `"promptSource":"system",`), domain.MessageSourceInjectedContext},
		{"isMeta", claudeUserLine("caveat text", `"isMeta":true,`), domain.MessageSourceInjectedContext},
		{"legacy task-notification marker", claudeUserLine("<task-notification><task-id>1</task-id></task-notification>", ""), domain.MessageSourceNotification},
		{"legacy teammate marker", claudeUserLine(`<teammate-message teammate_id="a">hi</teammate-message>`, ""), domain.MessageSourceTeammate},
		{"legacy local command", claudeUserLine("<local-command-stdout>ok</local-command-stdout>", ""), domain.MessageSourceInjectedContext},
		{"legacy system notification", claudeUserLine("[SYSTEM NOTIFICATION - not a user message]", ""), domain.MessageSourceNotification},
		{"legacy system-reminder only", claudeUserLine("<system-reminder>be brief</system-reminder>", ""), domain.MessageSourceHook},
		{"legacy typed text", claudeUserLine("please continue", ""), domain.MessageSourceOperator},
		{"wake typed into a human session", claudeUserLine("[wake] child run finished", `"origin":{"kind":"human"},`), domain.MessageSourceWake},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := userSources(t, NewClaudeForTest(), tc.line)
			if len(got) != 1 || got[0] != tc.want {
				t.Fatalf("sources = %v, want [%s]", got, tc.want)
			}
		})
	}
}

func codexUserMessage(text string) string {
	return fmt.Sprintf(`{"type":"event_msg","payload":{"type":"user_message","message":%q}}`, text)
}

func codexResponseUser(text string) string {
	return fmt.Sprintf(`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":%q}]}}`, text)
}

func codexResponseUserMeta(text, kinds, metadata string) string {
	return fmt.Sprintf(`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":%q}],"internal_chat_message_metadata_passthrough":{"content_item_kinds":%s,"content_item_metadata":%s}}}`, text, kinds, metadata)
}

// A sub-agent rollout's user-role text is the parent agent's assignment or
// report: it must never count as something the operator said (P4-W4).
func TestCodexSubagentThreadUserTextIsTeammate(t *testing.T) {
	metas := map[string]string{
		"spawned worker":  `{"type":"session_meta","payload":{"id":"t1","thread_source":"subagent","source":{"subagent":{"thread_spawn":{"parent_thread_id":"p1","depth":1}}}}}`,
		"guardian review": `{"type":"session_meta","payload":{"id":"t2","thread_source":"guardian_review","source":{"subagent":{"other":"guardian"}}}}`,
	}
	for name, meta := range metas {
		t.Run(name, func(t *testing.T) {
			parser := NewCodexForTest().NewTranscriptParser()
			parser.(interface{ SetTranscriptRetention(bool) }).SetTranscriptRetention(true)
			parser.ParseTranscriptLine(uuid.New(), meta)
			var got []string
			for _, line := range []string{
				codexResponseUserMeta("Audit the parser and report back", `["user.text"]`, `[]`),
				codexUserMessage("Handoff received: 12:04Z exactT3response verified"),
				codexResponseUser("# AGENTS.md instructions for /repo\n\n<INSTRUCTIONS>x</INSTRUCTIONS>"),
			} {
				for _, event := range parser.ParseTranscriptLine(uuid.New(), line).Events {
					got = append(got, domain.MessageSource(event))
				}
			}
			want := []string{domain.MessageSourceTeammate, domain.MessageSourceTeammate, domain.MessageSourceInjectedContext}
			if fmt.Sprint(got) != fmt.Sprint(want) {
				t.Fatalf("sources = %v, want %v", got, want)
			}
		})
	}
	parser := NewCodexForTest().NewTranscriptParser()
	parser.(interface{ SetTranscriptRetention(bool) }).SetTranscriptRetention(true)
	parser.ParseTranscriptLine(uuid.New(), `{"type":"session_meta","payload":{"id":"t3","thread_source":"user","source":"vscode"}}`)
	events := parser.ParseTranscriptLine(uuid.New(), codexUserMessage("keep going")).Events
	if len(events) != 1 || domain.MessageSource(events[0]) != domain.MessageSourceOperator {
		t.Fatalf("top-level thread typed text must stay the operator's: %+v", events)
	}
}

func TestCodexTranscriptMessageSource(t *testing.T) {
	cases := []struct {
		name, line, want string
	}{
		{"typed message", codexUserMessage("add a regression test"), domain.MessageSourceOperator},
		{"goal continuation", codexResponseUser(`<codex_internal_context source="goal">Continue working toward the goal</codex_internal_context>`), domain.MessageSourceGoalContinuation},
		{"other internal context", codexResponseUser(`<codex_internal_context source="memory">x</codex_internal_context>`), domain.MessageSourceInjectedContext},
		{"AGENTS.md injection", codexResponseUser("# AGENTS.md instructions for /home/u/repo\n\n<INSTRUCTIONS>...</INSTRUCTIONS>"), domain.MessageSourceInjectedContext},
		{"environment context", codexResponseUser("<environment_context><cwd>/x</cwd></environment_context>"), domain.MessageSourceInjectedContext},
		{"browser context", codexResponseUser("<in-app-browser-context>tab</in-app-browser-context>"), domain.MessageSourceInjectedContext},
		{"question reply is the operator's answer", codexResponseUser("<send_user_message_question_reply>yes</send_user_message_question_reply>"), domain.MessageSourceOperator},
		{"typed item with provenance", codexResponseUserMeta("Web-console is down", `["user.text"]`, `[{"provenance":{"type":"user"}}]`), domain.MessageSourceOperator},
		{"harness item without markers", codexResponseUserMeta("plain injected text", `["agents_md.instructions","environments.environment_context"]`, `[{"provenance":{"type":"agents_md"}},{"provenance":{"type":"harness","hook":false}}]`), domain.MessageSourceInjectedContext},
		{"agent manager wake", codexUserMessage("[wake: timer] No child run ended since your last wake."), domain.MessageSourceWake},
		{"awaited result", codexResponseUser("[awaited result available] job has completed."), domain.MessageSourceWake},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := userSources(t, NewCodexForTest(), tc.line)
			if len(got) != 1 || got[0] != tc.want {
				t.Fatalf("sources = %v, want [%s]", got, tc.want)
			}
		})
	}
}

func TestCodexToolResultOutcomeFromTextExitCode(t *testing.T) {
	cases := []struct {
		name, output string
		wantExit     *int
	}{
		{"exit zero", "Chunk ID: a\nWall time: 0.1 seconds\nProcess exited with code 0\nOutput:\nok", intPtr(0)},
		{"no match exit one", "Chunk ID: a\nProcess exited with code 1\nOutput:\n", intPtr(1)},
		{"still running", "Chunk ID: a\nProcess running with session ID 7\nOutput:\n", nil},
		{"code only in output body", "Output:\nthe line 'Process exited with code 9' is quoted here", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			line := fmt.Sprintf(`{"type":"response_item","payload":{"type":"function_call_output","call_id":"c1","output":%q}}`, tc.output)
			result := NewCodexForTest().NewTranscriptParser().ParseTranscriptLine(uuid.New(), line)
			var data *domain.ToolResultEventData
			for _, event := range result.Events {
				if d, ok := event.Data.(*domain.ToolResultEventData); ok {
					data = d
				}
			}
			if data == nil {
				t.Fatal("no tool result event")
			}
			if tc.wantExit == nil {
				if data.ExitCode != nil || (data.OutcomeKnown != nil && *data.OutcomeKnown) {
					t.Fatalf("outcome must stay unknown: %+v", data)
				}
				return
			}
			if data.ExitCode == nil || *data.ExitCode != *tc.wantExit || data.OutcomeKnown == nil || !*data.OutcomeKnown || data.Success != (*tc.wantExit == 0) {
				t.Fatalf("exit/outcome = %+v, want exit %d known", data, *tc.wantExit)
			}
		})
	}
}

func intPtr(v int) *int { return &v }
