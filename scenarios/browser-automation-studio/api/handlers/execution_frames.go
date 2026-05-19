package handlers

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	gorillawebsocket "github.com/gorilla/websocket"

	"github.com/vrooli/browser-automation-studio/websocket"
)

// RESTException: ops_probe — playwright-driver pushes binary frames over WS
// and a thin JSON callback. Not RPC-shaped; stays on chi.

// ReceiveExecutionFrame receives frames from the playwright-driver during
// workflow execution. Frames are broadcast to WebSocket clients subscribed to
// the execution's frame stream.
func (h *Handler) ReceiveExecutionFrame(w http.ResponseWriter, r *http.Request) {
	executionID := chi.URLParam(r, "executionId")
	if executionID == "" {
		http.Error(w, "missing execution_id", http.StatusBadRequest)
		return
	}
	if !h.wsHub.HasExecutionFrameSubscribers(executionID) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var frame struct {
		Data      string `json:"data"`
		MediaType string `json:"media_type"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	}
	if err := json.NewDecoder(r.Body).Decode(&frame); err != nil {
		http.Error(w, "invalid frame data", http.StatusBadRequest)
		return
	}
	h.wsHub.BroadcastExecutionFrame(executionID, &websocket.ExecutionFrame{
		ExecutionID: executionID,
		Data:        frame.Data,
		MediaType:   frame.MediaType,
		Width:       frame.Width,
		Height:      frame.Height,
		CapturedAt:  time.Now().UTC().Format(time.RFC3339Nano),
	})
	w.WriteHeader(http.StatusOK)
}

// HandleDriverExecutionFrameStream handles WebSocket binary frame streaming
// from playwright-driver during workflow execution. GET /ws/execution/{executionId}/frames.
func (h *Handler) HandleDriverExecutionFrameStream(w http.ResponseWriter, r *http.Request) {
	executionID := chi.URLParam(r, "executionId")
	if executionID == "" {
		h.log.Error("Missing executionId in driver execution frame stream request")
		http.Error(w, "Missing executionId", http.StatusBadRequest)
		return
	}
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.WithError(err).Error("Failed to upgrade driver execution frame stream connection")
		return
	}
	defer conn.Close()

	h.log.WithField("execution_id", executionID).Info("Driver execution frame stream connected")
	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if gorillawebsocket.IsCloseError(err, gorillawebsocket.CloseNormalClosure, gorillawebsocket.CloseGoingAway) {
				h.log.WithField("execution_id", executionID).Debug("Driver execution frame stream closed normally")
			} else {
				h.log.WithError(err).WithField("execution_id", executionID).Warn("Driver execution frame stream read error")
			}
			break
		}
		if messageType != gorillawebsocket.BinaryMessage {
			continue
		}
		if !h.wsHub.HasExecutionFrameSubscribers(executionID) {
			continue
		}
		frameData := base64.StdEncoding.EncodeToString(data)
		h.wsHub.BroadcastExecutionFrame(executionID, &websocket.ExecutionFrame{
			ExecutionID: executionID,
			Data:        frameData,
			MediaType:   "image/jpeg",
			Width:       0,
			Height:      0,
			CapturedAt:  time.Now().UTC().Format(time.RFC3339Nano),
		})
	}
	h.log.WithField("execution_id", executionID).Info("Driver execution frame stream disconnected")
}
