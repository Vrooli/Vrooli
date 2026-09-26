//go:build linux

package maintenance

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func TestDemandGroupHelper(t *testing.T) {
	if os.Getenv("VROOLI_DEMAND_GROUP_HELPER") != "1" {
		return
	}
	if os.Getenv("VROOLI_DEMAND_GROUP_IGNORE_TERM") == "1" {
		signal.Ignore(syscall.SIGTERM)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	fmt.Println(listener.Addr().String())
	_, _ = io.Copy(io.Discard, os.Stdin)
	os.Exit(0)
}

func startDemandGroupMember(t *testing.T, group int, variant string, ignoreTerm bool) (*exec.Cmd, process.Handle, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDemandGroupHelper$")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: group}
	cmd.Env = append(os.Environ(), "VROOLI_DEMAND_GROUP_HELPER=1", "VROOLI_RUNTIME_INSTANCE_ID=group-fixture", "VROOLI_SCENARIO=fixture", "VROOLI_VARIANT="+variant)
	if ignoreTerm {
		cmd.Env = append(cmd.Env, "VROOLI_DEMAND_GROUP_IGNORE_TERM=1")
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	address, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatalf("helper readiness: %v", err)
	}
	address = strings.TrimSpace(address)
	if _, _, err := net.SplitHostPort(address); err != nil {
		t.Fatal(err)
	}
	handle, err := process.OpenHandle(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = handle.Close() })
	return cmd, handle, address
}

func TestDemandStopValidatesEveryProcessGroupMember(t *testing.T) {
	for _, variant := range []string{"live", "foreign", "forced", "cancelled-after-signal", "retry-finalization"} {
		t.Run(variant, func(t *testing.T) {
			workerVariant := variant
			if variant == "forced" || variant == "cancelled-after-signal" || variant == "retry-finalization" {
				workerVariant = "live"
			}
			leader, leaderHandle, leaderAddress := startDemandGroupMember(t, 0, "live", variant == "forced")
			_, workerHandle, workerAddress := startDemandGroupMember(t, leader.Process.Pid, workerVariant, variant == "forced")
			_, portText, _ := net.SplitHostPort(workerAddress)
			port, err := strconv.Atoi(portText)
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			dbPath := filepath.Join(t.TempDir(), "runtime.db")
			store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath})
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			instance, err := store.CreateInstance(ctx, scenarioruntime.Instance{InstanceID: "group-fixture", Scenario: "fixture", Status: scenarioruntime.StatusRunning, SupervisionPolicy: scenarioruntime.SupervisionPolicyDemand})
			if err != nil {
				t.Fatal(err)
			}
			pid := leader.Process.Pid
			if _, err := store.AddProcessRef(ctx, scenarioruntime.ProcessRef{RefID: "leader", InstanceID: instance.InstanceID, PID: &pid, PGID: &pid, Status: "running"}); err != nil {
				t.Fatal(err)
			}
			if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{ClaimID: "group-port", InstanceID: instance.InstanceID, Scenario: instance.Scenario, PortName: "api", Port: port, Status: scenarioruntime.ClaimStatusBound}); err != nil {
				t.Fatal(err)
			}
			claimed, err := store.ClaimDemandStopCandidates(ctx, time.Now(), "expired")
			if err != nil || len(claimed) != 1 {
				t.Fatalf("claim = %v, %v", claimed, err)
			}
			workCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			var stopStore runtimeMaintenanceStore = store
			if variant == "cancelled-after-signal" {
				stopStore = &cancelDemandRefUpdateStore{runtimeMaintenanceStore: store, cancel: cancel}
			}
			if variant == "retry-finalization" {
				stopStore = &failDemandFinalizationStore{runtimeMaintenanceStore: store}
			}
			err = NewController(t.TempDir(), t.TempDir()).stopDemandManagedInstance(workCtx, stopStore, claimed[0])
			if variant == "retry-finalization" {
				if !errors.Is(err, errInjectedDemandFinalization) {
					t.Fatalf("expected finalization failure, got %v", err)
				}
				after, readErr := store.GetInstance(ctx, instance.InstanceID)
				if readErr != nil || after.Status != scenarioruntime.StatusStopping {
					t.Fatalf("partial teardown admitted consumers: %+v %v", after, readErr)
				}
				claims, readErr := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{InstanceID: instance.InstanceID})
				if readErr != nil || len(claims) != 1 || claims[0].Status != scenarioruntime.ClaimStatusBound {
					t.Fatalf("partial teardown lost claims: %+v %v", claims, readErr)
				}
				// A new controller opens a fresh registry handle and discovers
				// the persisted intent without the previous candidate snapshot.
				report, retryErr := NewControllerWithDBPath(t.TempDir(), t.TempDir(), dbPath).ReconcileDemandLeases(ctx)
				if retryErr != nil || len(report.Failed) != 0 || len(report.Stopped) != 1 {
					t.Fatalf("recovery report=%+v err=%v", report, retryErr)
				}
				err = nil
			}
			if variant == "cancelled-after-signal" && workCtx.Err() == nil {
				t.Fatal("fixture did not cancel during teardown")
			}
			foreign := variant == "foreign"
			if (err != nil) != foreign {
				t.Fatalf("stop error = %v; foreign=%v", err, foreign)
			}
			for _, handle := range []process.Handle{leaderHandle, workerHandle} {
				if alive, err := handle.Alive(); err != nil || alive != foreign {
					t.Fatalf("group member alive=%v err=%v; foreign=%v", alive, err, foreign)
				}
			}
			wantStatus := scenarioruntime.StatusStopped
			wantClaim := scenarioruntime.ClaimStatusReleased
			if foreign {
				wantStatus = scenarioruntime.StatusRunning
				wantClaim = scenarioruntime.ClaimStatusBound
			}
			after, err := store.GetInstance(ctx, instance.InstanceID)
			if err != nil || after.Status != wantStatus {
				t.Fatalf("instance=%+v err=%v, want %s", after, err, wantStatus)
			}
			for _, address := range []string{leaderAddress, workerAddress} {
				if foreign {
					conn, err := net.DialTimeout("tcp", address, time.Second)
					if err != nil {
						t.Fatalf("foreign group lost listener %s: %v", address, err)
					}
					_ = conn.Close()
				} else {
					listener, err := net.Listen("tcp", address)
					if err != nil {
						t.Fatalf("stopped group retained listener %s: %v", address, err)
					}
					_ = listener.Close()
				}
			}
			claims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{InstanceID: instance.InstanceID})
			if err != nil || len(claims) != 1 || claims[0].Status != wantClaim {
				t.Fatalf("claims=%+v err=%v, want %s", claims, err, wantClaim)
			}
		})
	}
}

// The first live ref update happens after signaling and exit confirmation.
// Cancel the caller there to prove teardown persistence has its own budget.
type cancelDemandRefUpdateStore struct {
	runtimeMaintenanceStore
	cancel context.CancelFunc
}

func (s *cancelDemandRefUpdateStore) UpdateProcessRefStatus(ctx context.Context, refID, status string, endedAt *time.Time) (scenarioruntime.ProcessRef, error) {
	s.cancel()
	return s.runtimeMaintenanceStore.UpdateProcessRefStatus(ctx, refID, status, endedAt)
}

var errInjectedDemandFinalization = errors.New("injected demand finalization failure")

type failDemandFinalizationStore struct{ runtimeMaintenanceStore }

func (s *failDemandFinalizationStore) CompleteDemandStop(context.Context, string, int64) error {
	return errInjectedDemandFinalization
}
