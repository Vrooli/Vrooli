// Package attemptstore owns durable, redacted attempt artifact I/O.
package attemptstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"swarm-manager/internal/jsonutil"
	"swarm-manager/internal/pathredact"
)

// Numbered is the common disk contract for attempt records.
type Numbered interface{ RoundNumber() int }

func RoundFilename(n int) string { return fmt.Sprintf("round-%03d.json", n) }

// NextRoundNumber returns the next monotonically increasing round number for
// a durable attempt directory. It intentionally reads filenames rather than a
// payload so every domain can use it before it has a decoder for its result.
func NextRoundNumber(root, relativeDir string) (int, error) {
	entries, err := os.ReadDir(filepath.Join(root, relativeDir))
	if os.IsNotExist(err) {
		return 1, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read attempt dir: %w", err)
	}
	max := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(entry.Name(), "round-%03d.json", &n); err == nil && n > max {
			max = n
		}
	}
	return max + 1, nil
}

// LoadRounds loads every numbered attempt from a relative artifact directory.
// A malformed/truncated file is recovered when possible and otherwise skipped,
// matching the historical review reader's resilient behavior.
func LoadRounds[T Numbered](root, relativeDir string, decode func([]byte) (T, error)) ([]T, error) {
	dir := filepath.Join(root, relativeDir)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read attempt dir: %w", err)
	}
	out := make([]T, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "round-") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		value, decodeErr := decode(data)
		if decodeErr != nil {
			repaired := jsonutil.RepairTruncatedJSON(data)
			if repaired == nil {
				continue
			}
			value, decodeErr = decode(repaired)
			if decodeErr != nil {
				continue
			}
		}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RoundNumber() < out[j].RoundNumber() })
	return out, nil
}

func LoadRound[T any](root, relativeDir string, round int, decode func([]byte) (T, error)) (*T, error) {
	data, err := os.ReadFile(filepath.Join(root, relativeDir, RoundFilename(round)))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read attempt %d: %w", round, err)
	}
	value, err := decode(data)
	if err != nil {
		repaired := jsonutil.RepairTruncatedJSON(data)
		if repaired == nil {
			return nil, fmt.Errorf("decode attempt %d: %w", round, err)
		}
		value, err = decode(repaired)
		if err != nil {
			return nil, fmt.Errorf("decode repaired attempt %d: %w", round, err)
		}
	}
	return &value, nil
}

func SaveRound[T Numbered](root, relativeDir string, value T) error {
	dir := filepath.Join(root, relativeDir)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create attempt dir: %w", err)
	}
	path := filepath.Join(dir, RoundFilename(value.RoundNumber()))
	stored := any(value)
	if redacted, changed, err := pathredact.NewForArtifactPath(path).RedactJSONValue(value); err != nil {
		return fmt.Errorf("redact attempt %d: %w", value.RoundNumber(), err)
	} else if changed {
		stored = redacted
	}
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal attempt %d: %w", value.RoundNumber(), err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write attempt %d: %w", value.RoundNumber(), err)
	}
	return nil
}

func EnsureCapturesDir(root, relativeDir string) (string, error) {
	dir := filepath.Join(root, relativeDir, "captures")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}
	return dir, nil
}

func SaveCapture(root, relativeDir, filename string, data []byte) (string, error) {
	clean := filepath.Clean(filename)
	if clean == "." || strings.Contains(clean, "..") || filepath.IsAbs(clean) {
		return "", fmt.Errorf("invalid capture filename: %s", filename)
	}
	dir, err := EnsureCapturesDir(root, relativeDir)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, clean)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return "", fmt.Errorf("create capture parent: %w", err)
	}
	if redacted, changed := pathredact.NewForArtifactPath(path).RedactBytes(path, data); changed {
		data = redacted
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return filepath.Join("captures", clean), nil
}

func LoadCapture(root, relativeDir, relativePath string) ([]byte, error) {
	clean := filepath.Clean(relativePath)
	if clean == "." || strings.Contains(clean, "..") || filepath.IsAbs(clean) {
		return nil, fmt.Errorf("invalid capture path: %s", relativePath)
	}
	return os.ReadFile(filepath.Join(root, relativeDir, clean))
}
