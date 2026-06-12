// Package checks provides Vrooli state reading abstractions for testability
// [REQ:VROOLI-STATE-001] [REQ:TEST-SEAM-001]
package checks

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	vroolicli "github.com/vrooli/vrooli-cli-go"
)

var errCLITrackedProcessesUnavailable = errors.New("vrooli cli state reader does not expose tracked process records")

// TrackedProcess represents a process tracked by Vrooli's lifecycle system.
// Data comes from ~/.vrooli/processes/scenarios/[scenario]/[step].json
type TrackedProcess struct {
	PID       int       `json:"pid"`
	PGID      int       `json:"pgid"`
	ProcessID string    `json:"process_id"` // e.g., "vrooli.develop.app-monitor.start-api"
	Phase     string    `json:"phase"`      // e.g., "develop"
	Scenario  string    `json:"scenario"`
	Step      string    `json:"step"`
	Command   string    `json:"command"`
	Port      int       `json:"port"`
	StartedAt time.Time `json:"started_at"`
	Status    string    `json:"status"`
}

// PortLock represents a port lock file from Vrooli's state directory.
// Format: ~/.vrooli/state/scenarios/.port_[port].lock
// Content: scenario_name:pid:timestamp
type PortLock struct {
	Port           int
	Scenario       string
	PID            int
	Timestamp      int64
	FilePath       string
	Source         string
	ClaimID        string
	InstanceID     string
	ClaimStatus    string
	InstanceStatus string
}

// VrooliStateReader abstracts access to Vrooli's state directories for testability.
// This interface allows checks to be unit tested without accessing the real filesystem.
// [REQ:TEST-SEAM-001]
type VrooliStateReader interface {
	// ListTrackedProcesses returns all processes tracked in ~/.vrooli/processes/scenarios/
	ListTrackedProcesses() ([]TrackedProcess, error)
	// ListPortLocks returns all port lock files from ~/.vrooli/state/scenarios/
	ListPortLocks() ([]PortLock, error)
	// RemovePortLock removes a specific port lock file
	RemovePortLock(lock PortLock) error
}

type CommandRunner interface {
	CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error)
}

// RealVrooliStateReader is the production implementation of VrooliStateReader.
type RealVrooliStateReader struct {
	homeDir string // Allow override for testing; empty means use os.UserHomeDir()
}

type VrooliCLIStateReader struct {
	runner CommandRunner
	cli    *vroolicli.Client
}

type FallbackVrooliStateReader struct {
	primary  VrooliStateReader
	fallback VrooliStateReader
}

type execCommandRunner struct{}

// NewRealVrooliStateReader creates a new state reader.
// If homeDir is empty, it uses the current user's home directory.
func NewRealVrooliStateReader(homeDir string) *RealVrooliStateReader {
	return &RealVrooliStateReader{homeDir: homeDir}
}

func NewVrooliCLIStateReader(runner CommandRunner) *VrooliCLIStateReader {
	if runner == nil {
		runner = execCommandRunner{}
	}
	return &VrooliCLIStateReader{
		runner: runner,
		// The runner already governs vrooli execution; suppress the client's own
		// --no-stale-check so the typed locks read issues exactly the invocation
		// this reader has always used.
		cli: vroolicli.New(
			vroolicli.WithRunner(commandRunnerAdapter{runner: runner}),
			vroolicli.WithStaleCheck(true),
		),
	}
}

// commandRunnerAdapter bridges the local combined-output-only CommandRunner to
// vroolicli.Runner.
type commandRunnerAdapter struct{ runner CommandRunner }

func (a commandRunnerAdapter) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return a.runner.CombinedOutput(ctx, name, args...)
}

func (a commandRunnerAdapter) RunCombined(ctx context.Context, name string, args ...string) ([]byte, error) {
	return a.runner.CombinedOutput(ctx, name, args...)
}

func NewFallbackVrooliStateReader(primary, fallback VrooliStateReader) *FallbackVrooliStateReader {
	return &FallbackVrooliStateReader{primary: primary, fallback: fallback}
}

func (execCommandRunner) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

func (r *RealVrooliStateReader) getHomeDir() (string, error) {
	if r.homeDir != "" {
		return r.homeDir, nil
	}
	return os.UserHomeDir()
}

// ListTrackedProcesses reads all process JSON files from ~/.vrooli/processes/scenarios/
func (r *RealVrooliStateReader) ListTrackedProcesses() ([]TrackedProcess, error) {
	homeDir, err := r.getHomeDir()
	if err != nil {
		return nil, err
	}

	processesDir := filepath.Join(homeDir, ".vrooli", "processes", "scenarios")

	// Check if directory exists
	if _, err := os.Stat(processesDir); os.IsNotExist(err) {
		return nil, nil // No processes tracked yet
	}

	var processes []TrackedProcess

	// Walk scenario directories
	scenarioDirs, err := os.ReadDir(processesDir)
	if err != nil {
		return nil, err
	}

	for _, scenarioDir := range scenarioDirs {
		if !scenarioDir.IsDir() {
			continue
		}

		scenarioPath := filepath.Join(processesDir, scenarioDir.Name())
		jsonFiles, err := os.ReadDir(scenarioPath)
		if err != nil {
			continue // Skip unreadable directories
		}

		for _, jsonFile := range jsonFiles {
			if jsonFile.IsDir() || !strings.HasSuffix(jsonFile.Name(), ".json") {
				continue
			}

			filePath := filepath.Join(scenarioPath, jsonFile.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				continue // Skip unreadable files
			}

			var proc TrackedProcess
			if err := json.Unmarshal(data, &proc); err != nil {
				continue // Skip malformed JSON
			}

			// Ensure scenario is set (might not be in older files)
			if proc.Scenario == "" {
				proc.Scenario = scenarioDir.Name()
			}

			processes = append(processes, proc)
		}
	}

	return processes, nil
}

// ListPortLocks reads all port lock files from ~/.vrooli/state/scenarios/
func (r *RealVrooliStateReader) ListPortLocks() ([]PortLock, error) {
	homeDir, err := r.getHomeDir()
	if err != nil {
		return nil, err
	}

	stateDir := filepath.Join(homeDir, ".vrooli", "state", "scenarios")

	// Check if directory exists
	if _, err := os.Stat(stateDir); os.IsNotExist(err) {
		return nil, nil // No state directory yet
	}

	entries, err := os.ReadDir(stateDir)
	if err != nil {
		return nil, err
	}

	var locks []PortLock

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Look for .port_[number].lock files
		if !strings.HasPrefix(name, ".port_") || !strings.HasSuffix(name, ".lock") {
			continue
		}

		// Extract port number from filename
		portStr := strings.TrimPrefix(name, ".port_")
		portStr = strings.TrimSuffix(portStr, ".lock")
		port, err := strconv.Atoi(portStr)
		if err != nil {
			continue // Invalid port number
		}

		filePath := filepath.Join(stateDir, name)
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue // Skip unreadable files
		}

		// Parse lock content: scenario_name:pid:timestamp
		content := strings.TrimSpace(string(data))
		parts := strings.Split(content, ":")
		if len(parts) < 2 {
			// Malformed lock file, but still track it
			locks = append(locks, PortLock{
				Port:     port,
				FilePath: filePath,
				Source:   "legacy_lock",
			})
			continue
		}

		scenario := parts[0]
		pid, _ := strconv.Atoi(parts[1])
		var timestamp int64
		if len(parts) >= 3 {
			timestamp, _ = strconv.ParseInt(parts[2], 10, 64)
		}

		locks = append(locks, PortLock{
			Port:      port,
			Scenario:  scenario,
			PID:       pid,
			Timestamp: timestamp,
			FilePath:  filePath,
			Source:    "legacy_lock",
		})
	}

	return locks, nil
}

// RemovePortLock removes a specific port lock file.
func (r *RealVrooliStateReader) RemovePortLock(lock PortLock) error {
	if lock.FilePath == "" {
		return nil // Nothing to remove
	}
	return os.Remove(lock.FilePath)
}

func (r *VrooliCLIStateReader) ListTrackedProcesses() ([]TrackedProcess, error) {
	return nil, errCLITrackedProcessesUnavailable
}

func (r *VrooliCLIStateReader) ListPortLocks() ([]PortLock, error) {
	resp, err := r.cli.Locks(context.Background())
	if err != nil {
		return nil, err
	}
	locks := make([]PortLock, 0, len(resp.GetRegistryClaims())+len(resp.GetLocks()))
	for _, claim := range resp.GetRegistryClaims() {
		locks = append(locks, PortLock{
			Port:           int(claim.GetPort()),
			Scenario:       claim.GetScenario(),
			Source:         "registry_claim",
			ClaimID:        claim.GetClaimId(),
			InstanceID:     claim.GetInstanceId(),
			ClaimStatus:    claim.GetClaimStatus(),
			InstanceStatus: claim.GetInstanceStatus(),
		})
	}
	for _, lock := range resp.GetLocks() {
		var timestamp int64
		if ts, err := time.Parse(time.RFC3339, lock.GetTimestamp()); err == nil {
			timestamp = ts.Unix()
		}
		locks = append(locks, PortLock{
			Port:      int(lock.GetPort()),
			Scenario:  lock.GetScenario(),
			PID:       int(lock.GetPid()),
			Timestamp: timestamp,
			FilePath:  lock.GetPath(),
			Source:    "legacy_lock",
		})
	}
	return locks, nil
}

func (r *VrooliCLIStateReader) RemovePortLock(lock PortLock) error {
	if lock.Source == "registry_claim" {
		_, err := r.runner.CombinedOutput(context.Background(), "vrooli", "cleanup", "locks")
		return err
	}
	return nil
}

func (r *FallbackVrooliStateReader) ListTrackedProcesses() ([]TrackedProcess, error) {
	processes, err := r.primary.ListTrackedProcesses()
	if err == nil {
		return processes, nil
	}
	return r.fallback.ListTrackedProcesses()
}

func (r *FallbackVrooliStateReader) ListPortLocks() ([]PortLock, error) {
	locks, err := r.primary.ListPortLocks()
	if err == nil {
		return locks, nil
	}
	return r.fallback.ListPortLocks()
}

func (r *FallbackVrooliStateReader) RemovePortLock(lock PortLock) error {
	if lock.Source == "registry_claim" {
		return r.primary.RemovePortLock(lock)
	}
	return r.fallback.RemovePortLock(lock)
}

// DefaultVrooliStateReader is the global state reader used when none is injected.
// It prefers the core `vrooli locks --json` contract so registry claims and
// legacy lock files are interpreted the same way autoheal would see them from
// the host CLI, with a filesystem fallback for older local installs.
var DefaultVrooliStateReader VrooliStateReader = NewFallbackVrooliStateReader(
	NewVrooliCLIStateReader(nil),
	NewRealVrooliStateReader(""),
)

// ProcessExists checks if a process with the given PID is running.
// This is a helper that uses the ProcReader interface.
func ProcessExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	// Check if /proc/[pid] exists (Linux-specific)
	_, err := os.Stat("/proc/" + strconv.Itoa(pid))
	return err == nil
}
