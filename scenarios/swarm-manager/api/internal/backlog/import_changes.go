// DOC: docs/reference/operational-targets.md
// DOC: docs/concepts/ARCHITECTURE.md
package backlog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// createItemData holds data needed to create a new item during apply.
type createItemData struct {
	kind        BacklogKind
	name        string
	title       string
	description string
	priority    int
	tags        []string
}

// updateItemData holds data needed to update an existing item during apply.
type updateItemData struct {
	kind             BacklogKind
	name             string
	item             BacklogItem
	description      *string
	status           *string
	priority         *int
	tags             []string
	hasTags          bool
	clarifyAnswers   map[int]string
	clarifyNotes     map[int]string
	suggestAccepted  map[int]bool
	suggestRejection map[int]string
	notes            string
}

// extractQuestionNumber extracts the 0-based question index from a heading like "#### Q1" or "#### Q2: ...".
func extractQuestionNumber(heading string) int {
	re := regexp.MustCompile(`[Qq](\d+)`)
	m := re.FindStringSubmatch(heading)
	if m == nil {
		return -1
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return -1
	}
	return n - 1 // Convert to 0-based index.
}

// extractSuggestionNumber extracts the 0-based suggestion index from a heading like "#### S1" or "#### 1.".
func extractSuggestionNumber(heading string) int {
	// Try S1 format first.
	re := regexp.MustCompile(`[Ss](\d+)`)
	if m := re.FindStringSubmatch(heading); m != nil {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return -1
		}
		return n - 1
	}
	// Try "#### 1." format.
	re2 := regexp.MustCompile(`####\s+(\d+)\.`)
	if m := re2.FindStringSubmatch(heading); m != nil {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return -1
		}
		return n - 1
	}
	return -1
}

// buildCreateChange builds a change for creating a new item.
func (h *Handler) buildCreateChange(parsed parsedItemSection) (importChange, []string) {
	var errs []string
	kind, err := ParseBacklogKind(parsed.kind)
	if err != nil {
		return importChange{}, []string{fmt.Sprintf("new item: invalid kind %q", parsed.kind)}
	}

	name := sanitizeName(parsed.name)
	if name == "" {
		return importChange{}, []string{"new item: name is empty after sanitization"}
	}

	itemKey := parsed.kind + "/" + name

	change := importChange{
		item:   itemKey,
		action: "create",
	}

	// Store the creation data on the change for apply phase.
	var details []string
	details = append(details, fmt.Sprintf("create %s with title %q", itemKey, parsed.title))
	if parsed.description != "" {
		details = append(details, "set description")
	}
	if parsed.hasPriority {
		details = append(details, fmt.Sprintf("priority: %d", parsed.priority))
	}
	if parsed.hasTags && len(parsed.tags) > 0 {
		details = append(details, fmt.Sprintf("tags: %s", strings.Join(parsed.tags, ", ")))
	}
	change.details = details

	// Store item data for apply.
	change.createData = &createItemData{
		kind:        kind,
		name:        name,
		title:       parsed.title,
		description: parsed.description,
		priority:    parsed.priority,
		tags:        parsed.tags,
	}

	return change, errs
}

// buildUpdateChange builds a change for an existing item.
func (h *Handler) buildUpdateChange(parsed parsedItemSection) (importChange, []string) {
	kind, err := ParseBacklogKind(parsed.kind)
	if err != nil {
		return importChange{}, []string{fmt.Sprintf("item %s/%s: invalid kind", parsed.kind, parsed.name)}
	}

	itemKey := parsed.kind + "/" + parsed.name
	change := importChange{
		item:   itemKey,
		action: "update",
	}

	// Load existing item.
	existing, err := h.store.LoadItem(kind, parsed.name)
	if err != nil {
		return importChange{}, []string{fmt.Sprintf("item %s: failed to load: %v", itemKey, err)}
	}

	var details []string
	ud := &updateItemData{
		kind:             kind,
		name:             parsed.name,
		item:             existing,
		clarifyAnswers:   parsed.clarifyAnswers,
		clarifyNotes:     parsed.clarifyNotes,
		suggestAccepted:  parsed.suggestAccepted,
		suggestRejection: parsed.suggestRejection,
		notes:            parsed.notes,
	}

	// Compare description.
	if parsed.description != "" && parsed.description != existing.Description {
		details = append(details, "description changed")
		desc := parsed.description
		ud.description = &desc
	}

	// Compare status.
	if parsed.hasStatus && parsed.status != string(existing.Status) {
		details = append(details, fmt.Sprintf("status: %s -> %s", existing.Status, parsed.status))
		ud.status = &parsed.status
	}

	// Compare priority.
	if parsed.hasPriority && parsed.priority != existing.Priority {
		details = append(details, fmt.Sprintf("priority: %d -> %d", existing.Priority, parsed.priority))
		ud.priority = &parsed.priority
	}

	// Compare tags.
	if parsed.hasTags && !tagsEqual(parsed.tags, existing.Tags) {
		details = append(details, fmt.Sprintf("tags: [%s] -> [%s]", strings.Join(existing.Tags, ", "), strings.Join(parsed.tags, ", ")))
		ud.tags = parsed.tags
		ud.hasTags = true
	}

	// Check clarify changes.
	if len(parsed.clarifyAnswers) > 0 || len(parsed.clarifyNotes) > 0 {
		questionsPath := filepath.Join(h.store.ItemDir(kind, parsed.name), "clarify", "questions.json")
		if _, err := os.Stat(questionsPath); err == nil {
			questions, err := loadQuestions(questionsPath)
			if err == nil {
				for idx, answer := range parsed.clarifyAnswers {
					if idx < len(questions) && answer != "" {
						if questions[idx].Answer != answer {
							details = append(details, fmt.Sprintf("clarify Q%d answer: %q", idx+1, answer))
						}
					}
				}
				for idx, notes := range parsed.clarifyNotes {
					if idx < len(questions) && notes != "" {
						if questions[idx].Notes != notes {
							details = append(details, fmt.Sprintf("clarify Q%d notes updated", idx+1))
						}
					}
				}
			}
		}
	}

	// Check suggest changes.
	if len(parsed.suggestAccepted) > 0 || len(parsed.suggestRejection) > 0 {
		suggestionsPath := filepath.Join(h.store.ItemDir(kind, parsed.name), "suggest", "suggestions.json")
		if _, err := os.Stat(suggestionsPath); err == nil {
			suggestions, err := loadSuggestions(suggestionsPath)
			if err == nil {
				for idx, accepted := range parsed.suggestAccepted {
					if idx < len(suggestions) {
						if suggestions[idx].Accepted != accepted {
							if accepted {
								details = append(details, fmt.Sprintf("suggestion S%d accepted", idx+1))
							} else {
								details = append(details, fmt.Sprintf("suggestion S%d rejected", idx+1))
							}
						}
					}
				}
				for idx, reason := range parsed.suggestRejection {
					if idx < len(suggestions) && reason != "" {
						if suggestions[idx].RejectionReason != reason {
							details = append(details, fmt.Sprintf("suggestion S%d rejection reason updated", idx+1))
						}
					}
				}
			}
		}
	}

	// Check notes changes.
	if parsed.notes != "" {
		notesPath := filepath.Join(h.store.ItemDir(kind, parsed.name), "notes.md")
		existingNotes := ""
		if data, err := os.ReadFile(notesPath); err == nil {
			existingNotes = strings.TrimSpace(string(data))
		}
		if parsed.notes != existingNotes {
			details = append(details, "notes updated")
		}
	}

	change.details = details
	change.updateData = ud

	return change, nil
}

// loadQuestions reads and parses a questions.json file.
func loadQuestions(path string) ([]clarifyQuestion, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var questions []clarifyQuestion
	if err := json.Unmarshal(data, &questions); err != nil {
		return nil, fmt.Errorf("failed to parse questions.json: %w", err)
	}
	return questions, nil
}

// loadSuggestions reads and parses a suggestions.json file.
func loadSuggestions(path string) ([]suggestion, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var suggestions []suggestion
	if err := json.Unmarshal(data, &suggestions); err != nil {
		return nil, fmt.Errorf("failed to parse suggestions.json: %w", err)
	}
	return suggestions, nil
}

// tagsEqual compares two tag slices for equality.
func tagsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
