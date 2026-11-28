//go:build !windows

package cliinstall

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
)

func snapshotJobObjectName(prefix, lockPath string) (string, error) {
	canonicalDir, err := filepath.EvalSymlinks(filepath.Dir(lockPath))
	if err != nil {
		return "", fmt.Errorf("resolve snapshot lock directory: %w", err)
	}
	canonical := filepath.Join(canonicalDir, filepath.Base(lockPath))
	digest := sha256.Sum256([]byte(canonical))
	return prefix + hex.EncodeToString(digest[:]), nil
}

func inspectSnapshotLeaseJob(string, string) (snapshotJobState, error) {
	return snapshotJobAbsent, nil
}

func inspectSnapshotRootJob(string) (snapshotJobState, error) {
	return snapshotJobAbsent, nil
}
