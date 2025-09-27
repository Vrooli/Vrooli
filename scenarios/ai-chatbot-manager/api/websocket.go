package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

// ConnectionManager manages WebSocket connections
type ConnectionManager struct {
	connections map[string]*WebSocketConnection
	register    chan *WebSocketConnection
	unregister  chan *WebSocketConnection
	broadcast   chan WebSocketMessage
	mu          sync.RWMutex
	logger      *Logger
}

// WebSocketConnection represents a single WebSocket connection
type WebSocketConnection struct {
	conn       *websocket.Conn
	sessionID  string
	chatbotID  string
	send       chan WebSocketMessage
	manager    *ConnectionManager
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(logger *Logger) *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]*WebSocketConnection),
		register:    make(chan *WebSocketConnection),
		unregister:  make(chan *WebSocketConnection),
		broadcast:   make(chan WebSocketMessage),
		logger:      logger,
	}
}

// Start runs the connection manager
func (cm *ConnectionManager) Start() {
	for {
		select {
		case conn := <-cm.register:
			cm.mu.Lock()
			cm.connections[conn.sessionID] = conn
			cm.mu.Unlock()
			cm.logger.Printf("WebSocket client connected: %s", conn.sessionID)

		case conn := <-cm.unregister:
			cm.mu.Lock()
			if _, ok := cm.connections[conn.sessionID]; ok {
				delete(cm.connections, conn.sessionID)
				close(conn.send)
				cm.mu.Unlock()
				cm.logger.Printf("WebSocket client disconnected: %s", conn.sessionID)
			} else {
				cm.mu.Unlock()
			}

		case message := <-cm.broadcast:
			cm.mu.RLock()
			for _, conn := range cm.connections {
				select {
				case conn.send <- message:
				default:
					// Connection's send channel is full, close it
					close(conn.send)
					delete(cm.connections, conn.sessionID)
				}
			}
			cm.mu.RUnlock()
		}
	}
}

// WebSocketHandler handles WebSocket upgrade and communication
type WebSocketHandler struct {
	upgrader  websocket.Upgrader
	manager   *ConnectionManager
	server    *Server
	logger    *Logger
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(server *Server, manager *ConnectionManager, logger *Logger) *WebSocketHandler {
	return &WebSocketHandler{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// In production, implement proper origin checking
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		manager: manager,
		server:  server,
		logger:  logger,
	}
}

// HandleWebSocket handles WebSocket connections
func (wsh *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatbotID := vars["id"]

	// Verify chatbot exists
	chatbot, err := wsh.server.db.GetChatbot(chatbotID)
	if err != nil {
		wsh.logger.Printf("Chatbot not found: %v", err)
		http.Error(w, "Chatbot not found", http.StatusNotFound)
		return
	}

	// Upgrade to WebSocket
	conn, err := wsh.upgrader.Upgrade(w, r, nil)
	if err != nil {
		wsh.logger.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Create WebSocket connection
	sessionID := fmt.Sprintf("ws-%s-%d", uuid.New().String()[:8], time.Now().Unix())
	wsConn := &WebSocketConnection{
		conn:      conn,
		sessionID: sessionID,
		chatbotID: chatbotID,
		send:      make(chan WebSocketMessage, 256),
		manager:   wsh.manager,
	}

	// Register connection
	wsh.manager.register <- wsConn

	// Start goroutines for reading and writing
	go wsConn.writePump()
	go wsConn.readPump(wsh.server, chatbot)
}

// readPump handles incoming messages from WebSocket
func (wsc *WebSocketConnection) readPump(server *Server, chatbot *Chatbot) {
	conversationStartTime := time.Now()
	var conversationID string
	messageCount := 0
	
	defer func() {
		// Publish conversation ended event if we had an active conversation
		if conversationID != "" {
			duration := int(time.Since(conversationStartTime).Seconds())
			server.eventPublisher.ConversationEndedEvent(wsc.chatbotID, conversationID, duration, messageCount)
		}
		wsc.manager.unregister <- wsc
		wsc.conn.Close()
	}()

	wsc.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	wsc.conn.SetPongHandler(func(string) error {
		wsc.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var req ChatRequest
		err := wsc.conn.ReadJSON(&req)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				server.logger.Printf("WebSocket read error: %v", err)
			}
			break
		}

		// Set session ID if not provided
		if req.SessionID == "" {
			req.SessionID = wsc.sessionID
		}

		// Process chat message
		response, err := server.ProcessChatMessage(wsc.chatbotID, req)
		if err != nil {
			// Send error message
			errorMsg := WebSocketMessage{
				Type: "error",
				Payload: map[string]interface{}{
					"message": "Failed to process message",
					"error":   err.Error(),
				},
			}
			select {
			case wsc.send <- errorMsg:
			default:
				// Channel full, close connection
				return
			}
			continue
		}

		// Track conversation ID and message count
		if conversationID == "" {
			conversationID = response.ConversationID
		}
		messageCount++

		// Send response
		responseMsg := WebSocketMessage{
			Type: "message",
			Payload: map[string]interface{}{
				"response":           response.Response,
				"confidence":         response.Confidence,
				"should_escalate":    response.ShouldEscalate,
				"lead_qualification": response.LeadQualification,
				"conversation_id":    response.ConversationID,
				"timestamp":          time.Now().Unix(),
			},
		}

		select {
		case wsc.send <- responseMsg:
		default:
			// Channel full, close connection
			return
		}
	}
}

// writePump handles sending messages to WebSocket
func (wsc *WebSocketConnection) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		wsc.conn.Close()
	}()

	for {
		select {
		case message, ok := <-wsc.send:
			wsc.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Channel closed
				wsc.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := wsc.conn.WriteJSON(message); err != nil {
				return
			}

		case <-ticker.C:
			wsc.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := wsc.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// BroadcastMessage sends a message to all connected clients
func (cm *ConnectionManager) BroadcastMessage(message WebSocketMessage) {
	cm.broadcast <- message
}

// SendToSession sends a message to a specific session
func (cm *ConnectionManager) SendToSession(sessionID string, message WebSocketMessage) error {
	cm.mu.RLock()
	conn, exists := cm.connections[sessionID]
	cm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	select {
	case conn.send <- message:
		return nil
	default:
		return fmt.Errorf("connection buffer full for session: %s", sessionID)
	}
}