package sessions

import "testing"

func TestSessionRunIdentityHandoffIsExactAndOneUse(t *testing.T) {
	const sessionID = "a244-fixture-session"
	if !putSessionRunIdentity(sessionID, "signed-child-token") {
		t.Fatal("failed to stage identity")
	}
	if putSessionRunIdentity(sessionID, "replacement-token") {
		t.Fatal("duplicate reserved session replaced the staged identity")
	}
	if _, ok := TakeSessionRunIdentity(sessionID + "-other"); ok {
		t.Fatal("identity was available to a different session")
	}
	if got, ok := TakeSessionRunIdentity(sessionID); !ok || got != "signed-child-token" {
		t.Fatalf("Take() = (%q, %t), want the exact staged token", got, ok)
	}
	if _, ok := TakeSessionRunIdentity(sessionID); ok {
		t.Fatal("identity handoff was not one-use")
	}
	if !putSessionRunIdentity(sessionID, "failed-create-token") {
		t.Fatal("failed to stage cleanup fixture")
	}
	deleteSessionRunIdentity(sessionID)
	if _, ok := TakeSessionRunIdentity(sessionID); ok {
		t.Fatal("failed Create left a token in the handoff")
	}
}
