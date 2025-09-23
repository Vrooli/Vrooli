package api

import (
	"strings"
)

/*
Rule: Content-Type Headers
Description: API responses must set appropriate Content-Type headers
Reason: Ensures clients can properly parse responses and prevents security issues
Category: api
Severity: medium
Standard: api-design-v1
Targets: api

<test-case id="json-without-content-type" should-fail="true">
  <description>JSON response without Content-Type header</description>
  <input language="go">
func handleAPI(w http.ResponseWriter, r *http.Request) {
    data := map[string]string{
        "message": "Hello World",
        "status": "success",
    }
    json.NewEncoder(w).Encode(data)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Incorrect Content-Type header</expected-message>
</test-case>

<test-case id="write-without-content-type" should-fail="true">
  <description>Direct write without Content-Type header</description>
  <input language="go">
func handleResponse(w http.ResponseWriter, r *http.Request) {
    response := `{"result":"ok"}`
    w.Write([]byte(response))
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Missing Content-Type Header</expected-message>
</test-case>

<test-case id="proper-json-content-type" should-fail="false">
  <description>JSON with proper Content-Type header</description>
  <input language="go">
func handleAPI(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    data := map[string]interface{}{
        "users": getUserList(),
        "total": 100,
    }
    json.NewEncoder(w).Encode(data)
}
  </input>
</test-case>

<test-case id="constants-json-header" should-fail="false">
  <description>JSON header set using shared constants</description>
  <input language="go">
const headerContentType = "Content-Type"
const mimeApplicationJSON = "application/json"

func handleAPI(w http.ResponseWriter, r *http.Request) {
    w.Header().Set(headerContentType, mimeApplicationJSON)
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
  </input>
</test-case>

<test-case id="helper-json-header" should-fail="false">
  <description>Helper function ensures JSON headers</description>
  <input language="go">
func handleAPI(w http.ResponseWriter, r *http.Request) {
    respondJSON(w, map[string]string{"status": "ok"})
}

func respondJSON(w http.ResponseWriter, payload interface{}) {
    setJSONHeader(w)
    json.NewEncoder(w).Encode(payload)
}

func setJSONHeader(w http.ResponseWriter) {
    w.Header().Set("Content-Type", "application/json")
}
  </input>
</test-case>

<test-case id="multiple-content-types" should-fail="false">
  <description>Different content types properly set</description>
  <input language="go">
func handleMultiFormat(w http.ResponseWriter, r *http.Request) {
    format := r.URL.Query().Get("format")

    if format == "xml" {
        w.Header().Set("Content-Type", "application/xml")
        xml.NewEncoder(w).Encode(data)
    } else {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(data)
    }
}
  </input>
</test-case>
*/

var jsonHeaderHelperIndicators = []string{
	"setjsonheader(",
	"ensurejsonheader(",
	"addjsonheader(",
	"appendjsonheader(",
	"withjsonheader(",
}

var jsonResponseHelperIndicators = []string{
	"writejson(",
	"respondjson(",
	"renderjson(",
	"sendjson(",
	"jsonresponse(",
	"returnjson(",
}

// CheckContentTypeHeaders validates Content-Type header usage
func CheckContentTypeHeaders(content []byte, filePath string) []Violation {
	var violations []Violation
	contentStr := string(content)

	// Only check Go API handler files
	if !strings.HasSuffix(filePath, ".go") {
		return violations
	}

	lines := strings.Split(contentStr, "\n")
	hasJSON := hasJSONResponse(contentStr)

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		lowerLine := strings.ToLower(line)

		if strings.Contains(lowerLine, "json.newencoder") || strings.Contains(lowerLine, "json.marshal") {
			if !hasHeaderBefore(lines, i, true) {
				violations = append(violations, Violation{
					Type:           "content_type_headers",
					Severity:       "medium",
					Title:          "Missing Content-Type Header",
					Description:    "Incorrect Content-Type header",
					FilePath:       filePath,
					LineNumber:     i + 1,
					CodeSnippet:    line,
					Recommendation: "Add w.Header().Set(\"Content-Type\", \"application/json\") before writing response",
					Standard:       "api-design-v1",
				})
			}
			continue
		}

		if strings.Contains(lowerLine, "w.write(") && !strings.Contains(line, "//") && !strings.Contains(lowerLine, "http.error(") {
			if hasJSON || likelyJSONWrite(lines, i) {
				if !hasHeaderBefore(lines, i, true) {
					violations = append(violations, Violation{
						Type:           "content_type_headers",
						Severity:       "medium",
						Title:          "Missing Content-Type Header",
						Description:    "Missing Content-Type Header",
						FilePath:       filePath,
						LineNumber:     i + 1,
						CodeSnippet:    line,
						Recommendation: "Set appropriate Content-Type header before writing response",
						Standard:       "api-design-v1",
					})
				}
			}
		}
	}

	return violations
}

func hasJSONResponse(content string) bool {
	lower := strings.ToLower(content)
	if strings.Contains(lower, "json.newencoder") || strings.Contains(lower, "json.marshal") || strings.Contains(lower, "application/json") {
		return true
	}
	for _, indicator := range jsonResponseHelperIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}
	return false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func hasHeaderBefore(lines []string, idx int, requireJSON bool) bool {
	from := max(0, idx-10)
	return windowContainsHeader(lines, from, idx, requireJSON)
}

func windowContainsHeader(lines []string, start, end int, requireJSON bool) bool {
	if start >= end {
		return false
	}
	lowerWindow := strings.ToLower(strings.Join(lines[start:end], "\n"))

	if strings.Contains(lowerWindow, ".header().set") && (strings.Contains(lowerWindow, "content-type") || strings.Contains(lowerWindow, "contenttype")) {
		if !requireJSON || strings.Contains(lowerWindow, "json") {
			return true
		}
	}

	if requireJSON {
		for _, indicator := range jsonHeaderHelperIndicators {
			if strings.Contains(lowerWindow, indicator) {
				return true
			}
		}
	}

	return false
}

func windowContainsJSONHelper(lines []string, start, end int) bool {
	if start >= end {
		return false
	}
	lowerWindow := strings.ToLower(strings.Join(lines[start:end], "\n"))
	for _, indicator := range jsonResponseHelperIndicators {
		if strings.Contains(lowerWindow, indicator) {
			return true
		}
	}
	return false
}

func likelyJSONWrite(lines []string, index int) bool {
	lowerLine := strings.ToLower(lines[index])
	if strings.Contains(lowerLine, "json") {
		return true
	}
	if strings.Contains(lowerLine, "{") || strings.Contains(lowerLine, "[") {
		return true
	}

	if windowContainsJSONHelper(lines, max(0, index-5), index+1) {
		return true
	}

	target := extractWriteTarget(lines[index])
	if target == "" {
		return false
	}

	for j := max(0, index-5); j < index; j++ {
		assignLine := strings.TrimSpace(lines[j])
		if strings.Contains(assignLine, target+" :=") || strings.Contains(assignLine, target+" =") {
			if isLikelyJSONLiteral(assignLine) {
				return true
			}
		}
	}

	return false
}

func extractWriteTarget(line string) string {
	start := strings.Index(line, "w.Write(")
	if start == -1 {
		return ""
	}

	rest := line[start+len("w.Write("):]
	rest = strings.TrimSpace(rest)

	if strings.HasPrefix(rest, "[]byte(") {
		rest = rest[len("[]byte("):]
	}

	// Trim trailing characters after argument
	for i := 0; i < len(rest); i++ {
		if rest[i] == ')' || rest[i] == ',' {
			rest = rest[:i]
			break
		}
	}

	rest = strings.TrimSpace(rest)
	trimChars := "\"`')"
	rest = strings.Trim(rest, trimChars)
	return strings.TrimSpace(rest)
}

func isLikelyJSONLiteral(line string) bool {
	return strings.Contains(line, "{") || strings.Contains(line, "[")
}
