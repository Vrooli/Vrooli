package rules

import (
	"regexp"
	"strings"
)

/*
Rule: Versioned API Endpoints
Description: API endpoints should include version in path (e.g., /api/v1/)
Reason: Enables backward compatibility and smooth API evolution without breaking existing clients
Category: api
Severity: medium
Standard: api-design-v1

<test-case id="unversioned-endpoints" should-fail="true">
  <description>API endpoints without version prefix</description>
  <input language="go">
func setupRoutes(r *mux.Router) {
    r.HandleFunc("/users", getUsersHandler).Methods("GET")
    r.HandleFunc("/products", getProductsHandler).Methods("GET")
    r.HandleFunc("/orders", createOrderHandler).Methods("POST")
}
  </input>
  <expected-violations>3</expected-violations>
  <expected-message>Unversioned API Endpoint</expected-message>
</test-case>

<test-case id="mixed-versioned-endpoints" should-fail="true">
  <description>Some endpoints versioned, others not</description>
  <input language="go">
func setupAPI(r *mux.Router) {
    r.HandleFunc("/api/v1/users", getUsersHandler).Methods("GET")
    r.HandleFunc("/products", getProductsHandler).Methods("GET")
    r.HandleFunc("/health", healthHandler).Methods("GET")
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Unversioned API Endpoint</expected-message>
</test-case>

<test-case id="properly-versioned-endpoints" should-fail="false">
  <description>All API endpoints properly versioned</description>
  <input language="go">
func setupRoutes(r *mux.Router) {
    api := r.PathPrefix("/api/v1").Subrouter()
    api.HandleFunc("/users", getUsersHandler).Methods("GET")
    api.HandleFunc("/products", getProductsHandler).Methods("GET")
    api.HandleFunc("/orders", createOrderHandler).Methods("POST")
    
    // Special endpoints don't need versioning
    r.HandleFunc("/health", healthHandler).Methods("GET")
    r.HandleFunc("/metrics", metricsHandler).Methods("GET")
}
  </input>
</test-case>

<test-case id="fiber-versioned-endpoints" should-fail="false">
  <description>Fiber framework with versioned endpoints</description>
  <input language="go">
func setupFiberRoutes(app *fiber.App) {
    api := app.Group("/api/v1")
    api.Get("/users", getUsersHandler)
    api.Post("/users", createUserHandler)
    api.Put("/users/:id", updateUserHandler)
    
    app.Get("/health", healthCheck)
}
  </input>
</test-case>
*/

// CheckVersionedEndpoints validates that API endpoints include versioning
func CheckVersionedEndpoints(content []byte, filePath string) []Violation {
	var violations []Violation
	contentStr := string(content)
	
	// Only check Go API files
	if !strings.HasSuffix(filePath, ".go") || !isAPIFile(contentStr) {
		return violations
	}
	
	// Patterns for unversioned endpoints
	patterns := []string{
		`\.HandleFunc\("/([\w-]+)"`,            // Router patterns
		`\.Handle\("/([\w-]+)"`,                 
		`app\.(Get|Post|Put|Delete|Patch)\("/([\w-]+)"`, // Fiber patterns
		`http\.HandleFunc\("/([\w-]+)"`,        // Standard HTTP
	}
	
	lines := strings.Split(contentStr, "\n")
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		for i, line := range lines {
			if matches := re.FindStringSubmatch(line); matches != nil {
				endpoint := matches[0]
				// Check if it's not versioned and not a special endpoint
				if !strings.Contains(endpoint, "/api/v") && 
				   !strings.Contains(endpoint, "/health") &&
				   !strings.Contains(endpoint, "/metrics") {
					violations = append(violations, Violation{
						Type:        "versioned_endpoints",
						Severity:    "medium",
						Title:       "Unversioned API Endpoint",
						Description: "API endpoint lacks version prefix",
						FilePath:    filePath,
						LineNumber:  i + 1,
						CodeSnippet: line,
						Recommendation: "Add version prefix like /api/v1/ to the endpoint",
						Standard:    "api-design-v1",
					})
				}
			}
		}
	}
	
	return violations
}

func isAPIFile(content string) bool {
	apiIndicators := []string{
		"HandleFunc",
		"http.Handler",
		"gin.Engine",
		"fiber.App",
		"echo.Echo",
		"chi.Router",
		"mux.Router",
	}
	
	for _, indicator := range apiIndicators {
		if strings.Contains(content, indicator) {
			return true
		}
	}
	return false
}