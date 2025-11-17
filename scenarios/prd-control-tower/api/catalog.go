package main

import (
	"bytes"
	"database/sql"
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
	Type            string `json:"type"`             // "scenario" or "resource"
	Name            string `json:"name"`             // Entity name
	HasPRD          bool   `json:"has_prd"`          // Whether PRD.md exists
	PRDPath         string `json:"prd_path"`         // Absolute path to PRD.md
	HasDraft        bool   `json:"has_draft"`        // Whether a draft exists
	HasRequirements bool   `json:"has_requirements"` // Whether requirements/index.json exists
	Description     string `json:"description"`      // Brief description (if available)
	Requirements    *CatalogRequirementSummary `json:"requirements_summary,omitempty"`
}

type CatalogRequirementSummary struct {
	Total      int `json:"total"`
	Completed  int `json:"completed"`
	InProgress int `json:"in_progress"`
	Pending    int `json:"pending"`
	P0         int `json:"p0"`
	P1         int `json:"p1"`
	P2         int `json:"p2"`
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
	entries, err := listCatalogEntries()
	if err != nil {
		respondInternalError(w, "Failed to load catalog", err)
		return
	}

	if shouldIncludeRequirementSummaries(r) {
		for i := range entries {
			if !entries[i].HasRequirements {
				continue
			}

			groups, err := loadRequirementsForEntity(entries[i].Type, entries[i].Name)
			if err != nil {
				slog.Warn("failed to summarize requirements", "entity", fmt.Sprintf("%s/%s", entries[i].Type, entries[i].Name), "error", err)
				continue
			}
			summary := summarizeRequirementGroupsForCatalog(groups)
			entries[i].Requirements = &summary
		}
	}

	response := CatalogResponse{
		Entries: entries,
		Total:   len(entries),
	}

	respondJSON(w, http.StatusOK, response)
}

func shouldIncludeRequirementSummaries(r *http.Request) bool {
	flag := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("include_requirements")))
	return flag == "1" || flag == "true" || flag == "yes"
}

func summarizeRequirementGroupsForCatalog(groups []RequirementGroup) CatalogRequirementSummary {
	var summary CatalogRequirementSummary
	visited := make(map[string]bool)
	var walk func(group RequirementGroup)
	walk = func(group RequirementGroup) {
		if visited[group.ID] {
			return
		}
		visited[group.ID] = true

		for _, req := range group.Requirements {
			summary.Total++
			switch strings.ToLower(strings.TrimSpace(req.Status)) {
			case "complete", "done":
				summary.Completed++
			case "in_progress", "in-progress":
				summary.InProgress++
			default:
				summary.Pending++
			}

			switch strings.ToUpper(strings.TrimSpace(req.Criticality)) {
			case "P0":
				summary.P0++
			case "P1":
				summary.P1++
			case "P2":
				summary.P2++
			}
		}

		for _, child := range group.Children {
			walk(child)
		}
	}

	for _, group := range groups {
		walk(group)
	}

	// Ensure pending count never goes negative if statuses were unrecognized
	if summary.Pending < 0 {
		summary.Pending = 0
	}

	return summary
}

// listCatalogEntries loads scenarios and resources along with PRD/draft status for reuse across endpoints.
func listCatalogEntries() ([]CatalogEntry, error) {
	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		return nil, err
	}

	entries := []CatalogEntry{}

	// Enumerate scenarios
	scenariosDir := filepath.Join(vrooliRoot, "scenarios")
	scenarios, err := enumerateEntities(scenariosDir, "scenario")
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate scenarios: %w", err)
	}
	entries = append(entries, scenarios...)

	// Enumerate resources
	resourcesDir := filepath.Join(vrooliRoot, "resources")
	resources, err := enumerateEntities(resourcesDir, "resource")
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate resources: %w", err)
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

	return entries, nil
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
		requirementsIndexPath := filepath.Join(baseDir, name, "requirements", "index.json")

		// Check if PRD.md exists
		hasPRD := false
		if _, err := os.Stat(prdPath); err == nil {
			hasPRD = true
		}

		// Check if requirements/index.json exists
		hasRequirements := false
		if _, err := os.Stat(requirementsIndexPath); err == nil {
			hasRequirements = true
		}

		entry := CatalogEntry{
			Type:            entityType,
			Name:            name,
			HasPRD:          hasPRD,
			PRDPath:         prdPath,
			HasRequirements: hasRequirements,
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

// hasDraft checks if a draft exists for the given entity using a two-tier approach.
//
// Performance characteristics:
//   - Database check: O(1) map lookup, very fast
//   - Filesystem check: O(1) stat syscall, slower due to disk I/O
//
// Why dual checking is needed:
//   - Database sync happens only during handleListDrafts (see syncDraftFilesystemWithDatabase)
//   - Drafts created directly on disk (e.g., by external tools or during initialization)
//     won't appear in the database until the next sync
//   - The filesystem fallback ensures catalog consistency even before first draft list access
//
// Trade-off: This adds ~1-5ms of I/O per catalog entry without a DB record, but ensures
// correctness and prevents confusing UX where drafts exist but aren't shown.
func hasDraft(entityType string, entityName string, dbPresence map[string]struct{}) bool {
	// Check database first (faster, O(1) map lookup)
	if _, ok := dbPresence[draftPresenceKey(entityType, entityName)]; ok {
		return true
	}

	// Fallback to filesystem check for drafts not yet synced to DB
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
	vars := mux.Vars(r)
	entityType := vars["type"]
	entityName := vars["name"]

	if !isValidEntityType(entityType) {
		respondInvalidEntityType(w)
		return
	}

	// Construct PRD path
	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		respondInternalError(w, "Failed to get Vrooli root", err)
		return
	}

	baseDir := filepath.Join(vrooliRoot, entityType+"s")
	prdPath := filepath.Join(baseDir, entityName, "PRD.md")

	// Read PRD content
	content, err := os.ReadFile(prdPath)
	if err != nil {
		if os.IsNotExist(err) {
			respondError(w, fmt.Sprintf("PRD not found for %s/%s", entityType, entityName), http.StatusNotFound)
			return
		}
		respondInternalError(w, "Failed to read PRD", err)
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

	respondJSON(w, http.StatusOK, response)
}

func handleEnsureDraftFromPublishedPRD(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityType := vars["type"]
	entityName := vars["name"]

	if !isValidEntityType(entityType) {
		respondInvalidEntityType(w)
		return
	}

	if strings.TrimSpace(entityName) == "" {
		respondBadRequest(w, "Entity name is required")
		return
	}

	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		respondInternalError(w, "Failed to get Vrooli root", err)
		return
	}

	prdPath := filepath.Join(vrooliRoot, entityType+"s", entityName, "PRD.md")
	content, err := os.ReadFile(prdPath)
	if err != nil {
		if os.IsNotExist(err) {
			respondError(w, fmt.Sprintf("PRD not found for %s/%s", entityType, entityName), http.StatusNotFound)
			return
		}
		respondInternalError(w, "Failed to read PRD", err)
		return
	}

	draft, err := ensureDraftFromPublishedPRD(sqlDB{db: db}, entityType, entityName, string(content))
	if err != nil {
		slog.Error("failed to ensure draft from published PRD", "entityType", entityType, "entityName", entityName, "error", err)
		respondInternalError(w, "Failed to prepare draft", err)
		return
	}

	if err := saveDraftToFile(entityType, entityName, draft.Content); err != nil {
		slog.Error("failed to persist draft file", "entityType", entityType, "entityName", entityName, "error", err)
		respondInternalError(w, "Failed to save draft file", err)
		return
	}

	respondJSON(w, http.StatusOK, draft)
}

func ensureDraftFromPublishedPRD(store dbQueryExecutor, entityType string, entityName string, content string) (Draft, error) {
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
			Status:     DraftStatusDraft,
		}

		if _, err := store.Exec(`
			INSERT INTO drafts (id, entity_type, entity_name, content, owner, created_at, updated_at, status)
			VALUES ($1, $2, $3, $4, $5, $6, $6, $7)
		`, draft.ID, entityType, entityName, content, sql.NullString{}, now, DraftStatusDraft); err != nil {
			return Draft{}, fmt.Errorf("failed to create draft: %w", err)
		}

		return draft, nil
	case scanErr != nil:
		return Draft{}, fmt.Errorf("failed to query draft: %w", scanErr)
	default:
		if owner.Valid {
			draft.Owner = owner.String
		}

		if draft.Status != DraftStatusDraft {
			now := time.Now()
			if _, err := store.Exec(`
				UPDATE drafts
				SET content = $1, status = $2, updated_at = $3
				WHERE id = $4
			`, content, DraftStatusDraft, now, draft.ID); err != nil {
				return Draft{}, fmt.Errorf("failed to reset draft: %w", err)
			}
			draft.Content = content
			draft.Status = DraftStatusDraft
			draft.UpdatedAt = now
		}

		return draft, nil
	}
}
