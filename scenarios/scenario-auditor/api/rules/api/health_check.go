package api

import (
	"regexp"
	"strings"
)

/*
Rule: Health Check Endpoint
Description: Services must implement a health check endpoint at EXACTLY "/health"
Reason: Required for proper service monitoring, orchestration, and lifecycle management

WHY EXACTLY "/health" (Not /healthz, /api/v1/health, /api/health, etc.):
=========================================================================
1. OVERWHELMING CONSENSUS: 96.5% of all Vrooli scenarios (109/113) use "/health"
   This is the de facto standard proven by actual implementation.

2. LIFECYCLE SYSTEM REQUIREMENT: Vrooli's lifecycle management (.vrooli/service.json)
   explicitly configures "lifecycle.health.endpoints.api" to "/health" by default.
   The process manager, service discovery, and orchestration expect this path.

3. INTEROPERABILITY: Standardized "/health" allows any scenario to check any other
   scenario's health without configuration lookup or service discovery.

4. ANTI-PATTERN: Versioning health endpoints (e.g., /api/v1/health) is incorrect because:
   - Health checks are operational concerns, not API features
   - Changing health endpoint paths breaks monitoring and orchestration
   - Load balancers and reverse proxies expect stable, unversioned health paths

5. SIMPLICITY: One exact path reduces cognitive load and configuration complexity.

ALLOWED: Scenarios MAY expose ADDITIONAL health endpoints (/api/v1/health, /healthz) for
specific integrations, but "/health" MUST exist as the primary monitoring endpoint.

RATIONALE FOR STRICT CHECKING: This rule targets ONLY api/main.go because:
- api/main.go is the service entry point and guaranteed to exist
- Validates the SERVICE exposes the endpoint (not every file that mentions health)
- Avoids false positives from utility files, tests, or client code

Category: api
Severity: medium
Standard: service-reliability-v1
Targets: main_go
Version: 2.0.0

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
  <expected-message>Missing Root-Level Health Check Endpoint</expected-message>
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
  <expected-message>Missing Root-Level Health Check Endpoint</expected-message>
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
  <expected-message>Missing Root-Level Health Check Endpoint</expected-message>
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
  <expected-message>Missing Root-Level Health Check Endpoint</expected-message>
</test-case>

<test-case id="versioned-health-endpoint-rejected" should-fail="true" path="api/main.go">
  <description>Has ONLY /api/v1/health without /health - must fail for standardization</description>
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
  <expected-violations>1</expected-violations>
  <expected-message>Missing Root-Level Health Check Endpoint</expected-message>
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

<test-case id="chi-router-get" should-fail="false" path="api/main.go">
  <description>Chi router with Get method (20+ scenarios use this)</description>
  <input language="go">
package main

import "github.com/go-chi/chi/v5"

func main() {
    router := chi.NewRouter()
    router.Get("/health", healthHandler)
    router.Get("/api/users", handlers.GetUsers)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(200)
}
  </input>
</test-case>

<test-case id="fiber-get" should-fail="false" path="api/main.go">
  <description>Fiber framework with app.Get</description>
  <input language="go">
package main

import "github.com/gofiber/fiber/v2"

func main() {
    app := fiber.New()
    app.Get("/health", healthCheck)
    app.Get("/api/users", getUsers)
    app.Listen(":8080")
}

func healthCheck(c *fiber.Ctx) error {
    return c.SendStatus(200)
}
  </input>
</test-case>

<test-case id="http-handlefunc" should-fail="false" path="api/main.go">
  <description>Standard library http.HandleFunc</description>
  <input language="go">
package main

import "net/http"

func main() {
    http.HandleFunc("/health", healthHandler)
    http.HandleFunc("/api/users", usersHandler)
    http.ListenAndServe(":8080", nil)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(200)
}
  </input>
</test-case>

<test-case id="both-exact-and-versioned" should-fail="false" path="api/main.go">
  <description>Has BOTH /health and /api/v1/health - should pass since /health exists</description>
  <input language="go">
package main

import (
    "net/http"
    "github.com/gorilla/mux"
)

func main() {
    r := mux.NewRouter()
    r.HandleFunc("/health", healthHandler).Methods("GET")
    r.HandleFunc("/api/v1/health", healthHandler).Methods("GET")
    r.HandleFunc("/api/v1/users", usersHandler).Methods("GET")
    http.ListenAndServe(":8080", r)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(200)
}
  </input>
</test-case>

<test-case id="health-subpath-not-enough" should-fail="true" path="api/main.go">
  <description>Has /health/status but not /health - must fail</description>
  <input language="go">
package main

import (
    "net/http"
    "github.com/gorilla/mux"
)

func main() {
    r := mux.NewRouter()
    r.HandleFunc("/health/status", healthStatusHandler).Methods("GET")
    r.HandleFunc("/health/detailed", healthDetailedHandler).Methods("GET")
    r.HandleFunc("/api/users", usersHandler).Methods("GET")
    http.ListenAndServe(":8080", r)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing Root-Level Health Check Endpoint</expected-message>
</test-case>

<test-case id="subrouter-with-prefix-only" should-fail="true" path="api/main.go">
  <description>CRITICAL: Has /health on PathPrefix subrouter only - creates /api/health NOT /health</description>
  <input language="go">
package main

import (
    "net/http"
    "github.com/gorilla/mux"
)

func main() {
    r := mux.NewRouter()
    api := r.PathPrefix("/api").Subrouter()
    api.HandleFunc("/health", healthHandler).Methods("GET")
    api.HandleFunc("/users", usersHandler).Methods("GET")
    http.ListenAndServe(":8080", r)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(200)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing Root-Level Health Check Endpoint</expected-message>
</test-case>

<test-case id="router-group-only" should-fail="true" path="api/main.go">
  <description>CRITICAL: Has /health on Group only - creates /api/v1/health NOT /health</description>
  <input language="go">
package main

import "github.com/gin-gonic/gin"

func main() {
    router := gin.New()
    api := router.Group("/api/v1")
    api.GET("/health", healthCheck)
    api.GET("/users", getUsers)
    router.Run(":8080")
}

func healthCheck(c *gin.Context) {
    c.JSON(200, gin.H{"status": "ok"})
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing Root-Level Health Check Endpoint</expected-message>
</test-case>

<test-case id="inline-subrouter-chain" should-fail="true" path="api/main.go">
  <description>CRITICAL: Inline chained subrouter - creates /api/health NOT /health</description>
  <input language="go">
package main

import (
    "net/http"
    "github.com/gorilla/mux"
)

func main() {
    r := mux.NewRouter()
    r.PathPrefix("/api").Subrouter().HandleFunc("/health", healthHandler)
    r.PathPrefix("/api").Subrouter().HandleFunc("/users", usersHandler)
    http.ListenAndServe(":8080", r)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(200)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing Root-Level Health Check Endpoint</expected-message>
</test-case>

<test-case id="variable-indirection" should-fail="true" path="api/main.go">
  <description>CRITICAL: Variable indirection - can't verify path at static analysis time</description>
  <input language="go">
package main

import (
    "net/http"
    "github.com/gorilla/mux"
)

func main() {
    r := mux.NewRouter()
    healthPath := "/health"
    r.HandleFunc(healthPath, healthHandler).Methods("GET")
    r.HandleFunc("/api/users", usersHandler).Methods("GET")
    http.ListenAndServe(":8080", r)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(200)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing Root-Level Health Check Endpoint</expected-message>
</test-case>

<test-case id="const-indirection" should-fail="true" path="api/main.go">
  <description>CRITICAL: Const indirection - can't verify path at static analysis time</description>
  <input language="go">
package main

import (
    "net/http"
    "github.com/gorilla/mux"
)

const HealthPath = "/health"

func main() {
    r := mux.NewRouter()
    r.HandleFunc(HealthPath, healthHandler).Methods("GET")
    r.HandleFunc("/api/users", usersHandler).Methods("GET")
    http.ListenAndServe(":8080", r)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(200)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing Root-Level Health Check Endpoint</expected-message>
</test-case>

<test-case id="string-concatenation" should-fail="true" path="api/main.go">
  <description>CRITICAL: String concatenation - can't determine final path</description>
  <input language="go">
package main

import (
    "net/http"
    "github.com/gorilla/mux"
)

func main() {
    r := mux.NewRouter()
    prefix := "/api"
    r.HandleFunc(prefix + "/health", healthHandler).Methods("GET")
    r.HandleFunc("/api/users", usersHandler).Methods("GET")
    http.ListenAndServe(":8080", r)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(200)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing Root-Level Health Check Endpoint</expected-message>
</test-case>

<test-case id="root-plus-subrouter" should-fail="false" path="api/main.go">
  <description>Has BOTH root /health AND subrouter /health - should pass</description>
  <input language="go">
package main

import (
    "net/http"
    "github.com/gorilla/mux"
)

func main() {
    r := mux.NewRouter()
    r.HandleFunc("/health", healthHandler).Methods("GET")

    api := r.PathPrefix("/api/v1").Subrouter()
    api.HandleFunc("/health", healthHandler).Methods("GET")
    api.HandleFunc("/users", usersHandler).Methods("GET")

    http.ListenAndServe(":8080", r)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(200)
}
  </input>
</test-case>

<test-case id="chi-group-with-root" should-fail="false" path="api/main.go">
  <description>Chi router with root health + group health - should pass</description>
  <input language="go">
package main

import "github.com/go-chi/chi/v5"

func main() {
    router := chi.NewRouter()
    router.Get("/health", healthHandler)

    api := router.Group(func(r chi.Router) {
        r.Get("/api/v1/health", healthHandler)
        r.Get("/api/v1/users", getUsers)
    })
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(200)
}
  </input>
</test-case>
*/

// CheckHealthCheckImplementation validates health check endpoint implementation
// This rule ONLY checks api/main.go to ensure the service exposes EXACTLY a "/health" endpoint.
func CheckHealthCheckImplementation(content []byte, filePath string) []Violation {
	var violations []Violation
	contentStr := string(content)

	// STRICT FILE TARGETING: Only check api/main.go
	// Rationale: This ensures we check the service's main entry point, not every file
	// that might have router setup (like api/server.go, api/routes.go, etc.)
	if !strings.HasSuffix(filePath, "api/main.go") {
		return violations
	}

	// NEW APPROACH: Context-aware detection
	// 1. Find all prefixed routers (subrouters, groups with path prefixes)
	// 2. Find all "/health" route registrations
	// 3. Verify at least ONE registration is on a root-level router
	prefixedRouters := findPrefixedRouters(contentStr)
	healthRegistrations := findHealthRegistrations(contentStr)

	// Check if ANY registration is on root router (not a prefixed subrouter)
	hasRootHealthEndpoint := false
	for _, reg := range healthRegistrations {
		if !isPrefixedRouter(reg.RouterVar, prefixedRouters) {
			hasRootHealthEndpoint = true
			break
		}
	}

	if !hasRootHealthEndpoint {
		// Build helpful message based on what we found
		description := "The api/main.go file must register EXACTLY a \"/health\" endpoint at the ROOT level. "
		if len(healthRegistrations) > 0 {
			// Found /health but only on prefixed routers
			description += "Found \"/health\" registered on prefixed routers only (e.g., PathPrefix, Group). " +
				"These create endpoints like /api/health, NOT /health. "
		}
		description += "This is required for Vrooli's lifecycle system (96.5% of scenarios use this standard). " +
			"Versioned health endpoints like \"/api/v1/health\" are anti-patterns for operational monitoring. " +
			"Alternative patterns like /healthz, /livez, /healthy are NOT accepted for interoperability. " +
			"You MAY expose additional health endpoints, but \"/health\" MUST be present at root level."

		violations = append(violations, Violation{
			Type:        "health_check",
			Severity:    "medium",
			Title:       "Missing Root-Level Health Check Endpoint",
			Description: description,
			FilePath:    filePath,
			LineNumber:  1,
			CodeSnippet: "// No root-level \"/health\" endpoint registration found in api/main.go",
			Recommendation: "Add a health check endpoint at EXACTLY \"/health\" on the ROOT router in api/main.go:\n" +
				"  Examples for various frameworks:\n" +
				"  • Gorilla Mux:  r.HandleFunc(\"/health\", healthHandler).Methods(\"GET\")\n" +
				"  • Chi Router:   router.Get(\"/health\", handlers.HealthHandler)\n" +
				"  • Gin:          router.GET(\"/health\", healthCheck)\n" +
				"  • Fiber:        app.Get(\"/health\", healthCheck)\n" +
				"  • stdlib http:  http.HandleFunc(\"/health\", healthHandler)\n" +
				"IMPORTANT: Register on the main router, NOT on a subrouter with PathPrefix or Group.\n" +
				"The endpoint path must be EXACTLY \"/health\" for lifecycle system compatibility.",
			Standard: "service-reliability-v1",
		})
	} else {
		// Optional check: Verify a handler function/implementation exists
		// This is a weaker check since inline handlers are valid
		hasHealthHandler := containsHealthHandlerImplementation(contentStr)

		if !hasHealthHandler {
			violations = append(violations, Violation{
				Type:       "health_check",
				Severity:   "low",
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

// HealthRegistration represents a "/health" endpoint registration
type HealthRegistration struct {
	RouterVar string // Variable name of the router (e.g., "router", "api", "v1")
	Line      int    // Line number where registration occurs
}

// findPrefixedRouters identifies router variables that have path prefixes
// Returns map[routerVarName]prefix (e.g., {"api": "/api", "v1": "/api/v1"})
//
// Detects patterns like:
//   api := router.PathPrefix("/api").Subrouter()
//   v1 := router.PathPrefix("/api/v1").Subrouter()
//   apiGroup := router.Group("/api")
//   v1Group := router.Group("/api/v1")
func findPrefixedRouters(content string) map[string]string {
	prefixed := make(map[string]string)

	// Pattern 1: Gorilla Mux PathPrefix + Subrouter
	// Example: api := router.PathPrefix("/api").Subrouter()
	pathPrefixPattern := regexp.MustCompile(`(\w+)\s*:=\s*\w+\.PathPrefix\("([^"]+)"\)\.Subrouter\(\)`)
	matches := pathPrefixPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			routerVar := match[1] // e.g., "api"
			prefix := match[2]    // e.g., "/api"
			prefixed[routerVar] = prefix
		}
	}

	// Pattern 2: Gin/Chi Group
	// Example: api := router.Group("/api")
	groupPattern := regexp.MustCompile(`(\w+)\s*:=\s*\w+\.Group\("([^"]+)"\)`)
	matches = groupPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			routerVar := match[1]
			prefix := match[2]
			prefixed[routerVar] = prefix
		}
	}

	// Pattern 3: Inline chained PathPrefix (no variable assignment)
	// Example: router.PathPrefix("/api").Subrouter().HandleFunc("/health", h)
	// These are harder to track, but we can detect the pattern and mark as suspicious
	// For now, we'll handle this in findHealthRegistrations by detecting the chain

	return prefixed
}

// findHealthRegistrations finds all "/health" route registrations
// Returns list of registrations with router variable name and line number
//
// CRITICAL: Only matches LITERAL "/health" in route registration context
// This prevents false positives from:
//   - Variable assignments: path := "/health"
//   - Comments: // TODO: add r.HandleFunc("/health", h)
//   - String literals: log.Println("Check /health")
//   - HTTP client calls: client.Get("http://localhost/health")
//
// Matches patterns like:
//   router.HandleFunc("/health", handler)
//   r.Get("/health", handler)
//   app.GET("/health", handler)
//   http.HandleFunc("/health", handler)
func findHealthRegistrations(content string) []HealthRegistration {
	var registrations []HealthRegistration

	// Pattern: <routerVar>.<Method>("/ health", <handler>)
	// Where Method is HandleFunc, Handle, GET, Get, POST, etc.
	// This ensures we ONLY match actual route registrations, not variable assignments or comments
	//
	// Regex breakdown:
	//   (\w+)                           - Capture router variable name (group 1)
	//   \.                              - Literal dot
	//   (?:HandleFunc|Handle|GET|Get|POST|Put|DELETE|Delete|PATCH|Patch|HEAD|Head|OPTIONS|Options)
	//                                   - Route registration method (non-capturing)
	//   \s*\(\s*                        - Opening paren with optional whitespace
	//   ["']                            - Opening quote (double or single)
	//   /health                         - Literal "/health" path
	//   ["']                            - Closing quote
	//   \s*,                            - Comma with optional whitespace
	pattern := regexp.MustCompile(`(\w+)\.(HandleFunc|Handle|GET|Get|POST|Put|DELETE|Delete|PATCH|Patch|HEAD|Head|OPTIONS|Options)\s*\(\s*["']/health["']\s*,`)

	matches := pattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			routerVar := match[1]

			// Find line number (approximate - count newlines before match)
			matchIndex := strings.Index(content, match[0])
			lineNum := 1 + strings.Count(content[:matchIndex], "\n")

			registrations = append(registrations, HealthRegistration{
				RouterVar: routerVar,
				Line:      lineNum,
			})
		}
	}

	// Special case: stdlib http.HandleFunc (no router variable)
	// Example: http.HandleFunc("/health", handler)
	stdlibPattern := regexp.MustCompile(`http\.(HandleFunc|Handle)\s*\(\s*["']/health["']\s*,`)
	if stdlibPattern.MatchString(content) {
		matchIndex := strings.Index(content, stdlibPattern.FindString(content))
		lineNum := 1 + strings.Count(content[:matchIndex], "\n")

		registrations = append(registrations, HealthRegistration{
			RouterVar: "http", // Use "http" as the "router var" for stdlib
			Line:      lineNum,
		})
	}

	// Special case: Inline chained subrouter
	// Example: router.PathPrefix("/api").Subrouter().HandleFunc("/health", h)
	// This is a subrouter WITHOUT a variable assignment, so we need to detect it
	inlineSubrouterPattern := regexp.MustCompile(`\.PathPrefix\([^)]+\)\.Subrouter\(\)\.(HandleFunc|Handle|GET|Get)\s*\(\s*["']/health["']`)
	if inlineSubrouterPattern.MatchString(content) {
		matchStr := inlineSubrouterPattern.FindString(content)
		matchIndex := strings.Index(content, matchStr)
		lineNum := 1 + strings.Count(content[:matchIndex], "\n")

		// Mark this with a special router var to indicate it's an inline subrouter
		registrations = append(registrations, HealthRegistration{
			RouterVar: "__inline_subrouter__",
			Line:      lineNum,
		})
	}

	return registrations
}

// isPrefixedRouter checks if a router variable is a prefixed subrouter
func isPrefixedRouter(routerVar string, prefixed map[string]string) bool {
	// Special case: inline subrouters are always considered prefixed
	if routerVar == "__inline_subrouter__" {
		return true
	}

	// Check if this router var is in our prefixed map
	_, exists := prefixed[routerVar]
	return exists
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
