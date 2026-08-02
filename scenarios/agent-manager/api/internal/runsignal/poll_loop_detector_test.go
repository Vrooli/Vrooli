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
				events = append(events, &domain.RunEvent{ID: uuid.New(), Timestamp: start.Add(15 * time.Second), EventType: domain.EventTypeMetric, Data: &domain.CostEventData{InputTokens: tc.ExpectedTokens}})
			}
			got := detectPollLoops(EpisodeDetectorContext{Facts: facts, Events: events, EventsByID: eventMap(events)})
			if len(got) != tc.ExpectedCount {
				t.Fatalf("got %d, want %d", len(got), tc.ExpectedCount)
			}
			if tc.ExpectedCount == 1 && (got[0].WallClockMS != tc.ExpectedMS || got[0].Tokens != tc.ExpectedTokens) {
				t.Fatalf("episode=%+v, want wall_clock_ms=%d tokens=%d", got[0], tc.ExpectedMS, tc.ExpectedTokens)
			}
		})
	}
}

func eventMap(events []*domain.RunEvent) map[string]*domain.RunEvent {
	out := map[string]*domain.RunEvent{}
	for _, event := range events {
		out[event.ID.String()] = event
	}
	return out
}
