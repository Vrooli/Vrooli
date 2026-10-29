package androidadb

import (
	"bytes"
	"encoding/json"
	"strings"
)

type cappedBuffer struct {
	buffer   bytes.Buffer
	limit    int
	exceeded bool
}

func (b *cappedBuffer) Write(data []byte) (int, error) {
	remaining := b.limit - b.buffer.Len()
	if remaining <= 0 {
		b.exceeded = true
		return len(data), nil
	}
	if len(data) > remaining {
		_, _ = b.buffer.Write(data[:remaining])
		b.exceeded = true
		return len(data), nil
	}
	return b.buffer.Write(data)
}

// validExactAndroidComponent accepts package/class identities only. It rejects
// action strings, package-only filters, and shell syntax before logcat runs.
func validExactAndroidComponent(component string) bool {
	if len(component) == 0 || len(component) > 512 || strings.Count(component, "/") != 1 {
		return false
	}
	parts := strings.SplitN(component, "/", 2)
	if !applicationIdentityPattern.MatchString(parts[0]) {
		return false
	}
	className := parts[1]
	if strings.HasPrefix(className, ".") {
		className = parts[0] + className
	}
	return applicationIdentityPattern.MatchString(className)
}

// filterExactComponentLogLines retains only lines that name this exact Android
// component. Android may print either the fully qualified class or its
// package-relative shorthand in a ComponentName.
func filterExactComponentLogLines(output, component string) []string {
	if !validExactAndroidComponent(component) {
		return nil
	}
	parts := strings.SplitN(component, "/", 2)
	className := parts[1]
	if strings.HasPrefix(className, ".") {
		className = parts[0] + className
	}
	shortClass := className
	if strings.HasPrefix(className, parts[0]+".") {
		shortClass = strings.TrimPrefix(className, parts[0])
	}
	shortComponent := parts[0] + "/" + shortClass

	var matched []string
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		if containsComponentToken(line, component) || containsComponentToken(line, parts[0]+"/"+className) || containsComponentToken(line, shortComponent) {
			matched = append(matched, line)
		}
	}
	return matched
}

func containsComponentToken(line, component string) bool {
	for offset := 0; offset < len(line); {
		relative := strings.Index(line[offset:], component)
		if relative < 0 {
			return false
		}
		start := offset + relative
		end := offset + relative + len(component)
		leftBoundary := start == 0 || !isAndroidIdentifierByte(line[start-1])
		rightBoundary := end == len(line) || !isAndroidIdentifierByte(line[end])
		if leftBoundary && rightBoundary {
			return true
		}
		offset = end
	}
	return false
}

func isAndroidIdentifierByte(value byte) bool {
	return value == '.' || value == '$' || value == '_' || value >= '0' && value <= '9' || value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}

func summarizeFrameworkLogMatches(component, tag, priority string, lines []string) []byte {
	classes := make(map[string]int)
	for _, line := range lines {
		classes[classifyFrameworkLogLine(line)]++
	}
	result := "matching_component_lines_found"
	if len(lines) == 0 {
		result = "inconclusive_no_matching_component_lines"
	}
	summary, _ := json.Marshal(struct {
		Component         string         `json:"component"`
		Tag               string         `json:"tag"`
		Priority          string         `json:"priority"`
		WindowLineLimit   int            `json:"window_line_limit"`
		MatchingLineCount int            `json:"matching_line_count"`
		DiagnosticClasses map[string]int `json:"diagnostic_classes,omitempty"`
		Result            string         `json:"result"`
	}{Component: component, Tag: tag, Priority: priority, WindowLineLimit: 250, MatchingLineCount: len(lines), DiagnosticClasses: classes, Result: result})
	return append(summary, '\n')
}

func classifyFrameworkLogLine(line string) string {
	switch {
	case strings.Contains(line, "Permission Denial"):
		return "permission_denial"
	case strings.Contains(line, "Appop Denial"):
		return "appop_denial"
	case strings.Contains(line, "not found"):
		return "service_not_found"
	case strings.Contains(line, "Unable to launch app"), strings.Contains(line, "Unable to start service"):
		return "service_start_failure"
	case strings.Contains(line, "Exception when starting service"):
		return "service_start_exception"
	case strings.Contains(line, "Unable to find app for caller"):
		return "caller_not_found"
	default:
		return "other_matching_component_line"
	}
}
