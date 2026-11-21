package intent

import (
	"path/filepath"
	"strings"
)

// NormalizeRef parses validation[].ref into the frozen intent contract shape.
func NormalizeRef(raw, validationType string) Ref {
	ref := Ref{Raw: raw}
	trimmed := strings.TrimSpace(raw)
	if !isPathCheckedValidation(validationType) {
		ref.Kind = RefManual
		ref.Path = trimmed
		return ref
	}
	if trimmed == "" {
		ref.Kind = RefCode
		return ref
	}

	pathPart := trimmed
	if hash := strings.Index(pathPart, "#"); hash >= 0 {
		pathPart = pathPart[:hash]
	}
	if memberIdx := strings.Index(pathPart, "::"); memberIdx >= 0 {
		ref.Member = pathPart[memberIdx+2:]
		pathPart = pathPart[:memberIdx]
	} else if colonIdx := legacyMemberColon(pathPart); colonIdx >= 0 {
		ref.Member = pathPart[colonIdx+1:]
		pathPart = pathPart[:colonIdx]
	}

	ref.Glob = strings.ContainsAny(pathPart, "*?[")
	ref.Path = normalizeGlobPath(pathPart)
	ref.Kind = classifyRefKind(ref.Path)
	return ref
}

func isPathCheckedValidation(validationType string) bool {
	switch strings.ToLower(strings.TrimSpace(validationType)) {
	case "test", "unit", "integration", "code", "automation":
		return true
	default:
		return false
	}
}

func legacyMemberColon(value string) int {
	extensions := []string{".go:", ".ts:", ".tsx:", ".js:", ".jsx:", ".bats:", ".sh:", ".py:", ".json:"}
	for _, ext := range extensions {
		idx := strings.Index(value, ext)
		if idx == -1 {
			continue
		}
		colonIdx := idx + len(ext) - 1
		if colonIdx+1 < len(value) {
			return colonIdx
		}
	}
	return -1
}

func normalizeGlobPath(path string) string {
	cleaned := filepath.ToSlash(strings.TrimSpace(path))
	if cleaned == "" {
		return ""
	}
	if !strings.ContainsAny(cleaned, "*?[") {
		return cleaned
	}
	cut := len(cleaned)
	for _, marker := range []string{"*", "?", "["} {
		if idx := strings.Index(cleaned, marker); idx >= 0 && idx < cut {
			cut = idx
		}
	}
	prefix := strings.TrimRight(cleaned[:cut], "/")
	if prefix == "" {
		return "."
	}
	if strings.Contains(filepath.Base(prefix), ".") {
		return filepath.Dir(prefix)
	}
	return prefix
}

func classifyRefKind(path string) RefKind {
	lower := strings.ToLower(strings.TrimSpace(filepath.ToSlash(path)))
	if lower == "" {
		return RefCode
	}
	if strings.HasPrefix(lower, "docs/") || strings.HasSuffix(lower, ".md") {
		return RefDoc
	}
	return RefCode
}
