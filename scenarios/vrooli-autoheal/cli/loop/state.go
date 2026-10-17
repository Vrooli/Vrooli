package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
)

// loopStatus is the heartbeat and the last word. It is rewritten atomically
// on every tick and every state change, so a reader (the readiness check,
// the emergency watchdog the unit's OnFailure= points at) can tell a loop
// that is healing from one that gave up, and why, without the journal.
type loopStatus struct {
	StartedAt           time.Time        `json:"started_at"`
	LastTickAt          *time.Time       `json:"last_tick_at"`
	LastTickStatus      string           `json:"last_tick_status"`
	State               string           `json:"state"`
	ConsecutiveFailures int              `json:"consecutive_failures"`
	LastFailureClass    string           `json:"last_failure_class"`
	DegradedReason      string           `json:"degraded_reason"`
	Preflight           *PreflightResult `json:"preflight"`
	BinarySHA256        string           `json:"binary_sha256"`
	PID                 int              `json:"pid"`
	ExitCode            *int             `json:"exit_code,omitempty"`
	UpdatedAt           time.Time        `json:"updated_at"`
}

// statusFileDir is the loop's state directory under the runtime home's
// state entry, resolved through the repository contract.
func statusFileDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", fmt.Errorf("home directory unavailable: %w", err)
	}
	stateRoot, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return "", fmt.Errorf("resolve state entry: %w", err)
	}
	return filepath.Join(stateRoot, "vrooli-autoheal"), nil
}

// statusFilePath is ~/.vrooli/state/vrooli-autoheal/loop-status.json.
func statusFilePath() (string, error) {
	dir, err := statusFileDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "loop-status.json"), nil
}

// statusWriter owns the status file. A write is a temp file in the same
// directory followed by an atomic replace, so a reader never sees a partial
// document and a crash mid-write leaves the previous one intact.
type statusWriter struct {
	path string
}

// newStatusWriter resolves the path and proves the directory takes writes.
// The caller maps a failure to exit code 4.
func newStatusWriter() (*statusWriter, error) {
	path, err := statusFilePath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create state directory: %w", err)
	}
	return &statusWriter{path: path}, nil
}

func (w *statusWriter) write(status loopStatus) error {
	status.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(w.path), ".loop-status-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := atomicReplace(tmpPath, w.path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

// executableSHA256 hashes the running binary so a reader can tell whether
// the loop on disk is the loop in memory.
func executableSHA256() string {
	path, err := os.Executable()
	if err != nil {
		return ""
	}
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}
	return hex.EncodeToString(hash.Sum(nil))
}
