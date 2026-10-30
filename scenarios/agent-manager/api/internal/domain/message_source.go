// Responsibility: classify who authored a user-role message (the message source).
package domain

import "strings"

// Message sources. Detectors that need what a person actually said filter on
// MessageSourceOperator; every other source is text a harness or peer produced.
const (
	MessageSourceOperator         = "operator"
	MessageSourceInjectedContext  = "injected_context"
	MessageSourceNotification     = "notification"
	MessageSourceTeammate         = "teammate"
	MessageSourceGoalContinuation = "goal_continuation"
	MessageSourceWake             = "wake"
	MessageSourceHook             = "hook"
	MessageSourceToolOutput       = "tool_output"
	MessageSourceUnknown          = "unknown"
)

// Agent Manager wake message prefixes. orchestration/park.go builds the text
// from these and the classifier recognises them, so both stay in step.
const (
	WakeAwaitedTimedOutPrefix  = "[awaited result timed out]"
	WakeAwaitedResultPrefix    = "[awaited result available]"
	WakeTimerPrefix            = "[wake: timer]"
	WakePrefix                 = "[wake]"
	WakeDecisionPendingPrefix  = "[operator decision — still pending]"
	WakeDecisionResolvedPrefix = "[operator decision resolved]"
)

// coordinatorRecoveryPrefix opens the recovery notices a delivery-reliability
// coordinator sends a stalled run ("Reliability coordinator recovery after
// A83: ..."). They are control-plane text, not something the operator said.
const coordinatorRecoveryPrefix = "reliability coordinator recovery"

// instructionFilePrefixes are the exact headers harnesses inject ahead of an
// inlined instruction file in the first user-role record (Codex writes
// "# AGENTS.md instructions for <dir>"). A bare "# AGENTS.md" is a heading the
// operator may have pasted, so it is not a marker.
var instructionFilePrefixes = []string{"# agents.md instructions for ", "# claude.md instructions for "}

// wakePrefixes are the exact openings park.go gives an Agent Manager wake.
var wakePrefixes = []string{WakeAwaitedTimedOutPrefix, WakeAwaitedResultPrefix, WakeTimerPrefix, WakePrefix, WakeDecisionPendingPrefix, WakeDecisionResolvedPrefix}

func hasWakePrefix(text string) bool {
	for _, p := range wakePrefixes {
		if strings.HasPrefix(text, p) {
			return true
		}
	}
	return false
}

// injectedContextTags are the harness/provider blocks inlined into user-role
// records. Any other leading tag (<div>, <pasted_content>, ...) is text the
// operator typed or pasted.
var injectedContextTags = map[string]bool{
	"environment_context": true, "user_instructions": true, "instructions": true,
	"skills_instructions": true, "system-instructions": true,
	"local-command-caveat": true, "local-command-stdout": true, "local-command-stderr": true,
	"command-name": true, "command-message": true, "command-args": true,
	"in-app-browser-context": true, "send_user_message_question_reply": true,
	"turn_aborted": true, "subagent_notification": true,
}

// leadingTagName returns the lowercased name of an opening tag at the start of
// text ("<command-name>x" -> "command-name"), or "".
func leadingTagName(text string) string {
	if !strings.HasPrefix(text, "<") {
		return ""
	}
	end := strings.IndexAny(text[1:], "> \t\r\n/")
	if end <= 0 || end > 64 {
		return ""
	}
	return text[1 : 1+end]
}

// stripLeadingHarnessBlocks removes the closed harness blocks (system
// reminders, hook output, injected-context tags) that open text and returns
// what follows and how many were removed. An unclosed block is left in place.
func stripLeadingHarnessBlocks(text string) (rest string, stripped int) {
	rest = strings.TrimSpace(text)
	for {
		tag := strings.ToLower(leadingTagName(rest))
		if tag == "" || !(tag == "system-reminder" || tag == "user-prompt-submit-hook" || injectedContextTags[tag]) {
			return rest, stripped
		}
		closing := "</" + tag + ">"
		end := strings.Index(strings.ToLower(rest), closing)
		if end < 0 {
			return rest, stripped
		}
		rest = strings.TrimSpace(rest[end+len(closing):])
		stripped++
	}
}

// ClassifyMessageText derives the source of a user-role message from its text
// alone. Text without a harness marker is the operator's. Empty text is unknown.
func ClassifyMessageText(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return MessageSourceUnknown
	}
	lowered := strings.ToLower(trimmed)
	if hasWakePrefix(trimmed) {
		return MessageSourceWake
	}
	if strings.HasPrefix(lowered, coordinatorRecoveryPrefix) || strings.HasPrefix(trimmed, "[SYSTEM NOTIFICATION") {
		return MessageSourceNotification
	}
	if rest, stripped := stripLeadingHarnessBlocks(trimmed); stripped > 0 && rest != "" {
		// What follows the harness blocks decides: typed text is the operator's,
		// another injected form (a notification, a wake, a goal turn) is that form.
		return ClassifyMessageText(rest)
	}
	if tag := leadingTagName(lowered); tag != "" {
		switch tag {
		case "codex_internal_context":
			head := lowered
			if len(head) > 240 {
				head = head[:240]
			}
			if strings.Contains(head, `source="goal"`) {
				return MessageSourceGoalContinuation
			}
			return MessageSourceInjectedContext
		case "task-notification":
			return MessageSourceNotification
		case "teammate-message":
			return MessageSourceTeammate
		case "system-reminder", "user-prompt-submit-hook":
			return MessageSourceHook
		}
		if injectedContextTags[tag] {
			return MessageSourceInjectedContext
		}
	}
	for _, prefix := range instructionFilePrefixes {
		if strings.HasPrefix(lowered, prefix) {
			return MessageSourceInjectedContext
		}
	}
	return MessageSourceOperator
}

// IsInjectedContextText reports whether a candidate user message is not
// something a person typed. Such text is unusable as a run label.
func IsInjectedContextText(text string) bool {
	return ClassifyMessageText(text) != MessageSourceOperator
}

// ClassifyProviderMessage combines provider fields with content. Agent Manager
// wakes are recognised by content first, because they are typed into a session
// the provider records as human. Otherwise a recognised provider origin is
// decisive; unrecognised or absent origin falls back to content markers and
// then to promptSource / isMeta.
func ClassifyProviderMessage(text, originKind, promptSource string, isMeta bool) string {
	content := ClassifyMessageText(text)
	if content == MessageSourceWake || content == MessageSourceUnknown {
		return content
	}
	switch strings.ToLower(strings.TrimSpace(originKind)) {
	case "human":
		// Harnesses sometimes tag injected records as human; text that is
		// nothing but harness blocks is classified by content.
		if rest, stripped := stripLeadingHarnessBlocks(text); stripped > 0 && rest == "" && content != MessageSourceOperator {
			return content
		}
		return MessageSourceOperator
	case "task-notification":
		return MessageSourceNotification
	case "peer":
		return MessageSourceTeammate
	case "auto-continuation":
		return MessageSourceGoalContinuation
	}
	if content != MessageSourceOperator {
		return content
	}
	if isMeta {
		return MessageSourceInjectedContext
	}
	switch strings.ToLower(strings.TrimSpace(promptSource)) {
	case "system", "sdk":
		return MessageSourceInjectedContext
	}
	return MessageSourceOperator
}

// MessageSource returns the source of a user-role message event: the stored
// value when set, else one derived from content, so events stored before the
// field existed read correctly without a migration. Other roles and non-message
// events return "". The stored value is trusted forever: a change to the
// classifier does not reach events already stored and needs a re-import.
func MessageSource(ev *RunEvent) string {
	if ev == nil {
		return ""
	}
	data, ok := ev.Data.(*MessageEventData)
	if !ok || data == nil {
		return ""
	}
	if data.Source != "" {
		return data.Source
	}
	switch strings.ToLower(data.Role) {
	case "user", "operator":
		return ClassifyMessageText(data.Content)
	case "tool":
		return MessageSourceToolOutput
	}
	return ""
}

// userMessageSource is the source stamped on a new message: only user-role
// messages have one.
func userMessageSource(role, content string) string {
	if strings.EqualFold(role, "user") {
		return ClassifyMessageText(content)
	}
	return ""
}
