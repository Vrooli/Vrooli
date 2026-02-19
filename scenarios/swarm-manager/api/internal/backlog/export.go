// DOC: docs/reference/operational-targets.md
// DOC: docs/concepts/ARCHITECTURE.md
package backlog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/httputil"
)

// kindEmoji maps backlog kinds to their markdown emoji prefix.
var kindEmoji = map[BacklogKind]string{
	KindIdea:     "\U0001f4a1",
	KindFix:      "\U0001f527",
	KindResearch: "\U0001f52c",
	KindExecute:  "\u25b6\ufe0f",
}

// clarifyQuestion represents a single clarification question stored on disk.
type clarifyQuestion struct {
	ID         string   `json:"id"`
	Question   string   `json:"question"`
	Category   string   `json:"category"`
	Importance string   `json:"importance"`
	Options    []string `json:"options"`
	Answer     string   `json:"answer"`
	Notes      string   `json:"notes"`
}

// suggestion represents a single suggestion stored on disk.
type suggestion struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Impact          string `json:"impact"`
	Category        string `json:"category"`
	Rationale       string `json:"rationale"`
	Accepted        bool   `json:"accepted"`
	RejectionReason string `json:"rejection_reason"`
}

// defaultExportStatuses returns all non-archived statuses.
func defaultExportStatuses() map[BacklogStatus]bool {
	return map[BacklogStatus]bool{
		StatusBacklog:     true,
		StatusResearching: true,
		StatusReady:       true,
		StatusQueued:      true,
		StatusInProgress:  true,
		StatusCompleted:   true,
	}
}

// Export renders backlog items as a self-documenting markdown file with YAML frontmatter.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	var req apipb.ExportBacklogRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := httputil.DecodeProtoJSON(r, &req); err != nil {
			httputil.BadRequest(w, "[backlog] export", "invalid request body")
			return
		}
	}

	// Parse kind filters.
	var kinds []BacklogKind
	for _, raw := range req.GetKinds() {
		k, err := parseBacklogKind(raw)
		if err != nil {
			httputil.BadRequest(w, "[backlog] export", err.Error())
			return
		}
		kinds = append(kinds, k)
	}

	// Load all items matching kind filter.
	items, err := h.loadAllItems(kinds)
	if err != nil {
		httputil.InternalError(w, "[backlog] export", "failed to load backlog items")
		return
	}

	// Build status filter set.
	statusFilter := defaultExportStatuses()
	if len(req.GetStatuses()) > 0 {
		statusFilter = make(map[BacklogStatus]bool, len(req.GetStatuses()))
		for _, s := range req.GetStatuses() {
			if !validateBacklogStatus(s) {
				httputil.BadRequest(w, "[backlog] export", fmt.Sprintf("invalid status: %s", s))
				return
			}
			statusFilter[BacklogStatus(s)] = true
		}
	}

	// Build names filter set (kind/name format).
	var nameFilter map[string]bool
	if len(req.GetNames()) > 0 {
		nameFilter = make(map[string]bool, len(req.GetNames()))
		for _, n := range req.GetNames() {
			nameFilter[n] = true
		}
	}

	// Build tags filter set.
	var tagFilter map[string]bool
	if len(req.GetTags()) > 0 {
		tagFilter = make(map[string]bool, len(req.GetTags()))
		for _, t := range req.GetTags() {
			tagFilter[t] = true
		}
	}

	// Apply filters.
	var filtered []BacklogItem
	for _, item := range items {
		// Status filter.
		if !statusFilter[item.Status] {
			continue
		}

		// Names filter (kind/name format).
		if nameFilter != nil {
			key := string(item.Kind) + "/" + item.Name
			if !nameFilter[key] {
				continue
			}
		}

		// Priority max filter.
		if req.PriorityMax != nil && int32(item.Priority) > *req.PriorityMax {
			continue
		}

		// Tags filter: item must have at least one matching tag.
		if tagFilter != nil {
			found := false
			for _, t := range item.Tags {
				if tagFilter[t] {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		filtered = append(filtered, item)
	}

	// Sort by priority (ascending) then by updated (descending).
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Priority != filtered[j].Priority {
			return filtered[i].Priority < filtered[j].Priority
		}
		return filtered[i].Updated > filtered[j].Updated
	})

	// Determine include_prd (default true when nil).
	includePRD := true
	if req.IncludePrd != nil {
		includePRD = *req.IncludePrd
	}

	includeRequirements := false
	if req.IncludeRequirements != nil {
		includeRequirements = *req.IncludeRequirements
	}

	// Default true for content toggles (nil = include).
	includeClarify := req.IncludeClarifyQuestions == nil || *req.IncludeClarifyQuestions
	includeSuggestions := req.IncludeSuggestions == nil || *req.IncludeSuggestions
	includeNotes := req.IncludeNotes == nil || *req.IncludeNotes
	includeTemplate := req.IncludeTemplate == nil || *req.IncludeTemplate

	// Build the markdown document.
	now := time.Now().UTC().Format(time.RFC3339)
	var b strings.Builder

	// YAML frontmatter.
	b.WriteString("---\n")
	b.WriteString("version: 1\n")
	fmt.Fprintf(&b, "exported_at: %q\n", now)
	if len(req.GetKinds()) > 0 {
		fmt.Fprintf(&b, "filter_kinds: [%s]\n", strings.Join(req.GetKinds(), ", "))
	}
	if len(req.GetStatuses()) > 0 {
		fmt.Fprintf(&b, "filter_statuses: [%s]\n", strings.Join(req.GetStatuses(), ", "))
	}
	if len(req.GetNames()) > 0 {
		fmt.Fprintf(&b, "filter_names: [%s]\n", strings.Join(req.GetNames(), ", "))
	}
	if req.PriorityMax != nil {
		fmt.Fprintf(&b, "filter_priority_max: %d\n", *req.PriorityMax)
	}
	if len(req.GetTags()) > 0 {
		fmt.Fprintf(&b, "filter_tags: [%s]\n", strings.Join(req.GetTags(), ", "))
	}
	fmt.Fprintf(&b, "items_count: %d\n", len(filtered))
	b.WriteString("---\n\n")

	// Render each item.
	for _, item := range filtered {
		renderItem(&b, h, item, includePRD, includeRequirements, includeClarify, includeSuggestions, includeNotes)
	}

	// Append new-item template.
	if includeTemplate {
		renderNewItemTemplate(&b)
	}

	// Write response.
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"backlog-export.md\"")
	_, _ = w.Write([]byte(b.String()))
}

// renderItem writes a single backlog item as a markdown section.
func renderItem(b *strings.Builder, h *Handler, item BacklogItem, includePRD, includeRequirements, includeClarify, includeSuggestions, includeNotes bool) {
	emoji := kindEmoji[item.Kind]
	if emoji == "" {
		emoji = "\u2022"
	}

	// HTML comment marker.
	fmt.Fprintf(b, "<!-- item:%s/%s -->\n", item.Kind, item.Name)

	// Heading with emoji prefix.
	fmt.Fprintf(b, "## %s %s\n\n", emoji, item.Title)

	// Metadata table.
	b.WriteString("| Field | Value |\n")
	b.WriteString("|-------|-------|\n")
	fmt.Fprintf(b, "| **Status** | %s |\n", item.Status)
	fmt.Fprintf(b, "| **Priority** | %d |\n", item.Priority)
	if len(item.Tags) > 0 {
		fmt.Fprintf(b, "| **Tags** | %s |\n", strings.Join(item.Tags, ", "))
	} else {
		b.WriteString("| **Tags** | |\n")
	}
	fmt.Fprintf(b, "| **Created** | %s |\n", item.Created)
	fmt.Fprintf(b, "| **Updated** | %s |\n", item.Updated)
	b.WriteString("\n")

	// Description section.
	if strings.TrimSpace(item.Description) != "" {
		b.WriteString("### Description\n\n")
		b.WriteString(item.Description)
		b.WriteString("\n\n")
	}

	itemDir := h.itemDir(item.Kind, item.Name)

	// PRD content in <details> tag.
	if includePRD {
		renderPRD(b, itemDir)
	}

	// Requirements section.
	if includeRequirements {
		renderRequirements(b, itemDir)
	}

	// Clarify questions section.
	if includeClarify {
		renderClarifyQuestions(b, itemDir, item.Kind, item.Name)
	}

	// Suggestions section.
	if includeSuggestions {
		renderSuggestions(b, itemDir, item.Kind, item.Name)
	}

	// Notes section placeholder.
	if includeNotes {
		b.WriteString("### Notes\n\n")
		b.WriteString("_No notes._\n\n")
	}

	b.WriteString("---\n\n")
}

// renderPRD reads archive/PRD.md and renders it in a <details> block.
func renderPRD(b *strings.Builder, itemDir string) {
	prdPath := filepath.Join(itemDir, "archive", "PRD.md")
	data, err := os.ReadFile(prdPath)
	if err != nil {
		return
	}
	content := strings.TrimSpace(string(data))
	if content == "" {
		return
	}

	b.WriteString("<details>\n<summary>PRD</summary>\n\n")
	b.WriteString(content)
	b.WriteString("\n\n</details>\n\n")
}

// renderRequirements reads archive requirements and renders them.
func renderRequirements(b *strings.Builder, itemDir string) {
	archiveDir := filepath.Join(itemDir, "archive")
	groups, err := ParseArchiveRequirements(archiveDir)
	if err != nil || len(groups) == 0 {
		return
	}

	b.WriteString("### Requirements\n\n")
	renderRequirementGroups(b, groups, 0)
	b.WriteString("\n")
}

// renderRequirementGroups recursively renders requirement groups.
func renderRequirementGroups(b *strings.Builder, groups []ArchiveRequirementGroup, depth int) {
	indent := strings.Repeat("  ", depth)
	for _, g := range groups {
		fmt.Fprintf(b, "%s- **%s**\n", indent, g.Name)
		for _, r := range g.Requirements {
			status := " "
			if r.Status == "done" || r.Status == "completed" {
				status = "x"
			}
			fmt.Fprintf(b, "%s  - [%s] `%s` %s: %s\n", indent, status, r.ID, r.Title, r.Description)
		}
		if len(g.Children) > 0 {
			renderRequirementGroups(b, g.Children, depth+1)
		}
	}
}

// renderClarifyQuestions reads clarify/questions.json and renders with checkboxes.
func renderClarifyQuestions(b *strings.Builder, itemDir string, kind BacklogKind, name string) {
	qPath := filepath.Join(itemDir, "clarify", "questions.json")
	data, err := os.ReadFile(qPath)
	if err != nil {
		return
	}

	var questions []clarifyQuestion
	if err := json.Unmarshal(data, &questions); err != nil {
		return
	}
	if len(questions) == 0 {
		return
	}

	fmt.Fprintf(b, "<!-- clarify:%s/%s -->\n", kind, name)
	b.WriteString("### Clarify Questions\n\n")
	for i, q := range questions {
		fmt.Fprintf(b, "**Q%d: %s** (%s, %s)\n\n", i+1, q.Question, q.Category, q.Importance)

		if len(q.Options) > 0 {
			for _, opt := range q.Options {
				check := " "
				if q.Answer == opt {
					check = "x"
				}
				fmt.Fprintf(b, "- [%s] %s\n", check, opt)
			}
			// If answer is non-empty but doesn't match any option, show as freeform note.
			if q.Answer != "" && !containsOption(q.Options, q.Answer) {
				fmt.Fprintf(b, "\n> **Answer:** %s\n", q.Answer)
			}
		} else if q.Answer != "" {
			fmt.Fprintf(b, "> **Answer:** %s\n", q.Answer)
		}

		if q.Notes != "" {
			fmt.Fprintf(b, "\n> **Notes:** %s\n", q.Notes)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(b, "<!-- /clarify -->\n\n")
}

// containsOption checks if answer matches one of the provided options.
func containsOption(options []string, answer string) bool {
	for _, opt := range options {
		if opt == answer {
			return true
		}
	}
	return false
}

// renderSuggestions reads suggest/suggestions.json and renders with accept checkboxes.
func renderSuggestions(b *strings.Builder, itemDir string, kind BacklogKind, name string) {
	sPath := filepath.Join(itemDir, "suggest", "suggestions.json")
	data, err := os.ReadFile(sPath)
	if err != nil {
		return
	}

	var suggestions []suggestion
	if err := json.Unmarshal(data, &suggestions); err != nil {
		return
	}
	if len(suggestions) == 0 {
		return
	}

	fmt.Fprintf(b, "<!-- suggest:%s/%s -->\n", kind, name)
	b.WriteString("### Suggestions\n\n")
	for i, s := range suggestions {
		check := " "
		if s.Accepted {
			check = "x"
		}
		fmt.Fprintf(b, "#### S%d: %s\n", i+1, s.Title)
		fmt.Fprintf(b, "**Impact**: %s | **Category**: %s\n", s.Impact, s.Category)
		fmt.Fprintf(b, "- [%s] Accept this suggestion\n", check)
		if s.Rationale != "" {
			fmt.Fprintf(b, "  > %s\n", s.Rationale)
		}
		if s.RejectionReason != "" {
			fmt.Fprintf(b, "\n> Rejection reason: %s\n", s.RejectionReason)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(b, "<!-- /suggest -->\n\n")
}

// renderNewItemTemplate appends a template for adding new items to the export.
func renderNewItemTemplate(b *strings.Builder) {
	b.WriteString("---\n\n")
	b.WriteString("## New Item Template\n\n")
	b.WriteString("<!-- To add a new item, copy the marker below and fill in the fields. -->\n")
	b.WriteString("<!-- Marker: <!-- item:NEW --> then ## kind/name -\\- Title -->\n\n")
	b.WriteString("| Field | Value |\n")
	b.WriteString("|-------|-------|\n")
	b.WriteString("| **Status** | backlog |\n")
	b.WriteString("| **Priority** | 5 |\n")
	b.WriteString("| **Tags** | |\n")
	b.WriteString("| **Created** | |\n")
	b.WriteString("| **Updated** | |\n\n")
	b.WriteString("### Description\n\n")
	b.WriteString("_Describe the item here._\n\n")
	b.WriteString("### Notes\n\n")
	b.WriteString("_Add notes here._\n")
}
