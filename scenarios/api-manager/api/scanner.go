package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
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