package sessions

import (
	"testing"
	"time"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"

	"web-console/internal/backend"
	"web-console/session"
)

// [REQ:P0-017f] The sessions list carries each live session's activity.
func TestActivityToProtoCarriesWaitingPrompt(t *testing.T) {
	since := time.Date(2026, 9, 11, 3, 0, 0, 0, time.UTC)
	got := activityToProto("s1", session.Activity{
		State:      session.ActivityWaiting,
		Source:     session.SourceHook,
		Confidence: 0.9,
		Since:      since,
		Harness:    "claude",
		Prompt: &backend.PendingPrompt{Kind: "permission", Text: "Do you want to proceed?", Options: []backend.PromptOption{
			{Key: "1", Label: "Yes", Selected: true}, {Key: "2", Label: "No"},
		}},
	})
	if got.GetState() != sessionsv1.SessionActivityState_SESSION_ACTIVITY_STATE_WAITING ||
		got.GetSource() != sessionsv1.SessionActivitySource_SESSION_ACTIVITY_SOURCE_HOOK {
		t.Fatalf("state/source = %v/%v, want WAITING/HOOK", got.GetState(), got.GetSource())
	}
	if got.GetSince() != "2026-09-11T03:00:00Z" || got.GetLastOutputAt() != "" {
		t.Fatalf("times = %q/%q, want since set and no output time", got.GetSince(), got.GetLastOutputAt())
	}
	prompt := got.GetPrompt()
	if prompt.GetKind() != "permission" || len(prompt.GetOptions()) != 2 || !prompt.GetOptions()[0].GetSelected() {
		t.Fatalf("prompt = %+v, want the permission prompt with the first option selected", prompt)
	}
	if prompt.GetAnswerable() {
		t.Fatal("answering from Messages is off until Level 3 enables it")
	}
}
