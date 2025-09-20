package rules

import (
	"strings"
)

/*
Rule: Lifecycle Protection
Description: Ensure main.go files have VROOLI_LIFECYCLE_MANAGED environment check
Reason: Prevents direct execution of services, enforcing proper process management through the Vrooli lifecycle system
Category: api
Severity: critical
Standard: vrooli-lifecycle-v1
Targets: api, cli, main_go

<test-case id="missing-lifecycle-check" should-fail="true">
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

<test-case id="proper-lifecycle-check" should-fail="false">
  <description>main.go with proper lifecycle protection</description>
  <input language="go">
package main

import (
    "fmt"
    "net/http"
    "os"
)

func main() {
    // Protect against direct execution
    if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
        fmt.Fprintln(os.Stderr, "Error: Direct execution not allowed")
        fmt.Fprintln(os.Stderr, "Use: vrooli scenario run <name>")
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

<test-case id="lifecycle-check-with-lookupenv" should-fail="false">
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
        fmt.Fprintln(os.Stderr, "Error: Must be run through Vrooli lifecycle system")
        os.Exit(1)
    }

    startApplication()
}
  </input>
</test-case>
*/

// CheckLifecycleProtection validates that main functions include lifecycle protection
func CheckLifecycleProtection(content []byte, filePath string) *Violation {
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

	// Check for lifecycle protection
	hasLifecycleCheck := strings.Contains(contentStr, "VROOLI_LIFECYCLE_MANAGED") &&
		(strings.Contains(contentStr, "os.Getenv(\"VROOLI_LIFECYCLE_MANAGED\")") ||
			strings.Contains(contentStr, "os.LookupEnv(\"VROOLI_LIFECYCLE_MANAGED\")"))

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
				"    fmt.Fprintln(os.Stderr, \"Error: Direct execution not allowed\")\n" +
				"    os.Exit(1)\n" +
				"}",
			Standard: "vrooli-lifecycle-v1",
		}
	}

	return nil
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
