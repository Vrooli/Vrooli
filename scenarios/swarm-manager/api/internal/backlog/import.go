// DOC: docs/reference/operational-targets.md
// DOC: docs/concepts/ARCHITECTURE.md
package backlog

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/httputil"
)

// importChange is the internal representation before converting to proto.
type importChange struct {
	item       string // kind/name
	action     string // "update" or "create"
	details    []string
	createData *createItemData
	updateData *updateItemData
}

// itemMarkerRe matches <!-- item:KIND/NAME --> or <!-- item:NEW -->.
var itemMarkerRe = regexp.MustCompile(`^<!--\s*item:(\S+)\s*-->`)

// clarifyMarkerRe matches <!-- clarify:KIND/NAME -->.
var clarifyMarkerRe = regexp.MustCompile(`^<!--\s*clarify:(\S+)\s*-->`)

// suggestMarkerRe matches <!-- suggest:KIND/NAME -->.
var suggestMarkerRe = regexp.MustCompile(`^<!--\s*suggest:(\S+)\s*-->`)

// notesMarkerRe matches <!-- notes:KIND/NAME -->.
var notesMarkerRe = regexp.MustCompile(`^<!--\s*notes:(\S+)\s*-->`)

// newItemHeadingRe matches ## idea/my-new-idea -- Title Here (supports em-dash and double-dash).
var newItemHeadingRe = regexp.MustCompile(`^##\s+(\w+)/(\S+)\s*(?:—|--)\s*(.+)$`)

// checkboxRe matches [ ] or [x] checkbox lines.
var checkboxRe = regexp.MustCompile(`^\s*-\s*\[([ xX])\]\s*(.+)$`)

// metadataRowRe matches table rows like | **Status** | ready | or | Status | ready |.
var metadataRowRe = regexp.MustCompile(`^\|\s*\**(\w[\w\s]*?)\**\s*\|\s*(.*?)\s*\|`)

// Import handles the POST /api/v1/backlog/import endpoint.
// It parses an edited markdown export and applies (or previews) changes.
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		httputil.BadRequest(w, "[backlog] import", "failed to parse multipart form")
		return
	}

	applyChanges := r.FormValue("apply") == "true"

	file, _, err := r.FormFile("file")
	if err != nil {
		httputil.BadRequest(w, "[backlog] import", "file field is required")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		httputil.InternalError(w, "[backlog] import", "failed to read uploaded file")
		return
	}

	changes, parseErrors := h.parseImportMarkdown(string(content))

	if applyChanges {
		for i := range changes {
			if err := h.applyChange(&changes[i]); err != nil {
				parseErrors = append(parseErrors, fmt.Sprintf("%s: %v", changes[i].item, err))
			}
		}
	}

	updatedCount := 0
	createdCount := 0
	for _, c := range changes {
		switch c.action {
		case "update":
			updatedCount++
		case "create":
			createdCount++
		}
	}

	protoChanges := make([]*apipb.ImportChange, 0, len(changes))
	for _, c := range changes {
		protoChanges = append(protoChanges, &apipb.ImportChange{
			Item:    c.item,
			Action:  c.action,
			Details: c.details,
		})
	}

	resp := &apipb.ImportBacklogResponse{
		DryRun:  !applyChanges,
		Changes: protoChanges,
		Errors:  parseErrors,
		Summary: fmt.Sprintf("%d items updated, %d items created, %d errors", updatedCount, createdCount, len(parseErrors)),
	}

	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[backlog] import", "failed to encode response")
	}
}

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
	existing, err := h.loadItem(kind, parsed.name)
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
		questionsPath := filepath.Join(h.itemDir(kind, parsed.name), "clarify", "questions.json")
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
		suggestionsPath := filepath.Join(h.itemDir(kind, parsed.name), "suggest", "suggestions.json")
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
		notesPath := filepath.Join(h.itemDir(kind, parsed.name), "notes.md")
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

// applyChange executes a single import change.
func (h *Handler) applyChange(change *importChange) error {
	switch change.action {
	case "create":
		return h.applyCreate(change)
	case "update":
		return h.applyUpdate(change)
	default:
		return fmt.Errorf("unknown action: %s", change.action)
	}
}

// applyCreate creates a new backlog item from import data.
func (h *Handler) applyCreate(change *importChange) error {
	cd := change.createData
	if cd == nil {
		return fmt.Errorf("no create data")
	}

	itemDir := h.itemDir(cd.kind, cd.name)
	if _, err := os.Stat(itemDir); err == nil {
		return fmt.Errorf("item already exists: %s/%s", cd.kind, cd.name)
	}

	if err := os.MkdirAll(itemDir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	tags := cd.tags
	if tags == nil {
		tags = []string{}
	}

	item := BacklogItem{
		Name:        cd.name,
		Title:       cd.title,
		Description: cd.description,
		Status:      StatusBacklog,
		Priority:    cd.priority,
		Tags:        tags,
		Created:     now,
		Updated:     now,
		Kind:        cd.kind,
	}

	if err := h.saveItem(item); err != nil {
		_ = os.RemoveAll(itemDir)
		return fmt.Errorf("failed to save item: %w", err)
	}

	log.Printf("[backlog] import: created %s/%s", cd.kind, cd.name)
	return nil
}

// applyUpdate applies changes to an existing backlog item.
func (h *Handler) applyUpdate(change *importChange) error {
	ud := change.updateData
	if ud == nil {
		return fmt.Errorf("no update data")
	}

	item := ud.item
	modified := false

	if ud.description != nil {
		item.Description = *ud.description
		modified = true
	}
	if ud.status != nil {
		item.Status = BacklogStatus(*ud.status)
		modified = true
	}
	if ud.priority != nil {
		item.Priority = *ud.priority
		modified = true
	}
	if ud.hasTags {
		item.Tags = ud.tags
		if item.Tags == nil {
			item.Tags = []string{}
		}
		modified = true
	}

	if modified {
		item.Updated = time.Now().UTC().Format(time.RFC3339)
		if err := h.saveItem(item); err != nil {
			return fmt.Errorf("failed to save item: %w", err)
		}
	}

	// Apply clarify changes.
	if len(ud.clarifyAnswers) > 0 || len(ud.clarifyNotes) > 0 {
		questionsPath := filepath.Join(h.itemDir(ud.kind, ud.name), "clarify", "questions.json")
		if err := h.applyClarifyChanges(questionsPath, ud.clarifyAnswers, ud.clarifyNotes); err != nil {
			log.Printf("[backlog] import: failed to apply clarify changes for %s/%s: %v", ud.kind, ud.name, err)
		}
	}

	// Apply suggest changes.
	if len(ud.suggestAccepted) > 0 || len(ud.suggestRejection) > 0 {
		suggestionsPath := filepath.Join(h.itemDir(ud.kind, ud.name), "suggest", "suggestions.json")
		if err := h.applySuggestChanges(suggestionsPath, ud.suggestAccepted, ud.suggestRejection); err != nil {
			log.Printf("[backlog] import: failed to apply suggest changes for %s/%s: %v", ud.kind, ud.name, err)
		}
	}

	// Apply notes changes.
	if ud.notes != "" {
		notesPath := filepath.Join(h.itemDir(ud.kind, ud.name), "notes.md")
		if err := os.WriteFile(notesPath, []byte(ud.notes+"\n"), 0o644); err != nil {
			log.Printf("[backlog] import: failed to write notes for %s/%s: %v", ud.kind, ud.name, err)
		}
	}

	log.Printf("[backlog] import: updated %s/%s (%d changes)", ud.kind, ud.name, len(change.details))
	return nil
}

// applyClarifyChanges updates questions.json with new answers and notes.
func (h *Handler) applyClarifyChanges(questionsPath string, answers map[int]string, notes map[int]string) error {
	questions, err := loadQuestions(questionsPath)
	if err != nil {
		return err
	}

	modified := false
	for idx, answer := range answers {
		if idx < len(questions) && answer != "" {
			if questions[idx].Answer != answer {
				questions[idx].Answer = answer
				modified = true
			}
		}
	}
	for idx, note := range notes {
		if idx < len(questions) && note != "" {
			if questions[idx].Notes != note {
				questions[idx].Notes = note
				modified = true
			}
		}
	}

	if !modified {
		return nil
	}

	data, err := json.MarshalIndent(questions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal questions: %w", err)
	}
	return os.WriteFile(questionsPath, data, 0o644)
}

// applySuggestChanges updates suggestions.json with acceptance and rejection data.
func (h *Handler) applySuggestChanges(suggestionsPath string, accepted map[int]bool, rejections map[int]string) error {
	suggestions, err := loadSuggestions(suggestionsPath)
	if err != nil {
		return err
	}

	modified := false
	for idx, isAccepted := range accepted {
		if idx < len(suggestions) {
			if suggestions[idx].Accepted != isAccepted {
				suggestions[idx].Accepted = isAccepted
				modified = true
			}
		}
	}
	for idx, reason := range rejections {
		if idx < len(suggestions) && reason != "" {
			if suggestions[idx].RejectionReason != reason {
				suggestions[idx].RejectionReason = reason
				suggestions[idx].Accepted = false
				modified = true
			}
		}
	}

	if !modified {
		return nil
	}

	data, err := json.MarshalIndent(suggestions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal suggestions: %w", err)
	}
	return os.WriteFile(suggestionsPath, data, 0o644)
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
