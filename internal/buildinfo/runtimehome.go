package buildinfo

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	repocontract "github.com/vrooli/repo-contract-go"
)

// RuntimeHomeSnapshotBase64 is populated by the governed build owner. Native
// entrypoints pin it before resolving paths; an absent/invalid snapshot is an
// explicit unsupported artifact, never permission to guess a runtime layout.
var RuntimeHomeSnapshotBase64 string

// RuntimeHomeBuildFlag carries the canonical structural authority into a
// native artifact. It reads only the build's explicit source root, never an
// ambient pinned runtime-home authority from the build process.
func RuntimeHomeBuildFlag(root string) (string, error) {
	c, err := repocontract.LoadDefault(root)
	if err != nil {
		return "", err
	}
	data, err := c.RuntimeHomeSnapshot()
	if err != nil {
		return "", err
	}
	return "-X github.com/vrooli/vrooli/internal/buildinfo.RuntimeHomeSnapshotBase64=" + base64.StdEncoding.EncodeToString(data), nil
}

func compiledRuntimeHomeSnapshot() ([]byte, error) {
	if len(RuntimeHomeSnapshotBase64) == 0 || len(RuntimeHomeSnapshotBase64) > base64.StdEncoding.EncodedLen(64<<10) {
		return nil, fmt.Errorf("compiled runtime-home authority is missing or exceeds its bound")
	}
	data, err := base64.StdEncoding.Strict().DecodeString(RuntimeHomeSnapshotBase64)
	if err != nil {
		return nil, fmt.Errorf("compiled runtime-home authority has invalid encoding")
	}
	if err := repocontract.ValidateRuntimeHomeSnapshot(data); err != nil {
		return nil, fmt.Errorf("compiled runtime-home authority is invalid: %w", err)
	}
	return data, nil
}

// RuntimeHomeSnapshotDigest is a side-effect-free artifact identity probe.
func RuntimeHomeSnapshotDigest() (string, error) {
	data, err := compiledRuntimeHomeSnapshot()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// PinCompiledRuntimeHome must precede native service initialization. It does
// not resolve a source checkout, choose a user home or create any files.
func PinCompiledRuntimeHome() error {
	data, err := compiledRuntimeHomeSnapshot()
	if err != nil {
		return err
	}
	return repocontract.PinRuntimeHomeSnapshot(data)
}

// CompiledRuntimeHomeEntryPath resolves read-only initialization fences without
// altering the process's source or runtime-home authority.
func CompiledRuntimeHomeEntryPath(home, key string) (string, error) {
	data, err := compiledRuntimeHomeSnapshot()
	if err != nil {
		return "", err
	}
	entry, err := repocontract.RuntimeHomeSnapshotEntry(data, home, key)
	if err != nil {
		return "", err
	}
	return entry.AbsPath, nil
}
