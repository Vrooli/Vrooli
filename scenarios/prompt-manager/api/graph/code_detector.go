// DOC: docs/concepts/GRAPH.md#code-detection
package graph

import (
	"regexp"
	"strings"
)

// CLIDetector detects code references (CLI commands, scripts, API calls) in content.
type CLIDetector struct {
	scenarioCLIs map[string]bool
}

// Pattern regexes for code detection.
var (
	// Matches backtick-enclosed commands, e.g. `vrooli scenario start foo`
	backtickRE = regexp.MustCompile("`([^`]+)`")

	// Splits piped/chained commands: |, ||, &&, ;
	cmdSplitRE = regexp.MustCompile(`\s*(?:\|\||[|;]|&&)\s*`)

	// Matches HTTP patterns: GET/POST/PUT/DELETE https://...
	httpPatternRE = regexp.MustCompile(`\b(GET|POST|PUT|DELETE|PATCH)\s+(https?://\S+)`)

	// Matches common script extensions: *.sh, *.py, *.js
	scriptRefRE = regexp.MustCompile(`\b[\w/.-]+\.(sh|py|js|ts|rb|pl)\b`)
)

// NewCLIDetector creates a new code detector with known scenario CLI names.
func NewCLIDetector(scenarioNames []string) *CLIDetector {
	clis := make(map[string]bool, len(scenarioNames)+1)
	clis["vrooli"] = true
	for _, name := range scenarioNames {
		clis[name] = true
	}
	return &CLIDetector{scenarioCLIs: clis}
}

// Detect scans content for code references and returns all detected references.
// Backtick commands are split on pipes/chains (|, &&, ;) and each segment is
// classified independently: scenario CLIs → CodeScenarioCLI, everything else →
// CodeExternalTool. HTTP patterns and script references are detected separately.
func (d *CLIDetector) Detect(content string) []CodeReference {
	var refs []CodeReference
	lines := strings.Split(content, "\n")

	for lineIdx, line := range lines {
		lineNum := lineIdx + 1

		// Backtick commands: split pipes/chains, classify each segment
		for _, match := range backtickRE.FindAllStringSubmatch(line, -1) {
			if len(match) < 2 {
				continue
			}
			segments := cmdSplitRE.Split(strings.TrimSpace(match[1]), -1)
			for _, seg := range segments {
				seg = strings.TrimSpace(seg)
				fields := strings.Fields(seg)
				if len(fields) == 0 {
					continue
				}
				if d.scenarioCLIs[fields[0]] {
					refs = append(refs, CodeReference{
						Category: CodeScenarioCLI,
						Value:    seg,
						Line:     lineNum,
					})
				} else {
					refs = append(refs, CodeReference{
						Category: CodeExternalTool,
						Value:    seg,
						Line:     lineNum,
					})
				}
			}
		}

		// HTTP patterns (bare on line, documents API endpoints)
		if httpPatternRE.MatchString(line) {
			for _, m := range httpPatternRE.FindAllStringSubmatch(line, -1) {
				refs = append(refs, CodeReference{
					Category: CodeAPICall,
					Value:    m[0],
					Line:     lineNum,
				})
			}
		}

		// Script references
		if scriptRefRE.MatchString(line) {
			for _, m := range scriptRefRE.FindAllString(line, -1) {
				refs = append(refs, CodeReference{
					Category: CodeScript,
					Value:    m,
					Line:     lineNum,
				})
			}
		}
	}

	return refs
}
