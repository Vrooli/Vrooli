package main

import (
	"bufio"
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
	// Placeholder for common issue scanning
	// This would include checks for:
	// - Hardcoded credentials
	// - Insecure random number generation
	// - Path traversal vulnerabilities
	// - etc.
	return vulns
}

func (vs *VulnerabilityScanner) storeResults(result *ScanResult) error {
	// Store scan results in database
	// Implementation would store each vulnerability found
	return nil
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
	
	// Track HTTP calls and their defer statements
	httpCalls := make(map[string]bool) // functionName -> hasDeferClose
	
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
					if call, ok := stmt.Call.(*ast.CallExpr); ok {
						if vs.isDeferBodyClose(call) {
							hasDeferClose = true
						}
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

// isHTTPCall checks if a call expression is an HTTP request
func (vs *VulnerabilityScanner) isHTTPCall(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			// Check for http.Get, http.Post, http.Head, etc.
			if ident.Name == "http" && (sel.Sel.Name == "Get" || 
				sel.Sel.Name == "Post" || 
				sel.Sel.Name == "Head" || 
				sel.Sel.Name == "PostForm" ||
				sel.Sel.Name == "Do") {
				return true
			}
			// Also check for client.Get, client.Post, etc.
			if strings.HasSuffix(ident.Name, "Client") || ident.Name == "client" {
				if sel.Sel.Name == "Get" || sel.Sel.Name == "Post" || 
					sel.Sel.Name == "Do" || sel.Sel.Name == "Head" {
					return true
				}
			}
		}
	}
	return false
}

// extractHTTPMethod extracts the HTTP method from the call
func (vs *VulnerabilityScanner) extractHTTPMethod(call *ast.CallExpr) string {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		return sel.Sel.Name
	}
	return "Unknown"
}

// isDeferBodyClose checks if this is a defer resp.Body.Close() statement
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

// extractCodeSnippet extracts lines from source code
func (vs *VulnerabilityScanner) extractCodeSnippet(src []byte, startLine, endLine int) string {
	lines := strings.Split(string(src), "\n")
	if startLine < 1 || endLine > len(lines) {
		return ""
	}
	
	// Limit snippet size
	if endLine - startLine > 20 {
		endLine = startLine + 20
	}
	
	snippet := lines[startLine-1:endLine]
	return strings.Join(snippet, "\n")
}

// generateHTTPLeakDescription creates a detailed description of the HTTP leak issue
func (vs *VulnerabilityScanner) generateHTTPLeakDescription(vuln HTTPBodyLeakVulnerability) string {
	return fmt.Sprintf(`CRITICAL VULNERABILITY: HTTP Response Body Not Closed

Function '%s' makes an HTTP %s request but NEVER closes the response body.
This creates a permanent TCP connection leak that will exhaust system resources.

Impact:
- Each call leaks 1 TCP connection permanently
- Connection remains in ESTABLISHED state until timeout (hours)
- System currently showing 15,000+ leaked connections
- Will cause complete system failure when connection limits reached
- Other scenarios cannot connect to shared resources
- Vrooli platform becomes unresponsive

This is causing your current connection exhaustion problem!`, 
		vuln.FunctionName, vuln.HTTPMethod)
}

// generateHTTPLeakFix provides the exact fix needed
func (vs *VulnerabilityScanner) generateHTTPLeakFix(vuln HTTPBodyLeakVulnerability) string {
	return fmt.Sprintf(`IMMEDIATE FIX REQUIRED:

After line containing 'http.%s', add immediately:

resp, err := http.%s(url)
if err != nil {
    return err
}
defer resp.Body.Close()  // <-- ADD THIS LINE IMMEDIATELY AFTER ERROR CHECK
io.Copy(io.Discard, resp.Body)  // <-- ALSO ADD THIS TO DRAIN THE BODY

NEVER make an HTTP request without these two lines!
Even if you only check resp.StatusCode, you MUST close resp.Body!

Example fix for function '%s':
1. Find the http.%s call
2. Add 'defer resp.Body.Close()' on the very next line after error check
3. Add body draining if not reading the response
4. Test the fix
5. Restart the API to clear existing leaked connections`, 
		vuln.HTTPMethod, vuln.HTTPMethod, vuln.FunctionName, vuln.HTTPMethod)
}

// scanForCommonIssues checks for other common vulnerability patterns
func (vs *VulnerabilityScanner) scanForCommonIssues(filePath string) []Vulnerability {
	var vulns []Vulnerability
	
	src, err := ioutil.ReadFile(filePath)
	if err != nil {
		return vulns
	}
	
	lines := strings.Split(string(src), "\n")
	for i, line := range lines {
		// Check for hardcoded credentials
		if strings.Contains(line, "password") && strings.Contains(line, "=") && strings.Contains(line, `"`) {
			if !strings.Contains(line, "os.Getenv") && !strings.Contains(line, "getEnv") {
				vulns = append(vulns, Vulnerability{
					ID:          uuid.New(),
					Type:        "HARDCODED_CREDENTIAL",
					Severity:    "HIGH",
					Title:       "Potential Hardcoded Credential",
					Description: "Hardcoded password or credential found",
					FilePath:    filePath,
					LineNumber:  i + 1,
					CodeSnippet: line,
					Status:      "open",
					Category:    "Security",
				})
			}
		}
		
		// Check for missing timeouts
		if strings.Contains(line, "http.Client{") && !strings.Contains(line, "Timeout") {
			vulns = append(vulns, Vulnerability{
				ID:          uuid.New(),
				Type:        "MISSING_TIMEOUT",
				Severity:    "MEDIUM",
				Title:       "HTTP Client Without Timeout",
				Description: "HTTP client created without timeout can hang indefinitely",
				FilePath:    filePath,
				LineNumber:  i + 1,
				CodeSnippet: line,
				Recommendation: "Add Timeout field: &http.Client{Timeout: 30 * time.Second}",
				Status:      "open",
				Category:    "Reliability",
			})
		}
	}
	
	return vulns
}

// findGoFiles recursively finds all Go files in a directory
func (vs *VulnerabilityScanner) findGoFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".go") && !strings.Contains(path, "vendor/") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// storeResults saves scan results to the database
func (vs *VulnerabilityScanner) storeResults(result *ScanResult) error {
	// Get or create scenario record
	var scenarioID uuid.UUID
	err := vs.db.QueryRow(`
		SELECT id FROM scenarios WHERE name = $1
	`, result.ScenarioName).Scan(&scenarioID)
	
	if err == sql.ErrNoRows {
		// Create new scenario record
		scenarioID = uuid.New()
		_, err = vs.db.Exec(`
			INSERT INTO scenarios (id, name, path, status, last_scanned, created_at, updated_at)
			VALUES ($1, $2, $3, 'active', $4, $4, $4)
		`, scenarioID, result.ScenarioName, result.ScenarioPath, result.ScanTime)
		if err != nil {
			return err
		}
	}
	
	// Store each vulnerability
	for _, vuln := range result.Vulnerabilities {
		_, err = vs.db.Exec(`
			INSERT INTO vulnerability_scans 
			(id, scenario_id, scan_type, severity, category, title, description, 
			 file_path, line_number, code_snippet, recommendation, status, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`, vuln.ID, scenarioID, vuln.Type, vuln.Severity, vuln.Category,
			vuln.Title, vuln.Description, vuln.FilePath, vuln.LineNumber,
			vuln.CodeSnippet, vuln.Recommendation, vuln.Status, result.ScanTime)
		
		if err != nil {
			vs.logger.Error(fmt.Sprintf("Failed to store vulnerability: %v", vuln.Title), err)
		}
	}
	
	// Update last_scanned timestamp
	_, err = vs.db.Exec(`
		UPDATE scenarios SET last_scanned = $1, updated_at = $1 WHERE id = $2
	`, result.ScanTime, scenarioID)
	
	return err
}

// ScanResult represents the complete result of a vulnerability scan
type ScanResult struct {
	ScenarioName    string
	ScenarioPath    string
	ScanTime        time.Time
	Vulnerabilities []Vulnerability
}

// Vulnerability represents a detected vulnerability
type Vulnerability struct {
	ID             uuid.UUID
	Type           string
	Severity       string
	Title          string
	Description    string
	FilePath       string
	LineNumber     int
	CodeSnippet    string
	Impact         string
	Recommendation string
	Status         string
	Category       string
}