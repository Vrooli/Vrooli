package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

func handleRecoveryState(engine *RecoveryEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := engine.State()
		writeJSON(w, http.StatusOK, state)
	}
}

func handleRecoveryTrigger(engine *RecoveryEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Force bool `json:"force"`
		}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}

		ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
		defer cancel()

		evt, err := engine.TriggerManual(ctx, body.Force)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, evt)
	}
}

func handleRecoveryEvents(engine *RecoveryEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		events, err := engine.ListEvents(50)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if events == nil {
			events = []RecoveryEvent{}
		}
		writeJSON(w, http.StatusOK, events)
	}
}

func handleCircuitReset(engine *RecoveryEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		engine.ResetCircuit()
		writeJSON(w, http.StatusOK, map[string]string{"status": "circuit_reset"})
	}
}
