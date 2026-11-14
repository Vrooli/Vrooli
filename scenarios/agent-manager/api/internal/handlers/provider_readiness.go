package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"agent-manager/internal/domain"
)

type ProviderReadinessReader interface {
	ReadProviderReadiness(context.Context) domain.ProviderReadiness
}

// ProviderReadinessHandler exposes the owner-derived provider state. It always
// returns a bounded JSON observation; inability to verify is encoded as unknown.
func ProviderReadinessHandler(reader ProviderReadinessReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(reader.ReadProviderReadiness(r.Context()))
	}
}
