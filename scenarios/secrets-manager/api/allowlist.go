package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
)

// AllowlistRule exempts matching findings from being reported. When a file's
// path matches PathPattern, findings of any type listed in ExcludedTypes (or
// any type at all when ExcludedTypes contains "*") are suppressed.
type AllowlistRule struct {
	ID            string    `json:"id"`
	PathPattern   string    `json:"path_pattern"`
	ExcludedTypes []string  `json:"excluded_types"`
	Description   string    `json:"description,omitempty"`
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
}

type allowlistUpsertRequest struct {
	PathPattern   string   `json:"path_pattern"`
	ExcludedTypes []string `json:"excluded_types"`
	Description   string   `json:"description"`
	Enabled       *bool    `json:"enabled"`
}

// AllowlistHandlers owns HTTP handlers for CRUD against scan_allowlist_rules.
type AllowlistHandlers struct {
	db     *database.RoutedDB
	logger *Logger
}

// NewAllowlistHandlers returns a configured handler set.
func NewAllowlistHandlers(db *database.RoutedDB, logger *Logger) *AllowlistHandlers {
	return &AllowlistHandlers{db: db, logger: logger}
}

// RegisterRoutes mounts allowlist endpoints under /security/allowlist-rules.
func (h *AllowlistHandlers) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/allowlist-rules", h.List).Methods(http.MethodGet)
	r.HandleFunc("/allowlist-rules", h.Create).Methods(http.MethodPost)
	r.HandleFunc("/allowlist-rules/{id}", h.Update).Methods(http.MethodPut)
	r.HandleFunc("/allowlist-rules/{id}", h.Delete).Methods(http.MethodDelete)
}

// List returns every allowlist rule.
func (h *AllowlistHandlers) List(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		http.Error(w, "database not ready", http.StatusServiceUnavailable)
		return
	}
	rules, err := loadAllowlistRules(r.Context(), h.db, false)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list allowlist rules: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"rules": rules,
		"count": len(rules),
	})
}

// Create inserts a new allowlist rule.
func (h *AllowlistHandlers) Create(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		http.Error(w, "database not ready", http.StatusServiceUnavailable)
		return
	}
	var req allowlistUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if err := validateAllowlistRule(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	id := uuid.New().String()
	now := time.Now()
	excludedTypes, err := json.Marshal(req.ExcludedTypes)
	if err != nil {
		http.Error(w, "invalid excluded_types", http.StatusBadRequest)
		return
	}
	if _, err := h.db.ExecContext(r.Context(), `
		INSERT INTO scan_allowlist_rules (id, path_pattern, excluded_types, description, enabled, created_at)
		VALUES ($1, $2, ARRAY(SELECT jsonb_array_elements_text($3::jsonb)), $4, $5, $6)
	`, id, req.PathPattern, string(excludedTypes), nullString(req.Description), enabled, now); err != nil {
		http.Error(w, fmt.Sprintf("failed to create rule: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, AllowlistRule{
		ID:            id,
		PathPattern:   req.PathPattern,
		ExcludedTypes: req.ExcludedTypes,
		Description:   req.Description,
		Enabled:       enabled,
		CreatedAt:     now,
	})
}

// Update modifies an existing allowlist rule by id.
func (h *AllowlistHandlers) Update(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		http.Error(w, "database not ready", http.StatusServiceUnavailable)
		return
	}
	id := mux.Vars(r)["id"]
	if strings.TrimSpace(id) == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	var req allowlistUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if err := validateAllowlistRule(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	excludedTypes, err := json.Marshal(req.ExcludedTypes)
	if err != nil {
		http.Error(w, "invalid excluded_types", http.StatusBadRequest)
		return
	}
	res, err := h.db.ExecContext(r.Context(), `
		UPDATE scan_allowlist_rules
		SET path_pattern = $1,
		    excluded_types = ARRAY(SELECT jsonb_array_elements_text($2::jsonb)),
		    description = $3,
		    enabled = $4
		WHERE id = $5
	`, req.PathPattern, string(excludedTypes), nullString(req.Description), enabled, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to update rule: %v", err), http.StatusInternalServerError)
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, AllowlistRule{
		ID:            id,
		PathPattern:   req.PathPattern,
		ExcludedTypes: req.ExcludedTypes,
		Description:   req.Description,
		Enabled:       enabled,
	})
}

// Delete removes an allowlist rule by id.
func (h *AllowlistHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		http.Error(w, "database not ready", http.StatusServiceUnavailable)
		return
	}
	id := mux.Vars(r)["id"]
	if strings.TrimSpace(id) == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	res, err := h.db.ExecContext(r.Context(), `DELETE FROM scan_allowlist_rules WHERE id = $1`, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to delete rule: %v", err), http.StatusInternalServerError)
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// validateAllowlistRule rejects patterns and exclusion sets that would lead to
// over-broad exemptions or non-actionable rules.
func validateAllowlistRule(req allowlistUpsertRequest) error {
	pattern := strings.TrimSpace(req.PathPattern)
	if pattern == "" {
		return errors.New("path_pattern is required")
	}
	if pattern == "**" || pattern == "*" {
		return errors.New("path_pattern too broad")
	}
	if _, err := doublestar.Match(pattern, ""); err != nil {
		return fmt.Errorf("invalid path_pattern: %w", err)
	}
	if len(req.ExcludedTypes) == 0 {
		return errors.New("excluded_types is required")
	}
	return nil
}

// loadAllowlistRules reads all rules from the DB. When enabledOnly is true,
// only rules with enabled = true are returned.
func loadAllowlistRules(ctx context.Context, database *database.RoutedDB, enabledOnly bool) ([]AllowlistRule, error) {
	if database == nil {
		return nil, nil
	}
	query := `
		SELECT id, path_pattern, array_to_json(excluded_types), description, enabled, created_at
		FROM scan_allowlist_rules
	`
	if enabledOnly {
		query += ` WHERE enabled = true`
	}
	query += ` ORDER BY path_pattern`

	rows, err := database.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := []AllowlistRule{}
	for rows.Next() {
		var rule AllowlistRule
		var description sql.NullString
		var typesJSON []byte
		if err := rows.Scan(&rule.ID, &rule.PathPattern, &typesJSON, &description, &rule.Enabled, &rule.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(typesJSON, &rule.ExcludedTypes); err != nil {
			return nil, fmt.Errorf("decode excluded types: %w", err)
		}
		rule.Description = description.String
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

// shouldExcludeFile reports whether a finding of findingType against filePath
// is suppressed by any enabled allowlist rule. A rule whose ExcludedTypes
// contains "*" suppresses every finding type.
func shouldExcludeFile(rules []AllowlistRule, filePath, findingType string) bool {
	// Normalize to forward slashes for doublestar.
	normalized := strings.ReplaceAll(filePath, "\\", "/")
	base := normalized
	if idx := strings.LastIndex(normalized, "/"); idx >= 0 {
		base = normalized[idx+1:]
	}
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if !patternMatches(rule.PathPattern, normalized, base) {
			continue
		}
		for _, t := range rule.ExcludedTypes {
			if t == "*" || t == findingType {
				return true
			}
		}
	}
	return false
}

// patternMatches checks whether a doublestar pattern applies to the full path
// or the basename. A pattern like `*_test.go` only matches the basename in
// doublestar, so we try both so rules work regardless of the caller's path
// form (absolute vs relative).
func patternMatches(pattern, normalizedPath, base string) bool {
	if ok, err := doublestar.Match(pattern, normalizedPath); err == nil && ok {
		return true
	}
	if ok, err := doublestar.Match(pattern, base); err == nil && ok {
		return true
	}
	// Handle `**/foo/**`-style patterns against paths that don't start at a
	// meaningful root. doublestar considers `**` to match any number of
	// segments including zero, so this usually works, but on some inputs we
	// fall back to a suffix-style match.
	if strings.HasPrefix(pattern, "**/") {
		trimmed := strings.TrimPrefix(pattern, "**/")
		if ok, err := doublestar.Match(trimmed, normalizedPath); err == nil && ok {
			return true
		}
	}
	return false
}
