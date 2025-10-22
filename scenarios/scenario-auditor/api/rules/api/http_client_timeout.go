package api

import (
	"regexp"
	"strings"
)

/*
Rule: HTTP Client Timeout Enforcement
Description: Ensure HTTP clients specify explicit timeouts or avoid the global default client
Reason: Prevents hanging requests that exhaust resources in production deployments
Category: api
Severity: critical
Standard: resource-management-v1
Targets: api

<test-case id="http-client-without-timeout" should-fail="true">
  <description>Custom HTTP client constructed without Timeout</description>
  <input language="go"><![CDATA[
func makeRequest(url string) (*http.Response, error) {
    client := &http.Client{}
    return client.Get(url)
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>HTTP Client Without Timeout</expected-message>
</test-case>

<test-case id="default-http-client-usage" should-fail="true">
  <description>Using the package-level http.Get without a context timeout</description>
  <input language="go"><![CDATA[
func fetchPage(url string) ([]byte, error) {
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    return io.ReadAll(resp.Body)
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>Default HTTP Client Usage</expected-message>
</test-case>

<test-case id="http-client-with-timeout" should-fail="false">
  <description>Custom HTTP client that sets a Timeout field</description>
  <input language="go"><![CDATA[
func safeRequest(url string) (*http.Response, error) {
    client := &http.Client{
        Timeout: 30 * time.Second,
    }
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }
    return client.Do(req)
}
]]></input>
</test-case>
*/

// CheckHTTPClientTimeout enforces explicit HTTP client timeouts and discourages default client usage.
func CheckHTTPClientTimeout(content []byte, filePath string) []Violation {
	lines := strings.Split(string(content), "\n")
	var violations []Violation

	httpClientPattern := regexp.MustCompile(`&http\.Client\s*{`)
	httpCallPattern := regexp.MustCompile(`\bhttp\.(Get|Post|Head|Do)\(`)

	for i, line := range lines {
		if httpClientPattern.MatchString(line) {
			if !hasHTTPClientTimeout(lines, i) {
				violations = append(violations, Violation{
					Type:           "http_client_timeout",
					Severity:       "critical",
					Title:          "HTTP Client Without Timeout",
					Description:    "HTTP Client Without Timeout",
					FilePath:       filePath,
					LineNumber:     i + 1,
					CodeSnippet:    line,
					Recommendation: "Add Timeout: 30 * time.Second to http.Client",
					Standard:       "resource-management-v1",
				})
			}
		}

		if httpCallPattern.MatchString(line) {
			if isTestFile(filePath) {
				continue
			}
			violations = append(violations, Violation{
				Type:           "http_default_client",
				Severity:       "high",
				Title:          "Default HTTP Client Usage",
				Description:    "Default HTTP Client Usage",
				FilePath:       filePath,
				LineNumber:     i + 1,
				CodeSnippet:    line,
				Recommendation: "Create custom client with timeout or use context with timeout",
				Standard:       "resource-management-v1",
			})
		}
	}

	return violations
}

func hasHTTPClientTimeout(lines []string, start int) bool {
	openBraces := 0
	for i := start; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		openBraces += strings.Count(line, "{")
		openBraces -= strings.Count(line, "}")

		if strings.Contains(trimmed, "Timeout") {
			return true
		}

		if i > start && openBraces <= 0 {
			return false
		}
	}
	return false
}
