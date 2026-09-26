//go:build linux

package sysmounts

import (
	"context"
	"testing"
)

func TestPlatformDeviceByIdentityEnrichesPartitionFromWholeDisk(t *testing.T) {
	previous := runLSBlk
	t.Cleanup(func() { runLSBlk = previous })
	runLSBlk = func(context.Context, ...string) ([]byte, error) {
		return []byte(`PATH="/dev/sda" LABEL="" UUID="" MODEL="Elements" SERIAL="WD-123" FSTYPE="" SIZE="500000"` + "\n" +
			`PATH="/dev/sda1" LABEL="Elements" UUID="uuid-1" MODEL="" SERIAL="" FSTYPE="ntfs" SIZE="499000"` + "\n"), nil
	}

	got, err := platformDeviceByIdentity(context.Background(), "uuid-1", "WD-123")
	if err != nil {
		t.Fatalf("platformDeviceByIdentity: %v", err)
	}
	if got.DevicePath != "/dev/sda1" || got.Serial != "WD-123" || got.Model != "Elements" {
		t.Fatalf("identity enrichment = %+v", got)
	}
}

func TestWholeDiskDeviceHandlesLinuxPartitionNames(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"/dev/sda1", "/dev/sda"},
		{"/dev/nvme0n1p2", "/dev/nvme0n1"},
	} {
		if got := wholeDiskDevice(tc.in); got != tc.want {
			t.Errorf("wholeDiskDevice(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
