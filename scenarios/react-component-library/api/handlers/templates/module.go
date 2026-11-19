// Package templates is the browser-facing JSON surface for the design studio's
// template shelf. Like designs it is a small REST envelope rather than a
// Connect-RPC domain: the studio client (ui/src/workbench/templates/local.ts)
// speaks plain fetches and the TemplateNode spec inside `spec` is deliberately
// opaque here.
package templates

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"react-component-library/internal/module"
	domain "react-component-library/internal/templates"
)

// maxBodyBytes bounds one JSON upload. spec is the largest legal field (1 MiB)
// plus the 8 KiB palette and the small scalar fields; the extra MiB absorbs
// whatever surrounds them without letting a malformed client fill the table.
const maxBodyBytes = domain.MaxSpecBytes + domain.MaxPaletteBytes + (1 << 20)

// Handler serves the templates collection and its members.
type Handler struct {
	repository domain.Repository
	logger     *log.Logger
}

// NewHandler wires a repository onto the HTTP surface.
func NewHandler(repository domain.Repository, logger *log.Logger) *Handler {
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{repository: repository, logger: logger}
}

// Module mounts the templates routes. The DB is the scenario's primary pool.
func Module(db *sql.DB) module.Module {
	return ModuleFromRepository(domain.NewSQLiteRepository(db), nil)
}

// ModuleFromRepository is the constructor tests and alternate stores use.
func ModuleFromRepository(repository domain.Repository, logger *log.Logger) module.Module {
	h := NewHandler(repository, logger)
	return module.Module{
		Name: "templates",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/templates", h.list).Methods(http.MethodGet)
			r.HandleFunc("/api/templates", h.create).Methods(http.MethodPost)
			r.HandleFunc("/api/templates/{id}", h.update).Methods(http.MethodPut)
			r.HandleFunc("/api/templates/{id}", h.remove).Methods(http.MethodDelete)
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internal/templates.Schema for the modules registry.
func Schema() string { return domain.Schema() }

// Migrate re-exports internal/templates.Migrate for main.go's boot sequence.
func Migrate(ctx context.Context, db *sql.DB) error { return domain.Migrate(ctx, db) }

// browserREST documents why this surface is REST rather than Connect-RPC.
func browserREST() *module.RESTException {
	return &module.RESTException{
		Reason: module.RESTReasonBrowserSurface,
		Note:   "The design studio's template shelf saves opaque TemplateNode spec JSON with plain fetches and a fire-and-forget queue; the document shape is owned by the client, so there is no generated Connect payload counterpart.",
	}
}

// Endpoints is the machine-readable templates surface consumed by
// `make endpoints`.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:            "templates_list",
		Path:          "/api/templates",
		Method:        http.MethodGet,
		Summary:       "List templates",
		Description:   "Returns the studio's template shelf newest-updated first. palette and spec are opaque JSON objects; local on the client still means authoring draft, not a published asset.",
		Category:      "templates",
		Response:      &module.Schema{Type: "object", Properties: map[string]string{"templates": "array<Template>"}},
		RESTException: browserREST(),
	},
	{
		ID:          "templates_create",
		Path:        "/api/templates",
		Method:      http.MethodPost,
		Summary:     "Create a template",
		Description: "Persists a shelf template. id is client-assigned and must match ^[a-z0-9][a-z0-9._-]{0,127}$; a duplicate id returns 409. name is required. tagline/category/palette/spec/display/radius default to '', {}, 'sans' and 14.",
		Category:    "templates",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)", "name": "string (required, 1-120 chars)", "tagline": "string (<=240 chars)", "category": "string (<=80 chars)", "palette": "JSON object (<=8 KiB)", "display": "sans|serif", "radius": "integer 0-48", "spec": "JSON object (<=1 MiB)", "slotHint": "string", "idea": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"template": "Template"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "id/name missing or out of range, display not sans|serif, radius out of range, palette/spec not JSON objects or over their byte limit, or the body exceeds 2 MiB"},
			{Status: 409, Code: "conflict", Description: "A template with that id already exists"},
		},
		RESTException: browserREST(),
	},
	{
		ID:          "templates_update",
		Path:        "/api/templates/{id}",
		Method:      http.MethodPut,
		Summary:     "Replace a template",
		Description: "Replaces every field of an existing template and bumps updatedAt. The path id is authoritative; a body id is ignored. Same validation and defaults as create.",
		Category:    "templates",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"name": "string (required, 1-120 chars)", "tagline": "string (<=240 chars)", "category": "string (<=80 chars)", "palette": "JSON object (<=8 KiB)", "display": "sans|serif", "radius": "integer 0-48", "spec": "JSON object (<=1 MiB)", "slotHint": "string", "idea": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"template": "Template"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "name missing or out of range, display not sans|serif, radius out of range, palette/spec not JSON objects or over their byte limit, or the body exceeds 2 MiB"},
			{Status: 404, Code: "not_found", Description: "No template with that id"},
		},
		RESTException: browserREST(),
	},
	{
		ID:            "templates_delete",
		Path:          "/api/templates/{id}",
		Method:        http.MethodDelete,
		Summary:       "Delete a template",
		Description:   "Hard-deletes the shelf template. Deleting a missing template is success, so a retried or cross-tab delete does not fail.",
		Category:      "templates",
		Response:      &module.Schema{Type: "object", Properties: map[string]string{}},
		RESTException: browserREST(),
	},
}

// templateRequest decodes one template body. Pointers distinguish absent from
// an explicit zero value so the documented defaults apply only when omitted.
type templateRequest struct {
	ID       *string         `json:"id"`
	Name     *string         `json:"name"`
	Tagline  *string         `json:"tagline"`
	Category *string         `json:"category"`
	Palette  json.RawMessage `json:"palette"`
	Display  *string         `json:"display"`
	Radius   *int            `json:"radius"`
	Spec     json.RawMessage `json:"spec"`
	SlotHint *string         `json:"slotHint"`
	Idea     *string         `json:"idea"`
}

// input validates required presence and flattens the body. Field-range and
// JSON-shape checks belong to the repository, which names the field.
func (body templateRequest) input(requireID bool) (domain.Input, error) {
	input := domain.Input{}
	if body.ID != nil {
		input.ID = strings.TrimSpace(*body.ID)
	}
	if requireID && input.ID == "" {
		return domain.Input{}, errors.New("id is required")
	}
	if body.Name == nil {
		return domain.Input{}, errors.New("name is required")
	}
	input.Name = *body.Name
	if body.Tagline != nil {
		input.Tagline = *body.Tagline
	}
	if body.Category != nil {
		input.Category = *body.Category
	}
	input.Palette = body.Palette
	if body.Display != nil {
		input.Display = *body.Display
	}
	input.Radius = body.Radius
	input.Spec = body.Spec
	if body.SlotHint != nil {
		input.SlotHint = *body.SlotHint
	}
	if body.Idea != nil {
		input.Idea = *body.Idea
	}
	return input, nil
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	templates, err := h.repository.List(r.Context())
	if err != nil {
		h.internalError(w, "list templates", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"templates": templates})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var body templateRequest
	if err := decodeTemplate(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	input, err := body.input(true)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	template, err := h.repository.Create(r.Context(), input)
	if err != nil {
		h.repositoryError(w, "create template", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"template": template})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var body templateRequest
	if err := decodeTemplate(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	input, err := body.input(false)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	template, err := h.repository.Update(r.Context(), mux.Vars(r)["id"], input)
	if err != nil {
		h.repositoryError(w, "update template", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"template": template})
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	if err := h.repository.Delete(r.Context(), mux.Vars(r)["id"]); err != nil {
		h.internalError(w, "delete template", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// decodeTemplate rejects unknown fields and an oversized body with a message
// that names the offending field.
func decodeTemplate(w http.ResponseWriter, r *http.Request, target *templateRequest) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return fmt.Errorf("body must be at most %d bytes", maxBodyBytes)
		}
		const unknownPrefix = "json: unknown field "
		if strings.HasPrefix(err.Error(), unknownPrefix) {
			field := strings.Trim(err.Error()[len(unknownPrefix):], `"`)
			return fmt.Errorf("unknown field %q", field)
		}
		return errors.New("request body must be a JSON template")
	}
	if decoder.More() {
		return errors.New("request body must contain exactly one JSON object")
	}
	return nil
}

func (h *Handler) repositoryError(w http.ResponseWriter, operation string, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "template not found")
	case errors.Is(err, domain.ErrConflict):
		writeError(w, http.StatusConflict, "conflict", "a template with that id already exists")
	case errors.Is(err, domain.ErrInvalid):
		writeError(w, http.StatusBadRequest, "invalid_argument", strings.TrimPrefix(err.Error(), domain.ErrInvalid.Error()+": "))
	default:
		h.internalError(w, operation, err)
	}
}

func (h *Handler) internalError(w http.ResponseWriter, operation string, err error) {
	h.logger.Printf("templates: %s: %v", operation, err)
	writeError(w, http.StatusInternalServerError, "internal", "templates storage failure")
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
