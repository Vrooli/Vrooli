package server

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"app-issue-tracker-api/internal/logging"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

type originAllowlist struct {
	allowAll bool
	values   map[string]struct{}
}

func newOriginAllowlist(origins []string) originAllowlist {
	allowlist := originAllowlist{values: make(map[string]struct{})}
	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		if trimmed == "*" {
			allowlist.allowAll = true
			return allowlist
		}
		lower := strings.ToLower(trimmed)
		allowlist.values[lower] = struct{}{}
		if parsed, err := url.Parse(trimmed); err == nil && parsed.Host != "" {
			allowlist.values[strings.ToLower(parsed.Host)] = struct{}{}
		}
	}
	return allowlist
}

func (o originAllowlist) contains(value string) bool {
	if o.allowAll {
		return true
	}
	if len(o.values) == 0 {
		return false
	}
	if _, ok := o.values[strings.ToLower(strings.TrimSpace(value))]; ok {
		return true
	}
	return false
}

func newWebSocketUpgrader(cfg *Config) websocket.Upgrader {
	allowlist := newOriginAllowlist(cfg.WebsocketAllowedOrigins)
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin == "" {
				return true
			}
			if allowlist.allowAll {
				return true
			}
			parsed, err := url.Parse(origin)
			if err != nil {
				logging.LogWarn("Rejected websocket connection due to invalid origin", "origin", origin, "error", err)
				return false
			}
			host := strings.ToLower(parsed.Host)
			requestHost := strings.ToLower(r.Host)
			if host == requestHost {
				return true
			}
			if allowlist.contains(origin) || allowlist.contains(host) {
				return true
			}
			logging.LogWarn("Rejected websocket connection due to disallowed origin", "origin", origin, "request_host", r.Host)
			return false
		},
	}
}

// Hub maintains the set of active clients and broadcasts events to them
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound events from the application
	broadcast chan Event

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread-safe operations
	mu sync.RWMutex

	stop     chan struct{}
	done     chan struct{}
	stopOnce sync.Once
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Event, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	defer close(h.done)
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			current := len(h.clients)
			h.mu.Unlock()
			logging.LogInfo("WebSocket client registered", "clients", current)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			current := len(h.clients)
			h.mu.Unlock()
			logging.LogInfo("WebSocket client unregistered", "clients", current)

		case event := <-h.broadcast:
			h.mu.RLock()
			clientCount := len(h.clients)
			h.mu.RUnlock()

			if clientCount == 0 {
				continue
			}

			// Serialize event once
			message, err := event.ToJSON()
			if err != nil {
				logging.LogErrorErr("Failed to serialize websocket event", err, "event_type", event.Type)
				continue
			}

			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					h.mu.RUnlock()
					h.mu.Lock()
					close(client.send)
					delete(h.clients, client)
					h.mu.Unlock()
					h.mu.RLock()
					logging.LogWarn("WebSocket client send buffer full - disconnecting")
				}
			}
			h.mu.RUnlock()

		case <-h.stop:
			h.disconnectAll()
			return
		}
	}
}

func (h *Hub) disconnectAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		close(client.send)
		delete(h.clients, client)
	}
}

// Publish sends an event to all connected clients
func (h *Hub) Publish(event Event) {
	select {
	case <-h.stop:
		logging.LogWarn("WebSocket hub shutting down; dropping event", "event_type", event.Type)
		return
	case h.broadcast <- event:
		logging.LogDebug("Queued websocket event", "event_type", event.Type)
	default:
		logging.LogWarn("Websocket broadcast channel full", "event_type", event.Type)
	}
}

// Shutdown signals the hub to stop and waits for cleanup to finish.
func (h *Hub) Shutdown() {
	h.stopOnce.Do(func() {
		close(h.stop)
	})
	<-h.done
}

// Client represents a WebSocket client connection
type Client struct {
	hub *Hub

	// The websocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan []byte
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// We don't expect clients to send messages, only receive
	// But we need to read to detect disconnections
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logging.LogWarn("Unexpected websocket close", "error", err)
			}
			break
		}
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleWebSocket handles WebSocket upgrade requests
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		logging.LogErrorErr("Websocket upgrade failed", err)
		return
	}

	client := &Client{
		hub:  s.hub,
		conn: conn,
		send: make(chan []byte, 256),
	}

	client.hub.register <- client

	// Start client goroutines
	go client.writePump()
	go client.readPump()
}
