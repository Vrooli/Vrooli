package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type ResourceHandlers struct {
	db *sql.DB
}

func NewResourceHandlers(db *sql.DB) *ResourceHandlers {
	return &ResourceHandlers{db: db}
}

// RegisterRoutes mounts resource intelligence endpoints under the provided router.
func (h *ResourceHandlers) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/{resource}", h.ResourceDetail).Methods("GET")
	router.HandleFunc("/{resource}/secrets/{secret}", h.ResourceSecretUpdate).Methods("PATCH")
	router.HandleFunc("/{resource}/secrets/{secret}/strategy", h.SecretStrategy).Methods("POST")
}

func (h *ResourceHandlers) ResourceDetail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resource := vars["resource"]
	detail, err := fetchResourceDetail(r.Context(), h.db, resource)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to load resource detail: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

type resourceSecretUpdateRequest struct {
	Classification *string `json:"classification"`
	Description    *string `json:"description"`
	Required       *bool   `json:"required"`
	OwnerTeam      *string `json:"owner_team"`
	OwnerContact   *string `json:"owner_contact"`
}

func (h *ResourceHandlers) ResourceSecretUpdate(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		http.Error(w, "database not ready", http.StatusServiceUnavailable)
		return
	}
	vars := mux.Vars(r)
	resource := vars["resource"]
	secretKey := vars["secret"]
	var req resourceSecretUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	updates := []string{}
	args := []interface{}{}
	idx := 1
	if req.Classification != nil {
		value := strings.TrimSpace(*req.Classification)
		if value == "" {
			http.Error(w, "classification cannot be empty", http.StatusBadRequest)
			return
		}
		allowed := map[string]struct{}{"infrastructure": {}, "service": {}, "user": {}}
		if _, ok := allowed[value]; !ok {
			http.Error(w, "invalid classification", http.StatusBadRequest)
			return
		}
		updates = append(updates, fmt.Sprintf("classification = $%d", idx))
		args = append(args, value)
		idx++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", idx))
		args = append(args, strings.TrimSpace(*req.Description))
		idx++
	}
	if req.Required != nil {
		updates = append(updates, fmt.Sprintf("required = $%d", idx))
		args = append(args, *req.Required)
		idx++
	}
	if req.OwnerTeam != nil {
		updates = append(updates, fmt.Sprintf("owner_team = $%d", idx))
		args = append(args, strings.TrimSpace(*req.OwnerTeam))
		idx++
	}
	if req.OwnerContact != nil {
		updates = append(updates, fmt.Sprintf("owner_contact = $%d", idx))
		args = append(args, strings.TrimSpace(*req.OwnerContact))
		idx++
	}
	if len(updates) == 0 {
		http.Error(w, "no updates provided", http.StatusBadRequest)
		return
	}
	query := fmt.Sprintf("UPDATE resource_secrets SET %s, updated_at = CURRENT_TIMESTAMP WHERE resource_name = $%d AND secret_key = $%d RETURNING id", strings.Join(updates, ", "), idx, idx+1)
	args = append(args, resource, secretKey)
	var secretID string
	if err := h.db.QueryRowContext(r.Context(), query, args...).Scan(&secretID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "secret not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to update secret: %v", err), http.StatusInternalServerError)
		return
	}
	_ = secretID
	secret, err := fetchSingleSecretDetail(r.Context(), h.db, resource, secretKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch secret detail: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(secret)
}

type secretStrategyRequest struct {
	Tier              string                 `json:"tier"`
	HandlingStrategy  string                 `json:"handling_strategy"`
	FallbackStrategy  string                 `json:"fallback_strategy"`
	RequiresUserInput bool                   `json:"requires_user_input"`
	PromptLabel       string                 `json:"prompt_label"`
	PromptDescription string                 `json:"prompt_description"`
	GeneratorTemplate map[string]interface{} `json:"generator_template"`
	BundleHints       map[string]interface{} `json:"bundle_hints"`
}

func (h *ResourceHandlers) SecretStrategy(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		http.Error(w, "database not ready", http.StatusServiceUnavailable)
		return
	}
	vars := mux.Vars(r)
	resource := vars["resource"]
	secretKey := vars["secret"]
	var req secretStrategyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	tier := strings.TrimSpace(req.Tier)
	strategy := strings.TrimSpace(req.HandlingStrategy)
	if tier == "" || strategy == "" {
		http.Error(w, "tier and handling_strategy are required", http.StatusBadRequest)
		return
	}
	allowedStrategies := map[string]struct{}{"strip": {}, "generate": {}, "prompt": {}, "delegate": {}}
	if _, ok := allowedStrategies[strategy]; !ok {
		http.Error(w, "invalid handling strategy", http.StatusBadRequest)
		return
	}
	secretID, err := getResourceSecretID(r.Context(), h.db, resource, secretKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "secret not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to load secret: %v", err), http.StatusInternalServerError)
		return
	}
	generatorJSON, _ := json.Marshal(req.GeneratorTemplate)
	bundleJSON, _ := json.Marshal(req.BundleHints)
	if string(generatorJSON) == "null" {
		generatorJSON = nil
	}
	if string(bundleJSON) == "null" {
		bundleJSON = nil
	}
	_, err = h.db.ExecContext(r.Context(), `
		INSERT INTO secret_deployment_strategies (
			resource_secret_id, tier, handling_strategy, fallback_strategy,
			requires_user_input, prompt_label, prompt_description,
			generator_template, bundle_hints
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (resource_secret_id, tier)
		DO UPDATE SET
			handling_strategy = EXCLUDED.handling_strategy,
			fallback_strategy = EXCLUDED.fallback_strategy,
			requires_user_input = EXCLUDED.requires_user_input,
			prompt_label = EXCLUDED.prompt_label,
			prompt_description = EXCLUDED.prompt_description,
			generator_template = EXCLUDED.generator_template,
			bundle_hints = EXCLUDED.bundle_hints,
			updated_at = CURRENT_TIMESTAMP
	`, secretID, tier, strategy, nullString(req.FallbackStrategy), req.RequiresUserInput, req.PromptLabel, req.PromptDescription, nullBytes(generatorJSON), nullBytes(bundleJSON))
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to persist strategy: %v", err), http.StatusInternalServerError)
		return
	}
	secret, err := fetchSingleSecretDetail(r.Context(), h.db, resource, secretKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch secret detail: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(secret)
}

// NOTE: storeDiscoveredSecret has been moved to resource_queries.go
// to separate infrastructure/persistence concerns from HTTP handlers.
