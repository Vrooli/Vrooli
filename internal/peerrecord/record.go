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

	"github.com/vrooli/vrooli/internal/tuning"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
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
	return filepath.Join(home, repocontractmeta.ProjectConfigDir, "peers", name+".json")
}

func Write(home string, record Record) error {
	if strings.TrimSpace(record.Scenario) == "" {
		return errors.New("peer record scenario is required")
	}
	record.SchemaVersion = SchemaVersion
	if record.Ports == nil {
		record.Ports = map[string]int{}
	}
	dir := filepath.Join(home, repocontractmeta.ProjectConfigDir, "peers")
	if err := os.MkdirAll(dir, tuning.PermPrivateDir); err != nil {
		return fmt.Errorf("create peer directory: %w", err)
	}
	if err := os.Chmod(dir, tuning.PermPrivateDir); err != nil {
		return fmt.Errorf("secure peer directory: %w", err)
	}
	payload, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	return config.WriteOwnedFileAtomic(Path(home, record.Scenario), payload, tuning.PermSecret)
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
