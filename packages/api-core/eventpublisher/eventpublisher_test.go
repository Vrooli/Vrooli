package eventpublisher

import "testing"

func TestEventPublishersPublishAsTheirOwnScenario(t *testing.T) {
	if got := Principal("storage-manager"); got != "storage-manager-event-publisher" {
		t.Fatalf("Principal = %q", got)
	}
	if got, ok := SourceOf("storage-manager-event-publisher"); !ok || got != "storage-manager" {
		t.Fatalf("SourceOf = %q, %v", got, ok)
	}
	// Another Hub-facing role's name is never an event publisher.
	for _, subject := range []string{"storage-manager-ask-reader", "storage-manager-notification-hub", "storage-manager", "ai-gateway"} {
		if got, ok := SourceOf(subject); ok {
			t.Fatalf("SourceOf(%q) = %q, want it refused", subject, got)
		}
	}
	want := "scenario-authenticator/service-principal/storage-manager-event-publisher/vrooli-events"
	if got := CredentialIdentity("storage-manager"); got != want {
		t.Fatalf("CredentialIdentity = %q, want %q", got, want)
	}
}
