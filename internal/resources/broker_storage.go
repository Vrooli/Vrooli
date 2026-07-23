package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type BrokerState struct {
	Instances map[string]ManagedInstance `json:"instances"`
	Leases    map[string]Lease           `json:"leases"`
	Sequence  uint64                     `json:"sequence"`
}

type BrokerStore interface {
	Load() (BrokerState, error)
	Save(BrokerState) error
}

type FileBrokerStore struct{ Path string }

func (s FileBrokerStore) Load() (BrokerState, error) {
	data, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return BrokerState{Instances: map[string]ManagedInstance{}, Leases: map[string]Lease{}}, nil
	}
	if err != nil {
		return BrokerState{}, fmt.Errorf("read broker state: %w", err)
	}
	var state BrokerState
	if err := json.Unmarshal(data, &state); err != nil {
		return BrokerState{}, fmt.Errorf("parse broker state: %w", err)
	}
	return state, nil
}

func (s FileBrokerStore) Save(state BrokerState) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return fmt.Errorf("create broker state directory: %w", err)
	}
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode broker state: %w", err)
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write broker state: %w", err)
	}
	if err := os.Rename(tmp, s.Path); err != nil {
		return fmt.Errorf("commit broker state: %w", err)
	}
	return nil
}
