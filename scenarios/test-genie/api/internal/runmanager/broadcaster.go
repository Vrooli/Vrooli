package runmanager

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
)

// ErrStaleCursor means the requested replay point was evicted by the bounded
// tail. Callers must use durable evidence/detail endpoints rather than asking
// the run to retain more live history.
var ErrStaleCursor = errors.New("run event cursor is older than retained replay")

// subscriberBuffer is the per-follower channel depth. A run emits on the order
// of tens-to-low-hundreds of events (phase boundaries, significant log lines,
// occasional heartbeats), so an actively-draining follower never fills this;
// only a follower that has stopped reading (already detached/dead) can, and a
// non-blocking publish simply skips it without stalling the run.
const (
	defaultSubscriberBuffer = 64
	defaultReplayEventLimit = 512
	defaultReplayByteLimit  = 1 << 20 // 1 MiB
	maxEventMessageBytes    = 4 << 10 // 4 KiB
)

// replayLimits are deliberately owned by the run lifecycle rather than a
// transport. A reconnect must never make the server retain an unbounded run.
type replayLimits struct {
	events int
	bytes  int
}

func replayLimitsFromEnvironment() replayLimits {
	return replayLimits{
		events: boundedPositiveEnv("TEST_GENIE_EVENT_REPLAY_MAX_EVENTS", defaultReplayEventLimit),
		bytes:  boundedPositiveEnv("TEST_GENIE_EVENT_REPLAY_MAX_BYTES", defaultReplayByteLimit),
	}
}

func boundedPositiveEnv(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

// broadcaster fans canonical run events out to N live followers. Replay is a
// bounded diagnostic tail, not a second owner of execution evidence. A late
// follower that needs older or terminal detail must use the durable detail
// surface rather than forcing an active run to retain every event.
type broadcaster struct {
	mu           sync.Mutex
	history      []sizedEvent
	historyBytes int
	limits       replayLimits
	subs         map[*subscriber]struct{}
	closed       bool
	nextSequence uint64
}

type sizedEvent struct {
	event Event
	bytes int
}

type subscriber struct {
	ch chan Event
}

func newBroadcaster() *broadcaster {
	return newBroadcasterWithLimits(replayLimitsFromEnvironment())
}

func newBroadcasterWithLimits(limits replayLimits) *broadcaster {
	if limits.events < 1 {
		limits.events = defaultReplayEventLimit
	}
	if limits.bytes < 1 {
		limits.bytes = defaultReplayByteLimit
	}
	return &broadcaster{limits: limits, subs: make(map[*subscriber]struct{})}
}

// publish appends a compact event to the bounded replay tail and delivers it to
// every live follower. Delivery is non-blocking: a follower whose buffer is
// full is skipped rather than stalling suite execution.
func (b *broadcaster) publish(ev Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	ev = compactEvent(ev)
	b.nextSequence++
	ev.Sequence = b.nextSequence
	entry := sizedEvent{event: ev, bytes: eventSize(ev)}
	b.history = append(b.history, entry)
	b.historyBytes += entry.bytes
	for len(b.history) > b.limits.events || (len(b.history) > 1 && b.historyBytes > b.limits.bytes) {
		b.historyBytes -= b.history[0].bytes
		b.history[0] = sizedEvent{}
		b.history = b.history[1:]
	}
	for s := range b.subs {
		select {
		case s.ch <- ev:
		default:
		}
	}
}

func compactEvent(ev Event) Event {
	// Do not let an event accidentally become an alternative owner of detailed
	// execution evidence. The only potentially open-ended wire field is message.
	ev.Message = truncateEventMessage(ev.Message)
	return ev
}

func truncateEventMessage(message string) string {
	if len(message) <= maxEventMessageBytes {
		return message
	}
	return message[:maxEventMessageBytes] + "… [truncated]"
}

func eventSize(ev Event) int {
	return len(ev.Kind) + len(ev.RunID) + len(ev.Scenario) + len(ev.ArtifactDir) +
		len(ev.Preset) + len(ev.Phase) + len(ev.Status) + len(ev.Message) +
		len(ev.Verdict) + len(ev.Error) + 128
}

// subscribe returns the replay history (events published before now) plus a
// live channel for subsequent events. The live channel closes when the run
// terminates (close) or the returned cancel removes the follower.
func (b *broadcaster) subscribe() (replay []Event, ch <-chan Event, cancel func()) {
	replay, ch, cancel, _ = b.subscribeAfter(0)
	return replay, ch, cancel
}

// subscribeAfter returns only events strictly newer than after. A non-zero
// cursor older than the retained ring is rejected explicitly rather than
// silently presenting an incomplete event history as complete.
func (b *broadcaster) subscribeAfter(after uint64) (replay []Event, ch <-chan Event, cancel func(), err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if after > 0 && len(b.history) > 0 && after < b.history[0].event.Sequence-1 {
		return nil, nil, nil, ErrStaleCursor
	}

	replay = make([]Event, 0, len(b.history))
	for _, entry := range b.history {
		if entry.event.Sequence > after {
			replay = append(replay, entry.event)
		}
	}
	s := &subscriber{ch: make(chan Event, defaultSubscriberBuffer)}

	if b.closed {
		// Run already terminal: the bounded diagnostic tail may contain the
		// terminal boundary. Durable evidence remains the source for complete
		// terminal detail, so hand back an already-closed live channel.
		close(s.ch)
		return replay, s.ch, func() {}, nil
	}

	b.subs[s] = struct{}{}
	cancel = func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.subs[s]; ok {
			delete(b.subs, s)
			close(s.ch)
		}
	}
	return replay, s.ch, cancel, nil
}

// close marks the broadcaster terminal and closes every live follower channel.
func (b *broadcaster) close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for s := range b.subs {
		close(s.ch)
		delete(b.subs, s)
	}
}
