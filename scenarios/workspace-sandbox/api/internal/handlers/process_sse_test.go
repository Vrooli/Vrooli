// SSE-specific live-HTTP tests covering the StreamProcessLogs handler.
//
// Why these tests exist (Round 4 Phase 3):
//
// The 2026-04-28 SSE flusher bug shipped because no test exercised the
// live HTTP middleware. httptest.ResponseRecorder natively implements
// http.Flusher, so the missing Flusher pass-through in the
// responseWriter wrapper was invisible until production sent a real
// SSE response and 500'd. These tests close that gap by:
//
//  1. Booting the production middleware + handler stack via
//     testutil/httpx.NewLiveServer.
//  2. Driving the real *process.Logger and *process.Tracker with
//     temp-dir-backed log files so the SSE pipeline behaves exactly
//     like production.
//  3. Asserting the wire contract `data*, event: exit, event: end`
//     using the strict SSE frame parser.
//
// Frame-ordering invariants tested:
//   - `event: end` never precedes `event: exit` in any successful run.
//   - Fast-exit processes still emit `event: exit` (with structured
//     ExitInfo JSON) — the regression case from the 2026-04-28 incident.
//   - Multi-subscriber fanout: every concurrent client sees the full
//     sequence; a mid-stream disconnect of one subscriber does not
//     break the other.

package handlers_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/clock"
	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/handlers"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/sse"
	"workspace-sandbox/internal/testutil/assertx"
	"workspace-sandbox/internal/testutil/httpx"
	"workspace-sandbox/internal/testutil/mocks"
	"workspace-sandbox/internal/testutil/mocks/sandboxiface"
	"workspace-sandbox/internal/types"
)

// sseFixture pre-wires a *handlers.Handlers with real Logger + Tracker
// rooted at a per-test temp dir, plus the FakeService that feeds the
// sandbox lookup. Returns the harness, the live server, and the chosen
// (sandboxID, pid) the test should use to write/exit.
type sseFixture struct {
	live    *httpx.LiveServer
	logger  *process.Logger
	tracker *process.Tracker
	pending *process.PendingLogPair
	id      uuid.UUID
	pid     int
}

func newSSEFixture(t *testing.T) *sseFixture {
	t.Helper()

	clk := clock.System{}
	tmp := t.TempDir()
	logger := process.NewLogger(process.LogConfig{BaseDir: tmp}, clk)
	tracker := process.NewTracker(clk)

	id := uuid.New()
	sb := &types.Sandbox{
		ID:        id,
		ScopePath: "/project/src", ProjectRoot: "/project",
		Status: types.StatusActive, DriverID: "overlayfs-userns",
	}
	svc := &sandboxiface.FakeService{
		GetFn: func(ctx context.Context, gid uuid.UUID) (*types.Sandbox, error) {
			return sb, nil
		},
	}

	pending, err := logger.CreatePendingLogPair(id)
	if err != nil {
		t.Fatalf("CreatePendingLogPair: %v", err)
	}

	// Pick a synthetic PID that is unlikely to collide with anything
	// the OS might report. The handler doesn't actually probe /proc;
	// it just identifies the tracked entry by (sandboxID, pid).
	pid := 1_000_000 + int(time.Now().UnixNano()%900_000)

	if _, _, err := logger.FinalizePair(pending, pid); err != nil {
		t.Fatalf("FinalizePair: %v", err)
	}
	if _, err := tracker.Track(id, pid, "test-cmd", ""); err != nil {
		t.Fatalf("tracker.Track: %v", err)
	}

	h := &handlers.Handlers{
		Clock:          clk,
		DB:             mocks.NewFakePinger(),
		DriverSlot:     driver.NewSlot(mocks.NewFakeDriver()),
		Service:        svc,
		Config:         config.Config{},
		ProcessTracker: tracker,
		ProcessLogger:  logger,
	}
	live := httpx.NewLiveServer(t, h)
	return &sseFixture{
		live:    live,
		logger:  logger,
		tracker: tracker,
		pending: pending,
		id:      id,
		pid:     pid,
	}
}

// streamURL builds the canonical /processes/{pid}/logs/stream URL
// for this fixture's sandbox + PID, requesting stdout.
func (f *sseFixture) streamURL() string {
	return f.live.URL("/api/v1/sandboxes/" + f.id.String() +
		"/processes/" + intToStr(f.pid) + "/logs/stream?stream=stdout")
}

// markExited writes the exit footer + closes the writer (so
// subscribers see EOF) and tells the tracker the process is done.
// Mirrors the production sequence in process_start.go (RecordExit then
// CloseLogPair).
func (f *sseFixture) markExited(t *testing.T, info process.ExitInfo) {
	t.Helper()
	f.tracker.RecordExit(f.id, f.pid, info)
	if err := f.logger.CloseLogPair(f.id, f.pid, info); err != nil {
		t.Fatalf("CloseLogPair: %v", err)
	}
}

func intToStr(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var buf [20]byte
	n := len(buf)
	for i > 0 {
		n--
		buf[n] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		n--
		buf[n] = '-'
	}
	return string(buf[n:])
}

// findFrame returns the first frame matching `event` or nil.
func findFrame(frames []sse.Frame, event string) *sse.Frame {
	for i := range frames {
		if frames[i].Event == event {
			return &frames[i]
		}
	}
	return nil
}

// indexOf returns the index of the first frame with the given event,
// or -1.
func indexOf(frames []sse.Frame, event string) int {
	for i, f := range frames {
		if f.Event == event {
			return i
		}
	}
	return -1
}

// TestStreamProcessLogs_FastExit is the regression gate for the
// 2026-04-28 SSE flusher bug. The process exits before the SSE consumer
// subscribes; the on-disk log content is replayed, then `event: exit`
// must arrive carrying the structured ExitInfo, then `event: end`.
//
// Pre-Round-3 behavior: missing Flusher pass-through dropped the exit
// frame and the agent-manager log consumer wrote
// ErrSandboxNoExitInfo (silently treating the failure as success).
func TestStreamProcessLogs_FastExit(t *testing.T) {
	f := newSSEFixture(t)

	// Process emits a chunk and exits with code 1 BEFORE the consumer
	// subscribes. CloseLogPair removes the writer from the registry;
	// StreamLog will replay disk content and short-circuit Subscribe.
	if _, err := f.pending.Stdout.Write([]byte("hello, world\n")); err != nil {
		t.Fatalf("write chunk: %v", err)
	}
	exit := process.ExitInfo{
		ExitCode: 1, Signal: 0, OOMKilled: false,
		StoppedAt: time.Now().UTC(),
	}
	f.markExited(t, exit)

	resp, body := f.live.Do(t, "GET", f.streamURL(), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}

	frames := sse.ParseStream(strings.NewReader(string(body)))

	if exitFrame := findFrame(frames, "exit"); exitFrame == nil {
		t.Fatalf("missing event: exit (regression of 2026-04-28 SSE flusher bug); frames=%+v", frames)
	} else {
		var got process.ExitInfo
		if err := json.Unmarshal(exitFrame.Data, &got); err != nil {
			t.Errorf("exit data not valid JSON: %v (data=%q)", err, exitFrame.Data)
		} else if got.ExitCode != 1 {
			t.Errorf("ExitInfo.ExitCode = %d, want 1", got.ExitCode)
		}
	}
	if endFrame := findFrame(frames, "end"); endFrame == nil {
		t.Fatalf("missing event: end; frames=%+v", frames)
	}

	// Frame-ordering invariant: end never precedes exit.
	exitIdx := indexOf(frames, "exit")
	endIdx := indexOf(frames, "end")
	if exitIdx >= endIdx {
		t.Errorf("frame ordering invariant violated: exit at %d, end at %d (end must come AFTER exit)", exitIdx, endIdx)
	}

	// Replayed data must reach the client as a structured default-event
	// (`message`) frame, not just raw body bytes. The Phase-5 SSE writer
	// owns the multi-line encoding, so per-frame assertions are the
	// authoritative test surface here.
	dataFrames := framesByEvent(frames, "")
	if len(dataFrames) == 0 {
		t.Fatalf("expected ≥ 1 default-event frame carrying replayed content; frames=%+v", frames)
	}
	joined := joinFrameData(dataFrames)
	if !strings.Contains(joined, "hello, world") {
		t.Errorf("default-event frames missing replayed 'hello, world'; got=%q", joined)
	}
}

// TestStreamProcessLogs_SlowExit covers the streaming case: chunks are
// produced over the lifetime of the SSE connection, then exit. The
// frame-ordering invariant must still hold (end is last) and every
// chunk written must reach the client.
func TestStreamProcessLogs_SlowExit(t *testing.T) {
	f := newSSEFixture(t)

	// Issue the SSE request in a goroutine so the handler can subscribe
	// before chunks are written.
	type result struct {
		body []byte
		err  error
	}
	resCh := make(chan result, 1)
	go func() {
		req, _ := http.NewRequest("GET", f.streamURL(), nil)
		resp, err := f.live.Client.Do(req)
		if err != nil {
			resCh <- result{err: err}
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		resCh <- result{body: body, err: err}
	}()

	// Wait briefly so the subscribe path runs.
	time.Sleep(50 * time.Millisecond)

	// Stream three chunks over ~150ms.
	for _, msg := range []string{"chunk-A\n", "chunk-B\n", "chunk-C\n"} {
		if _, err := f.pending.Stdout.Write([]byte(msg)); err != nil {
			t.Fatalf("write: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	f.markExited(t, process.ExitInfo{ExitCode: 0, StoppedAt: time.Now().UTC()})

	r := <-resCh
	if r.err != nil {
		t.Fatalf("client.Do: %v", r.err)
	}

	frames := sse.ParseStream(strings.NewReader(string(r.body)))

	exitIdx := indexOf(frames, "exit")
	endIdx := indexOf(frames, "end")
	if exitIdx < 0 || endIdx < 0 {
		t.Fatalf("missing exit/end frames; frames=%+v", frames)
	}
	if exitIdx >= endIdx {
		t.Errorf("end must come after exit (got exit=%d, end=%d)", exitIdx, endIdx)
	}

	// Every chunk we wrote must reach the client as one or more
	// structured default-event frames (no raw-body fallback). The
	// Phase-5 SSE writer guarantees multi-line data is round-trip safe.
	dataFrames := framesByEvent(frames, "")
	if len(dataFrames) == 0 {
		t.Fatalf("expected ≥ 1 default-event data frame; frames=%+v", frames)
	}
	joined := joinFrameData(dataFrames)
	for _, want := range []string{"chunk-A", "chunk-B", "chunk-C"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing chunk %q in concatenated data frames: %q", want, joined)
		}
	}
}

// TestStreamProcessLogs_FrameOrderingInvariant pins the ordering
// contract directly: across multiple successful runs (with different
// chunk patterns), the parsed frame sequence must end with `end`, and
// `exit` must precede `end`.
func TestStreamProcessLogs_FrameOrderingInvariant(t *testing.T) {
	// One of the subtests is named "I-SSE-1" so the invariant ID listed
	// in docs/internal/INVARIANTS.md has a t.Run home that
	// scripts/check-invariants.sh picks up.
	patterns := []struct {
		name   string
		chunks []string
	}{
		{name: "I-SSE-1", chunks: nil},
		{name: "single-chunk", chunks: []string{"only\n"}},
		{name: "many-chunks", chunks: []string{"a\n", "b\n", "c\n", "d\n", "e\n"}},
	}

	for _, tc := range patterns {
		t.Run(tc.name, func(t *testing.T) {
			f := newSSEFixture(t)
			for _, c := range tc.chunks {
				if _, err := f.pending.Stdout.Write([]byte(c)); err != nil {
					t.Fatalf("write: %v", err)
				}
			}
			f.markExited(t, process.ExitInfo{ExitCode: 0, StoppedAt: time.Now().UTC()})

			resp, body := f.live.Do(t, "GET", f.streamURL(), nil)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d", resp.StatusCode)
			}
			frames := sse.ParseStream(strings.NewReader(string(body)))

			// Spec: every successful run ends with `end`, and `end`
			// is preceded by `exit`.
			assertx.AssertSSEFrameSequence(t, last2(frames), []assertx.FrameSpec{
				{Event: "exit"},
				{Event: "end"},
			})
		})
	}
}

// last2 returns the last two frames of `frames`, or all of them if
// fewer than two exist.
func last2(frames []sse.Frame) []sse.Frame {
	if len(frames) <= 2 {
		return frames
	}
	return frames[len(frames)-2:]
}

// TestStreamProcessLogs_MultiSubscriberFanout opens two concurrent SSE
// streams against the same (sandbox, pid). Both subscribers must see
// every chunk written to the underlying log, and the exit/end frame
// pair, regardless of subscription order. Mirrors agent-manager log
// streaming where a UI client and a server-side audit consumer
// subscribe simultaneously.
func TestStreamProcessLogs_MultiSubscriberFanout(t *testing.T) {
	f := newSSEFixture(t)

	type clientResult struct {
		frames []sse.Frame
		err    error
	}
	resCh := make(chan clientResult, 2)
	startReq := func() {
		req, _ := http.NewRequest("GET", f.streamURL(), nil)
		resp, err := f.live.Client.Do(req)
		if err != nil {
			resCh <- clientResult{err: err}
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			resCh <- clientResult{err: err}
			return
		}
		resCh <- clientResult{frames: sse.ParseStream(strings.NewReader(string(body)))}
	}

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); startReq() }()
	}

	// Give both subscribers time to attach.
	time.Sleep(80 * time.Millisecond)

	// Write a couple of chunks before exit.
	for _, msg := range []string{"shared-1\n", "shared-2\n"} {
		if _, err := f.pending.Stdout.Write([]byte(msg)); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	time.Sleep(40 * time.Millisecond)

	f.markExited(t, process.ExitInfo{ExitCode: 0, StoppedAt: time.Now().UTC()})

	wg.Wait()
	close(resCh)

	got := 0
	for r := range resCh {
		got++
		if r.err != nil {
			t.Errorf("subscriber %d: client error: %v", got, r.err)
			continue
		}
		// Each subscriber must see exit + end in that order.
		exitIdx := indexOf(r.frames, "exit")
		endIdx := indexOf(r.frames, "end")
		if exitIdx < 0 || endIdx < 0 || exitIdx >= endIdx {
			t.Errorf("subscriber %d: bad frame order (exit=%d, end=%d); frames=%+v",
				got, exitIdx, endIdx, r.frames)
		}
		// Each subscriber must see both streamed chunks. The Phase-5
		// SSE writer means multi-line content round-trips through the
		// parser cleanly, so we can assert on the joined default-event
		// frame data without raw-body substring fallbacks.
		joined := joinFrameData(framesByEvent(r.frames, ""))
		if !strings.Contains(joined, "shared-1") || !strings.Contains(joined, "shared-2") {
			t.Errorf("subscriber %d: missing streamed chunks; data=%q", got, joined)
		}
	}
	if got != 2 {
		t.Errorf("expected 2 subscriber results, got %d", got)
	}
}

// TestStreamProcessLogs_ClientDisconnectMidStream cancels the request
// context after a brief streaming window; the server must clean up the
// subscriber without panicking and the rest of the test (a subsequent
// fresh subscriber) must still complete cleanly.
func TestStreamProcessLogs_ClientDisconnectMidStream(t *testing.T) {
	f := newSSEFixture(t)

	ctx, cancel := context.WithCancel(context.Background())
	type clientResult struct {
		err error
	}
	resCh := make(chan clientResult, 1)
	go func() {
		req, err := http.NewRequestWithContext(ctx, "GET", f.streamURL(), nil)
		if err != nil {
			resCh <- clientResult{err: err}
			return
		}
		resp, err := f.live.Client.Do(req)
		if err != nil {
			// Cancellation surfaces as an error; that's expected.
			resCh <- clientResult{err: err}
			return
		}
		// Read a tiny prefix then drain via cancel.
		buf := make([]byte, 64)
		_, _ = resp.Body.Read(buf)
		_ = resp.Body.Close()
		resCh <- clientResult{}
	}()

	time.Sleep(50 * time.Millisecond)
	if _, err := f.pending.Stdout.Write([]byte("partial\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	cancel()

	// Wait for the disconnected client goroutine to settle.
	select {
	case <-resCh:
	case <-time.After(2 * time.Second):
		t.Fatal("disconnected client goroutine did not return within 2s — server likely leaked")
	}

	// Now mark exit and prove the server is still healthy: a second
	// fresh subscriber must complete successfully.
	f.markExited(t, process.ExitInfo{ExitCode: 0, StoppedAt: time.Now().UTC()})
	resp, body := f.live.Do(t, "GET", f.streamURL(), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("post-disconnect subscriber status = %d", resp.StatusCode)
	}
	frames := sse.ParseStream(strings.NewReader(string(body)))
	if findFrame(frames, "exit") == nil || findFrame(frames, "end") == nil {
		t.Errorf("post-disconnect subscriber missing exit/end; frames=%+v", frames)
	}
}

// TestStreamProcessLogs_MultiLineDataPreserved is the Phase-5
// regression gate for multi-line data encoding. Before the SSEWriter
// seam, a chunk like "alpha\nbeta\n" was emitted as
// `data: alpha\nbeta\n\n\n` — the parser saw `data: alpha`, an unknown
// field `beta`, and dispatched only `alpha`. Every subsequent log line
// in a multi-line write was silently dropped at the consumer.
//
// With the seam, multi-line data must round-trip: the consumer sees
// the chunk's bytes verbatim in the parsed frame data.
func TestStreamProcessLogs_MultiLineDataPreserved(t *testing.T) {
	f := newSSEFixture(t)

	chunk := []byte("alpha\nbeta\ngamma\n")
	if _, err := f.pending.Stdout.Write(chunk); err != nil {
		t.Fatalf("write chunk: %v", err)
	}
	f.markExited(t, process.ExitInfo{ExitCode: 0, StoppedAt: time.Now().UTC()})

	resp, body := f.live.Do(t, "GET", f.streamURL(), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	frames := sse.ParseStream(strings.NewReader(string(body)))
	dataFrames := framesByEvent(frames, "")
	if len(dataFrames) == 0 {
		t.Fatalf("no default-event data frames; frames=%+v", frames)
	}

	joined := joinFrameData(dataFrames)
	for _, want := range []string{"alpha", "beta", "gamma"} {
		if !strings.Contains(joined, want) {
			t.Errorf("multi-line line %q lost across SSE round-trip; got=%q", want, joined)
		}
	}
}

// framesByEvent returns frames whose Event matches `event` (use ""
// for default-event / `message` frames). Helper for assertions that
// only care about a particular frame class.
func framesByEvent(frames []sse.Frame, event string) []sse.Frame {
	out := make([]sse.Frame, 0, len(frames))
	for _, f := range frames {
		if f.Event == event {
			out = append(out, f)
		}
	}
	return out
}

// joinFrameData concatenates frame data with `\n` separators. Used by
// content-presence assertions so multi-line replays surface as one
// scannable string.
func joinFrameData(frames []sse.Frame) string {
	var b strings.Builder
	for i, f := range frames {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.Write(f.Data)
	}
	return b.String()
}
