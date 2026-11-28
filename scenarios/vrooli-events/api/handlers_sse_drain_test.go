package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	apilifecycle "github.com/vrooli/api-core/lifecycle"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/broker"
)

// An open event stream must end when a restart closes admission; otherwise the
// provider fence waits on it forever and every publish gets 503 (2026-11-20).
func TestAnOpenStreamEndsWhenARestartDrains(t *testing.T) {
	p, err := apilifecycle.NewProvider("vrooli-events", "instance", apilifecycle.Hooks{})
	if err != nil {
		t.Fatal(err)
	}
	ended := make(chan struct{})
	entered := make(chan struct{})
	s := &Server{}
	handler := p.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		s.streamLiveEvents(w, w.(http.Flusher), make(chan broker.SSEMessage), context.Background(), r.Context())
		close(ended)
	}))
	go handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/events/stream", nil))
	<-entered
	if _, err := p.Prepare(context.Background(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "op", Reason: "restart"})); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ended:
	case <-time.After(2 * time.Second):
		t.Fatal("the stream kept running after admission closed; the drain would never finish")
	}
}
