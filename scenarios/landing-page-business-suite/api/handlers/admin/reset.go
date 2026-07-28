package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type ResetDependencies struct {
	Reset    func(context.Context) error
	Now      func() time.Time
	LogError func(string, map[string]any)
}

func ResetDemoData(deps ResetDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := deps.Reset(r.Context()); err != nil {
			deps.LogError("admin_reset_failed", map[string]any{"error": err.Error()})
			http.Error(w, "failed to reset demo data", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"reset": true, "timestamp": deps.Now().UTC().Format(time.RFC3339)}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
