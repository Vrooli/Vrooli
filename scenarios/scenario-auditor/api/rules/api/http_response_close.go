package api

import (
	"regexp"
	"strings"
)

/*
Rule: HTTP Response Body Close
Description: Ensure HTTP response bodies are closed after a successful call
Reason: Prevents socket/file descriptor leaks that exhaust connection pools
Category: api
Severity: critical
Standard: resource-management-v1
Targets: api

<test-case id="response-body-not-closed" should-fail="true">
  <description>HTTP response body retrieved without a corresponding Close</description>
  <input language="go"><![CDATA[
func fetchData(url string) ([]byte, error) {
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    return io.ReadAll(resp.Body)
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>HTTP Response Body Not Closed</expected-message>
</test-case>

<test-case id="response-body-closed" should-fail="false">
  <description>HTTP response body closed via defer immediately after error handling</description>
  <input language="go"><![CDATA[
func download(url string) ([]byte, error) {
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    return io.ReadAll(resp.Body)
}
]]></input>
</test-case>
*/

// CheckHTTPResponseClose verifies HTTP response bodies are properly closed.
func CheckHTTPResponseClose(content []byte, filePath string) []Violation {
	if isTestFile(filePath) {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	var violations []Violation

	respPattern := regexp.MustCompile(`(\w+)\s*,\s*err\s*:=\s*http\.(Get|Post|Do)\(`)
	clientDoPattern := regexp.MustCompile(`(\w+)\s*,\s*err\s*:=\s*\w+\.Do\(`)

	checked := make(map[string]bool)

	for i, line := range lines {
		var respVar string
		if m := respPattern.FindStringSubmatch(line); m != nil {
			respVar = m[1]
		} else if m := clientDoPattern.FindStringSubmatch(line); m != nil {
			respVar = m[1]
		}

		if respVar == "" || checked[respVar] {
			continue
		}
		checked[respVar] = true

		if !findWithinWindow(lines, i+1, 80, func(next string) bool {
			trimmed := strings.TrimSpace(next)
			return strings.Contains(trimmed, "defer "+respVar+".Body.Close()") || strings.Contains(trimmed, respVar+".Body.Close()")
		}) {
			violations = append(violations, Violation{
				Type:           "http_response_close",
				Severity:       "critical",
				Title:          "HTTP Response Body Not Closed",
				Description:    "HTTP Response Body Not Closed",
				FilePath:       filePath,
				LineNumber:     i + 1,
				CodeSnippet:    line,
				Recommendation: "Add defer " + respVar + ".Body.Close() after error check",
				Standard:       "resource-management-v1",
			})
		}
	}

	return violations
}
