package flows

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	internalflows "device-control/internal/flows"
	"device-control/internal/module"

	"github.com/gorilla/mux"
)

type resolveTargetRequest struct {
	Target              string    `json:"target"`
	FrameBase64         string    `json:"frame_base64"`
	MediaType           string    `json:"media_type"`
	ConfidenceThreshold float64   `json:"confidence_threshold"`
	FallbackBounds      []float64 `json:"fallback_bounds,omitempty"`
	FallbackConfidence  float64   `json:"fallback_confidence,omitempty"`
	MaxDimension        int       `json:"max_dimension,omitempty"`
}

func Module(resolver *internalflows.Resolver) module.Module {
	return module.Module{
		Name: "flows",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/v1/flows/resolve-target", func(w http.ResponseWriter, req *http.Request) {
				handleResolveTarget(w, req, resolver)
			}).Methods(http.MethodPost)
		},
		Endpoints: Endpoints,
	}
}

func handleResolveTarget(w http.ResponseWriter, req *http.Request, resolver *internalflows.Resolver) {
	if resolver == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "unavailable", "code": "vision_route_unavailable", "reason": "resolver_not_configured"})
		return
	}
	req.Body = http.MaxBytesReader(w, req.Body, 12<<20)
	var input resolveTargetRequest
	if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": "invalid_argument", "code": "invalid_request", "message": "request body must be valid JSON"})
		return
	}
	if input.MediaType == "" {
		input.MediaType = "image/png"
	}
	raw, err := base64.StdEncoding.DecodeString(input.FrameBase64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": "invalid_argument", "code": "invalid_frame", "message": "frame_base64 is not valid base64"})
		return
	}
	frame, err := internalflows.PrepareFrame(raw, input.MediaType, input.MaxDimension)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": "invalid_argument", "code": "invalid_frame", "message": safeError(err)})
		return
	}
	result, err := resolver.Resolve(req.Context(), internalflows.Request{
		Target:              input.Target,
		Frame:               frame,
		ConfidenceThreshold: input.ConfidenceThreshold,
		FallbackBounds:      input.FallbackBounds,
		FallbackConfidence:  input.FallbackConfidence,
	})
	if err != nil {
		var unavailable *internalflows.UnavailableError
		if errors.As(err, &unavailable) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "unavailable", "code": "vision_route_unavailable", "reason": unavailable.Reason, "evidence": result.Evidence})
			return
		}
		if errors.Is(err, internalflows.ErrUnresolved) {
			writeJSON(w, http.StatusUnprocessableEntity, result)
			return
		}
		if errors.Is(err, internalflows.ErrInvalidRequest) || errors.Is(err, internalflows.ErrInvalidFrame) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"status": "invalid_argument", "code": "invalid_request", "message": safeError(err)})
			return
		}
		writeJSON(w, http.StatusBadGateway, map[string]any{"status": "failed", "code": "gateway_failed", "message": "ai-gateway request failed"})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func safeError(err error) string {
	if err == nil {
		return ""
	}
	message := strings.TrimSpace(err.Error())
	if len(message) > 240 {
		return message[:240]
	}
	return message
}

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "flows_resolve_target",
		Path:        "/api/v1/flows/resolve-target",
		Method:      http.MethodPost,
		Summary:     "Resolve a target through the device-control ladder.",
		Description: "Downscales a caller-owned frame, resolves locate.visual through ai-gateway, and records provider-neutral rung evidence.",
		Category:    "flows",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"target": "string (required)", "frame_base64": "base64 image (required)", "media_type": "image/png or image/jpeg",
			"confidence_threshold": "number 0..1", "fallback_bounds": "array<number> (optional normalized anchor)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"status": "resolved | unavailable | unresolved", "rung": "vision | visual-anchor", "confidence": "number 0..1", "evidence": "array<EvidenceEvent>",
		}},
		Errors: []module.ErrorDesc{
			{Status: http.StatusBadRequest, Code: "invalid_argument", Description: "Malformed frame, target, or threshold"},
			{Status: http.StatusServiceUnavailable, Code: "vision_route_unavailable", Description: "No ai-gateway vision route is available; no provider fallback is attempted"},
			{Status: http.StatusUnprocessableEntity, Code: "unresolved", Description: "Vision confidence was below the caller threshold and no anchor fallback resolved the target"},
		},
		// The flow executor accepts caller-captured base64 image JSON. Until
		// the scenario flow envelope gets its own proto, this is the same
		// explicit REST exception used by operational JSON probes; it is
		// still provider-neutral and covered by the handler contract tests.
		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			Note:   "Caller-owned frame execution uses a JSON transport until the device-control flow envelope is proto-typed.",
		},
		Examples: []module.Example{{Name: "Resolve a screenshot target", Curl: "curl http://localhost:${API_PORT}/api/v1/flows/resolve-target -H 'Content-Type: application/json' -d '{\"target\":\"settings\",\"frame_base64\":\"...\",\"media_type\":\"image/png\"}'"}},
	},
}
