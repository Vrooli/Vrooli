// DOC: docs/reference/operational-targets.md
// DOC: docs/concepts/ARCHITECTURE.md
package backlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// ArchiveTarget represents an operational target parsed from a PRD archive.
type ArchiveTarget struct {
	ID                 string   `json:"id"`
	Criticality        string   `json:"criticality"`
	Title              string   `json:"title"`
	Notes              string   `json:"notes"`
	Status             string   `json:"status"`
	LinkedRequirements []string `json:"linked_requirement_ids"`
}

// ArchiveRequirement represents a single requirement within a group.
type ArchiveRequirement struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Status        string `json:"status"`
	Category      string `json:"category"`
	PRDRef        string `json:"prd_ref"`
	ReviewedAt    string `json:"reviewed_at,omitempty"`
	ReviewComment string `json:"review_comment,omitempty"`
	ReviewStatus  string `json:"review_status,omitempty"`
}

// ArchiveRequirementGroup represents a hierarchical group of requirements.
type ArchiveRequirementGroup struct {
	ID           string                    `json:"id"`
	Name         string                    `json:"name"`
	Requirements []ArchiveRequirement      `json:"requirements"`
	Children     []ArchiveRequirementGroup `json:"children"`
}

// Internal types for JSON parsing.
type requirementsFile struct {
	Metadata     map[string]any           `json:"_metadata"`
	ModuleID     string                   `json:"module_id"`
	Title        string                   `json:"title"`
	Priority     string                   `json:"priority"`
	Description  string                   `json:"description"`
	Imports      []string                 `json:"imports"`
	Requirements []requirementRecordInput `json:"requirements"`
}

type requirementRecordInput struct {
	ID            string `json:"id"`
	Category      string `json:"category"`
	PRDRef        string `json:"prd_ref"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Status        string `json:"status"`
	ReviewedAt    string `json:"reviewed_at,omitempty"`
	ReviewComment string `json:"review_comment,omitempty"`
	ReviewStatus  string `json:"review_status,omitempty"`
}

var (
	checkboxPattern = regexp.MustCompile(`^- \[(?i:x| )\]`)
	reqLinkPattern  = regexp.MustCompile("`\\[req:([A-Z0-9,-]+)\\]`")
)

const (
	modernOperationalHeader = "## 🎯 Operational Targets"
	modernP0Heading         = "### 🔴 P0 – Must ship for viability"
	modernP1Heading         = "### 🟠 P1 – Should have post-launch"
	modernP2Heading         = "### 🟢 P2 – Future / expansion"
)

// ParseArchiveTargets reads a PRD.md file from the archive directory and returns
// parsed operational targets.
func ParseArchiveTargets(archiveDir string) ([]ArchiveTarget, error) {
	prdPath := filepath.Join(archiveDir, "PRD.md")
	data, err := os.ReadFile(prdPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ArchiveTarget{}, nil
		}
		return nil, err
	}
	return parseOperationalTargets(string(data)), nil
}

// parseOperationalTargets dispatches to modern or legacy parser based on content.
func parseOperationalTargets(content string) []ArchiveTarget {
	if strings.Contains(content, modernOperationalHeader) {
		return parseModernOperationalTargets(content)
	}
	return parseLegacyOperationalTargets(content)
}

// parseModernOperationalTargets parses the modern PRD format with P0/P1/P2 headings.
func parseModernOperationalTargets(content string) []ArchiveTarget {
	lines := strings.Split(content, "\n")
	var targets []ArchiveTarget
	inSection := false
	currentCriticality := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect section start.
		if strings.HasPrefix(trimmed, modernOperationalHeader) {
			inSection = true
			continue
		}

		if !inSection {
			continue
		}

		// End of section: another ## heading.
		if strings.HasPrefix(trimmed, "## ") && trimmed != modernOperationalHeader {
			break
		}

		// Detect P0/P1/P2 sub-headings.
		switch {
		case strings.HasPrefix(trimmed, modernP0Heading):
			currentCriticality = "P0"
			continue
		case strings.HasPrefix(trimmed, modernP1Heading):
			currentCriticality = "P1"
			continue
		case strings.HasPrefix(trimmed, modernP2Heading):
			currentCriticality = "P2"
			continue
		}

		// Parse checkbox lines.
		if checkboxPattern.MatchString(trimmed) {
			target := parseModernTargetLine(trimmed, currentCriticality)
			targets = append(targets, target)
		}
	}

	if targets == nil {
		targets = []ArchiveTarget{}
	}
	return targets
}

// parseModernTargetLine parses a single modern-format target line.
// Format: - [x] ID | Title | Notes
func parseModernTargetLine(line, criticality string) ArchiveTarget {
	status := "pending"
	lower := strings.ToLower(line)
	if strings.HasPrefix(lower, "- [x]") {
		status = "complete"
	}

	// Strip the checkbox prefix.
	content := checkboxPattern.ReplaceAllString(line, "")
	content = strings.TrimSpace(content)

	// Extract requirement links before splitting.
	linkedReqs := extractReqLinks(content)
	// Remove req link markers from content for cleaner parsing.
	cleaned := reqLinkPattern.ReplaceAllString(content, "")

	parts := strings.SplitN(cleaned, "|", 3)
	id := ""
	title := ""
	notes := ""

	if len(parts) >= 1 {
		id = strings.TrimSpace(parts[0])
	}
	if len(parts) >= 2 {
		title = strings.TrimSpace(parts[1])
	}
	if len(parts) >= 3 {
		notes = strings.TrimSpace(parts[2])
	}

	return ArchiveTarget{
		ID:                 id,
		Criticality:        criticality,
		Title:              title,
		Notes:              notes,
		Status:             status,
		LinkedRequirements: linkedReqs,
	}
}

// parseLegacyOperationalTargets parses the legacy PRD format with ### Functional Requirements.
func parseLegacyOperationalTargets(content string) []ArchiveTarget {
	lines := strings.Split(content, "\n")
	var targets []ArchiveTarget
	inSection := false
	currentCriticality := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "### Functional Requirements") {
			inSection = true
			continue
		}

		if !inSection {
			continue
		}

		// End of section: another ### or ## heading.
		if (strings.HasPrefix(trimmed, "### ") && !strings.HasPrefix(trimmed, "### Functional Requirements")) ||
			strings.HasPrefix(trimmed, "## ") {
			break
		}

		// Bold category lines: - **Category (P0)**
		if strings.HasPrefix(trimmed, "- **") && strings.Contains(trimmed, "**") {
			_, crit := parseCategoryLabel(trimmed)
			if crit != "" {
				currentCriticality = crit
			}
			continue
		}

		// Checkbox lines are targets.
		if checkboxPattern.MatchString(trimmed) {
			status := "pending"
			lower := strings.ToLower(trimmed)
			if strings.HasPrefix(lower, "- [x]") {
				status = "complete"
			}

			afterCheckbox := checkboxPattern.ReplaceAllString(trimmed, "")
			afterCheckbox = strings.TrimSpace(afterCheckbox)

			linkedReqs := extractReqLinks(afterCheckbox)

			title, notes := splitLegacyTargetLine(afterCheckbox)

			targets = append(targets, ArchiveTarget{
				ID:                 slugify(title),
				Criticality:        currentCriticality,
				Title:              title,
				Notes:              notes,
				Status:             status,
				LinkedRequirements: linkedReqs,
			})
		}
	}

	if targets == nil {
		targets = []ArchiveTarget{}
	}
	return targets
}

// extractReqLinks finds all `[req:XXX]` markers in the text.
func extractReqLinks(text string) []string {
	matches := reqLinkPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return []string{}
	}
	var links []string
	for _, m := range matches {
		if len(m) >= 2 {
			// Split on comma for multi-ref like [req:REQ-001,REQ-002]
			parts := strings.Split(m[1], ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					links = append(links, p)
				}
			}
		}
	}
	if links == nil {
		links = []string{}
	}
	return links
}

// splitLegacyTargetLine extracts title and notes from a legacy checkbox line.
// Notes are typically in _(notes)_ format.
func splitLegacyTargetLine(line string) (string, string) {
	// Remove req link patterns for cleaner parsing.
	cleaned := reqLinkPattern.ReplaceAllString(line, "")
	cleaned = strings.TrimSpace(cleaned)

	// Look for _(notes)_ pattern.
	if idx := strings.Index(cleaned, "_("); idx >= 0 {
		title := strings.TrimSpace(cleaned[:idx])
		notesPart := cleaned[idx:]
		notesPart = strings.TrimPrefix(notesPart, "_(")
		notesPart = strings.TrimSuffix(notesPart, ")_")
		notesPart = strings.TrimSpace(notesPart)
		return title, notesPart
	}

	return cleaned, ""
}

// parseCategoryLabel extracts the label and criticality from a bold category line.
// Input format: - **Category (P0)**
func parseCategoryLabel(input string) (string, string) {
	// Strip leading "- " and bold markers.
	s := strings.TrimPrefix(strings.TrimSpace(input), "- ")
	s = strings.TrimPrefix(s, "**")
	s = strings.TrimSuffix(s, "**")
	s = strings.TrimSpace(s)

	// Extract criticality from parenthesized suffix.
	if idx := strings.LastIndex(s, "("); idx >= 0 {
		label := strings.TrimSpace(s[:idx])
		crit := strings.TrimSpace(s[idx+1:])
		crit = strings.TrimSuffix(crit, ")")
		crit = strings.TrimSpace(crit)
		return label, crit
	}

	return s, ""
}

// slugify creates a URL-safe identifier from the given string.
func slugify(value string) string {
	s := strings.ToLower(value)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	result := b.String()
	// Trim leading/trailing dashes.
	result = strings.Trim(result, "-")
	return result
}

// ParseArchiveRequirements reads requirement JSON files from the archive's
// requirements directory and returns a hierarchical group structure.
func ParseArchiveRequirements(archiveDir string) ([]ArchiveRequirementGroup, error) {
	reqDir := filepath.Join(archiveDir, "requirements")
	indexPath := filepath.Join(reqDir, "index.json")

	if _, err := os.Stat(indexPath); err != nil {
		if os.IsNotExist(err) {
			return []ArchiveRequirementGroup{}, nil
		}
		return nil, err
	}

	visited := make(map[string]bool)
	groups, err := parseRequirementGroups(reqDir, "index.json", visited)
	if err != nil {
		return nil, err
	}

	if groups == nil {
		groups = []ArchiveRequirementGroup{}
	}
	return groups, nil
}

// parseRequirementGroups recursively parses a requirements JSON file and its imports.
func parseRequirementGroups(baseDir, relPath string, visited map[string]bool) ([]ArchiveRequirementGroup, error) {
	fullPath := filepath.Join(baseDir, relPath)

	// Prevent cycles.
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, err
	}
	if visited[absPath] {
		return nil, nil
	}
	visited[absPath] = true

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var rf requirementsFile
	if err := json.Unmarshal(data, &rf); err != nil {
		return nil, err
	}

	groupID := slugify(strings.TrimSuffix(relPath, filepath.Ext(relPath)))
	groupName := groupNameFromPath(relPath, filepath.Base(fullPath))

	reqs := make([]ArchiveRequirement, 0, len(rf.Requirements))
	for _, r := range rf.Requirements {
		reqs = append(reqs, ArchiveRequirement{
			ID:            r.ID,
			Title:         r.Title,
			Description:   r.Description,
			Status:        r.Status,
			Category:      r.Category,
			PRDRef:        r.PRDRef,
			ReviewedAt:    r.ReviewedAt,
			ReviewComment: r.ReviewComment,
			ReviewStatus:  r.ReviewStatus,
		})
	}

	var children []ArchiveRequirementGroup
	for _, imp := range rf.Imports {
		// Imports are relative to baseDir.
		childGroups, err := parseRequirementGroups(baseDir, imp, visited)
		if err != nil {
			continue
		}
		children = append(children, childGroups...)
	}
	if children == nil {
		children = []ArchiveRequirementGroup{}
	}

	group := ArchiveRequirementGroup{
		ID:           groupID,
		Name:         groupName,
		Requirements: reqs,
		Children:     children,
	}

	return []ArchiveRequirementGroup{group}, nil
}

// groupNameFromPath derives a human-readable group name from the file path.
func groupNameFromPath(relPath, fileName string) string {
	dir := filepath.Dir(relPath)
	if dir == "." || dir == "" {
		// Root file: use filename without extension.
		name := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		return cleanFolderName(name)
	}

	// Use the deepest directory name.
	parts := strings.Split(filepath.ToSlash(dir), "/")
	deepest := parts[len(parts)-1]
	return cleanFolderName(deepest)
}

// cleanFolderName removes numeric prefixes and converts kebab-case to title case.
// e.g., "01-core" -> "Core", "02-web-server" -> "Web Server"
func cleanFolderName(folderName string) string {
	// Remove leading numeric prefix (e.g., "01-").
	name := folderName
	if idx := strings.Index(name, "-"); idx > 0 && isNumeric(name[:idx]) {
		name = name[idx+1:]
	}

	// Convert kebab-case to title case.
	words := strings.Split(name, "-")
	for i, w := range words {
		if w == "" {
			continue
		}
		runes := []rune(w)
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}

	return strings.Join(words, " ")
}

// isNumeric returns true if all characters in s are digits.
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
