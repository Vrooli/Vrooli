package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

func fetchAllBacklogEntries() ([]BacklogEntry, error) {
	if db == nil {
		return nil, ErrDatabaseNotAvailable
	}

	rows, err := db.Query(`
		SELECT id, idea_text, entity_type, suggested_name, status, converted_draft_id, created_at, updated_at
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
		if err := rows.Scan(&entry.ID, &entry.IdeaText, &entry.EntityType, &entry.SuggestedName, &entry.Status, &draftID, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
			return nil, err
		}
		if draftID.Valid {
			entry.ConvertedDraftID = &draftID.String
		}
		entries = append(entries, entry)
	}

	return entries, nil
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
	// Use the standard PRD template structure with placeholders pre-filled where possible
	return fmt.Sprintf(`# Product Requirements Document (PRD)

> **Initial Idea:** %s

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
[Define the fundamental capability this scenario provides that will persist forever in the system. Initial idea: %s]

### Intelligence Amplification
**How does this capability make future agents smarter?**
[Describe how this capability compounds with existing capabilities to enable more complex problem-solving.]

### Recursive Value
**What new scenarios become possible after this exists?**
- [Example future scenario 1]
- [Example future scenario 2]
- [Example future scenario 3]

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] [Core requirement that defines minimum viable capability]
  - [ ] [Essential integration with shared resources]
  - [ ] [Critical data persistence requirement]

- **Should Have (P1)**
  - [ ] [Enhancement that significantly improves capability]
  - [ ] [Additional resource integration]
  - [ ] [Performance optimization]

- **Nice to Have (P2)**
  - [ ] [Future enhancement]
  - [ ] [Advanced feature]

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < [X]ms for 95%% of requests | API monitoring |
| Throughput | [Y] operations/second | Load testing |
| Accuracy | > [Z]%% for [specific task] | Validation suite |
| Resource Usage | < [N]GB memory, < [M]%% CPU | System monitoring |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with all required resources
- [ ] Performance targets met under load
- [ ] Documentation complete (README, API docs, CLI help)
- [ ] Scenario can be invoked by other agents via API/CLI

## 🏗️ Technical Architecture

### Resource Dependencies
**Required:**
- resource_name: [e.g., postgres]
  purpose: [Why this resource is essential]
  access_method: [How it's accessed - CLI, API, or shared workflow]

**Optional:**
- resource_name: [e.g., redis]
  purpose: [Enhancement this enables]
  fallback: [What happens if unavailable]
  access_method: [How this resource is accessed]

### Data Models
Core data structures that define the capability:
- [Entity name]: [Description of primary entity]
  - Storage: [postgres/qdrant/minio]
  - Key fields: [List main fields]

### API Contract
Primary endpoints this capability exposes:
- **POST /api/v1/[capability]/[action]**
  - Purpose: [What this enables other systems to do]
  - Input: [Required fields]
  - Output: [What calling systems can expect]

## 🖥️ CLI Interface Contract

### Command Structure
Primary CLI executable: **%s**

Core commands:
- **status**: Show operational status and resource health
- **help**: Display command help and usage
- **version**: Show CLI and API version information

Custom commands:
- **[action]**: [What this command does]

## 💰 Value Proposition

### Business Value
- **Primary Value**: [Core business problem solved]
- **Revenue Potential**: $[X]K - $[Y]K per deployment
- **Cost Savings**: [Time/resource savings quantified]
- **Market Differentiator**: [What makes this unique]

### Technical Value
- **Reusability Score**: [How many other scenarios can leverage this]
- **Complexity Reduction**: [What complex tasks become simple]
- **Innovation Enablement**: [New possibilities this creates]

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core capability implementation
- Basic resource integration
- Essential API/CLI interface

### Future Enhancements
- [Enhanced capability based on learnings]
- [Additional resource integrations]
- [Performance optimizations]

## ✅ Validation Criteria

### Capability Validated When:
- [ ] Solves the defined problem completely
- [ ] Integrates with upstream dependencies
- [ ] Enables downstream capabilities
- [ ] Maintains data consistency
- [ ] All P0 requirements complete and tested

## 📝 Implementation Notes

### Next Steps (Converted from Backlog)
1. Expand capability definition based on initial idea
2. Define specific functional requirements (P0/P1/P2)
3. Identify required resources and dependencies
4. Design data models and API contracts
5. Implement core functionality
6. Add comprehensive testing
7. Create CLI interface
8. Document usage and examples

---

**Status**: Draft (Converted from Backlog)
**Owner**: [AI Agent or Human maintainer]
**Created**: %s
`, ideaText, ideaText, entityName, "Backlog conversion")
}
