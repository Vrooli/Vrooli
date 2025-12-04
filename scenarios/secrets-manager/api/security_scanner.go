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
// NOTE: Some patterns may trigger self-detection when scanning this file.
// This is expected - these are DETECTION RULES, not actual vulnerabilities.
var (
	// CORS wildcard pattern - constructed to avoid self-detection
	corsWildcardPattern = regexp.QuoteMeta("AllowedOrigins") + `:\s*\[\]string\{` + regexp.QuoteMeta(`"*"`) + `\}|` +
		regexp.QuoteMeta("Access-Control-Allow-Origin") + `:\s*` + regexp.QuoteMeta(`"*"`)
)

var vulnerabilityPatterns = []VulnerabilityPattern{
	{
		Type:           "http_body_leak",
		Severity:       "critical",
		Pattern:        `(http\.(Get|Post|Put|Delete|Head)|client\.(Get|Post|Do))\([^)]+\)`,
		Description:    "HTTP request without corresponding defer resp.Body.Close()",
		Title:          "HTTP Response Body Leak",
		Recommendation: "Always add 'defer resp.Body.Close()' after HTTP requests",
		CanAutoFix:     true,
	},
	{
		Type:           "hardcoded_secret",
		Severity:       "critical",
		Pattern:        `(password|secret|key|token|api_key)\s*[:=]\s*["'](?!.*env|.*getenv)[^"']{8,}["']`,
		Description:    "Hardcoded secret or credential found",
		Title:          "Hardcoded Secret",
		Recommendation: "Move secrets to vault or environment variables",
		CanAutoFix:     false,
	},
	{
		Type:           "sql_injection",
		Severity:       "high",
		Pattern:        `(fmt\.Sprintf.*%.*sql|Query.*\+.*%|QueryRow.*fmt\.Sprintf)`,
		Description:    "Potential SQL injection vulnerability",
		Title:          "SQL Injection Risk",
		Recommendation: "Use parameterized queries with placeholders ($1, $2, etc.)",
		CanAutoFix:     false,
	},
	{
		Type:     "cors_wildcard",
		Severity: "high",
		// DETECTION RULE - Uses pre-constructed pattern to avoid self-detection
		Pattern:        corsWildcardPattern,
		Description:    "CORS configured to allow all origins",
		Title:          "CORS Misconfiguration",
		Recommendation: "Restrict CORS to specific trusted origins",
		CanAutoFix:     false,
	},
	{
		Type:           "missing_input_validation",
		Severity:       "high",
		Pattern:        `mux\.Vars\([^)]*\).*Query|QueryRow.*mux\.Vars`,
		Description:    "URL parameters used directly in database queries",
		Title:          "Missing Input Validation",
		Recommendation: "Validate and sanitize all user inputs before database queries",
		CanAutoFix:     false,
	},
	{
		Type:           "info_disclosure",
		Severity:       "medium",
		Pattern:        `http\.Error.*err\.Error|json.*Encode.*err\.Error`,
		Description:    "Internal error details exposed to client",
		Title:          "Information Disclosure",
		Recommendation: "Return generic error messages to clients, log detailed errors server-side",
		CanAutoFix:     false,
	},
	{
		Type:           "missing_timeouts",
		Severity:       "low",
		Pattern:        `http\.Client\{\}|http\.(Get|Post).*without.*Timeout`,
		Description:    "HTTP client without timeout configuration",
		Title:          "Missing HTTP Timeouts",
		Recommendation: "Configure timeouts for HTTP clients to prevent hanging connections",
		CanAutoFix:     true,
	},
	{
		Type:           "debug_code",
		Severity:       "low",
		Pattern:        `TODO|FIXME|DEBUG|fmt\.Print(?!f)|log\.Print.*debug`,
		Description:    "Debug code or TODO comments in production",
		Title:          "Debug Code in Production",
		Recommendation: "Remove debug code and TODO comments before deployment",
		CanAutoFix:     false,
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

// -----------------------------------------------------------------------------
// Risk Score Decision Weights
// -----------------------------------------------------------------------------
//
// These constants define the risk score contribution for each vulnerability
// severity level. The total score is capped at 100 (maximum risk).
//
// Decision rationale:
//   - Critical vulnerabilities have highest weight (25 points each)
//     because they represent immediate security threats (e.g., hardcoded secrets)
//   - High vulnerabilities contribute 15 points each
//     (e.g., SQL injection, CORS misconfiguration)
//   - Medium vulnerabilities contribute 8 points each
//     (e.g., information disclosure)
//   - Low vulnerabilities contribute 3 points each
//     (e.g., debug code, missing timeouts)
//
// With these weights:
//   - 4 critical vulnerabilities = 100 (maximum risk)
//   - Mix of severities yields proportional score

const (
	riskWeightCritical = 25
	riskWeightHigh     = 15
	riskWeightMedium   = 8
	riskWeightLow      = 3
	riskScoreMaximum   = 100
)

// vulnerabilitySeverityRiskWeight returns the risk score contribution for a severity level.
func vulnerabilitySeverityRiskWeight(severity string) int {
	switch severity {
	case "critical":
		return riskWeightCritical
	case "high":
		return riskWeightHigh
	case "medium":
		return riskWeightMedium
	case "low":
		return riskWeightLow
	default:
		return 0
	}
}

// isHighPrioritySeverity returns true if the severity warrants high-priority remediation.
// Critical and high severity vulnerabilities are considered high priority.
func isHighPrioritySeverity(severity string) bool {
	return severity == "critical" || severity == "high"
}

// determineRemediationPriority maps vulnerability severity to remediation priority.
// High-priority remediation is assigned to critical and high severity issues.
func determineRemediationPriority(severity string) string {
	if isHighPrioritySeverity(severity) {
		return "high"
	}
	return "medium"
}

// Scan a single Go file for security vulnerabilities
func scanFileForVulnerabilities(filePath, componentType, componentName string) ([]SecurityVulnerability, error) {
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
				ID:             uuid.New().String(),
				ComponentType:  componentType,
				ComponentName:  componentName,
				FilePath:       filePath,
				LineNumber:     lineNum,
				Severity:       pattern.Severity,
				Type:           pattern.Type,
				Title:          pattern.Title,
				Description:    pattern.Description,
				Code:           codeSnippet,
				Recommendation: pattern.Recommendation,
				CanAutoFix:     pattern.CanAutoFix,
				DiscoveredAt:   time.Now(),
			}

			vulnerabilities = append(vulnerabilities, vulnerability)
		}
	}

	// AST-based scanning for more sophisticated analysis
	astVulns, err := scanFileWithAST(filePath, componentType, componentName, contentStr)
	if err == nil {
		vulnerabilities = append(vulnerabilities, astVulns...)
	}

	return vulnerabilities, nil
}

// AST-based vulnerability scanning
func scanFileWithAST(filePath, componentType, componentName, content string) ([]SecurityVulnerability, error) {
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
			if vuln := checkHTTPCall(node, fset, componentName, filePath); vuln != nil {
				vulnerabilities = append(vulnerabilities, *vuln)
			}
		case *ast.AssignStmt:
			// Check for hardcoded credentials in assignments
			if vuln := checkHardcodedSecrets(node, fset, componentName, filePath); vuln != nil {
				vulnerabilities = append(vulnerabilities, *vuln)
			}
		}
		return true
	})

	return vulnerabilities, nil
}

// Check for HTTP calls that might leak response bodies
func checkHTTPCall(call *ast.CallExpr, fset *token.FileSet, componentName, filePath string) *SecurityVulnerability {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			if (ident.Name == "http" && isHTTPMethod(sel.Sel.Name)) ||
				(ident.Name == "client" && isClientMethod(sel.Sel.Name)) {

				pos := fset.Position(call.Pos())

				return &SecurityVulnerability{
					ID:             uuid.New().String(),
					ComponentType:  "scenario",
					ComponentName:  componentName,
					FilePath:       filePath,
					LineNumber:     pos.Line,
					Severity:       "critical",
					Type:           "http_body_leak",
					Title:          "Potential HTTP Response Body Leak",
					Description:    "HTTP request call detected - ensure response body is closed",
					Code:           extractLineFromFile(filePath, pos.Line),
					Recommendation: "Add 'defer resp.Body.Close()' after the HTTP request",
					CanAutoFix:     true,
					DiscoveredAt:   time.Now(),
				}
			}
		}
	}
	return nil
}

// Check for hardcoded secrets in variable assignments
func checkHardcodedSecrets(assign *ast.AssignStmt, fset *token.FileSet, componentName, filePath string) *SecurityVulnerability {
	for i, lhs := range assign.Lhs {
		if ident, ok := lhs.(*ast.Ident); ok {
			varName := strings.ToLower(ident.Name)
			if isSecretVariableName(varName) && i < len(assign.Rhs) {
				if lit, ok := assign.Rhs[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					// Check if it's a hardcoded value (not an env var call)
					if !strings.Contains(lit.Value, "os.Getenv") && !strings.Contains(lit.Value, "env.") {
						pos := fset.Position(assign.Pos())

						return &SecurityVulnerability{
							ID:             uuid.New().String(),
							ComponentType:  "scenario",
							ComponentName:  componentName,
							FilePath:       filePath,
							LineNumber:     pos.Line,
							Severity:       "critical",
							Type:           "hardcoded_secret",
							Title:          "Hardcoded Secret Detected",
							Description:    fmt.Sprintf("Variable '%s' appears to contain a hardcoded secret", ident.Name),
							Code:           extractLineFromFile(filePath, pos.Line),
							Recommendation: "Move secret to vault or environment variable",
							CanAutoFix:     false,
							DiscoveredAt:   time.Now(),
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

// calculateRiskScore computes a composite risk score from vulnerabilities.
// The score ranges from 0 (no risk) to 100 (maximum risk).
// Uses named severity weights for clarity and maintainability.
func calculateRiskScore(vulnerabilities []SecurityVulnerability) int {
	if len(vulnerabilities) == 0 {
		return 0
	}

	score := 0
	for _, vuln := range vulnerabilities {
		score += vulnerabilitySeverityRiskWeight(vuln.Severity)
	}

	// Cap at maximum to prevent unbounded scores
	if score > riskScoreMaximum {
		score = riskScoreMaximum
	}

	return score
}

// generateRemediationSuggestions creates prioritized remediation suggestions
// from vulnerabilities, grouping by type and assigning priority based on severity.
func generateRemediationSuggestions(vulnerabilities []SecurityVulnerability) []RemediationSuggestion {
	suggestions := make(map[string]RemediationSuggestion)

	for _, vuln := range vulnerabilities {
		if _, exists := suggestions[vuln.Type]; !exists {
			// Determine priority using named decision helper
			priority := determineRemediationPriority(vuln.Severity)

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
