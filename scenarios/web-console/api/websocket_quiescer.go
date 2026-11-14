package main

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const websocketRestartCloseCode = 1012

// webSocketQuiescer tracks upgraded HTTP handlers so lifecycle drain can close
// long-lived connections before waiting for the API's admitted request count.
type webSocketQuiescer struct {
	mu          sync.Mutex
	connections map[*websocket.Conn][]func()
	quiescing   bool
}

func newWebSocketQuiescer() *webSocketQuiescer {
	return &webSocketQuiescer{connections: make(map[*websocket.Conn][]func())}
}

// Register adds a connection unless lifecycle quiescence already started.
// closeAdditional is used by proxy handlers to also close their upstream peer.
func (q *webSocketQuiescer) Register(conn *websocket.Conn, closeAdditional ...func()) (func(), bool) {
	if q == nil || conn == nil {
		return func() {}, true
	}
	q.mu.Lock()
	if q.quiescing {
		q.mu.Unlock()
		closeWebSocketForRestart(conn)
		for _, closeFn := range closeAdditional {
			if closeFn != nil {
				closeFn()
			}
		}
		_ = conn.Close()
		return func() {}, false
	}
	q.connections[conn] = append([]func(){}, closeAdditional...)
	q.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			q.mu.Lock()
			delete(q.connections, conn)
			q.mu.Unlock()
		})
	}, true
}

// CloseAll blocks new registrations and asks every active peer to reconnect
// after replacement. Closing the socket unblocks handlers waiting in Read.
func (q *webSocketQuiescer) CloseAll() {
	if q == nil {
		return
	}
	q.mu.Lock()
	q.quiescing = true
	connections := make(map[*websocket.Conn][]func(), len(q.connections))
	for conn, closeAdditional := range q.connections {
		connections[conn] = closeAdditional
	}
	q.mu.Unlock()

	for conn, closeAdditional := range connections {
		closeWebSocketForRestart(conn)
		for _, closeFn := range closeAdditional {
			if closeFn != nil {
				closeFn()
			}
		}
		_ = conn.Close()
	}
}

// Open permits new upgraded handlers after a failed replacement resumes this
// API process.
func (q *webSocketQuiescer) Open() {
	if q == nil {
		return
	}
	q.mu.Lock()
	q.quiescing = false
	q.mu.Unlock()
}

func closeWebSocketForRestart(conn *websocket.Conn) {
	if conn == nil {
		return
	}
	_ = conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocketRestartCloseCode, "service restarting"),
		time.Now().Add(time.Second),
	)
}
