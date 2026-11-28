package handlers

// Long-lived WebSocket handlers are admitted HTTP requests for the whole life
// of the connection. A driver frame socket can stay open for days, so leaving
// it open while the lifecycle provider closes admission makes drain impossible
// and strands the scenario behind a fence. These helpers give a streaming
// handler a way to observe the provider's closing signal and release the
// connection before the control plane waits on it.

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vrooli/api-core/lifecycle"
)

// closeOnMaintenance closes conn when the lifecycle provider for ctx closes
// admission. The returned stop function must run when the handler exits so the
// watcher does not outlive its connection. It is a no-op outside provider
// middleware, keeping direct handler tests and non-managed servers unchanged.
func closeOnMaintenance(ctx context.Context, conn *websocket.Conn) func() {
	return closeOnMaintenanceSignal(lifecycle.Closing(ctx), conn)
}

func closeOnMaintenanceSignal(closing <-chan struct{}, conn *websocket.Conn) func() {
	if closing == nil || conn == nil {
		return func() {}
	}
	done := make(chan struct{})
	var once sync.Once
	stop := func() { once.Do(func() { close(done) }) }
	go func() {
		select {
		case <-closing:
			// A close control frame tells a well-behaved peer why the stream
			// ended; closing the socket also unblocks the read loop.
			_ = conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseGoingAway, "lifecycle maintenance"),
				time.Now().Add(time.Second),
			)
			_ = conn.Close()
		case <-done:
		}
	}()
	return stop
}
