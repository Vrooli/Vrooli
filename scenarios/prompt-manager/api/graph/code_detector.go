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

	// Matches HTTP patterns: GET/POST/PUT/DELETE https://...
	httpPatternRE = regexp.MustCompile(`\b(GET|POST|PUT|DELETE|PATCH)\s+(https?://\S+)`)

	// Matches curl commands
	curlRE = regexp.MustCompile(`\bcurl\s+`)

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
func (d *CLIDetector) Detect(content string) []CodeReference {
	var refs []CodeReference
	lines := strings.Split(content, "\n")

	for lineIdx, line := range lines {
		lineNum := lineIdx + 1

		// Check backtick commands for scenario CLIs
		for _, match := range backtickRE.FindAllStringSubmatch(line, -1) {
			if len(match) < 2 {
				continue
			}
			cmd := strings.TrimSpace(match[1])
			firstWord := strings.Fields(cmd)
			if len(firstWord) == 0 {
				continue
			}
			if d.scenarioCLIs[firstWord[0]] {
				refs = append(refs, CodeReference{
					Category: CodeScenarioCLI,
					Value:    cmd,
					Line:     lineNum,
				})
			}
		}

		// Check for HTTP/API calls
		if httpPatternRE.MatchString(line) {
			matches := httpPatternRE.FindAllStringSubmatch(line, -1)
			for _, m := range matches {
				refs = append(refs, CodeReference{
					Category: CodeAPICall,
					Value:    m[0],
					Line:     lineNum,
				})
			}
		}

		// Check for curl
		if curlRE.MatchString(line) {
			refs = append(refs, CodeReference{
				Category: CodeAPICall,
				Value:    strings.TrimSpace(line),
				Line:     lineNum,
			})
		}

		// Check for script references
		if scriptRefRE.MatchString(line) {
			matches := scriptRefRE.FindAllString(line, -1)
			for _, m := range matches {
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
