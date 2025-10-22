package api

import (
	"regexp"
	"strings"
)

/*
Rule: Database Rows Close
Description: Ensure database query result sets are closed
Reason: Prevents exhausting database connections by leaking result cursors
Category: api
Severity: high
Standard: database-v1
Targets: api

<test-case id="rows-not-closed" should-fail="true">
  <description>Query rows without a corresponding Close call</description>
  <input language="go"><![CDATA[
func listUsers(db *sql.DB) ([]User, error) {
    rows, err := db.Query("SELECT id FROM users")
    if err != nil {
        return nil, err
    }
    var out []User
    for rows.Next() {
        var u User
        rows.Scan(&u.ID)
        out = append(out, u)
    }
    return out, nil
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>Database Rows Not Closed</expected-message>
</test-case>

<test-case id="rows-closed" should-fail="false">
  <description>Query rows with defer Close immediately after the error check</description>
  <input language="go"><![CDATA[
func listUsersSafely(db *sql.DB) ([]User, error) {
    rows, err := db.Query("SELECT id FROM users")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var out []User
    for rows.Next() {
        var u User
        rows.Scan(&u.ID)
        out = append(out, u)
    }
    return out, rows.Err()
}
]]></input>
</test-case>
*/

// CheckDBRowsClose ensures sql.Rows results are properly closed.
func CheckDBRowsClose(content []byte, filePath string) []Violation {
	lines := strings.Split(string(content), "\n")
	var violations []Violation

	pattern := regexp.MustCompile(`(\w+)\s*,\s*err\s*:=\s*\w+\.(Query|QueryContext|Queryx|QueryContextx)\(`)
	seen := make(map[string]bool)

	for i, line := range lines {
		if strings.Contains(line, "QueryRow") {
			continue
		}

		match := pattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		rowsVar := match[1]
		if seen[rowsVar] {
			continue
		}
		seen[rowsVar] = true

		if !findWithinWindow(lines, i+1, 120, func(next string) bool {
			trimmed := strings.TrimSpace(next)
			return strings.Contains(trimmed, "defer "+rowsVar+".Close()") || strings.Contains(trimmed, rowsVar+".Close()")
		}) {
			violations = append(violations, Violation{
				Type:           "db_rows_close",
				Severity:       "high",
				Title:          "Database Rows Not Closed",
				Description:    "Database Rows Not Closed",
				FilePath:       filePath,
				LineNumber:     i + 1,
				CodeSnippet:    line,
				Recommendation: "Add defer " + rowsVar + ".Close() after error check",
				Standard:       "database-v1",
			})
		}
	}

	return violations
}
