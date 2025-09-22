package cli

import (
	"strings"
)

/*
Rule: Lightweight Main Function
Description: Main functions should delegate to cmd.Execute() or similar
Reason: Keeps main() testable and separates concerns between entry point and business logic
Category: cli
Severity: medium
Standard: code-structure-v1
Targets: cli, main_go

<test-case id="complex-main-function" should-fail="true" path="cli/main.go">
  <description>Main function with too much business logic</description>
  <input language="go">
package main

import (
    "database/sql"
    "net/http"
    "log"
)

func main() {
    // Database setup
    db, err := sql.Open("postgres", "connection-string")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Create tables
    if err := createTables(db); err != nil {
        log.Fatal(err)
    }

    // Setup routes
    mux := http.NewServeMux()
    mux.HandleFunc("/api/users", handleUsers)
    mux.HandleFunc("/api/products", handleProducts)

    // Setup middleware
    handler := loggingMiddleware(mux)
    handler = authMiddleware(handler)

    // Start server
    log.Println("Starting server on :8080")
    if err := http.ListenAndServe(":8080", handler); err != nil {
        log.Fatal(err)
    }
}
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>Main function contains too much logic</expected-message>
</test-case>

<test-case id="lightweight-cobra-main" should-fail="false" path="cli/main.go">
  <description>Main function delegating to cobra cmd.Execute()</description>
  <input language="go">
package main

import (
    "fmt"
    "os"
    "myapp/cmd"
)

func main() {
    if err := cmd.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
  </input>
</test-case>

<test-case id="lightweight-urfave-cli" should-fail="false" path="cli/main.go">
  <description>Main function using urfave/cli pattern</description>
  <input language="go">
package main

import (
    "log"
    "os"
    "github.com/urfave/cli/v2"
)

func main() {
    app := &cli.App{
        Name:  "myapp",
        Usage: "Application description",
        Commands: setupCommands(),
    }

    if err := app.Run(os.Args); err != nil {
        log.Fatal(err)
    }
}
  </input>
</test-case>

<test-case id="minimal-main-with-setup" should-fail="false" path="cli/main.go">
  <description>Minimal main that calls a setup function</description>
  <input language="go">
package main

import (
    "log"
)

func main() {
    if err := runApplication(); err != nil {
        log.Fatal(err)
    }
}

func runApplication() error {
    // All business logic is here
    return nil
}
  </input>
</test-case>
*/

// CheckLightweightMain validates that main functions are properly structured
func CheckLightweightMain(content []byte, filePath string) []Violation {
	var violations []Violation
	contentStr := string(content)

	// Only check main.go files
	hasMainFunc := strings.Contains(contentStr, "func main()")
	if !(strings.HasSuffix(filePath, "main.go") && hasMainFunc) {
		if !(strings.Contains(contentStr, "package main") && hasMainFunc) {
			return violations
		}
	}

	// Find main function
	lines := strings.Split(contentStr, "\n")
	mainStart := -1
	mainEnd := -1
	braceCount := 0

	for i, line := range lines {
		if strings.Contains(line, "func main()") {
			mainStart = i
			braceCount = 1
			continue
		}

		if mainStart >= 0 {
			braceCount += strings.Count(line, "{")
			braceCount -= strings.Count(line, "}")

			if braceCount == 0 {
				mainEnd = i
				break
			}
		}
	}

	if mainStart >= 0 && mainEnd >= 0 {
		mainLines := mainEnd - mainStart

		// Check if main is too complex (more than 20 lines suggests it's doing too much)
		if mainLines > 20 {
			// Check if it's delegating to cmd.Execute or similar
			hasCmdPattern := false
			for i := mainStart; i <= mainEnd; i++ {
				line := lines[i]
				if strings.Contains(line, "cmd.Execute") ||
					strings.Contains(line, "app.Run") ||
					strings.Contains(line, "rootCmd.Execute") ||
					strings.Contains(line, "cli.Run") {
					hasCmdPattern = true
					break
				}
			}

			if !hasCmdPattern {
				violations = append(violations, Violation{
					Type:           "lightweight_main",
					Severity:       "medium",
					Title:          "Complex Main Function",
					Description:    "Main function contains too much logic (>20 lines)",
					FilePath:       filePath,
					LineNumber:     mainStart + 1,
					CodeSnippet:    "func main() { ... " + strings.TrimSpace(lines[mainStart+1]) + " ... }",
					Recommendation: "Refactor to use cmd.Execute() pattern or extract logic to separate functions",
					Standard:       "code-structure-v1",
				})
			}
		}

		// Check for direct business logic in main
		businessLogicIndicators := []string{
			"database",
			"sql.Open",
			"http.ListenAndServe",
			"gin.Run",
			"fiber.Listen",
			"for {",    // Main loops
			"select {", // Channel operations
		}

		businessLogicReported := false
		for i := mainStart + 1; i < mainEnd; i++ {
			line := lines[i]
			for _, indicator := range businessLogicIndicators {
				if strings.Contains(line, indicator) && !strings.Contains(line, "//") {
					violations = append(violations, Violation{
						Type:           "lightweight_main",
						Severity:       "medium",
						Title:          "Business Logic in Main",
						Description:    "Main function contains direct business logic",
						FilePath:       filePath,
						LineNumber:     i + 1,
						CodeSnippet:    line,
						Recommendation: "Move to a Run() or Execute() function called from main()",
						Standard:       "code-structure-v1",
					})
					businessLogicReported = true
					break
				}
			}

			if businessLogicReported {
				break
			}
		}
	}

	return violations
}
