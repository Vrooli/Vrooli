package autosteer

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SecurityMetricsCollector handles collection of security metrics
type SecurityMetricsCollector struct {
	projectRoot string
}

// NewSecurityMetricsCollector creates a new security metrics collector
func NewSecurityMetricsCollector(projectRoot string) *SecurityMetricsCollector {
	return &SecurityMetricsCollector{
		projectRoot: projectRoot,
	}
}

// Collect gathers all security metrics for a scenario
func (c *SecurityMetricsCollector) Collect(scenarioName string) (*SecurityMetrics, error) {
	metrics := &SecurityMetrics{}

	// Collect each metric independently (best effort)
	metrics.VulnerabilityCount = c.getVulnerabilityCount(scenarioName)
	metrics.InputValidationCoverage = c.assessInputValidation(scenarioName)
	metrics.AuthImplementationScore = c.assessAuthImplementation(scenarioName)
	metrics.SecurityScanScore = c.calculateSecurityScore(metrics.VulnerabilityCount)

	return metrics, nil
}

// getVulnerabilityCount runs vulnerability scans on dependencies
func (c *SecurityMetricsCollector) getVulnerabilityCount(scenarioName string) int {
	npmVulns := c.getNPMVulnerabilities(scenarioName)
	goVulns := c.getGoVulnerabilities(scenarioName)

	return npmVulns + goVulns
}

// getNPMVulnerabilities scans npm dependencies for vulnerabilities
func (c *SecurityMetricsCollector) getNPMVulnerabilities(scenarioName string) int {
	uiPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "ui")

	if _, err := os.Stat(filepath.Join(uiPath, "package.json")); os.IsNotExist(err) {
		return 0
	}

	// Run npm audit
	cmd := exec.Command("npm", "audit", "--json")
	cmd.Dir = uiPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		// npm audit returns non-zero if vulnerabilities found
		// Continue parsing output
	}

	var audit struct {
		Metadata struct {
			Vulnerabilities struct {
				Info     int `json:"info"`
				Low      int `json:"low"`
				Moderate int `json:"moderate"`
				High     int `json:"high"`
				Critical int `json:"critical"`
				Total    int `json:"total"`
			} `json:"vulnerabilities"`
		} `json:"metadata"`
	}

	if err := json.Unmarshal(output, &audit); err != nil {
		return 0
	}

	// Weight vulnerabilities by severity
	weighted := audit.Metadata.Vulnerabilities.Critical*5 +
		audit.Metadata.Vulnerabilities.High*3 +
		audit.Metadata.Vulnerabilities.Moderate*2 +
		audit.Metadata.Vulnerabilities.Low

	return weighted
}

// getGoVulnerabilities scans Go dependencies for vulnerabilities
func (c *SecurityMetricsCollector) getGoVulnerabilities(scenarioName string) int {
	apiPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "api")

	if _, err := os.Stat(filepath.Join(apiPath, "go.mod")); os.IsNotExist(err) {
		return 0
	}

	// Check if govulncheck is available
	if _, err := exec.LookPath("govulncheck"); err != nil {
		return 0
	}

	// Run govulncheck
	cmd := exec.Command("govulncheck", "-json", "./...")
	cmd.Dir = apiPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		// govulncheck returns non-zero if vulnerabilities found
	}

	// Count vulnerabilities from output
	count := 0
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.Contains(line, `"finding"`) || strings.Contains(line, "vulnerability") {
			count++
		}
	}

	return count
}

// assessInputValidation estimates input validation coverage
func (c *SecurityMetricsCollector) assessInputValidation(scenarioName string) float64 {
	// Analyze code for input validation patterns
	apiCoverage := c.assessGoInputValidation(scenarioName)
	uiCoverage := c.assessTSInputValidation(scenarioName)

	// Average if both exist
	if apiCoverage > 0 && uiCoverage > 0 {
		return (apiCoverage + uiCoverage) / 2.0
	} else if apiCoverage > 0 {
		return apiCoverage
	} else if uiCoverage > 0 {
		return uiCoverage
	}

	return 0
}

// assessGoInputValidation checks Go code for input validation
func (c *SecurityMetricsCollector) assessGoInputValidation(scenarioName string) float64 {
	apiPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "api")

	if _, err := os.Stat(apiPath); os.IsNotExist(err) {
		return 0
	}

	totalHandlers := 0
	validatedHandlers := 0

	// Walk through Go files
	filepath.Walk(apiPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".go" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fileContent := string(content)

		// Count HTTP handlers
		if strings.Contains(fileContent, "http.ResponseWriter") ||
			strings.Contains(fileContent, "gin.Context") ||
			strings.Contains(fileContent, "echo.Context") {
			totalHandlers++

			// Check for validation patterns
			validationPatterns := []string{
				"Validate()",
				"validator.",
				"if err :=",
				"return err",
				"StatusBadRequest",
				"invalid",
			}

			hasValidation := false
			for _, pattern := range validationPatterns {
				if strings.Contains(fileContent, pattern) {
					hasValidation = true
					break
				}
			}

			if hasValidation {
				validatedHandlers++
			}
		}

		return nil
	})

	if totalHandlers == 0 {
		return 100.0 // No handlers = no validation needed
	}

	return (float64(validatedHandlers) / float64(totalHandlers)) * 100
}

// assessTSInputValidation checks TypeScript code for input validation
func (c *SecurityMetricsCollector) assessTSInputValidation(scenarioName string) float64 {
	srcPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "ui", "src")

	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return 0
	}

	totalForms := 0
	validatedForms := 0

	filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".tsx" && ext != ".ts" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fileContent := string(content)

		// Count form components
		if strings.Contains(fileContent, "<form") ||
			strings.Contains(fileContent, "onSubmit") {
			totalForms++

			// Check for validation patterns
			validationPatterns := []string{
				"zod",
				"yup",
				"validate",
				"schema",
				"required",
				"pattern",
				"min",
				"max",
			}

			hasValidation := false
			for _, pattern := range validationPatterns {
				if strings.Contains(fileContent, pattern) {
					hasValidation = true
					break
				}
			}

			if hasValidation {
				validatedForms++
			}
		}

		return nil
	})

	if totalForms == 0 {
		return 100.0
	}

	return (float64(validatedForms) / float64(totalForms)) * 100
}

// assessAuthImplementation checks for proper auth implementation
func (c *SecurityMetricsCollector) assessAuthImplementation(scenarioName string) float64 {
	score := 100.0

	// Check for auth files/patterns
	hasAuthMiddleware := c.hasAuthMiddleware(scenarioName)
	hasAuthRoutes := c.hasAuthRoutes(scenarioName)
	hasPasswordHashing := c.hasPasswordHashing(scenarioName)
	hasJWT := c.hasJWT(scenarioName)
	hasSessionManagement := c.hasSessionManagement(scenarioName)

	// Deduct points for missing components
	if !hasAuthMiddleware {
		score -= 20
	}
	if !hasAuthRoutes {
		score -= 20
	}
	if !hasPasswordHashing {
		score -= 25
	}
	if !hasJWT && !hasSessionManagement {
		score -= 20
	}

	if score < 0 {
		score = 0
	}

	return score
}

// hasAuthMiddleware checks for authentication middleware
func (c *SecurityMetricsCollector) hasAuthMiddleware(scenarioName string) bool {
	apiPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "api")

	found := false
	filepath.Walk(apiPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || found {
			return nil
		}

		if filepath.Ext(path) != ".go" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fileContent := string(content)

		patterns := []string{"AuthMiddleware", "authenticate", "requireAuth", "verifyToken"}
		for _, pattern := range patterns {
			if strings.Contains(fileContent, pattern) {
				found = true
				return nil
			}
		}

		return nil
	})

	return found
}

// hasAuthRoutes checks for authentication routes
func (c *SecurityMetricsCollector) hasAuthRoutes(scenarioName string) bool {
	apiPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "api")

	found := false
	filepath.Walk(apiPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || found {
			return nil
		}

		if filepath.Ext(path) != ".go" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fileContent := string(content)

		patterns := []string{"/login", "/register", "/auth", "Login", "Register"}
		for _, pattern := range patterns {
			if strings.Contains(fileContent, pattern) {
				found = true
				return nil
			}
		}

		return nil
	})

	return found
}

// hasPasswordHashing checks for password hashing implementation
func (c *SecurityMetricsCollector) hasPasswordHashing(scenarioName string) bool {
	apiPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "api")

	found := false
	filepath.Walk(apiPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || found {
			return nil
		}

		if filepath.Ext(path) != ".go" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fileContent := string(content)

		patterns := []string{"bcrypt", "argon2", "scrypt", "HashPassword", "CompareHash"}
		for _, pattern := range patterns {
			if strings.Contains(fileContent, pattern) {
				found = true
				return nil
			}
		}

		return nil
	})

	return found
}

// hasJWT checks for JWT implementation
func (c *SecurityMetricsCollector) hasJWT(scenarioName string) bool {
	apiPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "api")

	found := false
	filepath.Walk(apiPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || found {
			return nil
		}

		if filepath.Ext(path) != ".go" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fileContent := string(content)

		patterns := []string{"jwt", "JWT", "GenerateToken", "VerifyToken"}
		for _, pattern := range patterns {
			if strings.Contains(fileContent, pattern) {
				found = true
				return nil
			}
		}

		return nil
	})

	return found
}

// hasSessionManagement checks for session management
func (c *SecurityMetricsCollector) hasSessionManagement(scenarioName string) bool {
	apiPath := filepath.Join(c.projectRoot, "scenarios", scenarioName, "api")

	found := false
	filepath.Walk(apiPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || found {
			return nil
		}

		if filepath.Ext(path) != ".go" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fileContent := string(content)

		patterns := []string{"session", "Session", "cookie", "Cookie"}
		for _, pattern := range patterns {
			if strings.Contains(fileContent, pattern) {
				found = true
				return nil
			}
		}

		return nil
	})

	return found
}

// calculateSecurityScore calculates overall security score based on vulnerabilities
func (c *SecurityMetricsCollector) calculateSecurityScore(vulnerabilityCount int) float64 {
	score := 100.0

	// Deduct points based on vulnerability count
	score -= float64(vulnerabilityCount) * 5.0

	if score < 0 {
		score = 0
	}

	return score
}

// RunSecurityScan runs a comprehensive security scan
func (c *SecurityMetricsCollector) RunSecurityScan(scenarioName string) (*SecurityReport, error) {
	report := &SecurityReport{
		ScenarioName: scenarioName,
	}

	// Collect all security metrics
	metrics, err := c.Collect(scenarioName)
	if err != nil {
		return nil, err
	}

	report.Metrics = *metrics

	// Add detailed findings
	report.Findings = c.collectSecurityFindings(scenarioName)

	return report, nil
}

// SecurityReport represents a detailed security assessment
type SecurityReport struct {
	ScenarioName string
	Metrics      SecurityMetrics
	Findings     []SecurityFinding
}

// SecurityFinding represents a specific security issue
type SecurityFinding struct {
	Severity    string
	Category    string
	Description string
	Location    string
	Remediation string
}

// collectSecurityFindings collects detailed security findings
func (c *SecurityMetricsCollector) collectSecurityFindings(scenarioName string) []SecurityFinding {
	findings := []SecurityFinding{}

	// Check for common security issues
	findings = append(findings, c.checkForHardcodedSecrets(scenarioName)...)
	findings = append(findings, c.checkForInsecureConfigs(scenarioName)...)

	return findings
}

// checkForHardcodedSecrets checks for hardcoded secrets
func (c *SecurityMetricsCollector) checkForHardcodedSecrets(scenarioName string) []SecurityFinding {
	findings := []SecurityFinding{}
	scenarioPath := filepath.Join(c.projectRoot, "scenarios", scenarioName)

	filepath.Walk(scenarioPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Skip certain directories
		if strings.Contains(path, "node_modules") || strings.Contains(path, ".git") {
			return nil
		}

		ext := filepath.Ext(path)
		validExts := map[string]bool{
			".go": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true,
			".env": true, ".config": true, ".json": true,
		}

		if !validExts[ext] {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fileContent := strings.ToLower(string(content))

		// Check for secret patterns
		secretPatterns := []string{
			"password =",
			"api_key =",
			"secret =",
			"token =",
			"private_key =",
		}

		for _, pattern := range secretPatterns {
			if strings.Contains(fileContent, pattern) {
				findings = append(findings, SecurityFinding{
					Severity:    "HIGH",
					Category:    "Hardcoded Secrets",
					Description: fmt.Sprintf("Potential hardcoded secret found: %s", pattern),
					Location:    path,
					Remediation: "Use environment variables or secret management service",
				})
			}
		}

		return nil
	})

	return findings
}

// checkForInsecureConfigs checks for insecure configurations
func (c *SecurityMetricsCollector) checkForInsecureConfigs(scenarioName string) []SecurityFinding {
	findings := []SecurityFinding{}
	scenarioPath := filepath.Join(c.projectRoot, "scenarios", scenarioName)

	// Check for exposed .env files
	envPath := filepath.Join(scenarioPath, ".env")
	if _, err := os.Stat(envPath); err == nil {
		findings = append(findings, SecurityFinding{
			Severity:    "MEDIUM",
			Category:    "Configuration Security",
			Description: ".env file found - ensure it's in .gitignore",
			Location:    envPath,
			Remediation: "Add .env to .gitignore and use .env.example for documentation",
		})
	}

	return findings
}
