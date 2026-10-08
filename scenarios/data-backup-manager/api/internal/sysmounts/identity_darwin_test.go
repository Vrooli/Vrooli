//go:build darwin

package sysmounts

import (
	"context"
	"testing"
)

func TestPlatformDeviceByIdentityUsesDiskutilUUID(t *testing.T) {
	previous := runDiskutil
	t.Cleanup(func() { runDiskutil = previous })
	runDiskutil = func(context.Context, ...string) ([]byte, error) {
		return []byte("Device Node: /dev/disk2s1\nVolume UUID: uuid-1\nVolume Name: Elements\nFile System Personality: APFS\n"), nil
	}
	got, err := platformDeviceByIdentity(context.Background(), "uuid-1", "")
	if err != nil {
		t.Fatalf("platformDeviceByIdentity: %v", err)
	}
	if got.DevicePath != "/dev/disk2s1" || got.UUID != "uuid-1" {
		t.Fatalf("got %+v", got)
	}
}
