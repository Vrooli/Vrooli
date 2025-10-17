package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func TestNewWebSocketHandler(t *testing.T) {
	t.Run("WithUpgrader", func(t *testing.T) {
		upgrader := &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		handler := NewWebSocketHandler(upgrader, nil)

		if handler == nil {
			t.Fatal("Expected non-nil WebSocket handler")
		}

		if handler.upgrader != upgrader {
			t.Error("Expected upgrader to be set")
		}
	})

	t.Run("WithRedis", func(t *testing.T) {
		upgrader := &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		// Note: We don't create a real Redis client here as it would require a running Redis instance
		handler := NewWebSocketHandler(upgrader, nil)

		if handler == nil {
			t.Fatal("Expected non-nil WebSocket handler")
		}

		if handler.redis != nil {
			t.Error("Expected redis to be nil")
		}
	})
}

func TestWebSocketMessage(t *testing.T) {
	t.Run("ValidateStructure", func(t *testing.T) {
		msg := WebSocketMessage{
			Type:    "test",
			Payload: map[string]string{"key": "value"},
		}

		if msg.Type != "test" {
			t.Errorf("Expected type 'test', got %s", msg.Type)
		}

		payload, ok := msg.Payload.(map[string]string)
		if !ok {
			t.Error("Expected payload to be map[string]string")
		}

		if payload["key"] != "value" {
			t.Errorf("Expected payload key to be 'value', got %s", payload["key"])
		}
	})
}

func TestHandleWebSocket(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("UpgradeSuccess", func(t *testing.T) {
		upgrader := &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		handler := NewWebSocketHandler(upgrader, nil)

		router := gin.New()
		router.GET("/ws", handler.HandleWebSocket)

		server := httptest.NewServer(router)
		defer server.Close()

		// Convert http:// to ws://
		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

		// Connect to WebSocket
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to connect to WebSocket: %v", err)
		}
		defer conn.Close()

		// Read initial connection message
		var msg WebSocketMessage
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("Failed to read connection message: %v", err)
		}

		if msg.Type != "connection" {
			t.Errorf("Expected type 'connection', got %s", msg.Type)
		}
	})

	t.Run("PingPong", func(t *testing.T) {
		upgrader := &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		handler := NewWebSocketHandler(upgrader, nil)

		router := gin.New()
		router.GET("/ws", handler.HandleWebSocket)

		server := httptest.NewServer(router)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to connect to WebSocket: %v", err)
		}
		defer conn.Close()

		// Read connection message
		var connMsg WebSocketMessage
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		conn.ReadJSON(&connMsg)

		// Send ping message
		pingMsg := WebSocketMessage{
			Type: "ping",
		}
		if err := conn.WriteJSON(pingMsg); err != nil {
			t.Fatalf("Failed to send ping: %v", err)
		}

		// Read pong response
		var pongMsg WebSocketMessage
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if err := conn.ReadJSON(&pongMsg); err != nil {
			t.Fatalf("Failed to read pong: %v", err)
		}

		if pongMsg.Type != "pong" {
			t.Errorf("Expected type 'pong', got %s", pongMsg.Type)
		}
	})

	t.Run("SubscribeMessage", func(t *testing.T) {
		upgrader := &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		handler := NewWebSocketHandler(upgrader, nil)

		router := gin.New()
		router.GET("/ws", handler.HandleWebSocket)

		server := httptest.NewServer(router)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to connect to WebSocket: %v", err)
		}
		defer conn.Close()

		// Read connection message
		var connMsg WebSocketMessage
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		conn.ReadJSON(&connMsg)

		// Send subscribe message
		subscribeMsg := WebSocketMessage{
			Type: "subscribe",
			Payload: map[string]interface{}{
				"appId": "test-app",
			},
		}
		if err := conn.WriteJSON(subscribeMsg); err != nil {
			t.Fatalf("Failed to send subscribe: %v", err)
		}

		// Just verify it doesn't error
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("UnsubscribeMessage", func(t *testing.T) {
		upgrader := &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		handler := NewWebSocketHandler(upgrader, nil)

		router := gin.New()
		router.GET("/ws", handler.HandleWebSocket)

		server := httptest.NewServer(router)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to connect to WebSocket: %v", err)
		}
		defer conn.Close()

		// Read connection message
		var connMsg WebSocketMessage
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		conn.ReadJSON(&connMsg)

		// Send unsubscribe message
		unsubscribeMsg := WebSocketMessage{
			Type: "unsubscribe",
			Payload: map[string]interface{}{
				"appId": "test-app",
			},
		}
		if err := conn.WriteJSON(unsubscribeMsg); err != nil {
			t.Fatalf("Failed to send unsubscribe: %v", err)
		}

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("CommandMessage", func(t *testing.T) {
		upgrader := &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		handler := NewWebSocketHandler(upgrader, nil)

		router := gin.New()
		router.GET("/ws", handler.HandleWebSocket)

		server := httptest.NewServer(router)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to connect to WebSocket: %v", err)
		}
		defer conn.Close()

		// Read connection message
		var connMsg WebSocketMessage
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		conn.ReadJSON(&connMsg)

		// Send command message
		commandMsg := WebSocketMessage{
			Type: "command",
			Payload: map[string]interface{}{
				"command": "test-command",
			},
		}
		if err := conn.WriteJSON(commandMsg); err != nil {
			t.Fatalf("Failed to send command: %v", err)
		}

		// Read command response
		var responseMsg WebSocketMessage
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if err := conn.ReadJSON(&responseMsg); err != nil {
			t.Fatalf("Failed to read command response: %v", err)
		}

		if responseMsg.Type != "command_response" {
			t.Errorf("Expected type 'command_response', got %s", responseMsg.Type)
		}
	})

	t.Run("UnknownMessageType", func(t *testing.T) {
		upgrader := &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		handler := NewWebSocketHandler(upgrader, nil)

		router := gin.New()
		router.GET("/ws", handler.HandleWebSocket)

		server := httptest.NewServer(router)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to connect to WebSocket: %v", err)
		}
		defer conn.Close()

		// Read connection message
		var connMsg WebSocketMessage
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		conn.ReadJSON(&connMsg)

		// Send unknown message type
		unknownMsg := WebSocketMessage{
			Type:    "unknown",
			Payload: map[string]string{"test": "data"},
		}
		if err := conn.WriteJSON(unknownMsg); err != nil {
			t.Fatalf("Failed to send unknown message: %v", err)
		}

		// Just verify it doesn't crash
		time.Sleep(100 * time.Millisecond)
	})
}

func TestWebSocketEdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("InvalidPayload", func(t *testing.T) {
		upgrader := &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		handler := NewWebSocketHandler(upgrader, nil)

		router := gin.New()
		router.GET("/ws", handler.HandleWebSocket)

		server := httptest.NewServer(router)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to connect to WebSocket: %v", err)
		}
		defer conn.Close()

		// Read connection message
		var connMsg WebSocketMessage
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		conn.ReadJSON(&connMsg)

		// Send subscribe with invalid payload
		invalidMsg := WebSocketMessage{
			Type:    "subscribe",
			Payload: "invalid",
		}
		if err := conn.WriteJSON(invalidMsg); err != nil {
			t.Fatalf("Failed to send invalid message: %v", err)
		}

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("ClientDisconnect", func(t *testing.T) {
		upgrader := &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		handler := NewWebSocketHandler(upgrader, nil)

		router := gin.New()
		router.GET("/ws", handler.HandleWebSocket)

		server := httptest.NewServer(router)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to connect to WebSocket: %v", err)
		}

		// Read connection message
		var connMsg WebSocketMessage
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		conn.ReadJSON(&connMsg)

		// Close connection immediately
		conn.Close()

		// Give server time to detect disconnect
		time.Sleep(100 * time.Millisecond)
	})
}

func TestWebSocketConcurrency(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("MultipleConnections", func(t *testing.T) {
		upgrader := &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		handler := NewWebSocketHandler(upgrader, nil)

		router := gin.New()
		router.GET("/ws", handler.HandleWebSocket)

		server := httptest.NewServer(router)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

		// Create multiple concurrent connections
		connections := make([]*websocket.Conn, 5)
		for i := 0; i < 5; i++ {
			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				t.Fatalf("Failed to connect WebSocket %d: %v", i, err)
			}
			connections[i] = conn
			defer conn.Close()

			// Read connection message
			var connMsg WebSocketMessage
			conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			conn.ReadJSON(&connMsg)
		}

		// Send messages from all connections
		for i, conn := range connections {
			msg := WebSocketMessage{
				Type:    "ping",
				Payload: map[string]int{"connection": i},
			}
			if err := conn.WriteJSON(msg); err != nil {
				t.Errorf("Failed to send from connection %d: %v", i, err)
			}
		}

		// Read responses
		for i, conn := range connections {
			var pongMsg WebSocketMessage
			conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			if err := conn.ReadJSON(&pongMsg); err != nil {
				t.Errorf("Failed to read pong from connection %d: %v", i, err)
			}
		}
	})
}
