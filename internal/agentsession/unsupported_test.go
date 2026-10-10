//go:build !linux

package agentsession

import (
	"context"
	"errors"
	"testing"

	platformgo "github.com/vrooli/platform-go"
)

// Off Linux no process reader or stop primitive is host-verified: each says
// unsupported instead of answering, so owners stay unknown (never reaped) and
// a reap reports unsupported instead of believing a tree is gone.
func TestNonLinuxReadersAndStopperReportUnsupported(t *testing.T) {
	if _, err := processStartTime(1); !errors.Is(err, platformgo.ErrUnsupported) {
		t.Fatalf("processStartTime = %v", err)
	}
	if _, err := desktopAppProcesses("/data"); !errors.Is(err, platformgo.ErrUnsupported) {
		t.Fatalf("desktopAppProcesses = %v", err)
	}
	session := Session{SessionID: "s", Scope: "vrooli-agent-s"}
	if err := NewPlatformStopper().Stop(context.Background(), session); !errors.Is(err, platformgo.ErrUnsupported) {
		t.Fatalf("PlatformStopper.Stop = %v, want ErrUnsupported", err)
	}
}

func TestNonLinuxProcessMemoryIsUnsupported(t *testing.T) {
	if _, _, err := processMemory(1); !errors.Is(err, platformgo.ErrUnsupported) {
		t.Fatalf("processMemory = %v", err)
	}
}
