// Investigation only: calls production modules with fake persistence and hub.
// No API, browser, database, or filesystem execution artifacts are started.
// From api/: GOPROXY=off GOTOOLCHAIN=local go run ../docs/internal/refactor_probes.go
// false is an observed mismatch against desired behavior, not a test-suite pass.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/automation/events"
	"github.com/vrooli/browser-automation-studio/performance"
	livecapture "github.com/vrooli/browser-automation-studio/services/live-capture"
	"github.com/vrooli/browser-automation-studio/services/recording"
	"github.com/vrooli/browser-automation-studio/services/recording/persistence"
	uxcollector "github.com/vrooli/browser-automation-studio/services/uxmetrics/collector"
	"github.com/vrooli/browser-automation-studio/websocket"
)

type result struct {
	ID       string `json:"id"`
	Expected string `json:"expected"`
	Actual   any    `json:"actual"`
	Met      bool   `json:"expected_behavior_met"`
}

type failingRepository struct {
	persistence.Repository
	saves int
}

func (r *failingRepository) AppendTimelineEntry(context.Context, *persistence.UnifiedTimelineEntry) (bool, error) {
	r.saves++
	return false, errors.New("synthetic persistence failure")
}

type countingRepository struct {
	persistence.Repository
	entries      []persistence.UnifiedTimelineEntry
	saves, reads int
}

func (r *countingRepository) AppendTimelineEntry(_ context.Context, entry *persistence.UnifiedTimelineEntry) (bool, error) {
	r.saves++
	entry.Sequence = r.saves
	r.entries = append(r.entries, *entry)
	return true, nil
}
func (r *countingRepository) GetTimeline(_ context.Context, q persistence.TimelineQuery) (*persistence.TimelineResponse, error) {
	r.reads++
	q.ApplyDefaults()
	end := min(q.Offset+q.Limit, len(r.entries))
	return &persistence.TimelineResponse{Entries: r.entries[q.Offset:end], TotalCount: len(r.entries), HasMore: end < len(r.entries)}, nil
}

type fakeHub struct {
	websocket.HubInterface
	seen   chan struct{}
	closed atomic.Int64
}

func (h *fakeHub) BroadcastEnvelope(any)    { h.seen <- struct{}{} }
func (h *fakeHub) CloseExecution(uuid.UUID) { h.closed.Add(1) }

type blockedHub struct {
	websocket.HubInterface
	entered chan struct{}
	release chan struct{}
	sends   chan any
}

func (h *blockedHub) BroadcastEnvelope(value any) {
	h.entered <- struct{}{}
	<-h.release
	h.sends <- value
}
func (h *blockedHub) CloseExecution(uuid.UUID) {}

func queues() int {
	buf := make([]byte, 4*1024*1024)
	n := runtime.Stack(buf, true)
	return strings.Count(string(buf[:n]), "events.(*executionQueue).run(")
}

func main() {
	ctx := context.Background()
	log := logrus.New()
	log.SetOutput(io.Discard)
	var results []result
	add := func(id, expected string, actual any, met bool) {
		results = append(results, result{id, expected, actual, met})
	}
	action := func(text, page, frame string) driver.RecordedAction {
		return driver.RecordedAction{
			ID: uuid.NewString(), Timestamp: "2026-09-22T00:00:00Z", ActionType: "type", PageID: page, FrameID: frame,
			Selector: &driver.SelectorSet{Primary: "#same"}, Payload: map[string]any{"text": text},
		}
	}
	merged := livecapture.MergeConsecutiveActions([]driver.RecordedAction{action("a", "one", ""), action("ab", "one", "")})
	add("merge-input-snapshots", "The latest full-value snapshot is ab", merged[0].Payload["text"], merged[0].Payload["text"] == "ab")
	multiPage := livecapture.MergeConsecutiveActions([]driver.RecordedAction{action("first", "one", ""), action("second", "two", "")})
	add("merge-page-identity", "Identical selectors in separate pages retain two actions", len(multiPage), len(multiPage) == 2)
	multiFrame := livecapture.MergeConsecutiveActions([]driver.RecordedAction{action("first", "one", "frame-a"), action("second", "one", "frame-b")})
	add("merge-frame-identity", "Identical selectors in separate frames retain two actions", len(multiFrame), len(multiFrame) == 2)

	repo := &failingRepository{}
	broadcasts := 0
	svc := recording.NewService(repo, recording.ServiceConfig{OnAction: func(string, *persistence.UnifiedTimelineEntry) { broadcasts++ }})
	synthetic := action("synthetic", "one", "")
	err := svc.RecordAction(ctx, "synthetic-session", &synthetic, uuid.New(), recording.ActionSourceAuto)
	add("recording-persistence-ack", "A failed durable write does not report success", map[string]any{"save_attempts": repo.saves, "returned_error": err != nil, "broadcasts": broadcasts}, err != nil)
	counting := &countingRepository{}
	longSession := recording.NewService(counting, recording.ServiceConfig{})
	for i := 0; i < 1001; i++ {
		synthetic.ID = uuid.NewString()
		if err := longSession.RecordAction(ctx, "long-session", &synthetic, uuid.Nil, recording.ActionSourceAuto); err != nil {
			panic(err)
		}
	}
	first, _ := longSession.GetTimeline(ctx, persistence.TimelineQuery{SessionID: "long-session", Limit: 100})
	next, _ := longSession.GetTimeline(ctx, persistence.TimelineQuery{SessionID: "long-session", Limit: 100, Offset: 100})
	add("timeline-cache-pagination", "A 1001-event durable session reports 1001 and offset changes the page", map[string]any{
		"durable_saves": counting.saves, "reported_total": first.TotalCount, "database_reads": counting.reads,
		"first_page_start_sequence": first.Entries[0].Sequence, "offset_page_start_sequence": next.Entries[0].Sequence,
	}, first.TotalCount == 1001 && first.Entries[0].Sequence != next.Entries[0].Sequence)

	frames := performance.NewCollector("synthetic", 30, 2)
	for i := 0; i < 3; i++ {
		frames.Record(&performance.FrameTimings{Skipped: true})
	}
	for i := 0; i < 2; i++ {
		frames.Record(&performance.FrameTimings{FrameBytes: 1000})
	}
	stats := frames.GetAggregated()
	add("frame-window-accounting", "Two retained non-skipped 1000-byte frames average 1000 bytes", map[string]any{"retained_frames": len(frames.GetRecentFrames(0)), "lifetime_frames": stats.FrameCount, "lifetime_skips": stats.SkippedCount, "average_frame_bytes": stats.AvgFrameBytes}, stats.AvgFrameBytes == 1000)

	// Exercise the production sink lifecycle through its actual wrapper.
	// This is an isolated composition probe, not a full workflow execution.
	hub := &fakeHub{seen: make(chan struct{}, 24)}
	baseCount := queues()
	var bases []*events.WSHubSink
	var ids []uuid.UUID
	for i := 0; i < 24; i++ {
		id := uuid.New()
		base := events.NewWSHubSink(hub, log, contracts.DefaultEventBufferLimits)
		var sink events.Sink = uxcollector.NewCollector(base, nil)
		_ = sink.Publish(ctx, contracts.EventEnvelope{ExecutionID: id, Kind: contracts.EventKindExecutionCompleted, Payload: map[string]any{"status": "completed"}})
		sink.CloseExecution(id)
		bases = append(bases, base)
		ids = append(ids, id)
	}
	for i := 0; i < 24; i++ {
		select {
		case <-hub.seen:
		case <-time.After(time.Second):
			panic("synthetic hub did not receive event")
		}
	}
	deadline := time.Now().Add(time.Second)
	for queues() > baseCount && time.Now().Before(deadline) {
		runtime.Gosched()
	}
	leaked := queues() - baseCount
	add("wrapped-sink-cleanup", "Completing 24 wrapped sinks closes their execution queues", map[string]any{"remaining_queue_goroutines": leaked, "hub_close_calls": hub.closed.Load()}, leaked == 0 && hub.closed.Load() == 24)
	for i, base := range bases {
		base.CloseExecution(ids[i])
	}
	deadline = time.Now().Add(time.Second)
	for queues() > baseCount && time.Now().Before(deadline) {
		runtime.Gosched()
	}
	add("probe-resource-cleanup", "Explicitly closing the underlying sinks releases all probe queues", queues()-baseCount, queues() == baseCount)

	blocked := &blockedHub{entered: make(chan struct{}, 2), release: make(chan struct{}), sends: make(chan any, 2)}
	sink := events.NewWSHubSink(blocked, log, contracts.DefaultEventBufferLimits)
	id := uuid.New()
	_ = sink.Publish(ctx, contracts.EventEnvelope{ExecutionID: id, Kind: contracts.EventKindExecutionStarted})
	select {
	case <-blocked.entered:
	case <-time.After(time.Second):
		panic("blocked hub not entered")
	}
	_ = sink.Publish(ctx, contracts.EventEnvelope{ExecutionID: id, Kind: contracts.EventKindExecutionCompleted})
	sink.CloseExecution(id)
	close(blocked.release)
	deadline = time.Now().Add(time.Second)
	for queues() > baseCount && time.Now().Before(deadline) {
		runtime.Gosched()
	}
	add("sink-close-drains-terminal-event", "Closing a queue delivers its already-accepted terminal event", map[string]any{"published_events": 2, "delivered_events": len(blocked.sends)}, len(blocked.sends) == 2)
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"schema_version": 1, "observed_at": time.Now().UTC(), "scope": "isolated production-module composition and pure-function probes; no live workflows", "results": results})
}
