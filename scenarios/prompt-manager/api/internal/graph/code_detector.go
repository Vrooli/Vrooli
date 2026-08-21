// DOC: docs/concepts/GRAPH.md#code-detection
package graph

import (
	"regexp"
	"strings"
)

// CLIDetector detects code references (CLI commands, scripts, API calls) in content.
// Script detection is backtick-only: `.sh` and `.bash` files in backticks are
// classified as CodeScript; plain-text file references are ignored.
type CLIDetector struct {
	scenarioCLIs      map[string]bool
	knownExternalCLIs map[string]bool
}

// defaultExternalCLIs is the set of well-known external command-line tools.
// Backtick content whose first word is not in scenarioCLIs or this set is
// silently ignored (it's likely an inline code reference, not a CLI command).
var defaultExternalCLIs = map[string]bool{
	// Shell basics
	"bash": true, "sh": true,
	"ls": true, "cd": true, "cat": true, "head": true, "tail": true,
	"echo": true, "cp": true, "mv": true, "rm": true, "mkdir": true,
	"chmod": true, "chown": true, "find": true, "xargs": true, "env": true,
	"export": true, "source": true, "basename": true, "dirname": true,
	"which": true, "whoami": true, "sudo": true, "su": true, "man": true,
	// Text processing
	"grep": true, "rg": true, "sed": true, "awk": true, "tr": true,
	"cut": true, "sort": true, "uniq": true, "wc": true, "diff": true,
	"tee": true, "less": true, "more": true,
	// JSON/YAML
	"jq": true, "yq": true,
	// Network
	"curl": true, "wget": true, "ssh": true, "scp": true, "rsync": true,
	"nc": true, "nmap": true, "dig": true, "ping": true,
	// Containers & orchestration
	"docker": true, "podman": true, "kubectl": true, "helm": true,
	// Version control
	"git": true, "gh": true,
	// Build tools
	"make": true, "cmake": true, "ninja": true,
	// Package managers
	"npm": true, "npx": true, "pnpm": true, "yarn": true, "bun": true,
	"pip": true, "pip3": true, "apt": true, "brew": true, "snap": true,
	// Language runtimes & tools
	"go": true, "python": true, "python3": true, "node": true, "deno": true,
	"ruby": true, "perl": true, "java": true, "javac": true, "rustc": true,
	"cargo": true,
	// Go tools
	"gofumpt": true, "golangci-lint": true, "gopls": true, "dlv": true,
	// Testing
	"pytest": true, "jest": true, "vitest": true, "bats": true,
	// Code search
	"ast-grep": true, "sg": true,
	// Database clients
	"psql": true, "mysql": true, "redis-cli": true, "sqlite3": true,
	// Servers & infra
	"caddy": true, "nginx": true, "systemctl": true, "journalctl": true,
	// Packaging
	"fpm": true, "dpkg": true, "rpm": true, "tar": true, "zip": true,
	"unzip": true, "gzip": true,
	// Misc
	"wine": true, "xdg-open": true, "killall": true, "pkill": true,
	"pgrep": true, "htop": true, "top": true, "ps": true, "lsof": true,
	"nohup": true, "screen": true, "tmux": true,
}

// Pattern regexes for code detection.
var (
	// Matches backtick-enclosed commands, e.g. `vrooli scenario start foo`.
	// [^`]+ in Go regex matches newlines (negated character class), so this
	// also captures multi-line backtick spans.
	backtickRE = regexp.MustCompile("`([^`]+)`")

	// Matches triple-backtick fenced code blocks (``` ... ```).
	codeFenceRE = regexp.MustCompile("(?s)```[^\n]*\n.*?```")

	// Splits piped/chained commands: |, ||, &&, ;
	cmdSplitRE = regexp.MustCompile(`\s*(?:\|\||[|;]|&&)\s*`)

	// Matches HTTP patterns: GET/POST/PUT/DELETE https://...
	httpPatternRE = regexp.MustCompile(`\b(GET|POST|PUT|DELETE|PATCH)\s+(https?://\S+)`)
)

// NewCLIDetector creates a new code detector with known scenario CLI names.
func NewCLIDetector(scenarioNames []string) *CLIDetector {
	clis := make(map[string]bool, len(scenarioNames)+1)
	clis["vrooli"] = true
	for _, name := range scenarioNames {
		clis[name] = true
	}
	return &CLIDetector{
		scenarioCLIs:      clis,
		knownExternalCLIs: defaultExternalCLIs,
	}
}

// stripCodeFences replaces triple-backtick fenced blocks with equal-length
// newline sequences so that byte offsets (and therefore line numbers) are
// preserved while preventing backtickRE from matching content inside fences.
func stripCodeFences(content string) string {
	return codeFenceRE.ReplaceAllStringFunc(content, func(match string) string {
		n := strings.Count(match, "\n")
		return strings.Repeat("\n", n)
	})
}

// lineNumberAtOffset returns the 1-based line number for a byte offset in content.
func lineNumberAtOffset(content string, offset int) int {
	return strings.Count(content[:offset], "\n") + 1
}

// isShellScript returns true if the token looks like a shell script path (.sh or .bash).
func isShellScript(cmd string) bool {
	return strings.HasSuffix(cmd, ".sh") || strings.HasSuffix(cmd, ".bash")
}

// Detect scans content for code references and returns all detected references.
// Backtick commands are matched on the full content (after stripping code fences)
// so that multi-line backtick spans are captured. Each match is split on
// pipes/chains (|, &&, ;) and classified independently: scenario CLIs →
// CodeScenarioCLI, known external tools → CodeExternalTool, .sh/.bash paths →
// CodeScript. HTTP patterns are detected per-line on the original content.
// Plain-text file references (outside backticks) are not detected.
func (d *CLIDetector) Detect(content string) []CodeReference {
	var refs []CodeReference

	// --- Backtick commands (full-content, code-fence-free) ---
	stripped := stripCodeFences(content)
	for _, idx := range backtickRE.FindAllStringSubmatchIndex(stripped, -1) {
		// idx[2]:idx[3] is the capture group (content between backticks)
		if idx[2] < 0 {
			continue
		}
		inner := stripped[idx[2]:idx[3]]
		lineNum := lineNumberAtOffset(stripped, idx[0])

		segments := cmdSplitRE.Split(strings.TrimSpace(inner), -1)
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
			} else if d.knownExternalCLIs[fields[0]] {
				refs = append(refs, CodeReference{
					Category: CodeExternalTool,
					Value:    seg,
					Line:     lineNum,
				})
			} else if isShellScript(fields[0]) {
				refs = append(refs, CodeReference{
					Category: CodeScript,
					Value:    seg,
					Line:     lineNum,
				})
			}
			// else: not a recognized CLI — skip (inline code reference)
		}
	}

	// --- HTTP patterns (per-line on original content) ---
	lines := strings.Split(content, "\n")
	for lineIdx, line := range lines {
		lineNum := lineIdx + 1

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
	}

	return refs
}
