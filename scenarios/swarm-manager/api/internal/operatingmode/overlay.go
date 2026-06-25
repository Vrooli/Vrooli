package operatingmode

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// DOC: docs/internal/SEAMS.md
// Overlay seam — keeps the in-code mode registry pure while allowing
// operators to override user-visible fields (label, description) without
// recompiling. Persisted to disk so edits survive API restarts.

// Override holds the user-editable subset of a mode definition. Pointer fields
// distinguish "absent" (use registry default) from "present" (apply this
// value).
type Override struct {
	Label       *string `json:"label,omitempty"`
	Description *string `json:"description,omitempty"`
}

// HasChanges reports whether the override carries any user-visible fields.
func (o Override) HasChanges() bool {
	return o.Label != nil || o.Description != nil
}

// OverlayStore persists per-mode user-editable overrides to a single JSON
// file. The zero value is unusable — construct via NewOverlayStore.
type OverlayStore struct {
	path string
	mu   sync.Mutex
}

// NewOverlayStore returns an OverlayStore backed by the given absolute file
// path. Callers are responsible for choosing the path (typically
// <scenarioRoot>/.vrooli/operating-modes/overrides.json). An empty path is
// allowed and disables persistence — Load returns an empty map and Save
// returns an error.
func NewOverlayStore(path string) *OverlayStore {
	return &OverlayStore{path: strings.TrimSpace(path)}
}

// Load returns the current set of overrides keyed by mode ID. A missing file
// is not an error and yields an empty map. Malformed JSON is also reported as
// an empty map plus a wrapped error so callers can decide whether to fail
// open.
func (s *OverlayStore) Load() (map[Mode]Override, error) {
	if s == nil || s.path == "" {
		return map[Mode]Override{}, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *OverlayStore) loadLocked() (map[Mode]Override, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[Mode]Override{}, nil
		}
		return map[Mode]Override{}, fmt.Errorf("operating-mode overlay: read %s: %w", s.path, err)
	}
	if len(data) == 0 {
		return map[Mode]Override{}, nil
	}
	raw := map[string]Override{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return map[Mode]Override{}, fmt.Errorf("operating-mode overlay: parse %s: %w", s.path, err)
	}
	out := make(map[Mode]Override, len(raw))
	for key, override := range raw {
		mode := NormalizeMode(key)
		if !ValidateMode(string(mode)) {
			continue
		}
		out[mode] = override
	}
	return out, nil
}

// Save merges the override for a single mode into the on-disk overlay. A nil
// or zero-value override clears any existing overlay row for the mode (i.e.
// restores the registry default). Writes are atomic via tmp-file + rename.
func (s *OverlayStore) Save(mode Mode, override Override) error {
	if s == nil || s.path == "" {
		return errors.New("operating-mode overlay: store path is not configured")
	}
	if !ValidateMode(string(mode)) {
		return fmt.Errorf("operating-mode overlay: unknown mode %q", mode)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	if override.HasChanges() {
		current[NormalizeMode(string(mode))] = override
	} else {
		delete(current, NormalizeMode(string(mode)))
	}
	return s.writeLocked(current)
}

func (s *OverlayStore) writeLocked(overrides map[Mode]Override) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o750); err != nil {
		return fmt.Errorf("operating-mode overlay: mkdir: %w", err)
	}
	keys := make([]string, 0, len(overrides))
	for mode := range overrides {
		keys = append(keys, string(mode))
	}
	sort.Strings(keys)
	encoded := make(map[string]Override, len(overrides))
	for _, key := range keys {
		encoded[key] = overrides[Mode(key)]
	}
	data, err := json.MarshalIndent(encoded, "", "  ")
	if err != nil {
		return fmt.Errorf("operating-mode overlay: encode: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".overrides-*.json")
	if err != nil {
		return fmt.Errorf("operating-mode overlay: tmp file: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		closeAndRemoveTemp(tmp, tmpName)
		return fmt.Errorf("operating-mode overlay: write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		removeTemp(tmpName)
		return fmt.Errorf("operating-mode overlay: close: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		removeTemp(tmpName)
		return fmt.Errorf("operating-mode overlay: rename: %w", err)
	}
	return nil
}

// applyOverlay merges any persisted overlay onto the registry definition. The
// registry remains canonical — overlays only adjust user-visible fields.
func applyOverlay(def Definition, override Override) Definition {
	if override.Label != nil {
		if trimmed := strings.TrimSpace(*override.Label); trimmed != "" {
			def.Label = trimmed
		}
	}
	if override.Description != nil {
		def.Description = strings.TrimSpace(*override.Description)
	}
	return def
}

func closeAndRemoveTemp(f *os.File, name string) {
	if closeErr := f.Close(); closeErr != nil {
		slog.Debug("operatingmode: close temp overlay failed", "err", closeErr)
	}
	removeTemp(name)
}

func removeTemp(name string) {
	if rmErr := os.Remove(name); rmErr != nil && !os.IsNotExist(rmErr) {
		slog.Debug("operatingmode: remove temp overlay failed", "err", rmErr, "path", name)
	}
}
