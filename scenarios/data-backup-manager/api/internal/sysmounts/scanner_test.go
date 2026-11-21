package sysmounts

import (
	"context"
	"testing"

	"github.com/shirou/gopsutil/v3/disk"
)

func TestScanKeepsNTFSFilesystemIdentityForFuseblkMount(t *testing.T) {
	device := "/dev/sdz1"
	mountpoint := "/media/Elements"
	scanner := &Scanner{
		partitions: func(context.Context, bool) ([]disk.PartitionStat, error) {
			return []disk.PartitionStat{{Device: device, Mountpoint: mountpoint, Fstype: "fuseblk", Opts: []string{"rw"}}}, nil
		},
		usage: func(context.Context, string) (*disk.UsageStat, error) {
			return &disk.UsageStat{Free: 50, Total: 100}, nil
		},
		classifier: linuxClassifier(t, map[string]string{"sdz": "1"}),
		state:      newStateProber(),
		identity: func(context.Context, string) (VolumeIdentity, error) {
			return VolumeIdentity{Filesystem: "ntfs", UUID: "uuid-elements", Serial: "serial-elements"}, nil
		},
	}

	volumes, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(volumes) != 1 {
		t.Fatalf("volumes = %+v, want one mounted volume", volumes)
	}
	if volumes[0].Filesystem != "ntfs" || volumes[0].MountDriver != "fuseblk" {
		t.Fatalf("filesystem/driver = %q/%q, want ntfs/fuseblk", volumes[0].Filesystem, volumes[0].MountDriver)
	}
}
