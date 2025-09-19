package rules

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

// CheckContentTypeHeaders validates Content-Type header usage
func CheckContentTypeHeaders(content []byte, filePath string) []Violation {
	var violations []Violation
	contentStr := string(content)
	
	// Only check Go API handler files
	if !strings.HasSuffix(filePath, ".go") || !hasJSONResponse(contentStr) {
		return violations
	}
	
	lines := strings.Split(contentStr, "\n")
	
	// Look for JSON encoding without Content-Type
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		
		// Check for JSON encoding
		if strings.Contains(line, "json.NewEncoder(w)") || 
		   strings.Contains(line, "json.Marshal") {
			// Look backwards for Content-Type header
			hasContentType := false
			for j := max(0, i-10); j < i; j++ {
				if strings.Contains(lines[j], "Content-Type") &&
				   strings.Contains(lines[j], "application/json") {
					hasContentType = true
					break
				}
			}
			
			if !hasContentType {
				violations = append(violations, Violation{
					Type:        "content_type_headers",
					Severity:    "medium",
					Title:       "Missing Content-Type Header",
					Description: "JSON response without Content-Type header",
					FilePath:    filePath,
					LineNumber:  i + 1,
					CodeSnippet: line,
					Recommendation: "Add w.Header().Set(\"Content-Type\", \"application/json\") before writing response",
					Standard:    "api-design-v1",
				})
			}
		}
		
		// Check for Write without Content-Type
		if strings.Contains(line, "w.Write(") && !strings.Contains(line, "//") {
			hasContentType := false
			for j := max(0, i-10); j < i; j++ {
				if strings.Contains(lines[j], "Content-Type") {
					hasContentType = true
					break
				}
			}
			
			if !hasContentType {
				violations = append(violations, Violation{
					Type:        "content_type_headers",
					Severity:    "medium",
					Title:       "Missing Content-Type Header",
					Description: "Response without Content-Type header",
					FilePath:    filePath,
					LineNumber:  i + 1,
					CodeSnippet: line,
					Recommendation: "Set appropriate Content-Type header before writing response",
					Standard:    "api-design-v1",
				})
			}
		}
	}
	
	return violations
}

func hasJSONResponse(content string) bool {
	return strings.Contains(content, "json.NewEncoder") ||
	       strings.Contains(content, "json.Marshal") ||
	       strings.Contains(content, "application/json")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}