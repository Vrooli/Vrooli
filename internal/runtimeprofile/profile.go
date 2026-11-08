// Package runtimeprofile owns native scenario CPU-profile collection. A profile
// is read from an exact, pinned component launch through its private SDK socket.
package runtimeprofile

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/vrooli/api-core/runtimediagnostics"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// Directory follows the canonical runtime-home contract, including overrides.
func Directory(home string) (string, error) {
	if home == "" {
		var err error
		home, err = config.HomeDir()
		if err != nil {
			return "", err
		}
	}
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return "", err
	}
	return filepath.Join(state, "runtime-diagnostics"), nil
}

type Request struct {
	Scenario string
	Step     string
	Seconds  int
	Output   string
}

type Result struct {
	Identity            runtimediagnostics.Identity `json:"identity"`
	Step                string                      `json:"step"`
	Path                string                      `json:"path"`
	Seconds             int                         `json:"seconds"`
	Bytes               int                         `json:"bytes"`
	SHA256              string                      `json:"sha256"`
	ElapsedMilliseconds int64                       `json:"elapsed_milliseconds"`
}

func Capture(ctx context.Context, home string, request Request) (Result, error) {
	key, err := scenarioruntime.ParseInstanceKey(request.Scenario, "")
	if err != nil {
		return Result{}, err
	}
	if request.Step == "" || request.Seconds < 1 || request.Seconds > runtimediagnostics.MaxSeconds || request.Output == "" {
		return Result{}, errors.New("profile requires a step, output path, and duration between 1 and 30 seconds")
	}
	output, err := filepath.Abs(request.Output)
	if err != nil {
		return Result{}, err
	}
	if _, err := os.Lstat(output); err == nil {
		return Result{}, fmt.Errorf("profile destination must not already exist: %s", output)
	} else if !os.IsNotExist(err) {
		return Result{}, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(output), ".vrooli-cpu-*.pprof")
	if err != nil {
		return Result{}, err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	records, err := process.ReadScenarioRecords(home, key.Slug())
	if err != nil {
		return Result{}, err
	}
	record, err := selectRecord(records, request.Step)
	if err != nil {
		return Result{}, err
	}
	handle, err := process.PinRecord(record)
	if err != nil {
		return Result{}, fmt.Errorf("pin profiling target: %w", err)
	}
	defer handle.Close()
	directory, err := Directory(home)
	if err != nil {
		return Result{}, err
	}
	env, err := process.ReadEnvironment(record.PID)
	if err != nil {
		return Result{}, err
	}
	if record.RuntimeInstanceID == "" || env["VROOLI_RUNTIME_INSTANCE_ID"] != record.RuntimeInstanceID || env["VROOLI_SCENARIO"] != key.Scenario || env["VROOLI_VARIANT"] != key.Normalize().Variant {
		return Result{}, errors.New("profiling target does not match the recorded scenario instance")
	}
	if env[runtimediagnostics.DirectoryEnv] != directory {
		return Result{}, errors.New("runtime CPU diagnostics is unavailable for this launch; rebuild and restart through the control plane")
	}
	identity := runtimediagnostics.Identity{Scenario: key.Scenario, Variant: key.Normalize().Variant, RuntimeInstanceID: record.RuntimeInstanceID, Generation: record.Generation, PID: record.PID}
	client := runtimediagnostics.NewClient(runtimediagnostics.SocketPath(directory, identity))
	defer client.Close()
	capabilities, err := client.Capabilities(ctx)
	if err != nil {
		return Result{}, err
	}
	if capabilities.Identity != identity {
		return Result{}, errors.New("runtime diagnostics component identity mismatch")
	}
	started := time.Now()
	data, err := client.CPU(ctx, identity, request.Seconds)
	if err != nil {
		return Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	alive, err := handle.Alive()
	if err != nil || !alive {
		return Result{}, errors.New("profiling target exited during collection; profile discarded")
	}
	incarnation, err := process.Incarnation(record.PID)
	if err != nil || incarnation != record.Incarnation {
		return Result{}, errors.New("profiling target incarnation changed; profile discarded")
	}
	if _, err := tmp.Write(data); err != nil {
		return Result{}, err
	}
	if err := tmp.Sync(); err != nil {
		return Result{}, err
	}
	if err := tmp.Close(); err != nil {
		return Result{}, err
	}
	// Link publishes the complete artifact atomically and refuses overwrite,
	// including a destination created while the profile was being collected.
	if err := os.Link(tmp.Name(), output); err != nil {
		return Result{}, fmt.Errorf("publish CPU profile: %w", err)
	}
	sum := sha256.Sum256(data)
	return Result{Identity: identity, Step: record.Step, Path: output, Seconds: request.Seconds, Bytes: len(data), SHA256: hex.EncodeToString(sum[:]), ElapsedMilliseconds: time.Since(started).Milliseconds()}, nil
}

func selectRecord(records []process.Record, step string) (process.Record, error) {
	var matches []process.Record
	for _, record := range records {
		if record.Step == step && record.Status != "exited_verified" {
			matches = append(matches, record)
		}
	}
	if len(matches) != 1 {
		return process.Record{}, fmt.Errorf("expected exactly one recorded component for step %q, found %d", step, len(matches))
	}
	return matches[0], nil
}
