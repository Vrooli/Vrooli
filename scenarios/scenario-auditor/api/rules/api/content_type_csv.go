package api

/*
Rule: CSV Content-Type Headers
Description: CSV file responses must set Content-Type: text/csv header
Reason: Ensures spreadsheet applications and browsers handle CSV files correctly
Category: api
Severity: medium
Standard: api-design-v1
Targets: api

IMPLEMENTATION APPROACH:
This rule detects CSV export endpoints and ensures they set proper Content-Type headers.
CSV is commonly used for data exports, reports, and integrations.

Key Detection Patterns:
1. CSV encoding libraries:
   - encoding/csv: csv.NewWriter(w)
   - writer.Write() with string slices
   - writer.WriteAll() with 2D string slices
   - writer.Flush()

2. Manual CSV construction:
   - strings.Join(values, ",") followed by w.Write
   - fmt.Fprintf(w, "%s,%s,%s\\n", ...)
   - Building CSV rows in loops

3. Content-Disposition for downloads:
   - Content-Disposition: attachment; filename="export.csv"
   - Content-Disposition: attachment; filename="report.csv"

4. Common CSV endpoint patterns:
   - /export endpoint
   - /download/csv endpoint
   - Query param: ?format=csv
   - Accept header: text/csv

Edge Cases to Handle:
- CSV with custom delimiters (tab-separated, pipe-separated)
- CSV with headers vs no headers
- Large CSV streaming (chunked transfer)
- CSV with special characters requiring escaping
- RFC 4180 compliance considerations

Framework Patterns:
- Gin: c.CSV(200, data) - may auto-set header (check framework version)
- Echo: Usually manual CSV writing
- Chi: Manual CSV writing with encoding/csv

Detection Strategy:
1. Look for csv.NewWriter(w) or csv.Writer
2. Check for .csv in Content-Disposition filename
3. Detect comma-separated formatting patterns
4. Look for Content-Type: text/csv or application/csv
5. Check query params for format=csv

Content-Type Variations:
- text/csv (RFC 4180 standard, preferred)
- application/csv (older, less common)
- text/comma-separated-values (deprecated)

Best Practices:
- Use: Content-Type: text/csv; charset=utf-8
- Set Content-Disposition with .csv filename
- Include charset for international characters
- Use UTF-8 BOM for Excel compatibility (optional)

Lookback Window:
- 30 lines to handle:
  * Data query/preparation (10-15 lines)
  * CSV writer setup (2-3 lines)
  * Header row writing (2-3 lines)
  * Error handling (3-5 lines)
  * Content-Type setting (1 line)

Common False Positives to Avoid:
- CSV parsing/reading (not serving)
- CSV test data in strings
- Comments mentioning CSV
- CSV configuration/settings code
- Internal CSV processing (not HTTP)

Severity Considerations:
- MEDIUM severity because:
  * Missing Content-Type causes download issues
  * Excel may not open CSV properly
  * Browser may display as text instead of downloading
  * Important for data export features
  * Less critical than binary (usually renders OK)

Special Handling:
- TSV files (tab-separated): text/tab-separated-values
- Pipe-separated: text/plain with documentation
- Custom delimiters: text/plain with delimiter info

Real-World Examples to Check:
1. Financial data export:
   ```go
   w.Header().Set("Content-Type", "text/csv")
   w.Header().Set("Content-Disposition", `attachment; filename="transactions.csv"`)
   writer := csv.NewWriter(w)
   writer.Write([]string{"Date", "Amount", "Description"})
   ```

2. Report generation:
   ```go
   w.Header().Set("Content-Type", "text/csv; charset=utf-8")
   fmt.Fprintf(w, "Name,Email,Status\\n")
   for _, user := range users {
       fmt.Fprintf(w, "%s,%s,%s\\n", user.Name, user.Email, user.Status)
   }
   ```

TODO: Implementation required
- Detect csv.NewWriter(w) patterns
- Check for Content-Type: text/csv
- Detect manual CSV construction (comma-separated lines)
- Look for .csv in Content-Disposition
- Handle format=csv query parameters
- Check for proper charset inclusion
- Distinguish from plain text responses
*/

// CheckCSVContentTypeHeaders validates CSV export Content-Type headers
func CheckCSVContentTypeHeaders(content []byte, filePath string) []Violation {
	var violations []Violation

	// TODO: Implement CSV Content-Type checking
	// Look for csv.NewWriter or manual comma-separated formatting
	// Check Content-Disposition for .csv filename
	// Verify text/csv header present
	// This is important for data export features

	_ = content
	_ = filePath

	return violations
}
