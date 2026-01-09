package server

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"app-issue-tracker-api/internal/utils"
)

const (
	promptTemplatePath = "prompts/unified-resolver.md"
	fallbackValue      = "(not provided)"
)

// PromptBuilder prepares agent investigation prompts and related context.
type PromptBuilder struct {
	scenarioRoot string
	readFile     func(string) ([]byte, error)
}

func NewPromptBuilder(scenarioRoot string) *PromptBuilder {
	return &PromptBuilder{
		scenarioRoot: scenarioRoot,
		readFile:     os.ReadFile,
	}
}

func (pb *PromptBuilder) LoadTemplate() string {
	templatePath := filepath.Join(pb.scenarioRoot, promptTemplatePath)
	data, err := pb.readFile(templatePath)
	if err == nil {
		trimmed := strings.TrimSpace(string(data))
		if trimmed != "" {
			return string(data)
		}
	}
	return "You are an elite software engineer. Provide investigation findings based on the supplied metadata."
}

func (pb *PromptBuilder) BuildPrompt(issue *Issue, issueDir, agentID, projectPath, timestamp, template string) string {
	tmpl := strings.TrimSpace(template)
	if tmpl == "" {
		tmpl = pb.LoadTemplate()
	}

	replacements := map[string]string{
		"{{issue_id}}":           fallbackValue,
		"{{issue_title}}":        fallbackValue,
		"{{issue_description}}":  fallbackValue,
		"{{issue_type}}":         fallbackValue,
		"{{issue_priority}}":     fallbackValue,
		"{{app_name}}":           fallbackValue, // Deprecated but kept for backward compat
		"{{targets}}":            fallbackValue, // NEW
		"{{scenarios}}":          fallbackValue, // NEW
		"{{resources}}":          fallbackValue, // NEW
		"{{error_message}}":      fallbackValue,
		"{{stack_trace}}":        fallbackValue,
		"{{affected_files}}":     fallbackValue,
		"{{issue_metadata}}":     fallbackValue,
		"{{issue_artifacts}}":    fallbackValue,
		"{{issue_dir}}":          fallbackValue,
		"{{issue_dir_absolute}}": fallbackValue,
		"{{agent_id}}":           fallbackValue,
		"{{project_path}}":       fallbackValue,
		"{{timestamp}}":          fallbackValue,
	}

	if issue != nil {
		replacements["{{issue_id}}"] = utils.ValueOrFallback(issue.ID, fallbackValue)
		replacements["{{issue_title}}"] = utils.ValueOrFallback(issue.Title, fallbackValue)
		replacements["{{issue_description}}"] = utils.ValueOrFallback(issue.Description, fallbackValue)
		replacements["{{issue_type}}"] = utils.ValueOrFallback(issue.Type, fallbackValue)
		replacements["{{issue_priority}}"] = utils.ValueOrFallback(issue.Priority, fallbackValue)

		// NEW: Build target strings
		if len(issue.Targets) > 0 {
			var targetStrs, scenarioStrs, resourceStrs []string

			for _, t := range issue.Targets {
				targetStrs = append(targetStrs, fmt.Sprintf("%s:%s", t.Type, t.ID))

				if t.Type == "scenario" {
					scenarioStrs = append(scenarioStrs, t.ID)
				} else if t.Type == "resource" {
					resourceStrs = append(resourceStrs, t.ID)
				}
			}

			replacements["{{targets}}"] = strings.Join(targetStrs, ", ")
			replacements["{{scenarios}}"] = utils.ValueOrFallback(strings.Join(scenarioStrs, ", "), fallbackValue)
			replacements["{{resources}}"] = utils.ValueOrFallback(strings.Join(resourceStrs, ", "), fallbackValue)

			// Backward compat: use first scenario or first target as app_name
			if len(scenarioStrs) > 0 {
				replacements["{{app_name}}"] = scenarioStrs[0]
			} else if len(issue.Targets) > 0 {
				replacements["{{app_name}}"] = issue.Targets[0].ID
			} else {
				replacements["{{app_name}}"] = fallbackValue
			}
		} else {
			replacements["{{app_name}}"] = fallbackValue
		}

		replacements["{{error_message}}"] = utils.ValueOrFallback(issue.ErrorContext.ErrorMessage, fallbackValue)
		replacements["{{stack_trace}}"] = utils.ValueOrFallback(issue.ErrorContext.StackTrace, fallbackValue)
		replacements["{{affected_files}}"] = utils.ValueOrFallback(joinList(issue.ErrorContext.AffectedFiles), fallbackValue)
	}

	replacements["{{issue_metadata}}"] = utils.ValueOrFallback(pb.readIssueMetadataRaw(issueDir, issue), fallbackValue)
	replacements["{{issue_artifacts}}"] = utils.ValueOrFallback(pb.listArtifactPaths(issueDir, issue), fallbackValue)
	replacements["{{issue_dir}}"] = utils.ValueOrFallback(pb.computeIssueDirRelativePath(issueDir), fallbackValue)
	replacements["{{issue_dir_absolute}}"] = utils.ValueOrFallback(issueDir, fallbackValue)
	replacements["{{agent_id}}"] = utils.ValueOrFallback(agentID, fallbackValue)
	replacements["{{project_path}}"] = utils.ValueOrFallback(projectPath, fallbackValue)
	replacements["{{timestamp}}"] = utils.ValueOrFallback(timestamp, fallbackValue)

	// Build replacement pairs for strings.NewReplacer (more efficient for multiple replacements)
	pairs := make([]string, 0, len(replacements)*2)
	for placeholder, value := range replacements {
		pairs = append(pairs, placeholder, value)
	}
	replacer := strings.NewReplacer(pairs...)

	return replacer.Replace(tmpl)
}

func (pb *PromptBuilder) readIssueMetadataRaw(issueDir string, issue *Issue) string {
	if issueDir != "" {
		path := filepath.Join(issueDir, metadataFilename)
		if data, err := pb.readFile(path); err == nil {
			return string(data)
		}
	}
	if issue == nil {
		return fallbackValue
	}
	if marshaled, err := yaml.Marshal(issue); err == nil {
		return string(marshaled)
	}
	return fallbackValue
}

func (pb *PromptBuilder) listArtifactPaths(issueDir string, issue *Issue) string {
	var lines []string

	// Build lines with descriptions from issue attachments
	if issue != nil && len(issue.Attachments) > 0 {
		for _, att := range issue.Attachments {
			path := strings.TrimSpace(att.Path)
			if path == "" {
				continue
			}

			// Construct absolute path
			absPath := filepath.Join(issueDir, path)
			absPath = filepath.ToSlash(absPath) // Normalize to forward slashes

			line := fmt.Sprintf("- `%s`", absPath)
			if desc := strings.TrimSpace(att.Description); desc != "" {
				line = fmt.Sprintf("%s — %s", line, desc)
			} else if cat := strings.TrimSpace(att.Category); cat != "" {
				line = fmt.Sprintf("%s (%s)", line, cat)
			}
			lines = append(lines, line)
		}
	}

	// Fallback: scan directory if no attachment metadata
	if len(lines) == 0 && issueDir != "" {
		var paths []string
		artifactsDir := filepath.Join(issueDir, artifactsDirName)
		if info, err := os.Stat(artifactsDir); err == nil && info.IsDir() {
			filepath.WalkDir(artifactsDir, func(path string, d fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return nil
				}
				if d.IsDir() {
					return nil
				}
				// Use absolute path
				normalized := filepath.ToSlash(path)
				paths = append(paths, normalized)
				return nil
			})
		}

		if len(paths) > 0 {
			sort.Strings(paths)
			for _, p := range paths {
				lines = append(lines, fmt.Sprintf("- `%s`", p))
			}
		}
	}

	if len(lines) == 0 {
		return fallbackValue
	}

	return strings.Join(lines, "\n")
}

// computeIssueDirRelativePath converts an absolute issueDir path to a path relative to scenarioRoot
func (pb *PromptBuilder) computeIssueDirRelativePath(issueDir string) string {
	if issueDir == "" {
		return fallbackValue
	}

	// Compute relative path from scenarioRoot to issueDir
	rel, err := filepath.Rel(pb.scenarioRoot, issueDir)
	if err != nil {
		// Fallback: if we can't compute relative path, return the absolute path
		return issueDir
	}

	// Normalize to forward slashes for cross-platform consistency
	return filepath.ToSlash(rel)
}

func joinList(items []string) string {
	if len(items) == 0 {
		return fallbackValue
	}
	trimmed := make([]string, 0, len(items))
	for _, item := range items {
		if s := strings.TrimSpace(item); s != "" {
			trimmed = append(trimmed, s)
		}
	}
	if len(trimmed) == 0 {
		return fallbackValue
	}
	return strings.Join(trimmed, ", ")
}
