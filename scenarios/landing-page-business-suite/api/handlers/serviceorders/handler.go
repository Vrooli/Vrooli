// Package serviceorders exposes the test-only LPBS service-order transport.
package serviceorders

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"landing-page-business-suite-api/internal/commerce"
)

const maxRequestBytes = 64 << 10

// Runtime is deliberately limited to the service-order operations needed by
// the maintained test transport.
type Runtime interface {
	BeginFromOffer(context.Context, commerce.ServiceOrderOfferSource, commerce.ServiceOrderOfferRequest) (*commerce.ServiceOrder, error)
	GetOrder(context.Context, string) (*commerce.ServiceOrder, error)
	VerifyAndCapture(context.Context, string) (*commerce.ServiceOrder, error)
	Reconcile(context.Context, string) (*commerce.ServiceOrder, error)
}

type DeliveryResult struct {
	Format string
	Body   []byte
}

type ResultReader interface {
	ReadResult(context.Context, commerce.ServiceOrder) (DeliveryResult, error)
}

type Dependencies struct {
	Runtime  Runtime
	Offers   commerce.ServiceOrderOfferSource
	Results  ResultReader
	TestMode bool
}

type Handler struct{ deps Dependencies }

func NewHandler(deps Dependencies) (*Handler, error) {
	if !deps.TestMode {
		return nil, errors.New("service-order HTTP transport is available only in test mode")
	}
	if deps.Runtime == nil || deps.Offers == nil {
		return nil, errors.New("service-order runtime and canonical offer source are required")
	}
	return &Handler{deps: deps}, nil
}

func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/v1/service-orders/offer", h.offer).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/service-orders", h.begin).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/service-orders/{order_id}", h.get).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/service-orders/{order_id}/result", h.result).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/service-orders/{order_id}/verify-capture", h.verifyCapture).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/service-orders/{order_id}/reconcile", h.reconcile).Methods(http.MethodPost)
}

type offerResponse struct {
	OfferID      string `json:"offer_id"`
	OfferVersion int    `json:"offer_version"`
	AmountMinor  int64  `json:"amount_minor"`
	Currency     string `json:"currency"`
}

func (h *Handler) offer(w http.ResponseWriter, r *http.Request) {
	if !isLoopbackRequest(r) {
		writeError(w, http.StatusForbidden, "test service-order transport requires a loopback client")
		return
	}
	query := r.URL.Query()
	offerIDs, versionValues := query["offer_id"], query["offer_version"]
	if len(query) != 2 || len(offerIDs) != 1 || len(versionValues) != 1 || strings.TrimSpace(offerIDs[0]) == "" {
		writeError(w, http.StatusBadRequest, "exactly one offer_id and offer_version are required")
		return
	}
	version, err := strconv.Atoi(versionValues[0])
	if err != nil || version < 1 {
		writeError(w, http.StatusBadRequest, "offer_version must be a positive integer")
		return
	}
	quote, err := h.deps.Offers.ResolveServiceOrderQuote(r.Context(), offerIDs[0], version)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if quote.OfferID != strings.TrimSpace(offerIDs[0]) || quote.OfferVersion != version || quote.AmountMinor <= 0 || len(quote.Currency) != 3 || quote.Currency != strings.ToUpper(quote.Currency) {
		writeError(w, http.StatusBadGateway, "canonical service offer returned invalid quote terms")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, offerResponse{OfferID: quote.OfferID, OfferVersion: quote.OfferVersion, AmountMinor: quote.AmountMinor, Currency: quote.Currency})
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !isLoopbackRequest(r) {
		writeError(w, http.StatusForbidden, "test service-order transport requires a loopback client")
		return
	}
	router := mux.NewRouter().StrictSlash(false)
	h.RegisterRoutes(router)
	router.ServeHTTP(w, r)
}

type beginRequest struct {
	OfferID           string          `json:"offer_id"`
	OfferVersion      int             `json:"offer_version"`
	Door              string          `json:"door"`
	AgentManagerRunID string          `json:"agent_manager_run_id,omitempty"`
	IdempotencyKey    string          `json:"idempotency_key"`
	JobInput          json.RawMessage `json:"job_input"`
	PaymentSignature  string          `json:"payment_signature,omitempty"`
}

func (h *Handler) begin(w http.ResponseWriter, r *http.Request) {
	if !isLoopbackRequest(r) {
		writeError(w, http.StatusForbidden, "test service-order transport requires a loopback client")
		return
	}
	var request beginRequest
	if err := decodeStrict(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(request.OfferID) == "" || request.OfferVersion < 1 || strings.TrimSpace(request.IdempotencyKey) == "" {
		writeError(w, http.StatusBadRequest, "offer_id, positive offer_version, and idempotency_key are required")
		return
	}
	order, err := h.deps.Runtime.BeginFromOffer(r.Context(), h.deps.Offers, commerce.ServiceOrderOfferRequest{
		OfferID: request.OfferID, OfferVersion: request.OfferVersion, Door: request.Door,
		AgentManagerRunID: request.AgentManagerRunID,
		IdempotencyKey:    request.IdempotencyKey, JobInput: request.JobInput, PaymentSignature: request.PaymentSignature,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	if !isLoopbackRequest(r) {
		writeError(w, http.StatusForbidden, "test service-order transport requires a loopback client")
		return
	}
	order, err := h.deps.Runtime.GetOrder(r.Context(), mux.Vars(r)["order_id"])
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (h *Handler) result(w http.ResponseWriter, r *http.Request) {
	if !isLoopbackRequest(r) {
		writeError(w, http.StatusForbidden, "test service-order transport requires a loopback client")
		return
	}
	order, err := h.deps.Runtime.GetOrder(r.Context(), mux.Vars(r)["order_id"])
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if order.State != commerce.ServiceOrderReconciled {
		writeError(w, http.StatusConflict, "order result is available only after reconciliation")
		return
	}
	if h.deps.Results == nil {
		writeError(w, http.StatusServiceUnavailable, "test delivery result reader is unavailable")
		return
	}
	result, err := h.deps.Results.ReadResult(r.Context(), *order)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "verified test delivery result is unavailable")
		return
	}
	if result.Format != "markdown" || len(result.Body) == 0 {
		writeError(w, http.StatusBadGateway, "test delivery result must be nonempty Markdown")
		return
	}
	digest := sha256.Sum256(result.Body)
	filename := fmt.Sprintf("company-market-scan-%s.md", order.JobID)
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("X-Content-SHA256", hex.EncodeToString(digest[:]))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(result.Body)
}

func (h *Handler) verifyCapture(w http.ResponseWriter, r *http.Request) {
	if !isLoopbackRequest(r) {
		writeError(w, http.StatusForbidden, "test service-order transport requires a loopback client")
		return
	}
	if err := requireEmptyBody(w, r); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	order, err := h.deps.Runtime.VerifyAndCapture(r.Context(), mux.Vars(r)["order_id"])
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (h *Handler) reconcile(w http.ResponseWriter, r *http.Request) {
	if !isLoopbackRequest(r) {
		writeError(w, http.StatusForbidden, "test service-order transport requires a loopback client")
		return
	}
	if err := requireEmptyBody(w, r); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	order, err := h.deps.Runtime.Reconcile(r.Context(), mux.Vars(r)["order_id"])
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func decodeStrict(w http.ResponseWriter, r *http.Request, target any) error {
	if r.Body == nil {
		return errors.New("request body is required")
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("request must contain one JSON value")
		}
		return err
	}
	return nil
}

func requireEmptyBody(w http.ResponseWriter, r *http.Request) error {
	if r.Body == nil {
		return nil
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBytes))
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return errors.New("request body must be empty")
}

func isLoopbackRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func writeServiceError(w http.ResponseWriter, err error) {
	status := http.StatusUnprocessableEntity
	if errors.Is(err, sql.ErrNoRows) {
		status = http.StatusNotFound
	}
	if errors.Is(err, commerce.ErrServiceOrderIdempotencyConflict) {
		status = http.StatusConflict
	}
	writeError(w, status, err.Error())
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
