package billing

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"landing-page-business-suite-api/internal/commerce"
)

// ReceiptDependencies keeps the transport layer unaware of platform receipt
// verification details. Validators are selected by source and all sources
// return the same entitlement payload.
type ReceiptDependencies struct {
	Validators   commerce.ReceiptValidators
	Register     func(context.Context, commerce.ReceiptValidators, commerce.Receipt) (*commerce.EntitlementPayload, error)
	UserIdentity func(context.Context) string
	WriteError   func(http.ResponseWriter, int, string, string)
}

type receiptRequest struct {
	Source string `json:"source"`
	Token  string `json:"token"`
}

// RegisterReceipt accepts a server-verifiable store receipt or purchase token.
// Identity comes from the authenticated consumer token, never from the body.
func RegisterReceipt(deps ReceiptDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity := strings.TrimSpace(deps.UserIdentity(r.Context()))
		if identity == "" {
			deps.WriteError(w, http.StatusUnauthorized, "Authentication required", "unauthorized")
			return
		}
		var request receiptRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			deps.WriteError(w, http.StatusBadRequest, "Invalid receipt request", "validation")
			return
		}
		if strings.TrimSpace(request.Source) == "" || strings.TrimSpace(request.Token) == "" {
			deps.WriteError(w, http.StatusBadRequest, "source and token are required", "validation")
			return
		}
		payload, err := deps.Register(r.Context(), deps.Validators, commerce.Receipt{Source: request.Source, Token: request.Token, UserIdentity: identity})
		if err != nil {
			status, kind := http.StatusUnprocessableEntity, "receipt_invalid"
			switch {
			case errors.Is(err, commerce.ErrReceiptBound):
				status, kind = http.StatusForbidden, "forbidden"
			case errors.Is(err, commerce.ErrReceiptReplay):
				status, kind = http.StatusConflict, "receipt_replay"
			case !errors.Is(err, commerce.ErrReceiptUnsupported) && !errors.Is(err, commerce.ErrReceiptInvalid):
				status, kind = http.StatusServiceUnavailable, "server_error"
			}
			deps.WriteError(w, status, "Receipt could not be registered", kind)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if encodeErr := json.NewEncoder(w).Encode(payload); encodeErr != nil {
			deps.WriteError(w, http.StatusInternalServerError, "Failed to encode entitlement", "server_error")
		}
	}
}
