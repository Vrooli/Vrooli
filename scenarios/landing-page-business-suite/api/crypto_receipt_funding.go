package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/database"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	"google.golang.org/protobuf/encoding/protojson"
)

type acceptedCryptoReceiptRequest struct {
	Receipt json.RawMessage `json:"receipt"`
}

// requireTreasuryReceiptService is intentionally service-only. Browser/admin
// sessions cannot submit accounting receipts or choose an LPBS grant amount.
func (s *Server) requireTreasuryReceiptService(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		const prefix = "Bearer "
		provided := ""
		if strings.HasPrefix(header, prefix) {
			provided = strings.TrimSpace(strings.TrimPrefix(header, prefix))
		}
		expected, err := resolveAuthorityCredential("LPBS_" + "SERVICE_SECRET")
		if err != nil || provided == "" || len(provided) != len(expected) || subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			writeJSONError(w, http.StatusUnauthorized, "Invalid service authorization.", ApiErrorTypeUnauthorized)
			return
		}
		next(w, r)
	}
}

func (s *Server) acceptTreasuryCryptoReceipt(w http.ResponseWriter, r *http.Request) {
	if s.cryptoReceiptGrant == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "Crypto receipt grant owner is unavailable.", ApiErrorTypeServerError)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 128*1024)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid accepted receipt request.", ApiErrorTypeValidation)
		return
	}
	var request acceptedCryptoReceiptRequest
	if err := json.Unmarshal(body, &request); err != nil || len(request.Receipt) == 0 {
		writeJSONError(w, http.StatusBadRequest, "An accepted Money Ledger receipt is required.", ApiErrorTypeValidation)
		return
	}
	receipt := new(ingestpb.AcceptedCryptoReceipt)
	if err := protojson.Unmarshal(request.Receipt, receipt); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid accepted Money Ledger receipt.", ApiErrorTypeValidation)
		return
	}
	ctx, err := acceptedReceiptDatabaseContext(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid test database mode.", ApiErrorTypeValidation)
		return
	}
	lot, err := s.cryptoReceiptGrant.GrantAcceptedReceipt(ctx, receipt)
	if err != nil {
		writeJSONError(w, http.StatusUnprocessableEntity, err.Error(), ApiErrorTypeValidation)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"credit_lot_id": lot.ID, "receipt_id": lot.ReceiptID, "credits": lot.Credits})
}

func acceptedReceiptDatabaseContext(r *http.Request) (context.Context, error) {
	ctx := r.Context()
	mode := strings.TrimSpace(r.Header.Get("X-Vrooli-Test-Mode"))
	if mode == "" {
		return ctx, nil
	}
	if mode != "1" {
		return nil, errors.New("invalid test database mode")
	}
	return database.WithTestMode(ctx), nil
}
