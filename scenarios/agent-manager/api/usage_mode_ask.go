package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"agent-manager/internal/usagemodes"

	"github.com/vrooli/api-core/operatordecision"
)

// UsageModeAskPath is where a caller asks the operator to start a timed mode.
const UsageModeAskPath = "/api/v1/usage/modes/ask"

// handleUsageModeAsk raises a decision; it never starts a mode. The mode starts
// only when a verified human answers it (usagemodes.Notifier.Apply), so any
// caller may ask and none can start one.
func (s *Server) handleUsageModeAsk(w http.ResponseWriter, r *http.Request) {
	if s.usageModeNotifier == nil {
		http.Error(w, "usage mode decisions are unavailable", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		Runner string `json:"runner"`
		Length string `json:"length"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil {
		http.Error(w, "body must be JSON {runner, length}", http.StatusBadRequest)
		return
	}
	length, err := time.ParseDuration(body.Length)
	if err != nil {
		http.Error(w, "length must be a duration such as 30m", http.StatusBadRequest)
		return
	}
	id, err := s.usageModeNotifier.AskToStart(r.Context(), body.Runner, length)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case isInvalidModeAsk(err):
			status = http.StatusBadRequest
		case errors.Is(err, operatordecision.ErrFlood):
			status = http.StatusTooManyRequests // enough undecided asks already wait for the operator
		}
		http.Error(w, err.Error(), status)
		return
	}
	state, note, _ := s.usageModeNotifier.AskState(id)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"decision_id": id, "state": state, "note": note})
}

func (s *Server) handleUsageModeAskState(w http.ResponseWriter, r *http.Request) {
	state, note, ok := s.usageModeNotifier.AskState(r.URL.Query().Get("id"))
	if !ok {
		http.Error(w, "unknown decision", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"state": state, "note": note})
}

func isInvalidModeAsk(err error) bool { return errors.Is(err, usagemodes.ErrInvalid) }
