package volumeremediation

import (
	"strings"
	"testing"
)

// A device path reaches a command line. Anything that is not a plain device
// specifier must be rejected at the boundary rather than escaped later.
func TestValidateDevicePathRejectsSmuggledArguments(t *testing.T) {
	rejected := []string{
		"",
		"   ",
		"/dev/sda1; rm -rf /",
		"/dev/sda1 && reboot",
		"/dev/sda1|tee",
		"/dev/../etc/passwd",
		"/dev/sda1 --force",
		"$(reboot)",
		"`reboot`",
		"/etc/passwd",
		"sda1",
		"--help",
		"-n",
		"/dev/sda1\nrm",
		strings.Repeat("/dev/sda1", 64),
	}
	for _, path := range rejected {
		t.Run(strings.ReplaceAll(path, "/", "_"), func(t *testing.T) {
			if err := validateDevicePath(path); err == nil {
				t.Fatalf("accepted %q", path)
			}
		})
	}

	accepted := []string{"/dev/sda1", "/dev/nvme0n1p2", "/dev/disk2s1", "/dev/mapper/vg-data", "E:", "c:"}
	for _, path := range accepted {
		t.Run("accept "+path, func(t *testing.T) {
			if err := validateDevicePath(path); err != nil {
				t.Fatalf("rejected %q: %v", path, err)
			}
		})
	}
}

func TestValidateMountpointKeepsMountsInsideMediaRoots(t *testing.T) {
	accepted := []string{"/media/user/Elements", "/run/media/user/USB", "/mnt/backup", "/Volumes/USB"}
	for _, mountpoint := range accepted {
		if err := validateMountpoint(mountpoint); err != nil {
			t.Fatalf("rejected %q: %v", mountpoint, err)
		}
	}

	rejected := []string{"", "relative/path", "/", "/home/user", "/usr/local", "/etc", "/srv/data", "/media/../etc", "/boot"}
	for _, mountpoint := range rejected {
		if err := validateMountpoint(mountpoint); err == nil {
			t.Fatalf("accepted %q", mountpoint)
		}
	}
}

func TestIsSystemMountpoint(t *testing.T) {
	system := []string{"/", "/boot", "/boot/efi", "/usr", "/usr/local/lib", "/etc", "/var/lib", "/home", "/home/user", "/root"}
	for _, mountpoint := range system {
		if !isSystemMountpoint(mountpoint) {
			t.Fatalf("%q must be treated as a system mountpoint", mountpoint)
		}
	}
	external := []string{"/media/user/Elements", "/mnt/backup", "/run/media/user/USB", "/Volumes/Backup", ""}
	for _, mountpoint := range external {
		if isSystemMountpoint(mountpoint) {
			t.Fatalf("%q must not be treated as a system mountpoint", mountpoint)
		}
	}
}

// The allowlist is a statement that an adapter exists, so every listed name
// must actually resolve to one.
func TestEverySupportedFilesystemHasBothAdapters(t *testing.T) {
	for _, name := range SupportedFilesystems() {
		family, ok := filesystemFamily(name)
		if !ok {
			t.Fatalf("%q is listed as supported but has no family", name)
		}
		switch family {
		case "apfs", "hfs":
			// macOS families are driven by diskutil, which takes the volume
			// rather than a per-family tool.
			continue
		}
		if _, err := checkArgs(family, "/dev/sda1"); err != nil {
			t.Fatalf("%q (family %q) has no check adapter: %v", name, family, err)
		}
		if _, err := repairArgs(family, "/dev/sda1"); err != nil {
			t.Fatalf("%q (family %q) has no repair adapter: %v", name, family, err)
		}
	}
}

// Formatting, partitioning and clearing destroy backups rather than recovering
// them. They must not be reachable through this vocabulary at all.
func TestActionRegistryExcludesDestructiveDataActions(t *testing.T) {
	for _, action := range []Action{"format", "clear", "partition", "relabel", "wipe", "", "shell"} {
		if action.Known() {
			t.Fatalf("%q must not be a known remediation action", action)
		}
	}
	if !ActionRepair.Destructive() {
		t.Fatal("repair must be classified destructive")
	}
	for _, action := range []Action{ActionInspect, ActionCheck, ActionUnmount, ActionMountReadWrite} {
		if action.Destructive() {
			t.Fatalf("%q must not be classified destructive", action)
		}
	}
}

func TestDeviceMatchesIgnoresMountpointButNotIdentity(t *testing.T) {
	approved := elementsDevice()

	unmounted := approved
	unmounted.Mountpoint = ""
	if !approved.Matches(unmounted) {
		t.Fatal("matching must survive the unmount the remediation performs")
	}

	remounted := approved
	remounted.Mountpoint = "/media/user/Elements2"
	if !approved.Matches(remounted) {
		t.Fatal("matching must survive a changed mountpoint")
	}

	swapped := approved
	swapped.UUID = "different"
	swapped.Serial = "different"
	if approved.Matches(swapped) {
		t.Fatal("a different disk must not match")
	}

	resized := approved
	resized.TotalBytes = 42
	if approved.Matches(resized) {
		t.Fatal("a different size must not match")
	}

	nothing := Device{}
	if nothing.Matches(Device{}) {
		t.Fatal("two identity-free devices must never match")
	}
}
