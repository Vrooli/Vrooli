package main

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// sseFrame is a parsed Server-Sent Events frame.
type sseFrame struct {
	id      string
	event   string
	data    string
	comment string
}

// readSSEFrames consumes frames from r until n frames are read or ctx is done.
// It tolerates heartbeat comment lines (":hb").
func readSSEFrames(t *testing.T, r *bufio.Reader, n int) []sseFrame {
	t.Helper()
	var frames []sseFrame
	var cur sseFrame
	have := false
	for len(frames) < n {
		line, err := r.ReadString('\n')
		if err != nil {
			return frames
		}
		line = strings.TrimRight(line, "\n")
		switch {
		case line == "":
			if have {
				frames = append(frames, cur)
				cur = sseFrame{}
				have = false
			}
		case strings.HasPrefix(line, ":"):
			cur.comment = strings.TrimPrefix(line, ":")
			have = true
		case strings.HasPrefix(line, "id:"):
			cur.id = strings.TrimPrefix(line, "id:")
			have = true
		case strings.HasPrefix(line, "event:"):
			cur.event = strings.TrimPrefix(line, "event:")
			have = true
		case strings.HasPrefix(line, "data:"):
			cur.data = strings.TrimPrefix(line, "data:")
			have = true
		}
	}
	return frames
}

// startStream runs handleEventStream against a pipe-backed recorder so the test
// can read frames as they are flushed while the handler keeps streaming. It
// returns a reader for the body, the cancel func to end the stream, and a
// done channel closed when the handler returns.
func startStream(t *testing.T, srv *Server, req *http.Request) (*bufio.Reader, context.CancelFunc, <-chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)

	pr, pw := newSyncPipe()
	rec := &flushPipeRecorder{w: pw, header: make(http.Header)}

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer pw.Close()
		srv.handleEventStream(rec, req)
	}()

	return bufio.NewReader(pr), cancel, done
}

func newEventStreamServer() *Server {
	return &Server{hub: NewConversationHub()}
}

func TestEventStream_SetsHeaders(t *testing.T) {
	srv := newEventStreamServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/stream", nil)
	_, cancel, done := startStream(t, srv, req)
	defer func() { cancel(); <-done }()

	// Headers are written before the first flush; give the goroutine a moment.
	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done
}

func TestEventStream_RejectsNonFlusher(t *testing.T) {
	srv := newEventStreamServer()
	rec := &nonFlusherRecorder{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/stream", nil)
	srv.handleEventStream(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when ResponseWriter is not a Flusher, got %d", rec.Code)
	}
}

func TestEventStream_EmitsWellFormedFrames(t *testing.T) {
	srv := newEventStreamServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/stream", nil)
	r, cancel, done := startStream(t, srv, req)
	defer func() { cancel(); <-done }()

	// Let the handler subscribe before publishing.
	time.Sleep(20 * time.Millisecond)
	id := srv.hub.Publish(HubEnvelope{
		SessionID: "s1",
		Kind:      HubKindConversationEvent,
		Sequence:  5,
		Payload:   conversationEventPayload{ID: "evt-1", Text: "hello", Role: "assistant"},
	})

	frames := readSSEFrames(t, r, 1)
	if len(frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(frames))
	}
	f := frames[0]
	if f.event != HubKindConversationEvent {
		t.Fatalf("expected event %q, got %q", HubKindConversationEvent, f.event)
	}
	if f.id == "" {
		t.Fatal("expected non-empty id line")
	}
	var env HubEnvelope
	if err := json.Unmarshal([]byte(f.data), &env); err != nil {
		t.Fatalf("data is not valid JSON: %v (%q)", err, f.data)
	}
	if env.ID != id {
		t.Fatalf("expected envelope id %d, got %d", id, env.ID)
	}
	if env.SessionID != "s1" || env.Sequence != 5 {
		t.Fatalf("unexpected envelope: %+v", env)
	}
}

func TestEventStream_Heartbeat(t *testing.T) {
	old := hubHeartbeatInterval
	hubHeartbeatInterval = 10 * time.Millisecond
	defer func() { hubHeartbeatInterval = old }()

	srv := newEventStreamServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/stream", nil)
	r, cancel, done := startStream(t, srv, req)
	defer func() { cancel(); <-done }()

	// First non-empty line should be a heartbeat comment within a few intervals.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("read error before heartbeat: %v", err)
		}
		if strings.HasPrefix(line, ":hb") {
			return
		}
	}
	t.Fatal("did not observe a heartbeat comment line")
}

func TestEventStream_ReplaysFromLastEventIDHeader(t *testing.T) {
	srv := newEventStreamServer()
	for i := 0; i < 5; i++ {
		srv.hub.Publish(HubEnvelope{SessionID: "s1", Kind: HubKindConversationEvent, Payload: conversationEventPayload{}})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/stream", nil)
	req.Header.Set("Last-Event-ID", "3")
	r, cancel, done := startStream(t, srv, req)
	defer func() { cancel(); <-done }()

	frames := readSSEFrames(t, r, 2)
	if len(frames) != 2 {
		t.Fatalf("expected 2 replayed frames (4,5), got %d", len(frames))
	}
	if frames[0].id != "4" || frames[1].id != "5" {
		t.Fatalf("expected replay ids 4,5, got %s,%s", frames[0].id, frames[1].id)
	}
}

func TestEventStream_ReplaysFromQueryParam(t *testing.T) {
	srv := newEventStreamServer()
	for i := 0; i < 4; i++ {
		srv.hub.Publish(HubEnvelope{SessionID: "s1", Kind: HubKindConversationEvent, Payload: conversationEventPayload{}})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/stream?last_event_id=2", nil)
	r, cancel, done := startStream(t, srv, req)
	defer func() { cancel(); <-done }()

	frames := readSSEFrames(t, r, 2)
	if len(frames) != 2 {
		t.Fatalf("expected 2 replayed frames (3,4), got %d", len(frames))
	}
	if frames[0].id != "3" || frames[1].id != "4" {
		t.Fatalf("expected replay ids 3,4, got %s,%s", frames[0].id, frames[1].id)
	}
}

func TestEventStream_HeaderWinsOverQuery(t *testing.T) {
	srv := newEventStreamServer()
	for i := 0; i < 5; i++ {
		srv.hub.Publish(HubEnvelope{SessionID: "s1", Kind: HubKindConversationEvent, Payload: conversationEventPayload{}})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/stream?last_event_id=1", nil)
	req.Header.Set("Last-Event-ID", "4")
	r, cancel, done := startStream(t, srv, req)
	defer func() { cancel(); <-done }()

	frames := readSSEFrames(t, r, 1)
	if len(frames) != 1 {
		t.Fatalf("expected 1 replayed frame (5), got %d", len(frames))
	}
	if frames[0].id != "5" {
		t.Fatalf("header should win → expected replay id 5, got %s", frames[0].id)
	}
}

func TestEventStream_GapYieldsOutOfSync(t *testing.T) {
	srv := newEventStreamServer()
	total := hubRingSize + 50
	for i := 0; i < total; i++ {
		srv.hub.Publish(HubEnvelope{SessionID: "s1", Kind: HubKindConversationEvent, Payload: conversationEventPayload{}})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/stream", nil)
	req.Header.Set("Last-Event-ID", "1") // far older than retained window
	r, cancel, done := startStream(t, srv, req)
	defer func() { cancel(); <-done }()

	frames := readSSEFrames(t, r, 1)
	if len(frames) != 1 {
		t.Fatalf("expected at least 1 frame, got %d", len(frames))
	}
	if frames[0].event != HubKindConversationOutOfSync {
		t.Fatalf("expected first frame to be conversation_out_of_sync, got %q", frames[0].event)
	}
}

// --- test plumbing ---------------------------------------------------------

// flushPipeRecorder is an http.ResponseWriter + http.Flusher that writes to a
// pipe so a test reader sees flushed bytes immediately.
type flushPipeRecorder struct {
	w      *syncPipe
	header http.Header
	code   int
}

func (f *flushPipeRecorder) Header() http.Header { return f.header }
func (f *flushPipeRecorder) WriteHeader(c int)   { f.code = c }
func (f *flushPipeRecorder) Write(b []byte) (int, error) {
	return f.w.Write(b)
}
func (f *flushPipeRecorder) Flush() {}

// nonFlusherRecorder is a bare ResponseWriter with NO Flush method so the
// handler's http.Flusher type-assertion fails and it 500s.
type nonFlusherRecorder struct {
	Code   int
	header http.Header
	body   strings.Builder
}

func (n *nonFlusherRecorder) Header() http.Header {
	if n.header == nil {
		n.header = make(http.Header)
	}
	return n.header
}
func (n *nonFlusherRecorder) Write(b []byte) (int, error) { return n.body.Write(b) }
func (n *nonFlusherRecorder) WriteHeader(c int)           { n.Code = c }

// syncPipe is a minimal in-memory byte pipe with blocking reads, usable across
// goroutines without the half-close quirks of io.Pipe for this test's needs.
type syncPipe struct {
	mu     sync.Mutex
	cond   *sync.Cond
	buf    []byte
	closed bool
}

func newSyncPipe() (*syncPipe, *syncPipe) {
	p := &syncPipe{}
	p.cond = sync.NewCond(&p.mu)
	return p, p
}

func (p *syncPipe) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return 0, http.ErrBodyNotAllowed
	}
	p.buf = append(p.buf, b...)
	p.cond.Broadcast()
	return len(b), nil
}

func (p *syncPipe) Read(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for len(p.buf) == 0 && !p.closed {
		p.cond.Wait()
	}
	if len(p.buf) == 0 && p.closed {
		return 0, errSyncPipeClosed
	}
	n := copy(b, p.buf)
	p.buf = p.buf[n:]
	return n, nil
}

func (p *syncPipe) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	p.cond.Broadcast()
	return nil
}

var errSyncPipeClosed = &pipeClosedError{}

type pipeClosedError struct{}

func (e *pipeClosedError) Error() string { return "sync pipe closed" }
