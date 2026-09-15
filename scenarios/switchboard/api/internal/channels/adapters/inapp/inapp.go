package inapp

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"switchboard/internal/channels"
)

type Adapter struct {
	mu       sync.Mutex
	writeMu  sync.Mutex
	clients  map[string]map[*websocket.Conn]struct{}
	receive  func(channels.Envelope) error
	upgrader websocket.Upgrader
	starter  starter
}

func New() *Adapter {
	return &Adapter{clients: make(map[string]map[*websocket.Conn]struct{}), upgrader: websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}}
}

func (a *Adapter) ID() string            { return "in-app" }
func (a *Adapter) HTTPPath() string      { return "/api/v1/channels/socket" }
func (a *Adapter) Handler() http.Handler { return http.HandlerFunc(a.serveHTTP) }
func (a *Adapter) BindReceive(fn func(channels.Envelope) error) {
	a.mu.Lock()
	a.receive = fn
	a.mu.Unlock()
}

func (a *Adapter) Connect(_ context.Context, fn func(channels.Envelope) error) error {
	a.BindReceive(fn)
	return nil
}

func (a *Adapter) serveHTTP(w http.ResponseWriter, r *http.Request) {
	threadKey := r.URL.Query().Get("thread_key")
	if threadKey == "" {
		http.Error(w, "thread_key is required", http.StatusBadRequest)
		return
	}
	conn, err := a.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	a.mu.Lock()
	if a.clients[threadKey] == nil {
		a.clients[threadKey] = make(map[*websocket.Conn]struct{})
	}
	a.clients[threadKey][conn] = struct{}{}
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		delete(a.clients[threadKey], conn)
		if len(a.clients[threadKey]) == 0 {
			delete(a.clients, threadKey)
		}
		a.mu.Unlock()
		_ = conn.Close()
	}()
	for {
		var envelope channels.Envelope
		if err := conn.ReadJSON(&envelope); err != nil {
			return
		}
		envelope.ChannelID = "in-app"
		if envelope.ThreadKey == "" {
			envelope.ThreadKey = threadKey
		}
		if envelope.ThreadKey != threadKey {
			a.writeMu.Lock()
			_ = conn.WriteJSON(map[string]string{"error": "thread_key does not match connection"})
			a.writeMu.Unlock()
			continue
		}
		a.mu.Lock()
		receive := a.receive
		a.mu.Unlock()
		if receive == nil {
			a.writeMu.Lock()
			_ = conn.WriteJSON(map[string]string{"error": "in-app ingress is unavailable"})
			a.writeMu.Unlock()
			continue
		}
		if err := receive(envelope); err != nil {
			a.writeMu.Lock()
			_ = conn.WriteJSON(map[string]string{"error": err.Error()})
			a.writeMu.Unlock()
		}
	}
}

func (a *Adapter) Send(_ context.Context, out channels.Outbound) error {
	a.mu.Lock()
	clients := make([]*websocket.Conn, 0, len(a.clients[out.ThreadKey]))
	for client := range a.clients[out.ThreadKey] {
		clients = append(clients, client)
	}
	a.mu.Unlock()
	if len(clients) == 0 {
		return fmt.Errorf("no in-app client is connected for thread %q", out.ThreadKey)
	}
	for _, client := range clients {
		a.writeMu.Lock()
		if err := client.WriteJSON(out); err != nil {
			a.writeMu.Unlock()
			return err
		}
		a.writeMu.Unlock()
	}
	return nil
}

func (a *Adapter) Probe(context.Context) channels.ProbeResult {
	return channels.ProbeResult{Available: true}
}

var _ channels.Adapter = (*Adapter)(nil)
