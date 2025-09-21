package rules

import (
	"path/filepath"
	"strings"
)

/*
Rule: Lifecycle Protection
Description: Ensure main.go files have VROOLI_LIFECYCLE_MANAGED environment check
Reason: Prevents direct execution of services, enforcing proper process management through the Vrooli lifecycle system
Category: api
Severity: critical
Standard: vrooli-lifecycle-v1
Targets: main_go

<test-case id="missing-lifecycle-check" should-fail="true" path="api/main.go" scenario="demo-app">
  <description>main.go without lifecycle protection</description>
  <input language="go">
package main

import (
    "fmt"
    "net/http"
    "os"
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    fmt.Printf("Starting server on port %s\n", port)
    http.ListenAndServe(":"+port, nil)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing Lifecycle Protection</expected-message>
</test-case>

<test-case id="proper-lifecycle-check" should-fail="false" path="api/main.go" scenario="demo-app">
  <description>main.go with proper lifecycle protection</description>
  <input language="go">
package main

import (
    "fmt"
    "net/http"
    "os"
)

func main() {
    // Protect against direct execution - must be run through lifecycle system
    if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
        fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start demo-app

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
        os.Exit(1)
    }

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    fmt.Printf("Starting server on port %s\n", port)
    http.ListenAndServe(":"+port, nil)
}
  </input>
</test-case>

<test-case id="incorrect-lifecycle-message" should-fail="true" path="api/main.go" scenario="demo-app">
  <description>lifecycle check present but message differs from required instructional text</description>
  <input language="go">
package main

import (
    "fmt"
    "net/http"
    "os"
)

func main() {
    // Protect against direct execution - must be run through lifecycle system
    if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
        fmt.Fprintln(os.Stderr, "Error: Direct execution not allowed")
        fmt.Fprintln(os.Stderr, "Use: vrooli scenario start prompt-manager")
        os.Exit(1)
    }

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    fmt.Printf("Starting server on port %s\n", port)
    http.ListenAndServe(":"+port, nil)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing Lifecycle Protection</expected-message>
</test-case>

<test-case id="lifecycle-check-with-lookupenv" should-fail="false" path="api/main.go" scenario="demo-app">
  <description>main.go using os.LookupEnv for lifecycle check</description>
  <input language="go">
package main

import (
    "fmt"
    "os"
)

func main() {
    // Check lifecycle management
    managed, exists := os.LookupEnv("VROOLI_LIFECYCLE_MANAGED")
    if !exists || managed != "true" {
        fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start demo-app

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
        os.Exit(1)
    }

    startApplication()
}
  </input>
</test-case>

<test-case id="missing-lifecycle-check-condition" should-fail="true" path="api/main.go" scenario="demo-app">
  <description>instructional message present without enforcing lifecycle condition</description>
  <input language="go">
package main

import (
    "fmt"
    "net/http"
)

func main() {
    fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start demo-app

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)

    port := "8080"
    fmt.Printf("Starting server on port %s\n", port)
    http.ListenAndServe(":"+port, nil)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing Lifecycle Protection</expected-message>
</test-case>
*/

// CheckLifecycleProtection validates that main functions include lifecycle protection
func CheckLifecycleProtection(content []byte, filePath string, scenario string) *Violation {
	contentStr := string(content)

	// Only check Go entrypoints. During unit tests the runner feeds code snippets
	// through synthetic filenames (e.g. test_<id>.go), so fall back to detecting
	// package main + func main() when the real filename hint is absent.
	hasMainFunc := strings.Contains(contentStr, "func main()")
	if !(strings.HasSuffix(filePath, "main.go") && hasMainFunc) {
		if !(strings.Contains(contentStr, "package main") && hasMainFunc) {
			return nil
		}
	}

	// Check for lifecycle protection with required instructional message
	scenarioName := resolveScenarioName(scenario, filePath)
	requiredFragments := []string{
		"fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.",
		"🚀 Instead, use:",
		"   vrooli scenario start " + scenarioName,
		"💡 The lifecycle system provides environment variables, port allocation,",
		"   and dependency management automatically. Direct execution is not supported.",
	}
	hasInstructionalMessage := true
	for _, fragment := range requiredFragments {
		if !strings.Contains(contentStr, fragment) {
			hasInstructionalMessage = false
			break
		}
	}

	hasLifecycleCheck := strings.Contains(contentStr, "VROOLI_LIFECYCLE_MANAGED") &&
		(strings.Contains(contentStr, "os.Getenv(\"VROOLI_LIFECYCLE_MANAGED\")") ||
			strings.Contains(contentStr, "os.LookupEnv(\"VROOLI_LIFECYCLE_MANAGED\")")) &&
		hasInstructionalMessage

	if !hasLifecycleCheck {
		return &Violation{
			Type:        "lifecycle_protection",
			Severity:    "critical",
			Title:       "Missing Lifecycle Protection",
			Description: "Missing Lifecycle Protection",
			FilePath:    filePath,
			LineNumber:  findMainFunctionLine(contentStr),
			CodeSnippet: extractCodeSnippet(contentStr, "func main()"),
			Recommendation: "Add lifecycle check at the start of main():\n" +
				"if os.Getenv(\"VROOLI_LIFECYCLE_MANAGED\") != \"true\" {\n" +
				"    fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.\n\n" +
				"🚀 Instead, use:\n" +
				"   vrooli scenario start " + scenarioName + "\n\n" +
				"💡 The lifecycle system provides environment variables, port allocation,\n" +
				"   and dependency management automatically. Direct execution is not supported.\n" +
				"`)\n" +
				"    os.Exit(1)\n" +
				"}\n",
			Standard: "vrooli-lifecycle-v1",
		}
	}

	return nil
}

func resolveScenarioName(provided string, filePath string) string {
	scenario := strings.TrimSpace(provided)
	if scenario != "" {
		return scenario
	}

	clean := strings.ReplaceAll(filePath, "\\", "/")
	parts := strings.Split(clean, "/")
	for i := 0; i < len(parts); i++ {
		if parts[i] == "scenarios" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	base := filepath.Base(filepath.Dir(filePath))
	if base != "." && base != "" {
		return base
	}

	return "your-scenario"
}

// Helper functions
func findMainFunctionLine(content string) int {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.Contains(line, "func main()") {
			return i + 1
		}
	}
	return 1
}

func extractCodeSnippet(content, pattern string) string {
	idx := strings.Index(content, pattern)
	if idx == -1 {
		return ""
	}

	// Extract a few lines around the pattern
	start := idx - 50
	if start < 0 {
		start = 0
	}
	end := idx + 150
	if end > len(content) {
		end = len(content)
	}

	return content[start:end]
}
