package handler

import (
	"context"
	"net/http"
	"time"
)

func HandleTunnelHealth(checker TunnelChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		status := checker.Check(ctx)
		writeJSON(w, http.StatusOK, status)
	}
}
