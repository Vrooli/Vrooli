package aisearch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

// searchjson_write.go is the write side of the search.json SSOT: the config-write
// contract's persistence primitive. search-hub's sweep finds a better tuning and
// calls a provider's config-write RPC; the provider persists the winner here. The
// write is the ONLY mutation of the file aisearch-go performs — the descriptor and
// tests blocks are never rewritten by code (the descriptor is registration data;
// the tests are the curated corpus). Only the `tuning` block of one provider moves.
//
// The write is validated (the new tuning must pass the factor taxonomy), atomic
// (write-temp-then-rename so a crash never leaves a half-written SSOT), and
// idempotent (an unchanged tuning is a no-op). It reports whether an INDEX-TIME
// factor changed so the caller knows to reindex.

// WriteProviderTuning persists tuning as the new `tuning` block of provider id in
// the search.json at path. It loads + validates the file, validates the incoming
// tuning against the factor taxonomy, and — unless dryRun — atomically rewrites
// the file with the new (defaults-filled) tuning.
//
// Returns:
//   - effective: the tuning now in effect (defaults-filled); on dryRun or a no-op
//     this is the resolved incoming tuning / the current tuning respectively.
//   - indexTimeChanged: whether an INDEX-TIME factor (engine / embed_model /
//     embed_task_prefix) differs from the prior tuning (the caller reindexes when
//     true).
//   - written: true only when the file was actually rewritten — false on dryRun
//     and false when the submitted tuning equals the current one (mirrors the
//     WriteConfigResponse.written contract).
//
// The provider must already exist in the file (config-write replaces a tuning
// block; it never creates a provider). An absent provider is ErrProviderNotInFile.
func WriteProviderTuning(path, id string, tuning TuningConfig, dryRun bool) (effective TuningConfig, indexTimeChanged, written bool, err error) {
	file, err := LoadSearchFile(path)
	if err != nil {
		return TuningConfig{}, false, false, err
	}

	idx := -1
	for i := range file.Providers {
		if file.Providers[i].ProviderID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return TuningConfig{}, false, false, ErrProviderNotInFile{ProviderID: id, Path: path}
	}

	effective = tuning.WithDefaults()
	if vErr := effective.Validate(); vErr != nil {
		return TuningConfig{}, false, false, fmt.Errorf("search.json: provider %q: %w", id, vErr)
	}

	current := file.Providers[idx].Tuning.WithDefaults()
	indexTimeChanged = effective.IndexTimeChanged(current)

	// A no-op (equal tuning) or a dry run never touches the file.
	if dryRun || reflect.DeepEqual(effective, current) {
		if reflect.DeepEqual(effective, current) {
			// Nothing changed: the effective tuning is the current one and no
			// reindex is implied regardless of the dryRun flag.
			return current, false, false, nil
		}
		return effective, indexTimeChanged, false, nil
	}

	file.Providers[idx].Tuning = effective
	if wErr := writeSearchFileAtomic(path, file); wErr != nil {
		return TuningConfig{}, false, false, wErr
	}
	return effective, indexTimeChanged, true, nil
}

// writeSearchFileAtomic serializes the whole search.json and replaces the file at
// path atomically (write a sibling temp file, fsync, rename). Rewriting the whole
// document (rather than splicing the tuning block) keeps the output deterministic;
// the descriptor sub-objects round-trip verbatim because they are carried as
// json.RawMessage, and the tests corpus round-trips through its typed shape. The
// only intentional change between the input and output bytes is the one rewritten
// tuning block (plus canonical indentation).
func writeSearchFileAtomic(path string, file SearchFile) error {
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal search.json: %w", err)
	}
	raw = append(raw, '\n')

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".search-*.json.tmp")
	if err != nil {
		return fmt.Errorf("create temp search.json: %w", err)
	}
	tmpName := tmp.Name()
	// Best-effort cleanup if we fail before the rename.
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp search.json: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temp search.json: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp search.json: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp search.json into place: %w", err)
	}
	return nil
}

// ErrProviderNotInFile is returned by WriteProviderTuning when the target provider
// is not present in the search.json. Config-write replaces an existing tuning
// block; it never registers a new provider.
type ErrProviderNotInFile struct {
	ProviderID string
	Path       string
}

func (e ErrProviderNotInFile) Error() string {
	return fmt.Sprintf("provider %q not found in %s", e.ProviderID, e.Path)
}
