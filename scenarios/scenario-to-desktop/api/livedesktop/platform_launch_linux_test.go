//go:build linux

package livedesktop

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestLinuxLaunchReportsEarlyFailure(t *testing.T) {
	backend := &LinuxBackend{}
	process, err := backend.LaunchApp(context.Background(), &mockDisplay{}, "/bin/sh", LaunchOptions{ExtraArgs: []string{"-c", "echo expected-startup-failure >&2; exit 3"}})
	if process != nil || err == nil || !strings.Contains(err.Error(), "expected-startup-failure") {
		t.Fatalf("process=%v err=%v", process, err)
	}
}

func TestLinuxLaunchReapsLaterExit(t *testing.T) {
	backend := &LinuxBackend{}
	process, err := backend.LaunchApp(context.Background(), &mockDisplay{}, "/bin/sleep", LaunchOptions{ExtraArgs: []string{"0.3"}})
	if err != nil {
		t.Fatal(err)
	}
	native := process.(*linuxProcess)
	select {
	case <-native.done:
	case <-time.After(2 * time.Second):
		t.Fatal("process was not reaped")
	}
	if native.IsRunning() {
		t.Fatal("exited app reported running")
	}
	backend.KillApp(native)
}
