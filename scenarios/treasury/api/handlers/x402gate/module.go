// Package x402gate exposes the protocol-shaped inbound payment gate. It stays
// REST because x402 interoperability is defined by HTTP 402 and payment
// headers, not by Treasury's Connect schema.
package x402gate

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"treasury/internal/module"
	"treasury/internal/operatorauth"
	x402rail "treasury/internal/rail/x402"

	"github.com/gorilla/mux"
)

const maxRequestBytes = 64 << 10

type handler struct {
	gate       *x402rail.Gate
	authorizer operatorauth.Authorizer
}

func Module(gate *x402rail.Gate, authorizer operatorauth.Authorizer) module.Module {
	h := &handler{gate: gate, authorizer: authorizer}
	return module.Module{Name: "x402gate", Mount: func(router *mux.Router) {
		router.HandleFunc("/api/v1/x402/prices", h.declare).Methods(http.MethodPost)
		router.HandleFunc("/api/v1/x402/prices/{price_id}/admit", h.admit).Methods(http.MethodPost)
	}, Endpoints: Endpoints}
}

func Schema() string { return x402rail.Schema() }

func (h *handler) declare(writer http.ResponseWriter, request *http.Request) {
	if h.gate == nil || h.authorizer == nil {
		writeError(writer, http.StatusServiceUnavailable, "x402 gate is unavailable")
		return
	}
	if _, err := h.authorizer.Authorize(request.Context(), request.Header); err != nil {
		status := http.StatusForbidden
		if errors.Is(err, operatorauth.ErrRequired) {
			status = http.StatusUnauthorized
		}
		if errors.Is(err, operatorauth.ErrUnavailable) {
			status = http.StatusServiceUnavailable
		}
		writeError(writer, status, "operator authorization failed")
		return
	}
	var price x402rail.Price
	decoder := json.NewDecoder(io.LimitReader(request.Body, maxRequestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&price); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid price declaration")
		return
	}
	created, err := h.gate.Declare(request.Context(), price)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(writer, http.StatusCreated, created)
}

func (h *handler) admit(writer http.ResponseWriter, request *http.Request) {
	if h.gate == nil {
		writeError(writer, http.StatusServiceUnavailable, "x402 gate is unavailable")
		return
	}
	priceID := strings.TrimSpace(mux.Vars(request)["price_id"])
	payment := strings.TrimSpace(request.Header.Get("Payment-Signature"))
	if payment == "" {
		h.challenge(writer, request.Context(), priceID, "payment signature is required")
		return
	}
	admission, err := h.gate.Admit(request.Context(), priceID, payment)
	if err != nil {
		if errors.Is(err, x402rail.ErrUnknown) || errors.Is(err, x402rail.ErrInProgress) {
			writeError(writer, http.StatusServiceUnavailable, "payment settlement is not yet conclusive")
			return
		}
		h.challenge(writer, request.Context(), priceID, err.Error())
		return
	}
	receipt, err := json.Marshal(map[string]any{"success": true, "payer": admission.Payer, "transaction": admission.TransactionID, "network": admission.Network})
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "encode payment receipt")
		return
	}
	writer.Header().Set("Payment-Response", base64.StdEncoding.EncodeToString(receipt))
	writeJSON(writer, http.StatusOK, admission)
}

func (h *handler) challenge(writer http.ResponseWriter, ctx context.Context, priceID, detail string) {
	header, err := h.gate.PaymentRequired(ctx, priceID)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, x402rail.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(writer, status, err.Error())
		return
	}
	writer.Header().Set("Payment-Required", header)
	writeJSON(writer, http.StatusPaymentRequired, map[string]string{"error": detail})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeError(writer http.ResponseWriter, status int, detail string) {
	writeJSON(writer, status, map[string]string{"error": detail})
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "x402_price_declare", Path: "/api/v1/x402/prices", Method: "POST", Summary: "Declare operator-owned x402 price terms", Category: "x402", RESTException: x402RESTException()},
	{ID: "x402_payment_admit", Path: "/api/v1/x402/prices/{price_id}/admit", Method: "POST", Summary: "Verify and settle an x402 payment before admitting a priced request", Category: "x402", RESTException: x402RESTException()},
}

func x402RESTException() *module.RESTException {
	externalJSON := module.RESTPayload{Transport: "json", Conformance: "external_shape"}
	return &module.RESTException{
		Reason: module.RESTReasonThirdPartyShape,
		Note:   "x402 is standardized as HTTP 402 plus Payment-Required, Payment-Signature, and Payment-Response headers; its request, success, challenge, and error bodies intentionally retain that external JSON shape.",
		ProtoPayloads: &module.RESTProtoPayloads{
			Request:  externalJSON,
			Response: externalJSON,
			Error:    externalJSON,
		},
	}
}
