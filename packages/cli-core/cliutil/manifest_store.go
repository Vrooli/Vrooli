package cliutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// FreshnessManifestSuffix is appended to an artifact path to locate its recorded
// freshness manifest (e.g. "api/scenario-api" -> "api/scenario-api.freshness.json").
const FreshnessManifestSuffix = ".freshness.json"

// FreshnessManifestPath returns the manifest path for a build artifact. The
// manifest lives next to the artifact so it is naturally instance/source-dir
// scoped (a shadow engagement's separate source tree gets its own artifact and
// thus its own manifest, never colliding with the live one).
func FreshnessManifestPath(artifactPath string) string {
	return artifactPath + FreshnessManifestSuffix
}

// ReadFreshnessManifest loads a recorded manifest. ok is false (with no error)
// when the manifest is absent — callers treat that as "stale once, then stamp"
// (bootstrap / self-healing). A malformed manifest returns an error so the
// caller can decide to re-stamp rather than silently trust nothing.
func ReadFreshnessManifest(path string) (manifest FreshnessManifest, ok bool, err error) {
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return FreshnessManifest{}, false, nil
		}
		return FreshnessManifest{}, false, readErr
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return FreshnessManifest{}, false, fmt.Errorf("parse freshness manifest %s: %w", path, err)
	}
	return manifest, true, nil
}

// WriteFreshnessManifest atomically writes a manifest to path via a temp file +
// rename, so a concurrent reader never observes a half-written manifest and a
// crash mid-write leaves the previous manifest intact. Pure-Go I/O only.
func WriteFreshnessManifest(path string, manifest FreshnessManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}
