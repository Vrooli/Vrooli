package rules

import (
	"regexp"
	"strings"
)

/*
Rule: Structured Logging
Description: Use structured logging with proper log levels
Reason: Enables efficient log aggregation, searching, and monitoring in production
Category: cli
Severity: medium
Standard: logging-v1
Targets: cli, api

<test-case id="unstructured-logging" should-fail="true" path="cli/service.go">
  <description>Using fmt.Print and basic log.Print for logging</description>
  <input language="go">
func processRequest(data string) error {
    fmt.Println("Processing request:", data)

    result, err := doWork(data)
    if err != nil {
        log.Printf("Error processing: %v", err)
        return err
    }

    fmt.Printf("Result: %s\n", result)
    return nil
}
  </input>
  <expected-violations>3</expected-violations>
  <expected-message>Unstructured Logging</expected-message>
</test-case>

<test-case id="structured-logging-with-zap" should-fail="false" path="cli/service.go">
  <description>Using structured logging with zap</description>
  <input language="go">
func processRequest(logger *zap.Logger, data string) error {
    logger.Info("Processing request", zap.String("data", data))

    result, err := doWork(data)
    if err != nil {
        logger.Error("Error processing", zap.Error(err))
        return err
    }

    logger.Info("Processing complete", zap.String("result", result))
    return nil
}
  </input>
</test-case>

<test-case id="structured-logging-with-slog" should-fail="false" path="cli/service.go">
  <description>Using structured logging with slog</description>
  <input language="go">
func handleEvent(ctx context.Context, event Event) {
    slog.InfoContext(ctx, "Event received",
        "eventID", event.ID,
        "eventType", event.Type,
    )

    if err := validateEvent(event); err != nil {
        slog.ErrorContext(ctx, "Event validation failed",
            "error", err,
            "eventID", event.ID,
        )
        return
    }

    slog.DebugContext(ctx, "Event processed successfully")
}
  </input>
</test-case>

<test-case id="custom-logger-wrapper" should-fail="false" path="cli/service.go">
  <description>Using custom NewLogger wrapper</description>
  <input language="go">
func setupService() {
    logger := NewLogger()
    logger.Info("Service starting")

    if err := connectDatabase(); err != nil {
        logger.Error("Database connection failed", err)
        os.Exit(1)
    }

    logger.Info("Service initialized successfully")
}
  </input>
</test-case>
*/

// CheckStructuredLogging validates logging patterns
func CheckStructuredLogging(content []byte, filePath string) []Violation {
	var violations []Violation
	contentStr := string(content)

	// Skip test files
	if strings.HasSuffix(filePath, "_test.go") {
		return violations
	}

	lines := strings.Split(contentStr, "\n")

	// Patterns for unstructured logging
	unstructuredPatterns := []*regexp.Regexp{
		regexp.MustCompile(`fmt\.Print(ln|f)?\(`),
		regexp.MustCompile(`log\.Print(ln|f)?\(`),
		regexp.MustCompile(`println\(`),
		regexp.MustCompile(`print\(`),
	}

	// Check for structured logging libraries
	hasStructuredLogger := strings.Contains(contentStr, "logrus") ||
		strings.Contains(contentStr, "zap.") ||
		strings.Contains(contentStr, "zerolog") ||
		strings.Contains(contentStr, "slog") ||
		strings.Contains(contentStr, "NewLogger()")

	unstructuredLines := make(map[int]bool)

	for i, line := range lines {
		// Skip comments
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "//") || trimmedLine == "" {
			continue
		}

		for _, pattern := range unstructuredPatterns {
			if pattern.MatchString(line) {
				// Check for exceptions
				if strings.Contains(line, "fmt.Sprintf") || // String formatting, not logging
					strings.Contains(line, "fmt.Errorf") || // Error creation
					strings.Contains(line, "// TODO") || // TODO comments
					strings.Contains(line, "// DEBUG") { // Debug comments
					continue
				}

				// Special case: fmt.Fprintf to os.Stderr is sometimes acceptable for startup
				if strings.Contains(line, "fmt.Fprintf(os.Stderr") {
					continue
				}

				severity := "medium"
				if hasStructuredLogger {
					severity = "low" // Less severe if they have structured logging available
				}

				violations = append(violations, Violation{
					Type:           "structured_logging",
					Severity:       severity,
					Title:          "Unstructured Logging",
					Description:    "Unstructured Logging",
					FilePath:       filePath,
					LineNumber:     i + 1,
					CodeSnippet:    line,
					Recommendation: "Use structured logging (e.g., logger.Info(), logger.Error())",
					Standard:       "logging-v1",
				})

				unstructuredLines[i] = true
			}
		}

		// Check for missing log levels
		if unstructuredLines[i] {
			continue
		}

		if strings.Contains(line, "log.") && !strings.Contains(line, "log.New") {
			if !strings.Contains(line, ".Info") &&
				!strings.Contains(line, ".Debug") &&
				!strings.Contains(line, ".Warn") &&
				!strings.Contains(line, ".Error") &&
				!strings.Contains(line, ".Fatal") &&
				!strings.Contains(line, ".Panic") {
				violations = append(violations, Violation{
					Type:           "structured_logging",
					Severity:       "low",
					Title:          "Missing Log Level",
					Description:    "Log statement without explicit level",
					FilePath:       filePath,
					LineNumber:     i + 1,
					CodeSnippet:    line,
					Recommendation: "Use explicit log levels (Info, Debug, Warn, Error)",
					Standard:       "logging-v1",
				})
			}
		}
	}

	return violations
}
