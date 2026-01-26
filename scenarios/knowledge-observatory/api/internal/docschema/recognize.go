package docschema

import (
	"path/filepath"
	"strings"
)

// DocTypeForFilename resolves a doc type by filename.
func DocTypeForFilename(name string) (DocType, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	if dt, ok := knownDocTypeNames()[name]; ok {
		return dt, true
	}
	return "", false
}

// DocTypeForPath resolves a doc type based on the file path.
func DocTypeForPath(path string) (DocType, bool) {
	if path == "" {
		return "", false
	}
	return DocTypeForFilename(filepath.Base(path))
}
