//go:build windows

package sysmounts

import (
	"context"
	"testing"
)

func TestPlatformDeviceByIdentityUsesPowerShellUniqueID(t *testing.T) {
	previous := runPowerShell
	t.Cleanup(func() { runPowerShell = previous })
	runPowerShell = func(context.Context, string) ([]byte, error) {
		return []byte(`[{"DriveLetter":"E","FileSystem":"NTFS","FileSystemLabel":"Elements","UniqueId":"\\\\?\\Volume{12345678-1234-1234-1234-123456789abc}\\","Size":42}]`), nil
	}
	got, err := platformDeviceByIdentity(context.Background(), `\\?\Volume{12345678-1234-1234-1234-123456789abc}\`, "")
	if err != nil {
		t.Fatalf("platformDeviceByIdentity: %v", err)
	}
	if got.DevicePath != got.UUID || got.UUID == "" || got.Mountpoint != `E:\` {
		t.Fatalf("got %+v", got)
	}
}

func TestPlatformDeviceByIdentityDoesNotInventMountpointWithoutDriveLetter(t *testing.T) {
	previous := runPowerShell
	t.Cleanup(func() { runPowerShell = previous })
	runPowerShell = func(context.Context, string) ([]byte, error) {
		return []byte(`{"DriveLetter":null,"FileSystem":"NTFS","FileSystemLabel":"Elements","UniqueId":"\\\\?\\Volume{12345678-1234-1234-1234-123456789abc}\\","Size":42}`), nil
	}
	got, err := platformDeviceByIdentity(context.Background(), `\\?\Volume{12345678-1234-1234-1234-123456789abc}\`, "")
	if err != nil {
		t.Fatalf("platformDeviceByIdentity: %v", err)
	}
	if got.Mountpoint != "" || got.DevicePath != got.UUID {
		t.Fatalf("unlettered volume got synthetic path/mountpoint: %+v", got)
	}
	if got.UUID == "" {
		t.Fatalf("stable identity was lost: %+v", got)
	}
}
