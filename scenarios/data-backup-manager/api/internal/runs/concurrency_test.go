package runs

import (
	"testing"

	"data-backup-manager/internal/destinationreadiness"
)

func TestRequiresVolumeLeaseOnlyForStrongFilesystemIdentity(t *testing.T) {
	strong := &destinationreadiness.DeviceIdentity{DevicePath: "/dev/sda1", UUID: "uuid-1", Mountpoint: "/media/backup", Filesystem: "ext4"}
	cases := []struct {
		name string
		dest DestinationForRun
		want bool
	}{
		{
			name: "strong filesystem identity is lease-bound",
			dest: DestinationForRun{BackendKind: "filesystem", DeviceIdentity: strong},
			want: true,
		},
		{
			name: "legacy path has no identity to lease",
			dest: DestinationForRun{BackendKind: "filesystem"},
			want: false,
		},
		{
			name: "identity without a stable anchor is not lease-bound",
			dest: DestinationForRun{BackendKind: "filesystem", DeviceIdentity: &destinationreadiness.DeviceIdentity{DevicePath: "/dev/sda1", Mountpoint: "/media/backup"}},
			want: false,
		},
		{
			name: "object storage does not use the host volume lease",
			dest: DestinationForRun{BackendKind: "s3", DeviceIdentity: strong},
			want: false,
		},
	}
	for _, tc := range cases {
		if got := requiresVolumeLease(tc.dest); got != tc.want {
			t.Fatalf("%s: requiresVolumeLease = %v, want %v", tc.name, got, tc.want)
		}
	}
}
