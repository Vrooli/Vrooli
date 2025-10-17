package api

/*
Rule: Binary Content-Type Headers
Description: Binary file downloads must set appropriate Content-Type headers
Reason: Ensures browsers handle downloads correctly and prevents security issues
Category: api
Severity: high
Standard: api-design-v1
Targets: api

IMPLEMENTATION APPROACH:
This rule detects file download endpoints and ensures they set proper Content-Type headers
based on file type. This is CRITICAL for security and user experience.

Key Detection Patterns:
1. File serving:
   - http.ServeFile(w, r, filepath)
   - http.ServeContent(w, r, name, modtime, content)
   - io.Copy(w, file)
   - w.Write(fileBytes) where fileBytes comes from ioutil.ReadFile

2. Content-Disposition headers (indicates download):
   - w.Header().Set("Content-Disposition", "attachment")
   - w.Header().Set("Content-Disposition", "inline")

3. Common file operations:
   - os.Open(filepath) followed by io.Copy(w, file)
   - ioutil.ReadFile(path) followed by w.Write(bytes)
   - base64.Decode() followed by w.Write() (uploaded files)

Content-Type Categories:
1. Images:
   - image/jpeg (.jpg, .jpeg)
   - image/png (.png)
   - image/gif (.gif)
   - image/webp (.webp)
   - image/svg+xml (.svg)

2. Documents:
   - application/pdf (.pdf)
   - application/msword (.doc)
   - application/vnd.openxmlformats-officedocument (.docx, .xlsx, .pptx)

3. Archives:
   - application/zip (.zip)
   - application/x-tar (.tar)
   - application/gzip (.gz)

4. Media:
   - audio/mpeg (.mp3)
   - video/mp4 (.mp4)
   - audio/wav (.wav)

5. Generic:
   - application/octet-stream (unknown binary)

Edge Cases to Handle:
- http.ServeFile sets Content-Type automatically (skip these)
- mime.TypeByExtension() usage (indicates proper handling)
- Content-Type determined from database/metadata
- Streaming large files (may not see entire file in buffer)
- Range requests (partial content)

Common False Positives to Avoid:
- Internal file operations (not HTTP responses)
- File uploads (reading files, not serving them)
- Temporary file creation
- File system operations without HTTP context

Detection Strategy:
1. Look for Content-Disposition header (strong signal of file download)
2. Check if followed by Content-Type header
3. Look for http.ServeFile/ServeContent (auto-sets headers - OK)
4. Detect file reading followed by w.Write without Content-Type
5. Check for mime.TypeByExtension usage

Lookback Window:
- 40 lines to handle:
  * File path determination (5-10 lines)
  * File reading/opening (5-10 lines)
  * Metadata lookup (5-10 lines)
  * Error handling (5-10 lines)
  * Header setting (2-3 lines)

Severity Considerations:
- HIGH severity because:
  * Missing Content-Type causes downloads to open incorrectly
  * Security risk: Browser may execute malicious content
  * Poor UX: Files save with wrong extension
  * MIME sniffing attacks possible

Best Practices:
- Use http.DetectContentType() for unknown files
- Set Content-Type before Content-Disposition
- Include charset for text-based formats
- Use application/octet-stream as safe fallback
- Set X-Content-Type-Options: nosniff header

Special Cases:
- CSV files: text/csv or application/csv (both acceptable)
- JSON downloads: application/json with Content-Disposition
- Text files: text/plain; charset=utf-8

Framework Patterns:
- Gin: c.File(filepath) - auto-sets Content-Type
- Echo: c.File(filepath) - auto-sets Content-Type
- Chi: Use http.ServeFile

TODO: Implementation required
- Detect file download patterns
- Check Content-Disposition presence
- Verify Content-Type set before download
- Handle http.ServeFile (skip - auto-sets)
- Detect mime.TypeByExtension usage (OK)
- Check for DetectContentType usage (OK)
- AST analysis for file handling patterns
*/

// CheckBinaryContentTypeHeaders validates binary file download Content-Type headers
func CheckBinaryContentTypeHeaders(content []byte, filePath string) []Violation {
	var violations []Violation

	// TODO: Implement binary Content-Type checking
	// This is more complex than JSON/XML because file types vary
	// Focus on detecting download patterns (Content-Disposition) without Content-Type
	// http.ServeFile is OK (auto-sets headers)
	// Raw w.Write(fileBytes) after ReadFile needs Content-Type

	_ = content
	_ = filePath

	return violations
}
