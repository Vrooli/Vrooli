package dochealing

import (
	"path/filepath"
	"strings"
)

func buildDiffPreview(runDiff *RunDiff, summary string) *DiffPreview {
	if runDiff == nil {
		return nil
	}
	chunks := splitUnifiedDiff(runDiff.Content)
	files := make([]FileDiff, 0, len(runDiff.Files))
	for _, file := range runDiff.Files {
		path := filepath.ToSlash(strings.TrimSpace(file.Path))
		if path == "" {
			continue
		}
		files = append(files, FileDiff{
			Path:      path,
			Operation: normalizeOperation(file.ChangeType),
			Diff:      chunks[path],
		})
	}
	return &DiffPreview{
		Files:   files,
		Summary: strings.TrimSpace(summary),
	}
}

func normalizeOperation(changeType string) string {
	switch strings.ToLower(strings.TrimSpace(changeType)) {
	case "added", "created", "add", "create":
		return "create"
	case "deleted", "removed", "delete":
		return "delete"
	default:
		return "modify"
	}
}

func splitUnifiedDiff(content string) map[string]string {
	content = strings.TrimSpace(content)
	if content == "" {
		return map[string]string{}
	}
	lines := strings.Split(content, "\n")
	out := make(map[string]string)
	var current string
	var builder strings.Builder
	flush := func() {
		if current == "" {
			builder.Reset()
			return
		}
		out[current] = strings.TrimRight(builder.String(), "\n")
		builder.Reset()
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			flush()
			current = parseDiffPath(line)
		}
		if current != "" {
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	}
	flush()
	return out
}

func parseDiffPath(line string) string {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return ""
	}
	path := fields[3]
	path = strings.TrimPrefix(path, "b/")
	path = strings.TrimPrefix(path, "a/")
	return filepath.ToSlash(path)
}
