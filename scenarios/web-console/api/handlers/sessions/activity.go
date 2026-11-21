package sessions

import (
	"time"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"

	"web-console/internal/backend"
	"web-console/session"
)

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#session-activity-contract

var activityStates = map[session.ActivityState]sessionsv1.SessionActivityState{
	session.ActivityUnknown: sessionsv1.SessionActivityState_SESSION_ACTIVITY_STATE_UNKNOWN,
	session.ActivityWorking: sessionsv1.SessionActivityState_SESSION_ACTIVITY_STATE_WORKING,
	session.ActivityIdle:    sessionsv1.SessionActivityState_SESSION_ACTIVITY_STATE_IDLE,
	session.ActivityWaiting: sessionsv1.SessionActivityState_SESSION_ACTIVITY_STATE_WAITING,
}

var activitySources = map[session.ActivitySource]sessionsv1.SessionActivitySource{
	session.SourceScreen:       sessionsv1.SessionActivitySource_SESSION_ACTIVITY_SOURCE_SCREEN,
	session.SourceOutputClock:  sessionsv1.SessionActivitySource_SESSION_ACTIVITY_SOURCE_OUTPUT_CLOCK,
	session.SourceHook:         sessionsv1.SessionActivitySource_SESSION_ACTIVITY_SOURCE_HOOK,
	session.SourceHarnessEvent: sessionsv1.SessionActivitySource_SESSION_ACTIVITY_SOURCE_HARNESS_EVENT,
}

// activityToProto projects a live session's activity onto the wire.
func activityToProto(sessionID string, a session.Activity) *sessionsv1.SessionActivity {
	out := &sessionsv1.SessionActivity{
		SessionId:      sessionID,
		State:          activityStates[a.State],
		Source:         activitySources[a.Source],
		Confidence:     a.Confidence,
		Since:          activityTime(a.Since),
		LastOutputAt:   activityTime(a.LastOutputAt),
		Harness:        a.Harness,
		HarnessVersion: a.HarnessVersion,
	}
	if a.Prompt != nil {
		prompt := &sessionsv1.PendingPrompt{
			Kind: a.Prompt.Kind, Text: a.Prompt.Text, FreeTextHint: a.Prompt.FreeTextHint,
			Answerable: a.Answerable, Hash: backend.PromptHash(a.Prompt), Cancellable: a.Prompt.Cancellable,
		}
		for _, option := range a.Prompt.Options {
			prompt.Options = append(prompt.Options, &sessionsv1.PromptOption{Key: option.Key, Label: option.Label, Selected: option.Selected})
		}
		out.Prompt = prompt
	}
	return out
}

func activityTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}
