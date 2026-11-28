package walletrecovery

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/receiptsigning"
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
	CommitAckJSON string `json:"commit_ack_json,omitempty"`
}

type ReservationAckStore interface {
	ReleaseAcknowledged(context.Context, receiptsigning.WalletRecoveryCommitAcknowledgement, string) error
}

type CommitAckVerifier interface {
	Verify(context.Context, receiptsigning.SignatureEnvelope, []byte) error
}

type signedCommitAcknowledgement struct {
	Acknowledgement receiptsigning.WalletRecoveryCommitAcknowledgement `json:"acknowledgement"`
	Signature       receiptsigning.SignatureEnvelope                   `json:"signature"`
}

// NewReservationFinishHandler exposes reservation commit and release actions.
// Operator authorization may pin an attestation; release additionally
// requires a Treasury signature bound to the durable reservation.
func NewReservationFinishHandler(store ReservationStore, auth Authorizer, verifier CommitAckVerifier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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
			if auth == nil {
				http.Error(w, "wallet recovery reservation is not configured", http.StatusServiceUnavailable)
				return
			}
			if err := auth.AuthorizeWalletRecovery(r); err != nil {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			err = store.Commit(r.Context(), req.ReservationID, req.TargetID)
		case "release":
			ackStore, ok := store.(ReservationAckStore)
			if !ok || verifier == nil || strings.TrimSpace(req.CommitAckJSON) == "" {
				err = ErrCommitAcknowledgementRequired
				break
			}
			var signed signedCommitAcknowledgement
			dec := json.NewDecoder(strings.NewReader(req.CommitAckJSON))
			dec.DisallowUnknownFields()
			if decodeErr := dec.Decode(&signed); decodeErr != nil || signed.Acknowledgement.ReservationID != req.ReservationID || signed.Acknowledgement.TargetID != req.TargetID {
				err = ErrCommitAcknowledgementRequired
				break
			}
			var extra any
			if decodeErr := dec.Decode(&extra); !errors.Is(decodeErr, io.EOF) {
				err = ErrCommitAcknowledgementRequired
				break
			}
			canonical, canonErr := receiptsigning.CanonicalJSON(signed.Acknowledgement)
			if canonErr != nil || signed.Signature.Purpose != receiptsigning.PurposeWalletRecoveryCommitAcknowledgement || verifier.Verify(r.Context(), signed.Signature, canonical) != nil {
				err = ErrCommitAcknowledgementRequired
				break
			}
			err = ackStore.ReleaseAcknowledged(r.Context(), signed.Acknowledgement, receiptsigning.Digest(canonical))
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
