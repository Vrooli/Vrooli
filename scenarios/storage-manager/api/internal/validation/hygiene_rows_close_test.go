package validation

import "testing"

// Parity tests for hygiene-rows-close vs the original scenario-auditor rule
// db_rows_close (scenarios/scenario-auditor/api/rules/api/db_rows_close.go,
// CheckDBRowsClose). Each case is one of that rule's <test-case> blocks: the
// SAME bare-function source fragment (the rule parses these via a `package
// main\n` wrapper, which hygiene-rows-close replicates) under the default
// doc-runner path "api/main.go", asserting the SAME should-fail verdict.
func TestHygieneRowsClose_Parity(t *testing.T) {
	a := hygieneRowsClose{}
	const path = "api/main.go"
	cases := []struct {
		name       string
		src        string
		shouldFail bool
	}{
		{"rows-not-closed", `
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
`, true},
		{"rows-closed", `
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
`, false},
		{"rows-returned", `
func listUsersRaw(db *sql.DB) (*sql.Rows, error) {
    rows, err := db.Query("SELECT id FROM users")
    if err != nil {
        return nil, err
    }
    return rows, nil
}
`, false},
		{"rows-blank-identifier", `
func countUsers(db *sql.DB) error {
    _, err := db.Query("SELECT id FROM users")
    return err
}
`, false},
		{"rows-helper-close", `
func listUsersWithHelper(db *sql.DB) ([]User, error) {
    rows, err := db.Query("SELECT id FROM users")
    if err != nil {
        return nil, err
    }
    defer closeRows(rows)
    return consume(rows)
}

func closeRows(rows *sql.Rows) {
    rows.Close()
}

func consume(rows *sql.Rows) ([]User, error) {
    return nil, nil
}
`, false},
		{"rows-helper-consume", `
func listUsersWithConsumer(db *sql.DB) ([]User, error) {
    rows, err := db.Query("SELECT id FROM users")
    if err != nil {
        return nil, err
    }
    defer drainRows(rows)
    return consume(rows)
}

func drainRows(rows *sql.Rows) {
    rows.Close()
}

func consume(rows *sql.Rows) ([]User, error) {
    return nil, nil
}
`, false},
		{"rows-else-branch-close", `
func listUsersConditional(db *sql.DB) ([]User, error) {
    rows, err := db.Query("SELECT id FROM users")
    if err != nil {
        return nil, err
    } else {
        defer rows.Close()
        for rows.Next() {
            var id int
            rows.Scan(&id)
        }
        return nil, rows.Err()
    }
}
`, false},
		{"rows-if-cleanup", `
func listUsersIfSafe(db *sql.DB) error {
    rows, err := db.Query("SELECT id FROM users")
    if err == nil {
        defer rows.Close()
        for rows.Next() {
            var id int
            rows.Scan(&id)
        }
        return rows.Err()
    }
    return err
}
`, false},
		{"rows-branch-assignment", `
func branchClose(db *sql.DB, stats bool) error {
    var rows *sql.Rows
    var err error

    if stats {
        rows, err = db.Query("SELECT 1")
    } else {
        rows, err = db.Query("SELECT 2")
    }
    if err != nil {
        return err
    }
    defer rows.Close()
    return nil
}
`, false},
		{"rows-conditional-defer", `
func conditionalClose(db *sql.DB, enabled bool) error {
    rows, err := db.Query("SELECT 1")
    if err != nil {
        return err
    }
    if enabled {
        defer rows.Close()
    }
    return nil
}
`, true},
		{"rows-guarded-and", `
func guardedClose(db *sql.DB) (map[string]int, error) {
    rows, err := db.Query("SELECT 1")
    counts := map[string]int{}
    if err == nil && rows != nil {
        defer rows.Close()
        for rows.Next() {
            var key string
            var value int
            if err := rows.Scan(&key, &value); err == nil {
                counts[key] = value
            }
        }
    }
    return counts, err
}
`, false},
		{"rows-non-row-name-leak", `
func leakedCursor(db *sql.DB) error {
    cursor, err := db.Query("SELECT 1")
    if err != nil {
        return err
    }
    return cursor.Err()
}
`, true},
		{"rows-non-row-name-closed", `
func closedCursor(db *sql.DB) error {
    cursor, err := db.Query("SELECT 1")
    if err != nil {
        return err
    }
    defer cursor.Close()
    return cursor.Err()
}
`, false},
		{"rows-helper-body-scan", `
func withHelper(db *sql.DB) error {
    rows, err := db.Query("SELECT 1")
    if err != nil {
        return err
    }
    defer handleRows(rows)
    return nil
}

func handleRows(r *sql.Rows) {
    if r != nil {
        r.Close()
    }
}
`, false},
		{"rows-defer-func-param", `
func literalDefer(db *sql.DB) error {
    rows, err := db.Query("SELECT 1")
    if err != nil {
        return err
    }
    defer func(r *sql.Rows) {
        r.Close()
    }(rows)
    return nil
}
`, false},
		{"rows-helper-method-close", `
type closer struct{}

func (c *closer) handle(rows *sql.Rows) {
    if rows != nil {
        rows.Close()
    }
}

func useMethodHelper(db *sql.DB, c *closer) error {
    rows, err := db.Query("SELECT 1")
    if err != nil {
        return err
    }
    defer c.handle(rows)
    return nil
}
`, false},
		{"rows-queryx-context", `
func leakQueryx(db interface{ QueryxContext(any, string, ...any) (*sql.Rows, error) }) error {
    rows, err := db.QueryxContext(nil, "SELECT 1")
    if err != nil {
        return err
    }
    return rows.Err()
}
`, true},
		{"rows-struct-selector", `
func structLeak(proc *Processor) error {
    rows, err := proc.db.Query("SELECT id FROM users")
    if err != nil {
        return err
    }
    return nil
}
`, true},
		{"rows-multiline-call", `
func multilineLeak(db *sql.DB) error {
    rows, err := db.
        Query("SELECT id FROM users")
    if err != nil {
        return err
    }
    return nil
}
`, true},
		{"rows-comment", `
func commentOnly() {
    // rows, err := db.Query("SELECT 1")
}
`, false},
		{"rows-conditional-return-leak", `
func conditionalLeak(db *sql.DB, shortcut bool) error {
    rows, err := db.Query("SELECT 1")
    if err != nil {
        return err
    }
    if shortcut {
        rows.Close()
        return nil
    }
    return rows.Err()
}
`, true},
		{"rows-defer-nonclosing", `
func leakWithLogger(db *sql.DB) error {
    rows, err := db.Query("SELECT id FROM users")
    if err != nil {
        return err
    }
    defer logRows(rows)
    return nil
}

func logRows(rows *sql.Rows) {
    _ = rows
}
`, true},
		{"http-request-query", `
func inspectQuery(req *http.Request) {
    q := req.URL.Query()
    _ = q.Get("id")
}
`, false},
		{"network-client-query-payload", `
func ask(client *modelClient) error {
    payload, err := client.Query("prompt")
    if err != nil {
        return err
    }
    _ = payload
    return nil
}
`, false},
		{"rows-nested-scope-leak", `
func usersNested(db *sql.DB, include bool) error {
    rows, err := db.Query("SELECT id FROM users")
    if err != nil {
        return err
    }
    defer rows.Close()

    if include {
        rows, err := db.Query("SELECT id FROM admins")
        if err != nil {
            return err
        }
        for rows.Next() {
            var id int
            rows.Scan(&id)
        }
    }
    return nil
}
`, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := a.analyzeSource(tc.src, path)
			if tc.shouldFail && len(got) == 0 {
				t.Fatalf("expected DB_ROWS_NOT_CLOSED finding, got none")
			}
			if !tc.shouldFail && len(got) != 0 {
				t.Fatalf("expected no finding, got %+v", got)
			}
			for _, f := range got {
				if f.Code != "DB_ROWS_NOT_CLOSED" {
					t.Fatalf("unexpected code %q", f.Code)
				}
				if f.Severity != SeverityError {
					t.Fatalf("severity = %v, want ERROR", f.Severity)
				}
			}
		})
	}
}
