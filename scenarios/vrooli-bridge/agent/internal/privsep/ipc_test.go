//go:build !windows

package privsep_test

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"vrooli-bridge/agent/internal/privsep"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	cleanupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/cleanup"
	provisionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/provision"
	"github.com/vrooli/vrooli/packages/proto/privilegedops"
	"github.com/vrooli/vrooli/packages/proto/sealing"
)

func TestIPCProvisionRoundTripUsesTypedEvents(t *testing.T) {
	workDir := t.TempDir()
	runGit(t, workDir, "init")
	runGit(t, workDir, "config", "user.email", "bridge-test@example.invalid")
	runGit(t, workDir, "config", "user.name", "Bridge Test")
	marker := filepath.Join(workDir, "marker")
	require.NoError(t, os.WriteFile(marker, []byte("ok"), 0o600))
	runGit(t, workDir, "add", "marker")
	runGit(t, workDir, "commit", "-m", "initial")
	revision := strings.TrimSpace(string(runGit(t, workDir, "rev-parse", "HEAD")))

	fakeVrooli := filepath.Join(t.TempDir(), "vrooli")
	require.NoError(t, os.WriteFile(fakeVrooli, []byte("#!/bin/sh\nexit 0\n"), 0o700))
	socket := filepath.Join(t.TempDir(), "run", "provision.sock")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serverErr := make(chan error, 1)
	go func() { serverErr <- privsep.Serve(ctx, socket, fakeVrooli, workDir, os.Getuid()) }()
	waitForSocket(t, socket)

	var events []*provisionv1.ProvisionEvent
	err := privsep.Run(ctx, socket, os.Getuid(), &channelv1.ProvisionCommand{OpId: "ipc-1", TargetRevision: revision}, func(event *provisionv1.ProvisionEvent) error {
		events = append(events, event)
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, events)
	require.Equal(t, provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_EXIT, events[len(events)-1].GetKind())
	require.Zero(t, events[len(events)-1].GetExitCode())
	require.Equal(t, "ipc-1", events[len(events)-1].GetOpId())
	cancel()
	select {
	case err := <-serverErr:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("provisioning helper did not stop after context cancellation")
	}
}

func TestIPCRejectsWrongRunnerPeer(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "provision.sock")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serverErr := make(chan error, 1)
	go func() { serverErr <- privsep.Serve(ctx, socket, "vrooli", t.TempDir(), os.Getuid()+1) }()
	waitForSocket(t, socket)
	err := privsep.Run(ctx, socket, -1, &channelv1.ProvisionCommand{OpId: "rejected", TargetRevision: "abc"}, func(*provisionv1.ProvisionEvent) error { return nil })
	require.Error(t, err)
}

func TestIPCCleanupRoundTripsEveryNamedOperation(t *testing.T) {
	stateDir := t.TempDir()
	nodePrivate, err := ecdh.X25519().GenerateKey(rand.Reader)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "encryption.key"), nodePrivate.Bytes(), 0o600))
	fakeVrooli := filepath.Join(t.TempDir(), "vrooli")
	require.NoError(t, os.WriteFile(fakeVrooli, []byte("#!/bin/sh\nprintf '{\"ok\":true,\"complete\":true,\"plan_hash\":\"frozen\"}\\n'\n"), 0o700))
	socket := filepath.Join(t.TempDir(), "run", "cleanup.sock")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serverErr := make(chan error, 1)
	go func() { serverErr <- privsep.Serve(ctx, socket, fakeVrooli, t.TempDir(), os.Getuid(), stateDir) }()
	waitForSocket(t, socket)

	public := nodePrivate.PublicKey().Bytes()
	operations := []channelv1.PrivilegedOperation{
		channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_INVENTORY_INSTALLATION,
		channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_PLAN_UNINSTALL,
		channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_PROVISION_BREAK_GLASS,
		channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_ISSUE_CLEANUP_CAPABILITY,
		channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_APPLY_FROZEN_PLAN,
		channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_VERIFY_RESULT,
		channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_ROTATE_BREAK_GLASS,
	}
	for _, operation := range operations {
		t.Run(privilegedops.Name(operation), func(t *testing.T) {
			cmd := &channelv1.CleanupCommand{
				Operation: operation, OpId: "ipc-" + privilegedops.Name(operation), PlanId: "plan-1",
				MachineId: "machine-1", NodeId: "node-1", Target: "mini.local", Scope: "all",
				PlanHash: "hash-1", OperatorId: "operator-1", OperatorConfirmed: true,
			}
			if operation == channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_APPLY_FROZEN_PLAN {
				cmd.Capability = []byte("opaque-capability")
			} else if operation != channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_INVENTORY_INSTALLATION && operation != channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_PLAN_UNINSTALL && operation != channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_VERIFY_RESULT {
				cmd.SealedPassphrase, err = sealing.Seal(public, []byte("correct horse"), sealing.Context(cmd.MachineId, cmd.NodeId, cmd.Target, cmd.Scope, cmd.PlanHash, cmd.PlanId, cmd.OperatorId))
				require.NoError(t, err)
			}
			var events []*cleanupv1.CleanupEvent
			require.NoError(t, privsep.RunCleanup(ctx, socket, os.Getuid(), cmd, func(event *cleanupv1.CleanupEvent) error {
				events = append(events, event)
				return nil
			}))
			require.NotEmpty(t, events)
			last := events[len(events)-1]
			require.Equal(t, cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_EXIT, last.GetKind())
			require.Zero(t, last.GetExitCode())
			require.Equal(t, cmd.GetOpId(), last.GetOperationId())
		})
	}
	cancel()
	select {
	case err := <-serverErr:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("cleanup helper did not stop after context cancellation")
	}
}

func TestPrivilegedOperationAdmissionIsClosedAndNamesUnknownOperations(t *testing.T) {
	for name := range privsep.SupportedOperations {
		t.Run("accepts_"+name, func(t *testing.T) {
			require.NoError(t, privsep.ValidateOperationName(name))
		})
	}
	for _, name := range []string{"", "remove-everything", "provision --shell", "future-operation"} {
		t.Run("rejects_"+strings.ReplaceAll(name, " ", "_"), func(t *testing.T) {
			err := privsep.ValidateOperationName(name)
			require.Error(t, err)
			require.Contains(t, err.Error(), name)
		})
	}
}

func waitForSocket(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if info, err := os.Stat(path); err == nil && info.Mode()&os.ModeSocket != 0 {
			return
		}
		time.Sleep(time.Millisecond * 5)
	}
	t.Fatalf("IPC socket %q was not created", path)
}

func runGit(t *testing.T, dir string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", out)
	return out
}
