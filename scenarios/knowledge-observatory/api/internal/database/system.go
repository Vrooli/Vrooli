// Package database owns cross-cutting database objects for this scenario.
//
// It deliberately owns no domain tables. See system.sql.
package database

import (
	_ "embed"
	"strings"
)

//go:embed system.sql
var systemSQL string

// SystemSchema returns the cross-cutting DDL applied before any domain schema.
//
// system.sql currently holds only comments explaining why it is empty. Comments
// are not statements, so this returns "" in that case and EnsureSchemas skips
// the provider entirely rather than handing a statement-free string to the
// driver.
func SystemSchema() string {
	if !containsStatement(systemSQL) {
		return ""
	}
	return systemSQL
}

// containsStatement reports whether s holds anything other than blank lines and
// `--` comments.
func containsStatement(s string) bool {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		return true
	}
	return false
}
