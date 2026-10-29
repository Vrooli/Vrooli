package runsignal

import (
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

var stallStart = time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)

func evt(kind domain.RunEventType, after time.Duration, data domain.EventPayload) *domain.RunEvent {
	return &domain.RunEvent{ID: uuid.New(), EventType: kind, Timestamp: stallStart.Add(after), Data: data}
}

func call(after time.Duration) *domain.RunEvent {
	return evt(domain.EventTypeToolCall, after, &domain.ToolCallEventData{ToolName: "Bash", ToolCallID: uuid.NewString()})
}

func result(after time.Duration) *domain.RunEvent {
	return evt(domain.EventTypeToolResult, after, &domain.ToolResultEventData{})
}

func say(role string, after time.Duration) *domain.RunEvent {
	return evt(domain.EventTypeMessage, after, &domain.MessageEventData{Role: role})
}
func status(after time.Duration) *domain.RunEvent { return evt(domain.EventTypeStatus, after, nil) }
func noise(kind domain.RunEventType, after time.Duration) *domain.RunEvent {
	return evt(kind, after, nil)
}

func stalls(events ...*domain.RunEvent) []FrictionEpisode {
	return detectStalls(EpisodeDetectorContext{Events: events, EventsByID: eventMap(events)})
}

// A stall is the agent sitting on its turn: an idle gap of 5 minutes to 2 hours
// between turn events, and nothing else.
func TestStallIsAnIdleGapInsideAnActiveTurn(t *testing.T) {
	min := time.Minute
	for name, tc := range map[string]struct {
		events []*domain.RunEvent
		want   int64 // stall cost in minutes, 0 for none
	}{
		"agent idle after a tool result":           {[]*domain.RunEvent{result(0), call(21 * min)}, 21},
		"usage and log noise falls inside the gap": {[]*domain.RunEvent{result(0), noise(domain.EventTypeMetric, 3*min), noise(domain.EventTypeLog, 9*min), say("assistant", 25*min)}, 25},
		"a gap under five minutes":                 {[]*domain.RunEvent{result(0), call(4 * min)}, 0},
		"a tool that ran for a long time":          {[]*domain.RunEvent{call(0), result(40 * min)}, 0},
		"waiting on the operator":                  {[]*domain.RunEvent{say("assistant", 0), say("user", 90*min)}, 0},
		"an operator reply after hours then work":  {[]*domain.RunEvent{say("user", 0), call(10 * min)}, 10},
		"a gap over two hours is not a stall":      {[]*domain.RunEvent{result(0), call(3 * time.Hour)}, 0},
		"a session resumed 41 days later":          {[]*domain.RunEvent{result(0), call(41 * 24 * time.Hour)}, 0},
		"across a park and wake":                   {[]*domain.RunEvent{result(0), status(time.Minute), status(50 * min), call(51 * min)}, 0},
		"across a lifecycle boundary":              {[]*domain.RunEvent{result(0), noise(domain.EventTypeLifecycle, 30*min), call(31 * min)}, 0},
	} {
		got := stalls(tc.events...)
		switch {
		case tc.want == 0 && len(got) != 0:
			t.Errorf("%s: %d stalls %+v, want none", name, len(got), got)
		case tc.want > 0 && (len(got) != 1 || got[0].WallClockMS != tc.want*60_000):
			t.Errorf("%s: stalls %+v, want one costing %d min", name, got, tc.want)
		}
	}
}

func TestWaitMisuseAndSlowStartStayWithinOneWorkingStretch(t *testing.T) {
	waits := func(events []*domain.RunEvent) []FrictionEpisode {
		facts := make([]InvocationFact, 0, len(events))
		for _, e := range events {
			if e.EventType == domain.EventTypeToolCall {
				facts = append(facts, InvocationFact{CallEventID: e.ID.String(), Capability: "wait", IntentClass: "wait:x"})
			}
		}
		return detectWaitMisuse(EpisodeDetectorContext{Facts: facts, Events: events, EventsByID: eventMap(events)})
	}
	min := time.Minute
	var continuous []*domain.RunEvent
	for i := 0; i < 4; i++ {
		continuous = append(continuous, call(time.Duration(i)*10*min), result(time.Duration(i)*10*min+time.Second))
	}
	if got := waits(continuous); len(got) != 1 || got[0].CycleCount != 4 {
		t.Fatalf("four waits ten minutes apart are one loop, got %+v", got)
	}
	// The same waits with a park in the middle, or a day of silence, are not one loop.
	parked := []*domain.RunEvent{call(0), result(time.Second), call(10 * min), result(10*min + time.Second), status(11 * min), call(12 * min), result(12*min + time.Second), call(13 * min), result(13*min + time.Second)}
	if got := waits(parked); len(got) != 0 {
		t.Fatalf("waits split by a park formed %+v", got)
	}
	silent := []*domain.RunEvent{call(0), result(time.Second), call(10 * min), result(10*min + time.Second), call(24 * time.Hour), result(24*time.Hour + time.Second), call(24*time.Hour + 10*min)}
	if got := waits(silent); len(got) != 0 {
		t.Fatalf("waits a day apart formed %+v", got)
	}

	// Time to first success: a slow start within one stretch counts; one that sat
	// through a park or a long silence does not.
	ttfs := func(events []*domain.RunEvent, success *domain.RunEvent) []FrictionEpisode {
		facts := []InvocationFact{{CallEventID: success.ID.String(), ResultEventID: success.ID.String(), Outcome: "success"}}
		return detectTimeToFirstSuccess(EpisodeDetectorContext{Facts: facts, Events: events, EventsByID: eventMap(events)})
	}
	first, ok := say("user", 0), result(20*min)
	if got := ttfs([]*domain.RunEvent{first, noise(domain.EventTypeMetric, 8*min), noise(domain.EventTypeMetric, 16*min), ok}, ok); len(got) != 1 || got[0].WallClockMS != 20*60_000 {
		t.Fatalf("a 20-minute start with activity throughout: %+v", got)
	}
	late := result(3 * time.Hour)
	if got := ttfs([]*domain.RunEvent{first, status(time.Minute), late}, late); len(got) != 0 {
		t.Fatalf("a start that waited out a park formed %+v", got)
	}
	if got := ttfs([]*domain.RunEvent{first, late}, late); len(got) != 0 {
		t.Fatalf("a start with three silent hours formed %+v", got)
	}
}
