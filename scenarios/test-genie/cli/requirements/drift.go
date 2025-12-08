package requirements

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"test-genie/internal/requirements/evidence"
	"test-genie/internal/requirements/snapshot"
)

type driftResult struct {
	Status           string   `json:"status"`
	MissingSnapshot  bool     `json:"missing_snapshot,omitempty"`
	SnapshotPath     string   `json:"snapshot_path,omitempty"`
	SnapshotTime     string   `json:"snapshot_time,omitempty"`
	RequirementDrift bool     `json:"requirement_drift,omitempty"`
	ArtifactStale    bool     `json:"artifact_stale,omitempty"`
	ManualExpired    []string `json:"manual_expired,omitempty"`
	Messages         []string `json:"messages,omitempty"`
}

func runDrift(args []string) error {
	fs := flag.NewFlagSet("requirements drift", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output JSON")
	dirFlag, scenarioFlag := parseCommonFlags(fs)

	if err := fs.Parse(args); err != nil {
		return err
	}

	dir, err := resolveDir(*dirFlag)
	if err != nil {
		return err
	}
	if err := ensureDir(dir); err != nil {
		return err
	}

	result, err := detectDrift(dir)
	if err != nil {
		return err
	}

	if *jsonOut {
		fmt.Println(toJSON(result))
		if result.Status == "ok" {
			return nil
		}
		return fmt.Errorf("drift detected")
	}

	switch result.Status {
	case "missing_snapshot":
		return fmt.Errorf("⚠️  Snapshot missing; run the test suite for '%s' to regenerate", scenarioNameFromDir(dir, *scenarioFlag))
	case "ok":
		fmt.Printf("✅ No drift detected for '%s'\n", scenarioNameFromDir(dir, *scenarioFlag))
		return nil
	default:
		fmt.Printf("❌ Drift detected for '%s'\n", scenarioNameFromDir(dir, *scenarioFlag))
		for _, msg := range result.Messages {
			fmt.Printf("  • %s\n", msg)
		}
		return fmt.Errorf("drift detected")
	}
}

func detectDrift(dir string) (driftResult, error) {
	snapshotPath := filepath.Join(dir, "coverage", "requirements-sync", "latest.json")
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		return driftResult{
			Status:          "missing_snapshot",
			MissingSnapshot: true,
			SnapshotPath:    snapshotPath,
		}, nil
	}

	var snap snapshot.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return driftResult{}, fmt.Errorf("failed to parse snapshot: %w", err)
	}

	result := driftResult{
		Status:       "ok",
		SnapshotPath: snapshotPath,
		SnapshotTime: snap.GeneratedAt.Format(time.RFC3339),
	}

	latestReqChange, err := newestRequirementChange(dir)
	if err == nil && latestReqChange.After(snap.GeneratedAt) {
		result.RequirementDrift = true
		result.Messages = append(result.Messages, fmt.Sprintf("Requirements changed at %s (after snapshot)", latestReqChange.UTC().Format(time.RFC3339)))
	}

	latestArtifact, err := newestArtifact(dir)
	if err == nil && latestArtifact.After(snap.GeneratedAt) {
		result.ArtifactStale = true
		result.Messages = append(result.Messages, fmt.Sprintf("Coverage artifacts newer than snapshot (%s)", latestArtifact.UTC().Format(time.RFC3339)))
	}

	expiredManual, err := expiredManualValidations(dir)
	if err == nil && len(expiredManual) > 0 {
		result.ManualExpired = expiredManual
		result.Messages = append(result.Messages, fmt.Sprintf("%d manual validation(s) expired", len(expiredManual)))
	}

	if result.RequirementDrift || result.ArtifactStale || len(result.ManualExpired) > 0 {
		result.Status = "drift_detected"
	}

	return result, nil
}

func newestRequirementChange(dir string) (time.Time, error) {
	reqDir := filepath.Join(dir, "requirements")
	if _, err := os.Stat(reqDir); err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	var latest time.Time
	err := filepath.Walk(reqDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(info.Name()) != ".json" {
			return nil
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		return nil
	})
	return latest, err
}

func newestArtifact(dir string) (time.Time, error) {
	var latest time.Time
	candidates := []string{
		filepath.Join(dir, "coverage", "phase-results"),
		filepath.Join(dir, "ui", "coverage", "vitest-requirements.json"),
		filepath.Join(dir, "coverage", "manual-validations"),
	}
	for _, path := range candidates {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.IsDir() {
			filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if !fi.IsDir() && fi.ModTime().After(latest) {
					latest = fi.ModTime()
				}
				return nil
			})
			continue
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
	}
	return latest, nil
}

func expiredManualValidations(dir string) ([]string, error) {
	loader := evidence.NewDefault()
	manifest, err := loader.LoadManualValidations(context.Background(), dir)
	if err != nil || manifest == nil {
		return nil, err
	}
	var expired []string
	now := time.Now()
	for _, entry := range manifest.Entries {
		if !entry.ExpiresAt.IsZero() && entry.ExpiresAt.Before(now) {
			expired = append(expired, entry.RequirementID)
		}
	}
	return expired, nil
}
