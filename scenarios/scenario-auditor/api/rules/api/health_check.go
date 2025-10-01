package api

import (
	"regexp"
	"strings"
)

/*
Rule: Health Check Endpoint
Description: Services must implement health check endpoints at exactly "/health"
Reason: Required for proper service monitoring, orchestration, and lifecycle management

WHY EXACTLY "/health" (Not /healthz, /livez, /ready, /ping, /api/v1/health, etc.):
====================================================================================
1. INTEROPERABILITY: Vrooli scenarios must integrate seamlessly across the ecosystem.
   A standardized "/health" endpoint ensures any scenario can check any other scenario's
   health without configuration or discovery.

2. TOOLING COMPATIBILITY: While Kubernetes uses /healthz and /readyz, those are for
   containerized deployments. Vrooli's lifecycle system (process management, service
   discovery, dependency orchestration) expects "/health" as the canonical endpoint.

3. SIMPLICITY: One standard endpoint reduces cognitive load. Scenarios may ADD other
   endpoints (/healthz, /api/v1/health, etc.) for specific integrations, but "/health"
   MUST exist as the primary contract.

4. CONSISTENCY: With 100+ scenarios, enforcing a single pattern prevents fragmentation.
   "/health" is chosen as the most common, simple, and universal pattern.

5. LOAD BALANCER INTEGRATION: Most reverse proxies and load balancers default to /health
   for health checks without additional configuration.

NOTE: This rule checks ONLY api/main.go (not api/server.go, api/routes.go, etc.) because:
- api/main.go is the entry point and guaranteed to exist
- Checking multiple files could create false positives if /health is in one but not another
- The rule validates the SERVICE has the endpoint, not every file

Category: api
Severity: high
Standard: service-reliability-v1
Targets: main_go

<test-case id="missing-health-endpoint" should-fail="true" path="api/main.go">
  <description>API service without health check endpoint</description>
  <input language="go">
package main

import (
    "net/http"
    "github.com/gorilla/mux"
)

func main() {
    r := mux.NewRouter()
    r.HandleFunc("/api/users", getUsersHandler).Methods("GET")
    r.HandleFunc("/api/products", getProductsHandler).Methods("GET")
    http.ListenAndServe(":8080", r)
}

func getUsersHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("users"))
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing Health Check Endpoint</expected-message>
</test-case>

<test-case id="health-endpoint-no-handler" should-fail="true" path="api/main.go">
  <description>Health endpoint registered but no handler implementation</description>
  <input language="go">
package main

import (
    "net/http"
    "github.com/gorilla/mux"
)

func main() {
    r := mux.NewRouter()
    r.HandleFunc("/health", nil).Methods("GET")
    r.HandleFunc("/api/users", getUsersHandler).Methods("GET")
    http.ListenAndServe(":8080", r)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Health Check Handler Not Found</expected-message>
</test-case>

<test-case id="proper-health-endpoint" should-fail="false" path="api/main.go">
  <description>Properly implemented health check endpoint</description>
  <input language="go">
package main

import (
    "net/http"
    "encoding/json"
    "github.com/gorilla/mux"
)

func main() {
    r := mux.NewRouter()
    r.HandleFunc("/health", healthHandler).Methods("GET")
    r.HandleFunc("/api/users", getUsersHandler).Methods("GET")
    http.ListenAndServe(":8080", r)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}
  </input>
</test-case>

<test-case id="gin-health-endpoint" should-fail="false" path="api/main.go">
  <description>Health endpoint with Gin framework</description>
  <input language="go">
package main

import "github.com/gin-gonic/gin"

func main() {
    r := gin.New()
    r.GET("/health", healthCheck)
    r.GET("/api/users", getUsers)
    r.Run(":8080")
}

func healthCheck(c *gin.Context) {
    c.JSON(200, gin.H{"status": "ok"})
}
  </input>
</test-case>

<test-case id="health-in-comment-false-positive" should-fail="true" path="api/main.go">
  <description>Comment mentions /health but no actual endpoint - must fail</description>
  <input language="go">
package main

import (
    "net/http"
    "github.com/gorilla/mux"
)

func main() {
    r := mux.NewRouter()
    // TODO: Add /health endpoint for monitoring
    r.HandleFunc("/api/users", getUsersHandler).Methods("GET")
    http.ListenAndServe(":8080", r)
}

func getUsersHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("users"))
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing Health Check Endpoint</expected-message>
</test-case>

<test-case id="health-in-string-literal-false-positive" should-fail="true" path="api/main.go">
  <description>String contains /health but no actual endpoint - must fail</description>
  <input language="go">
package main

import (
    "net/http"
    "github.com/gorilla/mux"
)

func main() {
    r := mux.NewRouter()
    apiDocs := "Check service status at https://api.example.com/health"
    r.HandleFunc("/api/data", getDataHandler).Methods("GET")
    http.ListenAndServe(":8080", r)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing Health Check Endpoint</expected-message>
</test-case>

<test-case id="healthz-alternative-not-accepted" should-fail="true" path="api/main.go">
  <description>Has /healthz instead of /health - must fail for standardization</description>
  <input language="go">
package main

import (
    "net/http"
    "github.com/gorilla/mux"
)

func main() {
    r := mux.NewRouter()
    r.HandleFunc("/healthz", healthHandler).Methods("GET")
    r.HandleFunc("/api/data", dataHandler).Methods("GET")
    http.ListenAndServe(":8080", r)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(200)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing Health Check Endpoint</expected-message>
</test-case>

<test-case id="versioned-health-endpoint-accepted" should-fail="false" path="api/main.go">
  <description>Has /api/v1/health which contains /health - should pass</description>
  <input language="go">
package main

import (
    "net/http"
    "encoding/json"
    "github.com/gorilla/mux"
)

func main() {
    r := mux.NewRouter()
    r.HandleFunc("/api/v1/health", healthHandler).Methods("GET")
    r.HandleFunc("/api/v1/users", usersHandler).Methods("GET")
    http.ListenAndServe(":8080", r)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}
  </input>
</test-case>

<test-case id="both-health-and-healthz" should-fail="false" path="api/main.go">
  <description>Has both /health and /healthz - should pass since /health exists</description>
  <input language="go">
package main

import (
    "net/http"
    "github.com/gorilla/mux"
)

func main() {
    r := mux.NewRouter()
    r.HandleFunc("/health", healthHandler).Methods("GET")
    r.HandleFunc("/healthz", healthHandler).Methods("GET")
    r.HandleFunc("/api/data", dataHandler).Methods("GET")
    http.ListenAndServe(":8080", r)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(200)
}
  </input>
</test-case>

<test-case id="inline-handler-function" should-fail="false" path="api/main.go">
  <description>Inline handler function instead of named handler - should pass</description>
  <input language="go">
package main

import (
    "encoding/json"
    "net/http"
    "github.com/gorilla/mux"
)

func main() {
    r := mux.NewRouter()
    r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
    }).Methods("GET")
    r.HandleFunc("/api/users", usersHandler).Methods("GET")
    http.ListenAndServe(":8080", r)
}
  </input>
</test-case>
*/

// CheckHealthCheckImplementation validates health check endpoint implementation
// This rule ONLY checks api/main.go to ensure the service exposes a /health endpoint.
func CheckHealthCheckImplementation(content []byte, filePath string) []Violation {
	var violations []Violation
	contentStr := string(content)

	// STRICT FILE TARGETING: Only check api/main.go
	// Rationale: This ensures we check the service's main entry point, not every file
	// that might have router setup (like api/server.go, api/routes.go, etc.)
	if !strings.HasSuffix(filePath, "api/main.go") {
		return violations
	}

	// Check for /health endpoint registration using regex to avoid false positives
	// from comments, string literals, or references to external services
	hasHealthEndpoint := containsHealthRouteRegistration(contentStr)

	if !hasHealthEndpoint {
		violations = append(violations, Violation{
			Type:       "health_check",
			Severity:   "high",
			Title:      "Missing Health Check Endpoint",
			Description: "The api/main.go file must register a /health endpoint. " +
				"This is required for Vrooli's lifecycle system, service discovery, and monitoring. " +
				"The endpoint path must contain '/health' (e.g., /health, /api/v1/health). " +
				"Alternative patterns like /healthz, /livez, /ready are NOT accepted for interoperability.",
			FilePath:       filePath,
			LineNumber:     1,
			CodeSnippet:    "// No /health endpoint registration found in api/main.go",
			Recommendation: "Add a health check endpoint in api/main.go:\n" +
				"  r.HandleFunc(\"/health\", healthHandler).Methods(\"GET\")\n" +
				"or:\n" +
				"  r.GET(\"/health\", healthCheck)\n" +
				"The endpoint must contain '/health' in its path for standardization.",
			Standard: "service-reliability-v1",
		})
	} else {
		// Optional check: Verify a handler function/implementation exists
		// This is a weaker check since inline handlers are valid
		hasHealthHandler := containsHealthHandlerImplementation(contentStr)

		if !hasHealthHandler {
			violations = append(violations, Violation{
				Type:       "health_check",
				Severity:   "medium",
				Title:      "Health Check Handler Not Found",
				Description: "The /health endpoint is registered but no handler implementation was detected. " +
					"This may be a false positive if you're using an inline handler function.",
				FilePath:       filePath,
				LineNumber:     1,
				CodeSnippet:    "// Health endpoint registered but handler not found",
				Recommendation: "Ensure your health endpoint has a proper handler implementation that validates service dependencies",
				Standard:       "service-reliability-v1",
			})
		}
	}

	return violations
}

// containsHealthRouteRegistration checks if the content has a route registration
// for an endpoint containing "/health" (not /healthz, /readyz, etc.)
// Uses regex to avoid false positives from comments or string literals
func containsHealthRouteRegistration(content string) bool {
	// Pattern explanation:
	// - HandleFunc/Handle/GET/Get: Route registration methods
	// - \s*\(\s*: Optional whitespace, open paren, optional whitespace
	// - ["'`]: String delimiter (double quote, single quote, or backtick)
	// - [^"'`]*: Any characters except string delimiters (the path prefix)
	// - /health(?:[^a-z]|$): The required "/health" substring followed by non-letter or end
	//   This ensures we match "/health" and "/health/" but NOT "/healthz" or "/healthy"
	// - [^"'`]*: Any characters except string delimiters (the path suffix)
	// - ["'`]: Closing string delimiter
	//
	// This matches:
	//   HandleFunc("/health", ...)
	//   HandleFunc("/health/", ...)
	//   HandleFunc("/api/v1/health", ...)
	//   .GET("/health", ...)
	// But NOT:
	//   HandleFunc("/healthz", ...)  - /health followed by 'z'
	//   HandleFunc("/healthcheck", ...) - /health followed by 'c'
	//   // TODO: Add /health endpoint (comment)
	//   msg := "Check /health" (string literal outside route context)
	//   client.Get("http://other.com/health") (not a route registration)
	patterns := []string{
		`HandleFunc\s*\(\s*["'][^"']*\/health(?:[^a-zA-Z]|["'])[^"']*["']`,    // HandleFunc("/health" or "/path/health"
		`Handle\s*\(\s*["'][^"']*\/health(?:[^a-zA-Z]|["'])[^"']*["']`,        // Handle("/health" or "/path/health"
		`\.GET\s*\(\s*["'][^"']*\/health(?:[^a-zA-Z]|["'])[^"']*["']`,         // .GET("/health" or "/path/health"
		`\.Get\s*\(\s*["'][^"']*\/health(?:[^a-zA-Z]|["'])[^"']*["']`,         // .Get("/health" or "/path/health"
		`\.POST\s*\(\s*["'][^"']*\/health(?:[^a-zA-Z]|["'])[^"']*["']`,        // .POST("/health" - rare but possible
		`\.Post\s*\(\s*["'][^"']*\/health(?:[^a-zA-Z]|["'])[^"']*["']`,        // .Post("/health"
		`HandleFunc\s*\(\s*` + "`" + `[^` + "`" + `]*\/health(?:[^a-zA-Z]|` + "`" + `)[^` + "`" + `]*` + "`", // Backtick strings
		`Handle\s*\(\s*` + "`" + `[^` + "`" + `]*\/health(?:[^a-zA-Z]|` + "`" + `)[^` + "`" + `]*` + "`",
	}

	for _, pattern := range patterns {
		matched, err := regexp.MatchString(pattern, content)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// containsHealthHandlerImplementation checks for common handler naming patterns
// This is a secondary check and may produce false negatives with inline handlers
func containsHealthHandlerImplementation(content string) bool {
	// Common handler naming patterns
	patterns := []string{
		`func\s+healthHandler`,   // func healthHandler(...)
		`func\s+HealthHandler`,   // func HealthHandler(...)
		`func\s+healthCheck`,     // func healthCheck(...)
		`func\s+HealthCheck`,     // func HealthCheck(...)
		`func\s+handleHealth`,    // func handleHealth(...)
		`func\s+HandleHealth`,    // func HandleHealth(...)
		`func\s*\(\s*[^)]+\)\s*healthHandler`,  // Method: func (s *Server) healthHandler
		`func\s*\(\s*[^)]+\)\s*HealthHandler`,  // Method: func (s *Server) HealthHandler
		`func\s*\([^)]*\)\s*\{[^}]*status[^}]*\}`, // Inline handler with "status"
	}

	for _, pattern := range patterns {
		matched, err := regexp.MatchString(pattern, content)
		if err == nil && matched {
			return true
		}
	}

	// Also accept inline handlers (less strict check)
	// Look for HandleFunc/Handle followed by func( indicating inline handler
	inlinePattern := `(?:HandleFunc|Handle|GET|Get)\s*\([^,]*\/health[^,]*,\s*func\s*\(`
	if matched, err := regexp.MatchString(inlinePattern, content); err == nil && matched {
		return true
	}

	return false
}
