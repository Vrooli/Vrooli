package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

// Replica describes a replication target.
type Replica struct {
	Database string        `json:"database"`
	Target   string        `json:"target"`
	Interval time.Duration `json:"interval"`
	Enabled  bool          `json:"enabled"`
}

type replicaState struct {
	Replicas []Replica `json:"replicas"`
	LastSync *struct {
		Database  string    `json:"database"`
		Timestamp time.Time `json:"timestamp"`
	} `json:"last_sync"` // legacy field (single last sync)
	LastSyncTimes map[string]time.Time `json:"last_sync_times,omitempty"`
}

func (s *Service) replicaStatePath() string {
	return filepath.Join(s.Config.ReplicaPath, "state.json")
}

func (s *Service) lockReplicaState() (*flock.Flock, error) {
	path := s.replicaStatePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	lock := flock.New(path + ".lock")
	if err := lock.Lock(); err != nil {
		return nil, err
	}
	return lock, nil
}

func (s *Service) loadReplicaStateLocked() (replicaState, error) {
	path := s.replicaStatePath()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return replicaState{Replicas: []Replica{}, LastSyncTimes: map[string]time.Time{}}, nil
	}
	if err != nil {
		return replicaState{}, err
	}
	var state replicaState
	if err := json.Unmarshal(data, &state); err != nil {
		return replicaState{}, err
	}
	if state.LastSyncTimes == nil {
		state.LastSyncTimes = map[string]time.Time{}
	}
	// Migrate legacy last_sync into map if present.
	if state.LastSync != nil {
		key := fmt.Sprintf("%s|%s", state.LastSync.Database, "")
		state.LastSyncTimes[key] = state.LastSync.Timestamp
	}
	return state, nil
}

func (s *Service) saveReplicaStateLocked(state replicaState) error {
	path := s.replicaStatePath()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "state-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

func (s *Service) AddReplica(dbName, target string, interval time.Duration) error {
	if err := s.ValidateName(dbName); err != nil {
		return err
	}
	if target == "" {
		return errors.New("target path required")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("target dir not accessible: %w", err)
	}

	lock, err := s.lockReplicaState()
	if err != nil {
		return err
	}
	defer lock.Unlock()

	state, err := s.loadReplicaStateLocked()
	if err != nil {
		return err
	}
	state.Replicas = append(state.Replicas, Replica{
		Database: dbName,
		Target:   target,
		Interval: interval,
		Enabled:  true,
	})
	if state.LastSyncTimes == nil {
		state.LastSyncTimes = map[string]time.Time{}
	}
	return s.saveReplicaStateLocked(state)
}

func (s *Service) RemoveReplica(dbName, target string) error {
	lock, err := s.lockReplicaState()
	if err != nil {
		return err
	}
	defer lock.Unlock()

	state, err := s.loadReplicaStateLocked()
	if err != nil {
		return err
	}
	var next []Replica
	for _, r := range state.Replicas {
		if !(r.Database == dbName && r.Target == target) {
			next = append(next, r)
		}
	}
	state.Replicas = next
	return s.saveReplicaStateLocked(state)
}

func (s *Service) ListReplicas() ([]Replica, error) {
	lock, err := s.lockReplicaState()
	if err != nil {
		return nil, err
	}
	defer lock.Unlock()

	state, err := s.loadReplicaStateLocked()
	if err != nil {
		return nil, err
	}
	return state.Replicas, nil
}

func (s *Service) ToggleReplica(dbName, target string, enabled bool) error {
	lock, err := s.lockReplicaState()
	if err != nil {
		return err
	}
	defer lock.Unlock()

	state, err := s.loadReplicaStateLocked()
	if err != nil {
		return err
	}
	changed := false
	for i, r := range state.Replicas {
		if r.Database == dbName && r.Target == target {
			state.Replicas[i].Enabled = enabled
			changed = true
		}
	}
	if !changed {
		return errors.New("replica not found")
	}
	return s.saveReplicaStateLocked(state)
}

func (s *Service) SyncReplica(ctx context.Context, dbName string, force bool) (int, int, error) {
	sourcePath, err := s.databasePath(dbName)
	if err != nil {
		return 0, 0, err
	}
	if _, err := os.Stat(sourcePath); err != nil {
		return 0, 0, fmt.Errorf("database not found: %s", dbName)
	}
	lock, err := s.lockReplicaState()
	if err != nil {
		return 0, 0, err
	}
	defer lock.Unlock()

	state, err := s.loadReplicaStateLocked()
	if err != nil {
		return 0, 0, err
	}
	success, failed := 0, 0
	for _, r := range state.Replicas {
		if r.Database != dbName || !r.Enabled {
			continue
		}
		if err := s.copyDatabase(ctx, sourcePath, r.Target, force); err != nil {
			failed++
		} else {
			success++
			if state.LastSyncTimes == nil {
				state.LastSyncTimes = map[string]time.Time{}
			}
			state.LastSyncTimes[s.replicaKey(r)] = time.Now()
		}
	}
	// Update last sync metadata.
	state.LastSync = &struct {
		Database  string    `json:"database"`
		Timestamp time.Time `json:"timestamp"`
	}{Database: dbName, Timestamp: time.Now()}
	_ = s.saveReplicaStateLocked(state)
	return success, failed, nil
}

func (s *Service) VerifyReplicas(ctx context.Context, dbName string) ([]string, error) {
	sourcePath, err := s.databasePath(dbName)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(sourcePath); err != nil {
		return nil, fmt.Errorf("database not found: %s", dbName)
	}
	lock, err := s.lockReplicaState()
	if err != nil {
		return nil, err
	}
	defer lock.Unlock()

	state, err := s.loadReplicaStateLocked()
	if err != nil {
		return nil, err
	}
	var issues []string
	for _, r := range state.Replicas {
		if r.Database != dbName || !r.Enabled {
			continue
		}
		if _, err := os.Stat(r.Target); err != nil {
			issues = append(issues, fmt.Sprintf("replica missing: %s", r.Target))
			continue
		}
		sourceTables, _ := s.countTables(ctx, sourcePath)
		replicaTables, _ := s.countTables(ctx, r.Target)
		if sourceTables != replicaTables {
			issues = append(issues, fmt.Sprintf("table count mismatch %s (src %d vs %d)", r.Target, sourceTables, replicaTables))
		}
		if err := s.integrityCheck(ctx, r.Target); err != nil {
			issues = append(issues, fmt.Sprintf("integrity check failed %s: %v", r.Target, err))
		}
	}
	return issues, nil
}

// SyncDueReplicas syncs enabled replicas whose interval has elapsed.
func (s *Service) SyncDueReplicas(ctx context.Context, force bool) (int, int, error) {
	lock, err := s.lockReplicaState()
	if err != nil {
		return 0, 0, err
	}
	defer lock.Unlock()

	state, err := s.loadReplicaStateLocked()
	if err != nil {
		return 0, 0, err
	}
	now := time.Now()
	success, failed := 0, 0
	for _, r := range state.Replicas {
		if !r.Enabled {
			continue
		}
		if r.Interval <= 0 {
			continue
		}
		last := state.LastSyncTimes[s.replicaKey(r)]
		if !force && now.Sub(last) < r.Interval {
			continue
		}
		sourcePath, err := s.databasePath(r.Database)
		if err != nil {
			failed++
			continue
		}
		if err := s.copyDatabase(ctx, sourcePath, r.Target, true); err != nil {
			failed++
			continue
		}
		success++
		if state.LastSyncTimes == nil {
			state.LastSyncTimes = map[string]time.Time{}
		}
		state.LastSyncTimes[s.replicaKey(r)] = now
	}
	_ = s.saveReplicaStateLocked(state)
	return success, failed, nil
}

func (s *Service) replicaKey(r Replica) string {
	return fmt.Sprintf("%s|%s", r.Database, r.Target)
}
