//go:build !windows

package privsep_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	provisionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/provision"
	"vrooli-bridge/agent/internal/privsep"
)

func TestIPCProvisionRoundTripUsesTypedEvents(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the production Windows service uses its native IPC adapter")
	}
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
	if runtime.GOOS == "windows" {
		t.Skip("the production Windows service uses its native IPC adapter")
	}
	socket := filepath.Join(t.TempDir(), "provision.sock")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serverErr := make(chan error, 1)
	go func() { serverErr <- privsep.Serve(ctx, socket, "vrooli", t.TempDir(), os.Getuid()+1) }()
	waitForSocket(t, socket)
	err := privsep.Run(ctx, socket, -1, &channelv1.ProvisionCommand{OpId: "rejected", TargetRevision: "abc"}, func(*provisionv1.ProvisionEvent) error { return nil })
	require.Error(t, err)
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
