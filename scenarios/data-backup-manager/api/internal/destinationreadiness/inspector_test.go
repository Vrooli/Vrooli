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
