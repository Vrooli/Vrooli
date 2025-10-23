package server

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
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
		"{{issue_id}}":          fallbackValue,
		"{{issue_title}}":       fallbackValue,
		"{{issue_description}}": fallbackValue,
		"{{issue_type}}":        fallbackValue,
		"{{issue_priority}}":    fallbackValue,
		"{{app_name}}":          fallbackValue,
		"{{error_message}}":     fallbackValue,
		"{{stack_trace}}":       fallbackValue,
		"{{affected_files}}":    fallbackValue,
		"{{issue_metadata}}":    fallbackValue,
		"{{issue_artifacts}}":   fallbackValue,
		"{{agent_id}}":          fallbackValue,
		"{{project_path}}":      fallbackValue,
		"{{timestamp}}":         fallbackValue,
	}

	if issue != nil {
		replacements["{{issue_id}}"] = sanitizeValue(issue.ID)
		replacements["{{issue_title}}"] = sanitizeValue(issue.Title)
		replacements["{{issue_description}}"] = sanitizeValue(issue.Description)
		replacements["{{issue_type}}"] = sanitizeValue(issue.Type)
		replacements["{{issue_priority}}"] = sanitizeValue(issue.Priority)
		replacements["{{app_name}}"] = sanitizeValue(issue.AppID)
		replacements["{{error_message}}"] = sanitizeValue(issue.ErrorContext.ErrorMessage)
		replacements["{{stack_trace}}"] = sanitizeValue(issue.ErrorContext.StackTrace)
		replacements["{{affected_files}}"] = sanitizeValue(joinList(issue.ErrorContext.AffectedFiles))
	}

	replacements["{{issue_metadata}}"] = sanitizeValue(pb.readIssueMetadataRaw(issueDir, issue))
	replacements["{{issue_artifacts}}"] = sanitizeValue(pb.listArtifactPaths(issueDir, issue))
	replacements["{{agent_id}}"] = sanitizeValue(agentID)
	replacements["{{project_path}}"] = sanitizeValue(projectPath)
	replacements["{{timestamp}}"] = sanitizeValue(timestamp)

	result := tmpl
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
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
	var paths []string

	if issueDir != "" {
		artifactsDir := filepath.Join(issueDir, artifactsDirName)
		if info, err := os.Stat(artifactsDir); err == nil && info.IsDir() {
			filepath.WalkDir(artifactsDir, func(path string, d fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return nil
				}
				if d.IsDir() {
					return nil
				}
				rel, relErr := filepath.Rel(issueDir, path)
				if relErr != nil {
					return nil
				}
				normalized := filepath.ToSlash(rel)
				paths = append(paths, normalized)
				return nil
			})
		}
	}

	if len(paths) == 0 && issue != nil {
		for _, att := range issue.Attachments {
			if p := strings.TrimSpace(att.Path); p != "" {
				normalized := filepath.ToSlash(p)
				paths = append(paths, normalized)
			}
		}
	}

	if len(paths) == 0 {
		return fallbackValue
	}

	sort.Strings(paths)
	for i, p := range paths {
		paths[i] = "- " + p
	}
	return strings.Join(paths, "\n")
}

func sanitizeValue(str string) string {
	trimmed := strings.TrimSpace(str)
	if trimmed == "" {
		return fallbackValue
	}
	return trimmed
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
