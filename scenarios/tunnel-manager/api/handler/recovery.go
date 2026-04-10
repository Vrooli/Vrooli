package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"tunnel-manager/domain"
)

func HandleRecoveryState(mgr RecoveryManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := mgr.State()
		writeJSON(w, http.StatusOK, state)
	}
}

func HandleRecoveryTrigger(mgr RecoveryManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Force bool `json:"force"`
		}
		if r.Body != nil && r.Body != http.NoBody {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
				writeError(w, domain.ErrValidation("invalid JSON body"))
				return
			}
		}

		ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
		defer cancel()

		evt, err := mgr.TriggerManual(ctx, body.Force)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, evt)
	}
}

func HandleRecoveryEvents(mgr RecoveryManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		events, err := mgr.ListEvents(50)
		if err != nil {
			writeError(w, err)
			return
		}
		if events == nil {
			events = []domain.RecoveryEvent{}
		}
		writeJSON(w, http.StatusOK, events)
	}
}

func HandleCircuitReset(mgr RecoveryManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mgr.ResetCircuit()
		writeJSON(w, http.StatusOK, map[string]string{"status": "circuit_reset"})
	}
}
