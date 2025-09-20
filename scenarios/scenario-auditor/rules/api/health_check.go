package rules

import (
	"strings"
)

/*
Rule: Health Check Endpoint
Description: Services must implement health check endpoints
Reason: Required for proper service monitoring, orchestration, and lifecycle management
Category: api
Severity: high
Standard: service-reliability-v1

<test-case id="missing-health-endpoint" should-fail="true">
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

<test-case id="health-endpoint-no-handler" should-fail="true">
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

<test-case id="proper-health-endpoint" should-fail="false">
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

<test-case id="gin-health-endpoint" should-fail="false">
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
*/

// CheckHealthCheckImplementation validates health check endpoint implementation
func CheckHealthCheckImplementation(content []byte, filePath string) []Violation {
	var violations []Violation
	contentStr := string(content)

	// Only check main API files or router files
	if !strings.HasSuffix(filePath, ".go") || !isMainAPIFile(contentStr) {
		return violations
	}

	// Check for health endpoint registration
	hasHealthEndpoint := strings.Contains(contentStr, "/health") &&
		(strings.Contains(contentStr, "HandleFunc") ||
			strings.Contains(contentStr, "Handle(") ||
			strings.Contains(contentStr, ".Get(\"/health") ||
			strings.Contains(contentStr, ".GET(\"/health"))

	if !hasHealthEndpoint {
		violations = append(violations, Violation{
			Type:           "health_check",
			Severity:       "high",
			Title:          "Missing Health Check Endpoint",
			Description:    "Missing Health Check Endpoint",
			FilePath:       filePath,
			LineNumber:     1,
			CodeSnippet:    "// No health check endpoint found",
			Recommendation: "Implement a /health endpoint that returns service status",
			Standard:       "service-reliability-v1",
		})
	} else {
		// Check if health handler exists
		hasHealthHandler := strings.Contains(contentStr, "healthHandler") ||
			strings.Contains(contentStr, "HealthHandler") ||
			strings.Contains(contentStr, "healthCheck") ||
			strings.Contains(contentStr, "HealthCheck")

		if !hasHealthHandler {
			violations = append(violations, Violation{
				Type:           "health_check",
				Severity:       "medium",
				Title:          "Health Check Handler Not Found",
				Description:    "Health Check Handler Not Found",
				FilePath:       filePath,
				LineNumber:     1,
				CodeSnippet:    "// Health handler implementation missing",
				Recommendation: "Implement a proper health check handler that validates service dependencies",
				Standard:       "service-reliability-v1",
			})
		}
	}

	return violations
}

func isMainAPIFile(content string) bool {
	// Check if this is a main API file with routing setup
	return (strings.Contains(content, "func main()") ||
		strings.Contains(content, "mux.NewRouter") ||
		strings.Contains(content, "gin.New") ||
		strings.Contains(content, "fiber.New") ||
		strings.Contains(content, "echo.New")) &&
		strings.Contains(content, "HandleFunc")
}
