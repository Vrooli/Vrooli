package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Security vulnerability patterns based on our analysis
var vulnerabilityPatterns = []VulnerabilityPattern{
	{
		Type:        "http_body_leak",
		Severity:    "critical",
		Pattern:     `(http\.(Get|Post|Put|Delete|Head)|client\.(Get|Post|Do))\([^)]+\)`,
		Description: "HTTP request without corresponding defer resp.Body.Close()",
		Title:       "HTTP Response Body Leak",
		Recommendation: "Always add 'defer resp.Body.Close()' after HTTP requests",
		CanAutoFix:  true,
	},
	{
		Type:        "hardcoded_secret",
		Severity:    "critical",
		Pattern:     `(password|secret|key|token|api_key)\s*[:=]\s*["'](?!.*env|.*getenv)[^"']{8,}["']`,
		Description: "Hardcoded secret or credential found",
		Title:       "Hardcoded Secret",
		Recommendation: "Move secrets to vault or environment variables",
		CanAutoFix:  false,
	},
	{
		Type:        "sql_injection",
		Severity:    "high",
		Pattern:     `(fmt\.Sprintf.*%.*sql|Query.*\+.*%|QueryRow.*fmt\.Sprintf)`,
		Description: "Potential SQL injection vulnerability",
		Title:       "SQL Injection Risk",
		Recommendation: "Use parameterized queries with placeholders ($1, $2, etc.)",
		CanAutoFix:  false,
	},
	{
		Type:        "cors_wildcard",
		Severity:    "high",
		Pattern:     `AllowedOrigins.*\*|Access-Control-Allow-Origin.*\*`,
		Description: "CORS configured to allow all origins",
		Title:       "CORS Misconfiguration",
		Recommendation: "Restrict CORS to specific trusted origins",
		CanAutoFix:  false,
	},
	{
		Type:        "missing_input_validation",
		Severity:    "high",
		Pattern:     `mux\.Vars\([^)]*\).*Query|QueryRow.*mux\.Vars`,
		Description: "URL parameters used directly in database queries",
		Title:       "Missing Input Validation",
		Recommendation: "Validate and sanitize all user inputs before database queries",
		CanAutoFix:  false,
	},
	{
		Type:        "info_disclosure",
		Severity:    "medium",
		Pattern:     `http\.Error.*err\.Error|json.*Encode.*err\.Error`,
		Description: "Internal error details exposed to client",
		Title:       "Information Disclosure",
		Recommendation: "Return generic error messages to clients, log detailed errors server-side",
		CanAutoFix:  false,
	},
	{
		Type:        "missing_timeouts",
		Severity:    "low",
		Pattern:     `http\.Client\{\}|http\.(Get|Post).*without.*Timeout`,
		Description: "HTTP client without timeout configuration",
		Title:       "Missing HTTP Timeouts",
		Recommendation: "Configure timeouts for HTTP clients to prevent hanging connections",
		CanAutoFix:  true,
	},
	{
		Type:        "debug_code",
		Severity:    "low",
		Pattern:     `TODO|FIXME|DEBUG|fmt\.Print(?!f)|log\.Print.*debug`,
		Description: "Debug code or TODO comments in production",
		Title:       "Debug Code in Production",
		Recommendation: "Remove debug code and TODO comments before deployment",
		CanAutoFix:  false,
	},
}

type VulnerabilityPattern struct {
	Type           string
	Severity       string
	Pattern        string
	Description    string
	Title          string
	Recommendation string
	CanAutoFix     bool
}

// Scan a single Go file for security vulnerabilities
func scanFileForVulnerabilities(filePath, scenarioName string) ([]SecurityVulnerability, error) {
	var vulnerabilities []SecurityVulnerability

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")

	// Pattern-based scanning
	for _, pattern := range vulnerabilityPatterns {
		regex, err := regexp.Compile(pattern.Pattern)
		if err != nil {
			continue
		}

		matches := regex.FindAllStringIndex(contentStr, -1)
		for _, match := range matches {
			// Find line number
			lineNum := findLineNumber(contentStr, match[0])
			
			// Extract code snippet
			codeSnippet := extractCodeSnippet(lines, lineNum-1, 2)
			
			vulnerability := SecurityVulnerability{
				ID:           uuid.New().String(),
				ScenarioName: scenarioName,
				FilePath:     filePath,
				LineNumber:   lineNum,
				Severity:     pattern.Severity,
				Type:         pattern.Type,
				Title:        pattern.Title,
				Description:  pattern.Description,
				Code:         codeSnippet,
				Recommendation: pattern.Recommendation,
				CanAutoFix:   pattern.CanAutoFix,
				DiscoveredAt: time.Now(),
			}
			
			vulnerabilities = append(vulnerabilities, vulnerability)
		}
	}

	// AST-based scanning for more sophisticated analysis
	astVulns, err := scanFileWithAST(filePath, scenarioName, contentStr)
	if err == nil {
		vulnerabilities = append(vulnerabilities, astVulns...)
	}

	return vulnerabilities, nil
}

// AST-based vulnerability scanning
func scanFileWithAST(filePath, scenarioName, content string) ([]SecurityVulnerability, error) {
	var vulnerabilities []SecurityVulnerability

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return vulnerabilities, err // Return empty slice if parsing fails
	}

	// Walk the AST to find specific vulnerability patterns
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			// Check for HTTP calls without proper error handling
			if vuln := checkHTTPCall(node, fset, scenarioName, filePath); vuln != nil {
				vulnerabilities = append(vulnerabilities, *vuln)
			}
		case *ast.AssignStmt:
			// Check for hardcoded credentials in assignments
			if vuln := checkHardcodedSecrets(node, fset, scenarioName, filePath); vuln != nil {
				vulnerabilities = append(vulnerabilities, *vuln)
			}
		}
		return true
	})

	return vulnerabilities, nil
}

// Check for HTTP calls that might leak response bodies
func checkHTTPCall(call *ast.CallExpr, fset *token.FileSet, scenarioName, filePath string) *SecurityVulnerability {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			if (ident.Name == "http" && isHTTPMethod(sel.Sel.Name)) || 
			   (ident.Name == "client" && isClientMethod(sel.Sel.Name)) {
				
				pos := fset.Position(call.Pos())
				
				return &SecurityVulnerability{
					ID:           uuid.New().String(),
					ScenarioName: scenarioName,
					FilePath:     filePath,
					LineNumber:   pos.Line,
					Severity:     "critical",
					Type:         "http_body_leak",
					Title:        "Potential HTTP Response Body Leak",
					Description:  "HTTP request call detected - ensure response body is closed",
					Code:         extractLineFromFile(filePath, pos.Line),
					Recommendation: "Add 'defer resp.Body.Close()' after the HTTP request",
					CanAutoFix:   true,
					DiscoveredAt: time.Now(),
				}
			}
		}
	}
	return nil
}

// Check for hardcoded secrets in variable assignments
func checkHardcodedSecrets(assign *ast.AssignStmt, fset *token.FileSet, scenarioName, filePath string) *SecurityVulnerability {
	for i, lhs := range assign.Lhs {
		if ident, ok := lhs.(*ast.Ident); ok {
			varName := strings.ToLower(ident.Name)
			if isSecretVariableName(varName) && i < len(assign.Rhs) {
				if lit, ok := assign.Rhs[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					// Check if it's a hardcoded value (not an env var call)
					if !strings.Contains(lit.Value, "os.Getenv") && !strings.Contains(lit.Value, "env.") {
						pos := fset.Position(assign.Pos())
						
						return &SecurityVulnerability{
							ID:           uuid.New().String(),
							ScenarioName: scenarioName,
							FilePath:     filePath,
							LineNumber:   pos.Line,
							Severity:     "critical",
							Type:         "hardcoded_secret",
							Title:        "Hardcoded Secret Detected",
							Description:  fmt.Sprintf("Variable '%s' appears to contain a hardcoded secret", ident.Name),
							Code:         extractLineFromFile(filePath, pos.Line),
							Recommendation: "Move secret to vault or environment variable",
							CanAutoFix:   false,
							DiscoveredAt: time.Now(),
						}
					}
				}
			}
		}
	}
	return nil
}

// Helper functions
func isHTTPMethod(method string) bool {
	methods := []string{"Get", "Post", "Put", "Delete", "Head", "Patch"}
	for _, m := range methods {
		if method == m {
			return true
		}
	}
	return false
}

func isClientMethod(method string) bool {
	methods := []string{"Do", "Get", "Post", "Head"}
	for _, m := range methods {
		if method == m {
			return true
		}
	}
	return false
}

func isSecretVariableName(name string) bool {
	keywords := []string{"password", "secret", "key", "token", "apikey", "credential"}
	for _, keyword := range keywords {
		if strings.Contains(name, keyword) {
			return true
		}
	}
	return false
}

func findLineNumber(content string, pos int) int {
	return strings.Count(content[:pos], "\n") + 1
}

func extractCodeSnippet(lines []string, centerLine, context int) string {
	start := max(0, centerLine-context)
	end := min(len(lines), centerLine+context+1)
	
	snippet := strings.Join(lines[start:end], "\n")
	return snippet
}

func extractLineFromFile(filePath string, lineNum int) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	currentLine := 1
	
	for scanner.Scan() {
		if currentLine == lineNum {
			return scanner.Text()
		}
		currentLine++
	}
	
	return ""
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Calculate risk score based on vulnerabilities
func calculateRiskScore(vulnerabilities []SecurityVulnerability) int {
	if len(vulnerabilities) == 0 {
		return 0
	}
	
	score := 0
	for _, vuln := range vulnerabilities {
		switch vuln.Severity {
		case "critical":
			score += 25
		case "high":
			score += 15
		case "medium":
			score += 8
		case "low":
			score += 3
		}
	}
	
	// Cap at 100
	if score > 100 {
		score = 100
	}
	
	return score
}

// Generate remediation suggestions
func generateRemediationSuggestions(vulnerabilities []SecurityVulnerability) []RemediationSuggestion {
	suggestions := make(map[string]RemediationSuggestion)
	
	for _, vuln := range vulnerabilities {
		if _, exists := suggestions[vuln.Type]; !exists {
			priority := "medium"
			if vuln.Severity == "critical" || vuln.Severity == "high" {
				priority = "high"
			}
			
			suggestion := RemediationSuggestion{
				VulnerabilityType: vuln.Type,
				Priority:          priority,
				Description:       vuln.Recommendation,
			}
			
			// Add specific fix commands for auto-fixable issues
			switch vuln.Type {
			case "http_body_leak":
				suggestion.FixCommand = "Add 'defer resp.Body.Close()' after HTTP requests"
			case "hardcoded_secret":
				suggestion.FixCommand = "Move to vault: resource-vault secrets init <resource>"
			case "debug_code":
				suggestion.FixCommand = "Remove TODO/FIXME comments and debug prints"
			}
			
			suggestions[vuln.Type] = suggestion
		}
	}
	
	var result []RemediationSuggestion
	for _, suggestion := range suggestions {
		result = append(result, suggestion)
	}
	
	return result
}