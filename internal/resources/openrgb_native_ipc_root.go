package resources

import (
	"context"
	"path/filepath"

	"github.com/vrooli/vrooli/internal/config"
)

// Admission IPC belongs to the same ordinary operator as its durable state.
// Exact kernel peer identity and the single-use claim remain required before
// any image handoff; same-user filesystem privacy is not vendor isolation.
func openRGBNativeIPCRoot() (string, error) {
	root, err := OpenRGBNativeResourceRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "ipc"), nil
}
func prepareOpenRGBNativeIPCRoot(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	owner, err := readOpenRGBNativeOwnerContext(ctx)
	if err != nil {
		return err
	}
	root, err := openRGBNativeIPCRoot()
	if err != nil {
		return err
	}
	policy, err := owner.policyFor(root)
	if err != nil {
		return err
	}
	return config.ConfigurePrivateDirectoryAccess(root, policy)
}
