package autosteer

import "strings"

// titleize provides legacy ASCII title casing for internal labels.
func titleize(value string) string {
	//nolint:staticcheck // Legacy ASCII title-casing is sufficient for internal labels.
	return strings.Title(value)
}
