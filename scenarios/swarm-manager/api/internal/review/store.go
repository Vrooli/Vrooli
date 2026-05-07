package review

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

// LoadRounds reads all review/round-*.json files from the item directory,
// sorted by round number ascending.
func LoadRounds(itemDir string) ([]Round, error) {
	reviewDir := filepath.Join(itemDir, "review")
	entries, err := os.ReadDir(reviewDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read review dir: %w", err)
	}

	var rounds []Round
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "round-") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(reviewDir, entry.Name()))
		if err != nil {
			continue
		}
		var round Round
		if err := json.Unmarshal(data, &round); err != nil {
			if repaired := jsonutil.RepairTruncatedJSON(data); repaired != nil {
				if json.Unmarshal(repaired, &round) == nil {
					rounds = append(rounds, normalizeRound(round))
				}
			}
			continue
		}
		rounds = append(rounds, normalizeRound(round))
	}

	sort.Slice(rounds, func(i, j int) bool {
		return rounds[i].RoundNum < rounds[j].RoundNum
	})
	return rounds, nil
}

// LoadLatestRound returns the most recent review round and total round count.
// Returns nil round and 0 count if no rounds exist.
func LoadLatestRound(itemDir string) (*Round, int, error) {
	rounds, err := LoadRounds(itemDir)
	if err != nil {
		return nil, 0, err
	}
	if len(rounds) == 0 {
		return nil, 0, nil
	}
	return &rounds[len(rounds)-1], len(rounds), nil
}

// LoadRound loads a specific review round by number.
// Returns nil if the round does not exist.
func LoadRound(itemDir string, roundNum int) (*Round, error) {
	reviewDir := filepath.Join(itemDir, "review")
	data, err := os.ReadFile(filepath.Join(reviewDir, RoundFilename(roundNum)))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read round %d: %w", roundNum, err)
	}
	var round Round
	if err := json.Unmarshal(data, &round); err != nil {
		return nil, fmt.Errorf("unmarshal round %d: %w", roundNum, err)
	}
	round = normalizeRound(round)
	return &round, nil
}

// SaveRound writes a review round to disk. Creates the review/ directory
// if it does not exist.
func SaveRound(itemDir string, round Round) error {
	reviewDir := filepath.Join(itemDir, "review")
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		return fmt.Errorf("create review dir: %w", err)
	}
	value := any(round)
	path := filepath.Join(reviewDir, RoundFilename(round.RoundNum))
	if redacted, changed, redactErr := pathredact.NewForArtifactPath(path).RedactJSONValue(round); redactErr != nil {
		return fmt.Errorf("redact round %d: %w", round.RoundNum, redactErr)
	} else if changed {
		value = redacted
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal round %d: %w", round.RoundNum, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write round %d: %w", round.RoundNum, err)
	}
	return nil
}

// NextRoundNumber returns the next sequential round number for a new review round.
func NextRoundNumber(itemDir string) (int, error) {
	rounds, err := LoadRounds(itemDir)
	if err != nil {
		return 0, err
	}
	if len(rounds) == 0 {
		return 1, nil
	}
	return rounds[len(rounds)-1].RoundNum + 1, nil
}

// RoundFilename returns the standard zero-padded filename for a round number.
func RoundFilename(n int) string {
	return fmt.Sprintf("round-%03d.json", n)
}

// EnsureCapturesDir creates the review/captures/ directory if it does not exist
// and returns its path.
func EnsureCapturesDir(itemDir string) (string, error) {
	capturesDir := filepath.Join(itemDir, "review", "captures")
	if err := os.MkdirAll(capturesDir, 0o755); err != nil {
		return "", fmt.Errorf("create captures dir: %w", err)
	}
	return capturesDir, nil
}

// SaveCapture writes binary data to review/captures/ and returns the relative
// path from the item directory (e.g., "captures/dashboard-after.png").
func SaveCapture(itemDir, filename string, data []byte) (string, error) {
	capturesDir, err := EnsureCapturesDir(itemDir)
	if err != nil {
		return "", err
	}
	// Reject path traversal attempts.
	clean := filepath.Clean(filename)
	if strings.Contains(clean, "..") || filepath.IsAbs(clean) {
		return "", fmt.Errorf("invalid capture filename: %s", filename)
	}
	fullPath := filepath.Join(capturesDir, clean)
	if redacted, changed := pathredact.NewForArtifactPath(fullPath).RedactBytes(fullPath, data); changed {
		data = redacted
	}
	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return "", fmt.Errorf("write capture %s: %w", filename, err)
	}
	return filepath.Join("captures", clean), nil
}

// LoadCapture reads a capture file from the review/captures/ directory.
// The relativePath should be relative to review/ (e.g., "captures/screenshot.png").
func LoadCapture(itemDir, relativePath string) ([]byte, error) {
	clean := filepath.Clean(relativePath)
	if strings.Contains(clean, "..") || filepath.IsAbs(clean) {
		return nil, fmt.Errorf("invalid capture path: %s", relativePath)
	}
	fullPath := filepath.Join(itemDir, "review", clean)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("read capture %s: %w", relativePath, err)
	}
	return data, nil
}
