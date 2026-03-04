package backlog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// serializeTargetsSection rebuilds the entire "## 🎯 Operational Targets"
// markdown section from the given targets, grouped by criticality (P0/P1/P2).
func serializeTargetsSection(targets []ArchiveTarget) string {
	groups := map[string][]ArchiveTarget{
		"P0": {},
		"P1": {},
		"P2": {},
	}
	for _, t := range targets {
		key := strings.ToUpper(strings.TrimSpace(t.Criticality))
		if key == "" {
			key = "P2"
		}
		groups[key] = append(groups[key], t)
	}

	var b strings.Builder
	b.WriteString(modernOperationalHeader)
	b.WriteString("\n\n")

	order := []struct {
		heading     string
		criticality string
	}{
		{modernP0Heading, "P0"},
		{modernP1Heading, "P1"},
		{modernP2Heading, "P2"},
	}

	for idx, entry := range order {
		b.WriteString(entry.heading)
		b.WriteString("\n")
		for _, t := range groups[entry.criticality] {
			status := " "
			if strings.EqualFold(t.Status, "complete") {
				status = "x"
			}
			id := t.ID
			if id == "" {
				id = slugify(t.Title)
			}
			title := t.Title
			if title == "" {
				title = "Target"
			}

			line := fmt.Sprintf("- [%s] %s | %s", status, id, title)
			if t.Notes != "" {
				line += " | " + t.Notes
			}
			if len(t.LinkedRequirements) > 0 {
				line += " `[req:" + strings.Join(t.LinkedRequirements, ",") + "]`"
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
		if idx < len(order)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// replaceTargetsSection finds the "## 🎯 Operational Targets" section in PRD
// content and replaces it with the given replacement text. If the section does
// not exist, it is appended.
func replaceTargetsSection(content, replacement string) string {
	lines := strings.Split(content, "\n")
	var result []string
	replaced := false

	for i := 0; i < len(lines); i++ {
		if strings.EqualFold(strings.TrimSpace(lines[i]), modernOperationalHeader) {
			replaced = true
			result = append(result, strings.TrimRight(replacement, "\n"))

			// Skip past the old section until next ## heading or EOF.
			for j := i + 1; j < len(lines); j++ {
				trimmed := strings.TrimSpace(lines[j])
				if strings.HasPrefix(trimmed, "## ") && !strings.EqualFold(trimmed, modernOperationalHeader) {
					i = j - 1
					break
				}
				if j == len(lines)-1 {
					i = j
				}
			}
			continue
		}
		result = append(result, lines[i])
	}

	if !replaced {
		if strings.TrimSpace(content) == "" {
			return strings.TrimRight(replacement, "\n") + "\n"
		}
		return strings.TrimRight(content, "\n") + "\n\n" + strings.TrimRight(replacement, "\n") + "\n"
	}

	return strings.Join(result, "\n")
}

// resolvePRDPath returns the path to PRD.md, checking archive/ first then item root.
func resolvePRDPath(itemDir string) string {
	archivePRD := filepath.Join(itemDir, "archive", "PRD.md")
	if _, err := os.Stat(archivePRD); err == nil {
		return archivePRD
	}
	return filepath.Join(itemDir, "PRD.md")
}

// ReadTargetsFromPRD reads PRD.md, parses targets, and returns the targets
// along with the full file content (for later write-back).
func ReadTargetsFromPRD(itemDir string) ([]ArchiveTarget, string, error) {
	prdPath := resolvePRDPath(itemDir)
	data, err := os.ReadFile(prdPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ArchiveTarget{}, "", nil
		}
		return nil, "", err
	}
	content := string(data)
	targets := parseOperationalTargets(content)
	return targets, content, nil
}

// WriteTargets reads PRD.md, replaces the targets section, and writes back atomically.
func WriteTargets(itemDir string, targets []ArchiveTarget) error {
	prdPath := resolvePRDPath(itemDir)
	content := ""
	data, err := os.ReadFile(prdPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read PRD.md: %w", err)
	}
	if err == nil {
		content = string(data)
	}

	section := serializeTargetsSection(targets)
	updated := replaceTargetsSection(content, section)
	return writeFileAtomic(prdPath, updated)
}

// CreateTarget appends a new target to the existing targets and writes back.
func CreateTarget(itemDir string, target ArchiveTarget) error {
	targets, _, err := ReadTargetsFromPRD(itemDir)
	if err != nil {
		return fmt.Errorf("read targets: %w", err)
	}

	if target.ID == "" {
		target.ID = slugify(target.Title)
	}

	// Check for duplicate ID.
	for _, t := range targets {
		if t.ID == target.ID {
			return fmt.Errorf("target with ID %q already exists", target.ID)
		}
	}

	targets = append(targets, target)
	return WriteTargets(itemDir, targets)
}

// UpdateTarget finds a target by ID and replaces it.
func UpdateTarget(itemDir string, targetID string, target ArchiveTarget) error {
	targets, _, err := ReadTargetsFromPRD(itemDir)
	if err != nil {
		return fmt.Errorf("read targets: %w", err)
	}

	found := false
	for i, t := range targets {
		if t.ID == targetID {
			// Preserve the original ID.
			target.ID = targetID
			targets[i] = target
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("target %q not found", targetID)
	}

	return WriteTargets(itemDir, targets)
}

// DeleteTarget removes a target by ID.
func DeleteTarget(itemDir string, targetID string) error {
	targets, _, err := ReadTargetsFromPRD(itemDir)
	if err != nil {
		return fmt.Errorf("read targets: %w", err)
	}

	found := false
	filtered := make([]ArchiveTarget, 0, len(targets))
	for _, t := range targets {
		if t.ID == targetID {
			found = true
			continue
		}
		filtered = append(filtered, t)
	}
	if !found {
		return fmt.Errorf("target %q not found", targetID)
	}

	return WriteTargets(itemDir, filtered)
}

// writeFileAtomic writes content to a file atomically via temp+rename.
func writeFileAtomic(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, "tmp-*.md")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp to target: %w", err)
	}
	return nil
}
