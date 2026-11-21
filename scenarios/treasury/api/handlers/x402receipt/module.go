// Package x402receipt exposes a narrow, service-authenticated readback for an
// accepted Treasury x402 receipt. LPBS links its job to this receipt; it does
// not create another seller-side Money Ledger posting.
package x402receipt

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"treasury/internal/ledger"
	"treasury/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
)

const (
	path               = "/api/v1/x402/receipts"
	audience           = "scenario-authenticator:treasury-x402-receipt"
	requiredCapability = "treasury:x402-receipt-read"
	externalIDPrefix   = "treasury-x402-inflow:"
)

var transactionHash = regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)

type Reader interface {
	GetAcceptedReceiptByExternalID(context.Context, string) (ledger.AcceptedReceipt, error)
}

type handler struct{ reader Reader }

func Module(reader Reader) module.Module {
	h := &handler{reader: reader}
	identity := authn.NewScenarioAuthenticatorProvider(authn.JWTConfig{
		Issuer: strings.TrimSpace(os.Getenv("VROOLI_AUTH_SCENARIO_ISSUER")), Audience: audience,
		JWKSURL: strings.TrimSpace(os.Getenv("VROOLI_AUTH_SCENARIO_JWKS_URL")), ExclusiveAudience: true,
	})
	protected := authn.Middleware(authn.Config{Providers: []authn.Provider{identity}})(http.HandlerFunc(h.get))
	return module.Module{Name: "x402receipt", Mount: func(router *mux.Router) {
		router.Handle(path, protected).Methods(http.MethodGet)
	}, Endpoints: Endpoints}
}

type receiptResponse struct {
	ExternalID  string `json:"external_id"`
	PostingID   string `json:"posting_id"`
	ReceiptID   string `json:"receipt_id"`
	AcceptedAt  string `json:"accepted_at"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
	Basis       string `json:"basis"`
}

func (h *handler) get(writer http.ResponseWriter, request *http.Request) {
	if _, err := authn.RequireService(request.Context()); err != nil {
		writeError(writer, http.StatusUnauthorized, "verified service identity is required")
		return
	}
	if _, err := authn.RequireCapability(request.Context(), requiredCapability); err != nil {
		writeError(writer, http.StatusForbidden, "Treasury x402 receipt-read capability is required")
		return
	}
	if h.reader == nil {
		writeError(writer, http.StatusServiceUnavailable, "Treasury receipt readback is unavailable")
		return
	}
	externalID := strings.TrimSpace(request.URL.Query().Get("external_id"))
	if !strings.HasPrefix(externalID, externalIDPrefix) || !transactionHash.MatchString(strings.TrimPrefix(externalID, externalIDPrefix)) {
		writeError(writer, http.StatusBadRequest, "one Treasury x402 transaction external_id is required")
		return
	}
	value, err := h.reader.GetAcceptedReceiptByExternalID(request.Context(), externalID)
	if errors.Is(err, ledger.ErrNotFound) {
		writeError(writer, http.StatusNotFound, "accepted Treasury receipt was not found")
		return
	}
	if errors.Is(err, ledger.ErrAmbiguousAcceptedReceipt) {
		writeError(writer, http.StatusConflict, "Treasury receipt identity is ambiguous")
		return
	}
	if err != nil {
		writeError(writer, http.StatusServiceUnavailable, "Treasury receipt readback failed")
		return
	}
	if value.ExternalID != externalID || value.PostingID == "" || value.ReceiptID == "" || value.AcceptedAt.IsZero() {
		writeError(writer, http.StatusServiceUnavailable, "Treasury returned an incomplete accepted receipt")
		return
	}
	writeJSON(writer, http.StatusOK, receiptResponse{
		ExternalID: value.ExternalID, PostingID: value.PostingID, ReceiptID: value.ReceiptID,
		AcceptedAt: value.AcceptedAt.UTC().Format(time.RFC3339Nano), AmountMinor: value.AmountMinor,
		Currency: value.Currency, Basis: value.Basis,
	})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, map[string]string{"error": message})
}

var _ Reader = (*ledger.SQLiteRepository)(nil)
