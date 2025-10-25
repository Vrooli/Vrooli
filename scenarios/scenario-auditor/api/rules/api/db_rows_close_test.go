//go:build ruletests
// +build ruletests

package api

import "testing"

func TestDBRowsCloseDocCases(t *testing.T) {
	runDocTestsViolations(t, "db_rows_close.go", "api/main.go", CheckDBRowsClose)
}
