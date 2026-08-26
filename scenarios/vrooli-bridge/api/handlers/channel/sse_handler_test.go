package channel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vrooli-bridge/internal/presence"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"
)

type presenceProbeWriter struct {
	header   http.Header
	onHeader func()
}

func (w *presenceProbeWriter) Header() http.Header { return w.header }

func (w *presenceProbeWriter) WriteHeader(int) {
	if w.onHeader != nil {
		w.onHeader()
	}
}

func (w *presenceProbeWriter) Write(p []byte) (int, error) { return len(p), nil }

func (w *presenceProbeWriter) Flush() {}

// [REQ:BRG-P0-003] Presence is registered before the stream acknowledgement,
// so an operator read concurrent with the SSE handshake cannot see a node as
// offline after the transport has already connected.
func TestSSEHandler_RegistersPresenceBeforeAcknowledgingStream(t *testing.T) {
	hub := presence.NewHub(scheduletest.New(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)))
	h := newSSEHandler(sseDeps{Hub: hub})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/channel/events?node=n1", nil)

	onlineAtHeader := make(chan bool, 1)
	w := &presenceProbeWriter{
		header: make(http.Header),
		onHeader: func() {
			onlineAtHeader <- hub.IsOnline("n1")
			cancel()
		},
	}
	done := make(chan struct{})
	go func() {
		h.handleEvents(w, req)
		close(done)
	}()

	select {
	case online := <-onlineAtHeader:
		require.True(t, online)
	case <-time.After(time.Second):
		t.Fatal("SSE handler did not acknowledge the stream")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SSE handler did not stop after request cancellation")
	}
}
