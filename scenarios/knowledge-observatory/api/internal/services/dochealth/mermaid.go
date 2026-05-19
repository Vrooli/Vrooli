package dochealth

import (
	"fmt"
	"regexp"
	"strings"
)

var mermaidHeaderPattern = regexp.MustCompile(`^(graph|flowchart|flowchart\s+(TB|TD|LR|RL)|sequenceDiagram|classDiagram|stateDiagram|stateDiagram-v2|gantt|journey|erDiagram|pie)\b`)

func validateMermaidBlock(file string, line int, content string, strict bool, out *[]Finding, summary *fileMetrics) {
	summary.MermaidValidated++
	if mermaidHeaderPattern.MatchString(strings.TrimSpace(content)) && balancedBrackets(content) {
		return
	}
	message := fmt.Sprintf("%s:%d mermaid diagram appears invalid", file, line)
	if strict {
		summary.MermaidFailures++
		*out = append(*out, Finding{
			Code:     "mermaid_invalid",
			Severity: SeverityFailure,
			Message:  message,
			Path:     file,
			Line:     line,
		})
	} else {
		summary.MarkdownWarnings++
		*out = append(*out, Finding{
			Code:     "mermaid_invalid",
			Severity: SeverityWarning,
			Message:  message,
			Path:     file,
			Line:     line,
		})
	}
}

func balancedBrackets(content string) bool {
	var stack []rune
	pairs := map[rune]rune{')': '(', ']': '[', '}': '{'}
	for _, r := range content {
		switch r {
		case '(', '[', '{':
			stack = append(stack, r)
		case ')', ']', '}':
			if len(stack) == 0 || stack[len(stack)-1] != pairs[r] {
				return false
			}
			stack = stack[:len(stack)-1]
		}
	}
	return len(stack) == 0
}
