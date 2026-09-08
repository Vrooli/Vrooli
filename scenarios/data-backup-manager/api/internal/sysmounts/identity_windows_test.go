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
		return []byte(`[{"DriveLetter":"E","FileSystem":"NTFS","FileSystemLabel":"Elements","UniqueId":"\\\\?\\Volume{uuid-1}\\","Size":42}]`), nil
	}
	got, err := platformDeviceByIdentity(context.Background(), `\\?\Volume{uuid-1}\`, "")
	if err != nil {
		t.Fatalf("platformDeviceByIdentity: %v", err)
	}
	if got.DevicePath != "E:" || got.UUID == "" {
		t.Fatalf("got %+v", got)
	}
}
