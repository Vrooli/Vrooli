package buildinfo

import (
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// readSourceRootPointer interprets the persisted source checkout locator.
func readSourceRootPointer(home string) (string, bool) {
	contents, err := os.ReadFile(repocontract.SourceRootPointerPath(home))
	if err != nil {
		return "", false
	}
	candidate := filepath.Clean(strings.TrimSpace(string(contents)))
	root, ok := findModuleRoot(candidate)
	return root, ok && root == candidate
}
