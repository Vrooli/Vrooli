package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

// CatalogEntry represents a scenario or resource with its PRD status
type CatalogEntry struct {
	Type        string `json:"type"`        // "scenario" or "resource"
	Name        string `json:"name"`        // Entity name
	HasPRD      bool   `json:"has_prd"`     // Whether PRD.md exists
	PRDPath     string `json:"prd_path"`    // Absolute path to PRD.md
	HasDraft    bool   `json:"has_draft"`   // Whether a draft exists
	Description string `json:"description"` // Brief description (if available)
}

// CatalogResponse represents the catalog API response
type CatalogResponse struct {
	Entries []CatalogEntry `json:"entries"`
	Total   int            `json:"total"`
}

// PublishedPRDResponse represents a published PRD content
type PublishedPRDResponse struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Content     string `json:"content"`
	Path        string `json:"path"`
	ContentHTML string `json:"content_html"`
}

type rowScanner interface {
	Scan(dest ...any) error
}

type draftStore interface {
	QueryRow(query string, args ...any) rowScanner
	Exec(query string, args ...any) (sql.Result, error)
}

type sqlDraftStore struct {
	db *sql.DB
}

func (s sqlDraftStore) QueryRow(query string, args ...any) rowScanner {
	return s.db.QueryRow(query, args...)
}

func (s sqlDraftStore) Exec(query string, args ...any) (sql.Result, error) {
	return s.db.Exec(query, args...)
}

var markdownRenderer = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
	),
	goldmark.WithRendererOptions(
		gmhtml.WithHardWraps(),
		gmhtml.WithXHTML(),
	),
)

func markdownToHTML(markdown []byte) (string, error) {
	var buf bytes.Buffer
	if err := markdownRenderer.Convert(markdown, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func handleGetCatalog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	entries := []CatalogEntry{}

	// Enumerate scenarios
	scenariosDir := filepath.Join(vrooliRoot, "scenarios")
	scenarios, err := enumerateEntities(scenariosDir, "scenario")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to enumerate scenarios: %v", err), http.StatusInternalServerError)
		return
	}
	entries = append(entries, scenarios...)

	// Enumerate resources
	resourcesDir := filepath.Join(vrooliRoot, "resources")
	resources, err := enumerateEntities(resourcesDir, "resource")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to enumerate resources: %v", err), http.StatusInternalServerError)
		return
	}
	entries = append(entries, resources...)

	// Check for drafts
	draftPresence, err := loadDraftPresence()
	if err != nil {
		slog.Warn("failed to load draft presence from database", "error", err)
	}
	for i := range entries {
		entries[i].HasDraft = hasDraft(entries[i].Type, entries[i].Name, draftPresence)
	}

	response := CatalogResponse{
		Entries: entries,
		Total:   len(entries),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func enumerateEntities(baseDir string, entityType string) ([]CatalogEntry, error) {
	entries := []CatalogEntry{}

	// Check if directory exists
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return entries, nil // Return empty list if directory doesn't exist
	}

	// Read directory entries
	dirEntries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", baseDir, err)
	}

	for _, dirEntry := range dirEntries {
		if !dirEntry.IsDir() {
			continue
		}

		name := dirEntry.Name()
		prdPath := filepath.Join(baseDir, name, "PRD.md")

		// Check if PRD.md exists
		hasPRD := false
		if _, err := os.Stat(prdPath); err == nil {
			hasPRD = true
		}

		entry := CatalogEntry{
			Type:    entityType,
			Name:    name,
			HasPRD:  hasPRD,
			PRDPath: prdPath,
		}

		// Extract brief description from PRD if available
		if hasPRD {
			description := extractDescription(prdPath)
			entry.Description = description
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func extractDescription(prdPath string) string {
	content, err := os.ReadFile(prdPath)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(content), "\n")

	// Look for the first non-empty line after the title
	foundTitle := false
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip markdown header
		if strings.HasPrefix(line, "#") {
			foundTitle = true
			continue
		}

		// Return first substantial line after title
		if foundTitle && len(line) > 0 && !strings.HasPrefix(line, "##") {
			// Remove markdown formatting
			line = strings.TrimPrefix(line, "> ")
			line = strings.TrimPrefix(line, "**")
			line = strings.TrimSuffix(line, "**")

			// Truncate if too long
			if len(line) > 200 {
				line = line[:197] + "..."
			}

			return line
		}
	}

	return ""
}

func hasDraft(entityType string, entityName string, dbPresence map[string]struct{}) bool {
	if len(dbPresence) > 0 {
		if _, ok := dbPresence[draftPresenceKey(entityType, entityName)]; ok {
			return true
		}
	}

	return hasDraftOnDisk(entityType, entityName)
}

func hasDraftOnDisk(entityType string, entityName string) bool {
	draftPath := getDraftPath(entityType, entityName)
	_, err := os.Stat(draftPath)
	return err == nil
}

func getDraftPath(entityType string, entityName string) string {
	return filepath.Join("../data/prd-drafts", entityType, entityName+".md")
}

func loadDraftPresence() (map[string]struct{}, error) {
	if db == nil {
		return nil, nil
	}

	rows, err := db.Query(`SELECT entity_type, entity_name FROM drafts`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	presence := make(map[string]struct{})
	for rows.Next() {
		var entityType, entityName string
		if err := rows.Scan(&entityType, &entityName); err != nil {
			return nil, err
		}
		presence[draftPresenceKey(entityType, entityName)] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return presence, nil
}

func draftPresenceKey(entityType, entityName string) string {
	return fmt.Sprintf("%s/%s", entityType, entityName)
}

func handleGetPublishedPRD(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	entityType := vars["type"]
	entityName := vars["name"]

	// Validate entity type
	if entityType != "scenario" && entityType != "resource" {
		http.Error(w, "Invalid entity type. Must be 'scenario' or 'resource'", http.StatusBadRequest)
		return
	}

	// Construct PRD path
	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	baseDir := filepath.Join(vrooliRoot, entityType+"s")
	prdPath := filepath.Join(baseDir, entityName, "PRD.md")

	// Read PRD content
	content, err := os.ReadFile(prdPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, fmt.Sprintf("PRD not found for %s/%s", entityType, entityName), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to read PRD: %v", err), http.StatusInternalServerError)
		return
	}

	htmlContent, err := markdownToHTML(content)
	if err != nil {
		slog.Warn("failed to render markdown", "entityType", entityType, "entityName", entityName, "error", err)
	}

	response := PublishedPRDResponse{
		Type:        entityType,
		Name:        entityName,
		Content:     string(content),
		Path:        prdPath,
		ContentHTML: htmlContent,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleEnsureDraftFromPublishedPRD(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if db == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	entityType := vars["type"]
	entityName := vars["name"]

	if entityType != "scenario" && entityType != "resource" {
		http.Error(w, "Invalid entity type. Must be 'scenario' or 'resource'", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(entityName) == "" {
		http.Error(w, "Entity name is required", http.StatusBadRequest)
		return
	}

	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	prdPath := filepath.Join(vrooliRoot, entityType+"s", entityName, "PRD.md")
	content, err := os.ReadFile(prdPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, fmt.Sprintf("PRD not found for %s/%s", entityType, entityName), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to read PRD: %v", err), http.StatusInternalServerError)
		return
	}

	draft, err := ensureDraftFromPublishedPRD(sqlDraftStore{db: db}, entityType, entityName, string(content))
	if err != nil {
		slog.Error("failed to ensure draft from published PRD", "entityType", entityType, "entityName", entityName, "error", err)
		http.Error(w, fmt.Sprintf("Failed to prepare draft: %v", err), http.StatusInternalServerError)
		return
	}

	if err := saveDraftToFile(entityType, entityName, draft.Content); err != nil {
		slog.Error("failed to persist draft file", "entityType", entityType, "entityName", entityName, "error", err)
		http.Error(w, fmt.Sprintf("Failed to save draft file: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(draft)
}

func ensureDraftFromPublishedPRD(store draftStore, entityType string, entityName string, content string) (Draft, error) {
	var draft Draft
	var owner sql.NullString

	row := store.QueryRow(`
		SELECT id, entity_type, entity_name, content, owner, created_at, updated_at, status
		FROM drafts
		WHERE entity_type = $1 AND entity_name = $2
	`, entityType, entityName)

	scanErr := row.Scan(
		&draft.ID,
		&draft.EntityType,
		&draft.EntityName,
		&draft.Content,
		&owner,
		&draft.CreatedAt,
		&draft.UpdatedAt,
		&draft.Status,
	)

	switch {
	case errors.Is(scanErr, sql.ErrNoRows):
		now := time.Now()
		draft = Draft{
			ID:         uuid.New().String(),
			EntityType: entityType,
			EntityName: entityName,
			Content:    content,
			CreatedAt:  now,
			UpdatedAt:  now,
			Status:     "draft",
		}

		if _, err := store.Exec(`
			INSERT INTO drafts (id, entity_type, entity_name, content, owner, created_at, updated_at, status)
			VALUES ($1, $2, $3, $4, $5, $6, $6, 'draft')
		`, draft.ID, entityType, entityName, content, nullString(""), now); err != nil {
			return Draft{}, fmt.Errorf("failed to create draft: %w", err)
		}

		return draft, nil
	case scanErr != nil:
		return Draft{}, fmt.Errorf("failed to query draft: %w", scanErr)
	default:
		if owner.Valid {
			draft.Owner = owner.String
		}

		if draft.Status != "draft" {
			now := time.Now()
			if _, err := store.Exec(`
				UPDATE drafts
				SET content = $1, status = 'draft', updated_at = $2
				WHERE id = $3
			`, content, now, draft.ID); err != nil {
				return Draft{}, fmt.Errorf("failed to reset draft: %w", err)
			}
			draft.Content = content
			draft.Status = "draft"
			draft.UpdatedAt = now
		}

		return draft, nil
	}
}
