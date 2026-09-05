package sysmounts

import (
	"os"
	"path/filepath"
	"testing"
)

// writeSysBlock builds a fake /sys/block tree: each entry maps a whole-disk
// device name to its `removable` flag ("1" or "0").
func writeSysBlock(t *testing.T, devices map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for dev, removable := range devices {
		dir := filepath.Join(root, dev)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "removable"), []byte(removable+"\n"), 0o644); err != nil {
			t.Fatalf("write removable: %v", err)
		}
	}
	return root
}

func linuxClassifier(t *testing.T, devices map[string]string) *classifier {
	t.Helper()
	return &classifier{goos: "linux", sysBlockRoot: writeSysBlock(t, devices)}
}

func TestClassifyLinuxRemovableViaSysfs(t *testing.T) {
	c := linuxClassifier(t, map[string]string{"sda": "0", "sdb": "1", "nvme0n1": "0"})

	cases := []struct {
		name      string
		mount     mountInfo
		wantClass DriveClass
		wantRem   bool
	}{
		{
			name:      "internal sata disk is fixed",
			mount:     mountInfo{Device: "/dev/sda1", Mountpoint: "/", Fstype: "ext4"},
			wantClass: ClassFixed,
			wantRem:   false,
		},
		{
			name:      "usb stick flagged removable in sysfs",
			mount:     mountInfo{Device: "/dev/sdb1", Mountpoint: "/data", Fstype: "ext4"},
			wantClass: ClassRemovable,
			wantRem:   true,
		},
		{
			name:      "nvme partition resolves to whole-disk base",
			mount:     mountInfo{Device: "/dev/nvme0n1p2", Mountpoint: "/home", Fstype: "ext4"},
			wantClass: ClassFixed,
			wantRem:   false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotClass, gotRem := c.classify(tc.mount)
			if gotClass != tc.wantClass || gotRem != tc.wantRem {
				t.Fatalf("classify(%+v) = (%q, %v), want (%q, %v)", tc.mount, gotClass, gotRem, tc.wantClass, tc.wantRem)
			}
		})
	}
}

func TestClassifyLinuxRemovableViaMountpoint(t *testing.T) {
	// Even when sysfs says fixed (or the device is unknown), a /media or
	// /run/media mountpoint is treated as removable-candidate.
	c := linuxClassifier(t, map[string]string{"sdb": "0"})

	for _, mp := range []string{"/media/alice/USB", "/run/media/alice/STICK"} {
		gotClass, gotRem := c.classify(mountInfo{Device: "/dev/sdb1", Mountpoint: mp, Fstype: "vfat"})
		if gotClass != ClassRemovable || !gotRem {
			t.Fatalf("mountpoint %q: got (%q, %v), want removable", mp, gotClass, gotRem)
		}
	}
}

func TestClassifyNetworkFilesystem(t *testing.T) {
	c := linuxClassifier(t, nil)
	for _, fs := range []string{"nfs", "nfs4", "cifs", "smb3", "fuse.sshfs"} {
		gotClass, gotRem := c.classify(mountInfo{Device: "server:/export", Mountpoint: "/mnt/net", Fstype: fs})
		if gotClass != ClassNetwork || gotRem {
			t.Fatalf("fstype %q: got (%q, %v), want network/non-removable", fs, gotClass, gotRem)
		}
	}
}

func TestIsPseudoFS(t *testing.T) {
	c := newClassifier()
	pseudo := []string{"proc", "sysfs", "tmpfs", "cgroup2", "overlay", "devtmpfs", "DEVPTS"}
	for _, fs := range pseudo {
		if !c.isPseudoFS(fs) {
			t.Errorf("expected %q to be pseudo", fs)
		}
	}
	real := []string{"ext4", "apfs", "ntfs", "xfs", "btrfs", "vfat", "nfs4"}
	for _, fs := range real {
		if c.isPseudoFS(fs) {
			t.Errorf("expected %q to be a real filesystem", fs)
		}
	}
}

func TestIsReadOnly(t *testing.T) {
	if !isReadOnly([]string{"nosuid", "ro", "relatime"}) {
		t.Error("expected ro mount to be read-only")
	}
	if isReadOnly([]string{"rw", "relatime"}) {
		t.Error("expected rw mount to be writable")
	}
}

func TestFilesystemStateUsesOnlyExplicitMountSignals(t *testing.T) {
	cases := []struct {
		name string
		opts []string
		want FilesystemState
	}{
		{name: "unknown when not exposed", opts: []string{"rw", "relatime"}, want: FilesystemStateUnknown},
		{name: "clean", opts: []string{"rw", "state=clean"}, want: FilesystemStateClean},
		{name: "dirty", opts: []string{"rw", "dirty"}, want: FilesystemStateDirty},
		{name: "needs check", opts: []string{"rw", "needs-check"}, want: FilesystemStateNeedsCheck},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := filesystemState(tc.opts); got != tc.want {
				t.Fatalf("filesystemState(%v) = %q, want %q", tc.opts, got, tc.want)
			}
		})
	}
}

func TestDarwinRemovableUnderVolumes(t *testing.T) {
	c := &classifier{goos: "darwin", sysBlockRoot: "/sys/block"}
	if class, rem := c.classify(mountInfo{Mountpoint: "/Volumes/Backup", Fstype: "apfs"}); class != ClassRemovable || !rem {
		t.Fatalf("/Volumes mount: got (%q, %v), want removable", class, rem)
	}
	if class, rem := c.classify(mountInfo{Mountpoint: "/", Fstype: "apfs"}); class != ClassFixed || rem {
		t.Fatalf("root mount: got (%q, %v), want fixed", class, rem)
	}
}

func TestWindowsDriveTypeClassifiesRemovableAndFixed(t *testing.T) {
	c := &classifier{goos: "windows", driveType: func(root string) uint32 {
		if root == `E:\` {
			return driveTypeRemovable
		}
		return driveTypeFixed
	}}
	if class, removable := c.classify(mountInfo{Mountpoint: `E:\\Backups`, Fstype: "ntfs"}); class != ClassRemovable || !removable {
		t.Fatalf("removable drive: got (%q, %v)", class, removable)
	}
	if class, removable := c.classify(mountInfo{Mountpoint: `C:\\`, Fstype: "ntfs"}); class != ClassFixed || removable {
		t.Fatalf("fixed drive: got (%q, %v)", class, removable)
	}
}
