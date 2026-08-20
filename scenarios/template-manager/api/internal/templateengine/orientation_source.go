package templateengine

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	templatecontracts "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

func validateTemplateSource(info templatecontracts.TemplateInfo) []templatecontracts.TemplateValidationIssue {
	if info.Missing {
		return nil
	}
	var issues []templatecontracts.TemplateValidationIssue
	issues = append(issues, validateOrientationSource(info)...)
	_ = filepath.WalkDir(info.Path, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Base(path) != "go.mod" {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			issues = append(issues, templatecontracts.TemplateValidationIssue{
				Template: info.Name,
				Path:     filepath.ToSlash(filepath.Base(path)),
				Message:  fmt.Sprintf("read go.mod: %v", readErr),
			})
			return nil
		}
		for _, target := range parseLocalReplaceTargets(string(data)) {
			if strings.Contains(target, "{{") {
				continue
			}
			rel, relErr := filepath.Rel(info.Path, path)
			if relErr != nil {
				rel = path
			}
			issues = append(issues, templatecontracts.TemplateValidationIssue{
				Template: info.Name,
				Path:     filepath.ToSlash(rel),
				Message:  fmt.Sprintf("go.mod local replace target %q must use generator-computed placeholders", target),
			})
		}
		return nil
	})
	return issues
}

func validateOrientationSource(info templatecontracts.TemplateInfo) []templatecontracts.TemplateValidationIssue {
	if info.Manifest.Orientation == nil {
		return nil
	}
	orientation := info.Manifest.Orientation
	var issues []templatecontracts.TemplateValidationIssue
	add := func(path, message string) {
		issues = append(issues, templatecontracts.TemplateValidationIssue{Template: info.Name, Path: path, Message: message})
	}
	if strings.TrimSpace(info.Manifest.Version) == "" {
		add("template.json", "version is required when orientation is declared")
	}
	validateOrientationPaths(orientation, info.Manifest.StartDocument, add)
	seen := map[string]struct{}{}
	for index, step := range orientation.Steps {
		validateOrientationStepSource(index, step, seen, add)
	}
	validateOrientationCleanupSource(orientation.Finalize.Cleanup, add)
	return issues
}

type orientationSourceIssue func(path, message string)

func validateOrientationPaths(orientation *templatecontracts.TemplateOrientation, manifestStartDocument string, add orientationSourceIssue) {
	if _, err := cleanScenarioRelativePath(orientation.CopyTo); err != nil {
		add("orientation.copyTo", err.Error())
	}
	startDocument := strings.TrimSpace(orientation.StartDocument)
	if startDocument == "" {
		startDocument = strings.TrimSpace(manifestStartDocument)
	}
	if startDocument == "" {
		return
	}
	if _, err := cleanScenarioRelativePath(startDocument); err != nil {
		add("orientation.startDocument", err.Error())
	}
}

func validateOrientationStepSource(index int, step templatecontracts.TemplateOrientationStep, seen map[string]struct{}, add orientationSourceIssue) {
	stepPath := fmt.Sprintf("orientation.steps[%d]", index)
	id := strings.TrimSpace(step.ID)
	if id == "" {
		add(stepPath, "step id is required")
	} else if _, ok := seen[id]; ok {
		add(stepPath, fmt.Sprintf("duplicate step id %q", id))
	}
	seen[id] = struct{}{}
	if orientationStepRequired(step) && len(step.Checks) == 0 {
		add(stepPath, "required step must declare at least one check")
	}
	for checkIndex, check := range step.Checks {
		validateOrientationCheckSource(fmt.Sprintf("%s.checks[%d]", stepPath, checkIndex), check, add)
	}
}

func validateOrientationCheckSource(checkPath string, check templatecontracts.TemplateOrientationCheck, add orientationSourceIssue) {
	if !validOrientationCheckKind(check.Kind) {
		add(checkPath, fmt.Sprintf("unknown check kind %q", check.Kind))
	}
	switch check.Kind {
	case "file_exists", "file_absent", "directory_exists":
		validateOrientationCheckPath(checkPath, check.Path, add)
	case "glob_present", "glob_absent", "glob_min_count":
		validateOrientationGlobCheckSource(checkPath, check, add)
	case "json_path_exists", "json_min_entries":
		validateOrientationJSONCheckSource(checkPath, check, add)
	case "text_contains", "text_absent":
		validateOrientationCheckPath(checkPath, check.Path, add)
		validateOrientationRequiredText(checkPath, check.Text, add)
	case "text_absent_tree":
		validateOrientationRequiredText(checkPath, check.Text, add)
	case "content_adapted":
		validateOrientationContentAdaptedSource(checkPath, check, add)
	case "command":
		validateOrientationCommandSource(checkPath, check, add)
	}
}

func validateOrientationContentAdaptedSource(checkPath string, check templatecontracts.TemplateOrientationCheck, add orientationSourceIssue) {
	validateOrientationCheckPath(checkPath, check.Path, add)
	if check.MinCount < 0 {
		add(checkPath+".minCount", "minCount must not be negative")
	}
}

func validateOrientationCheckPath(checkPath, path string, add orientationSourceIssue) {
	if _, err := cleanScenarioRelativePath(path); err != nil {
		add(checkPath+".path", err.Error())
	}
}

func validateOrientationGlobCheckSource(checkPath string, check templatecontracts.TemplateOrientationCheck, add orientationSourceIssue) {
	if strings.TrimSpace(check.Pattern) == "" {
		add(checkPath+".pattern", "pattern is required")
	} else if _, err := cleanScenarioRelativePath(check.Pattern); err != nil {
		add(checkPath+".pattern", err.Error())
	}
	if check.Kind == "glob_min_count" && check.MinCount < 1 {
		add(checkPath+".minCount", "minCount must be greater than zero")
	}
}

func validateOrientationJSONCheckSource(checkPath string, check templatecontracts.TemplateOrientationCheck, add orientationSourceIssue) {
	validateOrientationCheckPath(checkPath, check.Path, add)
	if strings.TrimSpace(check.Query) == "" {
		add(checkPath+".query", "query is required")
	}
	if check.Kind == "json_min_entries" && check.MinCount < 1 {
		add(checkPath+".minCount", "minCount must be greater than zero")
	}
}

func validateOrientationRequiredText(checkPath, text string, add orientationSourceIssue) {
	if strings.TrimSpace(text) == "" {
		add(checkPath+".text", "text is required")
	}
}

func validateOrientationCommandSource(checkPath string, check templatecontracts.TemplateOrientationCheck, add orientationSourceIssue) {
	if len(check.Exec) == 0 || strings.TrimSpace(check.Exec[0]) == "" {
		add(checkPath+".exec", "exec requires an executable")
	}
	if strings.TrimSpace(check.Timeout) == "" {
		add(checkPath+".timeout", "timeout is required")
		return
	}
	if _, err := time.ParseDuration(check.Timeout); err != nil {
		add(checkPath+".timeout", fmt.Sprintf("invalid timeout: %v", err))
	}
}

func validateOrientationCleanupSource(cleanups []string, add orientationSourceIssue) {
	for _, cleanup := range cleanups {
		clean, err := cleanScenarioRelativePath(cleanup)
		if err != nil {
			add("orientation.finalize.cleanup", err.Error())
			continue
		}
		if isDurableOrientationCleanupTarget(clean) {
			add("orientation.finalize.cleanup", fmt.Sprintf("cleanup path %q targets durable scenario content", cleanup))
		}
	}
}

func orientationStepRequired(step templatecontracts.TemplateOrientationStep) bool {
	return step.Required == nil || *step.Required
}

func validOrientationCheckKind(kind string) bool {
	switch strings.TrimSpace(kind) {
	case "file_exists", "file_absent", "directory_exists", "glob_present", "glob_absent", "glob_min_count", "json_path_exists", "json_min_entries", "text_contains", "text_absent", "text_absent_tree", "content_adapted", "command":
		return true
	default:
		return false
	}
}

func cleanScenarioRelativePath(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("path is required")
	}
	clean := filepath.Clean(filepath.FromSlash(trimmed))
	if clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%q must be a scenario-relative path", value)
	}
	return clean, nil
}

func isDurableOrientationCleanupTarget(path string) bool {
	slash := filepath.ToSlash(path)
	if slash == "docs" || strings.HasPrefix(slash, "docs/") || slash == "DESIGN.md" || slash == "requirements" || strings.HasPrefix(slash, "requirements/") {
		return true
	}
	for _, prefix := range []string{"api/", "cli/", "ui/", "proto/", "runtime/"} {
		if strings.HasPrefix(slash, prefix) {
			return true
		}
	}
	return false
}
