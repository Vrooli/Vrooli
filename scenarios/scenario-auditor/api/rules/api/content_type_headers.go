package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

/*
Rule: JSON Content-Type Headers
Description: JSON API responses must set Content-Type: application/json header
Reason: Ensures clients can properly parse JSON responses and prevents security issues like content sniffing
Category: api
Severity: medium
Standard: api-design-v1
Targets: api
Note: This rule specifically checks JSON responses. For other content types (HTML, XML, PDF, CSV), see companion rules.

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
  <expected-message>JSON response missing Content-Type header</expected-message>
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
  <expected-message>JSON response missing Content-Type header</expected-message>
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

<test-case id="gin-framework" should-fail="false">
  <description>Gin framework automatically sets Content-Type</description>
  <input language="go">
func handleGin(c *gin.Context) {
    c.JSON(200, gin.H{"status": "ok"})
}

func handleGinIndented(c *gin.Context) {
    c.IndentedJSON(200, map[string]string{"status": "ok"})
}
  </input>
</test-case>

<test-case id="echo-framework" should-fail="false">
  <description>Echo framework automatically sets Content-Type</description>
  <input language="go">
func handleEcho(c echo.Context) error {
    return c.JSON(200, map[string]string{"status": "ok"})
}
  </input>
</test-case>

<test-case id="long-handler-with-header" should-fail="false">
  <description>Header set more than 10 but less than 30 lines before encoding</description>
  <input language="go">
func complexHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    // Validate input
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", 405)
        return
    }

    // Parse request
    var req Request
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", 400)
        return
    }

    // Fetch data
    data, err := fetchData(r.Context(), req.ID)
    if err != nil {
        http.Error(w, "Data fetch failed", 500)
        return
    }

    // Process data
    result := processData(data)

    // Encode response (line ~25 from header)
    json.NewEncoder(w).Encode(result)
}
  </input>
</test-case>

<test-case id="header-too-far" should-fail="true">
  <description>Header set more than 30 lines before encoding</description>
  <input language="go">
func veryLongHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    // Line 3
    // Line 4
    // Line 5
    // Line 6
    // Line 7
    // Line 8
    // Line 9
    // Line 10
    // Line 11
    // Line 12
    // Line 13
    // Line 14
    // Line 15
    // Line 16
    // Line 17
    // Line 18
    // Line 19
    // Line 20
    // Line 21
    // Line 22
    // Line 23
    // Line 24
    // Line 25
    // Line 26
    // Line 27
    // Line 28
    // Line 29
    // Line 30
    // Line 31
    // Line 32
    json.NewEncoder(w).Encode(data)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>JSON response missing Content-Type header</expected-message>
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

// Web framework patterns that automatically set Content-Type
var frameworkJSONPatterns = []string{
	// Gin framework
	"c.json(",
	"ctx.json(",
	"c.indentedjson(",
	"c.securejson(",
	"c.jsonp(",
	"c.asciijson(",
	"c.purejson(",
	// Echo framework
	"c.json(",
	"ctx.json(",
	"c.jsonblob(",
	"c.jsonpretty(",
	// Fiber framework
	"c.json(",
	"ctx.json(",
	// Chi with render
	"render.json(",
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

	// Get AST-detected helper functions that set JSON headers
	jsonHelpers := getJSONHelperFunctions(contentStr)

	// Detect middleware patterns (functions that wrap handlers and set headers)
	hasMiddleware := detectJSONMiddleware(contentStr)

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		lowerLine := strings.ToLower(line)

		// Skip framework patterns - they handle Content-Type automatically
		if isFrameworkJSONCall(lowerLine) {
			continue
		}

		// Skip if middleware detected (may set headers globally)
		if hasMiddleware && isInHandlerFunction(lines, i) {
			// Still check but be less strict - only flag if NO header anywhere in file
			if !strings.Contains(contentStr, "Content-Type") && !strings.Contains(contentStr, "content-type") {
				// No headers at all - flag this
			} else {
				// Middleware might be setting it - skip
				continue
			}
		}

		// Check for AST-detected helper function calls
		if containsJSONHelper(lowerLine, jsonHelpers) {
			continue
		}

		if strings.Contains(lowerLine, "json.newencoder") || strings.Contains(lowerLine, "json.marshal") {
			if !hasHeaderBefore(lines, i, true) && !containsJSONHelper(lowerLine, jsonHelpers) {
				violations = append(violations, Violation{
					Type:           "content_type_headers",
					Severity:       "medium",
					Title:          "Missing JSON Content-Type Header",
					Description:    "JSON response missing Content-Type header",
					Message:        "JSON response missing Content-Type header",
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
						Title:          "Missing JSON Content-Type Header",
						Description:    "JSON response missing Content-Type header",
						Message:        "JSON response missing Content-Type header",
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

// isFrameworkJSONCall detects web framework JSON methods that auto-set Content-Type
func isFrameworkJSONCall(lowerLine string) bool {
	for _, pattern := range frameworkJSONPatterns {
		if strings.Contains(lowerLine, pattern) {
			return true
		}
	}
	return false
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
	// Increased from 10 to 30 lines to handle realistic handlers with:
	// - Input validation (3-5 lines)
	// - Data fetching (5-10 lines)
	// - Error handling (3-5 lines)
	// - Business logic (5-10 lines)
	from := max(0, idx-30)
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

	// Increased lookback window for JSON helper detection
	if windowContainsJSONHelper(lines, max(0, index-15), index+1) {
		return true
	}

	target := extractWriteTarget(lines[index])
	if target == "" {
		return false
	}

	// Increased lookback window for variable assignment tracking
	for j := max(0, index-15); j < index; j++ {
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

// getJSONHelperFunctions uses AST to find functions that set JSON headers
// Returns a map of function names that are known to set Content-Type: application/json
func getJSONHelperFunctions(content string) map[string]bool {
	helpers := make(map[string]bool)

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		// Fallback to string-based detection if AST parsing fails
		return helpers
	}

	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		// Check if function body contains Content-Type header setting
		hasJSONHeader := false
		ast.Inspect(fn.Body, func(inner ast.Node) bool {
			call, ok := inner.(*ast.CallExpr)
			if !ok {
				return true
			}

			// Check for w.Header().Set("Content-Type", "application/json")
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			if sel.Sel.Name == "Set" && len(call.Args) >= 2 {
				// Check first arg is "Content-Type"
				if lit1, ok := call.Args[0].(*ast.BasicLit); ok {
					if strings.Contains(lit1.Value, "Content-Type") || strings.Contains(lit1.Value, "content-type") {
						// Check second arg contains "json"
						if lit2, ok := call.Args[1].(*ast.BasicLit); ok {
							if strings.Contains(strings.ToLower(lit2.Value), "json") {
								hasJSONHeader = true
								return false
							}
						}
					}
				}
			}
			return true
		})

		if hasJSONHeader {
			helpers[fn.Name.Name] = true
		}
		return true
	})

	return helpers
}

// checkHeaderOrderViolation detects if w.WriteHeader is called before setting Content-Type
func checkHeaderOrderViolation(lines []string, headerIdx int) bool {
	// Look backwards from header line to see if WriteHeader was already called
	for i := max(0, headerIdx-10); i < headerIdx; i++ {
		lowerLine := strings.ToLower(lines[i])
		if strings.Contains(lowerLine, "w.writeheader(") || strings.Contains(lowerLine, "w.write(") {
			// Found write before header - this is a violation
			return true
		}
	}
	return false
}

// containsJSONHelper checks if a line calls any AST-detected JSON helper function
func containsJSONHelper(lowerLine string, helpers map[string]bool) bool {
	for helperName := range helpers {
		if strings.Contains(lowerLine, strings.ToLower(helperName)+"(") {
			return true
		}
	}
	return false
}

// detectJSONMiddleware looks for middleware patterns that might set Content-Type globally
func detectJSONMiddleware(content string) bool {
	lower := strings.ToLower(content)

	// Common middleware patterns
	middlewarePatterns := []string{
		"func jsonmiddleware",
		"func contenttytemiddleware",
		"func setjsonheader",
		"return http.handlerfunc(func",
		".use(func",
	}

	for _, pattern := range middlewarePatterns {
		if strings.Contains(lower, pattern) {
			// Check if it sets Content-Type
			if strings.Contains(lower, "content-type") && strings.Contains(lower, "json") {
				return true
			}
		}
	}

	return false
}

// isInHandlerFunction checks if a line is within an HTTP handler function
func isInHandlerFunction(lines []string, idx int) bool {
	// Look backwards to find function declaration
	for i := idx; i >= max(0, idx-50); i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "func ") && strings.Contains(line, "http.ResponseWriter") {
			return true
		}
	}
	return false
}
