package scanners

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GosecScanner implements the Scanner interface for gosec
type GosecScanner struct {
	logger  Logger
	binPath string
}

// NewGosecScanner creates a new gosec scanner instance
func NewGosecScanner(logger Logger) *GosecScanner {
	// Try to find gosec in common locations
	binPath := "gosec"
	if _, err := exec.LookPath(binPath); err != nil {
		// Try in ~/go/bin
		homeBinPath := filepath.Join(os.Getenv("HOME"), "go", "bin", "gosec")
		if _, err := os.Stat(homeBinPath); err == nil {
			binPath = homeBinPath
		}
	}
	
	return &GosecScanner{
		logger:  logger,
		binPath: binPath,
	}
}

// GosecIssue represents a single issue found by gosec
type GosecIssue struct {
	Severity   string `json:"severity"`
	Confidence string `json:"confidence"`
	CWE        struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"cwe"`
	RuleID  string `json:"rule_id"`
	Details string `json:"details"`
	File    string `json:"file"`
	Code    string `json:"code"`
	Line    string `json:"line"`
	Column  string `json:"column"`
}

// GosecReport represents the JSON output from gosec
type GosecReport struct {
	Issues []GosecIssue          `json:"Issues"`
	Stats  map[string]interface{} `json:"Stats"`
}

// IsAvailable checks if gosec is installed
func (g *GosecScanner) IsAvailable() bool {
	cmd := exec.Command(g.binPath, "-version")
	err := cmd.Run()
	return err == nil
}

// GetType returns the scanner type
func (g *GosecScanner) GetType() ScannerType {
	return ScannerGosec
}

// GetVersion returns the gosec version
func (g *GosecScanner) GetVersion() (string, error) {
	cmd := exec.Command(g.binPath, "-version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	// Parse version from output
	versionRegex := regexp.MustCompile(`Version:\s*([0-9.]+)`)
	matches := versionRegex.FindSubmatch(output)
	if len(matches) > 1 {
		return string(matches[1]), nil
	}
	
	return "unknown", nil
}

// GetSupportedLanguages returns the languages gosec supports
func (g *GosecScanner) GetSupportedLanguages() []string {
	return []string{"go"}
}

// GetDefaultRules returns default gosec rules
func (g *GosecScanner) GetDefaultRules() []CustomRule {
	return []CustomRule{
		{
			ID:          "G101",
			Name:        "Hardcoded credentials",
			Category:    "Security",
			Severity:    SeverityHigh,
			Description: "Look for hardcoded credentials",
			CWE:         798,
		},
		{
			ID:          "G102",
			Name:        "Bind to all interfaces",
			Category:    "Network",
			Severity:    SeverityMedium,
			Description: "Binding to all network interfaces",
			CWE:         200,
		},
		{
			ID:          "G103",
			Name:        "Unsafe use of unsafe block",
			Category:    "Memory",
			Severity:    SeverityMedium,
			Description: "Audit the use of unsafe block",
			CWE:         242,
		},
		{
			ID:          "G104",
			Name:        "Unhandled errors",
			Category:    "Error Handling",
			Severity:    SeverityLow,
			Description: "Audit errors not checked",
			CWE:         703,
		},
		{
			ID:          "G201",
			Name:        "SQL injection via format string",
			Category:    "SQL Injection",
			Severity:    SeverityCritical,
			Description: "SQL query construction using format string",
			CWE:         89,
		},
		{
			ID:          "G202",
			Name:        "SQL injection via string concatenation",
			Category:    "SQL Injection",
			Severity:    SeverityCritical,
			Description: "SQL query construction using string concatenation",
			CWE:         89,
		},
		{
			ID:          "G301",
			Name:        "Poor file permissions",
			Category:    "File System",
			Severity:    SeverityMedium,
			Description: "Poor file permissions used when creating a directory",
			CWE:         276,
		},
		{
			ID:          "G302",
			Name:        "Poor file permissions",
			Category:    "File System",
			Severity:    SeverityMedium,
			Description: "Poor file permissions used when creation file or using chmod",
			CWE:         276,
		},
		{
			ID:          "G303",
			Name:        "Predictable temp file",
			Category:    "File System",
			Severity:    SeverityMedium,
			Description: "Creating tempfile using a predictable path",
			CWE:         377,
		},
		{
			ID:          "G304",
			Name:        "File path traversal",
			Category:    "Path Traversal",
			Severity:    SeverityHigh,
			Description: "File path provided as taint input",
			CWE:         22,
		},
		{
			ID:          "G305",
			Name:        "File path traversal",
			Category:    "Path Traversal",
			Severity:    SeverityMedium,
			Description: "File traversal when extracting zip archive",
			CWE:         22,
		},
		{
			ID:          "G401",
			Name:        "Use of weak cryptographic primitive",
			Category:    "Cryptography",
			Severity:    SeverityHigh,
			Description: "The use of DES, MD5 or SHA1",
			CWE:         326,
		},
		{
			ID:          "G501",
			Name:        "Blacklisted import crypto/md5",
			Category:    "Cryptography",
			Severity:    SeverityHigh,
			Description: "Do not import crypto/md5",
			CWE:         327,
		},
		{
			ID:          "G601",
			Name:        "Implicit memory aliasing",
			Category:    "Memory",
			Severity:    SeverityMedium,
			Description: "Implicit memory aliasing in for loop",
			CWE:         118,
		},
	}
}

// Scan performs a security scan using gosec
func (g *GosecScanner) Scan(opts ScanOptions) (*ScanResult, error) {
	startTime := time.Now()
	scanID := generateScanID()
	
	g.logger.Info("Starting gosec scan on %s", opts.Path)
	
	// Build gosec command
	args := g.buildGosecArgs(opts)
	cmd := exec.Command(g.binPath, args...)
	
	// Set timeout if specified
	if opts.Timeout > 0 {
		timer := time.AfterFunc(opts.Timeout, func() {
			cmd.Process.Kill()
		})
		defer timer.Stop()
	}
	
	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	// Run gosec
	err := cmd.Run()
	
	// Gosec returns non-zero exit code if issues are found, which is not an error
	if err != nil && !strings.Contains(err.Error(), "exit status") {
		return nil, fmt.Errorf("gosec execution failed: %v - stderr: %s", err, stderr.String())
	}
	
	// Parse gosec output
	report, parseErr := g.parseGosecOutput(stdout.Bytes())
	if parseErr != nil {
		g.logger.Error("Failed to parse gosec output: %v", parseErr)
		// Try to return partial results
		report = &GosecReport{Issues: []GosecIssue{}}
	}
	
	// Convert to our Finding format
	findings := g.convertToFindings(report)
	
	// Count files and lines scanned
	filesScanned, linesScanned := g.countScannedFiles(opts.Path)
	
	result := &ScanResult{
		ScanID:       scanID,
		ScannerType:  ScannerGosec,
		StartTime:    startTime,
		EndTime:      time.Now(),
		Duration:     time.Since(startTime),
		ScannedPath:  opts.Path,
		Findings:     findings,
		FilesScanned: filesScanned,
		LinesScanned: linesScanned,
	}
	
	// Get tool version
	if version, err := g.GetVersion(); err == nil {
		result.ToolVersion = version
	}
	
	g.logger.Info("Gosec scan completed: %d issues found", len(findings))
	
	return result, nil
}

// buildGosecArgs builds command line arguments for gosec
func (g *GosecScanner) buildGosecArgs(opts ScanOptions) []string {
	args := []string{
		"-fmt", "json", // JSON output for parsing
		"-confidence", "medium", // Include medium confidence issues
		"-severity", "low", // Include all severity levels
		"-quiet", // Suppress non-issue output
	}
	
	// Handle scan type
	switch opts.ScanType {
	case "quick":
		// For quick scans, only check critical rules
		args = append(args, "-include=G201,G202,G301,G401,G501")
	case "targeted":
		// For targeted scans, use specific checks if provided
		if len(opts.TargetedChecks) > 0 {
			includes := strings.Join(opts.TargetedChecks, ",")
			args = append(args, "-include="+includes)
		}
	default:
		// Full scan - use all rules
	}
	
	// Exclude vendor directories and test files for speed
	args = append(args, "-exclude-dir=vendor")
	args = append(args, "-exclude-dir=node_modules")
	args = append(args, "-exclude-dir=.git")
	
	// Add additional excludes from options
	for _, exclude := range opts.ExcludePatterns {
		args = append(args, "-exclude="+exclude)
	}
	
	// Add the path to scan
	args = append(args, opts.Path+"/...")
	
	return args
}

// parseGosecOutput parses the JSON output from gosec
func (g *GosecScanner) parseGosecOutput(output []byte) (*GosecReport, error) {
	var report GosecReport
	
	// Find the JSON part of the output (gosec sometimes outputs text before JSON)
	jsonStart := bytes.IndexByte(output, '{')
	if jsonStart == -1 {
		// Try to find array start
		jsonStart = bytes.IndexByte(output, '[')
		if jsonStart == -1 {
			return nil, fmt.Errorf("no JSON found in gosec output")
		}
	}
	
	jsonOutput := output[jsonStart:]
	
	if err := json.Unmarshal(jsonOutput, &report); err != nil {
		// Try alternative format
		var issues struct {
			Issues []GosecIssue `json:"Issues"`
		}
		if err := json.Unmarshal(jsonOutput, &issues); err != nil {
			return nil, fmt.Errorf("failed to parse gosec JSON: %v", err)
		}
		report.Issues = issues.Issues
	}
	
	return &report, nil
}

// convertToFindings converts gosec issues to our Finding format
func (g *GosecScanner) convertToFindings(report *GosecReport) []Finding {
	findings := make([]Finding, 0, len(report.Issues))
	
	for _, issue := range report.Issues {
		finding := Finding{
			ID:          fmt.Sprintf("gosec-%s-%s-%s", issue.RuleID, issue.File, issue.Line),
			ScannerType: ScannerGosec,
			RuleID:      issue.RuleID,
			Severity:    g.mapGosecSeverity(issue.Severity),
			Confidence:  strings.ToLower(issue.Confidence),
			Title:       g.getRuleTitle(issue.RuleID),
			Description: issue.Details,
			Category:    g.getRuleCategory(issue.RuleID),
			FilePath:    issue.File,
			CodeSnippet: issue.Code,
			Recommendation: g.getRuleRecommendation(issue.RuleID),
		}
		
		// Parse line number
		if lineNum, err := strconv.Atoi(issue.Line); err == nil {
			finding.LineNumber = lineNum
		}
		
		// Parse column if available
		if colNum, err := strconv.Atoi(issue.Column); err == nil {
			finding.ColumnNumber = colNum
		}
		
		// Add CWE if available
		if issue.CWE.ID != "" {
			if cweNum, err := strconv.Atoi(strings.TrimPrefix(issue.CWE.ID, "CWE-")); err == nil {
				finding.CWE = cweNum
			}
			finding.References = append(finding.References, issue.CWE.URL)
		}
		
		// Map to OWASP Top 10
		finding.OWASP = g.mapToOWASP(issue.RuleID)
		
		findings = append(findings, finding)
	}
	
	return findings
}

// mapGosecSeverity maps gosec severity to our severity levels
func (g *GosecScanner) mapGosecSeverity(gosecSeverity string) Severity {
	switch strings.ToUpper(gosecSeverity) {
	case "HIGH":
		return SeverityHigh
	case "MEDIUM":
		return SeverityMedium
	case "LOW":
		return SeverityLow
	default:
		return SeverityInfo
	}
}

// getRuleTitle returns a human-readable title for a gosec rule
func (g *GosecScanner) getRuleTitle(ruleID string) string {
	titles := map[string]string{
		"G101": "Hardcoded Credentials",
		"G102": "Bind to All Interfaces",
		"G103": "Unsafe Use of unsafe Block",
		"G104": "Unhandled Error",
		"G201": "SQL Injection via Format String",
		"G202": "SQL Injection via String Concatenation",
		"G301": "Poor Directory Permissions",
		"G302": "Poor File Permissions",
		"G303": "Predictable Temp File",
		"G304": "File Path Traversal",
		"G305": "Zip Archive Path Traversal",
		"G401": "Use of Weak Cryptographic Algorithm",
		"G501": "Blacklisted Import: crypto/md5",
		"G502": "Blacklisted Import: crypto/des",
		"G503": "Blacklisted Import: crypto/rc4",
		"G504": "Blacklisted Import: net/http/cgi",
		"G601": "Implicit Memory Aliasing in Loop",
	}
	
	if title, exists := titles[ruleID]; exists {
		return title
	}
	return fmt.Sprintf("Security Issue: %s", ruleID)
}

// getRuleCategory returns the category for a gosec rule
func (g *GosecScanner) getRuleCategory(ruleID string) string {
	categories := map[string]string{
		"G101": "Credential Exposure",
		"G102": "Network Security",
		"G103": "Memory Safety",
		"G104": "Error Handling",
		"G201": "SQL Injection",
		"G202": "SQL Injection",
		"G301": "Access Control",
		"G302": "Access Control",
		"G303": "File System",
		"G304": "Path Traversal",
		"G305": "Path Traversal",
		"G401": "Cryptography",
		"G501": "Cryptography",
		"G502": "Cryptography",
		"G503": "Cryptography",
		"G504": "Deprecated",
		"G601": "Memory Safety",
	}
	
	if category, exists := categories[ruleID]; exists {
		return category
	}
	return "Security"
}

// getRuleRecommendation returns remediation advice for a gosec rule
func (g *GosecScanner) getRuleRecommendation(ruleID string) string {
	recommendations := map[string]string{
		"G101": "Store credentials in environment variables or a secure secrets management system. Never hardcode credentials in source code.",
		"G102": "Bind to specific interfaces instead of 0.0.0.0 or empty string. Use 127.0.0.1 for local-only services.",
		"G103": "Avoid using unsafe blocks unless absolutely necessary. Document why unsafe is required and ensure proper bounds checking.",
		"G104": "Always check and handle errors appropriately. Use _ explicitly if you intentionally ignore an error.",
		"G201": "Use parameterized queries with placeholders ($1, $2) instead of fmt.Sprintf for SQL queries.",
		"G202": "Use parameterized queries instead of string concatenation for SQL queries to prevent injection attacks.",
		"G301": "Use restrictive permissions (0750 or stricter) when creating directories.",
		"G302": "Use restrictive permissions (0640 or stricter) when creating files.",
		"G303": "Use ioutil.TempFile() or os.CreateTemp() to create temporary files with unpredictable names.",
		"G304": "Validate and sanitize file paths. Use filepath.Clean() and ensure paths don't escape the intended directory.",
		"G305": "Validate extracted file paths to prevent directory traversal when extracting archives.",
		"G401": "Use strong cryptographic algorithms like AES, SHA-256, or stronger. Avoid MD5, SHA1, and DES.",
		"G501": "Use crypto/sha256 or stronger instead of crypto/md5 for cryptographic purposes.",
		"G601": "Be careful with implicit memory aliasing in loops. Use index to access elements or explicitly copy values.",
	}
	
	if recommendation, exists := recommendations[ruleID]; exists {
		return recommendation
	}
	return "Review and fix the security issue according to best practices."
}

// mapToOWASP maps gosec rules to OWASP Top 10 categories
func (g *GosecScanner) mapToOWASP(ruleID string) string {
	owaspMap := map[string]string{
		"G101": "A07:2021 – Identification and Authentication Failures",
		"G102": "A05:2021 – Security Misconfiguration",
		"G103": "A06:2021 – Vulnerable and Outdated Components",
		"G104": "A09:2021 – Security Logging and Monitoring Failures",
		"G201": "A03:2021 – Injection",
		"G202": "A03:2021 – Injection",
		"G301": "A01:2021 – Broken Access Control",
		"G302": "A01:2021 – Broken Access Control",
		"G303": "A01:2021 – Broken Access Control",
		"G304": "A01:2021 – Broken Access Control",
		"G305": "A01:2021 – Broken Access Control",
		"G401": "A02:2021 – Cryptographic Failures",
		"G501": "A02:2021 – Cryptographic Failures",
		"G601": "A03:2021 – Injection",
	}
	
	if owasp, exists := owaspMap[ruleID]; exists {
		return owasp
	}
	return ""
}

// countScannedFiles counts the number of Go files and lines in the scanned path
func (g *GosecScanner) countScannedFiles(path string) (int, int) {
	fileCount := 0
	lineCount := 0
	
	filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		// Skip vendor and other excluded directories
		if info.IsDir() && (info.Name() == "vendor" || info.Name() == "node_modules" || info.Name() == ".git") {
			return filepath.SkipDir
		}
		
		// Count Go files
		if strings.HasSuffix(filePath, ".go") && !strings.Contains(filePath, "_test.go") {
			fileCount++
			
			// Count lines
			if content, err := os.ReadFile(filePath); err == nil {
				lines := bytes.Count(content, []byte("\n"))
				lineCount += lines
			}
		}
		
		return nil
	})
	
	return fileCount, lineCount
}