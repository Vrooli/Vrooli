//go:build windows

package platform

import (
	"errors"
	"fmt"
	"testing"
)

func TestTaskHRESULTAbsenceIsTypedAndWrapped(t *testing.T) {
	for _, code := range []windowsTaskHRESULT{0x80070002} {
		if !taskNativeErrorMissing(fmt.Errorf("native query: %w", code)) {
			t.Fatal("missing task not recognized")
		}
	}
	if taskNativeErrorMissing(errors.New("text 0x80070002")) || taskNativeErrorMissing(windowsTaskHRESULT(0x80070005)) || taskNativeErrorMissing(fmt.Errorf("account: %w", windowsTaskHRESULT(0x8004130f))) {
		t.Fatal("unproven absence accepted")
	}
}

func TestWindowsUserModeNeverFallsBackToLocalSystem(t *testing.T) {
	if _, e := installService(ServiceInstallOptions{User: true}); !errors.Is(e, ErrUnsupported) {
		t.Fatalf("user install reached system path: %v", e)
	}
	if _, e := startInstalledService(ServiceInstallOptions{User: true}); !errors.Is(e, ErrUnsupported) {
		t.Fatalf("user start reached system path: %v", e)
	}
	if _, e := uninstallService(ServiceInstallOptions{User: true}); !errors.Is(e, ErrUnsupported) {
		t.Fatalf("user removal reached system path: %v", e)
	}
}
