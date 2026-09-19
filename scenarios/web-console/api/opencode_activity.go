package main

import (
	"strings"

	"web-console/backends/opencode"
	"web-console/internal/backend"
	"web-console/session"
)

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#session-activity-contract

// OpenCode reports its own state on the event stream, so an OpenCode pane's
// activity comes from the harness rather than from reading its screen:
//
//	permission.asked / question.asked      -> waiting, with the prompt
//	permission.replied / question.replied,
//	question.rejected                      -> the prompt is resolved
//	session.status busy / idle, session.idle -> working / idle

// applyActivity routes one OpenCode event to the claimed pane's activity
// detector. Events for unclaimed sessions are ignored.
func (w *OpenCodeWatcher) applyActivity(ev opencode.Event) {
	if w.server == nil {
		return
	}
	ocID := ev.SessionID()
	if ocID == "" {
		return
	}
	w.mu.Lock()
	wcID := w.claimed[ocID]
	w.mu.Unlock()
	if wcID == "" {
		return
	}
	switch ev.Type {
	case "permission.asked":
		if requestID, _ := ev.Properties["id"].(string); requestID != "" {
			w.setPendingPermission(wcID, requestID)
		}
	case "permission.replied":
		w.setPendingPermission(wcID, "")
	}
	detector := w.server.activityDetector(wcID)
	if detector == nil {
		return
	}
	switch ev.Type {
	case "permission.asked", "question.asked":
		if prompt := opencodePrompt(ev); prompt != nil {
			detector.OnHarnessPrompt(*prompt)
		}
	case "permission.replied", "question.replied", "question.rejected":
		detector.OnHarnessPromptResolved()
	case "session.status":
		status, _ := ev.Properties["status"].(map[string]any)
		switch statusType, _ := status["type"].(string); statusType {
		case "busy", "retry":
			detector.OnHarnessState(session.ActivityWorking)
		case "idle":
			detector.OnHarnessState(session.ActivityIdle)
		}
	case "session.idle":
		detector.OnHarnessState(session.ActivityIdle)
	}
}

// opencodePermissionOptions are OpenCode's three permission replies.
var opencodePermissionOptions = []backend.PromptOption{
	{Key: "once", Label: "Allow once"},
	{Key: "always", Label: "Always allow"},
	{Key: "reject", Label: "Reject"},
}

// opencodePrompt reads the pending prompt from a permission.asked or
// question.asked event.
func opencodePrompt(ev opencode.Event) *backend.PendingPrompt {
	switch ev.Type {
	case "permission.asked":
		text, _ := ev.Properties["permission"].(string)
		if patterns := stringList(ev.Properties["patterns"]); len(patterns) > 0 {
			text = strings.TrimSpace(text + " " + strings.Join(patterns, ", "))
		}
		return &backend.PendingPrompt{
			Kind:    "permission",
			Text:    text,
			Options: append([]backend.PromptOption(nil), opencodePermissionOptions...),
		}
	case "question.asked":
		questions, _ := ev.Properties["questions"].([]any)
		if len(questions) == 0 {
			return nil
		}
		// One card asks one question; the first leads, as in OpenCode's TUI.
		first, _ := questions[0].(map[string]any)
		text, _ := first["question"].(string)
		prompt := &backend.PendingPrompt{Kind: "question", Text: text}
		options, _ := first["options"].([]any)
		// OpenCode answers a question with the chosen label, so it is the key.
		for _, raw := range options {
			option, _ := raw.(map[string]any)
			label, _ := option["label"].(string)
			if label == "" {
				continue
			}
			prompt.Options = append(prompt.Options, backend.PromptOption{Key: label, Label: label})
		}
		if custom, ok := first["custom"].(bool); !ok || custom {
			prompt.FreeTextHint = "Type your own answer"
		}
		return prompt
	}
	return nil
}

func stringList(value any) []string {
	items, _ := value.([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok && text != "" {
			out = append(out, text)
		}
	}
	return out
}
