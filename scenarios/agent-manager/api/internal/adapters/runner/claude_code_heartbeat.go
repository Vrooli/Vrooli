package runner

import (
	"agent-manager/internal/domain"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// heartbeat tracks stream activity and fires debug log events when the
// stdout pipe has been quiet for longer than the configured threshold.
//
// A single heartbeat instance services one Execute/Continue call; start()
// must be called before the scanner loop and stop() after it exits. The
// goroutine exits cleanly on stop() via the done channel.
type heartbeat struct {
	runID         uuid.UUID
	sink          EventSink
	lastEventNs   atomic.Int64  // unix nano of most recent event
	lastSummary   atomic.Value  // string — short description of most recent event
	reportedIdle  atomic.Bool   // true while we've already emitted a warning for the current gap
	done          chan struct{} // closed by stop() to signal the goroutine to exit
	thresholdMs   int64
	tickMs        int64
	stopOnce      atomic.Bool
}

func newHeartbeat(runID uuid.UUID, sink EventSink) *heartbeat {
	hb := &heartbeat{
		runID:       runID,
		sink:        sink,
		done:        make(chan struct{}),
		thresholdMs: streamIdleHeartbeatMillis,
		tickMs:      streamIdleHeartbeatTickMillis,
	}
	hb.lastEventNs.Store(time.Now().UnixNano())
	hb.lastSummary.Store("<run-started>")
	return hb
}

// start launches the background goroutine. Safe to call even when sink is
// nil — the goroutine exits immediately in that case so callers don't have
// to guard every call-site.
func (h *heartbeat) start() {
	if h.sink == nil || h.thresholdMs <= 0 {
		return
	}
	go h.loop()
}

// stop signals the goroutine to exit. Safe to call multiple times.
func (h *heartbeat) stop() {
	if h.stopOnce.CompareAndSwap(false, true) {
		close(h.done)
	}
}

// recordEvent marks that a new stream event has just been observed. This
// resets the idle window and clears the "already reported" flag so the
// next gap will produce a fresh warning.
func (h *heartbeat) recordEvent(summary string) {
	h.lastEventNs.Store(time.Now().UnixNano())
	if summary != "" {
		h.lastSummary.Store(summary)
	}
	h.reportedIdle.Store(false)
}

func (h *heartbeat) loop() {
	tick := time.Duration(h.tickMs) * time.Millisecond
	if tick <= 0 {
		tick = 500 * time.Millisecond
	}
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-h.done:
			return
		case <-ticker.C:
			elapsed := time.Duration(time.Now().UnixNano()-h.lastEventNs.Load()) * time.Nanosecond
			thr := time.Duration(h.thresholdMs) * time.Millisecond
			if elapsed < thr {
				continue
			}
			// Only emit once per idle gap; the next real event will clear
			// the flag and re-arm the warning.
			if !h.reportedIdle.CompareAndSwap(false, true) {
				continue
			}
			last := ""
			if v, ok := h.lastSummary.Load().(string); ok {
				last = v
			}
			_ = h.sink.Emit(domain.NewLogEvent(
				h.runID,
				"debug",
				fmt.Sprintf("stream idle for %s (last event: %s)", elapsed.Truncate(time.Second), last),
			))
		}
	}
}

// eventSummary renders a one-liner describing an event so heartbeat messages
// can point investigators at the last thing the stream did. Intentionally
// short — this is for an idle-log line, not structured data.
func eventSummary(event *domain.RunEvent) string {
	if event == nil {
		return ""
	}
	switch d := event.Data.(type) {
	case *domain.LogEventData:
		return "log:" + d.Message
	case *domain.MessageEventData:
		return "message:" + d.Role
	case *domain.ToolCallEventData:
		return "tool_call:" + d.ToolName
	case *domain.ToolResultEventData:
		return "tool_result"
	case *domain.StatusEventData:
		return "status:" + d.NewStatus
	case *domain.ErrorEventData:
		return "error:" + d.Code
	default:
		return string(event.EventType)
	}
}
