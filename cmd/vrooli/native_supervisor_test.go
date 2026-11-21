package main

import (
	"bytes"
	"testing"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/runtimesupervisor"
)

func TestNativeSupervisorBootstrapAcceptsOnlyFixedStampedNativeInvocation(t *testing.T) {
	old := buildinfo.RuntimeHomeSnapshotBase64
	t.Cleanup(func() { buildinfo.RuntimeHomeSnapshotBase64 = old })
	t.Setenv(runtimesupervisor.ModeEnv, runtimesupervisor.ModeOn)
	buildinfo.RuntimeHomeSnapshotBase64 = "invalid-compiled-authority"
	for _, args := range [][]string{{"runtime", "supervisor", "status"}, {"runtime", "supervisor", "run", "--takeover"}, {"scenario", "restart", "agent-manager"}} {
		var out bytes.Buffer
		if handled, _ := runNativeSupervisorCommand(args, &out); handled || out.Len() != 0 {
			t.Fatalf("native bootstrap intercepted ordinary invocation: %v", args)
		}
	}
	var out bytes.Buffer
	if handled, code := runNativeSupervisorCommand([]string{"runtime", "supervisor", "run"}, &out); !handled || code != 1 {
		t.Fatal("invalid compiled authority did not refuse fixed native startup")
	}
	if bytes.Contains(out.Bytes(), []byte(buildinfo.RuntimeHomeSnapshotBase64)) {
		t.Fatal("compiled payload leaked into diagnostic")
	}
	buildinfo.RuntimeHomeSnapshotBase64 = ""
	out.Reset()
	if handled, _ := runNativeSupervisorCommand([]string{"runtime", "supervisor", "run"}, &out); handled {
		t.Fatal("unstamped development invocation claimed source-independent bootstrap")
	}
}
