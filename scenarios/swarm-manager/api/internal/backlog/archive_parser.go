// DOC: docs/reference/operational-targets.md
// DOC: docs/concepts/ARCHITECTURE.md
package backlog

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
