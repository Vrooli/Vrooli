// Package handlers provides HTTP handlers for the brand-manager API.
// Handlers are thin: validate input, delegate to repositories, format output.
// DOC: docs/reference/api-endpoints.md
// DOC: docs/concepts/ARCHITECTURE.md#data-flow
package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"brand-manager/aigen"
	"brand-manager/apierr"
	"brand-manager/config"
	"brand-manager/domain"
	"brand-manager/repository"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// IDFunc generates a new unique identifier. Defaults to uuid.New().String().
// Override in tests for deterministic IDs.
type IDFunc func() string

// idempotencyEntry caches the result of a create operation keyed by Idempotency-Key.
type idempotencyEntry struct {
	status int
	body   []byte
}

// Handlers wires repository dependencies for all HTTP handlers.
type Handlers struct {
	brands      repository.BrandRepository
	versions    repository.VersionRepository
	assignments repository.AssignmentRepository
	assets      repository.AssetRepository
	newID       IDFunc
	cfg         config.Config
	idempotency sync.Map // map[string]idempotencyEntry
	scanReg     *ScannerRegistry
	chain       *aigen.Chain
}

// New creates Handlers with the given repository implementations and default config.
func New(brands repository.BrandRepository, versions repository.VersionRepository, assignments repository.AssignmentRepository) *Handlers {
	return &Handlers{
		brands:      brands,
		versions:    versions,
		assignments: assignments,
		newID:       func() string { return uuid.New().String() },
		cfg:         config.Default(),
	}
}

// WithAssets returns a copy of Handlers with an AssetRepository.
func (h *Handlers) WithAssets(assets repository.AssetRepository) *Handlers {
	cp := *h
	cp.assets = assets
	return &cp
}

// WithConfig returns a copy of Handlers using the provided config.
func (h *Handlers) WithConfig(cfg config.Config) *Handlers {
	cp := *h
	cp.cfg = cfg
	return &cp
}

// WithIDFunc returns a copy of Handlers using a custom ID generator (for testing).
func (h *Handlers) WithIDFunc(fn IDFunc) *Handlers {
	cp := *h
	cp.newID = fn
	return &cp
}

// WithAIChain returns a copy of Handlers using a specific AI provider chain (for testing).
// [REQ:BM-REQ-AI-CHAIN]
func (h *Handlers) WithAIChain(chain *aigen.Chain) *Handlers {
	cp := *h
	cp.chain = chain
	return &cp
}

// RegisterRoutes registers all brand-manager API routes on the router.
// [REQ:BM-REQ-API-BRANDS] [REQ:BM-REQ-API-VERSIONS] [REQ:BM-REQ-API-ASSIGN]
func (h *Handlers) RegisterRoutes(r *mux.Router) {
	api := r.PathPrefix("/api/v1").Subrouter()

	// Brand CRUD
	api.HandleFunc("/brands", h.ListBrands).Methods("GET")
	api.HandleFunc("/brands", h.CreateBrand).Methods("POST")
	api.HandleFunc("/brands/{id}", h.GetBrand).Methods("GET")
	api.HandleFunc("/brands/{id}", h.UpdateBrand).Methods("PUT")
	api.HandleFunc("/brands/{id}", h.DeleteBrand).Methods("DELETE")

	// Brand versions
	api.HandleFunc("/brands/{id}/versions", h.ListVersions).Methods("GET")

	// Assignments
	api.HandleFunc("/assignments", h.CreateAssignment).Methods("POST")
	api.HandleFunc("/assignments/{id}", h.DeleteAssignment).Methods("DELETE")
	api.HandleFunc("/scenarios/{name}/status", h.GetScenarioStatus).Methods("GET")

	// Standards / auditor integration [REQ:BM-REQ-API-STANDARDS] [REQ:BM-REQ-AUDIT-ENDPOINT]
	api.HandleFunc("/standards", h.GetStandards).Methods("GET")

	// Assets [REQ:BM-REQ-API-ASSETS]
	api.HandleFunc("/brands/{id}/assets", h.UploadAsset).Methods("POST")
	api.HandleFunc("/brands/{id}/assets", h.ListAssets).Methods("GET")
	api.HandleFunc("/assets/{id}", h.GetAsset).Methods("GET")
	api.HandleFunc("/assets/{id}", h.DeleteAsset).Methods("DELETE")
	api.HandleFunc("/assets/{id}/file", h.ServeAssetFile).Methods("GET")

	// Inline validation scanner [REQ:BM-REQ-SCAN-CSS] [REQ:BM-REQ-SCAN-JSON]
	api.HandleFunc("/scan/{scenario}", h.ScanScenario).Methods("GET")

	// Design language generation [REQ:BM-REQ-DESIGN-GEN]
	api.HandleFunc("/brands/{id}/design-language", h.GenerateDesignLanguage).Methods("POST")

	// Programmatic application [REQ:BM-REQ-APPLY-CSS] [REQ:BM-REQ-APPLY-JSON] [REQ:BM-REQ-APPLY-ASSETS] [REQ:BM-REQ-APPLY-PARTIAL]
	api.HandleFunc("/brands/{id}/apply", h.ApplyBrand).Methods("POST")

	// Discovery scanner [REQ:BM-REQ-DISC-SCAN] [REQ:BM-REQ-DISC-IMPORT] [REQ:BM-REQ-DISC-LPBS]
	api.HandleFunc("/discover/{scenario}", h.DiscoverScenario).Methods("GET")
	api.HandleFunc("/discover/{scenario}/import", h.ImportDiscovery).Methods("POST")

	// Audit provider [REQ:BM-REQ-AUDIT-PROVIDER]
	api.HandleFunc("/audit/rules", h.GetAuditRules).Methods("GET")
	api.HandleFunc("/audit/evaluate/{scenario}", h.EvaluateScenario).Methods("POST")

	// WCAG contrast validation [REQ:BM-REQ-WCAG-CALC] [REQ:BM-REQ-WCAG-VALIDATE]
	api.HandleFunc("/contrast/check", h.CheckContrast).Methods("POST")
	api.HandleFunc("/contrast/brand", h.CheckBrandContrast).Methods("POST")

	// AI Generation [REQ:BM-REQ-AI-CHAIN] [REQ:BM-REQ-AI-TEXT] [REQ:BM-REQ-AI-IMAGE]
	api.HandleFunc("/brands/{id}/generate", h.GenerateBrandElements).Methods("POST")
	api.HandleFunc("/brands/{id}/generate/image", h.GenerateBrandImage).Methods("POST")

	// Agent-assisted application [REQ:BM-REQ-AGENT-SPAWN] [REQ:BM-REQ-AGENT-INSTRUCT] [REQ:BM-REQ-AGENT-VALIDATE]
	api.HandleFunc("/brands/{id}/agent-apply", h.AgentApply).Methods("POST")
	api.HandleFunc("/brands/{id}/agent-validate", h.AgentValidate).Methods("POST")

	// Lighthouse WCAG audit [REQ:BM-REQ-LIGHTHOUSE]
	api.HandleFunc("/brands/{id}/lighthouse", h.LighthouseAudit).Methods("POST")

	// Extended plugin-based routes [REQ:BM-REQ-SCAN-PLUGINS] [REQ:BM-REQ-UI-APPLY] [REQ:BM-REQ-UI-THEME] [REQ:BM-REQ-UI-GENERATE]
	h.RegisterExtendedRoutes(r)
}

// CreateBrand handles POST /api/v1/brands. [REQ:BM-REQ-CRUD-CREATE] [REQ:BM-REQ-API-BRANDS]
//
// Supports Idempotency-Key header: if the same key is sent again, the cached
// response from the first successful create is returned without re-executing.
// This prevents duplicate brands on retried requests.
func (h *Handlers) CreateBrand(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := r.Header.Get("Idempotency-Key")

	// Return cached response for duplicate requests
	if idempotencyKey != "" {
		if cached, ok := h.idempotency.Load(idempotencyKey); ok {
			entry := cached.(idempotencyEntry)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Idempotent-Replayed", "true")
			w.WriteHeader(entry.status)
			w.Write(entry.body)
			return
		}
	}

	var brand domain.Brand
	if err := json.NewDecoder(r.Body).Decode(&brand); err != nil {
		apierr.Write(w, apierr.Validation("invalid request body"))
		return
	}

	if brand.Name == "" {
		apierr.Write(w, apierr.Validation("name is required"))
		return
	}

	brand.ID = h.newID()

	if isDryRun(r) {
		brand.Version = 1
		writeJSON(w, http.StatusOK, dryRunResponse(brand))
		return
	}

	if err := h.brands.Create(r.Context(), &brand); err != nil {
		apierr.Write(w, apierr.Internal("create brand", err))
		return
	}

	// Create initial version snapshot [REQ:BM-REQ-CRUD-VERSION]
	h.snapshotVersion(r.Context(), &brand)

	// Cache response for idempotency replay
	if idempotencyKey != "" {
		body, _ := json.Marshal(brand)
		h.idempotency.Store(idempotencyKey, idempotencyEntry{
			status: http.StatusCreated,
			body:   body,
		})
	}

	writeBrandJSON(w, http.StatusCreated, &brand)
}

// GetBrand handles GET /api/v1/brands/{id}. [REQ:BM-REQ-CRUD-READ] [REQ:BM-REQ-API-BRANDS]
func (h *Handlers) GetBrand(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	brand, done := getOrNotFound(w, func() (*domain.Brand, error) {
		return h.brands.GetByID(r.Context(), id)
	}, "brand")
	if done {
		return
	}
	writeBrandJSON(w, http.StatusOK, brand)
}

// ListBrands handles GET /api/v1/brands. [REQ:BM-REQ-CRUD-READ] [REQ:BM-REQ-API-BRANDS]
func (h *Handlers) ListBrands(w http.ResponseWriter, r *http.Request) {
	filter := domain.BrandFilter{
		NameContains: r.URL.Query().Get("name"),
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		filter.Limit, _ = strconv.Atoi(v)
	}
	filter.Limit = h.cfg.ClampLimit(filter.Limit)
	if v := r.URL.Query().Get("offset"); v != "" {
		filter.Offset, _ = strconv.Atoi(v)
	}

	brands, err := h.brands.List(r.Context(), filter)
	if err != nil {
		apierr.Write(w, apierr.Internal("list brands", err))
		return
	}

	if brands == nil {
		brands = []*domain.Brand{}
	}
	writeJSON(w, http.StatusOK, brands)
}

// UpdateBrand handles PUT /api/v1/brands/{id}. [REQ:BM-REQ-CRUD-UPDATE] [REQ:BM-REQ-API-BRANDS]
func (h *Handlers) UpdateBrand(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	existing, done := getOrNotFound(w, func() (*domain.Brand, error) {
		return h.brands.GetByID(r.Context(), id)
	}, "brand")
	if done {
		return
	}

	var update domain.Brand
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		apierr.Write(w, apierr.Validation("invalid request body"))
		return
	}

	// Optimistic locking: If-Match header carries the expected version.
	// If the brand has been modified since the client last read it, reject the update
	// with 409 Conflict so the caller can re-read and retry.
	if ifMatch := r.Header.Get("If-Match"); ifMatch != "" {
		expectedVersion, parseErr := strconv.Atoi(ifMatch)
		if parseErr != nil {
			apierr.Write(w, apierr.Validation("If-Match header must be an integer version"))
			return
		}
		if existing.Version != expectedVersion {
			apierr.Write(w, apierr.Conflict("brand has been modified (current version: "+strconv.Itoa(existing.Version)+")"))
			return
		}
	}

	existing.ApplyPartialUpdate(update)

	if isDryRun(r) {
		existing.Version++
		writeJSON(w, http.StatusOK, dryRunResponse(existing))
		return
	}

	if err := h.brands.Update(r.Context(), existing); err != nil {
		apierr.Write(w, apierr.Internal("update brand", err))
		return
	}

	// Create version snapshot [REQ:BM-REQ-CRUD-VERSION]
	h.snapshotVersion(r.Context(), existing)

	writeBrandJSON(w, http.StatusOK, existing)
}

// DeleteBrand handles DELETE /api/v1/brands/{id}. [REQ:BM-REQ-API-BRANDS]
// Idempotent: returns 204 whether the brand existed or was already deleted.
// Dry-run mode still validates existence (returns 404 if missing).
func (h *Handlers) DeleteBrand(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	if isDryRun(r) {
		// Dry-run validates existence — 404 if missing, so the caller
		// can distinguish "would succeed" from "target doesn't exist".
		if _, done := getOrNotFound(w, func() (*domain.Brand, error) {
			return h.brands.GetByID(r.Context(), id)
		}, "brand"); done {
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"dry_run": true,
			"success": true,
			"deleted": id,
		})
		return
	}

	// Real delete — idempotent: already-deleted is treated as success.
	err := h.brands.Delete(r.Context(), id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		apierr.Write(w, apierr.Internal("delete brand", err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListVersions handles GET /api/v1/brands/{id}/versions. [REQ:BM-REQ-CRUD-VERSION] [REQ:BM-REQ-API-VERSIONS]
func (h *Handlers) ListVersions(w http.ResponseWriter, r *http.Request) {
	brandID := mux.Vars(r)["id"]
	versions, err := h.versions.ListByBrandID(r.Context(), brandID)
	if err != nil {
		apierr.Write(w, apierr.Internal("list versions", err))
		return
	}
	if versions == nil {
		versions = []*domain.BrandVersion{}
	}
	writeJSON(w, http.StatusOK, versions)
}

// CreateAssignment handles POST /api/v1/assignments. [REQ:BM-REQ-ASSIGN-LINK] [REQ:BM-REQ-API-ASSIGN]
func (h *Handlers) CreateAssignment(w http.ResponseWriter, r *http.Request) {
	var a domain.Assignment
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		apierr.Write(w, apierr.Validation("invalid request body"))
		return
	}
	if a.BrandID == "" || a.ScenarioName == "" {
		apierr.Write(w, apierr.Validation("brand_id and scenario_name are required"))
		return
	}

	a.ID = h.newID()

	// Verify brand exists and get current version
	brand, done := getOrNotFound(w, func() (*domain.Brand, error) {
		return h.brands.GetByID(r.Context(), a.BrandID)
	}, "brand")
	if done {
		return
	}
	a.BrandVersion = brand.Version

	if isDryRun(r) {
		writeJSON(w, http.StatusOK, dryRunResponse(a))
		return
	}

	if err := h.assignments.Create(r.Context(), &a); err != nil {
		apierr.Write(w, apierr.Internal("create assignment", err))
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

// DeleteAssignment handles DELETE /api/v1/assignments/{id}. [REQ:BM-REQ-API-ASSIGN]
// Idempotent: returns 204 whether the assignment existed or was already deleted.
func (h *Handlers) DeleteAssignment(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	if isDryRun(r) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"dry_run": true,
			"success": true,
			"deleted": id,
		})
		return
	}

	// Idempotent: already-deleted is treated as success.
	err := h.assignments.Delete(r.Context(), id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		apierr.Write(w, apierr.Internal("delete assignment", err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetScenarioStatus handles GET /api/v1/scenarios/{name}/status. [REQ:BM-REQ-API-STATUS]
func (h *Handlers) GetScenarioStatus(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	assignment, err := h.assignments.GetByScenario(r.Context(), name)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusOK, domain.ScenarioStatusUnassigned(name))
		return
	}
	if err != nil {
		apierr.Write(w, apierr.Internal("get scenario status", err))
		return
	}
	writeJSON(w, http.StatusOK, domain.ScenarioStatusFromAssignment(name, assignment))
}

// snapshotVersion creates a version snapshot for the given brand.
// Logs a warning on failure — version history is degraded but the brand itself is usable.
// [REQ:BM-REQ-CRUD-VERSION]
func (h *Handlers) snapshotVersion(ctx context.Context, brand *domain.Brand) {
	snapshot, _ := json.Marshal(brand)
	version := &domain.BrandVersion{
		ID:       h.newID(),
		BrandID:  brand.ID,
		Version:  brand.Version,
		Snapshot: string(snapshot),
	}
	if err := h.versions.Create(ctx, version); err != nil {
		log.Printf("[warn] brand %s v%d: version snapshot failed: %v", brand.ID, brand.Version, err)
	}
}

// resolveScenarioDir validates that a scenario directory exists and returns its
// absolute path. Writes a 404 error and returns ("", true) if the directory does not exist.
func (h *Handlers) resolveScenarioDir(w http.ResponseWriter, scenario string) (string, bool) {
	dir := filepath.Join(h.cfg.ScenariosDir, scenario)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		apierr.Write(w, apierr.NotFound("scenario directory"))
		return "", true
	}
	return dir, false
}

// getOrNotFound fetches a resource and writes the appropriate error response
// if the fetch fails. Returns (nil, true) when an error was written to the
// response, so the caller can return early.
func getOrNotFound[T any](w http.ResponseWriter, fetch func() (T, error), resource string) (T, bool) {
	v, err := fetch()
	if errors.Is(err, sql.ErrNoRows) {
		apierr.Write(w, apierr.NotFound(resource))
		return v, true
	}
	if err != nil {
		apierr.Write(w, apierr.Internal("get "+resource, err))
		return v, true
	}
	return v, false
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeBrandJSON writes a brand response with an ETag header for optimistic locking.
func writeBrandJSON(w http.ResponseWriter, status int, brand *domain.Brand) {
	w.Header().Set("ETag", strconv.Itoa(brand.Version))
	writeJSON(w, status, brand)
}

// isDryRun reports whether the request has the X-Dry-Run header set.
// Compatible with cli-core's --dry-run global flag which sets this header automatically.
func isDryRun(r *http.Request) bool {
	return r.Header.Get("X-Dry-Run") == "true"
}

// dryRunResponse wraps any value with a dry_run marker.
func dryRunResponse(v interface{}) map[string]interface{} {
	data, _ := json.Marshal(v)
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	if m == nil {
		m = make(map[string]interface{})
	}
	m["dry_run"] = true
	m["success"] = true
	return m
}
