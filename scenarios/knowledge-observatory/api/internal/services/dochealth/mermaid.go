package dochealth

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

type DiagramBlock struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}
type DiagramVerdict struct {
	ID    string
	Valid bool
	Error string
	Line  int
}
type DiagramValidation struct {
	Engine   string
	Verdicts []DiagramVerdict
}
type DiagramValidator interface {
	ValidateDiagrams(context.Context, []DiagramBlock) (DiagramValidation, error)
}

type extractedDiagramBlock struct {
	DiagramBlock
	startLine int // The opening fence line, one-based.
}

// extractMermaidBlocks is the shared fence parser used by both doc health and
// the caller-facing diagram-validation RPC. Only closed Mermaid fences are
// returned; callers retain responsibility for reporting unclosed fences.
func extractMermaidBlocks(content string) []extractedDiagramBlock {
	var blocks []extractedDiagramBlock
	lines := strings.Split(content, "\n")
	inFence := false
	marker := ""
	mermaid := false
	startLine := 0
	var body []string

	for i, line := range lines {
		trim := strings.TrimSpace(line)
		matches := codeFencePattern.FindStringSubmatch(trim)
		if len(matches) == 0 {
			if inFence && mermaid {
				body = append(body, line)
			}
			continue
		}
		if !inFence {
			inFence = true
			marker = matches[1]
			mermaid = matches[2] == "mermaid" || matches[2] == "mermaidjs"
			startLine = i + 1
			body = nil
			continue
		}
		if matches[1] != marker {
			if mermaid {
				body = append(body, line)
			}
			continue
		}
		if mermaid {
			blocks = append(blocks, extractedDiagramBlock{
				DiagramBlock: DiagramBlock{ID: fmt.Sprintf("%d", len(blocks)), Content: strings.Join(body, "\n")},
				startLine:    startLine,
			})
		}
		inFence = false
		marker = ""
		mermaid = false
		body = nil
	}
	return blocks
}

func validateMermaidBlocksWithValidator(file string, blocks []extractedDiagramBlock, strict bool, validator DiagramValidator, out *[]Finding, summary *fileMetrics) {
	if len(blocks) == 0 {
		return
	}
	summary.MermaidValidated += len(blocks)
	if validator == nil {
		for _, block := range blocks {
			validateMermaidFallback(file, block.startLine, block.Content, strict, "diagram parser is unavailable", out, summary)
		}
		return
	}
	input := make([]DiagramBlock, len(blocks))
	for i, block := range blocks {
		input[i] = block.DiagramBlock
	}
	result, err := validator.ValidateDiagrams(context.Background(), input)
	if err != nil || len(result.Verdicts) != len(blocks) {
		reason := "diagram parser returned incomplete verdicts"
		if err != nil {
			reason = err.Error()
		}
		for _, block := range blocks {
			validateMermaidFallback(file, block.startLine, block.Content, strict, reason, out, summary)
		}
		return
	}
	for i, verdict := range result.Verdicts {
		if verdict.Valid {
			continue
		}
		severity := SeverityWarning
		if strict {
			severity = SeverityFailure
			summary.MermaidFailures++
		} else {
			summary.MarkdownWarnings++
		}
		line := blocks[i].startLine
		if verdict.Line > 0 {
			line += verdict.Line
		}
		*out = append(*out, Finding{Code: "mermaid_invalid", Severity: severity, Message: fmt.Sprintf("%s:%d %s", file, line, verdict.Error), Path: file, Line: line})
	}
}

func validateMermaidBlockWithValidator(file string, line int, content string, strict bool, validator DiagramValidator, out *[]Finding, summary *fileMetrics) {
	validateMermaidBlocksWithValidator(file, []extractedDiagramBlock{{DiagramBlock: DiagramBlock{ID: "block", Content: content}, startLine: line}}, strict, validator, out, summary)
}

func validateMermaidFallback(file string, line int, content string, strict bool, reason string, out *[]Finding, summary *fileMetrics) {
	if !(mermaidHeaderPattern.MatchString(strings.TrimSpace(content)) && balancedBrackets(content)) {
		severity := SeverityWarning
		if strict {
			severity = SeverityFailure
			summary.MermaidFailures++
		} else {
			summary.MarkdownWarnings++
		}
		*out = append(*out, Finding{
			Code:     "mermaid_invalid",
			Severity: severity,
			Message:  fmt.Sprintf("%s:%d mermaid diagram appears invalid", file, line),
			Path:     file,
			Line:     line,
		})
	}
	summary.MarkdownWarnings++
	*out = append(*out, Finding{Code: "mermaid_unverified", Severity: SeverityWarning, Message: fmt.Sprintf("%s:%d Mermaid parser unavailable: %s", file, line, reason), Path: file, Line: line})
}

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
