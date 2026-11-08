//go:build linux

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/repo-contract-go/cliinvoke"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// This opt-in fixture uses a governed real root CLI image and an empty private
// runtime home. It never alters the serving unit, source checkout or registry.
// Both source hints, CWD and the executable's ancestors lack a source tree.
func TestNativeSupervisorCurrentHostSourceIndependentBootstrapProbe(t *testing.T) {
	if os.Getenv("VROOLI_NATIVE_BOOTSTRAP_PROBE") != "1" {
		t.Skip("set VROOLI_NATIVE_BOOTSTRAP_PROBE=1 for the isolated actual-CLI native bootstrap drill")
	}
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	image := filepath.Join(root, ".vrooli", "build", "vrooli")
	if selected := os.Getenv("VROOLI_NATIVE_BOOTSTRAP_IMAGE"); selected != "" {
		image = selected
	}
	home := t.TempDir()
	bin := filepath.Join(home, ".vrooli", "bin")
	if err := os.MkdirAll(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	frozen := filepath.Join(bin, "vrooli")
	if err := cliinstall.AtomicInstall(image, frozen); err != nil {
		t.Fatalf("freeze qualified actual CLI: %v", err)
	}
	missingSource := filepath.Join(home, "source-unavailable")
	env := []string{"HOME=" + home, "PATH=/usr/local/bin:/usr/bin:/bin", "VROOLI_SOURCE_ROOT=" + missingSource, "VROOLI_ROOT=" + missingSource, "VROOLI_RUNTIME_SUPERVISOR=on", "VROOLI_RUNTIME_SUPERVISOR_RENEW_INTERVAL=200ms"}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	query := exec.CommandContext(ctx, frozen, "version", "--json")
	query.Dir = home
	query.Env = env
	data, err := query.Output()
	if err != nil {
		t.Fatal(err)
	}
	var identity struct {
		BuildIdentity             string `json:"build_identity"`
		RuntimeHomeContractSHA256 string `json:"runtime_home_contract_sha256"`
		Root                      string `json:"root"`
	}
	if err := json.Unmarshal(data, &identity); err != nil {
		t.Fatal(err)
	}
	c, err := repocontract.LoadDefault(root)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := c.RuntimeHomeSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(snapshot)
	// Root is the explicitly bound source hint, not an availability witness.
	// The version probe may retain that hint without opening its directory.
	if identity.Root != missingSource || identity.RuntimeHomeContractSHA256 != hex.EncodeToString(sum[:]) || len(identity.BuildIdentity) != 64 {
		t.Fatalf("artifact/source isolation identity mismatch: %+v", identity)
	}
	dbEntry, err := c.RuntimeHomeEntry(home, repocontract.HomeKeyRuntimeDB)
	if err != nil {
		t.Fatal(err)
	}
	spec := platform.OperationSpec{Path: frozen, Args: cliinvoke.RuntimeSupervisorRun(), Env: env, Dir: home, Limits: platform.OperationLimits{MemoryMaxBytes: 384 << 20, TasksMax: 128, CPUWeight: 100, WallTime: 30 * time.Second}}
	intent, err := platform.NewOperationIntent("root-bootstrap", "source-absent-"+strconv.FormatInt(time.Now().UnixNano(), 10), spec)
	if err != nil {
		t.Fatal(err)
	}
	// Own cleanup before dispatch, including an ambiguous start result.
	t.Cleanup(func() {
		cleanup, done := context.WithTimeout(context.Background(), 12*time.Second)
		defer done()
		ref, err := platform.LookupOwnedOperation(cleanup, intent)
		if err == nil {
			_, _ = platform.StopOwnedOperation(cleanup, ref)
			_ = platform.ReleaseOwnedOperation(cleanup, ref)
		}
	})
	started := time.Now()
	op, err := platform.StartOwnedOperation(ctx, intent, spec)
	if err != nil {
		t.Fatal(err)
	}
	var session scenarioruntime.SupervisorSession
	leaseMatches := func(s scenarioruntime.SupervisorSession) bool {
		var owner struct {
			BuildIdentity string `json:"build_identity"`
		}
		if err := json.Unmarshal([]byte(s.MetadataJSON), &owner); err != nil {
			return false
		}
		return s.PID != nil && *s.PID == op.Ref.RootPID && owner.BuildIdentity == identity.BuildIdentity && s.LastHeartbeatAt.After(started) && s.LastHeartbeatAt.After(s.StartedAt.Add(300*time.Millisecond)) && s.HeartbeatDeadlineAt.After(time.Now())
	}
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(dbEntry.AbsPath); err == nil {
			store, openErr := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbEntry.AbsPath, ReadOnly: true})
			if openErr == nil {
				rows, readErr := store.ListSupervisorSessions(ctx, scenarioruntime.SupervisorSessionFilter{Statuses: []string{scenarioruntime.SupervisorStatusRunning}})
				_ = store.Close()
				if readErr == nil && len(rows) == 1 {
					session = rows[0]
					if leaseMatches(session) {
						break
					}
				}
			}
		}
		time.Sleep(40 * time.Millisecond)
	}
	if !leaseMatches(session) {
		t.Fatalf("actual native CLI did not establish a source-independent supervisor lease: %+v", session)
	}
	if _, err := os.Stat(missingSource); !os.IsNotExist(err) {
		t.Fatal("native bootstrap created or recovered source instead of remaining independent")
	}
	if receipt, err := platform.StopOwnedOperation(ctx, op.Ref); err != nil || !receipt.NativeQuiescent {
		t.Fatalf("native fixture retirement failed: %+v %v", receipt, err)
	}
	if err := platform.ReleaseOwnedOperation(ctx, op.Ref); err != nil {
		t.Fatal(err)
	}
	t.Logf("actual root native bootstrap: source_unavailable=true runtime_home_contract=%s build=%s lease_owner_pid=%d periodic_renewal=true frozen_image=true native_tree_retired=true unit=%s", identity.RuntimeHomeContractSHA256, identity.BuildIdentity, *session.PID, op.Ref.Unit)
}
