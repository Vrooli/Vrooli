package hub_test

import (
	"testing"

	"notification-hub/internal/hub"
)

func TestAnAudienceSeesAnAskThroughItsLineageAndAKeyOnlyThroughTheSameOwner(t *testing.T) {
	ask := hub.AskRecord{CreatedBy: "run:1", CreatedConversation: "thread-1", CreatedSubject: "owner-a"}
	for name, tc := range map[string]struct {
		audience hub.AskAudience
		sees     bool
		byKey    bool
	}{
		"the sending run":                  {hub.AskAudience{CreatedBy: "run:1"}, true, true},
		"a run of the conversation":        {hub.AskAudience{CreatedBy: "run:2", Conversation: "thread-1"}, true, true},
		"another conversation, same owner": {hub.AskAudience{CreatedBy: "run:3", Conversation: "thread-2", Subject: "owner-a"}, false, true},
		"another conversation, no owner":   {hub.AskAudience{CreatedBy: "run:3", Conversation: "thread-2"}, false, false},
		"another owner":                    {hub.AskAudience{CreatedBy: "run:4", Subject: "owner-b"}, false, false},
		"the owner":                        {hub.OwnerAudience(), true, true},
		"the zero audience":                {hub.AskAudience{}, false, false},
	} {
		if got := tc.audience.Sees(ask); got != tc.sees {
			t.Errorf("%s: Sees = %v, want %v", name, got, tc.sees)
		}
		if got := tc.audience.SeesByKey(ask); got != tc.byKey {
			t.Errorf("%s: SeesByKey = %v, want %v", name, got, tc.byKey)
		}
	}
}

func TestASubjectKeyIsNoCapabilityForAnAskWithoutAVerifiedOwnerOrAnEventAsk(t *testing.T) {
	reader := hub.AskAudience{CreatedBy: "run:9", Subject: "owner-a"}
	for name, ask := range map[string]hub.AskRecord{
		"an ask sent with no verified owner": {CreatedBy: "run:1"},
		"an event-sourced ask":               {CreatedSubject: "owner-a", Source: "autoheal", SourceEventType: "autoheal.x.decision_requested.v1"},
		"an ask of another owner":            {CreatedSubject: "owner-b"},
	} {
		if reader.SeesByKey(ask) {
			t.Errorf("%s must not be reachable by naming its key", name)
		}
	}
	if (hub.AskAudience{CreatedBy: "run:9"}).SeesByKey(hub.AskRecord{}) {
		t.Error("a reader with no verified owner must not match an ask with none")
	}
}

func TestDirectSourceNamespaceIsNotAuthorizationWithoutStoredAttestation(t *testing.T) {
	reader := hub.AskAudience{CreatedBy: "service:portfolio-steward", DirectSubjectSource: "portfolio-steward"}
	ask := hub.AskRecord{CreatedBy: "run:steward-pass", CreatedSubject: "owner-a", Source: "operator", SubjectKey: "portfolio-steward:roadmap:2026-11-16"}
	if reader.SeesByKey(ask) {
		t.Fatal("a subject prefix and verified sender identity alone must not authorize a direct ask")
	}
	for name, denied := range map[string]hub.AskRecord{
		"foreign subject":          {CreatedBy: "run:steward-pass", CreatedSubject: "owner-a", Source: "operator", SubjectKey: "treasury:approval:1"},
		"unverified creator owner": {CreatedBy: "", SubjectKey: "portfolio-steward:roadmap:1"},
		"event sourced ask":        {CreatedBy: "", CreatedSubject: "owner-a", Source: "portfolio-steward", SourceEventType: "portfolio_steward.decision_requested.v1", SubjectKey: "portfolio-steward:roadmap:1"},
	} {
		if reader.SeesByKey(denied) {
			t.Errorf("%s must not be reachable through the direct source path", name)
		}
	}
	other := hub.AskAudience{CreatedBy: "service:treasury", DirectSubjectSource: "treasury"}
	if other.SeesByKey(ask) {
		t.Fatal("a different verified sender must not read the steward ask")
	}
}
