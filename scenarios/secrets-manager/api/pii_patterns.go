package main

import (
	"regexp"
	"strings"
)

// piiVulnerabilityPatterns are regex-based PII detection patterns, registered
// alongside the secret/vulnerability patterns. They use Go's RE2 engine so
// backtracking cannot blow up; per-file timeouts in scanFileList provide an
// additional safeguard against pathological inputs.
var piiVulnerabilityPatterns = []VulnerabilityPattern{
	{
		Type:           "pii_email",
		Severity:       "high",
		Pattern:        `(?i)\b[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}\b`,
		Description:    "Email address detected in source",
		Title:          "Email Address (PII)",
		Recommendation: "Move personal emails to configuration; use fixtures for tests",
		CanAutoFix:     false,
	},
	{
		Type:           "pii_phone_us",
		Severity:       "high",
		Pattern:        `\b(?:\+?1[-. ]?)?\(?\d{3}\)?[-. ]?\d{3}[-. ]?\d{4}\b`,
		Description:    "US phone number detected in source",
		Title:          "Phone Number (PII)",
		Recommendation: "Move personal phone numbers out of source; use synthetic values in tests",
		CanAutoFix:     false,
	},
	{
		Type:           "pii_ssn",
		Severity:       "critical",
		Pattern:        `\b\d{3}-\d{2}-\d{4}\b`,
		Description:    "US social security number detected",
		Title:          "SSN (PII)",
		Recommendation: "Remove immediately; SSNs must never appear in source",
		CanAutoFix:     false,
	},
	{
		Type:           "pii_credit_card",
		Severity:       "critical",
		Pattern:        `\b(?:4\d{3}|5[1-5]\d{2}|6011|3[47]\d{2})[- ]?\d{4}[- ]?\d{4}[- ]?\d{4}\b`,
		Description:    "Credit card number detected",
		Title:          "Credit Card (PII)",
		Recommendation: "Remove immediately; use synthetic test card numbers",
		CanAutoFix:     false,
	},
	{
		Type:           "pii_ip_address",
		Severity:       "medium",
		Pattern:        `\b(?:(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\b`,
		Description:    "IPv4 address detected",
		Title:          "IP Address (PII)",
		Recommendation: "Parameterize network addresses via config; suppress false positives with allowlist",
		CanAutoFix:     false,
	},
	{
		Type:           "pii_aws_key",
		Severity:       "critical",
		Pattern:        `\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`,
		Description:    "AWS access key ID detected",
		Title:          "AWS Access Key (PII)",
		Recommendation: "Rotate immediately; move to the credential authority or an AWS IAM role",
		CanAutoFix:     false,
	},
	{
		Type:     "pii_home_dir",
		Severity: "medium",
		// Allow either line start or a non-newline separator (tab/space/quote)
		// before the path. Using \s here would match the trailing \n of the
		// previous line and misattribute the finding to line N-1.
		Pattern:        `(?m)(?:^|[\t "])(?:/home/|/Users/)[A-Za-z0-9_\-]+`,
		Description:    "Home-directory path containing a username detected",
		Title:          "Home Directory Path (PII)",
		Recommendation: "Use $HOME or config placeholders to avoid leaking usernames",
		CanAutoFix:     false,
	},
}

// piiTypeSet captures the set of pattern types considered PII, for callers
// that need to distinguish PII from other vulnerabilities.
var piiTypeSet = func() map[string]struct{} {
	set := make(map[string]struct{}, len(piiVulnerabilityPatterns))
	for _, p := range piiVulnerabilityPatterns {
		set[p.Type] = struct{}{}
	}
	return set
}()

// isPIIType reports whether the vulnerability type represents a PII pattern.
func isPIIType(t string) bool {
	_, ok := piiTypeSet[t]
	return ok
}

// -----------------------------------------------------------------------------
// Context-aware filtering
// -----------------------------------------------------------------------------

// fileScanContext tracks per-file state needed to suppress false-positive
// matches (for example, whether the current line is inside a Go import block).
type fileScanContext struct {
	filePath      string
	baseName      string
	lines         []string
	inImportBlock []bool // per-line: true if this line is inside `import ( ... )`
	isGoFile      bool
	isLockfile    bool
	isGoSum       bool
}

// newFileScanContext precomputes per-line state for a file so that the
// filter can answer context questions cheaply for each match.
func newFileScanContext(filePath string, lines []string) *fileScanContext {
	base := lowerBase(filePath)
	ctx := &fileScanContext{
		filePath:   filePath,
		baseName:   base,
		lines:      lines,
		isGoFile:   strings.HasSuffix(strings.ToLower(filePath), ".go"),
		isLockfile: base == "package-lock.json" || base == "yarn.lock",
		isGoSum:    base == "go.sum",
	}

	if ctx.isGoFile {
		ctx.inImportBlock = computeGoImportBlockMask(lines)
	}
	return ctx
}

func lowerBase(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		path = path[idx+1:]
	}
	return strings.ToLower(path)
}

// computeGoImportBlockMask returns a per-line slice where true means the line
// is inside an `import ( ... )` block (the opening `import (` line and the
// closing `)` line are also considered "inside" the block, so imports formatted
// on those lines are suppressed too).
func computeGoImportBlockMask(lines []string) []bool {
	mask := make([]bool, len(lines))
	inBlock := false
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if !inBlock {
			if strings.HasPrefix(line, "import (") || line == "import (" {
				inBlock = true
				mask[i] = true
				continue
			}
			if strings.HasPrefix(line, "import \"") || strings.HasPrefix(line, "import `") {
				mask[i] = true
			}
			continue
		}
		mask[i] = true
		if line == ")" || strings.HasPrefix(line, ")") {
			inBlock = false
		}
	}
	return mask
}

// contextAwareFilter returns true if a match should be suppressed because the
// surrounding context is known to produce false positives (go imports, build
// tags, lockfiles, URLs in comments, version strings, etc.).
func contextAwareFilter(ctx *fileScanContext, lineNum int, findingType string) bool {
	if ctx == nil || lineNum < 1 || lineNum > len(ctx.lines) {
		return false
	}
	line := ctx.lines[lineNum-1]
	trimmed := strings.TrimSpace(line)

	// Lockfiles and go.sum are noisy by design.
	if ctx.isLockfile || ctx.isGoSum {
		return true
	}

	// Go build tag lines.
	if ctx.isGoFile && (strings.HasPrefix(trimmed, "//go:build") || strings.HasPrefix(trimmed, "// +build")) {
		return true
	}

	// Go import block — suppresses email/home-dir matches that come from
	// module paths like `github.com/foo/bar` or versioned deps.
	if ctx.isGoFile && lineNum-1 < len(ctx.inImportBlock) && ctx.inImportBlock[lineNum-1] {
		return true
	}

	// Version strings on recognized pragma lines: suppress IP-like matches that
	// are actually dotted-quad version numbers.
	if findingType == "pii_ip_address" && isVersionPragmaLine(trimmed) {
		return true
	}

	// URLs embedded in `//` comments: suppress IP/email matches that land inside
	// the URL path or host.
	if isCommentedURLContext(trimmed, findingType) {
		return true
	}

	return false
}

var versionPragmaPrefixes = []string{
	"version:", "version =", "version=", "@version", "// version",
	"# version", "version ", "kernel:", "kernel ",
}

func isVersionPragmaLine(trimmed string) bool {
	lower := strings.ToLower(trimmed)
	for _, prefix := range versionPragmaPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

// commentedURLRe matches lines whose meaningful content is a `//`- or `#`-style
// comment containing a URL. Used to suppress IP/email PII matches that come
// from documentation URLs (e.g., `// see https://example.com/admin/1.2.3.4`).
var commentedURLRe = regexp.MustCompile(`(?i)(?://|#)\s*.*https?://\S+`)

func isCommentedURLContext(trimmed string, findingType string) bool {
	switch findingType {
	case "pii_ip_address", "pii_email", "pii_home_dir":
	default:
		return false
	}
	return commentedURLRe.MatchString(trimmed)
}
