package process

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/vrooli/vrooli/internal/artifactlock"
)

func lockScenarioRecord(path string) (func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return artifactlock.AcquireWithPathContext(ctx, "process-record:"+path, func(string) (string, error) { return path + ".lock", nil })
}

// RemoveScenarioRecordIfMatches retires only the exact observed record. All
// record writers use the same cross-process lock, including the PID sidecar,
// so stale maintenance cannot erase a newly registered component generation.
func RemoveScenarioRecordIfMatches(home, name, step string, expected Record) (bool, error) {
	dir, err := ScenarioProcessDir(home, name)
	if err != nil {
		return false, err
	}
	path := filepath.Join(dir, step+".json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	release, err := lockScenarioRecord(path)
	if err != nil {
		return false, err
	}
	defer release()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var current Record
	if err := json.Unmarshal(data, &current); err != nil {
		return false, fmt.Errorf("parse current process record: %w", err)
	}
	if current.Step == "" {
		current.Step = step
	}
	if current.Scenario == "" {
		current.Scenario = name
	}
	if !current.StartedAt.Equal(expected.StartedAt) {
		return false, nil
	}
	// Compare the time value independently of location/monotonic representation.
	current.StartedAt = expected.StartedAt
	if current != expected {
		return false, nil
	}
	if err := removeScenarioRecordLocked(dir, step); err != nil {
		return false, err
	}
	return true, nil
}
