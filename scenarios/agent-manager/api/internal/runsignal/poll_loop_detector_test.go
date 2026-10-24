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

func TestPollLoopDetectorIgnoresInformationalQuotaObservations(t *testing.T) {
	start := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	facts, events := quotaObservationPollLoop(start)
	for _, second := range []int{5, 15} {
		percent := 47.0
		events = append(events, &domain.RunEvent{
			ID: uuid.New(), RunID: events[0].RunID, Timestamp: start.Add(time.Duration(second) * time.Second),
			EventType: domain.EventTypeError,
			Data: &domain.RateLimitEventData{
				Provider: "openai", Pool: "primary", UsedPercent: &percent,
				WindowMinutes: 10080, Provenance: "codex:event_msg.token_count.rate_limits",
			},
		})
	}
	got := detectPollLoops(EpisodeDetectorContext{Facts: facts, Events: events, EventsByID: eventMap(events)})
	if len(got) != 1 {
		t.Fatalf("informational quota observations hid repeated read-only work: episodes=%d", len(got))
	}
}

func TestPollLoopDetectorRetainsOperationalErrorTransitions(t *testing.T) {
	start := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		data domain.EventPayload
	}{
		{name: "exhausted quota", data: &domain.RateLimitEventData{UsedPercent: float64Pointer(100)}},
		{name: "unknown quota", data: &domain.RateLimitEventData{}},
		{name: "ordinary error", data: &domain.ErrorEventData{Code: "execution_error", Message: "failed"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			facts, events := quotaObservationPollLoop(start)
			events = append(events, &domain.RunEvent{
				ID: uuid.New(), RunID: events[0].RunID, Timestamp: start.Add(5 * time.Second),
				EventType: domain.EventTypeError, Data: tc.data,
			})
			got := detectPollLoops(EpisodeDetectorContext{Facts: facts, Events: events, EventsByID: eventMap(events)})
			if len(got) != 0 {
				t.Fatalf("operational error transition did not interrupt repeated-work episode: %+v", got)
			}
		})
	}
}

func TestIsOperationalErrorEventClassifiesNilAndQuotaPayloads(t *testing.T) {
	percent := 47.0
	exhausted := 100.0
	var nilQuota *domain.RateLimitEventData
	cases := []struct {
		name  string
		event *domain.RunEvent
		want  bool
	}{
		{name: "nil event", event: nil, want: false},
		{name: "typed nil quota is unknown error", event: &domain.RunEvent{EventType: domain.EventTypeError, Data: nilQuota}, want: true},
		{name: "unknown quota is operational error", event: &domain.RunEvent{EventType: domain.EventTypeError, Data: &domain.RateLimitEventData{}}, want: true},
		{name: "informational quota is not operational error", event: &domain.RunEvent{EventType: domain.EventTypeError, Data: &domain.RateLimitEventData{UsedPercent: &percent}}, want: false},
		{name: "exhausted quota is operational error", event: &domain.RunEvent{EventType: domain.EventTypeError, Data: &domain.RateLimitEventData{UsedPercent: &exhausted}}, want: true},
		{name: "ordinary error is operational error", event: &domain.RunEvent{EventType: domain.EventTypeError, Data: &domain.ErrorEventData{Code: "execution_error"}}, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := domain.IsOperationalErrorEvent(tc.event); got != tc.want {
				t.Fatalf("IsOperationalErrorEvent()=%t, want %t", got, tc.want)
			}
		})
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
