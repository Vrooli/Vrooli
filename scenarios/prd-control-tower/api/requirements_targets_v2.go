package main

import (
	"fmt"
	"regexp"
	"strings"
)

// Enhanced operational target parsing with explicit requirement linkages
//
// Supports both implicit (heuristic) and explicit (markdown tag) linkages:
// - Implicit: "Visual workflow builder" matches requirement with prd_ref containing "Visual workflow builder"
// - Explicit: `[req:BAS-FUNC-001,BAS-UI-002]` at end of line
//
// Format examples:
//   - [x] Visual workflow builder _(notes)_ `[req:BAS-FUNC-001]`
//   - [ ] AI debugging _(pending)_ `[req:AI-001,AI-002]`
//   - [ ] Performance dashboard  (no explicit linkages, uses heuristic)

var reqLinkPattern = regexp.MustCompile("`\\[req:([A-Z0-9,-]+)\\]`")

// parseOperationalTargetsV2 extracts operational targets with explicit requirement linkages
func parseOperationalTargetsV2(content, entityType, entityName string) []OperationalTarget {
	lines := strings.Split(content, "\n")
	var targets []OperationalTarget
	inFunctional := false
	currentCategory := ""
	currentCriticality := ""

	for _, raw := range lines {
		line := strings.TrimSpace(raw)

		// Section detection
		if strings.HasPrefix(line, "###") {
			if line == "### Functional Requirements" {
				inFunctional = true
				continue
			}
			if inFunctional {
				break // Exited functional requirements section
			}
		}

		if !inFunctional || line == "" {
			continue
		}

		// Category detection: - **Must Have (P0)**
		if strings.HasPrefix(line, "- **") && strings.HasSuffix(line, "**") {
			category := strings.TrimSuffix(strings.TrimPrefix(line, "- **"), "**")
			label, critic := parseCategoryLabel(category)
			currentCategory = label
			currentCriticality = critic
			continue
		}

		// Checkbox line detection: - [ ] or - [x]
		if !checkboxPattern.MatchString(line) {
			continue
		}

		// Parse target with explicit linkages
		target := parseTargetLine(line, entityType, entityName, currentCategory, currentCriticality)
		targets = append(targets, target)
	}

	return targets
}

// parseTargetLine extracts a single operational target from a markdown line
func parseTargetLine(line, entityType, entityName, category, criticality string) OperationalTarget {
	// Extract status from checkbox
	status := "pending"
	if strings.HasPrefix(strings.ToLower(line), "- [x]") {
		status = "complete"
	}

	// Remove checkbox prefix: "- [x] " or "- [ ] "
	content := strings.TrimSpace(line[4:])

	// Extract explicit requirement linkages: `[req:ID1,ID2]`
	var linkedReqs []string
	if matches := reqLinkPattern.FindStringSubmatch(content); len(matches) > 1 {
		// Parse comma-separated requirement IDs
		reqIDs := strings.Split(matches[1], ",")
		for _, id := range reqIDs {
			id = strings.TrimSpace(id)
			if id != "" {
				linkedReqs = append(linkedReqs, id)
			}
		}
		// Remove linkage tag from content
		content = strings.TrimSpace(reqLinkPattern.ReplaceAllString(content, ""))
	}

	// Split title and notes: "Title _(notes)_"
	title, notes := splitTitleAndNotes(content)

	// Generate stable ID
	id := slugify(fmt.Sprintf("%s-%s-%s", entityName, category, title))

	return OperationalTarget{
		ID:                 id,
		EntityType:         entityType,
		EntityName:         entityName,
		Category:           category,
		Criticality:        criticality,
		Title:              title,
		Notes:              notes,
		Status:             status,
		Path:               fmt.Sprintf("Functional Requirements > %s > %s", category, title),
		LinkedRequirements: linkedReqs,
	}
}

// splitTitleAndNotes separates "Title _(notes)_" into components
func splitTitleAndNotes(content string) (string, string) {
	// Match _(...)_ pattern for notes
	notesPattern := regexp.MustCompile(`_\((.*?)\)_`)
	if matches := notesPattern.FindStringSubmatch(content); len(matches) > 1 {
		notes := strings.TrimSpace(matches[1])
		title := strings.TrimSpace(notesPattern.ReplaceAllString(content, ""))
		return title, notes
	}
	return strings.TrimSpace(content), ""
}

// serializeOperationalTargets converts targets back to PRD markdown format
func serializeOperationalTargets(targets []OperationalTarget) string {
	if len(targets) == 0 {
		return ""
	}

	// Group targets by category
	categoryMap := make(map[string][]OperationalTarget)
	var categoryOrder []string // Preserve insertion order

	for _, target := range targets {
		key := target.Category
		if _, exists := categoryMap[key]; !exists {
			categoryOrder = append(categoryOrder, key)
		}
		categoryMap[key] = append(categoryMap[key], target)
	}

	var builder strings.Builder
	builder.WriteString("### Functional Requirements\n")

	for _, category := range categoryOrder {
		items := categoryMap[category]
		if len(items) == 0 {
			continue
		}

		// Determine criticality from first item (should be same for category)
		criticality := items[0].Criticality
		if criticality != "" {
			builder.WriteString(fmt.Sprintf("- **%s (%s)**\n", category, criticality))
		} else {
			builder.WriteString(fmt.Sprintf("- **%s**\n", category))
		}

		for _, target := range items {
			checkbox := "[ ]"
			if target.Status == "complete" {
				checkbox = "[x]"
			}

			line := fmt.Sprintf("  - %s %s", checkbox, target.Title)

			// Add notes if present
			if target.Notes != "" {
				line += fmt.Sprintf(" _(%s)_", target.Notes)
			}

			// Add explicit requirement linkages if present
			if len(target.LinkedRequirements) > 0 {
				reqList := strings.Join(target.LinkedRequirements, ",")
				line += fmt.Sprintf(" `[req:%s]`", reqList)
			}

			builder.WriteString(line + "\n")
		}
	}

	return builder.String()
}

// updateDraftOperationalTargets replaces the Functional Requirements section in a draft
func updateDraftOperationalTargets(draftContent string, targets []OperationalTarget) string {
	lines := strings.Split(draftContent, "\n")
	var result []string
	inFunctional := false
	functionalStart := -1

	// Find functional requirements section
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "### Functional Requirements" {
			inFunctional = true
			functionalStart = i
			continue
		}
		if inFunctional && strings.HasPrefix(trimmed, "###") {
			// End of functional requirements section
			// Insert everything before functional requirements
			result = append(result, lines[:functionalStart]...)
			// Insert updated functional requirements
			result = append(result, serializeOperationalTargets(targets))
			// Insert everything after functional requirements
			result = append(result, lines[i:]...)
			return strings.Join(result, "\n")
		}
	}

	// If we're still in functional requirements at end of file
	if inFunctional {
		result = append(result, lines[:functionalStart]...)
		result = append(result, serializeOperationalTargets(targets))
		return strings.Join(result, "\n")
	}

	// Functional requirements section not found - append at end
	result = append(result, lines...)
	result = append(result, "", serializeOperationalTargets(targets))
	return strings.Join(result, "\n")
}
