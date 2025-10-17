package scanners

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// CustomPatternScanner implements pattern-based vulnerability scanning
type CustomPatternScanner struct {
	logger   Logger
	patterns []PatternRule
}

// PatternRule defines a security pattern to check
type PatternRule struct {
	ID             string
	Name           string
	Pattern        *regexp.Regexp
	FileExtensions []string
	Severity       Severity
	Category       string
	Description    string
	Recommendation string
	CWE            int
	OWASP          string
}

// NewCustomPatternScanner creates a new custom pattern scanner
func NewCustomPatternScanner(logger Logger) *CustomPatternScanner {
	scanner := &CustomPatternScanner{
		logger: logger,
	}
	scanner.initializePatterns()
	return scanner
}

// initializePatterns sets up all the security patterns
func (c *CustomPatternScanner) initializePatterns() {
	c.patterns = []PatternRule{
		// SQL Injection patterns
		{
			ID:             "SQL-001",
			Name:           "SQL Injection via String Concatenation",
			Pattern:        regexp.MustCompile(`(?i)(query|execute|exec)\s*\(\s*['"].*['"].*\+.*\$`),
			FileExtensions: []string{".go", ".js", ".ts", ".py", ".java", ".php"},
			Severity:       SeverityCritical,
			Category:       "SQL Injection",
			Description:    "SQL query constructed with string concatenation using user input",
			Recommendation: "Use parameterized queries or prepared statements",
			CWE:            89,
			OWASP:          "A03:2021 – Injection",
		},
		{
			ID:             "SQL-002",
			Name:           "SQL Injection via Format String",
			Pattern:        regexp.MustCompile(`(?i)fmt\.sprintf.*select.*from.*where`),
			FileExtensions: []string{".go"},
			Severity:       SeverityCritical,
			Category:       "SQL Injection",
			Description:    "SQL query constructed using format strings",
			Recommendation: "Use parameterized queries with placeholders ($1, $2)",
			CWE:            89,
			OWASP:          "A03:2021 – Injection",
		},

		// Command Injection patterns
		{
			ID:             "CMD-001",
			Name:           "Command Injection Risk",
			Pattern:        regexp.MustCompile(`(?i)exec\.(command|cmd)\s*\([^,)]*\$`),
			FileExtensions: []string{".go"},
			Severity:       SeverityCritical,
			Category:       "Command Injection",
			Description:    "Potential command injection through exec.Command",
			Recommendation: "Validate and sanitize input, use allowlists for commands",
			CWE:            78,
			OWASP:          "A03:2021 – Injection",
		},
		{
			ID:             "CMD-002",
			Name:           "Shell Injection via eval",
			Pattern:        regexp.MustCompile(`(?i)eval\s*\(`),
			FileExtensions: []string{".js", ".ts", ".py", ".rb", ".php"},
			Severity:       SeverityCritical,
			Category:       "Command Injection",
			Description:    "Use of eval() function with potential user input",
			Recommendation: "Avoid eval(), use safer alternatives",
			CWE:            95,
			OWASP:          "A03:2021 – Injection",
		},

		// Path Traversal patterns
		{
			ID:             "PATH-001",
			Name:           "Path Traversal Pattern",
			Pattern:        regexp.MustCompile(`\.\./.*\.\./`),
			FileExtensions: []string{".go", ".js", ".ts", ".py", ".java", ".php"},
			Severity:       SeverityHigh,
			Category:       "Path Traversal",
			Description:    "Potential path traversal vulnerability",
			Recommendation: "Validate and sanitize file paths, use filepath.Clean()",
			CWE:            22,
			OWASP:          "A01:2021 – Broken Access Control",
		},
		{
			ID:             "PATH-002",
			Name:           "Unsanitized File Path",
			Pattern:        regexp.MustCompile(`(?i)(open|readfile|writefile)\s*\([^)]*\$`),
			FileExtensions: []string{".go", ".js", ".ts", ".py"},
			Severity:       SeverityHigh,
			Category:       "Path Traversal",
			Description:    "File operation with unsanitized user input",
			Recommendation: "Validate file paths and restrict to safe directories",
			CWE:            22,
			OWASP:          "A01:2021 – Broken Access Control",
		},

		// XSS patterns
		{
			ID:             "XSS-001",
			Name:           "Potential XSS in HTML Output",
			Pattern:        regexp.MustCompile(`(?i)innerhtml\s*=.*\$`),
			FileExtensions: []string{".js", ".ts", ".tsx"},
			Severity:       SeverityHigh,
			Category:       "Cross-Site Scripting",
			Description:    "Direct HTML injection via innerHTML",
			Recommendation: "Use textContent or proper escaping functions",
			CWE:            79,
			OWASP:          "A03:2021 – Injection",
		},
		{
			ID:             "XSS-002",
			Name:           "Unescaped Template Output",
			Pattern:        regexp.MustCompile(`\{\{\{.*\}\}\}`),
			FileExtensions: []string{".html", ".vue", ".hbs"},
			Severity:       SeverityHigh,
			Category:       "Cross-Site Scripting",
			Description:    "Unescaped template output (triple braces)",
			Recommendation: "Use double braces for automatic escaping",
			CWE:            79,
			OWASP:          "A03:2021 – Injection",
		},

		// Authentication/Authorization patterns
		{
			ID:             "AUTH-001",
			Name:           "Missing Authentication Check",
			Pattern:        regexp.MustCompile(`(?i)func\s+\w+handler`),
			FileExtensions: []string{".go"},
			Severity:       SeverityMedium,
			Category:       "Authentication",
			Description:    "HTTP handler function detected - verify authentication is implemented",
			Recommendation: "Add authentication middleware to all sensitive endpoints",
			CWE:            306,
			OWASP:          "A07:2021 – Identification and Authentication Failures",
		},
		{
			ID:             "AUTH-002",
			Name:           "Hardcoded Token/Password",
			Pattern:        regexp.MustCompile(`(?i)(token|password|apikey|api_key)\s*[:=]\s*["'][^"']+["']`),
			FileExtensions: []string{".go", ".js", ".ts", ".py", ".java"},
			Severity:       SeverityCritical,
			Category:       "Credential Exposure",
			Description:    "Hardcoded credential in source code",
			Recommendation: "Use environment variables or secrets management",
			CWE:            798,
			OWASP:          "A07:2021 – Identification and Authentication Failures",
		},

		// Cryptography patterns
		{
			ID:             "CRYPTO-001",
			Name:           "Weak Hash Algorithm",
			Pattern:        regexp.MustCompile(`(?i)(md5|sha1)\.(new|sum)`),
			FileExtensions: []string{".go", ".js", ".ts", ".py", ".java"},
			Severity:       SeverityHigh,
			Category:       "Cryptography",
			Description:    "Use of weak hash algorithm (MD5/SHA1)",
			Recommendation: "Use SHA-256 or stronger hash algorithms",
			CWE:            328,
			OWASP:          "A02:2021 – Cryptographic Failures",
		},
		{
			ID:             "CRYPTO-002",
			Name:           "Weak Random Number Generation",
			Pattern:        regexp.MustCompile(`(?i)math\.rand|math/rand`),
			FileExtensions: []string{".go"},
			Severity:       SeverityMedium,
			Category:       "Cryptography",
			Description:    "Use of weak random number generator",
			Recommendation: "Use crypto/rand for security-sensitive randomness",
			CWE:            338,
			OWASP:          "A02:2021 – Cryptographic Failures",
		},

		// Error Handling patterns
		{
			ID:             "ERROR-001",
			Name:           "Ignored Error",
			Pattern:        regexp.MustCompile(`_\s*,\s*err\s*:=`),
			FileExtensions: []string{".go"},
			Severity:       SeverityLow,
			Category:       "Error Handling",
			Description:    "Error explicitly ignored",
			Recommendation: "Handle errors appropriately or document why ignoring is safe",
			CWE:            391,
			OWASP:          "",
		},
		{
			ID:             "ERROR-002",
			Name:           "Stack Trace Exposure",
			Pattern:        regexp.MustCompile(`(?i)stack\.?trace|traceback`),
			FileExtensions: []string{".go", ".js", ".ts", ".py", ".java"},
			Severity:       SeverityMedium,
			Category:       "Information Disclosure",
			Description:    "Stack trace potentially exposed to users",
			Recommendation: "Log detailed errors internally, return generic errors to users",
			CWE:            209,
			OWASP:          "A09:2021 – Security Logging and Monitoring Failures",
		},

		// HTTP Security Headers
		{
			ID:             "HTTP-001",
			Name:           "Missing Security Headers",
			Pattern:        regexp.MustCompile(`(?i)w\.header\.set\s*\(\s*["']content-type`),
			FileExtensions: []string{".go"},
			Severity:       SeverityMedium,
			Category:       "Security Misconfiguration",
			Description:    "HTTP response without security headers",
			Recommendation: "Add security headers: X-Content-Type-Options, X-Frame-Options, etc.",
			CWE:            16,
			OWASP:          "A05:2021 – Security Misconfiguration",
		},
		{
			ID:             "HTTP-002",
			Name:           "CORS Wildcard",
			Pattern:        regexp.MustCompile(`(?i)access-control-allow-origin.*\*`),
			FileExtensions: []string{".go", ".js", ".ts", ".py", ".java"},
			Severity:       SeverityHigh,
			Category:       "Security Misconfiguration",
			Description:    "CORS configured with wildcard origin",
			Recommendation: "Specify allowed origins explicitly",
			CWE:            942,
			OWASP:          "A05:2021 – Security Misconfiguration",
		},

		// Resource Management
		{
			ID:             "RES-001",
			Name:           "HTTP Request Without Body.Close",
			Pattern:        regexp.MustCompile(`(?i)http\.(get|post|put|delete)`),
			FileExtensions: []string{".go"},
			Severity:       SeverityMedium,
			Category:       "Resource Leak",
			Description:    "HTTP request detected - verify response body is properly closed",
			Recommendation: "Add 'defer resp.Body.Close()' after error check",
			CWE:            404,
			OWASP:          "",
		},
		{
			ID:             "RES-002",
			Name:           "File Operation Detected",
			Pattern:        regexp.MustCompile(`(?i)os\.(open|create)`),
			FileExtensions: []string{".go"},
			Severity:       SeverityMedium,
			Category:       "Resource Leak",
			Description:    "File handle not closed",
			Recommendation: "Add 'defer file.Close()' after error check",
			CWE:            404,
			OWASP:          "",
		},

		// Debug/Development Code
		{
			ID:             "DEBUG-001",
			Name:           "Debug Code in Production",
			Pattern:        regexp.MustCompile(`(?i)(console\.(log|debug)|fmt\.print|print\(|debugger)`),
			FileExtensions: []string{".go", ".js", ".ts", ".py"},
			Severity:       SeverityLow,
			Category:       "Debug Code",
			Description:    "Debug statements found in code",
			Recommendation: "Remove debug statements or use proper logging",
			CWE:            489,
			OWASP:          "",
		},
		{
			ID:             "DEBUG-002",
			Name:           "TODO/FIXME Comments",
			Pattern:        regexp.MustCompile(`(?i)(todo|fixme|hack|xxx):`),
			FileExtensions: []string{".go", ".js", ".ts", ".py", ".java"},
			Severity:       SeverityInfo,
			Category:       "Code Quality",
			Description:    "Incomplete implementation marker",
			Recommendation: "Address TODO items before production deployment",
			CWE:            0,
			OWASP:          "",
		},
	}
}

// IsAvailable always returns true as custom patterns don't require external tools
func (c *CustomPatternScanner) IsAvailable() bool {
	return true
}

// GetType returns the scanner type
func (c *CustomPatternScanner) GetType() ScannerType {
	return ScannerCustom
}

// GetVersion returns the scanner version
func (c *CustomPatternScanner) GetVersion() (string, error) {
	return "1.0.0", nil
}

// GetSupportedLanguages returns supported languages
func (c *CustomPatternScanner) GetSupportedLanguages() []string {
	return []string{"go", "javascript", "typescript", "python", "java", "php", "ruby"}
}

// GetDefaultRules returns the default pattern rules
func (c *CustomPatternScanner) GetDefaultRules() []CustomRule {
	rules := make([]CustomRule, 0, len(c.patterns))
	for _, pattern := range c.patterns {
		rules = append(rules, CustomRule{
			ID:          pattern.ID,
			Name:        pattern.Name,
			Pattern:     pattern.Pattern.String(),
			FileTypes:   pattern.FileExtensions,
			Severity:    pattern.Severity,
			Category:    pattern.Category,
			Description: pattern.Description,
			CWE:         pattern.CWE,
		})
	}
	return rules
}

// Scan performs pattern-based security scanning
func (c *CustomPatternScanner) Scan(opts ScanOptions) (*ScanResult, error) {
	startTime := time.Now()
	scanID := generateScanID()

	c.logger.Info("Starting custom pattern scan on %s", opts.Path)

	findings := make([]Finding, 0)
	filesScanned := 0
	linesScanned := 0

	// Walk through all files
	err := filepath.Walk(opts.Path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories
		if info.IsDir() {
			// Skip vendor, node_modules, etc.
			if shouldSkipDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if we should scan this file
		if !c.shouldScanFile(filePath, opts) {
			return nil
		}

		// Scan the file
		fileFindings, lines, err := c.scanFile(filePath, opts)
		if err != nil {
			c.logger.Error("Error scanning file %s: %v", filePath, err)
			return nil
		}

		findings = append(findings, fileFindings...)
		filesScanned++
		linesScanned += lines

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory: %v", err)
	}

	result := &ScanResult{
		ScanID:       scanID,
		ScannerType:  ScannerCustom,
		StartTime:    startTime,
		EndTime:      time.Now(),
		Duration:     time.Since(startTime),
		ScannedPath:  opts.Path,
		Findings:     findings,
		FilesScanned: filesScanned,
		LinesScanned: linesScanned,
		ToolVersion:  "1.0.0",
	}

	c.logger.Info("Custom pattern scan completed: %d issues found", len(findings))

	return result, nil
}

// shouldScanFile checks if a file should be scanned
func (c *CustomPatternScanner) shouldScanFile(filePath string, opts ScanOptions) bool {
	// Check excludes
	for _, exclude := range opts.ExcludePatterns {
		// Support both glob patterns and path substring matches
		if matched, _ := filepath.Match(exclude, filepath.Base(filePath)); matched {
			return false
		}
		// Check for path-based exclusions (e.g., "*/test/*" or "*test*")
		if strings.Contains(filePath, "/test/") || strings.Contains(filePath, "/tests/") ||
			strings.Contains(filePath, "/testdata/") || strings.Contains(filePath, "/fixtures/") ||
			strings.HasSuffix(filePath, "_test.go") || strings.Contains(filepath.Base(filePath), "test") {
			return false
		}
	}

	// Check includes if specified
	if len(opts.IncludePatterns) > 0 {
		included := false
		for _, include := range opts.IncludePatterns {
			if matched, _ := filepath.Match(include, filepath.Base(filePath)); matched {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}

	// Check if it's a text file we can scan
	ext := filepath.Ext(filePath)
	return isTextFile(ext)
}

// scanFile scans a single file for security patterns
func (c *CustomPatternScanner) scanFile(filePath string, opts ScanOptions) ([]Finding, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	findings := make([]Finding, 0)
	lineNumber := 0
	fileExt := filepath.Ext(filePath)

	// Determine which patterns to use based on scan type
	patterns := c.getPatternsForScanType(opts.ScanType, fileExt)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// Check each pattern
		for _, pattern := range patterns {
			if c.shouldApplyPattern(pattern, fileExt) && pattern.Pattern.MatchString(line) {
				finding := Finding{
					ID:             fmt.Sprintf("custom-%s-%s-%d", pattern.ID, filepath.Base(filePath), lineNumber),
					ScannerType:    ScannerCustom,
					RuleID:         pattern.ID,
					Severity:       pattern.Severity,
					Confidence:     "medium",
					Title:          pattern.Name,
					Description:    pattern.Description,
					Category:       pattern.Category,
					FilePath:       filePath,
					LineNumber:     lineNumber,
					CodeSnippet:    truncateCode(line, 200),
					Recommendation: pattern.Recommendation,
					CWE:            pattern.CWE,
					OWASP:          pattern.OWASP,
				}
				findings = append(findings, finding)
			}
		}
	}

	return findings, lineNumber, scanner.Err()
}

// getPatternsForScanType returns patterns based on scan type
func (c *CustomPatternScanner) getPatternsForScanType(scanType, fileExt string) []PatternRule {
	switch scanType {
	case "quick":
		// For quick scans, only use critical and high severity patterns
		filtered := make([]PatternRule, 0)
		for _, p := range c.patterns {
			if p.Severity == SeverityCritical || p.Severity == SeverityHigh {
				filtered = append(filtered, p)
			}
		}
		return filtered
	case "targeted":
		// For targeted scans, return specific patterns (would be filtered by opts.TargetedChecks)
		return c.patterns
	default:
		// Full scan - use all patterns
		return c.patterns
	}
}

// shouldApplyPattern checks if a pattern applies to a file
func (c *CustomPatternScanner) shouldApplyPattern(pattern PatternRule, fileExt string) bool {
	if len(pattern.FileExtensions) == 0 {
		return true // Pattern applies to all files
	}

	for _, ext := range pattern.FileExtensions {
		if ext == fileExt {
			return true
		}
	}
	return false
}

// Helper functions

func shouldSkipDir(name string) bool {
	skipDirs := []string{
		"vendor", "node_modules", ".git", "dist", "build",
		".cache", "coverage", ".nyc_output", ".pytest_cache",
	}

	for _, skip := range skipDirs {
		if name == skip {
			return true
		}
	}
	return false
}

func isTextFile(ext string) bool {
	textExtensions := map[string]bool{
		".go": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true,
		".py": true, ".java": true, ".c": true, ".cpp": true, ".cc": true,
		".h": true, ".hpp": true, ".cs": true, ".rb": true, ".rs": true,
		".php": true, ".swift": true, ".kt": true, ".scala": true,
		".html": true, ".htm": true, ".xml": true, ".vue": true,
		".yaml": true, ".yml": true, ".json": true, ".toml": true,
		".ini": true, ".conf": true, ".config": true, ".env": true,
		".sh": true, ".bash": true, ".zsh": true, ".fish": true,
		".sql": true, ".md": true, ".txt": true, ".dockerfile": true,
	}

	return textExtensions[strings.ToLower(ext)]
}

func truncateCode(code string, maxLength int) string {
	code = strings.TrimSpace(code)
	if len(code) <= maxLength {
		return code
	}
	return code[:maxLength] + "..."
}
