//go:build unix

package storage

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

var currentEUID = os.Geteuid

var (
	invokingUserIDs     = hostreqkit.InvokingUserIDs
	invokingUserHomeDir = hostreqkit.InvokingUserHomeDir
)

func requireFileInfoOwnedByCurrentUser(path string, info fs.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return &Error{Kind: ErrResolve, Message: "inspect storage ownership", Details: path, Err: fmt.Errorf("filesystem did not provide POSIX ownership metadata")}
	}
	expected, _, _ := expectedStorageOwner()
	if stat.Uid != expected {
		return &Error{
			Kind:    ErrOwnership,
			Message: "refuse to create or use per-user resource storage owned by another account",
			Details: fmt.Sprintf("%s (owner uid %d, current uid %d)", path, stat.Uid, expected),
		}
	}
	return nil
}

func expectedStorageOwner() (uid uint32, gid int, invokingUser bool) {
	if invokingUID, invokingGID, ok := invokingUserIDs(); ok {
		return uint32(invokingUID), invokingGID, true
	}
	return uint32(currentEUID()), -1, false
}

// restoreCreatedStorageOwnership applies the existing sudo contract to the
// XDG resource roots. A sudo'd setup creates filesystem entries as root, but
// those entries belong to the invoking user. The operation is deliberately
// constrained to newly-created paths below that user's home; existing paths
// are never reassigned.
func restoreCreatedStorageOwnership(created []string) error {
	uid, gid, sudoed := expectedStorageOwner()
	if !sudoed || len(created) == 0 {
		return nil
	}
	home, err := invokingUserHomeDir()
	if err != nil {
		return &Error{Kind: ErrResolve, Message: "resolve invoking user home for storage ownership", Err: err}
	}
	for _, path := range created {
		if !withinStorageHome(path, home) {
			return &Error{Kind: ErrOwnership, Message: "refuse to reassign resource storage outside invoking user home", Details: path}
		}
		if err := os.Lchown(path, int(uid), gid); err != nil {
			return &Error{Kind: ErrOwnership, Message: "restore invoking user ownership for storage", Details: path, Err: err}
		}
	}
	return nil
}

func withinStorageHome(path, home string) bool {
	path = filepath.Clean(path)
	home = filepath.Clean(home)
	return path == home || strings.HasPrefix(path, home+string(filepath.Separator))
}
