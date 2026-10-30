package runsignal

import (
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestDeriveEpisodesDeduplicatesRepeatedImportedEventWindows(t *testing.T) {
	pair := []InvocationFact{
		{CallEventID: "call-1", ToolCallID: "t1", Fingerprint: "same", Outcome: "success"},
		{CallEventID: "call-2", ToolCallID: "t2", Fingerprint: "same", Outcome: "success"},
	}
	episodes := DeriveEpisodes(append(append([]InvocationFact{}, pair...), pair...), nil)
	if len(episodes) != 1 {
		t.Fatalf("episodes=%#v; want one durable window for repeated imported event ID", episodes)
	}
	if episodes[0].Pattern != "repeated-work" || episodes[0].EpisodeID == "" {
		t.Fatalf("episode=%#v", episodes[0])
	}
}

func TestReadThenRereadSkipsVerificationAfterFileChange(t *testing.T) {
	first := &domain.RunEvent{ID: uuid.New(), Data: &domain.ToolCallEventData{ToolName: "read_file", Input: map[string]any{"path": "notes.md"}}}
	edit := &domain.RunEvent{ID: uuid.New(), Data: &domain.ToolCallEventData{ToolName: "file_change", Input: map[string]any{"files": []map[string]string{{"path": "notes.md", "kind": "modify"}}}}}
	second := &domain.RunEvent{ID: uuid.New(), Data: &domain.ToolCallEventData{ToolName: "read_file", Input: map[string]any{"path": "notes.md"}}}
	events := []*domain.RunEvent{first, edit, second}
	facts := []InvocationFact{
		{CallEventID: first.ID.String(), Capability: "file-read", Fingerprint: "same-file", Outcome: OutcomeSuccess},
		{CallEventID: second.ID.String(), Capability: "file-read", Fingerprint: "same-file", Outcome: OutcomeSuccess},
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
		{CallEventID: first.ID.String(), Capability: "file-read", Fingerprint: "same-file", Outcome: OutcomeSuccess},
		{CallEventID: second.ID.String(), Capability: "file-read", Fingerprint: "same-file", Outcome: OutcomeSuccess},
	}
	if got := detectReadThenReread(EpisodeDetectorContext{Facts: facts, Events: events, EventsByID: eventMap(events)}); len(got) != 1 {
		t.Fatalf("unchanged reread episodes=%d, want 1", len(got))
	}
}

func timedRead(at time.Time, path string) (*domain.RunEvent, *domain.RunEvent) {
	call := &domain.RunEvent{ID: uuid.New(), Timestamp: at, Data: &domain.ToolCallEventData{ToolName: "read_file", Input: map[string]any{"path": path}}}
	result := &domain.RunEvent{ID: uuid.New(), Timestamp: at.Add(1500 * time.Millisecond)}
	return call, result
}

func rereadFacts(calls ...*domain.RunEvent) []InvocationFact {
	facts := make([]InvocationFact, len(calls)/2)
	for i := range facts {
		facts[i] = InvocationFact{CallEventID: calls[2*i].ID.String(), ResultEventID: calls[2*i+1].ID.String(), Capability: "file-read", Fingerprint: "same-file", Outcome: OutcomeSuccess}
	}
	return facts
}

// A re-read is the same file read twice within one working stretch; a read
// weeks later is new work. The cost of a re-read is the second read's own
// duration, not the time between the reads.
func TestReadThenRereadPairsOnlyWithinOneWorkingStretch(t *testing.T) {
	start := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	firstCall, firstResult := timedRead(start, "notes.md")
	nearCall, nearResult := timedRead(start.Add(10*time.Minute), "notes.md")
	farCall, farResult := timedRead(start.Add(19*24*time.Hour), "notes.md")

	near := []*domain.RunEvent{firstCall, firstResult, nearCall, nearResult}
	got := detectReadThenReread(EpisodeDetectorContext{Facts: rereadFacts(near...), Events: near, EventsByID: eventMap(near)})
	if len(got) != 1 || got[0].WallClockMS != 1500 {
		t.Fatalf("a re-read ten minutes later must be one episode costing the second read's 1.5 s, got %+v", got)
	}

	far := []*domain.RunEvent{firstCall, firstResult, farCall, farResult}
	if got := detectReadThenReread(EpisodeDetectorContext{Facts: rereadFacts(far...), Events: far, EventsByID: eventMap(far)}); len(got) != 0 {
		t.Fatalf("a read 19 days later is not a re-read: %+v", got)
	}

	// Activity keeps a stretch alive: hourly events chain the reads together.
	chain := []*domain.RunEvent{firstCall, firstResult}
	for i := 1; i <= 5; i++ {
		chain = append(chain, &domain.RunEvent{ID: uuid.New(), Timestamp: start.Add(time.Duration(i) * 20 * time.Minute)})
	}
	lateCall, lateResult := timedRead(start.Add(120*time.Minute), "notes.md")
	chain = append(chain, lateCall, lateResult)
	if got := detectReadThenReread(EpisodeDetectorContext{Facts: rereadFacts(firstCall, firstResult, lateCall, lateResult), Events: chain, EventsByID: eventMap(chain)}); len(got) != 1 {
		t.Fatalf("reads in one continuous stretch must pair, got %+v", got)
	}
}

func bashEvent(at time.Time, id, command string) *domain.RunEvent {
	return &domain.RunEvent{ID: uuid.New(), Timestamp: at, Data: &domain.ToolCallEventData{ToolName: "Bash", ToolCallID: id, Input: map[string]any{"command": command}}}
}

func resultEvent(at time.Time, id string) *domain.RunEvent {
	return &domain.RunEvent{ID: uuid.New(), Timestamp: at, Data: &domain.ToolResultEventData{ToolCallID: id}}
}

// An importer once appended a second copy of a run's events. A tool call whose
// events are duplicated is still one invocation: it yields one fact and pairs
// with nothing, so no "re-read" or "retry" forms from a call and its copy.
func TestDuplicatedToolCallEventsAreOneInvocationAndPairWithNothing(t *testing.T) {
	at := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	call := bashEvent(at, "toolu_1", "cat docs/SPEC.md")
	result := resultEvent(at.Add(2*time.Second), "toolu_1")
	copyOfCall := bashEvent(at.Add(6*time.Minute), "toolu_1", "cat docs/SPEC.md")
	copyOfResult := resultEvent(at.Add(6*time.Minute+2*time.Second), "toolu_1")
	events := []*domain.RunEvent{call, result, copyOfCall, copyOfResult}

	facts := DeriveInvocationFacts(events)
	if len(facts) != 1 || facts[0].CallEventID != call.ID.String() || facts[0].ResultEventID != result.ID.String() {
		t.Fatalf("facts = %+v, want the first call paired with its first result", facts)
	}
	if got := DeriveEpisodes(facts, events); len(got) != 0 {
		t.Fatalf("one call produced episodes %+v", got)
	}

	// Facts persisted before the fix still carry both copies; the detectors must
	// not pair them either.
	both := []InvocationFact{
		{CallEventID: call.ID.String(), ToolCallID: "toolu_1", Capability: "file-read", Fingerprint: "same", Outcome: "failure", Executable: "cat"},
		{CallEventID: copyOfCall.ID.String(), ToolCallID: "toolu_1", Capability: "file-read", Fingerprint: "same", Outcome: "failure", Executable: "cat"},
	}
	for _, episode := range DeriveEpisodes(both, events) {
		t.Errorf("a call and its own copy formed %s", episode.Pattern)
	}
	// Two distinct calls still pair.
	distinct := []InvocationFact{both[0], {CallEventID: copyOfCall.ID.String(), ToolCallID: "toolu_2", Capability: "file-read", Fingerprint: "same", Outcome: "failure", Executable: "cat"}}
	if got := DeriveEpisodes(distinct, events); len(got) == 0 {
		t.Error("two different calls with one fingerprint must still form an episode")
	}
}

// A pipeline filter reads standard input, not a file: `| head -30` is not a
// re-read, while `head -30 file` and `sed -n 1,80p file` are.
func TestPipelineFiltersAreNotFileReads(t *testing.T) {
	at := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	cases := map[string]string{
		"plan-manager plans list | head -30": "shell",
		"git log | tail -n 5":                "shell",
		"sed -n 1,80p money-ledger/PRD.md":   "file-read",
		"head -30 docs/README.md":            "file-read",
		"cat docs/SPEC.md":                   "file-read",
		"cat":                                "shell",
		"jq . config.json | sed 's/a/b/'":    "shell",
	}
	for command, want := range cases {
		facts := DeriveInvocationFacts([]*domain.RunEvent{bashEvent(at, "t", command)})
		var got []string
		for _, f := range facts {
			got = append(got, f.Capability)
		}
		found := false
		for _, c := range got {
			found = found || c == want
		}
		if want == "shell" && containsString(got, "file-read") || want == "file-read" && !found {
			t.Errorf("%q has segment capabilities %v, want %s", command, got, want)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
