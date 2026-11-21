//go:build linux

package recovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
)

// This explicitly armed operator probe starts a checksum-bound, independent
// native deadline restore witness for an approved recovery drill. It has no
// default side effects and never dispatches a coding agent or resets budgets.
// The caller must review the exact script/specification and lifecycle scope
// before providing the opt-in; ordinary test runs skip this native operation.
func TestArmRecoveryIndependentRestoreProbe(t *testing.T) {
	path := os.Getenv("VROOLI_RECOVERY_RESTORE_PROBE_SPEC")
	if path == "" {
		t.Skip("requires an explicitly reviewed independent native restore specification")
	}
	var spec struct {
		FaultID      string `json:"fault_id"`
		Script       string `json:"script"`
		ScriptSHA256 string `json:"script_sha256"`
		Config       string `json:"config"`
		ConfigSHA256 string `json:"config_sha256"`
		ArtifactDir  string `json:"artifact_dir"`
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{path, spec.Script, spec.Config, spec.ArtifactDir} {
		if !filepath.IsAbs(value) || filepath.Clean(value) != value {
			t.Fatal("probe requires canonical absolute paths")
		}
	}
	script, err := os.ReadFile(spec.Script)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(script)
	if hex.EncodeToString(digest[:]) != spec.ScriptSHA256 {
		t.Fatal("reviewed restore script identity changed")
	}
	configuration, err := os.ReadFile(spec.Config)
	if err != nil {
		t.Fatal(err)
	}
	digest = sha256.Sum256(configuration)
	if hex.EncodeToString(digest[:]) != spec.ConfigSHA256 {
		t.Fatal("reviewed restore configuration identity changed")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	operation := platform.OperationSpec{Path: "/usr/bin/python3", Args: []string{spec.Script, spec.Config, spec.ConfigSHA256}, Dir: spec.ArtifactDir,
		Env:    []string{"HOME=" + home, "PATH=" + platform.DefaultPath("linux", home)},
		Limits: platform.OperationLimits{MemoryMaxBytes: 2 << 30, TasksMax: 1024, CPUWeight: 50, WallTime: 40 * time.Minute}}
	intent, err := platform.NewOperationIntent("recovery-drill-restore", spec.FaultID, operation)
	if err != nil {
		t.Fatal(err)
	}
	// Never reuse an intent after an uncertain start, or replace a prior receipt.
	intentPath := filepath.Join(spec.ArtifactDir, "restore-native-intent.json")
	encoded, err := json.Marshal(intent)
	if err != nil {
		t.Fatal(err)
	}
	file, err := config.OpenOwnedFile(intentPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatalf("restore intent already exists or cannot be exclusively created: %v", err)
	}
	if _, err := file.Write(encoded); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	directory, err := os.Open(spec.ArtifactDir)
	if err != nil {
		t.Fatal(err)
	}
	err = directory.Sync()
	_ = directory.Close()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	owned, err := platform.StartOwnedOperation(ctx, intent, operation)
	if err != nil {
		t.Fatalf("native dispatch uncertain; retain intent and inspect exact owner without replay: %v", err)
	}
	encoded, err = json.Marshal(owned.Ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := config.WriteOwnedFileDurable(filepath.Join(spec.ArtifactDir, "restore-native-ref.json"), encoded, 0600); err != nil {
		t.Fatal(err)
	}
	// Deliberately no stop-on-test-exit: survival of caller loss is the safety
	// condition this independent, bounded operation supplies for the drill.
	t.Logf("independent native restore armed: unit=%s intent=%s", owned.Ref.Unit, intentPath)
}

func TestRetireRecoveryIndependentRestoreProbe(t *testing.T) {
	path := os.Getenv("VROOLI_RECOVERY_RESTORE_PROBE_SPEC")
	if path == "" {
		t.Skip("requires the exact previously armed native restore specification")
	}
	var spec struct {
		FaultID     string `json:"fault_id"`
		ArtifactDir string `json:"artifact_dir"`
	}
	data, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &spec) != nil {
		t.Fatal("native restore specification cannot be read")
	}
	data, err = os.ReadFile(filepath.Join(spec.ArtifactDir, "restore-native-ref.json"))
	var ref platform.OperationRef
	if err != nil || json.Unmarshal(data, &ref) != nil || ref.ValidateIdentity() != nil || ref.Intent.Owner != "recovery-drill-restore" || ref.Intent.AttemptID != spec.FaultID {
		t.Fatal("native restore reference does not match the reviewed exact operation")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 41*time.Minute)
	defer cancel()
	t.Logf("attached once to exact native restore termination: unit=%s", ref.Unit)
	if err := (&platform.OwnedOperation{Ref: ref}).Wait(ctx); err != nil {
		t.Fatalf("native restore wait ended without termination proof; retain exact handle: %v", err)
	}
	data, err = os.ReadFile(filepath.Join(spec.ArtifactDir, "restore-receipt.json"))
	var receipt struct {
		State         string `json:"state"`
		LifecycleExit int    `json:"lifecycle_exit"`
		Ready         bool   `json:"ready"`
	}
	if err != nil || json.Unmarshal(data, &receipt) != nil {
		t.Fatal("native restore result cannot be read")
	}
	if receipt.State != "repaired-before-deadline" && receipt.State != "no-fault-observed" && !(receipt.State == "deadline-restoration-finished" && receipt.LifecycleExit == 0 && receipt.Ready) {
		t.Fatal("native restore lacks a terminal safe-source receipt; preserve it for inspection")
	}
	observation, err := platform.InspectOwnedOperation(ctx, ref)
	if err != nil || observation.Populated || observation.RootAlive || !observation.ExitKnown {
		t.Fatalf("native restore exit is unproved; preserve exact operation: %+v %v", observation, err)
	}
	encoded, err := json.Marshal(observation)
	if err != nil {
		t.Fatal(err)
	}
	if err := config.WriteOwnedFileDurable(filepath.Join(spec.ArtifactDir, "restore-native-terminal.json"), encoded, 0600); err != nil {
		t.Fatal(err)
	}
	if err := platform.ReleaseOwnedOperation(ctx, ref); err != nil {
		t.Fatal(err)
	}
	absent, err := platform.OwnedOperationIntentAbsent(ctx, ref.Intent)
	if err != nil || !absent {
		t.Fatalf("native restore retirement remains unproved: absent=%v error=%v", absent, err)
	}
	if err := config.WriteOwnedFileDurable(filepath.Join(spec.ArtifactDir, "restore-native-retired.json"), []byte(`{"exact_native_absence":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("independent native restore retired: unit=%s", ref.Unit)
}
