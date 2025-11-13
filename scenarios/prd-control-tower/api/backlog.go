package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const (
	BacklogStatusPending   = "pending"
	BacklogStatusConverted = "converted"
	BacklogStatusArchived  = "archived"
)

var bulletStripper = regexp.MustCompile(`^[\s\-\*•‣◦⁃∙·]+`)

// BacklogEntry represents a jot-note style backlog item before draft creation.
type BacklogEntry struct {
	ID               string    `json:"id"`
	IdeaText         string    `json:"idea_text"`
	EntityType       string    `json:"entity_type"`
	SuggestedName    string    `json:"suggested_name"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	ConvertedDraftID *string   `json:"converted_draft_id,omitempty"`
}

type BacklogListResponse struct {
	Entries []BacklogEntry `json:"entries"`
	Total   int            `json:"total"`
}

type BacklogCreateEntry struct {
	IdeaText      string `json:"idea_text"`
	EntityType    string `json:"entity_type"`
	SuggestedName string `json:"suggested_name"`
}

type BacklogCreateRequest struct {
	RawInput   string               `json:"raw_input"`
	EntityType string               `json:"entity_type"`
	Entries    []BacklogCreateEntry `json:"entries"`
}

type BacklogCreateResponse struct {
	Entries []BacklogEntry `json:"entries"`
}

type BacklogConvertRequest struct {
	EntryIDs []string `json:"entry_ids"`
}

type BacklogConvertResult struct {
	Entry BacklogEntry `json:"entry"`
	Draft *Draft       `json:"draft,omitempty"`
	Error string       `json:"error,omitempty"`
}

type BacklogConvertResponse struct {
	Results []BacklogConvertResult `json:"results"`
}

func handleListBacklog(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		respondServiceUnavailable(w, "Database not available")
		return
	}

	rows, err := db.Query(`
        SELECT id, idea_text, entity_type, suggested_name, status, converted_draft_id, created_at, updated_at
        FROM backlog_entries
        ORDER BY created_at DESC
    `)
	if err != nil {
		respondInternalError(w, "Failed to list backlog entries", err)
		return
	}
	defer rows.Close()

	entries := make([]BacklogEntry, 0)
	for rows.Next() {
		var entry BacklogEntry
		var draftID sql.NullString
		if err := rows.Scan(&entry.ID, &entry.IdeaText, &entry.EntityType, &entry.SuggestedName, &entry.Status, &draftID, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
			respondInternalError(w, "Failed to scan backlog entry", err)
			return
		}
		if draftID.Valid {
			entry.ConvertedDraftID = &draftID.String
		}
		entries = append(entries, entry)
	}

	respondJSON(w, http.StatusOK, BacklogListResponse{Entries: entries, Total: len(entries)})
}

func handleCreateBacklogEntries(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		respondServiceUnavailable(w, "Database not available")
		return
	}

	var req BacklogCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondInvalidJSON(w, err)
		return
	}

	entries := req.Entries
	if req.RawInput != "" {
		parsed := parseBacklogInput(req.RawInput, req.EntityType)
		entries = append(entries, parsed...)
	}

	if len(entries) == 0 {
		respondBadRequest(w, "No backlog items provided")
		return
	}

	if len(entries) > 50 {
		respondBadRequest(w, "Too many backlog items in a single request (max 50)")
		return
	}

	created, err := insertBacklogEntries(entries)
	if err != nil {
		respondInternalError(w, "Failed to create backlog entries", err)
		return
	}

	if len(created) == 0 {
		respondBadRequest(w, "No valid backlog entries could be created")
		return
	}

	respondJSON(w, http.StatusCreated, BacklogCreateResponse{Entries: created})
}

func handleDeleteBacklogEntry(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		respondServiceUnavailable(w, "Database not available")
		return
	}

	entryID := mux.Vars(r)["id"]
	if entryID == "" {
		respondBadRequest(w, "Backlog entry id required")
		return
	}

	result, err := db.Exec(`DELETE FROM backlog_entries WHERE id = $1`, entryID)
	if err != nil {
		respondInternalError(w, "Failed to delete backlog entry", err)
		return
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		respondNotFound(w, "Backlog entry")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func handleConvertSingleBacklogEntry(w http.ResponseWriter, r *http.Request) {
	entryID := mux.Vars(r)["id"]
	if entryID == "" {
		respondBadRequest(w, "Backlog entry id required")
		return
	}

	results, err := convertBacklogEntries([]string{entryID})
	if err != nil {
		respondInternalError(w, "Failed to convert backlog entry", err)
		return
	}

	if len(results) == 0 {
		respondNotFound(w, "Backlog entry")
		return
	}

	status := http.StatusOK
	if results[0].Error != "" {
		status = http.StatusConflict
	}

	respondJSON(w, status, results[0])
}

func handleConvertBacklogEntries(w http.ResponseWriter, r *http.Request) {
	var req BacklogConvertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondInvalidJSON(w, err)
		return
	}

	if len(req.EntryIDs) == 0 {
		respondBadRequest(w, "entry_ids is required")
		return
	}

	results, err := convertBacklogEntries(req.EntryIDs)
	if err != nil {
		respondInternalError(w, "Failed to convert backlog entries", err)
		return
	}

	respondJSON(w, http.StatusOK, BacklogConvertResponse{Results: results})
}

func insertBacklogEntries(entries []BacklogCreateEntry) ([]BacklogEntry, error) {
	if db == nil {
		return nil, ErrDatabaseNotAvailable
	}

	created := make([]BacklogEntry, 0, len(entries))
	for _, entry := range entries {
		idea := strings.TrimSpace(entry.IdeaText)
		if idea == "" {
			continue
		}

		entityType := strings.ToLower(strings.TrimSpace(entry.EntityType))
		if entityType == "" {
			entityType = EntityTypeScenario
		}
		if !isValidEntityType(entityType) {
			return nil, fmt.Errorf("invalid entity type %q", entry.EntityType)
		}

		suggestedName := entry.SuggestedName
		if suggestedName == "" {
			suggestedName = generateSlug(idea)
		}

		id := uuid.New().String()
		var createdAt, updatedAt time.Time
		var status string
		err := db.QueryRow(`
            INSERT INTO backlog_entries (id, idea_text, entity_type, suggested_name)
            VALUES ($1, $2, $3, $4)
            RETURNING created_at, updated_at, status
        `, id, idea, entityType, suggestedName).Scan(&createdAt, &updatedAt, &status)
		if err != nil {
			return nil, err
		}

		created = append(created, BacklogEntry{
			ID:            id,
			IdeaText:      idea,
			EntityType:    entityType,
			SuggestedName: suggestedName,
			Status:        status,
			CreatedAt:     createdAt,
			UpdatedAt:     updatedAt,
		})
	}

	return created, nil
}

func convertBacklogEntries(entryIDs []string) ([]BacklogConvertResult, error) {
	if db == nil {
		return nil, ErrDatabaseNotAvailable
	}

	results := make([]BacklogConvertResult, 0, len(entryIDs))
	for _, entryID := range entryIDs {
		entry, err := getBacklogEntryByID(entryID)
		if err != nil {
			results = append(results, BacklogConvertResult{Entry: BacklogEntry{ID: entryID}, Error: fmt.Sprintf("%v", err)})
			continue
		}

		if entry.Status == BacklogStatusConverted && entry.ConvertedDraftID != nil {
			draft, draftErr := getDraftByID(*entry.ConvertedDraftID)
			if draftErr != nil {
				results = append(results, BacklogConvertResult{Entry: entry, Error: draftErr.Error()})
				continue
			}
			results = append(results, BacklogConvertResult{Entry: entry, Draft: &draft})
			continue
		}

		entityName, err := ensureUniqueEntityName(entry.EntityType, entry.SuggestedName)
		if err != nil {
			results = append(results, BacklogConvertResult{Entry: entry, Error: err.Error()})
			continue
		}

		content := buildBacklogDraftContent(entry.IdeaText, entityName)
		draft, err := upsertDraft(entry.EntityType, entityName, content, "")
		if err != nil {
			results = append(results, BacklogConvertResult{Entry: entry, Error: err.Error()})
			continue
		}

		if err := saveDraftToFile(draft.EntityType, draft.EntityName, draft.Content); err != nil {
			results = append(results, BacklogConvertResult{Entry: entry, Error: err.Error()})
			continue
		}

		updatedEntry, err := markBacklogEntryConverted(entry.ID, draft.ID, entityName)
		if err != nil {
			results = append(results, BacklogConvertResult{Entry: entry, Error: err.Error()})
			continue
		}

		results = append(results, BacklogConvertResult{Entry: updatedEntry, Draft: &draft})
	}

	return results, nil
}

func getBacklogEntryByID(id string) (BacklogEntry, error) {
	if db == nil {
		return BacklogEntry{}, ErrDatabaseNotAvailable
	}

	var entry BacklogEntry
	var draftID sql.NullString
	err := db.QueryRow(`
        SELECT id, idea_text, entity_type, suggested_name, status, converted_draft_id, created_at, updated_at
        FROM backlog_entries
        WHERE id = $1
    `, id).Scan(&entry.ID, &entry.IdeaText, &entry.EntityType, &entry.SuggestedName, &entry.Status, &draftID, &entry.CreatedAt, &entry.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return BacklogEntry{}, fmt.Errorf("backlog entry not found")
		}
		return BacklogEntry{}, err
	}

	if draftID.Valid {
		entry.ConvertedDraftID = &draftID.String
	}
	return entry, nil
}

func markBacklogEntryConverted(id string, draftID string, entityName string) (BacklogEntry, error) {
	if db == nil {
		return BacklogEntry{}, ErrDatabaseNotAvailable
	}

	var entry BacklogEntry
	var converted sql.NullString
	err := db.QueryRow(`
        UPDATE backlog_entries
        SET status = $2, converted_draft_id = $3, suggested_name = $4, updated_at = NOW()
        WHERE id = $1
        RETURNING id, idea_text, entity_type, suggested_name, status, converted_draft_id, created_at, updated_at
    `, id, BacklogStatusConverted, draftID, entityName).Scan(&entry.ID, &entry.IdeaText, &entry.EntityType, &entry.SuggestedName, &entry.Status, &converted, &entry.CreatedAt, &entry.UpdatedAt)
	if err != nil {
		return BacklogEntry{}, err
	}

	if converted.Valid {
		entry.ConvertedDraftID = &converted.String
	}

	return entry, nil
}

func parseBacklogInput(raw string, defaultType string) []BacklogCreateEntry {
	lines := strings.Split(raw, "\n")
	entries := make([]BacklogCreateEntry, 0, len(lines))
	fallback := strings.ToLower(strings.TrimSpace(defaultType))
	if fallback == "" {
		fallback = EntityTypeScenario
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		trimmed = bulletStripper.ReplaceAllString(trimmed, "")
		trimmed = stripLeadingNumber(trimmed)

		entryType := fallback
		lower := strings.ToLower(trimmed)
		switch {
		case strings.HasPrefix(lower, "[scenario]"):
			entryType = EntityTypeScenario
			trimmed = strings.TrimSpace(trimmed[len("[scenario]"):])
		case strings.HasPrefix(lower, "[resource]"):
			entryType = EntityTypeResource
			trimmed = strings.TrimSpace(trimmed[len("[resource]"):])
		}

		if trimmed == "" {
			continue
		}

		slug := generateSlug(trimmed)
		entries = append(entries, BacklogCreateEntry{
			IdeaText:      trimmed,
			EntityType:    entryType,
			SuggestedName: slug,
		})
	}

	return entries
}

func stripLeadingNumber(input string) string {
	i := 0
	for i < len(input) && input[i] >= '0' && input[i] <= '9' {
		i++
	}
	if i < len(input) && (input[i] == '.' || input[i] == ')') {
		i++
	}
	for i < len(input) && (input[i] == ' ' || input[i] == '\t') {
		i++
	}
	return strings.TrimSpace(input[i:])
}

func generateSlug(input string) string {
	lower := strings.ToLower(strings.TrimSpace(input))
	var builder strings.Builder
	prevHyphen := false
	for _, r := range lower {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			prevHyphen = false
			continue
		}
		if r == ' ' || r == '-' || r == '_' {
			if !prevHyphen && builder.Len() > 0 {
				builder.WriteRune('-')
				prevHyphen = true
			}
			continue
		}
	}

	slug := strings.Trim(builder.String(), "-")
	if slug == "" {
		slug = fmt.Sprintf("scenario-%d", time.Now().Unix())
	}
	if len(slug) > 80 {
		slug = slug[:80]
	}
	return slug
}

func ensureUniqueEntityName(entityType string, base string) (string, error) {
	name := base
	for i := 1; i <= 50; i++ {
		exists, err := draftExists(entityType, name)
		if err != nil {
			return "", err
		}
		if !exists {
			return name, nil
		}
		name = fmt.Sprintf("%s-%d", base, i)
	}
	return "", fmt.Errorf("unable to determine unique name for %s", base)
}

func draftExists(entityType, entityName string) (bool, error) {
	if db == nil {
		return false, ErrDatabaseNotAvailable
	}

	var exists bool
	err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM drafts WHERE entity_type = $1 AND entity_name = $2)`, entityType, entityName).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func buildBacklogDraftContent(ideaText, entityName string) string {
	return fmt.Sprintf(`# Product Requirements Document (PRD)

## Overview
- **Working name:** %s
- **Idea capture:** %s

## Problem Statement
Describe the problem this scenario/resource solves.

## Proposed Solution
Outline how %s should operate and the core capabilities it needs to deliver.

## Next Steps
1. Flesh out user journeys
2. Validate technical/resource dependencies
3. Convert this backlog item into a complete draft PRD
`, entityName, ideaText, entityName)
}
