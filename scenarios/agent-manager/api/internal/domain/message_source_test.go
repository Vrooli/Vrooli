package domain

import (
	"testing"

	"github.com/google/uuid"
)

// Events stored before the Source field existed must read correctly without a
// migration: MessageSource derives the source from content.
func TestMessageSourceDerivesForLegacyEvents(t *testing.T) {
	cases := []struct {
		role, content, stored, want string
	}{
		{"user", "fix the build", "", MessageSourceOperator},
		{"user", "[wake] child ended", "", MessageSourceWake},
		{"user", "[operator decision resolved] approved", "", MessageSourceWake},
		{"user", "<task-notification>x</task-notification>", "", MessageSourceNotification},
		{"user", "Reliability coordinator recovery after A83: reads are qualified; stay in this task.", "", MessageSourceNotification},
		{"user", "fix the build", MessageSourceHook, MessageSourceHook},
		{"user", "   ", "", MessageSourceUnknown},
		{"assistant", "[wake] quoted by the assistant", "", ""},
	}
	for _, tc := range cases {
		ev := &RunEvent{ID: uuid.New(), EventType: EventTypeMessage, Data: &MessageEventData{Role: tc.role, Content: tc.content, Source: tc.stored}}
		if got := MessageSource(ev); got != tc.want {
			t.Errorf("MessageSource(%s %q stored=%q) = %q, want %q", tc.role, tc.content, tc.stored, got, tc.want)
		}
	}
}

func TestNewMessageEventsStampSource(t *testing.T) {
	if got := NewMessageEvent(uuid.New(), "user", "[wake] x").Data.(*MessageEventData).Source; got != MessageSourceWake {
		t.Fatalf("source = %q", got)
	}
	if got := NewMessageEvent(uuid.New(), "assistant", "hi").Data.(*MessageEventData).Source; got != "" {
		t.Fatalf("assistant source = %q, want empty", got)
	}
}

func TestClassifyMessageTextSources(t *testing.T) {
	cases := []struct{ text, want string }{
		{"<div> why is this broken", MessageSourceOperator},
		{"<pasted_content>stack trace</pasted_content> fix", MessageSourceOperator},
		{"Is this another Claude session talking to me?", MessageSourceOperator},
		{"[wake up the build] please", MessageSourceOperator},
		{"[wake] child ended", MessageSourceWake},
		{WakeTimerPrefix + " nothing", MessageSourceWake},
		{WakeAwaitedResultPrefix + " done", MessageSourceWake},
		{WakeAwaitedTimedOutPrefix + " x", MessageSourceWake},
		{WakeDecisionPendingPrefix + " x", MessageSourceWake},
		{"<codex_internal_context source=\"goal\">x", MessageSourceGoalContinuation},
		{"<codex_internal_context>x", MessageSourceInjectedContext},
		{"<task-notification>x", MessageSourceNotification},
		{"<teammate-message from=\"a\">x", MessageSourceTeammate},
		{"<system-reminder>x", MessageSourceHook},
		{"<user-prompt-submit-hook>x", MessageSourceHook},
		{"<environment_context>x", MessageSourceInjectedContext},
		{"<local-command-stdout>x", MessageSourceInjectedContext},
		{"<command-name>/x</command-name>", MessageSourceInjectedContext},
		{"<in-app-browser-context>x", MessageSourceInjectedContext},
		{"<send_user_message_question_reply>x", MessageSourceInjectedContext},
		{"# AGENTS.md instructions for /x", MessageSourceInjectedContext},
		{"# AGENTS.md notes I pasted", MessageSourceOperator},
		{"<system-reminder>ctx</system-reminder>\n\nplease fix the bug", MessageSourceOperator},
		{"<system-reminder>a</system-reminder><system-reminder>b</system-reminder>", MessageSourceHook},
		{"<system-reminder>unclosed ctx", MessageSourceHook},
		{"<environment_context>x</environment_context>\nrun the tests", MessageSourceOperator},
	}
	for _, tc := range cases {
		if got := ClassifyMessageText(tc.text); got != tc.want {
			t.Errorf("ClassifyMessageText(%q) = %q, want %q", tc.text, got, tc.want)
		}
	}
}

func TestClassifyProviderMessageHumanOriginInjectedBlock(t *testing.T) {
	cases := []struct {
		text, origin, want string
	}{
		{"<system-reminder>ctx</system-reminder>", "human", MessageSourceHook},
		{"<environment_context>x</environment_context>", "human", MessageSourceInjectedContext},
		{"<system-reminder>ctx</system-reminder>\nreal words", "human", MessageSourceOperator},
		{"just typed", "human", MessageSourceOperator},
		{"<system-reminder>ctx</system-reminder>\nreal words", "peer", MessageSourceTeammate},
	}
	for _, tc := range cases {
		if got := ClassifyProviderMessage(tc.text, tc.origin, "", false); got != tc.want {
			t.Errorf("ClassifyProviderMessage(%q, %q) = %q, want %q", tc.text, tc.origin, got, tc.want)
		}
	}
}

// Harness blocks in front of another injected form leave that form's source.
func TestHarnessBlocksBeforeInjectedFormsKeepTheirSource(t *testing.T) {
	reminder := "<system-reminder>context</system-reminder>\n"
	cases := map[string]string{
		reminder + "<task-notification><task-id>a</task-id></task-notification>": MessageSourceNotification,
		reminder + "<teammate-message teammate_id=\"x\">hi</teammate-message>":   MessageSourceTeammate,
		reminder + "Please fix the failing test":                                 MessageSourceOperator,
	}
	for text, want := range cases {
		if got := ClassifyMessageText(text); got != want {
			t.Errorf("ClassifyMessageText(%q) = %q, want %q", text, got, want)
		}
	}
}
