package world

import (
	"context"
	"sync"
	"time"

	"prompt-manager/internal/heartbeat"
)

// EventKind mirrors the proto enum without importing it.
type EventKind string

const (
	KindSnapshot           EventKind = "snapshot"
	KindRunStarted         EventKind = "run.started"
	KindRunFinished        EventKind = "run.finished"
	KindRunFailed          EventKind = "run.failed"
	KindHeartbeatUpcoming  EventKind = "heartbeat.upcoming"
	KindHeartbeatCancelled EventKind = "heartbeat.cancelled"
	KindAgentMessage       EventKind = "agent.message"
)

// ActiveRun is a running heartbeat as the snapshot reports it.
type ActiveRun struct {
	TeamID    string
	AgentID   string
	RunID     string
	StartedAt time.Time
}

// Upcoming is a scheduled heartbeat as the snapshot reports it.
type Upcoming struct {
	TeamID      string
	AgentID     string
	ScheduledAt time.Time
}

// Event is one world feed entry.
type Event struct {
	Kind        EventKind
	Seq         uint64
	At          time.Time
	AgentID     string
	TeamID      string
	RunID       string
	Message     string
	ScheduledAt time.Time
	ActiveRuns  []ActiveRun
	Upcoming    []Upcoming
}

// SnapshotSource provides the live lists the snapshot event carries.
type SnapshotSource interface {
	ActiveRuns() []ActiveRun
	UpcomingHeartbeats() []Upcoming
}

type subscriber struct {
	ch chan Event
}

// Hub buffers recent events and fans them out to subscribers. Publishing
// never blocks: a subscriber that falls behind by more than its channel
// capacity is dropped and must reconnect with since_seq.
type Hub struct {
	mu        sync.Mutex
	ring      []Event
	ringSize  int
	seq       uint64
	subs      map[*subscriber]struct{}
	source    SnapshotSource
	now       func() time.Time
	chanDepth int
}

// NewHub creates a hub keeping the last ringSize events.
func NewHub(ringSize, chanDepth int, source SnapshotSource) *Hub {
	if ringSize < 1 {
		ringSize = 1
	}
	if chanDepth < 1 {
		chanDepth = 1
	}
	return &Hub{ring: make([]Event, 0, ringSize), ringSize: ringSize, subs: map[*subscriber]struct{}{}, source: source, now: func() time.Time { return time.Now().UTC() }, chanDepth: chanDepth}
}

// Publish stamps and buffers an event, then delivers it to every subscriber.
func (h *Hub) Publish(event Event) Event {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.seq++
	event.Seq = h.seq
	if event.At.IsZero() {
		event.At = h.now()
	}
	if len(h.ring) == h.ringSize {
		copy(h.ring, h.ring[1:])
		h.ring = h.ring[:h.ringSize-1]
	}
	h.ring = append(h.ring, event)
	for sub := range h.subs {
		select {
		case sub.ch <- event:
		default:
			close(sub.ch)
			delete(h.subs, sub)
		}
	}
	return event
}

// Snapshot builds the opening event from the live source.
func (h *Hub) Snapshot() Event {
	event := Event{Kind: KindSnapshot, At: h.now()}
	if h.source != nil {
		event.ActiveRuns = h.source.ActiveRuns()
		event.Upcoming = h.source.UpcomingHeartbeats()
	}
	h.mu.Lock()
	event.Seq = h.seq
	h.mu.Unlock()
	return event
}

// Subscribe returns buffered events newer than sinceSeq plus a live channel.
// The channel closes when ctx ends or the subscriber falls behind.
func (h *Hub) Subscribe(ctx context.Context, sinceSeq uint64) (replay []Event, live <-chan Event) {
	sub := &subscriber{ch: make(chan Event, h.chanDepth)}
	h.mu.Lock()
	for _, event := range h.ring {
		if event.Seq > sinceSeq {
			replay = append(replay, event)
		}
	}
	h.subs[sub] = struct{}{}
	h.mu.Unlock()
	go func() {
		<-ctx.Done()
		h.mu.Lock()
		if _, ok := h.subs[sub]; ok {
			delete(h.subs, sub)
			close(sub.ch)
		}
		h.mu.Unlock()
	}()
	return replay, sub.ch
}

// SubscriberCount reports live subscribers (tests and diagnostics).
func (h *Hub) SubscriberCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs)
}

// RunStarted implements heartbeat.RunObserver.
func (h *Hub) RunStarted(run heartbeat.ActiveRun) {
	h.Publish(Event{Kind: KindRunStarted, At: run.StartedAt, AgentID: run.AgentID, TeamID: run.TeamID, RunID: run.RunID})
}

// RunEnded implements heartbeat.RunObserver.
func (h *Hub) RunEnded(run heartbeat.ActiveRun, endedAt time.Time, failed bool, message string) {
	kind := KindRunFinished
	if failed {
		kind = KindRunFailed
	}
	h.Publish(Event{Kind: kind, At: endedAt, AgentID: run.AgentID, TeamID: run.TeamID, RunID: run.RunID, Message: message})
}

// AgentMessage publishes free text attributed to an agent.
func (h *Hub) AgentMessage(agentID, teamID, message string) {
	h.Publish(Event{Kind: KindAgentMessage, AgentID: agentID, TeamID: teamID, Message: message})
}

var _ heartbeat.RunObserver = (*Hub)(nil)
