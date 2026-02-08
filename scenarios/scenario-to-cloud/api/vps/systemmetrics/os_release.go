package systemmetrics

import "strings"

// ParseOSRelease extracts ID and VERSION_ID from /etc/os-release content.
func ParseOSRelease(contents string) (id, versionID string) {
	for _, line := range strings.Split(contents, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ID=") {
			id = strings.Trim(strings.TrimPrefix(line, "ID="), `"`)
		}
		if strings.HasPrefix(line, "VERSION_ID=") {
			versionID = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), `"`)
		}
	}
	return strings.ToLower(id), versionID
}
