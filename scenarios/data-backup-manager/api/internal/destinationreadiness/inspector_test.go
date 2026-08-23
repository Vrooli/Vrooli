package destinationreadiness_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/sysmounts"
)

type fakeVolumeScanner struct {
	volumes []sysmounts.Volume
}

func (f fakeVolumeScanner) Scan(context.Context) ([]sysmounts.Volume, error) {
	return append([]sysmounts.Volume(nil), f.volumes...), nil
}

func TestReadOnlyInspectorMatchesDeepestMountAndDetectsInstallerMedia(t *testing.T) {
	root := t.TempDir()
	usb := filepath.Join(root, "USB")
	nested := filepath.Join(usb, "vrooli-backups")
	for _, dir := range []string{filepath.Join(usb, ".disk"), filepath.Join(usb, "casper"), nested} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("create test dir: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(usb, "boot.catalog"), []byte("fixture"), 0o644); err != nil {
		t.Fatalf("create fixture file: %v", err)
	}
	inspector := destinationreadiness.NewReadOnlyInspector(fakeVolumeScanner{volumes: []sysmounts.Volume{
		{DevicePath: "/dev/root", Mountpoint: root, Filesystem: "ext4", Class: sysmounts.ClassFixed, FreeBytes: 100, TotalBytes: 200},
		{DevicePath: "/dev/sdz1", Mountpoint: usb, Filesystem: "vfat", Class: sysmounts.ClassRemovable, Removable: true, FreeBytes: 50, TotalBytes: 64, MountOptions: []string{"rw", "nosuid"}},
	}})

	got, err := inspector.Inspect(context.Background(), nested)
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if got.Identity.DevicePath != "/dev/sdz1" {
		t.Fatalf("device = %q, want /dev/sdz1", got.Identity.DevicePath)
	}
	if got.Identity.Mountpoint != usb {
		t.Fatalf("mountpoint = %q, want %q", got.Identity.Mountpoint, usb)
	}
	if !got.NonEmptyRoot {
		t.Fatal("expected non-empty root")
	}
	if !got.InstallerMedia {
		t.Fatalf("expected installer-media detection, entries=%v", got.TopLevelEntries)
	}
	if got.ReadOnly {
		t.Fatal("did not expect read-only fixture")
	}
}

func TestReadOnlyInspectorRejectsUnmountedLocation(t *testing.T) {
	inspector := destinationreadiness.NewReadOnlyInspector(fakeVolumeScanner{volumes: []sysmounts.Volume{
		{DevicePath: "/dev/sdz1", Mountpoint: "/media/user/USB", Filesystem: "exfat"},
	}})
	_, err := inspector.Inspect(context.Background(), "/tmp/not-mounted")
	if err == nil {
		t.Fatal("expected error for unmounted location")
	}
}

func TestReadOnlyInspectorDistinguishesMissingChildFromMountedParent(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "not-created-yet")
	inspector := destinationreadiness.NewReadOnlyInspector(fakeVolumeScanner{volumes: []sysmounts.Volume{
		{DevicePath: "/dev/root", Mountpoint: root, Filesystem: "ext4", Class: sysmounts.ClassFixed, FreeBytes: 100, TotalBytes: 200},
	}})

	got, err := inspector.Inspect(context.Background(), missing)
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if got.LocationExists {
		t.Fatal("missing destination was reported as present")
	}
	if got.LocationIsDirectory {
		t.Fatal("missing destination was reported as a directory")
	}
	if got.Identity.Mountpoint != root {
		t.Fatalf("parent mount = %q, want %q", got.Identity.Mountpoint, root)
	}
}

// fakeDeviceScanner also resolves devices by path, including devices that are
// present but not mounted.
type fakeDeviceScanner struct {
	fakeVolumeScanner
	devices map[string]sysmounts.Volume
	err     error
}

func (f fakeDeviceScanner) Device(_ context.Context, devicePath string) (sysmounts.Volume, error) {
	if f.err != nil {
		return sysmounts.Volume{}, f.err
	}
	for _, v := range f.volumes {
		if v.DevicePath == devicePath {
			return v, nil
		}
	}
	if v, ok := f.devices[devicePath]; ok {
		return v, nil
	}
	return sysmounts.Volume{}, sysmounts.ErrDeviceNotFound
}

// Mid-repair the volume is deliberately unmounted. That must be an observation
// with intact identity, not an error, or the flow cannot re-verify what it is
// about to remount.
func TestInspectDeviceObservesAnUnmountedVolume(t *testing.T) {
	inspector := destinationreadiness.NewReadOnlyInspector(fakeDeviceScanner{
		devices: map[string]sysmounts.Volume{
			"/dev/sdz1": {
				DevicePath:      "/dev/sdz1",
				Filesystem:      "ntfs3",
				TotalBytes:      2 << 40,
				UUID:            "E26A883E6A881189",
				Serial:          "WD-TEST",
				Label:           "Elements",
				FilesystemState: sysmounts.FilesystemStateDirty,
				StateEvidence:   "/proc/fs/ntfs3/sdz1/volinfo",
			},
		},
	})

	got, err := inspector.InspectDevice(context.Background(), destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdz1"})
	if err != nil {
		t.Fatalf("InspectDevice returned error: %v", err)
	}
	if got.Mounted {
		t.Fatal("unmounted volume reported as mounted")
	}
	if got.Identity.UUID != "E26A883E6A881189" || got.Identity.Serial != "WD-TEST" {
		t.Fatalf("identity lost while unmounted: %+v", got.Identity)
	}
	if got.FilesystemState != sysmounts.FilesystemStateDirty {
		t.Fatalf("filesystem state = %q, want dirty", got.FilesystemState)
	}
	if got.LocationExists {
		t.Fatal("an unmounted volume has no location to exist at")
	}
}

func TestInspectDeviceReportsMountedVolumeContents(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "vrooli-backups"), 0o755); err != nil {
		t.Fatalf("create fixture dir: %v", err)
	}
	inspector := destinationreadiness.NewReadOnlyInspector(fakeDeviceScanner{
		fakeVolumeScanner: fakeVolumeScanner{volumes: []sysmounts.Volume{
			{DevicePath: "/dev/sdz1", Mountpoint: root, Filesystem: "exfat", ReadOnly: true, ReadOnlyCause: sysmounts.CauseFilesystemDirty},
		}},
	})

	got, err := inspector.InspectDevice(context.Background(), destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdz1"})
	if err != nil {
		t.Fatalf("InspectDevice returned error: %v", err)
	}
	if !got.Mounted || !got.LocationIsDirectory {
		t.Fatalf("mounted volume not reported as such: %+v", got)
	}
	if !got.NonEmptyRoot {
		t.Fatal("expected the fixture subdirectory to be observed")
	}
	if got.ReadOnlyCause != sysmounts.CauseFilesystemDirty {
		t.Fatalf("cause = %q, want %q", got.ReadOnlyCause, sysmounts.CauseFilesystemDirty)
	}
}

// A scanner with no device lens must say so rather than silently reporting an
// empty inspection that would read as "device is fine".
func TestInspectDeviceRequiresADeviceLens(t *testing.T) {
	inspector := destinationreadiness.NewReadOnlyInspector(fakeVolumeScanner{})

	if _, err := inspector.InspectDevice(context.Background(), destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdz1"}); err == nil {
		t.Fatal("expected an error when device-scoped inspection is unavailable")
	}
	if _, err := inspector.InspectDevice(context.Background(), destinationreadiness.DeviceIdentity{}); err == nil {
		t.Fatal("expected an error for an empty device path")
	}
}
