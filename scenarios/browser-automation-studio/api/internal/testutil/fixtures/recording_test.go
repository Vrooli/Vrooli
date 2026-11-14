package fixtures

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/domain"
	recordingpersistence "github.com/vrooli/browser-automation-studio/services/recording/persistence"
	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
)

func TestRecordingSessionDefaultsAndOptions(t *testing.T) {
	createdAt := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	session := RecordingSession(
		WithRecordingSessionID("session-123"),
		WithRecordingSessionProfileID("profile-123"),
		WithRecordingSessionStatus(domain.SessionStatusClosed),
		WithRecordingSessionViewport(1366, 768),
		WithRecordingSessionCreatedAt(createdAt),
	)

	if session.ID != "session-123" {
		t.Fatalf("expected session ID override, got %q", session.ID)
	}
	if session.ProfileID != "profile-123" {
		t.Fatalf("expected profile ID override, got %q", session.ProfileID)
	}
	if session.Status != domain.SessionStatusClosed {
		t.Fatalf("expected closed status, got %q", session.Status)
	}
	if session.ViewportWidth != 1366 || session.ViewportHeight != 768 {
		t.Fatalf("expected viewport override, got %dx%d", session.ViewportWidth, session.ViewportHeight)
	}
	if !session.CreatedAt.Equal(createdAt) {
		t.Fatalf("expected created_at override, got %s", session.CreatedAt)
	}
}

func TestTimelineEntryActionOptionSetsActionType(t *testing.T) {
	sessionID := "session-123"
	pageID := uuid.New()
	action := RecordingAction(
		WithRecordingActionSessionID(sessionID),
		WithRecordingActionPageID(pageID),
		WithRecordingActionType("navigate"),
	)

	entry := TimelineEntry(
		WithTimelineEntrySessionID(sessionID),
		WithTimelineEntryPageID(pageID),
		WithTimelineEntrySequence(42),
		WithTimelineEntryAction(action),
	)

	if entry.Type != recordingpersistence.TimelineEntryTypeAction {
		t.Fatalf("expected action entry type, got %q", entry.Type)
	}
	if entry.Action == nil {
		t.Fatal("expected action to be attached")
	}
	if entry.Action.ActionType != "navigate" {
		t.Fatalf("expected action type override, got %q", entry.Action.ActionType)
	}
	if entry.Sequence != 42 {
		t.Fatalf("expected sequence override, got %d", entry.Sequence)
	}
}

func TestTimelineEntryPageEventOptionSetsPageEventType(t *testing.T) {
	event := PageEvent(WithPageEventURL("https://example.test"))

	entry := TimelineEntry(WithTimelineEntryPageEvent(event))

	if entry.Type != recordingpersistence.TimelineEntryTypePageEvent {
		t.Fatalf("expected page event entry type, got %q", entry.Type)
	}
	if entry.PageEvent == nil {
		t.Fatal("expected page event to be attached")
	}
	if entry.PageEvent.URL != "https://example.test" {
		t.Fatalf("expected page event URL override, got %q", entry.PageEvent.URL)
	}
}

func TestSessionProfileDefaultsAndOptions(t *testing.T) {
	profile := SessionProfile(
		WithSessionProfileID(sessionprofilepersistence.ProfileID("profile-123")),
		WithSessionProfileName("Profile fixture"),
	)

	if profile.ID != "profile-123" {
		t.Fatalf("expected profile ID override, got %q", profile.ID)
	}
	if profile.Name != "Profile fixture" {
		t.Fatalf("expected profile name override, got %q", profile.Name)
	}
	if profile.CreatedAt.IsZero() || profile.UpdatedAt.IsZero() || profile.LastUsedAt.IsZero() {
		t.Fatal("expected profile timestamps to be populated")
	}
}
