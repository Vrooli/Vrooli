package scanners

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GitleaksScanner implements the Scanner interface for gitleaks
type GitleaksScanner struct {
	logger  Logger
	binPath string
}

// NewGitleaksScanner creates a new gitleaks scanner instance
func NewGitleaksScanner(logger Logger) *GitleaksScanner {
	// Try to find gitleaks in common locations
	binPath := "gitleaks"
	if _, err := exec.LookPath(binPath); err != nil {
		// Try in ~/go/bin
		homeBinPath := filepath.Join(os.Getenv("HOME"), "go", "bin", "gitleaks")
		if _, err := os.Stat(homeBinPath); err == nil {
			binPath = homeBinPath
		}
	}

	return &GitleaksScanner{
		logger:  logger,
		binPath: binPath,
	}
}

// GitleaksFinding represents a single secret found by gitleaks
type GitleaksFinding struct {
	Description string    `json:"Description"`
	StartLine   int       `json:"StartLine"`
	EndLine     int       `json:"EndLine"`
	StartColumn int       `json:"StartColumn"`
	EndColumn   int       `json:"EndColumn"`
	Match       string    `json:"Match"`
	Secret      string    `json:"Secret"`
	File        string    `json:"File"`
	Commit      string    `json:"Commit"`
	Entropy     float64   `json:"Entropy"`
	Author      string    `json:"Author"`
	Email       string    `json:"Email"`
	Date        time.Time `json:"Date"`
	Message     string    `json:"Message"`
	Tags        []string  `json:"Tags"`
	RuleID      string    `json:"RuleID"`
	Fingerprint string    `json:"Fingerprint"`
}

// IsAvailable checks if gitleaks is installed
func (g *GitleaksScanner) IsAvailable() bool {
	cmd := exec.Command(g.binPath, "version")
	err := cmd.Run()
	return err == nil
}

// GetType returns the scanner type
func (g *GitleaksScanner) GetType() ScannerType {
	return ScannerGitleaks
}

// GetVersion returns the gitleaks version
func (g *GitleaksScanner) GetVersion() (string, error) {
	cmd := exec.Command(g.binPath, "version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse version from output (format: "vX.Y.Z")
	version := strings.TrimSpace(string(output))
	version = strings.TrimPrefix(version, "v")
	return version, nil
}

// GetSupportedLanguages returns the languages gitleaks supports
func (g *GitleaksScanner) GetSupportedLanguages() []string {
	// Gitleaks is language-agnostic, it scans any text file
	return []string{"all"}
}

// GetDefaultRules returns default gitleaks rules
func (g *GitleaksScanner) GetDefaultRules() []CustomRule {
	return []CustomRule{
		{
			ID:          "aws-access-key",
			Name:        "AWS Access Key",
			Category:    "Credential",
			Severity:    SeverityCritical,
			Description: "AWS Access Key ID",
			CWE:         798,
		},
		{
			ID:          "aws-secret-key",
			Name:        "AWS Secret Key",
			Category:    "Credential",
			Severity:    SeverityCritical,
			Description: "AWS Secret Access Key",
			CWE:         798,
		},
		{
			ID:          "github-token",
			Name:        "GitHub Token",
			Category:    "Credential",
			Severity:    SeverityCritical,
			Description: "GitHub Personal Access Token",
			CWE:         798,
		},
		{
			ID:          "private-key",
			Name:        "Private Key",
			Category:    "Cryptographic Key",
			Severity:    SeverityCritical,
			Description: "Private cryptographic key",
			CWE:         798,
		},
		{
			ID:          "api-key",
			Name:        "Generic API Key",
			Category:    "Credential",
			Severity:    SeverityHigh,
			Description: "Generic API key pattern",
			CWE:         798,
		},
		{
			ID:          "jwt",
			Name:        "JSON Web Token",
			Category:    "Token",
			Severity:    SeverityHigh,
			Description: "Hardcoded JWT token",
			CWE:         798,
		},
		{
			ID:          "slack-token",
			Name:        "Slack Token",
			Category:    "Credential",
			Severity:    SeverityHigh,
			Description: "Slack API token",
			CWE:         798,
		},
		{
			ID:          "stripe-key",
			Name:        "Stripe API Key",
			Category:    "Credential",
			Severity:    SeverityCritical,
			Description: "Stripe payment API key",
			CWE:         798,
		},
		{
			ID:          "password-in-url",
			Name:        "Password in URL",
			Category:    "Credential",
			Severity:    SeverityHigh,
			Description: "Password embedded in URL",
			CWE:         256,
		},
		{
			ID:          "generic-secret",
			Name:        "Generic Secret",
			Category:    "Credential",
			Severity:    SeverityMedium,
			Description: "Generic secret pattern",
			CWE:         798,
		},
	}
}

// Scan performs a security scan using gitleaks
func (g *GitleaksScanner) Scan(opts ScanOptions) (*ScanResult, error) {
	startTime := time.Now()
	scanID := generateScanID()

	g.logger.Info("Starting gitleaks scan on %s", opts.Path)

	// Create temporary report file
	reportFile := filepath.Join(os.TempDir(), fmt.Sprintf("gitleaks-%s.json", scanID))
	defer os.Remove(reportFile)

	// Build gitleaks command
	args := g.buildGitleaksArgs(opts, reportFile)
	cmd := exec.Command(g.binPath, args...)

	// Set timeout if specified
	if opts.Timeout > 0 {
		timer := time.AfterFunc(opts.Timeout, func() {
			cmd.Process.Kill()
		})
		defer timer.Stop()
	}

	// Capture output
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Run gitleaks
	err := cmd.Run()

	// Gitleaks returns exit code 1 if leaks are found, which is expected
	if err != nil && !strings.Contains(err.Error(), "exit status 1") {
		return nil, fmt.Errorf("gitleaks execution failed: %v - stderr: %s", err, stderr.String())
	}

	// Read and parse the report file
	findings, parseErr := g.parseGitleaksReport(reportFile)
	if parseErr != nil {
		g.logger.Error("Failed to parse gitleaks report: %v", parseErr)
		findings = []Finding{}
	}

	// Count files and lines scanned
	filesScanned, linesScanned := g.countScannedFiles(opts.Path)

	result := &ScanResult{
		ScanID:       scanID,
		ScannerType:  ScannerGitleaks,
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

	g.logger.Info("Gitleaks scan completed: %d secrets found", len(findings))

	return result, nil
}

// buildGitleaksArgs builds command line arguments for gitleaks
func (g *GitleaksScanner) buildGitleaksArgs(opts ScanOptions, reportFile string) []string {
	args := []string{
		"detect",
		"--source", opts.Path,
		"--report-format", "json",
		"--report-path", reportFile,
		"--no-banner",
		"--redact", // Redact secrets in output for safety
	}

	// Handle scan type
	switch opts.ScanType {
	case "quick":
		// For quick scans, limit depth and exclude less critical patterns
		args = append(args, "--max-target-megabytes", "10")
	case "targeted":
		// For targeted scans, focus on specific file types if provided
		if len(opts.TargetedChecks) > 0 {
			// Gitleaks doesn't support targeted rules via CLI, but we can filter results
		}
	default:
		// Full scan - check everything including git history
		args = append(args, "--log-opts", "--all")
	}

	// Add excludes
	for _, exclude := range opts.ExcludePatterns {
		args = append(args, "--exclude-path", exclude)
	}

	// Common excludes for performance
	args = append(args, "--exclude-path", "**/vendor/**")
	args = append(args, "--exclude-path", "**/node_modules/**")
	args = append(args, "--exclude-path", "**/.git/**")
	args = append(args, "--exclude-path", "**/dist/**")
	args = append(args, "--exclude-path", "**/build/**")

	return args
}

// parseGitleaksReport parses the JSON report from gitleaks
func (g *GitleaksScanner) parseGitleaksReport(reportFile string) ([]Finding, error) {
	data, err := os.ReadFile(reportFile)
	if err != nil {
		// If file doesn't exist, no secrets were found
		if os.IsNotExist(err) {
			return []Finding{}, nil
		}
		return nil, err
	}

	// Handle empty file (no secrets found)
	if len(data) == 0 {
		return []Finding{}, nil
	}

	var gitleaksFindings []GitleaksFinding
	if err := json.Unmarshal(data, &gitleaksFindings); err != nil {
		return nil, fmt.Errorf("failed to parse gitleaks JSON: %v", err)
	}

	// Convert to our Finding format
	findings := make([]Finding, 0, len(gitleaksFindings))
	for _, gf := range gitleaksFindings {
		finding := Finding{
			ID:             fmt.Sprintf("gitleaks-%s-%d", gf.Fingerprint, gf.StartLine),
			ScannerType:    ScannerGitleaks,
			RuleID:         gf.RuleID,
			Severity:       g.getSecretSeverity(gf.RuleID),
			Confidence:     "high", // Gitleaks has high confidence when it finds a match
			Title:          g.getSecretTitle(gf.RuleID, gf.Description),
			Description:    fmt.Sprintf("Potential secret found: %s", gf.Description),
			Category:       "Secret Exposure",
			FilePath:       gf.File,
			LineNumber:     gf.StartLine,
			EndLine:        gf.EndLine,
			ColumnNumber:   gf.StartColumn,
			CodeSnippet:    g.redactSecret(gf.Match),
			Recommendation: g.getSecretRecommendation(gf.RuleID),
			CWE:            798, // CWE-798: Use of Hard-coded Credentials
			OWASP:          "A07:2021 – Identification and Authentication Failures",
		}

		// Add metadata
		finding.Metadata = map[string]interface{}{
			"entropy":     gf.Entropy,
			"fingerprint": gf.Fingerprint,
		}

		// Add commit info if available
		if gf.Commit != "" {
			finding.Metadata["commit"] = gf.Commit
			finding.Metadata["author"] = gf.Author
			finding.Metadata["date"] = gf.Date
		}

		findings = append(findings, finding)
	}

	return findings, nil
}

// getSecretSeverity returns the severity for a secret type
func (g *GitleaksScanner) getSecretSeverity(ruleID string) Severity {
	criticalSecrets := []string{
		"aws-access-key", "aws-secret-key",
		"github-token", "github-pat", "github-oauth",
		"private-key", "rsa-private-key", "ssh-private-key",
		"stripe-secret-key", "stripe-publishable-key",
		"gcp-api-key", "azure-api-key",
		"jwt-secret", "jwt-base64",
	}

	highSecrets := []string{
		"api-key", "generic-api-key",
		"slack-token", "slack-webhook",
		"password-in-url", "connection-string",
		"firebase-api-key", "mailgun-api-key",
		"twilio-api-key", "sendgrid-api-key",
	}

	for _, critical := range criticalSecrets {
		if strings.Contains(strings.ToLower(ruleID), critical) {
			return SeverityCritical
		}
	}

	for _, high := range highSecrets {
		if strings.Contains(strings.ToLower(ruleID), high) {
			return SeverityHigh
		}
	}

	// Default to medium for other secrets
	return SeverityMedium
}

// getSecretTitle returns a human-readable title for a secret finding
func (g *GitleaksScanner) getSecretTitle(ruleID, description string) string {
	if description != "" {
		return description
	}

	titles := map[string]string{
		"aws-access-key":    "AWS Access Key Exposed",
		"aws-secret-key":    "AWS Secret Key Exposed",
		"github-token":      "GitHub Token Exposed",
		"github-pat":        "GitHub Personal Access Token Exposed",
		"private-key":       "Private Key Exposed",
		"api-key":           "API Key Exposed",
		"jwt":               "JWT Token Exposed",
		"slack-token":       "Slack Token Exposed",
		"stripe-secret-key": "Stripe Secret Key Exposed",
		"password-in-url":   "Password in URL",
		"generic-secret":    "Potential Secret Exposed",
		"firebase-api-key":  "Firebase API Key Exposed",
		"gcp-api-key":       "Google Cloud API Key Exposed",
		"azure-api-key":     "Azure API Key Exposed",
	}

	for key, title := range titles {
		if strings.Contains(strings.ToLower(ruleID), key) {
			return title
		}
	}

	return fmt.Sprintf("Secret Exposed: %s", ruleID)
}

// getSecretRecommendation returns remediation advice for exposed secrets
func (g *GitleaksScanner) getSecretRecommendation(ruleID string) string {
	baseRecommendation := "1. Immediately rotate/revoke the exposed credential. " +
		"2. Remove the secret from source code. " +
		"3. Use environment variables or a secrets management system. " +
		"4. Audit logs for any unauthorized access. " +
		"5. Add the file to .gitignore if appropriate."

	specificRecommendations := map[string]string{
		"aws":         "Also review AWS CloudTrail logs for unauthorized API calls and enable MFA on the account.",
		"github":      "Revoke the token immediately in GitHub Settings > Developer Settings > Personal Access Tokens.",
		"stripe":      "Generate new keys in the Stripe Dashboard and update all applications using the old key.",
		"private-key": "Generate new key pairs and update all systems using the compromised key.",
		"password":    "Change the password immediately and enable two-factor authentication where possible.",
		"jwt":         "Rotate the JWT signing secret and invalidate all existing tokens.",
	}

	for key, specific := range specificRecommendations {
		if strings.Contains(strings.ToLower(ruleID), key) {
			return fmt.Sprintf("%s\n\nSpecific steps: %s", baseRecommendation, specific)
		}
	}

	return baseRecommendation
}

// redactSecret partially redacts a secret for safe display
func (g *GitleaksScanner) redactSecret(secret string) string {
	if len(secret) <= 8 {
		return "***REDACTED***"
	}

	// Show first 3 and last 3 characters
	return fmt.Sprintf("%s***REDACTED***%s",
		secret[:3],
		secret[len(secret)-3:])
}

// countScannedFiles counts files and lines in the scanned path
func (g *GitleaksScanner) countScannedFiles(path string) (int, int) {
	fileCount := 0
	lineCount := 0

	// Common text file extensions to count
	textExtensions := map[string]bool{
		".go": true, ".js": true, ".ts": true, ".tsx": true,
		".py": true, ".java": true, ".c": true, ".cpp": true,
		".h": true, ".hpp": true, ".cs": true, ".rb": true,
		".php": true, ".swift": true, ".kt": true, ".rs": true,
		".yaml": true, ".yml": true, ".json": true, ".xml": true,
		".toml": true, ".ini": true, ".conf": true, ".config": true,
		".env": true, ".properties": true, ".sh": true, ".bash": true,
		".sql": true, ".md": true, ".txt": true,
	}

	filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip vendor and other excluded directories
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == "node_modules" || name == ".git" ||
				name == "dist" || name == "build" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if it's a text file we should count
		ext := filepath.Ext(filePath)
		if textExtensions[ext] {
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
