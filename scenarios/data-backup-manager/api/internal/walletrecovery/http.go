package walletrecovery

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// Authorizer is mandatory for the internal HTTP bridge that scenario-to-cloud
// can call. This package intentionally does not choose or mint service
// credentials; production wiring must provide DBM's existing service identity
// verifier before mounting the handler.
type Authorizer interface{ AuthorizeWalletRecovery(*http.Request) error }

type AttestRequest struct {
	ReservationID string `json:"reservation_id"`
	TargetID      string `json:"target_id"`
	DestinationID string `json:"destination_id"`
}

// NewHTTPHandler implements POST /api/wallet-recovery/attest. It requires an
// explicit service-to-service authorizer and a configured signer/evidence
// reader; absent either it returns 503 without mutating wallet data.
func NewHTTPHandler(service Service, auth Authorizer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if auth == nil {
			http.Error(w, "wallet recovery attestation is not configured", http.StatusServiceUnavailable)
			return
		}
		if err := auth.AuthorizeWalletRecovery(r); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if r.Body == nil {
			http.Error(w, "request body required", http.StatusBadRequest)
			return
		}
		var req AttestRequest
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		var extra any
		if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
			http.Error(w, "request must contain one JSON object", http.StatusBadRequest)
			return
		}
		result, err := service.Attest(r.Context(), req.ReservationID, req.TargetID, req.DestinationID)
		if err != nil {
			if errors.Is(err, ErrNotConfigured) {
				http.Error(w, "wallet recovery attestation is not configured", http.StatusServiceUnavailable)
				return
			}
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	})
}

type FinishReservationRequest struct {
	ReservationID string `json:"reservation_id"`
	TargetID      string `json:"target_id"`
	State         string `json:"state"`
}

// NewReservationFinishHandler exposes reservation commit and release actions.
// A commit-pending reservation cannot be released until DBM receives a
// verifiable Treasury commit acknowledgment; that owner contract is not yet
// wired, so release fails closed.
func NewReservationFinishHandler(store ReservationStore, auth Authorizer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if auth == nil {
			http.Error(w, "wallet recovery reservation is not configured", http.StatusServiceUnavailable)
			return
		}
		if err := auth.AuthorizeWalletRecovery(r); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		var req FinishReservationRequest
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if store == nil || req.ReservationID == "" || req.TargetID == "" {
			http.Error(w, "reservation and target are required", http.StatusBadRequest)
			return
		}
		var err error
		switch req.State {
		case "commit":
			err = store.Commit(r.Context(), req.ReservationID, req.TargetID)
		case "release":
			err = store.Release(r.Context(), req.ReservationID, req.TargetID)
		default:
			http.Error(w, "state must be commit or release", http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, "reservation could not be finalized", http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"reservation_id": req.ReservationID, "state": req.State})
	})
}
