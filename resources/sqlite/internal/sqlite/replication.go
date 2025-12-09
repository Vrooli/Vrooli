package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
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
	} `json:"last_sync"`
}

func (s *Service) replicaStatePath() string {
	return filepath.Join(s.Config.ReplicaPath, "state.json")
}

func (s *Service) loadReplicaState() (replicaState, error) {
	path := s.replicaStatePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return replicaState{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return replicaState{Replicas: []Replica{}}, nil
	}
	if err != nil {
		return replicaState{}, err
	}
	var state replicaState
	if err := json.Unmarshal(data, &state); err != nil {
		return replicaState{}, err
	}
	return state, nil
}

func (s *Service) saveReplicaState(state replicaState) error {
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

	state, err := s.loadReplicaState()
	if err != nil {
		return err
	}
	state.Replicas = append(state.Replicas, Replica{
		Database: dbName,
		Target:   target,
		Interval: interval,
		Enabled:  true,
	})
	return s.saveReplicaState(state)
}

func (s *Service) RemoveReplica(dbName, target string) error {
	state, err := s.loadReplicaState()
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
	return s.saveReplicaState(state)
}

func (s *Service) ListReplicas() ([]Replica, error) {
	state, err := s.loadReplicaState()
	if err != nil {
		return nil, err
	}
	return state.Replicas, nil
}

func (s *Service) ToggleReplica(dbName, target string, enabled bool) error {
	state, err := s.loadReplicaState()
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
	return s.saveReplicaState(state)
}

func (s *Service) SyncReplica(ctx context.Context, dbName string, force bool) (int, int, error) {
	sourcePath, err := s.databasePath(dbName)
	if err != nil {
		return 0, 0, err
	}
	if _, err := os.Stat(sourcePath); err != nil {
		return 0, 0, fmt.Errorf("database not found: %s", dbName)
	}
	state, err := s.loadReplicaState()
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
		}
	}
	// Update last sync metadata.
	state.LastSync = &struct {
		Database  string    `json:"database"`
		Timestamp time.Time `json:"timestamp"`
	}{Database: dbName, Timestamp: time.Now()}
	_ = s.saveReplicaState(state)
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
	state, err := s.loadReplicaState()
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
