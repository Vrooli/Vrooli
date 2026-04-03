// DOC: docs/reference/operational-targets.md
// DOC: docs/concepts/ARCHITECTURE.md
package backlog

import (
	"fmt"
	"strconv"
	"strings"
)

// parsedItemSection holds the parsed content of a single item section from the markdown.
type parsedItemSection struct {
	kind        string
	name        string
	isNew       bool
	title       string
	description string
	status      string
	priority    int
	tags        []string

	// clarify section data
	clarifyAnswers map[int]string // question index -> selected option text
	clarifyNotes   map[int]string // question index -> notes text

	// suggest section data
	suggestAccepted  map[int]bool   // suggestion index -> accepted
	suggestRejection map[int]string // suggestion index -> rejection reason

	// notes section
	notes string

	// metadata parsed
	hasStatus   bool
	hasPriority bool
	hasTags     bool
}

// parseImportMarkdown parses the markdown content and returns changes and errors.
func (h *Handler) parseImportMarkdown(content string) ([]importChange, []string) {
	var changes []importChange
	var errs []string

	lines := strings.Split(content, "\n")

	// 1. Parse and validate frontmatter.
	fmStart, fmEnd, fmValid := parseFrontmatter(lines)
	if !fmValid {
		return nil, []string{"frontmatter missing or invalid: expected version: 1"}
	}
	_ = fmStart // consumed

	// 2. Split into item sections starting after frontmatter.
	sections := splitItemSections(lines[fmEnd+1:])

	// 3. Process each section.
	for _, section := range sections {
		parsed, sectionErrs := parseItemSection(section)
		if len(sectionErrs) > 0 {
			errs = append(errs, sectionErrs...)
			continue
		}

		if parsed.isNew {
			change, createErrs := h.buildCreateChange(parsed)
			if len(createErrs) > 0 {
				errs = append(errs, createErrs...)
				continue
			}
			changes = append(changes, change)
		} else {
			change, updateErrs := h.buildUpdateChange(parsed)
			if len(updateErrs) > 0 {
				errs = append(errs, updateErrs...)
				continue
			}
			if len(change.details) > 0 {
				changes = append(changes, change)
			}
		}
	}

	return changes, errs
}

// parseFrontmatter finds YAML frontmatter delimiters and validates version: 1.
// Returns (startLine, endLine, valid).
func parseFrontmatter(lines []string) (int, int, bool) {
	start := -1
	end := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if start == -1 {
				start = i
			} else {
				end = i
				break
			}
		}
	}
	if start == -1 || end == -1 {
		return 0, 0, false
	}

	// Check for version: 1 in frontmatter.
	foundVersion := false
	for i := start + 1; i < end; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "version:") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "version:"))
			if val == "1" {
				foundVersion = true
			}
		}
	}
	return start, end, foundVersion
}

// itemSection holds a raw section with its marker info.
type itemSection struct {
	marker string // the full marker text like "idea/my-app" or "NEW"
	lines  []string
}

// splitItemSections splits lines into sections delimited by <!-- item:... --> markers.
func splitItemSections(lines []string) []itemSection {
	var sections []itemSection
	var current *itemSection

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if m := itemMarkerRe.FindStringSubmatch(trimmed); m != nil {
			if current != nil {
				sections = append(sections, *current)
			}
			current = &itemSection{marker: m[1]}
			continue
		}
		if current != nil {
			current.lines = append(current.lines, line)
		}
	}
	if current != nil {
		sections = append(sections, *current)
	}
	return sections
}

// parseItemSection parses a single item section into a structured representation.
func parseItemSection(section itemSection) (parsedItemSection, []string) {
	parsed := parsedItemSection{
		clarifyAnswers:   make(map[int]string),
		clarifyNotes:     make(map[int]string),
		suggestAccepted:  make(map[int]bool),
		suggestRejection: make(map[int]string),
		priority:         5,
	}
	var errs []string

	if strings.ToUpper(section.marker) == "NEW" {
		parsed.isNew = true
	} else {
		parts := strings.SplitN(section.marker, "/", 2)
		if len(parts) != 2 {
			return parsed, []string{fmt.Sprintf("malformed item marker: %s", section.marker)}
		}
		parsed.kind = strings.TrimSpace(parts[0])
		parsed.name = strings.TrimSpace(parts[1])
	}

	// Sub-split by inner markers (clarify, suggest, notes) and detect heading/description/metadata.
	type subSection struct {
		sectionType string // "main", "clarify", "suggest", "notes"
		lines       []string
	}

	var subSections []subSection
	currentSub := subSection{sectionType: "main"}

	for _, line := range section.lines {
		trimmed := strings.TrimSpace(line)

		if m := clarifyMarkerRe.FindStringSubmatch(trimmed); m != nil {
			subSections = append(subSections, currentSub)
			currentSub = subSection{sectionType: "clarify"}
			continue
		}
		if m := suggestMarkerRe.FindStringSubmatch(trimmed); m != nil {
			subSections = append(subSections, currentSub)
			currentSub = subSection{sectionType: "suggest"}
			continue
		}
		if m := notesMarkerRe.FindStringSubmatch(trimmed); m != nil {
			subSections = append(subSections, currentSub)
			currentSub = subSection{sectionType: "notes"}
			continue
		}
		// Stop on next item marker (shouldn't happen in normal flow, but guard).
		if itemMarkerRe.MatchString(trimmed) {
			break
		}
		currentSub.lines = append(currentSub.lines, line)
	}
	subSections = append(subSections, currentSub)

	for _, sub := range subSections {
		switch sub.sectionType {
		case "main":
			parseMainSection(sub.lines, &parsed)
		case "clarify":
			parseClarifySection(sub.lines, &parsed)
		case "suggest":
			parseSuggestSection(sub.lines, &parsed)
		case "notes":
			parseNotesSection(sub.lines, &parsed)
		}
	}

	// For new items, extract kind/name/title from heading if not yet set.
	if parsed.isNew {
		for _, line := range section.lines {
			trimmed := strings.TrimSpace(line)
			if m := newItemHeadingRe.FindStringSubmatch(trimmed); m != nil {
				parsed.kind = strings.TrimSpace(m[1])
				parsed.name = strings.TrimSpace(m[2])
				parsed.title = strings.TrimSpace(m[3])
				break
			}
		}
		if parsed.kind == "" || parsed.name == "" {
			errs = append(errs, "new item section missing kind/name heading")
		}
	}

	return parsed, errs
}

// parseMainSection extracts description and metadata table from the main section lines.
func parseMainSection(lines []string, parsed *parsedItemSection) {
	// Find description block: text between ### Description and next ### or end.
	inDescription := false
	var descLines []string
	inMetadata := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Stop at horizontal rule (---) which separates items.
		if trimmed == "---" {
			break
		}

		// Detect ### Description heading.
		if strings.HasPrefix(trimmed, "### Description") {
			inDescription = true
			inMetadata = false
			continue
		}

		// Detect any other ### heading or metadata table start.
		if strings.HasPrefix(trimmed, "###") || strings.HasPrefix(trimmed, "<!-- ") {
			inDescription = false
		}

		// Detect metadata table rows.
		if m := metadataRowRe.FindStringSubmatch(trimmed); m != nil {
			key := strings.ToLower(strings.TrimSpace(m[1]))
			val := strings.TrimSpace(m[2])

			// Skip table header separator rows.
			if strings.Contains(key, "---") || strings.Contains(val, "---") {
				continue
			}
			// Skip header row.
			if key == "field" || key == "property" {
				continue
			}

			inMetadata = true
			inDescription = false

			switch key {
			case "status":
				if validateBacklogStatus(val) {
					parsed.status = val
					parsed.hasStatus = true
				}
			case "priority":
				if p, err := strconv.Atoi(val); err == nil && p >= 1 && p <= 10 {
					parsed.priority = p
					parsed.hasPriority = true
				}
			case "tags":
				tagList := strings.Split(val, ",")
				var tags []string
				for _, t := range tagList {
					t = strings.TrimSpace(t)
					if t != "" {
						tags = append(tags, t)
					}
				}
				if len(tags) > 0 {
					parsed.tags = tags
					parsed.hasTags = true
				}
			}
			continue
		}

		if inDescription {
			descLines = append(descLines, line)
		}
		if inMetadata && trimmed == "" {
			inMetadata = false
		}
	}

	parsed.description = strings.TrimSpace(strings.Join(descLines, "\n"))
}

// parseClarifySection parses checkbox answers and notes from a clarify section.
func parseClarifySection(lines []string, parsed *parsedItemSection) {
	questionIdx := -1
	inNotes := false
	var notesLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect question headers (#### Q1, #### Q2, etc.).
		if strings.HasPrefix(trimmed, "####") || strings.HasPrefix(trimmed, "**Q") {
			// Save previous notes.
			if inNotes && questionIdx >= 0 {
				parsed.clarifyNotes[questionIdx] = strings.TrimSpace(strings.Join(notesLines, "\n"))
				notesLines = nil
				inNotes = false
			}

			// Extract question number.
			qNum := extractQuestionNumber(trimmed)
			if qNum >= 0 {
				questionIdx = qNum
			}
			continue
		}

		// Check for checkbox lines.
		if m := checkboxRe.FindStringSubmatch(line); m != nil {
			checked := strings.ToLower(m[1]) == "x"
			optionText := strings.TrimSpace(m[2])
			if checked && questionIdx >= 0 {
				parsed.clarifyAnswers[questionIdx] = optionText
			}
			continue
		}

		// Check for > Notes: line.
		if strings.HasPrefix(trimmed, "> Notes:") || strings.HasPrefix(trimmed, ">Notes:") {
			inNotes = true
			noteContent := strings.TrimSpace(strings.TrimPrefix(trimmed, "> Notes:"))
			noteContent = strings.TrimSpace(strings.TrimPrefix(noteContent, ">Notes:"))
			if noteContent != "" {
				notesLines = append(notesLines, noteContent)
			}
			continue
		}

		// Continuation of notes (lines starting with >).
		if inNotes && strings.HasPrefix(trimmed, ">") {
			noteLine := strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))
			notesLines = append(notesLines, noteLine)
			continue
		}

		// End of notes on non-blockquote line.
		if inNotes && trimmed != "" && !strings.HasPrefix(trimmed, ">") {
			parsed.clarifyNotes[questionIdx] = strings.TrimSpace(strings.Join(notesLines, "\n"))
			notesLines = nil
			inNotes = false
		}
	}

	// Save any trailing notes.
	if inNotes && questionIdx >= 0 {
		parsed.clarifyNotes[questionIdx] = strings.TrimSpace(strings.Join(notesLines, "\n"))
	}
}

// parseSuggestSection parses accept checkboxes and rejection reasons.
func parseSuggestSection(lines []string, parsed *parsedItemSection) {
	suggestionIdx := -1
	inRejection := false
	var rejectionLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect suggestion headers (#### S1, #### S2, or #### 1., etc.).
		if strings.HasPrefix(trimmed, "####") {
			// Save previous rejection.
			if inRejection && suggestionIdx >= 0 {
				parsed.suggestRejection[suggestionIdx] = strings.TrimSpace(strings.Join(rejectionLines, "\n"))
				rejectionLines = nil
				inRejection = false
			}

			sNum := extractSuggestionNumber(trimmed)
			if sNum >= 0 {
				suggestionIdx = sNum
			}
			continue
		}

		// Check for accept checkbox.
		if m := checkboxRe.FindStringSubmatch(line); m != nil {
			checked := strings.ToLower(m[1]) == "x"
			optionText := strings.TrimSpace(m[2])
			if suggestionIdx >= 0 && strings.Contains(strings.ToLower(optionText), "accept") {
				parsed.suggestAccepted[suggestionIdx] = checked
			}
			continue
		}

		// Check for > Rejection reason: line.
		if strings.HasPrefix(trimmed, "> Rejection reason:") || strings.HasPrefix(trimmed, ">Rejection reason:") {
			inRejection = true
			reasonContent := strings.TrimSpace(strings.TrimPrefix(trimmed, "> Rejection reason:"))
			reasonContent = strings.TrimSpace(strings.TrimPrefix(reasonContent, ">Rejection reason:"))
			if reasonContent != "" {
				rejectionLines = append(rejectionLines, reasonContent)
			}
			continue
		}

		// Continuation of rejection reason (lines starting with >).
		if inRejection && strings.HasPrefix(trimmed, ">") {
			reasonLine := strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))
			rejectionLines = append(rejectionLines, reasonLine)
			continue
		}

		// End of rejection on non-blockquote line.
		if inRejection && trimmed != "" && !strings.HasPrefix(trimmed, ">") {
			parsed.suggestRejection[suggestionIdx] = strings.TrimSpace(strings.Join(rejectionLines, "\n"))
			rejectionLines = nil
			inRejection = false
		}
	}

	// Save any trailing rejection.
	if inRejection && suggestionIdx >= 0 {
		parsed.suggestRejection[suggestionIdx] = strings.TrimSpace(strings.Join(rejectionLines, "\n"))
	}
}

// parseNotesSection extracts freeform notes content.
func parseNotesSection(lines []string, parsed *parsedItemSection) {
	parsed.notes = strings.TrimSpace(strings.Join(lines, "\n"))
}
