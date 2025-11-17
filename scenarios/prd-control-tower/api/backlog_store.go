package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var errBacklogEntryNotFound = errors.New("backlog entry not found")

func fetchAllBacklogEntries() ([]BacklogEntry, error) {
	if db == nil {
		return nil, ErrDatabaseNotAvailable
	}

	rows, err := db.Query(`
		SELECT id, idea_text, entity_type, suggested_name, notes, status, converted_draft_id, created_at, updated_at
		FROM backlog_entries
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]BacklogEntry, 0)
	for rows.Next() {
		var entry BacklogEntry
		var draftID sql.NullString
		var notes sql.NullString
		if err := rows.Scan(&entry.ID, &entry.IdeaText, &entry.EntityType, &entry.SuggestedName, &notes, &entry.Status, &draftID, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
			return nil, err
		}
		if notes.Valid {
			entry.Notes = notes.String
		}
		if draftID.Valid {
			entry.ConvertedDraftID = &draftID.String
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
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

		notes := strings.TrimSpace(entry.Notes)
		id := uuid.New().String()
		var createdAt, updatedAt time.Time
		var status string
		err := db.QueryRow(`
			INSERT INTO backlog_entries (id, idea_text, entity_type, suggested_name, notes)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING created_at, updated_at, status
		`, id, idea, entityType, suggestedName, nullIfEmpty(notes)).Scan(&createdAt, &updatedAt, &status)
		if err != nil {
			return nil, err
		}

		backlogEntry := BacklogEntry{
			ID:            id,
			IdeaText:      idea,
			EntityType:    entityType,
			SuggestedName: suggestedName,
			Notes:         notes,
			Status:        status,
			CreatedAt:     createdAt,
			UpdatedAt:     updatedAt,
		}

		// Save to filesystem (git-backed persistence)
		if err := saveBacklogEntryToFile(backlogEntry); err != nil {
			return nil, fmt.Errorf("failed to save backlog entry to file: %w", err)
		}

		created = append(created, backlogEntry)
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
		draft, err := upsertDraftWithBacklog(entry.EntityType, entityName, content, "", entry.ID)
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
	var notes sql.NullString
	err := db.QueryRow(`
		SELECT id, idea_text, entity_type, suggested_name, notes, status, converted_draft_id, created_at, updated_at
		FROM backlog_entries
		WHERE id = $1
	`, id).Scan(&entry.ID, &entry.IdeaText, &entry.EntityType, &entry.SuggestedName, &notes, &entry.Status, &draftID, &entry.CreatedAt, &entry.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return BacklogEntry{}, errBacklogEntryNotFound
		}
		return BacklogEntry{}, err
	}

	if notes.Valid {
		entry.Notes = notes.String
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
	var notes sql.NullString
	err := db.QueryRow(`
		UPDATE backlog_entries
		SET status = $2, converted_draft_id = $3, suggested_name = $4, updated_at = NOW()
		WHERE id = $1
		RETURNING id, idea_text, entity_type, suggested_name, notes, status, converted_draft_id, created_at, updated_at
	`, id, BacklogStatusConverted, draftID, entityName).Scan(&entry.ID, &entry.IdeaText, &entry.EntityType, &entry.SuggestedName, &notes, &entry.Status, &converted, &entry.CreatedAt, &entry.UpdatedAt)
	if err != nil {
		return BacklogEntry{}, err
	}

	if notes.Valid {
		entry.Notes = notes.String
	}

	if converted.Valid {
		entry.ConvertedDraftID = &converted.String
	}

	// Save to filesystem (git-backed persistence)
	if err := saveBacklogEntryToFile(entry); err != nil {
		return BacklogEntry{}, fmt.Errorf("failed to save backlog entry to file: %w", err)
	}

	return entry, nil
}

func updateBacklogEntry(id string, update BacklogUpdateRequest) (BacklogEntry, error) {
	if db == nil {
		return BacklogEntry{}, ErrDatabaseNotAvailable
	}

	applyNotes := update.Notes != nil
	if !applyNotes {
		return BacklogEntry{}, fmt.Errorf("no fields provided to update")
	}

	var entry BacklogEntry
	var draftID sql.NullString
	var notes sql.NullString
	noteValue := strings.TrimSpace(*update.Notes)
	err := db.QueryRow(`
		UPDATE backlog_entries
		SET notes = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, idea_text, entity_type, suggested_name, notes, status, converted_draft_id, created_at, updated_at
	`, id, noteValue).Scan(&entry.ID, &entry.IdeaText, &entry.EntityType, &entry.SuggestedName, &notes, &entry.Status, &draftID, &entry.CreatedAt, &entry.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return BacklogEntry{}, errBacklogEntryNotFound
		}
		return BacklogEntry{}, err
	}

	if notes.Valid {
		entry.Notes = notes.String
	}
	if draftID.Valid {
		entry.ConvertedDraftID = &draftID.String
	}

	if err := saveBacklogEntryToFile(entry); err != nil {
		return BacklogEntry{}, fmt.Errorf("failed to persist backlog entry: %w", err)
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
	displayName := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(entityName, "-", " "), "_", " "))
	if displayName == "" {
		displayName = entityName
	}

	return fmt.Sprintf(`# %s

> Converted from backlog idea: %s

## Capability Narrative
Outline the permanent capability this scenario adds to Vrooli. Tie the story back to the backlog intent so future readers know why it matters.

## Why Now
- Urgency driver #1 (e.g. ecosystem dependency, missing automation, customer pull)
- Urgency driver #2
- Risk of waiting longer (cost, degraded experience, etc.)

## Stakeholders
- **Product Operations** — keeps the catalog honest and validates the PRD spine.
- **Scenario Maintainers** — will build + operate this capability.
- **Meta-scenarios & Agents** — depend on this scenario once it ships.

## User Journeys
1. Describe the most important human or agent flow that this scenario must power.
2. Keep them task oriented ("As product ops..." or "When the orchestrator...").
3. Reference linked requirements or operational targets if helpful.

## Requirements
- [ ] PRD-001 | Define the essential behavior and link to /requirements/*.json.
- [ ] PRD-002 | Capture integrations, resources, and automation surfaces.
- [ ] PRD-003 | Lock the success criteria for launch + observability.

## Operational Targets
- Target name (P0/P1) — Metric definition, measurement method, and which requirements back it.
- Add at least one per P0 requirement and keep them machine-readable.

## Launch Plan
1. Kickoff + scaffolding — Owner, due date, current status.
2. Instrumentation + quality gates — Owner, due date, status.
3. Ship + monitor with ecosystem-manager — Owner, due date, status.

## Status
**Yellow** — Converted from backlog. Flesh out the sections above, validate with scenario-auditor, and publish once ready.

`, displayName, ideaText)
}
