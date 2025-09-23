package api

import (
	"regexp"
	"strings"
)

/*
Rule: Security Headers
Description: Ensures API responses include proper security headers
Reason: Missing security headers can expose applications to various attacks
Category: api
Severity: high
Standard: OWASP
Targets: api

<test-case id="missing-cors-headers" should-fail="true">
  <description>API endpoint missing CORS headers</description>
  <input language="go">
func HandleRequest(w http.ResponseWriter, r *http.Request) {
    data := getData()
    json.NewEncoder(w).Encode(data)
}
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>Missing CORS headers</expected-message>
</test-case>

<test-case id="proper-cors-setup" should-fail="false">
  <description>Correctly configured CORS and security headers</description>
  <input language="go">
func HandleRequest(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "https://trusted.com")
    w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
    w.Header().Set("X-Content-Type-Options", "nosniff")
    w.Header().Set("X-Frame-Options", "DENY")
    w.Header().Set("X-XSS-Protection", "1; mode=block")
    w.Header().Set("Strict-Transport-Security", "max-age=31536000")
    json.NewEncoder(w).Encode(data)
}
  </input>
</test-case>

<test-case id="wildcard-cors-insecure" should-fail="true">
  <description>Insecure wildcard CORS configuration</description>
  <input language="go">
func HandleRequest(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
    w.Header().Set("Access-Control-Allow-Credentials", "true")
    json.NewEncoder(w).Encode(data)
}
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>Insecure CORS configuration</expected-message>
</test-case>

<test-case id="missing-security-headers" should-fail="true">
  <description>Missing security headers like X-Frame-Options</description>
  <input language="go">
func SecureEndpoint(w http.ResponseWriter, r *http.Request) {
    // Only has CORS, missing other security headers
    w.Header().Set("Access-Control-Allow-Origin", "https://app.com")
    w.Write([]byte("response"))
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing security headers</expected-message>
</test-case>

<test-case id="complete-security-headers" should-fail="false">
  <description>Complete set of security headers</description>
  <input language="go">
func SecureEndpoint(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("X-Content-Type-Options", "nosniff")
    w.Header().Set("X-Frame-Options", "DENY")
    w.Header().Set("X-XSS-Protection", "1; mode=block")
    w.Header().Set("Strict-Transport-Security", "max-age=31536000")
    w.Header().Set("Access-Control-Allow-Origin", "https://app.com")
    w.Write([]byte("response"))
}
  </input>
</test-case>
*/

type SecurityHeadersRule struct{}

// Check analyzes code for proper security header implementation
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

	// Check for security headers
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
				Line:     0, // Would need line number extraction
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

		// Check for missing security headers
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
