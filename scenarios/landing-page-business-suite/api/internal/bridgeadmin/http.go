package bridgeadmin

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
)

type Caller func(*http.Request) (string, bool)

func Handler(service *Service, caller Caller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if service == nil || caller == nil {
			http.Error(w, "Bridge dispatch is unavailable", http.StatusServiceUnavailable)
			return
		}
		who, ok := caller(r)
		if !ok || who == "" {
			http.Error(w, "administrator identity is unavailable", http.StatusUnauthorized)
			return
		}
		var req Request
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "invalid dispatch request", http.StatusBadRequest)
			return
		}
		receipt, err := service.Submit(r.Context(), who, req)
		if err != nil {
			status := http.StatusBadGateway
			if errors.Is(err, ErrOutOfScope) || errors.Is(err, ErrBridgeRefused) {
				status = http.StatusForbidden
			} else if receipt.Status == "" {
				status = http.StatusBadRequest
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error(), "receipt": receipt})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(receipt)
	}
}

func ReceiptHandler(service *Service, caller Caller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if service == nil || caller == nil {
			http.Error(w, "Bridge dispatch is unavailable", http.StatusServiceUnavailable)
			return
		}
		who, ok := caller(r)
		if !ok || who == "" {
			http.Error(w, "administrator identity is unavailable", http.StatusUnauthorized)
			return
		}
		receipt, err := service.Get(r.Context(), who, mux.Vars(r)["request_id"])
		if err != nil {
			http.Error(w, "dispatch receipt not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(receipt)
	}
}
