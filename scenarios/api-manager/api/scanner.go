package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// VulnerabilityScanner performs security and quality scans on scenario APIs
type VulnerabilityScanner struct {
	db     *sql.DB
	logger *Logger
}

// NewVulnerabilityScanner creates a new vulnerability scanner
func NewVulnerabilityScanner(db *sql.DB) *VulnerabilityScanner {
	return &VulnerabilityScanner{
		db:     db,
		logger: NewLogger(),
	}
}

// HTTPBodyLeakVulnerability represents the critical HTTP response body leak issue
type HTTPBodyLeakVulnerability struct {
	FilePath     string
	LineNumber   int
	FunctionName string
	CodeSnippet  string
	HTTPMethod   string // Get, Post, etc.
	HasDefer     bool
	Severity     string
}

// SecurityAuditResult represents the result of a security audit
type SecurityAuditResult struct {
	ScenarioName string          `json:"scenario_name"`
	ScenarioPath string          `json:"scenario_path"`
	AuditTime    time.Time       `json:"audit_time"`
	Issues       []SecurityIssue `json:"issues"`
}

// SecurityIssue represents a security issue found during audit
type SecurityIssue struct {
	ID             uuid.UUID `json:"id"`
	Severity       string    `json:"severity"` // CRITICAL, HIGH, MEDIUM, LOW
	Category       string    `json:"category"`
	Description    string    `json:"description"`
	FilePath       string    `json:"file_path,omitempty"`
	LineNumber     int       `json:"line_number,omitempty"`
	CodeSnippet    string    `json:"code_snippet,omitempty"`
	Recommendation string    `json:"recommendation,omitempty"`
	Status         string    `json:"status"` // open, fixed, ignored
	Source         string    `json:"source,omitempty"` // Pattern Scan, AI Analysis
}

// ScanResult represents the result of a vulnerability scan
type ScanResult struct {
	ScenarioName    string          `json:"scenario_name"`
	ScenarioPath    string          `json:"scenario_path"`
	ScanTime        time.Time       `json:"scan_time"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
}

// Vulnerability represents a vulnerability found during scanning
type Vulnerability struct {
	ID             uuid.UUID  `json:"id"`
	Type           string     `json:"type"`
	Severity       string     `json:"severity"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	FilePath       string     `json:"file_path,omitempty"`
	LineNumber     int        `json:"line_number,omitempty"`
	CodeSnippet    string     `json:"code_snippet,omitempty"`
	Impact         string     `json:"impact,omitempty"`
	Recommendation string     `json:"recommendation,omitempty"`
	Status         string     `json:"status"`
	Category       string     `json:"category"`
}

// ScanScenario performs comprehensive vulnerability scanning on a scenario
func (vs *VulnerabilityScanner) ScanScenario(scenarioPath, scenarioName string) (*ScanResult, error) {
	vs.logger.Info(fmt.Sprintf("Starting comprehensive scan for scenario: %s", scenarioName))
	
	result := &ScanResult{
		ScenarioName: scenarioName,
		ScenarioPath: scenarioPath,
		ScanTime:     time.Now(),
		Vulnerabilities: []Vulnerability{},
	}

	// Find all Go files in the scenario
	goFiles, err := vs.findGoFiles(scenarioPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find Go files: %w", err)
	}

	// Scan each file for vulnerabilities
	for _, file := range goFiles {
		vs.logger.Info(fmt.Sprintf("Scanning file: %s", file))
		
		// CRITICAL: Check for unclosed HTTP response bodies
		httpVulns, err := vs.scanForHTTPBodyLeaks(file)
		if err != nil {
			vs.logger.Error(fmt.Sprintf("Failed to scan %s for HTTP body leaks", file), err)
			continue
		}
		
		for _, vuln := range httpVulns {
			result.Vulnerabilities = append(result.Vulnerabilities, Vulnerability{
				ID:          uuid.New(),
				Type:        "HTTP_RESPONSE_BODY_LEAK",
				Severity:    "CRITICAL",
				Title:       "Unclosed HTTP Response Body - TCP Connection Leak",
				Description: vs.generateHTTPLeakDescription(vuln),
				FilePath:    vuln.FilePath,
				LineNumber:  vuln.LineNumber,
				CodeSnippet: vuln.CodeSnippet,
				Impact:      "CATASTROPHIC: Each unclosed response body leaks a TCP connection. This causes connection exhaustion, system instability, and can crash the entire Vrooli platform within hours.",
				Recommendation: vs.generateHTTPLeakFix(vuln),
				Status:      "open",
				Category:    "Resource Leak",
			})
		}
		
		// Check for other common issues
		otherVulns := vs.scanForCommonIssues(file)
		result.Vulnerabilities = append(result.Vulnerabilities, otherVulns...)
	}
	
	// Store results in database
	if err := vs.storeResults(result); err != nil {
		vs.logger.Error("Failed to store scan results", err)
	}
	
	vs.logger.Info(fmt.Sprintf("Scan completed. Found %d vulnerabilities", len(result.Vulnerabilities)))
	return result, nil
}

// SecurityAudit performs a comprehensive security audit using pattern matching and AI analysis
func (vs *VulnerabilityScanner) SecurityAudit(scenarioPath, scenarioName string) (*SecurityAuditResult, error) {
	vs.logger.Info(fmt.Sprintf("Starting security audit for scenario: %s", scenarioName))
	
	result := &SecurityAuditResult{
		ScenarioName: scenarioName,
		ScenarioPath: scenarioPath,
		AuditTime:    time.Now(),
		Issues:       []SecurityIssue{},
	}

	// Find all Go files in the scenario
	goFiles, err := vs.findGoFiles(scenarioPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find Go files: %w", err)
	}

	for _, file := range goFiles {
		vs.logger.Info(fmt.Sprintf("Auditing file: %s", file))
		
		// Read file content
		content, err := ioutil.ReadFile(file)
		if err != nil {
			vs.logger.Error(fmt.Sprintf("Failed to read %s", file), err)
			continue
		}
		
		// Basic security pattern scan
		issues := vs.basicSecurityScan(string(content), file)
		result.Issues = append(result.Issues, issues...)
		
		// AI-powered security analysis if Ollama is available
		if ollamaURL := os.Getenv("OLLAMA_URL"); ollamaURL != "" {
			aiIssues := vs.aiSecurityAnalysis(string(content), file, ollamaURL)
			result.Issues = append(result.Issues, aiIssues...)
		}
	}
	
	// Store audit results
	if err := vs.storeAuditResults(result); err != nil {
		vs.logger.Error("Failed to store audit results", err)
	}
	
	vs.logger.Info(fmt.Sprintf("Security audit completed. Found %d issues", len(result.Issues)))
	return result, nil
}

// basicSecurityScan performs pattern-based security scanning
func (vs *VulnerabilityScanner) basicSecurityScan(content, filePath string) []SecurityIssue {
	var issues []SecurityIssue
	lines := strings.Split(content, "\n")
	
	// Security patterns to check
	patterns := []struct {
		pattern     string
		severity    string
		category    string
		description string
	}{
		{`sql\.Query.*fmt\.Sprintf`, "HIGH", "SQL Injection", "Potential SQL injection via string formatting"},
		{`fmt\.Sprintf.*%s.*sql\.`, "HIGH", "SQL Injection", "SQL query built with string interpolation"},
		{`os\.Getenv.*password`, "MEDIUM", "Credential Exposure", "Password from environment variable without encryption"},
		{`\*\*\*`, "LOW", "Debug Code", "Debug markers found in production code"},
		{`TODO.*security`, "MEDIUM", "Incomplete Security", "Security-related TODO found"},
		{`FIXME.*auth`, "MEDIUM", "Authentication Issue", "Authentication issue marked for fixing"},
		{`exec\.Command`, "HIGH", "Command Injection", "Direct command execution detected"},
		{`ioutil\.ReadAll.*http\.`, "MEDIUM", "Memory Issue", "Unbounded memory allocation from HTTP response"},
		{`rand\.\w+\(\)`, "MEDIUM", "Weak Randomness", "Weak random number generation"},
		{`md5\.`, "HIGH", "Weak Cryptography", "MD5 hash usage detected"},
		{`sha1\.`, "MEDIUM", "Weak Cryptography", "SHA1 hash usage detected"},
	}
	
	for _, p := range patterns {
		re := regexp.MustCompile(p.pattern)
		for lineNum, line := range lines {
			if re.MatchString(line) {
				issues = append(issues, SecurityIssue{
					ID:          uuid.New(),
					Severity:    p.severity,
					Category:    p.category,
					Description: p.description,
					FilePath:    filePath,
					LineNumber:  lineNum + 1,
					CodeSnippet: strings.TrimSpace(line),
					Status:      "open",
				})
			}
		}
	}
	
	return issues
}

// aiSecurityAnalysis uses Ollama to perform AI-powered security analysis
func (vs *VulnerabilityScanner) aiSecurityAnalysis(content, filePath, ollamaURL string) []SecurityIssue {
	var issues []SecurityIssue
	
	prompt := fmt.Sprintf(`Analyze this Go API code for security vulnerabilities. Look for:
1. SQL injection risks
2. Missing input validation
3. Weak authentication/authorization
4. CORS misconfigurations
5. Information disclosure
6. Missing rate limiting
7. Insecure data storage
8. Cryptographic weaknesses

Code:
%s

Provide specific vulnerabilities found with line numbers if possible. Format as JSON array with fields: severity, category, description, line_number, recommendation.`, content)
	
	payload := map[string]interface{}{
		"model":       "llama3.2",
		"prompt":      prompt,
		"temperature": 0.1,
		"format":      "json",
	}
	
	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(ollamaURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		vs.logger.Error("Failed to call Ollama for security analysis", err)
		return issues
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		vs.logger.Error("Failed to decode Ollama response", err)
		return issues
	}
	
	if response, ok := result["response"].(string); ok {
		var aiFindings []map[string]interface{}
		if err := json.Unmarshal([]byte(response), &aiFindings); err == nil {
			for _, finding := range aiFindings {
				issue := SecurityIssue{
					ID:       uuid.New(),
					FilePath: filePath,
					Status:   "open",
					Source:   "AI Analysis",
				}
				
				if s, ok := finding["severity"].(string); ok {
					issue.Severity = s
				}
				if c, ok := finding["category"].(string); ok {
					issue.Category = c
				}
				if d, ok := finding["description"].(string); ok {
					issue.Description = d
				}
				if r, ok := finding["recommendation"].(string); ok {
					issue.Recommendation = r
				}
				if ln, ok := finding["line_number"].(float64); ok {
					issue.LineNumber = int(ln)
				}
				
				issues = append(issues, issue)
			}
		}
	}
	
	return issues
}

// storeAuditResults stores security audit results in the database
func (vs *VulnerabilityScanner) storeAuditResults(result *SecurityAuditResult) error {
	// First, get or create scenario ID
	var scenarioID uuid.UUID
	err := vs.db.QueryRow(`
		INSERT INTO scenarios (id, name, path, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'active', NOW(), NOW())
		ON CONFLICT (name) DO UPDATE SET 
			last_scanned = NOW(),
			updated_at = NOW()
		RETURNING id
	`, uuid.New(), result.ScenarioName, result.ScenarioPath).Scan(&scenarioID)
	
	if err != nil {
		return fmt.Errorf("failed to upsert scenario: %w", err)
	}
	
	// Store each security issue
	for _, issue := range result.Issues {
		_, err := vs.db.Exec(`
			INSERT INTO vulnerability_scans 
			(id, scenario_id, scan_type, severity, category, title, description, 
			 file_path, line_number, code_snippet, recommendation, status, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		`, issue.ID, scenarioID, "security_audit", issue.Severity, issue.Category,
			issue.Category+" Issue", issue.Description, issue.FilePath, 
			issue.LineNumber, issue.CodeSnippet, issue.Recommendation, issue.Status)
		
		if err != nil {
			vs.logger.Error("Failed to store security issue", err)
		}
	}
	
	return nil
}

// extractCodeSnippet extracts a code snippet from source between start and end lines
func (vs *VulnerabilityScanner) extractCodeSnippet(src []byte, startLine, endLine int) string {
	lines := strings.Split(string(src), "\n")
	if startLine < 1 || endLine > len(lines) {
		return ""
	}
	// Extract up to 10 lines around the issue
	start := startLine - 1
	end := endLine
	if end-start > 10 {
		end = start + 10
	}
	return strings.Join(lines[start:end], "\n")
}

// Helper functions for scanning
func (vs *VulnerabilityScanner) findGoFiles(dirPath string) ([]string, error) {
	var goFiles []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".go") && !strings.Contains(path, "vendor/") {
			goFiles = append(goFiles, path)
		}
		return nil
	})
	return goFiles, err
}

func (vs *VulnerabilityScanner) scanForCommonIssues(filePath string) []Vulnerability {
	var vulns []Vulnerability
	
	// Read file content
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		vs.logger.Error(fmt.Sprintf("Failed to read file for common issues scan: %s", filePath), err)
		return vulns
	}
	
	lines := strings.Split(string(content), "\n")
	
	// Common vulnerability patterns
	commonPatterns := []struct {
		pattern     string
		severity    string
		title       string
		category    string
		description string
		impact      string
		recommendation string
	}{
		{
			`password\s*[:=]\s*["'][^"']*["']`,
			"CRITICAL",
			"Hardcoded Password",
			"Credential Exposure",
			"Password hardcoded in source code",
			"Credentials exposed in version control and deployments",
			"Use environment variables or secure credential management",
		},
		{
			`api[_-]?key\s*[:=]\s*["'][^"']*["']`,
			"HIGH",
			"Hardcoded API Key",
			"Credential Exposure", 
			"API key hardcoded in source code",
			"API credentials exposed, potential unauthorized access",
			"Store API keys in environment variables or secrets manager",
		},
		{
			`math/rand\.\w+\(\)`,
			"MEDIUM",
			"Weak Random Generation",
			"Cryptographic Weakness",
			"Using math/rand for randomness",
			"Predictable random values, cryptographic vulnerabilities",
			"Use crypto/rand for security-sensitive random generation",
		},
		{
			`\.\./.*\.\./`,
			"HIGH",
			"Path Traversal Pattern",
			"Path Traversal",
			"Potential path traversal vulnerability",
			"Directory traversal attacks, unauthorized file access",
			"Validate and sanitize file paths, use filepath.Clean()",
		},
		{
			`exec\.Command\s*\(\s*[^,)]*\s*,.*\$`,
			"CRITICAL",
			"Command Injection Risk",
			"Command Injection",
			"User input passed to exec.Command",
			"Remote command execution, system compromise",
			"Validate input, use allowlists, avoid direct command execution",
		},
		{
			`fmt\.Print.*\(.*password.*\)`,
			"HIGH",
			"Password in Logs",
			"Information Disclosure",
			"Password potentially logged to output",
			"Credentials leaked in logs and console output",
			"Never log sensitive information, use structured logging",
		},
		{
			`http\.Handle.*\(\s*["']/\*["']`,
			"MEDIUM",
			"Overly Broad Route",
			"Access Control",
			"HTTP handler matches all paths under root",
			"Unintended route exposure, potential access to debug endpoints",
			"Use specific route patterns, implement proper access controls",
		},
		{
			`Listen.*:\s*80\d\d`,
			"LOW",
			"Hardcoded Port",
			"Configuration",
			"HTTP port hardcoded in source",
			"Deployment flexibility reduced, port conflicts",
			"Use environment variables for port configuration",
		},
		{
			`sql\.Query.*\+.*\$\{`,
			"CRITICAL",
			"SQL Injection Template",
			"SQL Injection",
			"SQL query built with template/string concatenation",
			"Database compromise, data exfiltration",
			"Use parameterized queries with $1, $2 placeholders",
		},
		{
			`\.Get\(.*\)\.Body`,
			"MEDIUM",
			"HTTP Response Not Checked",
			"Error Handling",
			"HTTP response body accessed without error checking",
			"Application crashes, undefined behavior",
			"Always check HTTP response errors before accessing body",
		},
	}
	
	for _, pattern := range commonPatterns {
		re := regexp.MustCompile(pattern.pattern)
		for lineNum, line := range lines {
			if re.MatchString(line) {
				vuln := Vulnerability{
					ID:             uuid.New(),
					Type:           pattern.title,
					Severity:       pattern.severity,
					Title:          pattern.title,
					Description:    pattern.description,
					FilePath:       filePath,
					LineNumber:     lineNum + 1,
					CodeSnippet:    strings.TrimSpace(line),
					Impact:         pattern.impact,
					Recommendation: pattern.recommendation,
					Status:         "open",
					Category:       pattern.category,
				}
				vulns = append(vulns, vuln)
			}
		}
	}
	
	return vulns
}

func (vs *VulnerabilityScanner) storeResults(result *ScanResult) error {
	// First, get or create scenario ID
	var scenarioID uuid.UUID
	err := vs.db.QueryRow(`
		INSERT INTO scenarios (id, name, path, status, created_at, updated_at, last_scanned)
		VALUES ($1, $2, $3, 'active', NOW(), NOW(), NOW())
		ON CONFLICT (name) DO UPDATE SET 
			last_scanned = NOW(),
			updated_at = NOW()
		RETURNING id
	`, uuid.New(), result.ScenarioName, result.ScenarioPath).Scan(&scenarioID)
	
	if err != nil {
		return fmt.Errorf("failed to upsert scenario: %w", err)
	}
	
	// Store each vulnerability
	for _, vuln := range result.Vulnerabilities {
		_, err := vs.db.Exec(`
			INSERT INTO vulnerability_scans 
			(id, scenario_id, scan_type, severity, category, title, description, 
			 file_path, line_number, code_snippet, recommendation, status, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
			ON CONFLICT (scenario_id, category, file_path, line_number) 
			DO UPDATE SET 
				severity = EXCLUDED.severity,
				description = EXCLUDED.description,
				recommendation = EXCLUDED.recommendation,
				updated_at = NOW()
		`, vuln.ID, scenarioID, "vulnerability_scan", vuln.Severity, vuln.Category,
			vuln.Title, vuln.Description, vuln.FilePath, 
			vuln.LineNumber, vuln.CodeSnippet, vuln.Recommendation, vuln.Status)
		
		if err != nil {
			vs.logger.Error("Failed to store vulnerability", err)
		}
	}
	
	// Record scan history
	_, err = vs.db.Exec(`
		INSERT INTO scan_history 
		(id, scenario_id, scan_type, status, results_summary, duration_ms, 
		 triggered_by, started_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
	`, uuid.New(), scenarioID, "vulnerability_scan", "completed",
		fmt.Sprintf(`{"vulnerabilities_found": %d, "critical": %d, "high": %d}`,
			len(result.Vulnerabilities),
			vs.countBySeverity(result.Vulnerabilities, "CRITICAL"),
			vs.countBySeverity(result.Vulnerabilities, "HIGH")),
		0, "api_scan", result.ScanTime)
	
	if err != nil {
		vs.logger.Error("Failed to record scan history", err)
	}
	
	return nil
}

// Helper function to count vulnerabilities by severity
func (vs *VulnerabilityScanner) countBySeverity(vulns []Vulnerability, severity string) int {
	count := 0
	for _, vuln := range vulns {
		if vuln.Severity == severity {
			count++
		}
	}
	return count
}

func (vs *VulnerabilityScanner) generateHTTPLeakDescription(vuln HTTPBodyLeakVulnerability) string {
	return fmt.Sprintf("HTTP response body not closed in function '%s'. This leaks TCP connections and causes resource exhaustion. %s request at line %d lacks proper defer resp.Body.Close()",
		vuln.FunctionName, vuln.HTTPMethod, vuln.LineNumber)
}

func (vs *VulnerabilityScanner) generateHTTPLeakFix(vuln HTTPBodyLeakVulnerability) string {
	return fmt.Sprintf("Add 'defer resp.Body.Close()' immediately after checking the error from the %s call at line %d in function '%s'",
		vuln.HTTPMethod, vuln.LineNumber, vuln.FunctionName)
}

func (vs *VulnerabilityScanner) isHTTPCall(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			if ident.Name == "http" && (sel.Sel.Name == "Get" || sel.Sel.Name == "Post" || 
				sel.Sel.Name == "Put" || sel.Sel.Name == "Delete" || sel.Sel.Name == "Head") {
				return true
			}
		}
	}
	return false
}

func (vs *VulnerabilityScanner) extractHTTPMethod(call *ast.CallExpr) string {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		return sel.Sel.Name
	}
	return "Unknown"
}

func (vs *VulnerabilityScanner) isDeferBodyClose(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if sel.Sel.Name == "Close" {
			if subSel, ok := sel.X.(*ast.SelectorExpr); ok {
				if subSel.Sel.Name == "Body" {
					return true
				}
			}
		}
	}
	return false
}

// scanForHTTPBodyLeaks uses AST parsing to find unclosed HTTP response bodies
func (vs *VulnerabilityScanner) scanForHTTPBodyLeaks(filePath string) ([]HTTPBodyLeakVulnerability, error) {
	var vulnerabilities []HTTPBodyLeakVulnerability
	
	// Parse the Go file
	fset := token.NewFileSet()
	src, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	node, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	
	// Walk the AST
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			// Check each function for HTTP calls
			funcName := x.Name.Name
			hasHTTPCall := false
			hasDeferClose := false
			var httpMethod string
			var callLine int
			
			// Look for HTTP calls in the function
			ast.Inspect(x.Body, func(inner ast.Node) bool {
				switch stmt := inner.(type) {
				case *ast.AssignStmt:
					// Look for patterns like: resp, err := http.Get(...)
					for _, rhs := range stmt.Rhs {
						if call, ok := rhs.(*ast.CallExpr); ok {
							if vs.isHTTPCall(call) {
								hasHTTPCall = true
								httpMethod = vs.extractHTTPMethod(call)
								callLine = fset.Position(call.Pos()).Line
							}
						}
					}
				case *ast.DeferStmt:
					// Look for defer resp.Body.Close()
					if vs.isDeferBodyClose(stmt.Call) {
						hasDeferClose = true
					}
				}
				return true
			})
			
			// If we found an HTTP call without defer close, it's a vulnerability
			if hasHTTPCall && !hasDeferClose {
				// Extract the code snippet
				snippet := vs.extractCodeSnippet(src, fset.Position(x.Pos()).Line, fset.Position(x.End()).Line)
				
				vulnerabilities = append(vulnerabilities, HTTPBodyLeakVulnerability{
					FilePath:     filePath,
					LineNumber:   callLine,
					FunctionName: funcName,
					CodeSnippet:  snippet,
					HTTPMethod:   httpMethod,
					HasDefer:     false,
					Severity:     "CRITICAL",
				})
			}
		}
		return true
	})
	
	return vulnerabilities, nil
}

// GenerateFixForVulnerability generates automated fix code for a vulnerability
func (vs *VulnerabilityScanner) GenerateFixForVulnerability(vulnType, category, filePath string, lineNumber int) (*AutomatedFix, error) {
	fix := &AutomatedFix{
		VulnerabilityType: vulnType,
		Category:          category,
		FilePath:          filePath,
		LineNumber:        lineNumber,
		Status:            "pending",
	}

	switch category {
	case "Resource Leak":
		fix.FixCode = vs.generateResourceLeakFix(vulnType, filePath, lineNumber)
		fix.Description = "Add proper resource cleanup with defer statement"
		fix.Confidence = "high"

	case "SQL Injection":
		fix.FixCode = vs.generateSQLInjectionFix(filePath, lineNumber)
		fix.Description = "Replace string formatting with parameterized queries"
		fix.Confidence = "high"

	case "Missing Input Validation":
		fix.FixCode = vs.generateInputValidationFix(filePath, lineNumber)
		fix.Description = "Add input validation and sanitization"
		fix.Confidence = "medium"

	case "Authentication Issue":
		fix.FixCode = vs.generateAuthFix(filePath, lineNumber)
		fix.Description = "Add authentication middleware"
		fix.Confidence = "low"

	case "Error Handling":
		fix.FixCode = vs.generateErrorHandlingFix(filePath, lineNumber)
		fix.Description = "Add proper error handling and logging"
		fix.Confidence = "high"

	case "Rate Limiting":
		fix.FixCode = vs.generateRateLimitingFix(filePath, lineNumber)
		fix.Description = "Add rate limiting middleware"
		fix.Confidence = "medium"

	default:
		fix.FixCode = ""
		fix.Description = "Manual fix required - automated fix not available"
		fix.Confidence = "none"
		fix.Status = "manual_required"
	}

	return fix, nil
}

// AutomatedFix represents an automated fix for a vulnerability
type AutomatedFix struct {
	VulnerabilityType string `json:"vulnerability_type"`
	Category          string `json:"category"`
	FilePath          string `json:"file_path"`
	LineNumber        int    `json:"line_number"`
	FixCode           string `json:"fix_code"`
	Description       string `json:"description"`
	Confidence        string `json:"confidence"` // high, medium, low, none
	Status            string `json:"status"`     // pending, applied, failed, manual_required
}

func (vs *VulnerabilityScanner) generateResourceLeakFix(vulnType, filePath string, lineNumber int) string {
	if vulnType == "HTTP_RESPONSE_BODY_LEAK" {
		return `// Add immediately after the HTTP call:
defer func() {
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
}()`
	}
	return `// Add defer statement to close resource
defer resource.Close()`
}

func (vs *VulnerabilityScanner) generateSQLInjectionFix(filePath string, lineNumber int) string {
	return `// Replace string formatting with parameterized query:
// Instead of: fmt.Sprintf("SELECT * FROM users WHERE id = %s", userID)
// Use: "SELECT * FROM users WHERE id = $1" with parameters
query := "SELECT * FROM users WHERE id = $1"
rows, err := db.Query(query, userID)`
}

func (vs *VulnerabilityScanner) generateInputValidationFix(filePath string, lineNumber int) string {
	return `// Add input validation:
if err := validateInput(input); err != nil {
	http.Error(w, "Invalid input: " + err.Error(), http.StatusBadRequest)
	return
}

func validateInput(input string) error {
	// Check for SQL injection patterns
	if strings.ContainsAny(input, "';--") {
		return fmt.Errorf("invalid characters detected")
	}
	// Check length
	if len(input) > 1000 {
		return fmt.Errorf("input too long")
	}
	// Additional validation as needed
	return nil
}`
}

func (vs *VulnerabilityScanner) generateAuthFix(filePath string, lineNumber int) string {
	return `// Add authentication middleware:
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		// Validate token
		if !validateToken(token) {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}`
}

func (vs *VulnerabilityScanner) generateErrorHandlingFix(filePath string, lineNumber int) string {
	return `// Add proper error handling:
if err != nil {
	// Log the error internally
	log.Printf("Error occurred: %v", err)
	
	// Return generic error to client
	http.Error(w, "An error occurred processing your request", http.StatusInternalServerError)
	return
}`
}

func (vs *VulnerabilityScanner) generateRateLimitingFix(filePath string, lineNumber int) string {
	return `// Add rate limiting middleware:
import "golang.org/x/time/rate"

var limiter = rate.NewLimiter(10, 100) // 10 requests per second, burst of 100

func rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}`
}

// ApplyAutomatedFix attempts to apply an automated fix to a file
func (vs *VulnerabilityScanner) ApplyAutomatedFix(fix *AutomatedFix) error {
	if fix.Confidence == "none" || fix.Status == "manual_required" {
		return fmt.Errorf("automated fix not available for this vulnerability")
	}

	// Read the file
	content, err := ioutil.ReadFile(fix.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	
	// Simple fix application - insert fix code after the vulnerable line
	// In production, this would use more sophisticated AST manipulation
	if fix.LineNumber > 0 && fix.LineNumber <= len(lines) {
		// Create backup
		backupPath := fix.FilePath + ".backup"
		if err := ioutil.WriteFile(backupPath, content, 0644); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}

		// Insert fix
		newLines := make([]string, 0, len(lines)+10)
		newLines = append(newLines, lines[:fix.LineNumber]...)
		newLines = append(newLines, "", "// Automated fix applied by API Manager")
		newLines = append(newLines, strings.Split(fix.FixCode, "\n")...)
		newLines = append(newLines, "")
		newLines = append(newLines, lines[fix.LineNumber:]...)

		// Write fixed content
		fixedContent := strings.Join(newLines, "\n")
		if err := ioutil.WriteFile(fix.FilePath, []byte(fixedContent), 0644); err != nil {
			// Restore backup on failure
			ioutil.WriteFile(fix.FilePath, content, 0644)
			return fmt.Errorf("failed to apply fix: %w", err)
		}

		fix.Status = "applied"
		vs.logger.Info(fmt.Sprintf("Successfully applied automated fix to %s at line %d", fix.FilePath, fix.LineNumber))
	}

	return nil
}

// ===============================================================================
// ENHANCED AUTOMATED FIX SYSTEM WITH SAFETY CONTROLS
// ===============================================================================

// AutomatedFixConfig represents the configuration for automated fixes
type AutomatedFixConfig struct {
	Enabled           bool     `json:"enabled"`              // Must be manually enabled
	AllowedCategories []string `json:"allowed_categories"`   // Categories allowed for auto-fix
	MaxConfidence     string   `json:"max_confidence"`       // Maximum confidence level to auto-apply
	RequireApproval   bool     `json:"require_approval"`     // Require manual approval before applying
	BackupEnabled     bool     `json:"backup_enabled"`       // Create backups before applying fixes
	RollbackWindow    int      `json:"rollback_window_hrs"`  // Hours to keep rollback capability
}

// AutomatedFixManager manages the safe application of automated fixes
type AutomatedFixManager struct {
	db     *sql.DB
	logger *Logger
	config AutomatedFixConfig
}

// NewAutomatedFixManager creates a new fix manager with safety defaults
func NewAutomatedFixManager(db *sql.DB) *AutomatedFixManager {
	return &AutomatedFixManager{
		db:     db,
		logger: NewLogger(),
		config: AutomatedFixConfig{
			Enabled:           false,                                    // CRITICAL: Disabled by default
			AllowedCategories: []string{"Resource Leak", "Error Handling"}, // Only safe categories
			MaxConfidence:     "high",                                   // Only high-confidence fixes
			RequireApproval:   true,                                     // Require manual approval
			BackupEnabled:     true,                                     // Always backup
			RollbackWindow:    24,                                       // 24-hour rollback window
		},
	}
}

// IsAutomatedFixEnabled checks if automated fixes are enabled (safety check)
func (afm *AutomatedFixManager) IsAutomatedFixEnabled() bool {
	return afm.config.Enabled
}

// EnableAutomatedFixes manually enables automated fixes with confirmation
func (afm *AutomatedFixManager) EnableAutomatedFixes(categories []string, maxConfidence string) error {
	afm.logger.Info("⚠️  ENABLING AUTOMATED FIXES - This will modify source code automatically")
	
	// Update configuration
	afm.config.Enabled = true
	afm.config.AllowedCategories = categories
	afm.config.MaxConfidence = maxConfidence
	
	// Log the activation for audit trail
	_, err := afm.db.Exec(`
		INSERT INTO scan_history 
		(id, scenario_id, scan_type, status, results_summary, triggered_by, started_at, completed_at)
		VALUES ($1, (SELECT id FROM scenarios LIMIT 1), 'automated_fix_config', 'enabled', $2, 'manual_activation', NOW(), NOW())
	`, uuid.New(), fmt.Sprintf(`{"enabled": true, "categories": %v, "max_confidence": "%s"}`, categories, maxConfidence))
	
	if err != nil {
		afm.logger.Error("Failed to log automated fix activation", err)
	}
	
	afm.logger.Info(fmt.Sprintf("✅ Automated fixes ENABLED for categories: %v (max confidence: %s)", categories, maxConfidence))
	return nil
}

// DisableAutomatedFixes disables automated fixes
func (afm *AutomatedFixManager) DisableAutomatedFixes() error {
	afm.config.Enabled = false
	
	// Log the deactivation
	_, err := afm.db.Exec(`
		INSERT INTO scan_history 
		(id, scenario_id, scan_type, status, results_summary, triggered_by, started_at, completed_at)
		VALUES ($1, (SELECT id FROM scenarios LIMIT 1), 'automated_fix_config', 'disabled', '{"enabled": false}', 'manual_deactivation', NOW(), NOW())
	`, uuid.New())
	
	if err != nil {
		afm.logger.Error("Failed to log automated fix deactivation", err)
	}
	
	afm.logger.Info("🛑 Automated fixes DISABLED")
	return nil
}

// SafelyApplyFix applies a fix with all safety checks and rollback capability
func (afm *AutomatedFixManager) SafelyApplyFix(fix *AutomatedFix, scenarioID uuid.UUID) error {
	// CRITICAL SAFETY CHECK: Ensure automated fixes are enabled
	if !afm.config.Enabled {
		return fmt.Errorf("🛑 SAFETY: Automated fixes are disabled. Enable them first with explicit confirmation")
	}
	
	// Check if category is allowed
	if !afm.isCategoryAllowed(fix.Category) {
		return fmt.Errorf("🛑 SAFETY: Category '%s' not in allowed list: %v", fix.Category, afm.config.AllowedCategories)
	}
	
	// Check confidence level
	if !afm.isConfidenceAllowed(fix.Confidence) {
		return fmt.Errorf("🛑 SAFETY: Confidence level '%s' exceeds maximum '%s'", fix.Confidence, afm.config.MaxConfidence)
	}
	
	// Check if approval is required
	if afm.config.RequireApproval {
		approved, err := afm.checkApprovalStatus(fix, scenarioID)
		if err != nil {
			return fmt.Errorf("failed to check approval status: %w", err)
		}
		if !approved {
			return fmt.Errorf("🛑 SAFETY: Manual approval required before applying fix")
		}
	}
	
	afm.logger.Info(fmt.Sprintf("🔧 Applying automated fix to %s:%d (Category: %s, Confidence: %s)", 
		fix.FilePath, fix.LineNumber, fix.Category, fix.Confidence))
	
	// Create backup if enabled
	var backupPath string
	if afm.config.BackupEnabled {
		var err error
		backupPath, err = afm.createBackup(fix.FilePath)
		if err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		afm.logger.Info(fmt.Sprintf("📁 Backup created: %s", backupPath))
	}
	
	// Apply the fix
	err := afm.applyFixWithValidation(fix)
	if err != nil {
		// Restore backup on failure
		if backupPath != "" {
			if restoreErr := afm.restoreBackup(backupPath, fix.FilePath); restoreErr != nil {
				afm.logger.Error("Failed to restore backup after fix failure", restoreErr)
			}
		}
		return fmt.Errorf("failed to apply fix: %w", err)
	}
	
	// Record the applied fix in database
	err = afm.recordAppliedFix(fix, scenarioID, backupPath)
	if err != nil {
		afm.logger.Error("Failed to record applied fix", err)
	}
	
	afm.logger.Info(fmt.Sprintf("✅ Successfully applied automated fix to %s:%d", fix.FilePath, fix.LineNumber))
	return nil
}

// Helper methods for AutomatedFixManager
func (afm *AutomatedFixManager) isCategoryAllowed(category string) bool {
	for _, allowed := range afm.config.AllowedCategories {
		if allowed == category {
			return true
		}
	}
	return false
}

func (afm *AutomatedFixManager) isConfidenceAllowed(confidence string) bool {
	confidenceLevels := map[string]int{
		"low":    1,
		"medium": 2,
		"high":   3,
	}
	
	maxLevel, exists := confidenceLevels[afm.config.MaxConfidence]
	if !exists {
		return false
	}
	
	fixLevel, exists := confidenceLevels[confidence]
	if !exists {
		return false
	}
	
	return fixLevel >= maxLevel
}

func (afm *AutomatedFixManager) checkApprovalStatus(fix *AutomatedFix, scenarioID uuid.UUID) (bool, error) {
	var approved bool
	err := afm.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM improvements 
			WHERE scenario_id = $1 AND category = $2 AND status = 'approved'
		)
	`, scenarioID, fix.Category).Scan(&approved)
	
	if err != nil {
		return false, err
	}
	
	return approved, nil
}

func (afm *AutomatedFixManager) createBackup(filePath string) (string, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	
	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.backup-%s", filePath, timestamp)
	
	err = ioutil.WriteFile(backupPath, content, 0644)
	if err != nil {
		return "", err
	}
	
	return backupPath, nil
}

func (afm *AutomatedFixManager) restoreBackup(backupPath, originalPath string) error {
	content, err := ioutil.ReadFile(backupPath)
	if err != nil {
		return err
	}
	
	return ioutil.WriteFile(originalPath, content, 0644)
}

func (afm *AutomatedFixManager) applyFixWithValidation(fix *AutomatedFix) error {
	// Read original content
	content, err := ioutil.ReadFile(fix.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	
	lines := strings.Split(string(content), "\n")
	
	// Validate line number
	if fix.LineNumber < 1 || fix.LineNumber > len(lines) {
		return fmt.Errorf("invalid line number: %d (file has %d lines)", fix.LineNumber, len(lines))
	}
	
	// Apply fix based on category with enhanced safety
	switch fix.Category {
	case "Resource Leak":
		err = afm.applyResourceLeakFix(lines, fix)
	case "Error Handling":
		err = afm.applyErrorHandlingFix(lines, fix)
	case "Input Validation":
		err = afm.applyInputValidationFix(lines, fix)
	default:
		return fmt.Errorf("unsupported fix category: %s", fix.Category)
	}
	
	if err != nil {
		return err
	}
	
	// Write the modified content
	modifiedContent := strings.Join(lines, "\n")
	return ioutil.WriteFile(fix.FilePath, []byte(modifiedContent), 0644)
}

func (afm *AutomatedFixManager) applyResourceLeakFix(lines []string, fix *AutomatedFix) error {
	if fix.LineNumber >= len(lines) {
		return fmt.Errorf("line number out of range")
	}
	
	// For HTTP response body leaks, add defer close after the HTTP call
	if strings.Contains(fix.VulnerabilityType, "HTTP_RESPONSE_BODY_LEAK") {
		// Find the HTTP call line and add defer statement after it
		targetLine := fix.LineNumber - 1
		
		// Insert defer statement after the HTTP call
		deferCode := "\tdefer func() { if resp != nil && resp.Body != nil { resp.Body.Close() } }()"
		
		// Insert at the appropriate position
		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:targetLine+1]...)
		newLines = append(newLines, deferCode)
		newLines = append(newLines, lines[targetLine+1:]...)
		
		// Update the lines slice
		copy(lines, newLines[:len(lines)])
		if len(newLines) > len(lines) {
			lines = append(lines, newLines[len(lines):]...)
		}
	}
	
	return nil
}

func (afm *AutomatedFixManager) applyErrorHandlingFix(lines []string, fix *AutomatedFix) error {
	// Add basic error handling pattern
	targetLine := fix.LineNumber - 1
	
	errorHandlingCode := []string{
		"\tif err != nil {",
		"\t\tlog.Printf(\"Error occurred: %v\", err)",
		"\t\thttp.Error(w, \"Internal server error\", http.StatusInternalServerError)",
		"\t\treturn",
		"\t}",
	}
	
	// Insert error handling after the line that should be checked
	newLines := make([]string, 0, len(lines)+len(errorHandlingCode))
	newLines = append(newLines, lines[:targetLine+1]...)
	newLines = append(newLines, errorHandlingCode...)
	newLines = append(newLines, lines[targetLine+1:]...)
	
	// Update the lines slice
	copy(lines, newLines[:len(lines)])
	if len(newLines) > len(lines) {
		lines = append(lines, newLines[len(lines):]...)
	}
	
	return nil
}

func (afm *AutomatedFixManager) applyInputValidationFix(lines []string, fix *AutomatedFix) error {
	// This would be a more complex fix requiring careful analysis
	// For now, just return an error indicating manual fix required
	return fmt.Errorf("input validation fixes require manual implementation")
}

func (afm *AutomatedFixManager) recordAppliedFix(fix *AutomatedFix, scenarioID uuid.UUID, backupPath string) error {
	_, err := afm.db.Exec(`
		INSERT INTO applied_fixes 
		(id, scenario_id, fix_type, file_path, line_range, rollback_info, success, applied_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`, uuid.New(), scenarioID, fix.Category, fix.FilePath, 
		fmt.Sprintf("%d", fix.LineNumber),
		fmt.Sprintf(`{"backup_path": "%s", "confidence": "%s"}`, backupPath, fix.Confidence),
		true)
	
	return err
}

// RollbackFix rolls back a previously applied fix
func (afm *AutomatedFixManager) RollbackFix(appliedFixID uuid.UUID) error {
	var backupInfo string
	var filePath string
	
	err := afm.db.QueryRow(`
		SELECT file_path, rollback_info 
		FROM applied_fixes 
		WHERE id = $1 AND success = true
	`, appliedFixID).Scan(&filePath, &backupInfo)
	
	if err != nil {
		return fmt.Errorf("failed to find applied fix: %w", err)
	}
	
	// Parse rollback info to get backup path
	var rollbackData map[string]string
	if err := json.Unmarshal([]byte(backupInfo), &rollbackData); err != nil {
		return fmt.Errorf("failed to parse rollback info: %w", err)
	}
	
	backupPath := rollbackData["backup_path"]
	if backupPath == "" {
		return fmt.Errorf("no backup path found in rollback info")
	}
	
	// Restore the backup
	if err := afm.restoreBackup(backupPath, filePath); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}
	
	// Mark the fix as rolled back
	_, err = afm.db.Exec(`
		UPDATE applied_fixes 
		SET success = false, error_message = 'Rolled back by user'
		WHERE id = $1
	`, appliedFixID)
	
	if err != nil {
		afm.logger.Error("Failed to update rollback status in database", err)
	}
	
	afm.logger.Info(fmt.Sprintf("✅ Successfully rolled back fix for %s", filePath))
	return nil
}
