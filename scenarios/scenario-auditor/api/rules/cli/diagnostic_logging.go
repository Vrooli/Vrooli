package cli

import (
	"regexp"
	"strings"
)

/*
Rule: CLI Diagnostic Logging
Description: CLI tools should use fmt.Print for user output, but structured logging for diagnostics
Reason: CLIs communicate with users via stdout/stderr (using fmt.Print*), which is expected and correct. However, when CLIs need diagnostic/debug logging (not user-facing), they should use proper structured logging to distinguish operational diagnostics from user output.
Category: cli
Severity: low
Standard: logging-v1
Targets: cli

<test-case id="cli-user-output-is-fine" should-fail="false" path="cli/main.go">
  <description>CLI tools should freely use fmt.Print for user-facing output</description>
  <input language="go">
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("MyApp CLI v1.0.0")
	fmt.Println("Usage: myapp [command] [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  start    Start the service")
	fmt.Println("  stop     Stop the service")
	fmt.Println("  status   Check status")
}

func printResults(items []string) {
	fmt.Println("\nResults:")
	fmt.Println("--------")
	for i, item := range items {
		fmt.Printf("%d. %s\n", i+1, item)
	}
}

func showError(msg string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	os.Exit(1)
}
  </input>
</test-case>

<test-case id="cli-version-and-help" should-fail="false" path="cli/main.go">
  <description>Version info and help text are user-facing output, not logging</description>
  <input language="go">
package main

import (
	"fmt"
)

const VERSION = "2.1.0"

func printVersion() {
	fmt.Printf("myapp version %s\n", VERSION)
	fmt.Println("Build: 2025-01-15")
	fmt.Println("Platform: linux/amd64")
}

func printHelp() {
	fmt.Println("MyApp - Amazing CLI tool")
	fmt.Println()
	fmt.Println("Usage: myapp [command]")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  help     Show this help message")
	fmt.Println("  version  Show version information")
}
  </input>
</test-case>

<test-case id="cli-interactive-prompts" should-fail="false" path="cli/interactive.go">
  <description>Interactive prompts are user communication, not logging</description>
  <input language="go">
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func promptUser() string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter your name: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	fmt.Printf("Hello, %s!\n", name)

	fmt.Print("Continue? (y/n): ")
	response, _ := reader.ReadString('\n')

	return strings.TrimSpace(response)
}
  </input>
</test-case>

<test-case id="cli-progress-indicators" should-fail="false" path="cli/progress.go">
  <description>Progress indicators are user-facing output</description>
  <input language="go">
package main

import (
	"fmt"
	"time"
)

func processItems(items []string) {
	total := len(items)

	for i, item := range items {
		// Show progress
		fmt.Printf("\rProcessing %d/%d...", i+1, total)

		// Process the item
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("\n✓ Processing complete!")
	fmt.Printf("Processed %d items successfully\n", total)
}
  </input>
</test-case>

<test-case id="cli-diagnostic-with-logger-available" should-fail="true" path="cli/service.go">
  <description>When structured logger exists, use it for diagnostics instead of log.Printf</description>
  <input language="go">
package main

import (
	"log"
	"go.uber.org/zap"
)

type CLI struct {
	logger *zap.Logger
}

func (c *CLI) processFile(path string) error {
	// Bad: Using log.Printf for diagnostics when logger is available
	log.Printf("DEBUG: Processing file: %s", path)
	log.Printf("DEBUG: Reading file contents")

	// Process file...

	log.Printf("DEBUG: File processed successfully")
	return nil
}
  </input>
  <expected-violations>3</expected-violations>
  <expected-message>Diagnostic Logging Without Structure</expected-message>
</test-case>

<test-case id="cli-structured-diagnostics-good" should-fail="false" path="cli/service.go">
  <description>Using structured logger for CLI diagnostics is good practice</description>
  <input language="go">
package main

import (
	"fmt"
	"go.uber.org/zap"
)

type CLI struct {
	logger *zap.Logger
}

func (c *CLI) processFile(path string) error {
	// Good: User-facing output
	fmt.Printf("Processing: %s\n", path)

	// Good: Diagnostic logging with structure
	c.logger.Debug("File processing started",
		zap.String("path", path),
		zap.Int64("size", getFileSize(path)))

	// Process file...

	// Good: User-facing success message
	fmt.Println("✓ Done")

	// Good: Diagnostic completion log
	c.logger.Info("File processing complete",
		zap.String("path", path))

	return nil
}
  </input>
</test-case>

<test-case id="cli-debug-patterns" should-fail="true" path="cli/debug.go">
  <description>Debug output using log.Printf should be flagged when logger available</description>
  <input language="go">
package main

import (
	"log"
	"log/slog"
)

var logger *slog.Logger

func debugOperation(data string) {
	// Bad: Logger is available globally, should use it
	log.Printf("DEBUG: Operation started with data: %s", data)
	log.Printf("DEBUG: Processing...")
	log.Printf("DEBUG: Operation complete")
}
  </input>
  <expected-violations>3</expected-violations>
  <expected-message>Diagnostic Logging Without Structure</expected-message>
</test-case>

<test-case id="cli-no-logger-available" should-fail="false" path="cli/simple.go">
  <description>If no structured logger exists, log.Printf is acceptable for simple CLIs</description>
  <input language="go">
package main

import (
	"fmt"
	"log"
)

func main() {
	// User output
	fmt.Println("Simple CLI Tool")

	// Simple diagnostic logging (no structured logger available)
	log.Printf("Starting operation...")

	// Do work...

	log.Printf("Operation complete")
}
  </input>
</test-case>
*/

// CheckCLIDiagnosticLogging validates that CLIs use appropriate logging
func CheckCLIDiagnosticLogging(content []byte, filePath string) []Violation {
	var violations []Violation
	contentStr := string(content)

	// Skip test files
	if strings.HasSuffix(filePath, "_test.go") {
		return violations
	}

	// Only check CLI files (check for /cli/ or cli/ prefix)
	if !strings.Contains(filePath, "/cli/") && !strings.HasPrefix(filePath, "cli/") {
		return violations
	}

	lines := strings.Split(contentStr, "\n")

	// Check if file has structured logging available
	hasStructuredLogger := strings.Contains(contentStr, "zap.Logger") ||
		strings.Contains(contentStr, "slog.Logger") ||
		strings.Contains(contentStr, "logrus.Logger") ||
		strings.Contains(contentStr, "zerolog.Logger") ||
		(strings.Contains(contentStr, "logger *") && strings.Contains(contentStr, "type "))

	// Pattern for basic log.Printf (not structured)
	logPrintfPattern := regexp.MustCompile(`log\.Print(ln|f)?\(`)

	// Only flag log.Printf if structured logger is available
	if !hasStructuredLogger {
		// No structured logger, simple log.Printf is fine for CLIs
		return violations
	}

	// We have a structured logger available, so flag log.Printf usage
	for i, line := range lines {
		lineNum := i + 1
		trimmedLine := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "//") {
			continue
		}

		// Check for log.Printf when structured logger is available
		if logPrintfPattern.MatchString(line) {
			// Exception: TODO/DEBUG comments (temporary code)
			if strings.Contains(line, "// TODO") ||
				strings.Contains(line, "// DEBUG") ||
				strings.Contains(line, "// FIXME") {
				continue
			}

			violations = append(violations, Violation{
				Type:           "cli_diagnostic_logging",
				Severity:       "low",
				Title:          "Diagnostic Logging Without Structure",
				Description:    "Structured logger is available - use it for diagnostic output instead of log.Printf",
				FilePath:       filePath,
				LineNumber:     lineNum,
				CodeSnippet:    line,
				Recommendation: "Use structured logger (logger.Debug(), logger.Info()) for diagnostic output. Keep fmt.Print* for user-facing messages.",
				Standard:       "logging-v1",
			})
		}
	}

	return violations
}
