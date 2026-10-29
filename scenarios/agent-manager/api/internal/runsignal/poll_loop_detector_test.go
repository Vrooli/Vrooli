package runsignal

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestPollLoopDetectorRejectsRetriesAndSlowCalls(t *testing.T) {
	start := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	var fixtures []struct {
		Name           string `json:"name"`
		GapSeconds     int    `json:"gapSeconds"`
		StateChange    bool   `json:"stateChange"`
		ExpectedCount  int    `json:"expectedCount"`
		ExpectedMS     int64  `json:"expectedWallClockMs"`
		ExpectedTokens int    `json:"expectedTokens"`
	}
	raw, err := os.ReadFile("testdata/classification/poll-loop-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &fixtures); err != nil {
		t.Fatal(err)
	}
	makeEvents := func(gap time.Duration, message bool) ([]InvocationFact, []*domain.RunEvent) {
		facts, events := []InvocationFact{}, []*domain.RunEvent{}
		for i := 0; i < 3; i++ {
			call, result := &domain.RunEvent{ID: uuid.New(), Timestamp: start.Add(time.Duration(i) * gap), EventType: domain.EventTypeToolCall, Data: &domain.ToolCallEventData{Input: map[string]any{"status": true}}}, &domain.RunEvent{ID: uuid.New(), Timestamp: start.Add(time.Duration(i)*gap + time.Second), EventType: domain.EventTypeToolResult, Data: &domain.ToolResultEventData{Success: true}}
			events = append(events, call, result)
			facts = append(facts, InvocationFact{CallEventID: call.ID.String(), ResultEventID: result.ID.String(), Outcome: "success", Fingerprint: "same"})
			if message && i < 2 {
				events = append(events, &domain.RunEvent{ID: uuid.New(), Timestamp: start.Add(time.Duration(i)*gap + 2*time.Second), EventType: domain.EventTypeMessage, Data: &domain.MessageEventData{Role: "assistant"}})
			}
		}
		return facts, events
	}
	for _, tc := range fixtures {
		t.Run(tc.Name, func(t *testing.T) {
			facts, events := makeEvents(time.Duration(tc.GapSeconds)*time.Second, tc.StateChange)
			if tc.ExpectedTokens > 0 {
				events = append(events, &domain.RunEvent{ID: uuid.New(), Timestamp: start.Add(15 * time.Second), EventType: domain.EventTypeMetric, Data: &domain.UsageEventData{InputTokens: tc.ExpectedTokens}})
			}
			// Episode tokens come from the run's usage account in DeriveEpisodes.
			var got []FrictionEpisode
			for _, episode := range DeriveEpisodes(facts, events) {
				if episode.Pattern == "poll-loop" {
					got = append(got, episode)
				}
			}
			if len(got) != tc.ExpectedCount {
				t.Fatalf("got %d, want %d", len(got), tc.ExpectedCount)
			}
			if tc.ExpectedCount == 1 && (got[0].WallClockMS != tc.ExpectedMS || got[0].Tokens != tc.ExpectedTokens) {
				t.Fatalf("episode=%+v, want wall_clock_ms=%d tokens=%d", got[0], tc.ExpectedMS, tc.ExpectedTokens)
			}
		})
	}
}

func TestPollLoopDetectorIgnoresQuotaObservations(t *testing.T) {
	start := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	facts, events := quotaObservationPollLoop(start)
	for _, data := range []*domain.RateLimitEventData{
		{Provider: "openai", Pool: "primary", UsedPercent: float64Pointer(47), WindowMinutes: 10080},
		{Provider: "anthropic", Pool: "five_hour", Status: "rejected"},
	} {
		quota := domain.NewRateLimitEvent(events[0].RunID, data.Pool, "", nil, 0)
		quota.Timestamp, quota.Data = start.Add(5*time.Second), data
		events = append(events, quota)
	}
	got := detectPollLoops(EpisodeDetectorContext{Facts: facts, Events: events, EventsByID: eventMap(events)})
	if len(got) != 1 {
		t.Fatalf("quota observations hid repeated read-only work: episodes=%d", len(got))
	}
}

func TestPollLoopDetectorRetainsErrorTransitions(t *testing.T) {
	start := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	facts, events := quotaObservationPollLoop(start)
	failure := domain.NewErrorEvent(events[0].RunID, "execution_error", "failed", false)
	failure.Timestamp = start.Add(5 * time.Second)
	events = append(events, failure)
	got := detectPollLoops(EpisodeDetectorContext{Facts: facts, Events: events, EventsByID: eventMap(events)})
	if len(got) != 0 {
		t.Fatalf("error transition did not interrupt repeated-work episode: %+v", got)
	}
}

func quotaObservationPollLoop(start time.Time) ([]InvocationFact, []*domain.RunEvent) {
	var facts []InvocationFact
	var events []*domain.RunEvent
	runID := uuid.New()
	for i := 0; i < 3; i++ {
		call := &domain.RunEvent{
			ID: uuid.New(), RunID: runID, Timestamp: start.Add(time.Duration(i*10) * time.Second),
			EventType: domain.EventTypeToolCall,
			Data:      &domain.ToolCallEventData{Input: map[string]any{"status": true}},
		}
		result := &domain.RunEvent{
			ID: uuid.New(), RunID: runID, Timestamp: call.Timestamp.Add(time.Second),
			EventType: domain.EventTypeToolResult,
			Data:      &domain.ToolResultEventData{Success: true},
		}
		events = append(events, call, result)
		facts = append(facts, InvocationFact{
			CallEventID: call.ID.String(), ResultEventID: result.ID.String(),
			Outcome: "success", Fingerprint: "same",
		})
	}
	return facts, events
}

func float64Pointer(value float64) *float64 { return &value }

func eventMap(events []*domain.RunEvent) map[string]*domain.RunEvent {
	out := map[string]*domain.RunEvent{}
	for _, event := range events {
		out[event.ID.String()] = event
	}
	return out
}
