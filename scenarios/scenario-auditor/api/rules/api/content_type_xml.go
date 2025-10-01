package api

/*
Rule: XML Content-Type Headers
Description: XML responses must set Content-Type: application/xml header
Reason: Ensures clients parse XML correctly and prevents content type confusion
Category: api
Severity: medium
Standard: api-design-v1
Targets: api

IMPLEMENTATION APPROACH:
This rule should detect XML responses and ensure they have appropriate Content-Type headers.

Key Detection Patterns:
1. XML encoding:
   - xml.NewEncoder(w).Encode(data)
   - xml.Marshal(data) followed by w.Write()
   - xml.MarshalIndent(data) followed by w.Write()

2. Direct XML writes:
   - w.Write([]byte("<?xml version="))
   - w.Write([]byte("<root>..."))
   - Variables containing XML declaration

3. Framework patterns (auto-set headers):
   - Gin: c.XML(200, data)
   - Echo: c.XML(200, data)
   - Chi with render: render.XML(w, data)

4. Common helper functions:
   - respondXML(w, data)
   - writeXML(w, content)
   - renderXML(w, data)

Content-Type Variations:
- application/xml (standard for data)
- text/xml (older, less preferred)
- application/soap+xml (SOAP services)
- application/rss+xml (RSS feeds)
- application/atom+xml (Atom feeds)

Edge Cases to Handle:
- SVG files (should use image/svg+xml, not this rule)
- XHTML (should use application/xhtml+xml)
- Mixed content APIs (JSON by default, XML on request)
- XML fragments vs complete documents
- Namespaced XML

Common False Positives to Avoid:
- XML in comments or documentation
- XML test fixtures in strings
- XML parsing/reading (xml.Unmarshal)
- String literals that look like XML

Detection Strategy:
1. Look for xml.NewEncoder or xml.Marshal
2. Check for <?xml declaration in writes
3. Verify Content-Type header within 30-line window
4. Skip if framework methods detected (c.XML)
5. Use AST to find helper functions

Lookback Window:
- 30 lines to handle:
  * XML struct preparation (5-10 lines)
  * Namespace setup (3-5 lines)
  * Error handling (3-5 lines)
  * Header setting (1 line)

Best Practice:
- Recommend: w.Header().Set("Content-Type", "application/xml; charset=utf-8")
- Include charset for international content

Severity:
- MEDIUM because XML clients are strict about Content-Type
- Missing header often causes parsing failures
- Important for SOAP/REST APIs

TODO: Implementation required
- Detect xml.NewEncoder/xml.Marshal patterns
- Check for Content-Type: application/xml
- Handle multiple XML content-type variants
- Framework detection (Gin/Echo/Chi)
- AST-based helper detection
*/

// CheckXMLContentTypeHeaders validates XML response Content-Type headers
func CheckXMLContentTypeHeaders(content []byte, filePath string) []Violation {
	var violations []Violation

	// TODO: Implement XML Content-Type checking
	// Follow approach similar to JSON rule but for XML patterns
	// Key difference: Look for xml.NewEncoder/Marshal instead of json.*

	_ = content
	_ = filePath

	return violations
}
