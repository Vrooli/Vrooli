package api

import (
	"regexp"
	"strings"
)

/*
Rule: Goroutine Cancellation Context
Description: Long-lived goroutines must support cancellation via context or done channels
Reason: Prevents runaway goroutines that leak resources during shutdown
Category: api
Severity: high
Standard: concurrency-v1
Targets: api

<test-case id="goroutine-no-context" should-fail="true">
  <description>Infinite goroutine loop without any cancellation mechanism</description>
  <input language="go"><![CDATA[
func startWorker() {
    go func() {
        for {
            doWork()
            time.Sleep(time.Second)
        }
    }()
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>Goroutine Without Context</expected-message>
</test-case>

<test-case id="goroutine-with-context" should-fail="false">
  <description>Goroutine that respects context cancellation</description>
  <input language="go"><![CDATA[
func startWorker(ctx context.Context) {
    go func(ctx context.Context) {
        for {
            select {
            case <-ctx.Done():
                return
            case <-time.After(time.Second):
                doWork()
            }
        }
    }(ctx)
}
]]></input>
</test-case>
*/

// CheckGoroutineContext ensures long-running goroutines expose cancellation.
func CheckGoroutineContext(content []byte, filePath string) []Violation {
	if isTestFile(filePath) {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	var violations []Violation

	pattern := regexp.MustCompile(`go\s+func\s*\(([^)]*)\)`)

	for i, line := range lines {
		match := pattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		params := strings.TrimSpace(match[1])
		if strings.Contains(params, "context.Context") {
			continue
		}

		if !goroutineHasLongLoop(lines, i) {
			continue
		}

		if goroutineHasCancellation(lines, i) {
			continue
		}

		violations = append(violations, Violation{
			Type:           "goroutine_context",
			Severity:       "high",
			Title:          "Goroutine Without Context",
			Description:    "Goroutine Without Context",
			FilePath:       filePath,
			LineNumber:     i + 1,
			CodeSnippet:    line,
			Recommendation: "Pass context.Context or done channel to enable graceful shutdown",
			Standard:       "concurrency-v1",
		})
	}

	return violations
}

func goroutineHasLongLoop(lines []string, start int) bool {
	return findWithinWindow(lines, start, 120, func(line string) bool {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "for ") || trimmed == "for {" {
			return true
		}
		return false
	})
}

func goroutineHasCancellation(lines []string, start int) bool {
	return findWithinWindow(lines, start, 160, func(line string) bool {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, ".Done()") {
			return true
		}
		if strings.Contains(trimmed, "<-ctx") || strings.Contains(trimmed, "<-done") {
			return true
		}
		if strings.Contains(trimmed, "close(done)") || strings.Contains(trimmed, "cancel()") {
			return true
		}
		return false
	})
}
