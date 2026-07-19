package templateengine

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vrooli/vrooli/internal/scenarioexec"
	templatecontracts "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

func validateGeneratedScenario(destination string, runCommands bool, run func(scenarioexec.SubprocessSpec) error, templateName string, manifest templatecontracts.TemplateManifest) []templatecontracts.TemplateValidationIssue {
	var issues []templatecontracts.TemplateValidationIssue
	issues = append(issues, validateGeneratedStartDocument(destination, templateName, manifest)...)
	issues = append(issues, validateGeneratedExperienceFoundation(destination, templateName)...)
	_ = filepath.WalkDir(destination, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Base(path) != "go.mod" {
			return err
		}
		moduleDir := filepath.Dir(path)
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			issues = append(issues, templatecontracts.TemplateValidationIssue{
				Template: templateName,
				Path:     filepath.ToSlash(strings.TrimPrefix(path, destination+string(filepath.Separator))),
				Message:  fmt.Sprintf("read generated go.mod: %v", readErr),
			})
			return nil
		}
		for _, target := range parseLocalReplaceTargets(string(data)) {
			resolved := target
			if !filepath.IsAbs(resolved) {
				resolved = filepath.Join(moduleDir, filepath.FromSlash(target))
			}
			if _, statErr := os.Stat(filepath.Clean(resolved)); statErr != nil {
				issues = append(issues, templatecontracts.TemplateValidationIssue{
					Template: templateName,
					Path:     filepath.ToSlash(strings.TrimPrefix(path, destination+string(filepath.Separator))),
					Message:  fmt.Sprintf("go.mod replace target %q does not resolve from generated module: %v", target, statErr),
				})
			}
		}
		if runCommands && moduleHasGoFiles(moduleDir) {
			if execErr := run(scenarioexec.SubprocessSpec{
				Name: "bash",
				Args: []string{"-lc", "GOWORK=off go mod tidy"},
				Dir:  moduleDir,
			}); execErr != nil {
				issues = append(issues, templatecontracts.TemplateValidationIssue{
					Template: templateName,
					Path:     filepath.ToSlash(strings.TrimPrefix(path, destination+string(filepath.Separator))),
					Message:  fmt.Sprintf("generated module validation failed: %v", execErr),
				})
			}
		}
		return nil
	})
	return issues
}

// validateGeneratedExperienceFoundation keeps the React/Vite starter's
// authored experience contract from silently disappearing during template
// evolution. Other templates may adopt the foundation independently; this
// check is deliberately scoped to the template that ships it today.
func validateGeneratedExperienceFoundation(destination, templateName string) []templatecontracts.TemplateValidationIssue {
	if templateName != "react-vite" {
		return nil
	}
	path := filepath.Join(destination, "experience", "index.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return []templatecontracts.TemplateValidationIssue{{Template: templateName, Path: "experience/index.json", Message: fmt.Sprintf("generated scenario is missing the required experience foundation: %v", err)}}
	}
	var index struct {
		Kind     string `json:"kind"`
		Contract struct {
			Kind   string `json:"kind"`
			Schema string `json:"schema"`
		} `json:"contract"`
	}
	if err := json.Unmarshal(raw, &index); err != nil || index.Kind != "experience-index" || index.Contract.Kind != "scenario-experience" || index.Contract.Schema != "scenario-experience-spec/v1" {
		return []templatecontracts.TemplateValidationIssue{{Template: templateName, Path: "experience/index.json", Message: "generated experience foundation must be a scenario-experience-spec/v1 experience-index"}}
	}
	primitive := filepath.Join(destination, "ui", "src", "components", "experience", "ExperienceSurface.tsx")
	if _, err := os.Stat(primitive); err != nil {
		return []templatecontracts.TemplateValidationIssue{{Template: templateName, Path: "ui/src/components/experience/ExperienceSurface.tsx", Message: "generated React/Vite scenario is missing the semantic experience surface foundation"}}
	}
	notesPath := filepath.Join(destination, "experience", "pages", "notes.json")
	notesRaw, err := os.ReadFile(notesPath)
	compactNotes := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == '\t' || r == '\r' {
			return -1
		}
		return r
	}, string(notesRaw))
	if err != nil || !strings.Contains(compactNotes, `"id":"notes"`) || !strings.Contains(compactNotes, `"component":"experience-surface"`) || !strings.Contains(compactNotes, `"kind":"async"`) || !strings.Contains(compactNotes, `"testid":"notes-surface"`) {
		return []templatecontracts.TemplateValidationIssue{{Template: templateName, Path: "experience/pages/notes.json", Message: "generated notes example must declare a bound async semantic experience region"}}
	}
	return nil
}

func validateGeneratedStartDocument(destination, templateName string, manifest templatecontracts.TemplateManifest) []templatecontracts.TemplateValidationIssue {
	startDocument := strings.TrimSpace(manifest.StartDocument)
	if startDocument == "" {
		return nil
	}
	cleanPath := filepath.Clean(filepath.FromSlash(startDocument))
	if cleanPath == "." || filepath.IsAbs(cleanPath) || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) || cleanPath == ".." {
		return []templatecontracts.TemplateValidationIssue{{
			Template: templateName,
			Path:     startDocument,
			Message:  "startDocument must be a scenario-relative path",
		}}
	}
	stat, err := os.Stat(filepath.Join(destination, cleanPath))
	if err != nil {
		return []templatecontracts.TemplateValidationIssue{{
			Template: templateName,
			Path:     filepath.ToSlash(cleanPath),
			Message:  fmt.Sprintf("startDocument is declared but missing from generated scenario: %v", err),
		}}
	}
	if stat.IsDir() {
		return []templatecontracts.TemplateValidationIssue{{
			Template: templateName,
			Path:     filepath.ToSlash(cleanPath),
			Message:  "startDocument must point to a file, not a directory",
		}}
	}
	return nil
}

var goModReplaceLinePattern = regexp.MustCompile(`^\s*([A-Za-z0-9._/\-{}]+)(?:\s+[^\s]+)?\s*=>\s*([^\s]+)(?:\s+[^\s]+)?\s*(?://.*)?$`)

func parseLocalReplaceTargets(content string) []string {
	var targets []string
	var inReplaceBlock bool
	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		switch line {
		case "replace (":
			inReplaceBlock = true
			continue
		case ")":
			inReplaceBlock = false
			continue
		}
		switch {
		case strings.HasPrefix(line, "replace "):
			if target, ok := parseGoReplaceTarget(strings.TrimSpace(strings.TrimPrefix(line, "replace "))); ok {
				targets = append(targets, target)
			}
		case inReplaceBlock:
			if target, ok := parseGoReplaceTarget(line); ok {
				targets = append(targets, target)
			}
		}
	}
	return targets
}

func parseGoReplaceTarget(line string) (string, bool) {
	matches := goModReplaceLinePattern.FindStringSubmatch(line)
	if len(matches) != 3 {
		return "", false
	}
	target := strings.TrimSpace(matches[2])
	if target == "" {
		return "", false
	}
	if !(strings.HasPrefix(target, ".") || strings.HasPrefix(target, "/") || strings.Contains(target, "{{")) {
		return "", false
	}
	return target, true
}

func moduleHasGoFiles(moduleDir string) bool {
	entries, err := os.ReadDir(moduleDir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".go") {
			return true
		}
	}
	return false
}
