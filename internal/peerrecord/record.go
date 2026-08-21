// Package peerrecord owns the cross-tier, same-user peer discovery file.
// It deliberately has no runtime-store dependency so Tier 1 and packaged
// desktop runtimes can share the exact codec and staleness rule.
package peerrecord

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
)

const SchemaVersion = 1

type Record struct {
	SchemaVersion int            `json:"schema_version"`
	Scenario      string         `json:"scenario"`
	Instance      string         `json:"instance"`
	Tier          int            `json:"tier"`
	OwnerPID      int            `json:"owner_pid"`
	StartedAt     time.Time      `json:"started_at"`
	Ports         map[string]int `json:"ports"`
	AuthTokenPath string         `json:"auth_token_path"`
}

func Path(home, name string) string {
	return filepath.Join(home, ".vrooli", "peers", name+".json")
}

func Write(home string, record Record) error {
	if strings.TrimSpace(record.Scenario) == "" {
		return errors.New("peer record scenario is required")
	}
	record.SchemaVersion = SchemaVersion
	if record.Ports == nil {
		record.Ports = map[string]int{}
	}
	dir := filepath.Join(home, ".vrooli", "peers")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create peer directory: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("secure peer directory: %w", err)
	}
	payload, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	tmp, err := os.CreateTemp(dir, ".peer-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(payload); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, Path(home, record.Scenario))
}

func Read(home, name string) (Record, error) {
	payload, err := os.ReadFile(Path(home, name))
	if err != nil {
		return Record{}, err
	}
	var record Record
	if err := json.Unmarshal(payload, &record); err != nil {
		return Record{}, err
	}
	if record.SchemaVersion != SchemaVersion || record.Scenario != name {
		return Record{}, fmt.Errorf("invalid peer record for %s", name)
	}
	if record.OwnerPID <= 0 || !platform.IsPIDRunning(record.OwnerPID) {
		return Record{}, os.ErrNotExist
	}
	return record, nil
}

func Remove(home, name string) error {
	err := os.Remove(Path(home, name))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
