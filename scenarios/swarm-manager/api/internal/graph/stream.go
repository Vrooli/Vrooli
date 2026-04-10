package graph

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// StreamHandler serves the WebSocket graph stream endpoint.
type StreamHandler struct {
	broker   *Broker
	upgrader websocket.Upgrader
}

// NewStreamHandler creates a WebSocket stream handler.
func NewStreamHandler(broker *Broker) *StreamHandler {
	return &StreamHandler{
		broker: broker,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
	}
}

// RegisterRoutes registers the WebSocket endpoint.
func (h *StreamHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/ws/graph", h.HandleWebSocket)
}

// ServeHTTP implements http.Handler for use with httptest.NewServer.
func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.HandleWebSocket(w, r)
}

// HandleWebSocket upgrades an HTTP connection to WebSocket and sends graph events.
func (h *StreamHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade error", "error", err)
		return
	}

	// Register with broker.
	h.broker.AddClient(conn)
	defer func() {
		h.broker.RemoveClient(conn)
		conn.Close()
	}()

	// Read loop — keeps connection alive and detects client disconnect.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
