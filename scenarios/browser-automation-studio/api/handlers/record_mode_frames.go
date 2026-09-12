package handlers

import (
	"encoding/binary"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vrooli/browser-automation-studio/performance"
	"github.com/vrooli/browser-automation-studio/websocket"
)

// decodeDriverFrame separates the optional driver performance header from the
// JPEG payload. A malformed optional header remains a frame payload rather
// than making the live stream unavailable.
func decodeDriverFrame(data []byte) ([]byte, *performance.FrameHeader, error) {
	if len(data) <= 4 || (data[0] == 0xFF && data[1] == 0xD8) {
		return data, nil, nil
	}
	headerLen := binary.BigEndian.Uint32(data[:4])
	if int(headerLen) > len(data)-4 {
		return data, nil, nil
	}
	var header performance.FrameHeader
	if err := json.Unmarshal(data[4:4+headerLen], &header); err != nil {
		return data, nil, err
	}
	return data[4+headerLen:], &header, nil
}

// ReceiveRecordingFrame accepts the low-latency HTTP frame fallback. Binary
// driver streaming remains in the same recording-frame boundary below.
func (h *Handler) ReceiveRecordingFrame(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "sessionId"}))
		return
	}
	var frame websocket.RecordingFrame
	if err := json.NewDecoder(r.Body).Decode(&frame); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "Invalid JSON body: " + err.Error()}))
		return
	}
	if h.wsHub.HasRecordingSubscribers(sessionID) {
		h.wsHub.BroadcastRecordingFrame(sessionID, &frame)
	}
	w.WriteHeader(http.StatusOK)
}
