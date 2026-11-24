package cliutil

import (
	"bufio"
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

// readFreshnessManifestHead reads a manifest's header fields and watch
// without its per-file entries, which dominate its size and are not needed to
// evaluate the watch. It stops reading at the file list; the writer encodes
// the watch before it.
func readFreshnessManifestHead(path string) (FreshnessManifest, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return FreshnessManifest{}, false, nil
		}
		return FreshnessManifest{}, false, err
	}
	defer file.Close()
	malformed := func(err error) (FreshnessManifest, bool, error) {
		return FreshnessManifest{}, false, fmt.Errorf("parse freshness manifest %s: %w", path, err)
	}
	decoder := json.NewDecoder(bufio.NewReaderSize(file, 64<<10))
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') {
		return malformed(fmt.Errorf("not a JSON object"))
	}
	var manifest FreshnessManifest
	fields := map[string]any{
		"version": &manifest.Version, "check_type": &manifest.CheckType, "written_at_ns": &manifest.WrittenAtNS,
		"digest": &manifest.Digest, "inputs": &manifest.Inputs, "key_inputs": &manifest.KeyInputs, "watch": &manifest.Watch,
	}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return malformed(err)
		}
		key, _ := token.(string)
		if key == "files" {
			break
		}
		target, known := fields[key]
		if !known {
			var skipped json.RawMessage
			target = &skipped
		}
		if err := decoder.Decode(target); err != nil {
			return malformed(err)
		}
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
