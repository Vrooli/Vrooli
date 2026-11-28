// Package dbvalue contains SQLite value conversions shared by storage domains.
package dbvalue

// BoolInt converts a boolean to SQLite's integer representation.
func BoolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
