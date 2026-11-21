package volumeremediation

import (
	"strings"
	"testing"
)

// Argv construction is tested on every platform, including the ones whose tools
// cannot run here: a command this table will not build is a command that can
// never run, so the table is the contract.
func TestCommandTable(t *testing.T) {
	cases := []struct {
		name    string
		backend Backend
		req     Request
		state   State
		want    []string
	}{
		{
			name:    "udisks unmount",
			backend: BackendUDisks,
			req:     Request{Action: ActionUnmount, Device: elementsDevice()},
			want:    []string{"udisksctl", "unmount", "--block-device", "/dev/sda1", "--no-user-interaction"},
		},
		{
			name:    "udisks NTFS mount selects the ntfs helper",
			backend: BackendUDisks,
			req:     Request{Action: ActionMountReadWrite, Device: elementsDevice()},
			want:    []string{"udisksctl", "mount", "--block-device", "/dev/sda1", "--filesystem-type", "ntfs", "--no-user-interaction"},
		},
		{
			name:    "udisks check",
			backend: BackendUDisks,
			req:     Request{Action: ActionCheck, Device: elementsDevice()},
			want:    []string{"busctl", "call", "org.freedesktop.UDisks2", "/org/freedesktop/UDisks2/block_devices/sda1", "org.freedesktop.UDisks2.Filesystem", "Check", "a{sv}", "0"},
		},
		{
			name:    "udisks repair",
			backend: BackendUDisks,
			req:     Request{Action: ActionRepair, Device: elementsDevice()},
			want:    []string{"busctl", "call", "org.freedesktop.UDisks2", "/org/freedesktop/UDisks2/block_devices/sda1", "org.freedesktop.UDisks2.Filesystem", "Repair", "a{sv}", "0"},
		},
		{
			name:    "native ntfs check does not write",
			backend: BackendNativeTools,
			req:     Request{Action: ActionCheck, Device: elementsDevice()},
			want:    []string{"ntfsfix", "-n", "/dev/sda1"},
		},
		{
			name:    "native ntfs repair clears the dirty flag",
			backend: BackendNativeTools,
			req:     Request{Action: ActionRepair, Device: elementsDevice()},
			want:    []string{"ntfsfix", "-d", "/dev/sda1"},
		},
		{
			name:    "native ext check does not write",
			backend: BackendNativeTools,
			req:     Request{Action: ActionCheck, Device: Device{Path: "/dev/sdb1", Filesystem: "ext4"}},
			want:    []string{"e2fsck", "-f", "-n", "/dev/sdb1"},
		},
		{
			name:    "native ext repair",
			backend: BackendNativeTools,
			req:     Request{Action: ActionRepair, Device: Device{Path: "/dev/sdb1", Filesystem: "ext4"}},
			want:    []string{"e2fsck", "-f", "-y", "/dev/sdb1"},
		},
		{
			name:    "native exfat repair",
			backend: BackendNativeTools,
			req:     Request{Action: ActionRepair, Device: Device{Path: "/dev/sdb1", Filesystem: "exfat"}},
			want:    []string{"fsck.exfat", "-y", "/dev/sdb1"},
		},
		{
			name:    "native vfat repair",
			backend: BackendNativeTools,
			req:     Request{Action: ActionRepair, Device: Device{Path: "/dev/sdb1", Filesystem: "vfat"}},
			want:    []string{"fsck.fat", "-a", "/dev/sdb1"},
		},
		{
			name:    "native unmount uses the observed mountpoint",
			backend: BackendNativeTools,
			req:     Request{Action: ActionUnmount, Device: elementsDevice()},
			state:   State{Device: Device{Mountpoint: "/media/user/Elements"}},
			want:    []string{"umount", "/media/user/Elements"},
		},
		{
			name:    "native mount",
			backend: BackendNativeTools,
			req:     Request{Action: ActionMountReadWrite, Device: elementsDevice(), DesiredMountpoint: "/media/user/Elements"},
			want:    []string{"mount", "-t", "ntfs-3g", "-o", "rw", "/dev/sda1", "/media/user/Elements"},
		},
		{
			name:    "darwin verify",
			backend: BackendDiskutil,
			req:     Request{Action: ActionCheck, Device: Device{Path: "/dev/disk2s1", Filesystem: "exfat"}},
			want:    []string{"diskutil", "verifyVolume", "/dev/disk2s1"},
		},
		{
			name:    "darwin repair",
			backend: BackendDiskutil,
			req:     Request{Action: ActionRepair, Device: Device{Path: "/dev/disk2s1", Filesystem: "exfat"}},
			want:    []string{"diskutil", "repairVolume", "/dev/disk2s1"},
		},
		{
			name:    "windows scan",
			backend: BackendRepairVolume,
			req:     Request{Action: ActionCheck, Device: Device{Path: "E:", UUID: `\\?\Volume{12345678-1234-1234-1234-123456789abc}\`, Filesystem: "ntfs"}},
			state:   State{Device: Device{Path: "E:", UUID: `\\?\Volume{12345678-1234-1234-1234-123456789abc}\`}},
		},
		{
			name:    "windows repair",
			backend: BackendRepairVolume,
			req:     Request{Action: ActionRepair, Device: Device{Path: `\\?\Volume{12345678-1234-1234-1234-123456789abc}\`, UUID: `\\?\Volume{12345678-1234-1234-1234-123456789abc}\`, Filesystem: "ntfs"}},
			state:   State{Device: Device{UUID: `\\?\Volume{12345678-1234-1234-1234-123456789abc}\`}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := commandFor(tc.backend, tc.req, tc.state)
			if err != nil {
				t.Fatalf("commandFor: %v", err)
			}
			if len(tc.want) > 0 && strings.Join(got, "\x00") != strings.Join(tc.want, "\x00") {
				t.Fatalf("argv = %q, want %q", got, tc.want)
			}
			if tc.name == "windows scan" && (!strings.Contains(got[4], "Get-Volume -UniqueId") || !strings.Contains(got[4], "DBM_CHECK_RESULT=")) {
				t.Fatalf("scan command is not identity-bound with an explicit result: %q", got[4])
			}
			if tc.name == "windows repair" && !strings.Contains(got[4], "Repair-Volume -InputObject $v -OfflineScanAndFix") {
				t.Fatalf("repair command is not bound to the resolved volume object: %q", got[4])
			}
		})
	}
}

func TestWindowsMountRestoresPriorLetterByStableID(t *testing.T) {
	volumeID := `\\?\Volume{12345678-1234-1234-1234-123456789abc}\`
	got, err := commandFor(BackendRepairVolume, Request{
		Action: ActionMountReadWrite, Device: Device{Path: volumeID, UUID: volumeID}, DesiredMountpoint: `E:\`,
	}, State{Device: Device{UUID: volumeID}})
	if err != nil {
		t.Fatalf("commandFor: %v", err)
	}
	script := got[4]
	for _, required := range []string{"Get-Volume -UniqueId", "already has a drive letter", "requested drive letter is already assigned", "already has a mounted-folder access path", "Add-PartitionAccessPath", "-AccessPath 'E:'"} {
		if !strings.Contains(script, required) {
			t.Fatalf("mount script %q is missing %q", script, required)
		}
	}
}

func TestWindowsRequiresADriveLetter(t *testing.T) {
	if _, err := commandFor(BackendRepairVolume, Request{Action: ActionRepair, Device: Device{Path: "/dev/sda1", Filesystem: "ntfs"}}, State{}); err == nil {
		t.Fatal("expected a validation error for a non-drive-letter device")
	}
}

// A native mount must not be redirected outside the permitted media roots.
func TestNativeMountRefusesATargetOutsideMediaRoots(t *testing.T) {
	_, err := commandFor(BackendNativeTools, Request{
		Action:            ActionMountReadWrite,
		Device:            elementsDevice(),
		DesiredMountpoint: "/home/user/secret",
	}, State{})
	if err == nil {
		t.Fatal("expected a refusal for a mountpoint outside the media roots")
	}
}

func asUnsupported(err error, target *ErrUnsupported) bool {
	if unsupported, ok := err.(ErrUnsupported); ok {
		*target = unsupported
		return true
	}
	return false
}
