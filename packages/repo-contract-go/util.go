package repocontract

import (
	"path"
	"path/filepath"
	"sort"
	"strings"
)

func isAbsolutePathLike(value string) bool {
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "/") {
		return true
	}
	if len(value) >= 3 && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')) && value[1] == ':' && (value[2] == '/' || value[2] == '\\') {
		return true
	}
	return filepath.IsAbs(value)
}

func filepathToSlashTrimmed(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, `\`, `/`)
	return filepath.ToSlash(value)
}

func cleanSlashPath(value string) string {
	value = strings.TrimPrefix(value, "/")
	value = path.Clean(value)
	if value == "." {
		return ""
	}
	return value
}

func sortStrings(values []string) {
	sort.Strings(values)
}
