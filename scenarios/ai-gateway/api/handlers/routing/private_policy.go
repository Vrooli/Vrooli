package routing

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/gorilla/mux"

	internalrouting "ai-gateway/internal/routing"
)

const (
	privatePlanPath    = "/api/v1/private/provider-routes/plan"
	privateExecutePath = "/api/v1/private/provider-routes/execute"
	privateReceiptPath = "/api/v1/private/provider-routes/receipt/{operation_id}"
)

func mountPrivatePolicyRoutes(router *mux.Router, h *connectHandler) {
	router.HandleFunc("/api/v1/private/provider-routes/models", h.modelsPrivate).Methods(http.MethodGet)
	router.HandleFunc(privatePlanPath, h.planPrivate).Methods(http.MethodPost)
	router.HandleFunc(privateExecutePath, h.executePrivate).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/private/provider-routes/stream", h.streamPrivate).Methods(http.MethodPost)
	router.HandleFunc(privateReceiptPath, h.receiptPrivate).Methods(http.MethodGet)
}

func (h *connectHandler) planPrivate(w http.ResponseWriter, r *http.Request) {
	var request internalrouting.PrivatePlanRequest
	if err := decodePrivateJSON(w, r, &request); err != nil {
		writePrivateError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	plan, err := h.private.Plan(r.Context(), request)
	if err != nil {
		if errors.Is(err, internalrouting.ErrPrivatePlanConflict) {
			writePrivateError(w, http.StatusConflict, "operation_conflict", err.Error())
			return
		}
		if errors.Is(err, internalrouting.ErrPrivateFeatureNotFound) {
			writePrivateError(w, http.StatusNotFound, "feature_not_found", err.Error())
			return
		}
		writePrivateError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writePrivateJSON(w, http.StatusOK, plan)
}

func (h *connectHandler) executePrivate(w http.ResponseWriter, r *http.Request) {
	var request internalrouting.PrivateExecuteRequest
	if err := decodePrivateJSON(w, r, &request); err != nil {
		writePrivateError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	receipt, err := h.private.Execute(r.Context(), request)
	if err != nil {
		if errors.Is(err, internalrouting.ErrPrivateReceiptConflict) {
			writePrivateError(w, http.StatusConflict, "operation_conflict", err.Error())
			return
		}
		if errors.Is(err, internalrouting.ErrPrivatePlanConflict) {
			writePrivateError(w, http.StatusConflict, "plan_conflict", err.Error())
			return
		}
		if errors.Is(err, internalrouting.ErrPrivatePlanNotEligible) {
			writePrivateError(w, http.StatusConflict, "plan_not_found", err.Error())
			return
		}
		writePrivateError(w, http.StatusInternalServerError, "receipt_unavailable", err.Error())
		return
	}
	writePrivateJSON(w, http.StatusOK, receipt)
}

func (h *connectHandler) receiptPrivate(w http.ResponseWriter, r *http.Request) {
	receipt, err := h.private.Receipt(r.Context(), mux.Vars(r)["operation_id"])
	if err != nil {
		if errors.Is(err, internalrouting.ErrPrivateReceiptMissing) {
			writePrivateError(w, http.StatusNotFound, "receipt_not_found", err.Error())
			return
		}
		writePrivateError(w, http.StatusInternalServerError, "receipt_unavailable", err.Error())
		return
	}
	writePrivateJSON(w, http.StatusOK, receipt)
}

func decodePrivateJSON(w http.ResponseWriter, r *http.Request, target any) error {
	if r.Method != http.MethodPost {
		return errors.New("method must be POST")
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return errors.New("content-type must be application/json")
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("request must contain one JSON value")
	}
	return nil
}

func writePrivateJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writePrivateError(w http.ResponseWriter, status int, code, message string) {
	writePrivateJSON(w, status, map[string]string{"error": code, "message": message})
}

func (h *connectHandler) streamPrivate(w http.ResponseWriter, r *http.Request) {
	var request internalrouting.PrivateStreamRequest
	if err := decodePrivateJSON(w, r, &request); err != nil {
		writePrivateError(w, 400, "invalid_request", err.Error())
		return
	}
	receipt, err := h.private.Stream(r.Context(), request, w)
	if err != nil {
		if errors.Is(err, internalrouting.ErrPrivateDispatchUncertain) {
			writePrivateError(w, http.StatusBadGateway, "receipt_pending", "provider outcome requires reconciliation")
			return
		}
		writePrivateError(w, 409, "stream_refused", err.Error())
		return
	}
	if w.Header().Get("Content-Type") != "text/event-stream" {
		writePrivateJSON(w, 200, receipt)
		return
	}
	if receipt.State != "completed" {
		_, _ = w.Write([]byte("data: {\"error\":{\"message\":\"upstream response was incomplete\",\"type\":\"upstream_error\"}}\n\n"))
	}
	data, _ := json.Marshal(receipt)
	_, _ = w.Write(append(append([]byte("event: receipt\ndata: "), data...), []byte("\n\ndata: [DONE]\n\n")...))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func (h *connectHandler) modelsPrivate(w http.ResponseWriter, r *http.Request) {
	models, reason, err := h.private.Models(r.Context(), "agent.session", r.URL.Query().Get("harness"))
	if err != nil {
		writePrivateError(w, 503, "policy_unavailable", err.Error())
		return
	}
	data := []map[string]any{}
	for _, model := range models {
		data = append(data, map[string]any{"id": model, "name": model})
	}
	writePrivateJSON(w, 200, map[string]any{"data": data, "reason": reason})
}
