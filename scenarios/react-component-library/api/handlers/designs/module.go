// Package designs is the browser-facing JSON surface for the studio's durable
// design documents and their design-scoped assets. It is not a Connect-RPC
// domain: the studio client (ui/src/workbench/designs.ts) speaks a small REST
// envelope, and the TemplateNode spec inside frames is deliberately opaque
// here.
package designs

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"react-component-library/internal/designinference"
	domain "react-component-library/internal/designs"
	"react-component-library/internal/module"
)

// maxDocumentBytes bounds one saved document. A design is a handful of frames
// of spec JSON; the ceiling stops a malformed client from filling the table.
const maxDocumentBytes = 8 << 20

// maxAssetRequestBytes bounds one JSON upload: the base64 envelope is ~4/3 of
// the decoded bytes, plus the small metadata fields.
const maxAssetRequestBytes = (domain.MaxAssetBytes/3+1)*4 + (1 << 20)

// Handler serves the designs collection, its members and their assets.
type Handler struct {
	repository domain.Repository
	logger     *log.Logger
	assistant  Assistant
}

// NewHandler wires a repository onto the HTTP surface.
func NewHandler(repository domain.Repository, logger *log.Logger) *Handler {
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{repository: repository, logger: logger}
}

// WithAssistant attaches the durable inference runner used by /assist.
func (h *Handler) WithAssistant(assistant Assistant) *Handler {
	h.assistant = assistant
	return h
}

// Module mounts the designs routes. The DB is the scenario's primary pool.
// The assistant reuses the same pool for its idempotency record and resolves
// the AI gateway lazily on the first proposal.
func Module(db *sql.DB) module.Module {
	h := NewHandler(domain.NewSQLiteRepository(db), nil).
		WithAssistant(designinference.NewService(designinference.NewSQLiteRepository(db)))
	return moduleFromHandler(h)
}

// ModuleWithAssistant lets tests inject a runner without a gateway.
func ModuleWithAssistant(repository domain.Repository, logger *log.Logger, assistant Assistant) module.Module {
	return moduleFromHandler(NewHandler(repository, logger).WithAssistant(assistant))
}

// ModuleFromRepository is the constructor alternate stores use.
func ModuleFromRepository(repository domain.Repository, logger *log.Logger) module.Module {
	return moduleFromHandler(NewHandler(repository, logger))
}

func moduleFromHandler(h *Handler) module.Module {
	return module.Module{
		Name: "designs",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/designs", h.list).Methods(http.MethodGet)
			r.HandleFunc("/api/designs", h.create).Methods(http.MethodPost)
			r.HandleFunc("/api/designs/{id}", h.get).Methods(http.MethodGet)
			r.HandleFunc("/api/designs/{id}", h.save).Methods(http.MethodPut)
			r.HandleFunc("/api/designs/{id}", h.remove).Methods(http.MethodDelete)
			r.HandleFunc("/api/designs/{id}/assist", h.assist).Methods(http.MethodPost)
			r.HandleFunc("/api/designs/{id}/assets", h.listAssets).Methods(http.MethodGet)
			r.HandleFunc("/api/designs/{id}/assets", h.createAsset).Methods(http.MethodPost)
			r.HandleFunc("/api/designs/{id}/assets/{assetId}", h.getAssetData).Methods(http.MethodGet)
			r.HandleFunc("/api/designs/{id}/assets/{assetId}", h.removeAsset).Methods(http.MethodDelete)
			r.HandleFunc("/api/design-assets", h.listAllAssets).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internal/designs.Schema for the modules registry.
func Schema() string { return domain.Schema() }

// Migrate re-exports internal/designs.Migrate for main.go's boot sequence.
func Migrate(ctx context.Context, db *sql.DB) error { return domain.Migrate(ctx, db) }

// browserREST documents why this surface is REST rather than Connect-RPC.
func browserREST() *module.RESTException {
	return &module.RESTException{
		Reason: module.RESTReasonBrowserSurface,
		Note:   "The design studio edits opaque TemplateNode spec JSON with plain fetches and autosave; the document shape is owned by the client, so there is no generated Connect payload counterpart.",
	}
}

// binaryREST documents the byte-serving asset route.
func binaryREST() *module.RESTException {
	return &module.RESTException{
		Reason: module.RESTReasonBrowserSurface,
		Note:   "The studio renders uploaded reference images directly in <img>/CSS; the byte route is browser-native and sets Content-Type, Cache-Control and ETag rather than returning a JSON envelope.",
	}
}

// Endpoints is the machine-readable designs surface consumed by
// `make endpoints`.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:            "designs_list",
		Path:          "/api/designs",
		Method:        http.MethodGet,
		Summary:       "List designs",
		Description:   "Returns design summaries newest-updated first, including frame counts and the optional scenario/page binding. Optional scenario and page query params filter by exact binding.",
		Category:      "designs",
		Response:      &module.Schema{Type: "object", Properties: map[string]string{"designs": "array<DesignSummary>"}},
		RESTException: browserREST(),
	},
	{
		ID:            "designs_create",
		Path:          "/api/designs",
		Method:        http.MethodPost,
		Summary:       "Create a design",
		Description:   "Persists a design at revision 1. The server assigns id, revision and updatedAt. name, frames and theme are required; frames:null means [] and theme:null means {}. options is optional design chrome: {radius?: number 0-48, display?: \"sans\"|\"serif\"}; absent or null means {}.",
		Category:      "designs",
		Request:       &module.Schema{Type: "object", Properties: map[string]string{"name": "string (required)", "frames": "array<Frame> (required; null means [])", "theme": "Theme (required; null means {})", "options": "{radius?: number 0-48, display?: \"sans\"|\"serif\"} (optional; null means {})", "scenario": "string|null", "page": "string|null"}},
		Response:      &module.Schema{Type: "object", Properties: map[string]string{"design": "DesignDocument"}},
		Errors:        []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "name, frames or theme is missing, frames is not an array of objects, theme is not an object, or options is not an object with a valid radius/display"}},
		RESTException: browserREST(),
	},
	{
		ID:            "designs_get",
		Path:          "/api/designs/{id}",
		Method:        http.MethodGet,
		Summary:       "Get a design",
		Description:   "Returns the full design document, including opaque frame specs, theme, validated chrome options (always an object) and the optional scenario/page binding.",
		Category:      "designs",
		Response:      &module.Schema{Type: "object", Properties: map[string]string{"design": "DesignDocument"}},
		Errors:        []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No design with that id"}},
		RESTException: browserREST(),
	},
	{
		ID:          "designs_save",
		Path:        "/api/designs/{id}",
		Method:      http.MethodPut,
		Summary:     "Save a design",
		Description: "Replaces the design document and bumps revision and updatedAt. name, frames and theme are required. options replaces the stored chrome and absent/null means {} (old clients keep working). Sending revision makes the save conditional: a stale revision returns 409 with the current document and writes nothing; omitting revision is last-write-wins.",
		Category:    "designs",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"name": "string (required)", "frames": "array<Frame> (required; null means [])", "theme": "Theme (required; null means {})", "options": "{radius?: number 0-48, display?: \"sans\"|\"serif\"} (optional; null or absent means {})", "scenario": "string|null", "page": "string|null", "revision": "integer (optional; enables optimistic concurrency)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"design": "DesignDocument"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "name, frames or theme is missing, frames is not an array of objects, theme is not an object, or options is not an object with a valid radius/display"},
			{Status: 404, Code: "not_found", Description: "No design with that id"},
			{Status: 409, Code: "conflict", Description: "revision did not match the stored revision; response also carries {design:DesignDocument}"},
		},
		RESTException: browserREST(),
	},
	{
		ID:            "designs_delete",
		Path:          "/api/designs/{id}",
		Method:        http.MethodDelete,
		Summary:       "Delete a design",
		Description:   "Hard-deletes the design and its assets in one transaction. Deleting a missing design is success.",
		Category:      "designs",
		Response:      &module.Schema{Type: "object", Properties: map[string]string{}},
		RESTException: browserREST(),
	},
	{
		ID:          "designs_assist",
		Path:        "/api/designs/{id}/assist",
		Method:      http.MethodPost,
		Summary:     "Propose a design patch",
		Description: "Sends the scoped design context through the durable design-inference service to the AI gateway and returns a validated, inert patch. Nothing is written: the studio applies the patch as one undoable command or discards it. The current revision is required so a proposal cannot be made against a stale document; a stale revision returns 409 with the current design. 422 means the model produced a patch this document cannot apply.",
		Category:    "designs",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"prompt": "string (required, 1-2000 chars)", "scope": "{kind: design|frame|layer, frameId?, nodeId?}", "revision": "integer (required)", "idempotencyKey": "string (required, 8-200 chars)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"operation": "{id,state,provider?,model?,detail?,inputTokens?,outputTokens?}", "revision": "integer", "proposal": "{summary,operations[]}"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Malformed prompt, scope, revision or idempotency key"},
			{Status: 404, Code: "not_found", Description: "No design with that id"},
			{Status: 409, Code: "conflict", Description: "revision did not match the stored revision; response also carries {design:DesignDocument}"},
			{Status: 422, Code: "invalid_proposal", Description: "The model returned a patch that is not applicable to this document"},
			{Status: 502, Code: "ai_failed", Description: "The gateway refused or failed the request"},
			{Status: 503, Code: "ai_unavailable", Description: "No assistant is configured or the gateway could not be reached"},
		},
		RESTException: browserREST(),
	},
	{
		ID:            "designs_assets_list",
		Path:          "/api/designs/{id}/assets",
		Method:        http.MethodGet,
		Summary:       "List a design's assets",
		Description:   "Returns upload metadata for one design, oldest first. Bytes are served by the per-asset route.",
		Category:      "designs",
		Response:      &module.Schema{Type: "object", Properties: map[string]string{"assets": "array<AssetMeta>"}},
		Errors:        []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No design with that id"}},
		RESTException: browserREST(),
	},
	{
		ID:          "designs_assets_create",
		Path:        "/api/designs/{id}/assets",
		Method:      http.MethodPost,
		Summary:     "Upload a design asset",
		Description: "Accepts a base64 image (png/jpeg/webp/gif/svg+xml, <=8 MiB decoded), stores the bytes in SQLite and returns the metadata with a byte-route url. Width/height are read from the image header; SVG and unknown formats report 0.",
		Category:    "designs",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"name": "string (required)", "mime": "string (required image/*)", "dataBase64": "string (required)", "fidelity": "reference|captured (optional, default reference)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"asset": "AssetMeta"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Empty name, non-image mime, invalid base64, decoded bytes over 8 MiB, or bad fidelity"},
			{Status: 404, Code: "not_found", Description: "No design with that id"},
		},
		RESTException: browserREST(),
	},
	{
		ID:            "designs_assets_get",
		Path:          "/api/designs/{id}/assets/{assetId}",
		Method:        http.MethodGet,
		Summary:       "Read design asset bytes",
		Description:   "Serves the raw image with its stored Content-Type, Cache-Control: private, max-age=3600 and an ETag equal to the asset id. If-None-Match returns 304.",
		Category:      "designs",
		Response:      &module.Schema{Type: "binary", Properties: map[string]string{"body": "raw image bytes"}},
		Errors:        []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No asset with that id for that design"}},
		RESTException: binaryREST(),
	},
	{
		ID:            "designs_assets_delete",
		Path:          "/api/designs/{id}/assets/{assetId}",
		Method:        http.MethodDelete,
		Summary:       "Delete a design asset",
		Description:   "Deletes the asset scoped to that design. Idempotent; an asset belonging to another design is untouched.",
		Category:      "designs",
		Response:      &module.Schema{Type: "object", Properties: map[string]string{}},
		RESTException: browserREST(),
	},
	{
		ID:            "designs_assets_list_all",
		Path:          "/api/design-assets",
		Method:        http.MethodGet,
		Summary:       "List every design asset",
		Description:   "Returns metadata for every upload across designs, newest first, for the studio's 'view uploads from other designs' surface.",
		Category:      "designs",
		Response:      &module.Schema{Type: "object", Properties: map[string]string{"assets": "array<AssetMeta>"}},
		RESTException: browserREST(),
	},
}

type documentRequest struct {
	Name     *string         `json:"name"`
	Frames   json.RawMessage `json:"frames"`
	Theme    json.RawMessage `json:"theme"`
	Options  json.RawMessage `json:"options"`
	Scenario *string         `json:"scenario"`
	Page     *string         `json:"page"`
	Revision *int            `json:"revision"`
}

// createInput checks presence separately from explicit null: a missing name,
// frames or theme is a 400, while frames:null/theme:null keep their long-
// standing "empty document" meaning. options is optional and absent/null means
// {}; a malformed options object is rejected by the domain validator.
func (body documentRequest) createInput() (domain.CreateInput, error) {
	if body.Name == nil {
		return domain.CreateInput{}, errors.New("name is required")
	}
	if body.Frames == nil {
		return domain.CreateInput{}, errors.New("frames is required")
	}
	if body.Theme == nil {
		return domain.CreateInput{}, errors.New("theme is required")
	}
	return domain.CreateInput{
		Name:     *body.Name,
		Frames:   body.Frames,
		Theme:    body.Theme,
		Options:  body.Options,
		Scenario: body.Scenario,
		Page:     body.Page,
	}, nil
}

func (body documentRequest) updateInput() (domain.UpdateInput, error) {
	create, err := body.createInput()
	if err != nil {
		return domain.UpdateInput{}, err
	}
	return domain.UpdateInput{
		Name:     create.Name,
		Frames:   create.Frames,
		Theme:    create.Theme,
		Options:  create.Options,
		Scenario: create.Scenario,
		Page:     create.Page,
		Revision: body.Revision,
	}, nil
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	summaries, err := h.repository.List(r.Context(), domain.ListFilter{
		Scenario: strings.TrimSpace(query.Get("scenario")),
		Page:     strings.TrimSpace(query.Get("page")),
	})
	if err != nil {
		h.internalError(w, "list designs", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"designs": summaries})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var body documentRequest
	if err := decodeDocument(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	input, err := body.createInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	doc, err := h.repository.Create(r.Context(), input)
	if err != nil {
		h.repositoryError(w, "create design", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"design": doc})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	doc, err := h.repository.Get(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		h.repositoryError(w, "get design", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"design": doc})
}

func (h *Handler) save(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var body documentRequest
	if err := decodeDocument(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	input, err := body.updateInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	doc, err := h.repository.Save(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			current, getErr := h.repository.Get(r.Context(), id)
			if getErr != nil {
				h.repositoryError(w, "read conflicting design", getErr)
				return
			}
			writeJSON(w, http.StatusConflict, map[string]any{
				"error":  map[string]string{"code": "conflict", "message": "design was modified elsewhere"},
				"design": current,
			})
			return
		}
		h.repositoryError(w, "save design", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"design": doc})
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	if err := h.repository.Delete(r.Context(), mux.Vars(r)["id"]); err != nil {
		h.repositoryError(w, "delete design", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- assets ---------------------------------------------------------------

type assetRequest struct {
	Name       string `json:"name"`
	Mime       string `json:"mime"`
	DataBase64 string `json:"dataBase64"`
	Fidelity   string `json:"fidelity"`
}

func (h *Handler) listAssets(w http.ResponseWriter, r *http.Request) {
	assets, err := h.repository.ListAssets(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		h.repositoryError(w, "list design assets", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"assets": assets})
}

func (h *Handler) listAllAssets(w http.ResponseWriter, r *http.Request) {
	assets, err := h.repository.ListAllAssets(r.Context())
	if err != nil {
		h.internalError(w, "list all design assets", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"assets": assets})
}

func (h *Handler) createAsset(w http.ResponseWriter, r *http.Request) {
	var body assetRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxAssetRequestBytes))
	if err := decoder.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", "request body must be a JSON asset upload")
		return
	}
	data, err := decodeBase64(body.DataBase64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	asset, err := h.repository.CreateAsset(r.Context(), domain.AssetInput{
		DesignID: mux.Vars(r)["id"],
		Name:     body.Name,
		Mime:     body.Mime,
		Data:     data,
		Fidelity: body.Fidelity,
	})
	if err != nil {
		h.repositoryError(w, "create design asset", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"asset": asset})
}

func (h *Handler) getAssetData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	asset, data, err := h.repository.GetAsset(r.Context(), vars["id"], vars["assetId"])
	if err != nil {
		h.repositoryError(w, "get design asset", err)
		return
	}
	etag := asset.ID
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	if etagMatches(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", asset.Mime)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *Handler) removeAsset(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := h.repository.DeleteAsset(r.Context(), vars["id"], vars["assetId"]); err != nil {
		h.repositoryError(w, "delete design asset", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeBase64(raw string) ([]byte, error) {
	value := strings.TrimSpace(raw)
	if strings.HasPrefix(value, "data:") {
		if comma := strings.Index(value, ","); comma >= 0 {
			value = value[comma+1:]
		}
	}
	if value == "" {
		return nil, errors.New("dataBase64 is required")
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, errors.New("dataBase64 is not valid base64")
	}
	return decoded, nil
}

// etagMatches accepts the exact id or a quoted/weak form, as browsers resend
// whatever was stored. `*` matches any existing asset.
func etagMatches(header, etag string) bool {
	for _, part := range strings.Split(header, ",") {
		candidate := strings.TrimSpace(part)
		candidate = strings.TrimPrefix(candidate, "W/")
		candidate = strings.Trim(candidate, `"`)
		if candidate == "*" || candidate == etag {
			return true
		}
	}
	return false
}

// --- errors ---------------------------------------------------------------

func (h *Handler) repositoryError(w http.ResponseWriter, operation string, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "design not found")
	case errors.Is(err, domain.ErrConflict):
		writeError(w, http.StatusConflict, "conflict", "design was modified elsewhere")
	case errors.Is(err, domain.ErrInvalid):
		writeError(w, http.StatusBadRequest, "invalid_argument", strings.TrimPrefix(err.Error(), domain.ErrInvalid.Error()+": "))
	default:
		h.internalError(w, operation, err)
	}
}

func (h *Handler) internalError(w http.ResponseWriter, operation string, err error) {
	h.logger.Printf("designs: %s: %v", operation, err)
	writeError(w, http.StatusInternalServerError, "internal", "designs storage failure")
}

func decodeDocument(w http.ResponseWriter, r *http.Request, target *documentRequest) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxDocumentBytes))
	// The client sends the whole DesignDocument on save; extra keys such as
	// id and updatedAt are intentionally ignored rather than rejected so the
	// server stays the authority for those fields.
	if err := decoder.Decode(target); err != nil {
		return errors.New("request body must be a JSON design document")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
