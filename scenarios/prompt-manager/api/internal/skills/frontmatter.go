package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Frontmatter is the small, runtime-neutral contract shared by all Vrooli
// skills. The metadata block is intentionally opaque here: prompt-manager's
// generated skill.json remains the compatibility index while runtimes only
// need the specification fields at the top level.
type Frontmatter struct {
	Name        string
	Description string
	License     string
	Metadata    map[string]any
	Raw         map[string]any
}

// StripFrontmatter returns the instruction body represented by a skill file.
// Native runtimes consume the envelope themselves; CLI consumers retain the
// historical body-only response contract.
func StripFrontmatter(content string) string {
	if !strings.HasPrefix(strings.TrimSpace(content), "---") {
		return content
	}
	lines := strings.SplitAfter(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return content
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(strings.TrimSuffix(lines[i], "\n")) == "---" {
			return strings.Join(lines[i+1:], "")
		}
	}
	return content
}

// ParseFrontmatter validates the Agent Skills frontmatter envelope and returns
// its required top-level fields. It deliberately rejects files that begin with
// prose, rather than treating legacy sidecars as an implicit format.
func ParseFrontmatter(path string) (Frontmatter, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Frontmatter{}, fmt.Errorf("%s: read: %w", path, err)
	}
	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return Frontmatter{}, fmt.Errorf("%s: missing YAML frontmatter opener", path)
	}
	closeAt := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			closeAt = i
			break
		}
	}
	if closeAt < 0 {
		return Frontmatter{}, fmt.Errorf("%s: unterminated YAML frontmatter", path)
	}
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:closeAt], "\n")), &raw); err != nil {
		return Frontmatter{}, fmt.Errorf("%s: invalid YAML frontmatter: %w", path, err)
	}
	fm := Frontmatter{Raw: raw}
	fm.Name, _ = raw["name"].(string)
	fm.Description, _ = raw["description"].(string)
	fm.License, _ = raw["license"].(string)
	if metadata, ok := raw["metadata"].(map[string]any); ok {
		fm.Metadata = metadata
	} else if raw["metadata"] != nil {
		return Frontmatter{}, fmt.Errorf("%s: metadata must be a mapping", path)
	}
	if fm.Name == "" || fm.Description == "" {
		return Frontmatter{}, fmt.Errorf("%s: frontmatter requires non-empty name and description", path)
	}
	if filepath.Base(filepath.Dir(path)) != fm.Name {
		return Frontmatter{}, fmt.Errorf("%s: frontmatter name %q does not match folder %q", path, fm.Name, filepath.Base(filepath.Dir(path)))
	}
	return fm, nil
}

// ValidateCorpus walks every scenario-owned and prompt-manager skill root.
// The discovered/processed comparison is intentionally set-based so adding a
// skill cannot silently bypass validation through a hard-coded count.
func ValidateCorpus(root string) error {
	// Discovery is by skill directory, not by SKILL.md. Counting the files that
	// exist can only ever prove that the files that exist are well formed; it
	// cannot notice a skill whose body was deleted or never written. A skill
	// directory is any directory holding a SKILL.md or a skill.json, so a
	// directory that has lost one of the pair is still discovered and still
	// has to answer for itself.
	discovered := make(map[string]struct{})
	if err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		switch info.Name() {
		case "SKILL.md", "skill.json":
			discovered[filepath.Dir(path)] = struct{}{}
		}
		return nil
	}); err != nil {
		return err
	}

	processed := make(map[string]struct{})
	var problems []string
	for dir := range discovered {
		skillPath := filepath.Join(dir, "SKILL.md")
		if _, err := os.Stat(skillPath); err != nil {
			problems = append(problems, fmt.Sprintf("%s: no SKILL.md beside its skill.json", dir))
			continue
		}
		if _, err := ParseFrontmatter(skillPath); err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", dir, err))
			continue
		}
		if err := validateGeneratedSidecar(skillPath); err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", dir, err))
			continue
		}
		processed[dir] = struct{}{}
	}

	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("frontmatter validation failed for %d of %d skill directories:\n  %s",
			len(problems), len(discovered), strings.Join(problems, "\n  "))
	}
	if len(discovered) != len(processed) {
		return fmt.Errorf("frontmatter validation incomplete: discovered=%d processed=%d", len(discovered), len(processed))
	}
	return nil
}

// GenerateSidecar regenerates the compatibility index fields that are owned by
// SKILL.md. The remaining sidecar fields are retained so older consumers keep
// their generated metadata until they are migrated to frontmatter directly.
func GenerateSidecar(skillPath string) error {
	fm, err := ParseFrontmatter(skillPath)
	if err != nil {
		return err
	}
	sidecarPath := filepath.Join(filepath.Dir(skillPath), "skill.json")
	data, err := os.ReadFile(sidecarPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	fields := make(map[string]any, len(fm.Metadata)+6)
	for key, value := range fm.Metadata {
		fields[key] = value
	}
	if len(data) != 0 {
		var existing map[string]any
		if err := json.Unmarshal(data, &existing); err != nil {
			return fmt.Errorf("%s: invalid sidecar: %w", sidecarPath, err)
		}
		for key, value := range existing {
			if _, generated := fields[key]; !generated {
				fields[key] = value
			}
		}
	}
	fields["id"] = fm.Name
	fields["description"] = fm.Description
	if fm.License != "" {
		fields["license"] = fm.License
	}
	if fields["name"] == nil {
		fields["name"] = fm.Name
	}
	if fields["entry"] == nil {
		fields["entry"] = "SKILL.md"
	}
	if fields["kind"] == nil {
		fields["kind"] = "skill"
	}
	if fields["schemaVersion"] == nil {
		fields["schemaVersion"] = 1
	}
	out, err := json.MarshalIndent(fields, "", "  ")
	if err != nil {
		return fmt.Errorf("%s: encode generated sidecar: %w", sidecarPath, err)
	}
	out = append(out, '\n')
	if err := os.WriteFile(sidecarPath, out, 0o644); err != nil {
		return fmt.Errorf("%s: write generated sidecar: %w", sidecarPath, err)
	}
	return nil
}

// GenerateCorpus regenerates sidecars for every skill beneath root and uses
// the live walk as its coverage boundary.
func GenerateCorpus(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || info.Name() != "SKILL.md" {
			return nil
		}
		return GenerateSidecar(path)
	})
}

func validateGeneratedSidecar(skillPath string) error {
	sidecarPath := filepath.Join(filepath.Dir(skillPath), "skill.json")
	data, err := os.ReadFile(sidecarPath)
	if os.IsNotExist(err) {
		return nil
	} // scenario-owned roots need no sidecar
	if err != nil {
		return err
	}
	var sidecar struct {
		ID          string `json:"id"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(data, &sidecar); err != nil {
		return fmt.Errorf("%s: invalid generated sidecar: %w", sidecarPath, err)
	}
	fm, err := ParseFrontmatter(skillPath)
	if err != nil {
		return err
	}
	if sidecar.ID != "" && sidecar.ID != fm.Name {
		return fmt.Errorf("%s: generated sidecar id %q does not match frontmatter %q", sidecarPath, sidecar.ID, fm.Name)
	}
	if sidecar.Description != "" && sidecar.Description != fm.Description {
		return fmt.Errorf("%s: generated sidecar description does not match frontmatter", sidecarPath)
	}
	return nil
}
