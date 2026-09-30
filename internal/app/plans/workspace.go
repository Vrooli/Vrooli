package plans

import (
	"fmt"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

func (s Service) workspaceScope(workspace string) (WorkspaceScope, error) {
	workspace = strings.TrimSpace(workspace)
	root := workspace
	if root == "" {
		root = strings.TrimSpace(s.Root)
	}
	if root == "" {
		return WorkspaceScope{}, nil
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return WorkspaceScope{}, fmt.Errorf("resolve workspace %q: %w", root, err)
	}
	clean := filepath.Clean(abs)
	if _, err := repocontract.LoadDefault(clean); err != nil {
		return WorkspaceScope{}, fmt.Errorf("invalid workspace root %q: %w", clean, err)
	}
	return WorkspaceScope{Root: clean}, nil
}
