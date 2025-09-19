package main

import (
	"regexp"
	"strings"
)

// RuleImplementation interface that all rules must implement
type RuleImplementation interface {
	Check(content string, filepath string) ([]Violation, error)
}

// RuleRegistry holds all rule implementations - Updated to match actual rule files
var RuleRegistry = map[string]RuleImplementation{
	"security_headers": &SecurityHeadersRule{},
	"env_validation":   &EnvValidationRule{},
	"exit_code_test":   &ExitCodeTestRule{},
}

// SecurityHeadersRule - Updated implementation matching /rules/api/security_headers.go
type SecurityHeadersRule struct{}

func (r *SecurityHeadersRule) Check(content string, filepath string) ([]Violation, error) {
	var violations []Violation
	
	// Check for HTTP response writer usage
	if !strings.Contains(content, "http.ResponseWriter") {
		// Not an HTTP handler, skip
		return violations, nil
	}
	
	// Check for CORS headers
	hasCORS := regexp.MustCompile(`w\.Header\(\)\.Set\(["']Access-Control-Allow-Origin["']`).MatchString(content)
	
	// Check for dangerous CORS configurations
	hasWildcardCORS := regexp.MustCompile(`w\.Header\(\)\.Set\(["']Access-Control-Allow-Origin["'],\s*["'][*]`).MatchString(content)
	hasCredentialsWithWildcard := hasWildcardCORS && 
		regexp.MustCompile(`w\.Header\(\)\.Set\(["']Access-Control-Allow-Credentials["'],\s*["']true`).MatchString(content)
	
	// Check for security headers (updated to match actual implementation)
	hasXFrameOptions := regexp.MustCompile(`w\.Header\(\)\.Set\(["']X-Frame-Options["']`).MatchString(content)
	hasXContentTypeOptions := regexp.MustCompile(`w\.Header\(\)\.Set\(["']X-Content-Type-Options["']`).MatchString(content)
	hasXSSProtection := regexp.MustCompile(`w\.Header\(\)\.Set\(["']X-XSS-Protection["']`).MatchString(content)
	hasHSTS := regexp.MustCompile(`w\.Header\(\)\.Set\(["']Strict-Transport-Security["']`).MatchString(content)
	
	// Check if handler writes responses
	writesResponse := regexp.MustCompile(`w\.(Write|WriteHeader|WriteString)`).MatchString(content) ||
		regexp.MustCompile(`(json|xml)\.NewEncoder\(w\)`).MatchString(content)
	
	if writesResponse {
		// Check for missing CORS headers
		if !hasCORS {
			violations = append(violations, Violation{
				RuleID:   "security_headers",
				Severity: "high",
				Message:  "Missing CORS headers in API endpoint",
				File:     filepath,
				Line:     0,
			})
		}
		
		// Check for insecure CORS
		if hasCredentialsWithWildcard {
			violations = append(violations, Violation{
				RuleID:   "security_headers",
				Severity: "critical",
				Message:  "Insecure CORS configuration: wildcard origin with credentials enabled",
				File:     filepath,
				Line:     0,
			})
		}
		
		// Check for missing security headers (updated logic)
		if !hasXFrameOptions || !hasXContentTypeOptions || !hasXSSProtection || !hasHSTS {
			missingHeaders := []string{}
			if !hasXFrameOptions {
				missingHeaders = append(missingHeaders, "X-Frame-Options")
			}
			if !hasXContentTypeOptions {
				missingHeaders = append(missingHeaders, "X-Content-Type-Options")
			}
			if !hasXSSProtection {
				missingHeaders = append(missingHeaders, "X-XSS-Protection")
			}
			if !hasHSTS {
				missingHeaders = append(missingHeaders, "Strict-Transport-Security")
			}
			
			if len(missingHeaders) > 0 {
				violations = append(violations, Violation{
					RuleID:   "security_headers",
					Severity: "high",
					Message:  "Missing security headers: " + strings.Join(missingHeaders, ", "),
					File:     filepath,
					Line:     0,
				})
			}
		}
	}
	
	return violations, nil
}

// EnvValidationRule - Updated implementation matching /rules/config/env_validation.go
type EnvValidationRule struct{}

func (r *EnvValidationRule) Check(content string, filepath string) ([]Violation, error) {
	var violations []Violation
	
	// Skip if no environment variable usage
	if !strings.Contains(content, "os.Getenv") {
		return violations, nil
	}
	
	lines := strings.Split(content, "\n")
	
	for i, line := range lines {
		// Check for os.Getenv usage
		if strings.Contains(line, "os.Getenv(") {
			envVarLine := i
			varName := extractEnvVarName(line)
			
			// Check if the variable is validated (simple heuristic)
			hasValidation := false
			sensitiveVar := isSensitiveVar(varName)
			
			// Look ahead for validation (next 5 lines)
			for j := i + 1; j < len(lines) && j < i+6; j++ {
				nextLine := lines[j]
				// Check for empty string validation
				if strings.Contains(nextLine, `== ""`) || 
				   strings.Contains(nextLine, `!= ""`) ||
				   strings.Contains(nextLine, "log.Fatal") ||
				   strings.Contains(nextLine, "panic") {
					hasValidation = true
					break
				}
			}
			
			// Check if it's being logged (potential security issue)
			if sensitiveVar {
				for j := i; j < len(lines) && j < i+10; j++ {
					if strings.Contains(lines[j], "log.") || 
					   strings.Contains(lines[j], "fmt.Print") {
						if strings.Contains(lines[j], varName) || 
						   (j == i && strings.Contains(lines[j], "os.Getenv")) {
							violations = append(violations, Violation{
								RuleID:   "env_validation",
								Severity: "high",
								Message:  "Sensitive environment variable logged: " + varName,
								File:     filepath,
								Line:     j + 1,
							})
						}
					}
				}
			}
			
			// Check for missing validation
			if !hasValidation && !strings.Contains(line, `if`) {
				// Check if there's a default value pattern
				hasDefault := false
				for j := i; j < len(lines) && j < i+3; j++ {
					if strings.Contains(lines[j], `if`) && strings.Contains(lines[j], `== ""`) {
						hasDefault = true
						break
					}
				}
				
				if !hasDefault {
					violations = append(violations, Violation{
						RuleID:   "env_validation",
						Severity: "medium",
						Message:  "Environment variable used without validation: " + varName,
						File:     filepath,
						Line:     envVarLine + 1,
					})
				}
			}
		}
	}
	
	return violations, nil
}

func extractEnvVarName(line string) string {
	start := strings.Index(line, `"`)
	if start == -1 {
		start = strings.Index(line, `'`)
	}
	if start == -1 {
		return "UNKNOWN"
	}
	
	end := strings.Index(line[start+1:], `"`)
	if end == -1 {
		end = strings.Index(line[start+1:], `'`)
	}
	if end == -1 {
		return "UNKNOWN"
	}
	
	return line[start+1 : start+1+end]
}

func isSensitiveVar(name string) bool {
	sensitive := []string{
		"PASSWORD", "SECRET", "KEY", "TOKEN", "API_KEY", "PRIVATE",
		"CREDENTIAL", "AUTH", "CERTIFICATE", "CERT",
	}
	
	upperName := strings.ToUpper(name)
	for _, s := range sensitive {
		if strings.Contains(upperName, s) {
			return true
		}
	}
	
	return false
}

// ExitCodeTestRule - Test rule for verifying exit code handling
type ExitCodeTestRule struct{}

func (r *ExitCodeTestRule) Check(content string, filepath string) ([]Violation, error) {
	// This rule doesn't check for violations
	// It's used to test exit code handling
	return []Violation{}, nil
}