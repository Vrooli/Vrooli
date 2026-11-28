package hubsender

import "testing"

func TestSenderPrincipalsSendAsTheirOwnScenario(t *testing.T) {
	if got := Principal("treasury"); got != "treasury-notification-hub" {
		t.Fatalf("Principal = %q", got)
	}
	if got, ok := SourceOf("treasury-notification-hub"); !ok || got != "treasury" {
		t.Fatalf("SourceOf = %q, %v", got, ok)
	}
	// The other Hub role's name is never a sender, even though it shares the stem.
	for _, subject := range []string{"treasury-ask-reader", "treasury", "notification-hub", "ai-gateway"} {
		if got, ok := SourceOf(subject); ok {
			t.Fatalf("SourceOf(%q) = %q, want it refused", subject, got)
		}
	}
	want := "scenario-authenticator/service-principal/treasury-notification-hub/notification-hub"
	if got := CredentialIdentity("treasury"); got != want {
		t.Fatalf("CredentialIdentity = %q, want %q", got, want)
	}
}

// The sender and ask-reader audiences differ, so one role's token can never be
// replayed as the other's.
func TestSenderAudienceAndScopeAreDistinctFromTheAskReaders(t *testing.T) {
	if Audience == "scenario-authenticator:notification-hub-ask-reader" || Scope == "notification-hub:ask-read" {
		t.Fatal("sender and ask-reader roles must not share an audience or a scope")
	}
}
