// DOC: docs/internal/SEAMS.md#api-handlers--domain-services
// DOC: docs/reference/api-endpoints.md#expectations
package handlers

import (
	"development-toolchain-validator/domain/expectation"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
)

// ExpectationHandler handles HTTP requests for expectation operations.
// [REQ:REQ-P0-004] Structural Expectation Config
// [REQ:REQ-P0-005] CLI Tool Expectation Config
type ExpectationHandler struct {
	service *expectation.Service
}

// NewExpectationHandler creates a new handler with the given service.
func NewExpectationHandler(svc *expectation.Service) *ExpectationHandler {
	return &ExpectationHandler{service: svc}
}

// RegisterRoutes adds expectation routes to the router.
func (h *ExpectationHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/expectations/structural", h.ListStructural).Methods("GET")
	r.HandleFunc("/api/v1/expectations/structural", h.CreateStructural).Methods("POST")
	r.HandleFunc("/api/v1/expectations/structural/{id}", h.GetStructural).Methods("GET")
	r.HandleFunc("/api/v1/expectations/structural/{id}", h.DeleteStructural).Methods("DELETE")

	r.HandleFunc("/api/v1/expectations/cli", h.ListCLI).Methods("GET")
	r.HandleFunc("/api/v1/expectations/cli", h.CreateCLI).Methods("POST")
	r.HandleFunc("/api/v1/expectations/cli/{id}", h.GetCLI).Methods("GET")
	r.HandleFunc("/api/v1/expectations/cli/{id}", h.DeleteCLI).Methods("DELETE")
}

// ListStructural handles GET /api/v1/expectations/structural
func (h *ExpectationHandler) ListStructural(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	connectionID := r.URL.Query().Get("connection_id")

	opts := expectation.ListOptions{
		ConnectionID: connectionID,
	}

	expectations, err := h.service.ListStructural(ctx, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list structural expectations")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"expectations": expectations,
		"count":        len(expectations),
	})
}

// GetStructural handles GET /api/v1/expectations/structural/{id}
func (h *ExpectationHandler) GetStructural(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	exp, err := h.service.GetStructuralByID(ctx, id)
	if err != nil {
		status, msg := mapExpectationError(err, id)
		writeError(w, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, exp)
}

// CreateStructural handles POST /api/v1/expectations/structural
func (h *ExpectationHandler) CreateStructural(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var input expectation.CreateStructuralInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Check for dry-run
	if isDryRun(r) {
		if err := h.service.ValidateStructuralInput(input); err != nil {
			status, msg := mapExpectationError(err, "")
			writeError(w, status, msg)
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"dry_run": true,
			"valid":   true,
		})
		return
	}

	exp, err := h.service.CreateStructural(ctx, input)
	if err != nil {
		status, msg := mapExpectationError(err, "")
		writeError(w, status, msg)
		return
	}

	writeJSON(w, http.StatusCreated, exp)
}

// DeleteStructural handles DELETE /api/v1/expectations/structural/{id}
func (h *ExpectationHandler) DeleteStructural(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	// Check for dry-run
	if isDryRun(r) {
		// Just verify the expectation exists
		_, err := h.service.GetStructuralByID(ctx, id)
		if err != nil {
			status, msg := mapExpectationError(err, id)
			writeError(w, status, msg)
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"dry_run": true,
			"deleted": id,
		})
		return
	}

	if err := h.service.DeleteStructural(ctx, id); err != nil {
		status, msg := mapExpectationError(err, id)
		writeError(w, status, msg)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListCLI handles GET /api/v1/expectations/cli
func (h *ExpectationHandler) ListCLI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	connectionID := r.URL.Query().Get("connection_id")

	opts := expectation.ListOptions{
		ConnectionID: connectionID,
	}

	assertions, err := h.service.ListCLI(ctx, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list CLI assertions")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"assertions": assertions,
		"count":      len(assertions),
	})
}

// GetCLI handles GET /api/v1/expectations/cli/{id}
func (h *ExpectationHandler) GetCLI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	assertion, err := h.service.GetCLIByID(ctx, id)
	if err != nil {
		status, msg := mapExpectationError(err, id)
		writeError(w, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, assertion)
}

// CreateCLI handles POST /api/v1/expectations/cli
func (h *ExpectationHandler) CreateCLI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var input expectation.CreateCLIInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Check for dry-run
	if isDryRun(r) {
		if err := h.service.ValidateCLIInput(input); err != nil {
			status, msg := mapExpectationError(err, "")
			writeError(w, status, msg)
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"dry_run": true,
			"valid":   true,
		})
		return
	}

	assertion, err := h.service.CreateCLI(ctx, input)
	if err != nil {
		status, msg := mapExpectationError(err, "")
		writeError(w, status, msg)
		return
	}

	writeJSON(w, http.StatusCreated, assertion)
}

// DeleteCLI handles DELETE /api/v1/expectations/cli/{id}
func (h *ExpectationHandler) DeleteCLI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	// Check for dry-run
	if isDryRun(r) {
		// Just verify the assertion exists
		_, err := h.service.GetCLIByID(ctx, id)
		if err != nil {
			status, msg := mapExpectationError(err, id)
			writeError(w, status, msg)
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"dry_run": true,
			"deleted": id,
		})
		return
	}

	if err := h.service.DeleteCLI(ctx, id); err != nil {
		status, msg := mapExpectationError(err, id)
		writeError(w, status, msg)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// mapExpectationError maps domain errors to HTTP status and message.
func mapExpectationError(err error, resourceID string) (int, string) {
	switch {
	case errors.Is(err, expectation.ErrNotFound):
		return http.StatusNotFound, "Expectation not found: " + resourceID
	case errors.Is(err, expectation.ErrInvalidConnectionID):
		return http.StatusBadRequest, "Invalid or missing connection ID"
	case errors.Is(err, expectation.ErrInvalidType):
		return http.StatusBadRequest, "Invalid expectation type (must be folder, file, or content_snippet)"
	case errors.Is(err, expectation.ErrInvalidPattern):
		return http.StatusBadRequest, "Invalid pattern (cannot be empty)"
	case errors.Is(err, expectation.ErrInvalidOperator):
		return http.StatusBadRequest, "Invalid assertion operator"
	case errors.Is(err, expectation.ErrInvalidCommand):
		return http.StatusBadRequest, "Invalid CLI command (cannot be empty)"
	case errors.Is(err, expectation.ErrInvalidJSONPath):
		return http.StatusBadRequest, "Invalid JSONPath expression"
	case errors.Is(err, expectation.ErrDangerousCommand):
		return http.StatusBadRequest, "Command contains potentially dangerous patterns"
	default:
		return http.StatusInternalServerError, "An unexpected error occurred"
	}
}
