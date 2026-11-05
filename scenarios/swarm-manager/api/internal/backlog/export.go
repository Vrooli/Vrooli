// DOC: docs/reference/operational-targets.md
// DOC: docs/concepts/ARCHITECTURE.md
package backlog

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// kindEmoji maps backlog kinds to their markdown emoji prefix.
var kindEmoji = map[BacklogKind]string{
	KindIdea:     "\U0001f4a1",
	KindFix:      "\U0001f527",
	KindResearch: "\U0001f52c",
	KindExecute:  "\u25b6\ufe0f",
	KindChore:    "\U0001f9f9",
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

// defaultExportStatuses returns all statuses included in default exports.
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
			apierr.MapError(w, "[backlog] export", apierr.BadRequest("invalid request body"))
			return
		}
	}

	// Parse kind/status/name/tag filters and load matching items.
	items, kindFilter, statusFilter, nameFilter, tagFilter, ok := h.loadExportItems(w, &req)
	if !ok {
		return
	}

	// Apply filters.
	filtered, report := applyExportFilters(items, kindFilter, statusFilter, nameFilter, tagFilter, req.PriorityMax, req.GetIncludeArchived())

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
	writeExportFrontmatter(&b, &req, now, report)

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

// loadExportItems parses kind filters, validates status filters, and loads
// all matching backlog items. Returns the loaded items and the three filter
// maps on success, or writes an error response and returns ok=false. Extracting
// this removes five branches (kind parse loop, LoadAll err, status loop with
// validate, name/tag set builds) from Export.
func (h *Handler) loadExportItems(w http.ResponseWriter, req *apipb.ExportBacklogRequest) ([]BacklogItem, map[BacklogKind]bool, map[BacklogStatus]bool, map[string]bool, map[string]bool, bool) {
	var kindFilter map[BacklogKind]bool
	if len(req.GetKinds()) > 0 {
		kindFilter = make(map[BacklogKind]bool, len(req.GetKinds()))
	}
	for _, raw := range req.GetKinds() {
		k, err := ParseBacklogKind(raw)
		if err != nil {
			apierr.MapError(w, "[backlog] export", apierr.BadRequest("%s", err.Error()))
			return nil, nil, nil, nil, nil, false
		}
		kindFilter[k] = true
	}
	items, err := h.store.LoadAll(nil)
	if err != nil {
		apierr.MapError(w, "[backlog] export", apierr.Internal("failed to load backlog items"))
		return nil, nil, nil, nil, nil, false
	}
	statusFilter := defaultExportStatuses()
	if len(req.GetStatuses()) > 0 {
		statusFilter = make(map[BacklogStatus]bool, len(req.GetStatuses()))
		for _, s := range req.GetStatuses() {
			if !validateBacklogStatus(s) {
				apierr.MapError(w, "[backlog] export", apierr.BadRequest("invalid status: %s", s))
				return nil, nil, nil, nil, nil, false
			}
			statusFilter[BacklogStatus(s)] = true
		}
	}
	nameFilter := stringSetFilter(req.GetNames())
	tagFilter := stringSetFilter(req.GetTags())
	return items, kindFilter, statusFilter, nameFilter, tagFilter, true
}

type exportExclusion struct {
	Name  string
	Rule  string
	Count int
}

type exportFilterReport struct {
	PreFilterTotal int
	ItemsCount     int
	Excluded       []exportExclusion
}

// applyExportFilters attributes every excluded item to the first filter it
// fails. The ordered, non-overlapping counts make the frontmatter arithmetic
// close exactly and ensure a new filter cannot remain invisible.
func applyExportFilters(items []BacklogItem, kindFilter map[BacklogKind]bool, statusFilter map[BacklogStatus]bool, nameFilter, tagFilter map[string]bool, priorityMax *int32, includeArchived bool) ([]BacklogItem, exportFilterReport) {
	exclusions := []exportExclusion{
		{Name: "archived", Rule: "exclude archived records"},
		{Name: "status", Rule: "include selected statuses"},
	}
	if kindFilter != nil {
		exclusions = append(exclusions, exportExclusion{Name: "kind", Rule: "include selected kinds"})
	}
	if nameFilter != nil {
		exclusions = append(exclusions, exportExclusion{Name: "name", Rule: "include selected kind/name keys"})
	}
	if priorityMax != nil {
		exclusions = append(exclusions, exportExclusion{Name: "priority_max", Rule: fmt.Sprintf("priority <= %d", *priorityMax)})
	}
	if tagFilter != nil {
		exclusions = append(exclusions, exportExclusion{Name: "tag", Rule: "include items matching any selected tag"})
	}

	filtered := make([]BacklogItem, 0, len(items))
	for _, item := range items {
		excludedBy := ""
		switch {
		case !includeArchived && item.ArchivedAt != nil:
			excludedBy = "archived"
		case !statusFilter[item.Status]:
			excludedBy = "status"
		case kindFilter != nil && !kindFilter[item.Kind]:
			excludedBy = "kind"
		case nameFilter != nil && !nameFilter[string(item.Kind)+"/"+item.Name]:
			excludedBy = "name"
		case priorityMax != nil && int32(item.Priority) > *priorityMax:
			excludedBy = "priority_max"
		case tagFilter != nil && !hasMatchingTag(item.Tags, tagFilter):
			excludedBy = "tag"
		}
		if excludedBy == "" {
			filtered = append(filtered, item)
			continue
		}
		for i := range exclusions {
			if exclusions[i].Name == excludedBy {
				exclusions[i].Count++
				break
			}
		}
	}
	if includeArchived {
		exclusions[0].Rule = "include archived records"
	}
	return filtered, exportFilterReport{PreFilterTotal: len(items), ItemsCount: len(filtered), Excluded: exclusions}
}

// writeExportFrontmatter writes the YAML frontmatter block, including any
// applied filters and the resulting item count.
func writeExportFrontmatter(b *strings.Builder, req *apipb.ExportBacklogRequest, exportedAt string, report exportFilterReport) {
	b.WriteString("---\n")
	b.WriteString("version: 1\n")
	fmt.Fprintf(b, "exported_at: %q\n", exportedAt)
	if len(req.GetKinds()) > 0 {
		fmt.Fprintf(b, "filter_kinds: [%s]\n", strings.Join(req.GetKinds(), ", "))
	}
	if len(req.GetStatuses()) > 0 {
		fmt.Fprintf(b, "filter_statuses: [%s]\n", strings.Join(req.GetStatuses(), ", "))
	}
	if len(req.GetNames()) > 0 {
		fmt.Fprintf(b, "filter_names: [%s]\n", strings.Join(req.GetNames(), ", "))
	}
	if req.PriorityMax != nil {
		fmt.Fprintf(b, "filter_priority_max: %d\n", *req.PriorityMax)
	}
	if len(req.GetTags()) > 0 {
		fmt.Fprintf(b, "filter_tags: [%s]\n", strings.Join(req.GetTags(), ", "))
	}
	fmt.Fprintf(b, "pre_filter_total: %d\n", report.PreFilterTotal)
	b.WriteString("excluded:\n")
	for _, exclusion := range report.Excluded {
		fmt.Fprintf(b, "  - filter: %q\n", exclusion.Name)
		fmt.Fprintf(b, "    rule: %q\n", exclusion.Rule)
		fmt.Fprintf(b, "    count: %d\n", exclusion.Count)
	}
	fmt.Fprintf(b, "items_count: %d\n", report.ItemsCount)
	b.WriteString("---\n\n")
}

// stringSetFilter builds a lookup set from the given values, returning nil when
// the input is empty (meaning "no filter").
func stringSetFilter(values []string) map[string]bool {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]bool, len(values))
	for _, v := range values {
		set[v] = true
	}
	return set
}

// exportItemPassesFilters reports whether an item satisfies all export filters.
// Archived items are always excluded. A nil nameFilter / tagFilter means that
// dimension is unfiltered.
func exportItemPassesFilters(item BacklogItem, statusFilter map[BacklogStatus]bool, nameFilter, tagFilter map[string]bool, priorityMax *int32) bool {
	// Exclude archived items by default.
	if item.ArchivedAt != nil {
		return false
	}
	// Status filter.
	if !statusFilter[item.Status] {
		return false
	}
	// Names filter (kind/name format).
	if nameFilter != nil && !nameFilter[string(item.Kind)+"/"+item.Name] {
		return false
	}
	// Priority max filter.
	if priorityMax != nil && int32(item.Priority) > *priorityMax {
		return false
	}
	// Tags filter: item must have at least one matching tag.
	if tagFilter != nil && !hasMatchingTag(item.Tags, tagFilter) {
		return false
	}
	return true
}

// hasMatchingTag reports whether any of the item's tags is present in the filter.
func hasMatchingTag(tags []string, tagFilter map[string]bool) bool {
	for _, t := range tags {
		if tagFilter[t] {
			return true
		}
	}
	return false
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

	itemDir := h.store.ItemDir(item.Kind, item.Name)

	// PRD content in <details> tag.
	if includePRD {
		renderPRD(b, itemDir)
	}

	// Requirements section.
	if includeRequirements {
		renderRequirements(b, itemDir)
	}

	// Notes are an editable round-trip subsection. The marker is the parser's
	// boundary; the human-facing heading stays outside it so it is not persisted
	// into notes.md.
	if includeNotes {
		b.WriteString("### Notes\n\n")
		fmt.Fprintf(b, "<!-- notes:%s/%s -->\n", item.Kind, item.Name)
		notes, err := os.ReadFile(filepath.Join(itemDir, "notes.md"))
		if err == nil && strings.TrimSpace(string(notes)) != "" {
			b.WriteString(strings.TrimSpace(string(notes)))
			b.WriteString("\n\n")
		} else {
			b.WriteString("_No notes._\n\n")
		}
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
