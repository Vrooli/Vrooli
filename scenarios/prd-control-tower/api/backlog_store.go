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

	return fmt.Sprintf(`# Product Requirements Document (PRD)

> Converted from backlog idea: %s

## 🎯 Overview

**What this scenario delivers:**
Describe the core capability this scenario adds to Vrooli. Connect it to the original backlog idea and explain why this matters to the ecosystem.

**Problem being solved:**
- What pain point or gap does this address?
- Who benefits from this capability?
- What's the cost of not building this?

**Success criteria:**
- Define what "done" looks like
- List 2-3 measurable outcomes

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] **[T.001]** _Brief target description_ — Define the essential capability for launch
- [ ] **[T.002]** _Brief target description_ — Add critical functionality required
- [ ] **[T.003]** _Brief target description_ — Implement core workflow

### 🟠 P1 – Should have post-launch
- [ ] **[T.101]** _Brief target description_ — Enhance user experience
- [ ] **[T.102]** _Brief target description_ — Add supporting features

### 🟢 P2 – Future / expansion
- [ ] **[T.201]** _Brief target description_ — Consider advanced features
- [ ] **[T.202]** _Brief target description_ — Explore future enhancements

## 🧱 Tech Direction Snapshot

**Architecture overview:**
- Core components and how they interact
- Key integrations with existing scenarios/resources

**Tech stack:**
- Language/framework: [e.g., Go, React, TypeScript]
- Storage: [e.g., PostgreSQL, Redis]
- Dependencies: [List key resources this scenario needs]

**Data model:**
- Key entities and relationships
- Storage strategy

## 🤝 Dependencies & Launch Plan

**Prerequisites:**
- Resource dependencies: [List required resources]
- Scenario dependencies: [List scenarios this depends on]
- External dependencies: [APIs, services, etc.]

**Launch phases:**
1. **Phase 1 - Foundation** — Setup scaffolding, database schema, core API
2. **Phase 2 - Core Features** — Implement P0 targets, basic UI
3. **Phase 3 - Polish & Ship** — Testing, documentation, deployment

**Risks & mitigation:**
- Risk: [Describe potential issue]
  - Mitigation: [How to address it]

## 🎨 UX & Branding

**User experience:**
- Primary user flows
- Key interactions and screens
- Navigation structure

**Visual design:**
- Color palette and theming
- Component style (matches Vrooli design system)
- Accessibility considerations

## 📎 Appendix

**References:**
- Original backlog idea: %s
- Related PRDs: [Link to related scenarios]
- Design assets: [Link to mockups if available]

**Notes:**
- Additional context or considerations
- Open questions to resolve

---

**Status:** 🟡 Draft — Converted from backlog. Review and expand sections, link operational targets to requirements, then publish.
`, displayName, ideaText)
}
