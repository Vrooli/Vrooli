package templateengine

import (
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
