package sysmounts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureProber builds a prober over a temp tree so the Linux evidence paths
// are exercised on any host, matching linuxClassifier's approach.
func fixtureProber(t *testing.T, files map[string]string) *stateProber {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	return &stateProber{
		goos:          "linux",
		sysClassBlock: filepath.Join(root, "sys", "class", "block"),
		procFS:        filepath.Join(root, "proc", "fs"),
		fstabPath:     filepath.Join(root, "etc", "fstab"),
		readFile:      os.ReadFile,
	}
}

// The Elements incident: ntfs3 refuses a read/write mount because the NTFS
// dirty flag is set. The mount options say only `ro`, so mount-option parsing
// alone reports unknown and the operator is told nothing actionable.
func TestProbeAttributesReadOnlyToDirtyNTFSVolume(t *testing.T) {
	p := fixtureProber(t, map[string]string{
		"sys/class/block/sda1/ro":    "0\n",
		"sys/class/block/sda/ro":     "0\n",
		"proc/fs/ntfs3/sda1/volinfo": "ntfs3.1\n4096\n488369919\n19712\n19274\ndirty\ndirty\n",
	})
	m := mountInfo{Device: "/dev/sda1", Mountpoint: "/media/user/Elements", Fstype: "ntfs3", Opts: []string{"ro", "nosuid", "relatime"}}

	got := p.probe(m, true)

	if got.FilesystemState != FilesystemStateDirty {
		t.Fatalf("filesystem state = %q, want %q", got.FilesystemState, FilesystemStateDirty)
	}
	if got.ReadOnlyCause != CauseFilesystemDirty {
		t.Fatalf("cause = %q, want %q", got.ReadOnlyCause, CauseFilesystemDirty)
	}
	if got.DeviceWriteProtected {
		t.Fatal("device must not be reported write-protected when sysfs ro=0")
	}
	if got.EvidenceSource == "" || got.EvidenceSource == "mount-options" {
		t.Fatalf("evidence source = %q, want the driver-published volinfo path", got.EvidenceSource)
	}
}

// NTFS marks the volume dirty while it is mounted read/write, so on such a
// mount the flag is ambiguous: it may be this mount's in-use marker, or damage
// that predates it. The probe must not resolve that ambiguity in either
// direction. Reporting dirty would mark every healthy NTFS drive as needing
// repair; reporting clean asserts a health this evidence cannot establish, and
// is what let a genuinely dirty volume serve backups until the host panicked on
// 2026-08-19.
func TestProbeCannotDistinguishTheDirtyFlagOnAReadWriteMount(t *testing.T) {
	p := fixtureProber(t, map[string]string{
		"sys/class/block/sda1/ro":    "0\n",
		"proc/fs/ntfs3/sda1/volinfo": "ntfs3.1\n4096\n488369919\n19712\n19274\nclean\ndirty\n",
	})
	m := mountInfo{Device: "/dev/sda1", Mountpoint: "/media/user/Elements", Fstype: "ntfs3", Opts: []string{"rw", "relatime"}}

	got := p.probe(m, false)

	if got.FilesystemState != FilesystemStateUnknown {
		t.Fatalf("filesystem state = %q, want unknown on a read/write mount", got.FilesystemState)
	}
	if got.ReadOnlyCause != CauseNotReadOnly {
		t.Fatalf("cause = %q, want empty", got.ReadOnlyCause)
	}
	if !strings.Contains(got.EvidenceSource, "cannot be distinguished") {
		t.Fatalf("evidence must state that the flag is ambiguous, got %q", got.EvidenceSource)
	}
}

// A clean read/write NTFS mount must still report clean — the ambiguity above
// applies only to the dirty flag, and must not degrade every NTFS volume to
// unknown.
func TestProbeReportsCleanNTFSOnAReadWriteMount(t *testing.T) {
	p := fixtureProber(t, map[string]string{
		"sys/class/block/sda1/ro":    "0\n",
		"proc/fs/ntfs3/sda1/volinfo": "ntfs3.1\n4096\n488369919\n19712\n19274\nclean\n",
	})
	m := mountInfo{Device: "/dev/sda1", Mountpoint: "/media/user/Elements", Fstype: "ntfs3", Opts: []string{"rw", "relatime"}}

	got := p.probe(m, false)

	if got.FilesystemState != FilesystemStateClean {
		t.Fatalf("filesystem state = %q, want clean", got.FilesystemState)
	}
}

// The same flag on a read-only mount is the real signal: the driver refused a
// read/write mount because of it.
func TestProbeTreatsTheDirtyFlagOnAReadOnlyMountAsAFault(t *testing.T) {
	p := fixtureProber(t, map[string]string{
		"sys/class/block/sda1/ro":    "0\n",
		"proc/fs/ntfs3/sda1/volinfo": "ntfs3.1\n4096\n488369919\n19712\n19274\nclean\ndirty\n",
	})
	m := mountInfo{Device: "/dev/sda1", Mountpoint: "/media/user/Elements", Fstype: "ntfs3", Opts: []string{"ro", "relatime"}}

	got := p.probe(m, true)

	if got.FilesystemState != FilesystemStateDirty {
		t.Fatalf("filesystem state = %q, want dirty on a read-only mount", got.FilesystemState)
	}
	if got.ReadOnlyCause != CauseFilesystemDirty {
		t.Fatalf("cause = %q, want %q", got.ReadOnlyCause, CauseFilesystemDirty)
	}
}

func TestProbeReportsCleanNTFSVolume(t *testing.T) {
	p := fixtureProber(t, map[string]string{
		"sys/class/block/sda1/ro":    "0\n",
		"proc/fs/ntfs3/sda1/volinfo": "ntfs3.1\n4096\n488369919\n19712\n19274\nclean\nclean\n",
	})
	m := mountInfo{Device: "/dev/sda1", Mountpoint: "/media/user/Elements", Fstype: "ntfs3", Opts: []string{"rw", "relatime"}}

	got := p.probe(m, false)

	if got.FilesystemState != FilesystemStateClean {
		t.Fatalf("filesystem state = %q, want %q", got.FilesystemState, FilesystemStateClean)
	}
	if got.ReadOnlyCause != CauseNotReadOnly {
		t.Fatalf("cause = %q, want empty for a read/write mount", got.ReadOnlyCause)
	}
}

// A write-protected block device outranks every filesystem signal: repairing
// the filesystem cannot restore writes, so proposing repair would be wrong.
func TestProbePrefersDeviceWriteProtectionOverDirtyFlag(t *testing.T) {
	p := fixtureProber(t, map[string]string{
		"sys/class/block/sdb1/ro":    "1\n",
		"proc/fs/ntfs3/sdb1/volinfo": "ntfs3.1\n4096\n1\n1\n1\ndirty\n",
	})
	m := mountInfo{Device: "/dev/sdb1", Mountpoint: "/media/user/USB", Fstype: "ntfs3", Opts: []string{"ro"}}

	got := p.probe(m, true)

	if !got.DeviceWriteProtected {
		t.Fatal("expected device write protection")
	}
	if got.ReadOnlyCause != CauseDeviceWriteProtected {
		t.Fatalf("cause = %q, want %q", got.ReadOnlyCause, CauseDeviceWriteProtected)
	}
}

// A partition inherits its whole disk's write-protect flag.
func TestProbeReadsWholeDiskWriteProtectFlag(t *testing.T) {
	cases := []struct{ name, device, roPath string }{
		{name: "sata partition", device: "/dev/sdc2", roPath: "sys/class/block/sdc/ro"},
		{name: "nvme partition", device: "/dev/nvme0n1p3", roPath: "sys/class/block/nvme0n1/ro"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := fixtureProber(t, map[string]string{tc.roPath: "1\n"})
			got := p.probe(mountInfo{Device: tc.device, Mountpoint: "/mnt", Fstype: "ext4", Opts: []string{"ro"}}, true)
			if got.ReadOnlyCause != CauseDeviceWriteProtected {
				t.Fatalf("cause = %q, want %q", got.ReadOnlyCause, CauseDeviceWriteProtected)
			}
		})
	}
}

// An operator who declared `ro` in fstab gets their intent reported back, not
// a repair proposal for a filesystem that is not broken.
func TestProbeAttributesDeclaredReadOnlyMount(t *testing.T) {
	p := fixtureProber(t, map[string]string{
		"sys/class/block/sdd1/ro": "0\n",
		"etc/fstab":               "# comment\nUUID=abc /srv/archive ext4 ro,noatime 0 2\n/dev/sdd1 /mnt/vault ext4 ro,nosuid 0 2\n",
	})

	byDevice := p.probe(mountInfo{Device: "/dev/sdd1", Mountpoint: "/mnt/vault", Fstype: "ext4", Opts: []string{"ro"}}, true)
	if byDevice.ReadOnlyCause != CauseMountOption {
		t.Fatalf("device-matched cause = %q, want %q", byDevice.ReadOnlyCause, CauseMountOption)
	}

	byMountpoint := p.probe(mountInfo{Device: "/dev/mapper/whatever", Mountpoint: "/srv/archive", Fstype: "ext4", Opts: []string{"ro"}}, true)
	if byMountpoint.ReadOnlyCause != CauseMountOption {
		t.Fatalf("mountpoint-matched cause = %q, want %q", byMountpoint.ReadOnlyCause, CauseMountOption)
	}
}

// Missing evidence must degrade to unknown, never to a confident verdict.
func TestProbeDegradesToUnknownWithoutEvidence(t *testing.T) {
	p := fixtureProber(t, nil)

	got := p.probe(mountInfo{Device: "/dev/sde1", Mountpoint: "/mnt/x", Fstype: "exfat", Opts: []string{"ro"}}, true)

	if got.ReadOnlyCause != CauseUnknown {
		t.Fatalf("cause = %q, want %q", got.ReadOnlyCause, CauseUnknown)
	}
	if got.FilesystemState != FilesystemStateUnknown {
		t.Fatalf("filesystem state = %q, want %q", got.FilesystemState, FilesystemStateUnknown)
	}
}

// Non-Linux hosts have no native adapter yet; the probe must say so rather
// than imply the same confidence as the Linux evidence paths.
func TestProbeOnUnsupportedPlatformStatesItsLimits(t *testing.T) {
	p := fixtureProber(t, nil)
	p.goos = "darwin"

	got := p.probe(mountInfo{Device: "/dev/disk2s1", Mountpoint: "/Volumes/USB", Fstype: "ntfs", Opts: []string{"ro"}}, true)

	if got.ReadOnlyCause != CauseUnknown {
		t.Fatalf("cause = %q, want %q", got.ReadOnlyCause, CauseUnknown)
	}
	if got.DeviceWriteProtected {
		t.Fatal("must not claim block-layer evidence on a platform without an adapter")
	}
	if got.EvidenceSource == "mount-options" {
		t.Fatal("evidence source must record the missing native adapter")
	}
}

func TestVolinfoStatePrefersDirtyOverClean(t *testing.T) {
	cases := []struct {
		name string
		body string
		want FilesystemState
	}{
		{name: "dirty anywhere wins", body: "ntfs3.1\n4096\n1\n2\n3\nclean\ndirty\n", want: FilesystemStateDirty},
		{name: "clean", body: "ntfs3.1\n4096\n1\n2\n3\nclean\nclean\n", want: FilesystemStateClean},
		{name: "no token", body: "ntfs3.1\n4096\n1\n2\n3\n", want: FilesystemStateUnknown},
		{name: "empty", body: "", want: FilesystemStateUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := volinfoState(tc.body); got != tc.want {
				t.Fatalf("volinfoState(%q) = %q, want %q", tc.body, got, tc.want)
			}
		})
	}
}
