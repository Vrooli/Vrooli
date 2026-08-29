package hostreqcheck

import "strings"

func containsCandidateReference(text, candidate string) bool {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || isIgnorableCommentLine(trimmed, candidate) {
			continue
		}
		if strings.Contains(line, "command -v "+candidate) ||
			strings.Contains(line, "which "+candidate) ||
			containsCommandCallReference(line, candidate) ||
			containsShellCommand(line, candidate) {
			return true
		}
	}
	return false
}

func isIgnorableCommentLine(line, candidate string) bool {
	switch {
	case strings.HasPrefix(line, "#!/"):
		return !(candidate == "bats" && containsShellCommand(line, candidate))
	case strings.HasPrefix(line, "#"),
		strings.HasPrefix(line, "//"),
		strings.HasPrefix(line, "/*"),
		strings.HasPrefix(line, "*"):
		return true
	default:
		return false
	}
}

func containsCommandCallReference(text, token string) bool {
	quoted := `"` + token + `"`
	patterns := []string{
		"exec.Command(" + quoted,
		"exec.CommandContext(",
		"exec.LookPath(" + quoted + ")",
		".LookPath(" + quoted + ")",
		".shell(",
		"ExecuteWithResult(ctx, " + quoted,
	}
	for _, pattern := range patterns {
		if !strings.Contains(text, pattern) {
			continue
		}
		if strings.Contains(pattern, "CommandContext(") || strings.Contains(pattern, ".shell(") {
			if strings.Contains(text, quoted) {
				return true
			}
			continue
		}
		return true
	}
	return false
}

func containsShellCommand(text, token string) bool {
	if token == "" {
		return false
	}
	index := 0
	for {
		offset := strings.Index(text[index:], token)
		if offset < 0 {
			return false
		}
		start := index + offset
		end := start + len(token)
		if shellCommandBoundary(text, start-1) && shellCommandTailBoundary(text, end) {
			return true
		}
		index = end
	}
}

func shellCommandBoundary(text string, index int) bool {
	return index < 0 || index >= len(text) || strings.ContainsRune(" \t(|&;", rune(text[index]))
}

func shellCommandTailBoundary(text string, index int) bool {
	if index < 0 || index >= len(text) {
		return true
	}
	next := index
	for next < len(text) && (text[next] == ' ' || text[next] == '\t') {
		next++
	}
	if next >= len(text) {
		return true
	}
	switch text[next] {
	case '\n', '\r', ')', ';', '|', '&', ',':
		return true
	case ':', '=':
		return false
	default:
		return text[index] == ' ' || text[index] == '\t'
	}
}
