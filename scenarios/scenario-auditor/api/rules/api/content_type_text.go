package api

/*
Rule: Plain Text Content-Type Headers
Description: Plain text responses must set Content-Type: text/plain header
Reason: Ensures proper text rendering and prevents browsers from misinterpreting content
Category: api
Severity: low
Standard: api-design-v1
Targets: api

IMPLEMENTATION APPROACH:
This rule detects plain text responses (logs, errors, plain data) and ensures
they have text/plain Content-Type headers.

Key Detection Patterns:
1. Direct string writes:
   - w.Write([]byte("plain text message"))
   - fmt.Fprintf(w, "text: %s", value)
   - io.WriteString(w, "message")

2. Common plain text use cases:
   - Health check endpoints returning "OK"
   - Status endpoints returning "healthy"
   - Prometheus metrics (/metrics endpoints)
   - Log streaming endpoints
   - Debug output endpoints

3. Framework patterns:
   - Gin: c.String(200, "message")
   - Echo: c.String(200, "message")
   - Chi: w.Write([]byte("text"))

Edge Cases to Handle:
- Single-word responses ("OK", "healthy", "success")
- Multi-line text output
- Log file streaming
- CSV data (should use text/csv instead)
- Markdown (could be text/markdown)

Distinguishing from Other Types:
- NOT JSON: No curly braces, json.* packages
- NOT HTML: No angle brackets, no template rendering
- NOT XML: No <?xml, no xml.* packages
- NOT binary: Not reading files, no Content-Disposition

Detection Heuristics:
1. Look for fmt.Fprintf(w, ...) or fmt.Fprint(w, ...)
2. Look for w.Write([]byte("...")) where content is plain text
3. Check if response is simple string without markup
4. Common patterns: "OK", "Success", "Healthy", etc.

Prometheus Metrics Special Case:
- /metrics endpoints often return text/plain; version=0.0.4
- This is correct and should not be flagged
- Detect by checking for "# HELP", "# TYPE" metrics format

Lookback Window:
- 20 lines (shorter than JSON because text responses are usually simpler)
  * Message construction (3-5 lines)
  * Error handling (2-3 lines)
  * Header setting (1 line)

Common False Positives to Avoid:
- http.Error() already sets text/plain (skip these)
- Debug logging to stdout/stderr (not HTTP responses)
- String building before JSON encoding
- Comments in code
- String literals in test files

Severity Considerations:
- LOW severity because:
  * Browsers usually render plain text OK even without header
  * Less security risk than HTML/JSON
  * Mainly affects display, not parsing
  * But still best practice for API design

Best Practices:
- Recommend: w.Header().Set("Content-Type", "text/plain; charset=utf-8")
- Include charset for international characters
- Consider text/plain vs text/* variants:
  * text/plain - generic text
  * text/csv - CSV data
  * text/html - HTML (use HTML rule)
  * text/markdown - Markdown content

Framework Handling:
- Gin c.String() auto-sets text/plain
- Echo c.String() auto-sets text/plain
- http.Error() auto-sets text/plain; charset=utf-8

TODO: Implementation required
- Detect fmt.Fprintf/Fprint patterns
- Identify simple text responses
- Skip http.Error (auto-sets)
- Skip framework string methods
- Handle Prometheus metrics format
- Check for text/plain header in window
- Distinguish from JSON/HTML/XML
*/

// CheckPlainTextContentTypeHeaders validates plain text response Content-Type headers
func CheckPlainTextContentTypeHeaders(content []byte, filePath string) []Violation {
	var violations []Violation

	// TODO: Implement plain text Content-Type checking
	// This is the trickiest because "plain text" is catch-all
	// Need to carefully distinguish from JSON/HTML/XML
	// Low priority because browsers handle missing header well
	// But important for API consistency

	_ = content
	_ = filePath

	return violations
}
