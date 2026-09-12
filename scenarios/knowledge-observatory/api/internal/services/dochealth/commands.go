package dochealth

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type commandSnippet struct {
	File    string
	Line    int
	Command string
}

func validateCommandSnippets(ctx context.Context, scenarioDir string, markdownFiles []string, commandValidator CommandReferenceValidator) []Finding {
	var findings []Finding
	for _, file := range markdownFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		for _, snippet := range extractCommandSnippets(scenarioDir, file, string(content)) {
			if commandValidator == nil {
				findings = append(findings, Finding{
					Code:     "unknown_command_snippet",
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("%s:%d command snippet %q could not be validated: CLI Health command validator is not configured", snippet.File, snippet.Line, snippet.Command),
					Path:     snippet.File,
					Line:     snippet.Line,
					Target:   snippet.Command,
				})
				continue
			}
			result, err := commandValidator.ValidateCommandReference(ctx, CommandReferenceRequest{CommandText: snippet.Command})
			if err != nil {
				findings = append(findings, Finding{
					Code:     "unknown_command_snippet",
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("%s:%d command snippet %q could not be fully validated: %v", snippet.File, snippet.Line, snippet.Command, err),
					Path:     snippet.File,
					Line:     snippet.Line,
					Target:   snippet.Command,
				})
				continue
			}
			findings = append(findings, commandIssueFindings(snippet, result)...)
			switch result.Verdict {
			case "valid", "skipped":
				continue
			case "partial":
				findings = append(findings, Finding{
					Code:     "partial_command_snippet",
					Severity: SeverityInfo,
					Message:  fmt.Sprintf("%s:%d partially validated command snippet %q: %v", snippet.File, snippet.Line, snippet.Command, formatCommandReferenceResult(result)),
					Path:     snippet.File,
					Line:     snippet.Line,
					Target:   snippet.Command,
				})
			case "invalid", "unsupported":
				findings = append(findings, Finding{
					Code:     "broken_command_snippet",
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("%s:%d broken command snippet %q: %v", snippet.File, snippet.Line, snippet.Command, formatCommandReferenceResult(result)),
					Path:     snippet.File,
					Line:     snippet.Line,
					Target:   snippet.Command,
				})
			default:
				findings = append(findings, Finding{
					Code:     "unknown_command_snippet",
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("%s:%d command snippet %q could not be fully validated: %v", snippet.File, snippet.Line, snippet.Command, formatCommandReferenceResult(result)),
					Path:     snippet.File,
					Line:     snippet.Line,
					Target:   snippet.Command,
				})
			}
		}
	}
	return findings
}

// commandIssueFindingCodes maps DOCS-policy cli-health issue codes to their
// own dochealth finding codes. Issue codes absent from this table stay folded
// into the verdict-level finding (broken/partial/unknown_command_snippet) —
// enum_placeholder_mismatch and invalid_literal_value deliberately remain
// under broken_command_snippet because they make the snippet wrong, whereas
// placeholder_style is a mechanical convention violation with a deterministic
// autofix.
var commandIssueFindingCodes = map[string]struct {
	Code     string
	Severity Severity
}{
	"unquoted_placeholder": {Code: "placeholder_style", Severity: SeverityWarning},
}

// commandIssueFindings surfaces per-issue findings for issue codes with a
// dedicated mapping, carrying the byte-exact fix payload through.
func commandIssueFindings(snippet commandSnippet, result CommandReferenceResult) []Finding {
	var findings []Finding
	for _, issue := range result.Issues {
		mapped, ok := commandIssueFindingCodes[issue.Code]
		if !ok {
			continue
		}
		findings = append(findings, Finding{
			Code:     mapped.Code,
			Severity: mapped.Severity,
			Message:  fmt.Sprintf("%s:%d command snippet %q: %s", snippet.File, snippet.Line, snippet.Command, issue.Message),
			Path:     snippet.File,
			Line:     snippet.Line,
			Target:   snippet.Command,
			Fix:      issue.Fix,
		})
	}
	return findings
}

func extractCommandSnippets(scenarioDir, file, content string) []commandSnippet {
	var snippets []commandSnippet
	lines := strings.Split(content, "\n")
	inFence := false
	fenceMarker := ""
	fenceLang := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if fenceMatch := codeFencePattern.FindStringSubmatch(trimmed); fenceMatch != nil {
			marker := fenceMatch[1]
			if inFence && marker == fenceMarker {
				inFence = false
				fenceMarker = ""
				fenceLang = ""
			} else if !inFence {
				inFence = true
				fenceMarker = marker
				fenceLang = strings.ToLower(strings.TrimSpace(fenceMatch[2]))
			}
			continue
		}
		if !inFence || !isShellFence(fenceLang) {
			continue
		}
		command := normalizeShellSnippetLine(trimmed)
		if command == "" || !isVrooliOwnedCommandRoot(scenarioDir, firstShellToken(command)) {
			continue
		}
		snippets = append(snippets, commandSnippet{
			File:    file,
			Line:    i + 1,
			Command: command,
		})
	}
	return snippets
}

func isShellFence(lang string) bool {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "", "sh", "shell", "bash", "console":
		return true
	default:
		return false
	}
}

func normalizeShellSnippetLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return ""
	}
	for _, prompt := range []string{"$ ", "> "} {
		if strings.HasPrefix(line, prompt) {
			return strings.TrimSpace(strings.TrimPrefix(line, prompt))
		}
	}
	return line
}

func firstShellToken(command string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}
	for i, r := range command {
		if r == ' ' || r == '\t' {
			return command[:i]
		}
	}
	return command
}

func isVrooliOwnedCommandRoot(scenarioDir, root string) bool {
	root = strings.TrimSpace(root)
	if root == "" {
		return false
	}
	if root == "vrooli" {
		return true
	}
	if !strings.Contains(root, "-") {
		return false
	}
	scenariosRoot := filepath.Dir(scenarioDir)
	info, err := os.Stat(filepath.Join(scenariosRoot, root))
	return err == nil && info.IsDir()
}
