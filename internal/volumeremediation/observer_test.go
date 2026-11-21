package volumeremediation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureObserver builds an observer over a temp tree so the Linux evidence
// paths run on any host.
func fixtureObserver(t *testing.T, files map[string]string) *HostObserver {
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
	return &HostObserver{
		goos:          "linux",
		procMounts:    filepath.Join(root, "proc", "mounts"),
		sysClassBlock: filepath.Join(root, "sys", "class", "block"),
		procFS:        filepath.Join(root, "proc", "fs"),
		devDiskByUUID: filepath.Join(root, "dev", "disk", "by-uuid"),
		readFile:      os.ReadFile,
		readDir:       os.ReadDir,
		evalSymlinks:  func(p string) (string, error) { return p, nil },
		run:           func(context.Context, []string) ([]byte, error) { return nil, errors.New("no lsblk in test") },
	}
}

// The Elements incident, observed end to end from host evidence.
func TestObserveLinuxReadsMountAndDirtyEvidence(t *testing.T) {
	o := fixtureObserver(t, map[string]string{
		"proc/mounts":                "/dev/nvme0n1p2 / ext4 rw,relatime 0 0\n/dev/sda1 /media/user/Elements ntfs3 ro,nosuid,relatime 0 0\n",
		"sys/class/block/sda1/size":  "3906959360\n",
		"sys/class/block/sda1/ro":    "0\n",
		"proc/fs/ntfs3/sda1/volinfo": "ntfs3.1\n4096\n488369919\n19712\n19274\ndirty\ndirty\n",
	})

	state, err := o.Observe(context.Background(), "/dev/sda1")
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}

	if !state.Mounted || state.Device.Mountpoint != "/media/user/Elements" { //nolint:goconst // fixture mountpoint is intentionally reused
		t.Fatalf("mount not observed: %+v", state)
	}
	if !state.ReadOnly {
		t.Fatal("ro mount option not observed")
	}
	if state.Device.Filesystem != "ntfs3" {
		t.Fatalf("filesystem = %q", state.Device.Filesystem)
	}
	if state.Dirty != TristateYes {
		t.Fatalf("dirty = %q, want yes", state.Dirty)
	}
	if state.Device.TotalBytes != 3906959360*512 {
		t.Fatalf("size = %d", state.Device.TotalBytes)
	}
	if state.Evidence == "" {
		t.Fatal("dirty verdict must cite its evidence source")
	}
}

func TestObserveLinuxSeparatesFuseblkDriverFromNTFSFormat(t *testing.T) {
	o := fixtureObserver(t, map[string]string{
		"proc/mounts":               "/dev/sda1 /media/user/Elements fuseblk rw,nosuid 0 0\n",
		"sys/class/block/sda1/size": "3906959360\n",
		"sys/class/block/sda1/ro":   "0\n",
	})
	o.run = func(_ context.Context, argv []string) ([]byte, error) {
		if len(argv) > 0 && argv[0] == "lsblk" {
			return []byte(`UUID="E26A883E6A881189" SERIAL="WD-WX52A946D6VL" FSTYPE="ntfs" LABEL="Elements"`), nil
		}
		return nil, errors.New("unexpected command")
	}

	state, err := o.Observe(context.Background(), "/dev/sda1")
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if state.Device.Filesystem != "ntfs" {
		t.Fatalf("on-disk filesystem = %q, want ntfs", state.Device.Filesystem)
	}
	if state.Device.MountDriver != "fuseblk" {
		t.Fatalf("mount driver = %q, want fuseblk", state.Device.MountDriver)
	}
	if !state.Mounted || state.Device.Mountpoint != "/media/user/Elements" {
		t.Fatalf("mount state = %+v, want mounted at Elements", state)
	}
	if state.Dirty != TristateUnknown {
		t.Fatalf("dirty = %q, want unknown when FUSE publishes no clean-state evidence", state.Dirty)
	}
}

func TestObserveLinuxReportsUnmountedDevice(t *testing.T) {
	o := fixtureObserver(t, map[string]string{
		"proc/mounts":                "/dev/nvme0n1p2 / ext4 rw,relatime 0 0\n",
		"sys/class/block/sda1/size":  "3906959360\n",
		"sys/class/block/sda1/ro":    "0\n",
		"proc/fs/ntfs3/sda1/volinfo": "ntfs3.1\n4096\n1\n2\n3\nclean\n",
	})

	state, err := o.Observe(context.Background(), "/dev/sda1")
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if state.Mounted {
		t.Fatal("device must be reported unmounted")
	}
	if state.Device.Mountpoint != "" {
		t.Fatalf("mountpoint = %q, want empty", state.Device.Mountpoint)
	}
	// The filesystem is unknown without a mount entry and without lsblk, so the
	// dirty probe cannot select an adapter — that must stay unknown, not clean.
	if state.Dirty != TristateUnknown {
		t.Fatalf("dirty = %q, want unknown when the filesystem is unproven", state.Dirty)
	}
}

// Block-layer write protection applies whether or not the volume is mounted.
func TestObserveLinuxReportsWriteProtectedDevice(t *testing.T) {
	o := fixtureObserver(t, map[string]string{
		"proc/mounts":               "",
		"sys/class/block/sdb1/size": "1024\n",
		"sys/class/block/sdb/ro":    "1\n",
	})

	state, err := o.Observe(context.Background(), "/dev/sdb1")
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if !state.ReadOnly {
		t.Fatal("whole-disk write protection must be reported")
	}
}

func TestObserveLinuxResolvesUUIDFromDiskByUUID(t *testing.T) {
	root := t.TempDir()
	byUUID := filepath.Join(root, "dev", "disk", "by-uuid")
	if err := os.MkdirAll(byUUID, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(byUUID, "E26A883E6A881189"), nil, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	o := fixtureObserver(t, map[string]string{"proc/mounts": ""})
	o.devDiskByUUID = byUUID
	// Resolve the by-uuid entry onto the device under test.
	o.evalSymlinks = func(p string) (string, error) {
		if filepath.Base(p) == "E26A883E6A881189" {
			return "/dev/sda1", nil
		}
		return p, nil
	}

	state, err := o.Observe(context.Background(), "/dev/sda1")
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if state.Device.UUID != "E26A883E6A881189" {
		t.Fatalf("uuid = %q", state.Device.UUID)
	}
}

func TestObserveRejectsAnInvalidDevicePath(t *testing.T) {
	o := fixtureObserver(t, nil)
	if _, err := o.Observe(context.Background(), "/dev/sda1; reboot"); err == nil {
		t.Fatal("expected a validation error")
	}
}

// An unsupported platform must say so and still name a native command.
func TestObserveOnUnsupportedPlatform(t *testing.T) {
	o := NewHostObserver("plan9")
	_, err := o.Observe(context.Background(), "/dev/sda1")
	var unsupported ErrUnsupported
	if !asUnsupported(err, &unsupported) {
		t.Fatalf("error = %v, want ErrUnsupported", err)
	}
}

func TestObserveDarwinParsesMountOutput(t *testing.T) {
	o := NewHostObserver("darwin")
	o.run = func(_ context.Context, argv []string) ([]byte, error) {
		if len(argv) > 0 && argv[0] == "diskutil" {
			return []byte("Volume UUID: volume-2\nFile System Personality: ExFAT\nMounted: Yes\nMount Point: /Volumes/Backup\nRead-Only Volume: Yes\n"), nil
		}
		return []byte("/dev/disk1s1 on / (apfs, local, journaled)\n/dev/disk2s1 on /Volumes/Backup (exfat, local, nodev, read-only)\n"), nil
	}

	state, err := o.Observe(context.Background(), "/dev/disk2s1")
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if !state.Mounted || state.Device.Mountpoint != "/Volumes/Backup" {
		t.Fatalf("mount not parsed: %+v", state)
	}
	if state.Device.Filesystem != "ExFAT" {
		t.Fatalf("on-disk filesystem = %q, want ExFAT from diskutil info", state.Device.Filesystem)
	}
	if state.Device.MountDriver != "exfat" {
		t.Fatalf("mount driver = %q, want exfat from mount output", state.Device.MountDriver)
	}
	if !state.ReadOnly {
		t.Fatal("read-only attribute not parsed")
	}
	if state.Dirty != TristateUnknown {
		t.Fatalf("macOS publishes no portable dirty flag; dirty = %q", state.Dirty)
	}
}

func TestObserveDarwinResolvesUnmountedVolumeByDiskutilIdentity(t *testing.T) {
	o := NewHostObserver("darwin")
	var calls int
	o.run = func(_ context.Context, argv []string) ([]byte, error) {
		calls++
		if len(argv) > 0 && argv[0] == "diskutil" {
			return []byte("Volume UUID: volume-2\nFile System Personality: ExFAT\nMounted: No\n"), nil
		}
		return []byte("/dev/disk1s1 on / (apfs, local, journaled)\n"), nil
	}

	state, err := o.Observe(context.Background(), "/dev/disk2s1")
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if calls != 2 || state.Mounted || state.Device.UUID != "volume-2" || state.Device.Filesystem != "ExFAT" {
		t.Fatalf("unmounted device identity was not resolved: calls=%d state=%+v", calls, state)
	}
}

func TestObserveWindowsReadsStableIdentityAndKeepsFilesystemHealthUnknown(t *testing.T) {
	o := NewHostObserver("windows")
	var argv []string
	o.run = func(_ context.Context, got []string) ([]byte, error) {
		argv = append([]string(nil), got...)
		return []byte(`{"DriveLetter":"E","FileSystem":"NTFS","UniqueId":"\\\\?\\Volume{abcd}\\","Size":4000000000,"FileSystemLabel":"Elements","SerialNumber":"DISK-42","IsReadOnly":false}`), nil
	}

	state, err := o.Observe(context.Background(), "e:")
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if !state.Mounted || state.Device.Mountpoint != `E:\` {
		t.Fatalf("mount state = %+v, want E:\\ mounted", state)
	}
	if state.Device.UUID != `\\?\Volume{abcd}\` || state.Device.Serial != "DISK-42" || state.Device.Filesystem != "NTFS" {
		t.Fatalf("identity = %+v", state.Device)
	}
	if state.Dirty != TristateUnknown {
		t.Fatalf("Dirty = %q, Windows inventory must not claim clean", state.Dirty)
	}
	if len(argv) != 5 || argv[0] != "powershell" || !strings.Contains(argv[4], "Get-Volume -DriveLetter 'E'") {
		t.Fatalf("argv = %q, want a read-only query for validated drive E", argv)
	}
}

func TestObserveWindowsRefusesInventoryWithoutStableVolumeIdentity(t *testing.T) {
	o := NewHostObserver("windows")
	o.run = func(context.Context, []string) ([]byte, error) {
		return []byte(`{"DriveLetter":"E","FileSystem":"NTFS","Size":4000000000}`), nil
	}
	if _, err := o.Observe(context.Background(), "E:"); err == nil || !strings.Contains(err.Error(), "stable volume identity") {
		t.Fatalf("Observe error = %v, want refusal for missing stable identity", err)
	}
}

func TestObserveWindowsResolvesUnletteredVolumeByStableGUID(t *testing.T) {
	o := NewHostObserver("windows")
	var script string
	o.run = func(_ context.Context, argv []string) ([]byte, error) {
		script = argv[4]
		return []byte(`{"DriveLetter":null,"FileSystem":"NTFS","UniqueId":"\\\\?\\Volume{12345678-1234-1234-1234-123456789abc}\\","Size":4000000000,"FileSystemLabel":"Elements","SerialNumber":"DISK-42","IsReadOnly":false}`), nil
	}
	volumeID := `\\?\Volume{12345678-1234-1234-1234-123456789abc}\`
	state, err := o.Observe(context.Background(), volumeID)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if state.Mounted || state.Device.Path != volumeID || state.Device.UUID != volumeID || state.Device.Mountpoint != "" {
		t.Fatalf("unlettered volume identity = %+v", state)
	}
	if !strings.Contains(script, "Get-Volume -UniqueId") {
		t.Fatalf("observer did not resolve by the stable ID: %q", script)
	}
}

func TestUnescapeMountPath(t *testing.T) {
	cases := map[string]string{
		`/media/user/My\040Drive`: "/media/user/My Drive",
		`/media/user/Elements`:    "/media/user/Elements",
		`/media/tab\011here`:      "/media/tab\there",
	}
	for in, want := range cases {
		if got := unescapeMountPath(in); got != want {
			t.Fatalf("unescapeMountPath(%q) = %q, want %q", in, got, want)
		}
	}
}

// NTFS marks a volume dirty while it is mounted read/write. Once the driver has
// accepted that mount the flag is an in-use marker, not a fault — and a
// remediation flow that read it as a fault would loop repairing a healthy disk.
func TestObserveTreatsTheDirtyFlagOnAReadWriteMountAsNormal(t *testing.T) {
	o := fixtureObserver(t, map[string]string{
		"proc/mounts":                "/dev/sda1 /media/user/Elements ntfs3 rw,relatime 0 0\n",
		"sys/class/block/sda1/ro":    "0\n",
		"proc/fs/ntfs3/sda1/volinfo": "ntfs3.1\n4096\n1\n2\n3\nclean\ndirty\n",
	})

	state, err := o.Observe(context.Background(), "/dev/sda1")
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if state.ReadOnly {
		t.Fatal("mount is rw")
	}
	if state.Dirty != TristateNo {
		t.Fatalf("dirty = %q, want no on a read/write mount", state.Dirty)
	}
	if !strings.Contains(state.Evidence, "in-use marker") {
		t.Fatalf("evidence must explain the verdict, got %q", state.Evidence)
	}
}

// The mount-read-write gate must not refuse a volume whose only "dirty" signal
// is that in-use marker.
func TestMountReadWriteIsNotBlockedByAnInUseDirtyMarker(t *testing.T) {
	state := unmountedDirtyState()
	state.Dirty = TristateNo
	runner := &fakeRunner{out: []byte("Mounted /dev/sda1 at /media/user/Elements")}
	svc := linuxService(t, &fakeObserver{state: state}, runner)

	res, err := svc.Execute(context.Background(), Request{Action: ActionMountReadWrite, Device: elementsDevice(), DesiredMountpoint: "/media/user/Elements"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !res.Changed {
		t.Fatalf("result = %+v, want a mount", res)
	}
}
