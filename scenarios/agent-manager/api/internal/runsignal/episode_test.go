package runsignal

import (
	"testing"

	"agent-manager/internal/domain"
)

func TestDeriveEpisodesDeduplicatesRepeatedImportedEventWindows(t *testing.T) {
	facts := []InvocationFact{
		{CallEventID: "call-1", Fingerprint: "same", Outcome: "success"},
		{CallEventID: "call-1", Fingerprint: "same", Outcome: "success"},
	}
	episodes := DeriveEpisodes(facts, nil)
	if len(episodes) != 1 {
		t.Fatalf("episodes=%#v; want one durable window for repeated imported event ID", episodes)
	}
	if episodes[0].Pattern != "repeated-work" || episodes[0].EpisodeID == "" {
		t.Fatalf("episode=%#v", episodes[0])
	}
}

func TestWaitMisuseToleratesInterveningToolFacts(t *testing.T) {
	facts := []InvocationFact{
		{CallEventID: "wait-1", Capability: "wait"},
		{CallEventID: "read", Capability: "file-read", Fingerprint: "read"},
		{CallEventID: "wait-2", Capability: "wait"},
		{CallEventID: "write", Capability: "file-write", Fingerprint: "write"},
		{CallEventID: "wait-3", Capability: "wait"},
	}
	episodes := detectWaitMisuse(EpisodeDetectorContext{Facts: facts, EventsByID: map[string]*domain.RunEvent{}, Events: nil})
	if len(episodes) != 1 || episodes[0].Pattern != "wait-misuse" || episodes[0].CycleCount != 3 {
		t.Fatalf("episodes=%#v; want one three-cycle wait-misuse episode", episodes)
	}
}
