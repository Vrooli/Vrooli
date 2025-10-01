package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

func handleWorkspaceWebSocket(w http.ResponseWriter, r *http.Request, ws *workspace, metrics *metricsRegistry) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		metrics.httpUpgradesFail.Add(1)
		log.Printf("workspace websocket upgrade failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ch := ws.subscribe()
	defer ws.unsubscribe(ch)

	// Send initial workspace state
	state, err := ws.getState()
	if err != nil {
		_ = conn.Close()
		return
	}
	if err := conn.WriteMessage(websocket.TextMessage, state); err != nil {
		_ = conn.Close()
		return
	}

	// Forward events from workspace to client
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
			// Ignore incoming messages (workspace is read-only over websocket)
		}
	}()

	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return
			}
			if err := conn.WriteJSON(event); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}
