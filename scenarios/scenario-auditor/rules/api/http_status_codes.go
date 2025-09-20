package rules

import (
	"regexp"
	"strings"
)

/*
Rule: HTTP Status Codes
Description: Use proper HTTP status codes in API responses
Reason: Ensures consistent API behavior and proper client error handling
Category: api
Severity: low
Standard: api-design-v1

<test-case id="raw-numeric-status" should-fail="true">
  <description>Using raw numeric HTTP status codes</description>
  <input language="go">
func handleRequest(w http.ResponseWriter, r *http.Request) {
    if err := validateInput(r); err != nil {
        w.WriteHeader(400)
        json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
        return
    }
    w.WriteHeader(200)
    json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>Raw HTTP Status Code</expected-message>
</test-case>

<test-case id="status-ok-with-error" should-fail="true">
  <description>Returning StatusOK (200) when there's an error</description>
  <input language="go">
func handleError(w http.ResponseWriter, r *http.Request) {
    err := processRequest(r)
    if err != nil {
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
        return
    }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Incorrect Status Code</expected-message>
</test-case>

<test-case id="proper-status-constants" should-fail="false">
  <description>Using proper HTTP status constants</description>
  <input language="go">
func handleRequest(w http.ResponseWriter, r *http.Request) {
    if err := validateInput(r); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
        return
    }

    data, err := processData(r)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]string{"error": "Internal error"})
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(data)
}
  </input>
</test-case>

<test-case id="gin-proper-status" should-fail="false">
  <description>Proper status codes with Gin framework</description>
  <input language="go">
func handleGinRequest(c *gin.Context) {
    var input RequestData
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    result, err := processInput(input)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Processing failed"})
        return
    }

    c.JSON(http.StatusCreated, result)
}
  </input>
</test-case>
*/

// CheckHTTPStatusCodes validates proper usage of HTTP status codes
func CheckHTTPStatusCodes(content []byte, filePath string) []Violation {
	var violations []Violation
	contentStr := string(content)

	// Only check Go API files
	if !strings.HasSuffix(filePath, ".go") || !isAPIHandler(contentStr) {
		return violations
	}

	// Check for raw numeric status codes (anti-pattern)
	rawStatusPattern := regexp.MustCompile(`\.WriteHeader\((\d{3})\)`)

	lines := strings.Split(contentStr, "\n")
	rawStatusLines := make(map[int]bool)
	incorrectStatusLines := make(map[int]bool)
	depth := 0
	var errBlockDepths []int

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Adjust block depth for closing braces before evaluating the line
		closeCount := strings.Count(line, "}")
		if closeCount > 0 {
			depth -= closeCount
			if depth < 0 {
				depth = 0
			}
			for len(errBlockDepths) > 0 && errBlockDepths[len(errBlockDepths)-1] > depth {
				errBlockDepths = errBlockDepths[:len(errBlockDepths)-1]
			}
		}

		// Check for raw numeric codes
		if matches := rawStatusPattern.FindStringSubmatch(line); matches != nil {
			if !rawStatusLines[i] {
				violations = append(violations, Violation{
					Type:           "http_status_codes",
					Severity:       "low",
					Title:          "Raw HTTP Status Code",
					Description:    "Raw HTTP Status Code",
					FilePath:       filePath,
					LineNumber:     i + 1,
					CodeSnippet:    line,
					Recommendation: "Use http.Status* constants (e.g., http.StatusOK, http.StatusNotFound)",
					Standard:       "api-design-v1",
				})
				rawStatusLines[i] = true
			}
		}

		if strings.Contains(line, "http.StatusOK") {
			inErrorBlock := len(errBlockDepths) > 0
			containsErrorText := strings.Contains(line, "error")
			if !containsErrorText {
				for j := i; j < len(lines) && j <= i+3; j++ {
					if strings.Contains(lines[j], "\"error\"") || strings.Contains(lines[j], "err") {
						containsErrorText = true
						break
					}
				}
			}

			if (inErrorBlock || containsErrorText) && !incorrectStatusLines[i] {
				violations = append(violations, Violation{
					Type:           "http_status_codes",
					Severity:       "medium",
					Title:          "Incorrect Status Code",
					Description:    "Incorrect Status Code",
					FilePath:       filePath,
					LineNumber:     i + 1,
					CodeSnippet:    line,
					Recommendation: "Use appropriate error status codes (4xx/5xx) for errors",
					Standard:       "api-design-v1",
				})
				incorrectStatusLines[i] = true
			}
		}

		openCount := strings.Count(line, "{")
		depth += openCount
		if strings.Contains(trimmed, "if err != nil") {
			errBlockDepths = append(errBlockDepths, depth)
		}
	}

	return violations
}

func isAPIHandler(content string) bool {
	handlerIndicators := []string{
		"http.ResponseWriter",
		"*http.Request",
		"gin.Context",
		"fiber.Ctx",
		"echo.Context",
	}

	for _, indicator := range handlerIndicators {
		if strings.Contains(content, indicator) {
			return true
		}
	}
	return false
}
