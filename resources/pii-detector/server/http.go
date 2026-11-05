package main

// http.go is the engine-agnostic HTTP contract for the PII detector:
//
//	POST /detect  {"text": "..."}            -> {"spans":[{category,start,end,text,confidence}], ...}
//	GET  /health                              -> {"status","model_loaded","model_id"}
//	GET  /info                                -> {"model_id","regex_pass":true,...}
//
// The detector always runs the deterministic regex/checksum pass and merges it
// with the NER model pass. When the model is unavailable the response is still
// a valid span set (regex-only) and `model_available:false` signals the
// degradation rather than failing the request.

import (
	"encoding/json"
	"io"
	"net/http"
)

const maxDetectBodyBytes = 1 << 20 // 1 MiB of text per request.

type detectRequest struct {
	Text string `json:"text"`
}

// detectResponse is the /detect wire contract. It is duplicated by the client
// type DetectResult (cli/internal/client/client.go) and the field shape of Span
// (detect.go) is mirrored by client.Span — the server and CLI are separate Go
// modules and cannot share a type. contractGoldenJSON (contract_test.go) freezes
// the exact wire bytes so a field drift on either side breaks a committed test.
type detectResponse struct {
	Spans          []Span `json:"spans"`
	ModelAvailable bool   `json:"model_available"`
	ModelID        string `json:"model_id,omitempty"`
}

type detectHandler struct {
	model NERModel
}

func newHandler(model NERModel) http.Handler {
	h := &detectHandler{model: model}
	mux := http.NewServeMux()
	mux.HandleFunc("/detect", h.detect)
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/info", h.info)
	return mux
}

func (h *detectHandler) detect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	var req detectRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxDetectBodyBytes)).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}

	spans, modelAvailable, modelID := h.spansFor(r, req.Text)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(detectResponse{
		Spans:          spans,
		ModelAvailable: modelAvailable,
		ModelID:        modelID,
	})
}

// spansFor is the shared detection path (used by /detect and reused in tests).
// It never returns a nil slice so the JSON `spans` field is always an array.
func (h *detectHandler) spansFor(r *http.Request, text string) (spans []Span, modelAvailable bool, modelID string) {
	regexSpans := RegexDetect(text)

	var modelSpans []Span
	if h.model != nil {
		ms, err := h.model.Detect(r.Context(), text)
		if err == nil {
			modelSpans = ms
			modelAvailable = true
			modelID = h.model.ModelID()
		}
		// On a model error (including the unavailable sentinel) we degrade to
		// regex-only and report model_available:false; we do not fail the call.
	}

	merged := MergeSpans(regexSpans, modelSpans)
	if merged == nil {
		merged = []Span{}
	}
	return merged, modelAvailable, modelID
}

func (h *detectHandler) health(w http.ResponseWriter, r *http.Request) {
	modelLoaded := false
	modelID := ""
	if h.model != nil {
		// Probe with the request context, never a nil ctx: a real ONNX engine
		// that inspects ctx.Done()/ctx.Err() would panic on nil.
		if _, err := h.model.Detect(r.Context(), ""); err == nil {
			modelLoaded = true
			modelID = h.model.ModelID()
		}
	}
	w.Header().Set("Content-Type", "application/json")
	// The service is healthy as long as the regex pass works; the model is a
	// capability enhancement, not a readiness precondition.
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":       "ok",
		"model_loaded": modelLoaded,
		"model_id":     modelID,
	})
}

func (h *detectHandler) info(w http.ResponseWriter, _ *http.Request) {
	modelID := ""
	if h.model != nil {
		modelID = h.model.ModelID()
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"model_id":   modelID,
		"regex_pass": true,
		"categories": []string{
			CategoryEmail, CategoryPhone, CategorySSN, CategoryCreditCard, CategoryIBAN,
		},
	})
}

func (h *detectHandler) writeError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"detail": detail})
}
