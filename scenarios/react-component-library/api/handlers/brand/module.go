// Package brand serves the workbench shell's brand identity.
//
// The route is browser-facing REST: the shell fetches it once and falls back
// to a neutral monogram when no brand is assigned. It is not a Connect domain
// because the payload is a tiny UI projection of another scenario's aggregate,
// not a resource this scenario owns.
package brand

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	internal "react-component-library/internal/brand"
	"react-component-library/internal/module"
)

// Handler adapts a brand.Resolver onto HTTP.
type Handler struct {
	resolver internal.Resolver
	logger   *log.Logger
}

// NewHandler wires a resolver. A nil logger uses the standard logger.
func NewHandler(resolver internal.Resolver, logger *log.Logger) *Handler {
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{resolver: resolver, logger: logger}
}

// Module mounts GET /api/brand.
func Module(resolver internal.Resolver) module.Module {
	h := NewHandler(resolver, nil)
	return module.Module{
		Name: "brand",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/brand", h.get).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "brand_get",
		Path:        "/api/brand",
		Method:      http.MethodGet,
		Summary:     "Read the assigned brand",
		Description: "Projects the RCL brand assigned in Brand Manager into the shell identity. Returns assigned=false with null fields when Brand Manager is unavailable or has no RCL record.",
		Category:    "brand",
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"assigned": "boolean",
			"name":     "string",
			"mark":     "string|null (data URI or absolute URL)",
			"accent":   "string|null (CSS colour)",
			"wordmark": "string|null",
		}},
		RESTException: &module.RESTException{
			Reason: module.RESTReasonBrowserSurface,
			Note:   "The workbench shell fetches a small UI projection with a plain GET; the aggregate is owned and served by Brand Manager over Connect-RPC.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
				Response: module.RESTPayload{Transport: "json", Conformance: "external_shape"},
				Error:    module.RESTPayload{ProtoFullName: "vrooli.react_component_library.v1.errors.ErrorEnvelope", Transport: "json", Conformance: "protojson"},
			},
		},
	},
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	resolved, err := h.resolver.Resolve(r.Context())
	if err != nil {
		// Resolve already degrades upstream failures to "unassigned"; an error
		// here is unexpected, and the shell's fallback is still correct.
		h.logger.Printf("brand: resolve: %v", err)
		resolved = internal.Brand{}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resolved)
}
