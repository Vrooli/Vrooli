package runsignal

import "testing"

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
