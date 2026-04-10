// DOC: docs/reference/operational-targets.md
// DOC: docs/concepts/ARCHITECTURE.md
package backlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

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
