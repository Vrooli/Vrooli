package api

/*
Rule: PDF Content-Type Headers
Description: PDF file responses must set Content-Type: application/pdf header
Reason: Ensures browsers render PDFs correctly and enables inline viewing
Category: api
Severity: high
Standard: api-design-v1
Targets: api

IMPLEMENTATION APPROACH:
This rule detects PDF generation/serving endpoints and ensures they set proper
Content-Type headers. PDF handling is critical for document generation features.

Key Detection Patterns:
1. PDF libraries:
   - github.com/jung-kurt/gofpdf
   - github.com/signintech/gopdf
   - github.com/unidoc/unipdf
   - pdf.Output(w) or pdf.OutputToWriter(w)

2. PDF file serving:
   - Reading .pdf files: ioutil.ReadFile("*.pdf")
   - http.ServeFile with .pdf extension
   - PDF bytes from database/storage

3. Content-Disposition patterns:
   - Content-Disposition: inline; filename="report.pdf" (view in browser)
   - Content-Disposition: attachment; filename="invoice.pdf" (download)

4. Common PDF endpoints:
   - /invoice/download
   - /report/generate
   - /document/pdf
   - /export/pdf

Detection Strategy:
1. Look for PDF library imports (gofpdf, unipdf)
2. Detect .Output() or .OutputToWriter() methods
3. Check for .pdf in filenames
4. Look for "application/pdf" in Content-Type
5. Detect PDF magic bytes (%PDF-) in output

Edge Cases to Handle:
- PDF generation libraries (many different APIs)
- PDF stored in database ([]byte columns)
- PDF from external APIs
- PDF streaming (large documents)
- PDF/A formats (archive format)
- Encrypted PDFs

Framework Patterns:
- Gin: c.Data(200, "application/pdf", pdfBytes)
- Gin: c.File("path/to/file.pdf") - auto-sets Content-Type
- Echo: c.Blob(200, "application/pdf", pdfBytes)
- Echo: c.File("path/to/file.pdf") - auto-sets Content-Type

Content-Disposition Usage:
- inline: PDF opens in browser (common for web apps)
- attachment: PDF downloads (common for invoices/reports)
- Both need Content-Type: application/pdf

Best Practices:
- Set Content-Type before Content-Disposition
- Use: Content-Type: application/pdf
- NO charset needed (PDF is binary)
- Set Content-Length for better UX
- Consider Cache-Control headers

Lookback Window:
- 40 lines to handle:
  * PDF generation/loading (15-20 lines)
  * Data preparation (5-10 lines)
  * Error handling (5-10 lines)
  * Header setting (2-3 lines)

Common False Positives to Avoid:
- PDF parsing/reading (not serving)
- PDF library imports without usage
- Comments mentioning PDF
- Test PDF data
- PDF validation code

Severity Considerations:
- HIGH severity because:
  * Missing Content-Type breaks PDF viewing
  * Browsers download instead of displaying inline
  * Poor user experience
  * Critical for invoice/report features
  * Security: Incorrect type could cause issues

Security Considerations:
- Set X-Content-Type-Options: nosniff
- Validate PDF content before serving
- Set Content-Security-Policy for PDF display
- Consider sandboxing PDF viewer

Real-World Examples:
1. Invoice generation:
   ```go
   pdf := gofpdf.New("P", "mm", "A4", "")
   // ... generate PDF
   w.Header().Set("Content-Type", "application/pdf")
   w.Header().Set("Content-Disposition", `attachment; filename="invoice.pdf"`)
   pdf.Output(w)
   ```

2. Stored PDF serving:
   ```go
   pdfBytes, _ := db.GetDocument(id)
   w.Header().Set("Content-Type", "application/pdf")
   w.Header().Set("Content-Disposition", `inline; filename="report.pdf"`)
   w.Write(pdfBytes)
   ```

3. File-based PDF:
   ```go
   // This is OK - ServeFile auto-sets Content-Type
   http.ServeFile(w, r, "path/to/document.pdf")
   ```

Detection Challenges:
- Many PDF libraries with different APIs
- PDF can come from various sources (file, DB, generation)
- Need to distinguish PDF serving from PDF processing
- Some libraries auto-set headers, some don't

TODO: Implementation required
- Detect PDF library usage (gofpdf, unipdf, gopdf)
- Look for .Output/.OutputToWriter methods
- Check for .pdf file extensions
- Verify application/pdf Content-Type
- Handle http.ServeFile (auto-sets - OK)
- Check Content-Disposition with .pdf filename
- Detect PDF bytes from database
- AST analysis for PDF generation patterns
*/

// CheckPDFContentTypeHeaders validates PDF response Content-Type headers
func CheckPDFContentTypeHeaders(content []byte, filePath string) []Violation {
	var violations []Violation

	// TODO: Implement PDF Content-Type checking
	// Focus on detecting PDF generation libraries
	// Check for .Output() calls without Content-Type
	// Verify Content-Disposition has matching Content-Type
	// High priority - PDFs are common in business apps

	_ = content
	_ = filePath

	return violations
}
