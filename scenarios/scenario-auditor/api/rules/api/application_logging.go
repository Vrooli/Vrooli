package api

import (
	"regexp"
	"strings"
)

/*
Rule: API Application Logging
Description: Ensure API servers use structured logging for operational visibility
Reason: APIs run as background services that need machine-parseable logs for monitoring, debugging, and alerting. Unstructured logging makes it impossible to aggregate, search, and alert on log data effectively.
Category: api
Severity: medium
Standard: logging-v1
Targets: api

<test-case id="api-handler-unstructured-logging" should-fail="true" path="api/handlers.go">
  <description>HTTP handlers should use structured logging, not fmt.Print or log.Print</description>
  <input language="go">
package main

import (
	"fmt"
	"log"
	"net/http"
)

func handleRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("Processing request from %s", r.RemoteAddr)
	fmt.Println("User action detected")

	// Process request
	w.WriteHeader(http.StatusOK)
}

func handleError(w http.ResponseWriter, r *http.Request) {
	log.Printf("Error occurred: database timeout")
	http.Error(w, "Internal error", 500)
}
  </input>
  <expected-violations>3</expected-violations>
  <expected-message>Unstructured Logging in API</expected-message>
</test-case>

<test-case id="api-structured-logging-zap" should-fail="false" path="api/handlers.go">
  <description>Using structured logging with zap in API handlers</description>
  <input language="go">
package main

import (
	"net/http"
	"go.uber.org/zap"
)

func handleRequest(logger *zap.Logger, w http.ResponseWriter, r *http.Request) {
	logger.Info("Processing request",
		zap.String("remote_addr", r.RemoteAddr),
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path))

	w.WriteHeader(http.StatusOK)
}

func handleError(logger *zap.Logger, w http.ResponseWriter, r *http.Request) {
	logger.Error("Database timeout",
		zap.String("operation", "query"),
		zap.Duration("timeout", 30*time.Second))
	http.Error(w, "Internal error", 500)
}
  </input>
</test-case>

<test-case id="api-structured-logging-slog" should-fail="false" path="api/handlers.go">
  <description>Using structured logging with slog in API handlers</description>
  <input language="go">
package main

import (
	"context"
	"log/slog"
	"net/http"
)

func handleRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	slog.InfoContext(ctx, "Processing request",
		"remote_addr", r.RemoteAddr,
		"method", r.Method,
		"path", r.URL.Path)

	w.WriteHeader(http.StatusOK)
}
  </input>
</test-case>

<test-case id="api-startup-output-allowed" should-fail="false" path="api/main.go">
  <description>Startup output to stderr is allowed in main function</description>
  <input language="go">
package main

import (
	"fmt"
	"os"
)

func main() {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Fprintf(os.Stderr, "Starting API server on port %s\n", port)
	fmt.Fprintf(os.Stderr, "Environment: %s\n", os.Getenv("ENV"))

	// Server initialization...
}
  </input>
</test-case>

<test-case id="api-string-formatting-allowed" should-fail="false" path="api/utils.go">
  <description>fmt.Sprintf and fmt.Errorf are allowed for string manipulation</description>
  <input language="go">
package main

import (
	"fmt"
)

func buildURL(host string, port int, path string) string {
	return fmt.Sprintf("http://%s:%d%s", host, port, path)
}

func validateInput(value string) error {
	if value == "" {
		return fmt.Errorf("value cannot be empty")
	}
	return nil
}

func formatMessage(template string, args ...interface{}) string {
	return fmt.Sprintf(template, args...)
}
  </input>
</test-case>

<test-case id="api-custom-logger-pattern" should-fail="false" path="api/logger.go">
  <description>Custom logger wrapper with structured methods is acceptable</description>
  <input language="go">
package main

import (
	"log"
	"os"
)

type Logger struct {
	*log.Logger
}

func NewLogger() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "[api] ", log.LstdFlags),
	}
}

func (l *Logger) Info(msg string) {
	l.Printf("INFO: %s", msg)
}

func (l *Logger) Error(msg string, err error) {
	l.Printf("ERROR: %s: %v", msg, err)
}

func handleRequest(logger *Logger) {
	logger.Info("Request received")
	logger.Error("Processing failed", someErr)
}
  </input>
</test-case>

<test-case id="api-mixed-bad-and-good" should-fail="true" path="api/service.go">
  <description>Even with structured logger available, raw logging is still a violation</description>
  <input language="go">
package main

import (
	"fmt"
	"log"
	"go.uber.org/zap"
)

type Service struct {
	logger *zap.Logger
}

func (s *Service) ProcessData(data string) error {
	// Bad: using fmt.Println even though logger is available
	fmt.Println("Processing started")

	// Bad: using log.Printf even though logger is available
	log.Printf("Data size: %d", len(data))

	// Good: using the structured logger
	s.logger.Info("Processing complete", zap.Int("size", len(data)))

	return nil
}
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>Unstructured Logging in API</expected-message>
</test-case>
*/

// CheckAPIApplicationLogging validates that API code uses structured logging
func CheckAPIApplicationLogging(content []byte, filePath string) []Violation {
	var violations []Violation
	contentStr := string(content)

	// Skip test files
	if strings.HasSuffix(filePath, "_test.go") {
		return violations
	}

	// Only check API files (check for /api/ or api/ prefix)
	if !strings.Contains(filePath, "/api/") && !strings.HasPrefix(filePath, "api/") {
		return violations
	}

	lines := strings.Split(contentStr, "\n")

	// Patterns for unstructured logging
	unstructuredPatterns := []*regexp.Regexp{
		regexp.MustCompile(`fmt\.Print(ln|f)?\(`),
		regexp.MustCompile(`log\.Print(ln|f)?\(`),
		regexp.MustCompile(`\bprintln\(`),
		regexp.MustCompile(`\bprint\(`),
	}

	// Check if file has structured logging setup
	hasStructuredLogger := strings.Contains(contentStr, "zap.Logger") ||
		strings.Contains(contentStr, "slog.") ||
		strings.Contains(contentStr, "logrus.") ||
		strings.Contains(contentStr, "zerolog.") ||
		strings.Contains(contentStr, "NewLogger()")

	// Track if we're in main function (for startup output exception)
	inMainFunc := false
	braceDepth := 0

	for i, line := range lines {
		lineNum := i + 1
		trimmedLine := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "//") {
			continue
		}

		// Track main function scope
		if strings.Contains(line, "func main()") {
			inMainFunc = true
			braceDepth = 0
		}
		if inMainFunc {
			braceDepth += strings.Count(line, "{") - strings.Count(line, "}")
			if braceDepth <= 0 && strings.Contains(line, "}") {
				inMainFunc = false
			}
		}

		// Check each unstructured logging pattern
		for _, pattern := range unstructuredPatterns {
			if !pattern.MatchString(line) {
				continue
			}

			// Exception 1: String formatting functions (not logging)
			if strings.Contains(line, "fmt.Sprintf") ||
				strings.Contains(line, "fmt.Errorf") {
				continue
			}

			// Exception 2: Startup output to stderr in main()
			if inMainFunc && strings.Contains(line, "fmt.Fprintf(os.Stderr") {
				continue
			}

			// Exception 3: TODO/DEBUG comments (temporary code)
			if strings.Contains(line, "// TODO") ||
				strings.Contains(line, "// DEBUG") ||
				strings.Contains(line, "// FIXME") {
				continue
			}

			// Exception 4: Assignment to variables (e.g., msg := fmt.Sprintf(...))
			if strings.Contains(line, ":=") && strings.Contains(line, "fmt.Sprintf") {
				continue
			}
			if strings.Contains(line, "=") && !strings.Contains(line, "==") &&
			   strings.Contains(line, "fmt.Sprintf") {
				continue
			}

			// This is a violation
			severity := "medium"
			recommendation := "Use structured logging (e.g., logger.Info(), logger.Error()) for API diagnostic output"

			if hasStructuredLogger {
				recommendation = "Structured logger is available in this file - use it instead of fmt.Print/log.Print"
			}

			violations = append(violations, Violation{
				Type:           "api_application_logging",
				Severity:       severity,
				Title:          "Unstructured Logging in API",
				Description:    "API code should use structured logging for better observability and monitoring",
				FilePath:       filePath,
				LineNumber:     lineNum,
				CodeSnippet:    line,
				Recommendation: recommendation,
				Standard:       "logging-v1",
			})
		}
	}

	return violations
}
