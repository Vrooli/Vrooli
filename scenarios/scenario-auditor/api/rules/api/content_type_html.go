package api

/*
Rule: HTML Content-Type Headers
Description: HTML responses must set Content-Type: text/html header
Reason: Ensures browsers render HTML correctly and prevents content sniffing attacks
Category: api
Severity: medium
Standard: api-design-v1
Targets: api

IMPLEMENTATION APPROACH:
This rule should detect HTML responses and ensure they have appropriate Content-Type headers.

Key Detection Patterns:
1. Direct HTML writes:
   - w.Write([]byte("<html>..."))
   - w.Write([]byte("<!DOCTYPE html>"))
   - Variables containing HTML tags: <html>, <head>, <body>, <div>, <!DOCTYPE>

2. Template rendering:
   - template.Execute(w, data)
   - tmpl.Execute(w, data)
   - html/template.ExecuteTemplate(w, "name", data)
   - Must check if w.Header().Set("Content-Type", "text/html") exists before Execute

3. Framework patterns (auto-set headers):
   - Gin: c.HTML(200, "template", data)
   - Echo: c.HTML(200, "<html>...")
   - Chi with render: render.HTML(w, r, data)

4. Common helper functions:
   - renderHTML(w, data)
   - respondHTML(w, html)
   - writeHTML(w, content)

Edge Cases to Handle:
- HTML fragments (partials) - may not have <!DOCTYPE> but still need header
- HTMX responses (partial HTML) - need text/html header
- SVG embedded in HTML - stays text/html
- Inline JavaScript in HTML - stays text/html
- Middleware that sets headers globally
- Template engines (text/template vs html/template)

Common False Positives to Avoid:
- HTML in comments or strings for documentation
- HTML test fixtures
- HTML escaping functions (html.EscapeString)
- Comparing strings against HTML (if contains "<html>")

Lookback Window:
- Use 30 lines like JSON rule to handle:
  * Template loading/parsing (5-10 lines)
  * Data preparation (5-10 lines)
  * Error handling (3-5 lines)
  * Content-Type setting (1 line)

Severity Considerations:
- Missing Content-Type on HTML is MEDIUM severity because:
  * Browsers often guess correctly (more forgiving than JSON)
  * But opens door to content sniffing attacks (security concern)
  * Can cause encoding issues if charset not specified

Best Practice:
- Recommend: w.Header().Set("Content-Type", "text/html; charset=utf-8")
- Include charset to prevent encoding ambiguity

TODO: Implementation required
- Parse Go files looking for HTML response patterns
- Check for Content-Type header within 30-line window
- Detect template.Execute and ensure header set
- Handle framework patterns (Gin/Echo/Chi)
- AST analysis for custom helper functions
- Test with real scenario HTML responses
*/

// CheckHTMLContentTypeHeaders validates HTML response Content-Type headers
func CheckHTMLContentTypeHeaders(content []byte, filePath string) []Violation {
	var violations []Violation

	// TODO: Implement HTML Content-Type checking
	// This is a placeholder for the actual implementation
	// Follow the approach outlined in the comments above

	_ = content
	_ = filePath

	return violations
}
