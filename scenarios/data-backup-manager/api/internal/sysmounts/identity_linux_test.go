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

// A partition row carries no MODEL/SERIAL, and `lsblk <disk>` prints the
// partitions after the disk row. Reading every row in order let the partition
// blank the disk's anchors, so a destination that recorded a serial failed its
// identity re-check with "not served by the verified mounted volume" even
// though the UUID matched and the volume was mounted.
func TestPlatformVolumeIdentityKeepsWholeDiskAnchors(t *testing.T) {
	previous := runLSBlk
	t.Cleanup(func() { runLSBlk = previous })
	runLSBlk = func(_ context.Context, args ...string) ([]byte, error) {
		device := args[len(args)-1]
		if device == "/dev/sda" {
			// `lsblk -d` is what the fallback must ask for; without it the
			// partition row follows and blanks SERIAL/MODEL.
			diskRow := `LABEL="" UUID="" MODEL="WDC WD20JDRW-11C7VS1" SERIAL="WD-WX52A946D6VL" FSTYPE=""` + "\n"
			partitionRow := `LABEL="Elements" UUID="E26A883E6A881189" MODEL="" SERIAL="" FSTYPE="ntfs"` + "\n"
			if len(args) > 0 && args[0] == "-d" {
				return []byte(diskRow), nil
			}
			return []byte(diskRow + partitionRow), nil
		}
		return []byte(`LABEL="Elements" UUID="E26A883E6A881189" MODEL="" SERIAL="" FSTYPE="ntfs"` + "\n"), nil
	}

	got, err := platformVolumeIdentity(context.Background(), "/dev/sda1")
	if err != nil {
		t.Fatalf("platformVolumeIdentity: %v", err)
	}
	if got.UUID != "E26A883E6A881189" || got.Serial != "WD-WX52A946D6VL" || got.Model != "WDC WD20JDRW-11C7VS1" {
		t.Fatalf("identity lost whole-disk anchors: %+v", got)
	}
}
