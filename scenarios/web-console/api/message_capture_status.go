package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"web-console/internal/sessionstore"

	conversationH "web-console/handlers/conversation"
)

// Message capture is the pipeline that turns what an agent says in a terminal
// into stored conversation events. It has several independent preconditions —
// the session must run a known agent, that agent must be identified, its
// transcript must exist and be readable — and until this file existed every one
// of them failed the same way: the reader skipped the session and the Messages
// view rendered "no events yet".
//
// That message was frequently false. This diagnosis exists so the reason a
// conversation is empty travels with the empty conversation, and so the states
// that resolve on their own (PENDING) are never dressed up as faults while the
// states that need an operator (UNAVAILABLE) are never dressed down as silence.
//
// Every reason code below is stable: the UI selects copy and remediation from
// it, and operators grep logs for it.
const (
	captureReasonNoAgent            = "no_agent"
	captureReasonSessionUnknown     = "session_unknown"
	captureReasonAgentUnidentified  = "agent_unidentified"
	captureReasonWorkingDirUnknown  = "working_dir_unknown"
	captureReasonTranscriptMissing  = "transcript_missing"
	captureReasonTranscriptUnread   = "transcript_unreadable"
	captureReasonAwaitingFirstTurn  = "awaiting_first_turn"
	captureReasonHookNotRegistered  = "hook_not_registered"
	captureReasonUnsupportedAgent   = "unsupported_agent"
	captureRemediationRegisterHooks = "Run 'web-console hooks register' to reconnect message capture, then use the session once."
)

// CaptureStatus diagnoses one session's message capture. It is deliberately
// read-only and cheap: a metadata lookup, at most one os.Stat, and — for Claude
// sessions that are not yet identified — the already-cached hook status. It is
// called on every conversation Get, so it must never block on the network.
func (a *conversationAdapter) CaptureStatus(ctx context.Context, sessionID string) conversationH.CaptureStatus {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" || a.srv == nil {
		return conversationH.CaptureStatus{
			State:      conversationH.CaptureUnavailable,
			ReasonCode: captureReasonSessionUnknown,
			Summary:    "This session could not be found.",
		}
	}

	meta, ok := a.sessionMetadata(ctx, sessionID)
	if !ok {
		return conversationH.CaptureStatus{
			State:      conversationH.CaptureUnavailable,
			ReasonCode: captureReasonSessionUnknown,
			Summary:    "This session could not be found.",
			Detail:     fmt.Sprintf("No metadata row exists for session %s.", sanitizeID(sessionID)),
		}
	}

	status := a.diagnose(meta)
	status.LastCapturedAt = a.lastCapturedAt(ctx, sessionID)
	// A session that has captured messages before is demonstrably wired up.
	// Reporting UNAVAILABLE over a transcript we already read would tell the
	// operator to repair something that is working; the honest reading is that
	// capture succeeded and has since gone quiet.
	if status.State == conversationH.CaptureUnavailable && status.LastCapturedAt != "" {
		status.State = conversationH.CapturePending
		status.Remediation = ""
	}
	return status
}

// diagnose resolves the capture state from session metadata alone. Split out
// from CaptureStatus so the decision table is testable without a store.
func (a *conversationAdapter) diagnose(meta sessionstore.Metadata) conversationH.CaptureStatus {
	switch meta.AgentType {
	case sessionstore.AgentNone, "":
		return conversationH.CaptureStatus{
			State:      conversationH.CaptureNotApplicable,
			ReasonCode: captureReasonNoAgent,
			Summary:    "This is a plain terminal, so there are no messages to show.",
			Detail:     "Messages are captured from coding agents. Start one in this terminal to see a conversation here.",
		}
	case sessionstore.AgentClaude:
		return a.diagnoseClaude(meta)
	case sessionstore.AgentCodex, sessionstore.AgentGrok, sessionstore.AgentOpenCode:
		return diagnoseSelfIdentifyingAgent(meta)
	default:
		return conversationH.CaptureStatus{
			State:      conversationH.CaptureUnavailable,
			ReasonCode: captureReasonUnsupportedAgent,
			Summary:    "Web Console can't record messages for this kind of agent yet.",
			Detail:     fmt.Sprintf("Unrecognized agent type %q.", string(meta.AgentType)),
		}
	}
}

// diagnoseClaude covers the agent with the most failure modes, because Claude
// is the only one that cannot name its own transcript. Codex and Grok write an
// identity record their tailers read back; Claude does not, so identification
// depends on the Stop hook or on the process-attribution fallback. Each of
// those can be absent for a different reason, and each gets its own answer.
func (a *conversationAdapter) diagnoseClaude(meta sessionstore.Metadata) conversationH.CaptureStatus {
	if meta.AgentSessionID == "" {
		// Not yet identified. Whether this resolves on its own depends entirely
		// on whether the hook that performs identification is actually wired
		// up, so ask before choosing between "wait" and "fix something".
		hookRegistered, hookCode, hookReason, settingsPath := a.srv.getClaudeHookStatus()
		if !hookRegistered {
			return conversationH.CaptureStatus{
				State:       conversationH.CaptureUnavailable,
				ReasonCode:  captureReasonHookNotRegistered,
				Summary:     "Messages aren't being captured — Web Console isn't connected to Claude Code in this project.",
				Detail:      fmt.Sprintf("%s (%s). Settings file: %s", hookReason, hookCode, settingsPath),
				Remediation: captureRemediationRegisterHooks,
			}
		}
		if meta.CWD == "" {
			return conversationH.CaptureStatus{
				State:      conversationH.CapturePending,
				ReasonCode: captureReasonWorkingDirUnknown,
				Summary:    "Waiting to identify the agent in this terminal.",
				Detail:     "This session has no recorded working directory yet, so its message history can't be located. It will be identified the next time the agent replies.",
			}
		}
		return conversationH.CaptureStatus{
			State:      conversationH.CapturePending,
			ReasonCode: captureReasonAwaitingFirstTurn,
			Summary:    "Waiting for the first reply in this session.",
			Detail:     "Message capture identifies a Claude session from its first completed turn. Send a message to start the conversation.",
		}
	}

	if meta.CWD == "" {
		return conversationH.CaptureStatus{
			State:       conversationH.CaptureUnavailable,
			ReasonCode:  captureReasonWorkingDirUnknown,
			Summary:     "Messages aren't being captured — Web Console doesn't know where this session is running.",
			Detail:      fmt.Sprintf("Session %s is identified as Claude session %s but has no recorded working directory, which is required to locate its message history.", sanitizeID(meta.ID), sanitizeID(meta.AgentSessionID)),
			Remediation: "Start a new session to restore message capture; sessions record their working directory when they are created.",
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return conversationH.CaptureStatus{
			State:      conversationH.CaptureUnavailable,
			ReasonCode: captureReasonTranscriptUnread,
			Summary:    "Messages aren't being captured — Web Console can't reach the message history on disk.",
			Detail:     fmt.Sprintf("Home directory unavailable: %v", err),
		}
	}
	return transcriptFileStatus(claudeTranscriptPath(home, meta.CWD, meta.AgentSessionID))
}

// diagnoseSelfIdentifyingAgent covers Codex, Grok and OpenCode. Their tailers
// recover identity from the transcript itself, so a missing agent session id
// only ever means "nothing written yet" — never a broken pipeline.
func diagnoseSelfIdentifyingAgent(meta sessionstore.Metadata) conversationH.CaptureStatus {
	if meta.AgentSessionID == "" {
		return conversationH.CaptureStatus{
			State:      conversationH.CapturePending,
			ReasonCode: captureReasonAwaitingFirstTurn,
			Summary:    "Waiting for the first reply in this session.",
			Detail:     "Message capture starts as soon as the agent writes its first response.",
		}
	}
	if meta.LastRolloutPath == "" {
		return conversationH.CaptureStatus{
			State:      conversationH.CaptureCapturing,
			ReasonCode: "",
			Summary:    "Messages are being captured.",
		}
	}
	return transcriptFileStatus(meta.LastRolloutPath)
}

// transcriptFileStatus turns the presence and readability of a transcript file
// into a capture state. A missing file for an already-identified session is a
// genuine fault: the identity came from somewhere, so the file existed once.
func transcriptFileStatus(path string) conversationH.CaptureStatus {
	info, err := os.Stat(path)
	switch {
	case err == nil && !info.IsDir():
		return conversationH.CaptureStatus{
			State:          conversationH.CaptureCapturing,
			Summary:        "Messages are being captured.",
			TranscriptPath: path,
		}
	case os.IsNotExist(err):
		return conversationH.CaptureStatus{
			State:          conversationH.CaptureUnavailable,
			ReasonCode:     captureReasonTranscriptMissing,
			Summary:        "Messages aren't being captured — this session's message history is missing.",
			Detail:         fmt.Sprintf("Expected the agent's history file at %s, but it no longer exists.", path),
			Remediation:    "The history file may have been cleaned up. Messages already captured are still shown; new ones will resume if the agent recreates it.",
			TranscriptPath: path,
		}
	default:
		detail := fmt.Sprintf("Could not read %s", path)
		if err != nil {
			detail = fmt.Sprintf("%s: %v", detail, err)
		}
		return conversationH.CaptureStatus{
			State:          conversationH.CaptureUnavailable,
			ReasonCode:     captureReasonTranscriptUnread,
			Summary:        "Messages aren't being captured — this session's message history can't be read.",
			Detail:         detail,
			Remediation:    "Check that the file is readable by the account running Web Console.",
			TranscriptPath: path,
		}
	}
}

// sessionMetadata prefers the durable row and falls back to the live session
// manager, so a session created before the store was reachable still reports a
// real state instead of "not found".
func (a *conversationAdapter) sessionMetadata(ctx context.Context, sessionID string) (sessionstore.Metadata, bool) {
	if a.srv.sessionStore != nil {
		if meta, err := a.srv.sessionStore.Get(ctx, sessionID); err == nil && meta.ID != "" {
			return meta, true
		}
	}
	if a.srv.sessions != nil {
		if _, ok := a.srv.sessions.Get(sessionID); ok {
			return sessionstore.Metadata{ID: sessionID}, true
		}
	}
	return sessionstore.Metadata{}, false
}

// lastCapturedAt reports when this session last produced a stored message. It
// is the one piece of evidence that outranks every static precondition: a
// session we have already read from is, by construction, one we could read.
func (a *conversationAdapter) lastCapturedAt(ctx context.Context, sessionID string) string {
	if a.srv.conversations == nil {
		return ""
	}
	state := a.srv.conversations.ListSession(ctx, sessionID)
	if len(state.Events) == 0 {
		return ""
	}
	last := state.Events[len(state.Events)-1]
	if last.CreatedAt.IsZero() {
		return ""
	}
	return last.CreatedAt.UTC().Format(time.RFC3339)
}
