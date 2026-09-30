package runsignal

import (
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
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

func TestReadThenRereadSkipsVerificationAfterFileChange(t *testing.T) {
	first := &domain.RunEvent{ID: uuid.New(), Data: &domain.ToolCallEventData{ToolName: "read_file", Input: map[string]any{"path": "notes.md"}}}
	edit := &domain.RunEvent{ID: uuid.New(), Data: &domain.ToolCallEventData{ToolName: "file_change", Input: map[string]any{"files": []map[string]string{{"path": "notes.md", "kind": "modify"}}}}}
	second := &domain.RunEvent{ID: uuid.New(), Data: &domain.ToolCallEventData{ToolName: "read_file", Input: map[string]any{"path": "notes.md"}}}
	events := []*domain.RunEvent{first, edit, second}
	facts := []InvocationFact{
		{CallEventID: first.ID.String(), Capability: "file-read", Fingerprint: "same-file"},
		{CallEventID: second.ID.String(), Capability: "file-read", Fingerprint: "same-file"},
	}
	if got := detectReadThenReread(EpisodeDetectorContext{Facts: facts, Events: events, EventsByID: eventMap(events)}); len(got) != 0 {
		t.Fatalf("verification reread was classified as repetition: %+v", got)
	}
}

func TestReadThenRereadReportsUnchangedFile(t *testing.T) {
	first := &domain.RunEvent{ID: uuid.New(), Data: &domain.ToolCallEventData{ToolName: "read_file", Input: map[string]any{"path": "notes.md"}}}
	second := &domain.RunEvent{ID: uuid.New(), Data: &domain.ToolCallEventData{ToolName: "read_file", Input: map[string]any{"path": "notes.md"}}}
	events := []*domain.RunEvent{first, second}
	facts := []InvocationFact{
		{CallEventID: first.ID.String(), Capability: "file-read", Fingerprint: "same-file"},
		{CallEventID: second.ID.String(), Capability: "file-read", Fingerprint: "same-file"},
	}
	if got := detectReadThenReread(EpisodeDetectorContext{Facts: facts, Events: events, EventsByID: eventMap(events)}); len(got) != 1 {
		t.Fatalf("unchanged reread episodes=%d, want 1", len(got))
	}
}
