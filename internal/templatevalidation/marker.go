package templatevalidation

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/config"
)

const (
	MarkerVersion = "1.0.0"
	MarkerRelPath = ".vrooli/template-validation-run.json"
)

type RunMarker struct {
	Version             string    `json:"version"`
	RunID               string    `json:"runId"`
	RepoRoot            string    `json:"repoRoot"`
	Template            string    `json:"template"`
	ScenarioID          string    `json:"scenarioId"`
	ScenarioPath        string    `json:"scenarioPath"`
	TempRoot            string    `json:"tempRoot"`
	CreatedAt           time.Time `json:"createdAt"`
	Retained            bool      `json:"retained"`
	CreatorPID          int       `json:"creatorPid"`
	Completed           bool      `json:"completed"`
	CleanupStatus       string    `json:"cleanupStatus,omitempty"`
	RelocationArtifacts []string  `json:"relocationArtifacts,omitempty"`
}

type NewRunMarkerInput struct {
	RepoRoot     string
	Template     string
	ScenarioID   string
	ScenarioPath string
	TempRoot     string
	Retained     bool
	Now          time.Time
}

func NewRunMarker(input NewRunMarkerInput) (RunMarker, error) {
	now := input.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	runID, err := newRunID(input.ScenarioID, now)
	if err != nil {
		return RunMarker{}, err
	}
	return RunMarker{
		Version:      MarkerVersion,
		RunID:        runID,
		RepoRoot:     filepath.Clean(input.RepoRoot),
		Template:     strings.TrimSpace(input.Template),
		ScenarioID:   strings.TrimSpace(input.ScenarioID),
		ScenarioPath: filepath.Clean(input.ScenarioPath),
		TempRoot:     filepath.Clean(input.TempRoot),
		CreatedAt:    now,
		Retained:     input.Retained,
		CreatorPID:   os.Getpid(),
	}, nil
}

func MarkerPath(tempRoot string) string {
	return filepath.Join(tempRoot, filepath.FromSlash(MarkerRelPath))
}

func WriteMarker(marker RunMarker) error {
	path := MarkerPath(marker.TempRoot)
	if err := os.MkdirAll(filepath.Dir(path), tuning.PermDir); err != nil {
		return err
	}
	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return config.WriteOwnedFileAtomic(path, data, tuning.PermFile)
}

func ReadMarker(path string) (RunMarker, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RunMarker{}, err
	}
	var marker RunMarker
	if err := json.Unmarshal(data, &marker); err != nil {
		return RunMarker{}, err
	}
	return marker, nil
}

func ValidateMarker(repoRoot, markerPath string, marker RunMarker) error {
	cleanRepoRoot := filepath.Clean(repoRoot)
	if marker.Version != MarkerVersion {
		return fmt.Errorf("unsupported marker version %q", marker.Version)
	}
	if strings.TrimSpace(marker.RunID) == "" {
		return fmt.Errorf("marker run id is empty")
	}
	if !filepath.IsAbs(marker.TempRoot) {
		return fmt.Errorf("temp root is not absolute: %s", marker.TempRoot)
	}
	expectedTempRoot := filepath.Dir(filepath.Dir(filepath.Clean(markerPath)))
	if filepath.Clean(marker.TempRoot) != expectedTempRoot {
		return fmt.Errorf("temp root %s does not match marker directory %s", marker.TempRoot, expectedTempRoot)
	}
	if filepath.Clean(marker.RepoRoot) != cleanRepoRoot {
		return fmt.Errorf("marker repo root %s does not match current repo root %s", marker.RepoRoot, cleanRepoRoot)
	}
	if !isInside(filepath.Clean(marker.ScenarioPath), filepath.Clean(marker.TempRoot)) {
		return fmt.Errorf("scenario path %s is outside temp root %s", marker.ScenarioPath, marker.TempRoot)
	}
	if !strings.HasPrefix(marker.ScenarioID, "template-validation-") || !strings.HasSuffix(marker.ScenarioID, "-deep") {
		return fmt.Errorf("scenario id %q is not a deep template-validation scenario", marker.ScenarioID)
	}
	for _, path := range marker.RelocationArtifacts {
		if !filepath.IsAbs(path) {
			return fmt.Errorf("relocation artifact is not absolute: %s", path)
		}
		if !isInside(filepath.Clean(path), cleanRepoRoot) {
			return fmt.Errorf("relocation artifact %s is outside repo root %s", path, cleanRepoRoot)
		}
	}
	return nil
}

func newRunID(scenarioID string, now time.Time) (string, error) {
	var random [4]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s-%s", scenarioID, now.UTC().Format("20060102-150405"), hex.EncodeToString(random[:])), nil
}

func isInside(path, parent string) bool {
	rel, err := filepath.Rel(parent, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
