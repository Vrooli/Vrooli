// Package scenariostale detects whether a scenario's compiled binaries are
// stale relative to the scenario's Go source tree.
//
// It works via a self-maintaining sidecar file (.build-fingerprint.json) written
// next to the scenario. The sidecar remembers the last observed source
// fingerprint along with a "binary signature" (a deterministic encoding of the
// mtimes of the executable files the scenario ships). When the binary signature
// changes (i.e. a rebuild happened), the sidecar is refreshed. When the source
// fingerprint changes without the binary signature changing, the scenario is
// reported as stale so callers can warn the user.
package scenariostale

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/config"
)

const (
	scenariostaleParameterA = 32
	scenariostaleParameterB = 8
)

// Status classifies the outcome of a stale check.
type Status string

const (
	// StatusNoSources means the scenario has no Go source under api/ or cli/
	// and therefore cannot be meaningfully checked for staleness.
	StatusNoSources Status = "no_sources"
	// StatusInitialBaseline means the sidecar did not exist and has now been
	// initialized. No warning should be emitted.
	StatusInitialBaseline Status = "initial_baseline"
	// StatusRebuildDetected means the binary signature changed since the last
	// check (i.e. the scenario was rebuilt). The sidecar has been refreshed.
	StatusRebuildDetected Status = "rebuild_detected"
	// StatusFresh means the source fingerprint matches the sidecar; no action
	// required.
	StatusFresh Status = "fresh"
	// StatusStale means the source fingerprint differs from the sidecar while
	// the binary signature is unchanged: the user edited sources without
	// rebuilding.
	StatusStale Status = "stale"
)

// SidecarFile is the name of the per-scenario sidecar file.
const SidecarFile = ".build-fingerprint.json"

// SidecarVersion is the schema version of the sidecar payload.
const SidecarVersion = 1

// Result captures the outcome of a Check call.
type Result struct {
	Scenario          string
	ScenarioDir       string
	SidecarPath       string
	Status            Status
	Fingerprint       string
	StoredFingerprint string
	BinarySignature   string
	StoredSignature   string
	GoFileCount       int
	BinaryCount       int
	ChangedFiles      []string
}

// Options permit tests to inject a clock.
type Options struct {
	Clock func() time.Time
}

type sidecarPayload struct {
	Version         int               `json:"version"`
	SourceHash      string            `json:"source_fingerprint"`
	BinarySignature string            `json:"binary_signature"`
	Files           map[string]string `json:"files,omitempty"`
	UpdatedAt       string            `json:"updated_at"`
	GoFileCount     int               `json:"go_file_count"`
	BinaryCount     int               `json:"binary_count"`
}

type fileEntry struct {
	rel  string
	hash string
	size int64
}

// Check inspects the scenario directory, refreshes the sidecar if a rebuild is
// detected, and returns the classification. Errors short-circuit the check;
// callers should treat errors as best-effort and continue.
func Check(scenarioDir, scenarioName string, opts Options) (Result, error) {
	if strings.TrimSpace(scenarioDir) == "" {
		return Result{}, errors.New("scenariostale: scenario directory is required")
	}
	info, err := os.Stat(scenarioDir)
	if err != nil {
		return Result{}, fmt.Errorf("stat scenario dir: %w", err)
	}
	if !info.IsDir() {
		return Result{}, fmt.Errorf("scenario path is not a directory: %s", scenarioDir)
	}

	clock := opts.Clock
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}

	files, err := collectSourceFiles(scenarioDir)
	if err != nil {
		return Result{}, err
	}

	result := Result{
		Scenario:    scenarioName,
		ScenarioDir: scenarioDir,
		SidecarPath: filepath.Join(scenarioDir, SidecarFile),
		GoFileCount: len(files),
	}

	if len(files) == 0 {
		result.Status = StatusNoSources
		return result, nil
	}

	fileHashes := make(map[string]string, len(files))
	for _, f := range files {
		fileHashes[f.rel] = f.hash
	}
	result.Fingerprint = aggregateFingerprint(files)

	binarySig, binaryCount, err := computeBinarySignature(scenarioDir)
	if err != nil {
		return result, err
	}
	result.BinarySignature = binarySig
	result.BinaryCount = binaryCount

	stored, sidecarExists, err := readSidecar(result.SidecarPath)
	if err != nil {
		// Treat an unreadable/corrupt sidecar as "no sidecar": we'll rewrite it.
		sidecarExists = false
	}

	currentPayload := sidecarPayload{
		Version:         SidecarVersion,
		SourceHash:      result.Fingerprint,
		BinarySignature: result.BinarySignature,
		Files:           fileHashes,
		UpdatedAt:       clock().Format(time.RFC3339),
		GoFileCount:     result.GoFileCount,
		BinaryCount:     result.BinaryCount,
	}

	if !sidecarExists {
		if err := writeSidecar(result.SidecarPath, currentPayload); err != nil {
			return result, err
		}
		result.Status = StatusInitialBaseline
		return result, nil
	}

	result.StoredFingerprint = stored.SourceHash
	result.StoredSignature = stored.BinarySignature

	if stored.BinarySignature != result.BinarySignature {
		if err := writeSidecar(result.SidecarPath, currentPayload); err != nil {
			return result, err
		}
		result.Status = StatusRebuildDetected
		return result, nil
	}

	if stored.SourceHash == result.Fingerprint {
		result.Status = StatusFresh
		return result, nil
	}

	result.Status = StatusStale
	result.ChangedFiles = diffFileHashes(stored.Files, fileHashes)
	return result, nil
}

// FormatWarning returns a multi-line warning suitable for writing to stderr
// when the status is StatusStale. Returns an empty string otherwise.
func FormatWarning(r Result) string {
	if r.Status != StatusStale {
		return ""
	}
	name := strings.TrimSpace(r.Scenario)
	if name == "" {
		name = filepath.Base(r.ScenarioDir)
	}
	changed := len(r.ChangedFiles)
	var body strings.Builder
	body.WriteString(fmt.Sprintf("WARNING: scenario '%s' binary is stale\n", name))
	if changed == 1 {
		body.WriteString("  1 Go file has changed since the last build\n")
	} else if changed > 1 {
		body.WriteString(fmt.Sprintf("  %d Go files have changed since the last build\n", changed))
	} else {
		body.WriteString("  scenario source differs from the last build fingerprint\n")
	}
	if changed > 0 && changed <= 5 {
		for _, path := range r.ChangedFiles {
			body.WriteString(fmt.Sprintf("    %s\n", path))
		}
	}
	body.WriteString(fmt.Sprintf("  Rebuild the scenario (e.g. cd scenarios/%s && go build ./api/... ./cli/...) to fully validate\n", name))
	body.WriteString("  Pass --no-stale-check to silence this warning\n")
	return body.String()
}

func collectSourceFiles(scenarioDir string) ([]fileEntry, error) {
	files := make([]fileEntry, 0, scenariostaleParameterA)
	for _, sub := range []string{"api", "cli"} {
		base := filepath.Join(scenarioDir, sub)
		info, err := os.Stat(base)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if !info.IsDir() {
			continue
		}
		err = filepath.WalkDir(base, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				name := d.Name()
				if name == "node_modules" || name == ".git" || name == "testdata" || name == "vendor" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}
			entry, err := hashFile(scenarioDir, path)
			if err != nil {
				return err
			}
			files = append(files, entry)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].rel < files[j].rel })
	return files, nil
}

func hashFile(scenarioDir, path string) (fileEntry, error) {
	info, err := os.Stat(path)
	if err != nil {
		return fileEntry{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fileEntry{}, fmt.Errorf("read %s: %w", path, err)
	}
	rel, err := filepath.Rel(scenarioDir, path)
	if err != nil {
		return fileEntry{}, err
	}
	rel = filepath.ToSlash(rel)
	sum := sha256.Sum256(data)
	return fileEntry{rel: rel, hash: hex.EncodeToString(sum[:]), size: info.Size()}, nil
}

func aggregateFingerprint(files []fileEntry) string {
	h := sha256.New()
	for _, f := range files {
		fmt.Fprintf(h, "%s|%d|%s\n", f.rel, f.size, f.hash)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func computeBinarySignature(scenarioDir string) (string, int, error) {
	parts := make([]string, 0, scenariostaleParameterB)
	count := 0
	for _, sub := range []string{"api", "cli"} {
		base := filepath.Join(scenarioDir, sub)
		entries, err := os.ReadDir(base)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", 0, fmt.Errorf("read scenario binary directory %q: %w", base, err)
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			mode := info.Mode()
			if mode&tuning.PermExecuteMask == 0 {
				continue
			}
			name := info.Name()
			// Skip test binaries and shell/install scripts; only count real scenario binaries.
			if strings.HasSuffix(name, ".test") {
				continue
			}
			if strings.HasSuffix(name, ".sh") || strings.HasSuffix(name, ".ps1") {
				continue
			}
			rel := filepath.ToSlash(filepath.Join(sub, name))
			parts = append(parts, fmt.Sprintf("%s:%d", rel, info.ModTime().UTC().UnixNano()))
			count++
		}
	}
	slices.Sort(parts)
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil)), count, nil
}

func readSidecar(path string) (sidecarPayload, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return sidecarPayload{}, false, nil
		}
		return sidecarPayload{}, false, err
	}
	var payload sidecarPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return sidecarPayload{}, true, err
	}
	return payload, true, nil
}

func writeSidecar(path string, payload sidecarPayload) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return config.WriteOwnedFileAtomic(path, data, tuning.PermFile)
}

func diffFileHashes(stored, current map[string]string) []string {
	changed := make([]string, 0, scenariostaleParameterB)
	seen := make(map[string]struct{}, len(stored))
	for path, hash := range current {
		seen[path] = struct{}{}
		if prev, ok := stored[path]; !ok || prev != hash {
			changed = append(changed, path)
		}
	}
	for path := range stored {
		if _, ok := seen[path]; !ok {
			changed = append(changed, path)
		}
	}
	slices.Sort(changed)
	return changed
}
