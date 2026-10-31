package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	applyReviewCleanupGrace = 24 * time.Hour
	applyArtifactRetention  = 7 * 24 * time.Hour
)

// cleanupOnboardingArtifacts is deliberately conservative: malformed files
// and non-terminal runs remain in place for manual recovery. User-selected
// support-export paths are not in this state-owned tree and are never deleted.
func cleanupOnboardingArtifacts() error {
	statePath, err := operatorStatePath()
	if err != nil {
		return err
	}
	root := filepath.Dir(statePath)
	now := operatorStateNow().UTC()
	if err := cleanupApplyReviews(filepath.Join(root, "apply-reviews"), filepath.Join(root, "apply-runs"), now); err != nil {
		return err
	}
	if err := cleanupApplyAdmissions(filepath.Join(root, "apply-admissions"), filepath.Join(root, "apply-runs"), now); err != nil {
		return err
	}
	return cleanupApplyRuns(filepath.Join(root, "apply-runs"), now)
}

func cleanupApplyReviews(dir, runsDir string, now time.Time) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read apply reviews: %w", err)
	}
	cutoff := now.Add(-applyReviewCleanupGrace)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var review applyReviewRecord
		if json.Unmarshal(data, &review) != nil {
			continue
		}
		expires, err := time.Parse(time.RFC3339, review.ExpiresAt)
		if err != nil || expires.After(cutoff) {
			continue
		}
		if review.ConsumedBy != "" {
			if run, ok := readRetainedApplyRun(filepath.Join(runsDir, review.ConsumedBy+".json")); !ok || !isTerminalApplyStatus(run.Status) {
				continue
			}
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove expired apply review: %w", err)
		}
	}
	return nil
}

func cleanupApplyRuns(dir string, now time.Time) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read apply runs: %w", err)
	}
	cutoff := now.Add(-applyArtifactRetention)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		run, ok := readRetainedApplyRun(path)
		if !ok || !isTerminalApplyStatus(run.Status) {
			continue
		}
		completed, err := time.Parse(time.RFC3339, run.CompletedAt)
		if err != nil || completed.After(cutoff) {
			continue
		}
		if err := removeApplyArtifact(path, strings.TrimSuffix(path, ".json")+".log"); err != nil {
			return err
		}
	}
	return nil
}

func cleanupApplyAdmissions(dir, runsDir string, now time.Time) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read apply admissions: %w", err)
	}
	cutoff := now.Add(-applyArtifactRetention)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var admission applyAdmissionRecord
		if json.Unmarshal(data, &admission) != nil {
			continue
		}
		created, err := time.Parse(time.RFC3339, admission.CreatedAt)
		if err != nil || created.After(cutoff) {
			continue
		}
		run, ok := readRetainedApplyRun(filepath.Join(runsDir, admission.RunID+".json"))
		if !ok || !isTerminalApplyStatus(run.Status) {
			continue
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove expired apply admission: %w", err)
		}
	}
	return nil
}

func readRetainedApplyRun(path string) (applyRun, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return applyRun{}, false
	}
	var run applyRun
	if json.Unmarshal(data, &run) != nil {
		return applyRun{}, false
	}
	return run, true
}

func removeApplyArtifact(paths ...string) error {
	for _, path := range paths {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove expired apply artifact: %w", err)
		}
	}
	return nil
}
