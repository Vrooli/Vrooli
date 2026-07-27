// Package config owns public landing configuration transport.
package config

import (
	"context"
	"net/http"
	"time"
)

type Dependencies struct {
	Get        func(context.Context, string) (any, error)
	TestMode   func(context.Context) bool
	Sleep      func(time.Duration)
	Delay      time.Duration
	WriteJSON  func(http.ResponseWriter, any)
	WriteError func(http.ResponseWriter, int, string, string)
	Log        func(string, map[string]any)
}

func Landing(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.TestMode(r.Context()) {
			deps.Sleep(deps.Delay)
		}
		variant := r.URL.Query().Get("variant")
		payload, err := deps.Get(r.Context(), variant)
		if err != nil {
			deps.Log("landing_config_failed", map[string]any{"variant": variant, "error": err.Error()})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to load landing config. Please try again.", "server_error")
			return
		}
		deps.WriteJSON(w, payload)
	}
}
