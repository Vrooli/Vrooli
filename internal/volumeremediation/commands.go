package volumeremediation

import (
	"fmt"
	"path/filepath"
	"strings"
)

const volumeFilesystemExfat = "exfat"

const (
	commandsNtfs = "ntfs"
)

// Backend names an execution path. Which one serves a request depends on what
// the host actually offers, and the answer is recorded in every result so an
// operator can tell how a change was made.
type Backend string

const (
	// BackendUDisks is the Linux desktop path. polkit authorises a non-system
	// device for an active session with no password and no root, so this path
	// never elevates.
	BackendUDisks Backend = "udisks2"
	// BackendNativeTools drives the filesystem tools directly. It needs root,
	// so it runs only through the setup-installed privilege broker.
	BackendNativeTools Backend = "native-tools"
	// BackendDiskutil is the macOS path.
	BackendDiskutil Backend = "diskutil"
	// BackendRepairVolume is the Windows path.
	BackendRepairVolume Backend = "repair-volume"
)

// NeedsElevation reports whether a backend can only run through the privilege
// broker. udisks2 deliberately does not: routing it through the broker would
// add an elevation the host does not require.
func (b Backend) NeedsElevation() bool { return b == BackendNativeTools }

// commandFor builds the exact argv for one action on one backend.
//
// Argv construction is separated from execution so it can be unit-tested on
// every platform, including the ones whose tools cannot run in CI. A command
// this function will not build is a command that cannot run.
func commandFor(back Backend, req Request, state State) ([]string, error) {
	switch back {
	case BackendUDisks:
		return udisksCommand(req)
	case BackendNativeTools:
		return nativeToolCommand(req, state)
	case BackendDiskutil:
		return diskutilCommand(req)
	case BackendRepairVolume:
		return repairVolumeCommand(req, state)
	default:
		return nil, ErrUnsupported{Reason: "unknown backend " + string(back)}
	}
}

// udisksBlockObject maps a device path to its UDisks2 D-Bus object path.
func udisksBlockObject(devicePath string) string {
	return "/org/freedesktop/UDisks2/block_devices/" + filepath.Base(strings.TrimSpace(devicePath))
}

func udisksCommand(req Request) ([]string, error) {
	device := strings.TrimSpace(req.Device.Path)
	switch req.Action {
	case ActionUnmount:
		return []string{"udisksctl", "unmount", "--block-device", device, "--no-user-interaction"}, nil
	case ActionMountReadWrite:
		// NTFS3 has a known kernel-context write-fault risk on supported Linux
		// hosts. UDisks recognizes the on-disk "ntfs" type and routes it through
		// mount.ntfs; on Linux this host helper is ntfs-3g. The literal
		// "ntfs-3g" type is rejected by UDisks as not being a filesystem type.
		if family, ok := filesystemFamily(req.Device.Filesystem); ok && family == commandsNtfs {
			return []string{"udisksctl", "mount", "--block-device", device, "--filesystem-type", "ntfs", "--no-user-interaction"}, nil
		}
		return []string{"udisksctl", "mount", "--block-device", device, "--no-user-interaction"}, nil
	case ActionCheck:
		return []string{"busctl", "call", "org.freedesktop.UDisks2", udisksBlockObject(device), "org.freedesktop.UDisks2.Filesystem", "Check", "a{sv}", "0"}, nil
	case ActionRepair:
		return []string{"busctl", "call", "org.freedesktop.UDisks2", udisksBlockObject(device), "org.freedesktop.UDisks2.Filesystem", "Repair", "a{sv}", "0"}, nil
	default:
		return nil, ErrUnsupported{Reason: fmt.Sprintf("udisks2 backend has no command for %q", req.Action)}
	}
}

func nativeToolCommand(req Request, state State) ([]string, error) {
	device := strings.TrimSpace(req.Device.Path)
	family, ok := filesystemFamily(req.Device.Filesystem)
	if !ok && req.Action != ActionUnmount {
		return nil, ErrRefused{Reason: "filesystem " + req.Device.Filesystem + " has no repair adapter"}
	}
	switch req.Action {
	case ActionUnmount:
		mountpoint := strings.TrimSpace(state.Device.Mountpoint)
		if mountpoint == "" {
			mountpoint = strings.TrimSpace(req.Device.Mountpoint)
		}
		if mountpoint == "" {
			return nil, ErrInvalid{Field: "mountpoint", Reason: "required to unmount with native tools"}
		}
		return []string{"umount", mountpoint}, nil
	case ActionMountReadWrite:
		mountpoint := strings.TrimSpace(req.DesiredMountpoint)
		if mountpoint == "" {
			mountpoint = strings.TrimSpace(req.Device.Mountpoint)
		}
		if err := validateMountpoint(mountpoint); err != nil {
			return nil, err
		}
		filesystemType := strings.ToLower(strings.TrimSpace(req.Device.Filesystem))
		if family == commandsNtfs {
			filesystemType = "ntfs-3g"
		}
		return []string{"mount", "-t", filesystemType, "-o", "rw", device, mountpoint}, nil
	case ActionCheck:
		return checkArgs(family, device)
	case ActionRepair:
		return repairArgs(family, device)
	default:
		return nil, ErrUnsupported{Reason: fmt.Sprintf("native tools have no command for %q", req.Action)}
	}
}

// checkArgs returns a strictly non-writing consistency check per filesystem
// family. Every one of these is the tool's explicit no-action mode.
func checkArgs(family, device string) ([]string, error) {
	switch family {
	case commandsNtfs:
		return []string{"ntfsfix", "-n", device}, nil
	case "ext":
		return []string{"e2fsck", "-f", "-n", device}, nil
	case volumeFilesystemExfat:
		return []string{"fsck.exfat", "-n", device}, nil
	case "vfat":
		return []string{"fsck.fat", "-n", device}, nil
	default:
		return nil, ErrUnsupported{Reason: "no non-writing check adapter for filesystem family " + family}
	}
}

// repairArgs returns the writing repair command per filesystem family.
//
// The ntfs entry clears the volume dirty flag, which is what makes an NTFS
// volume mountable read/write again. It is not a chkdsk replacement: it fixes a
// small set of known problems and resets the log. Callers are expected to
// verify repository-level integrity afterwards rather than treat a successful
// repair as proof the data is intact.
func repairArgs(family, device string) ([]string, error) {
	switch family {
	case commandsNtfs:
		return []string{"ntfsfix", "-d", device}, nil
	case "ext":
		return []string{"e2fsck", "-f", "-y", device}, nil
	case "exfat":
		return []string{"fsck.exfat", "-y", device}, nil
	case "vfat":
		return []string{"fsck.fat", "-a", device}, nil
	default:
		return nil, ErrUnsupported{Reason: "no repair adapter for filesystem family " + family}
	}
}

func diskutilCommand(req Request) ([]string, error) {
	device := strings.TrimSpace(req.Device.Path)
	switch req.Action {
	case ActionUnmount:
		return []string{"diskutil", "unmount", device}, nil
	case ActionMountReadWrite:
		return []string{"diskutil", "mount", device}, nil
	case ActionCheck:
		return []string{"diskutil", "verifyVolume", device}, nil
	case ActionRepair:
		return []string{"diskutil", "repairVolume", device}, nil
	default:
		return nil, ErrUnsupported{Reason: fmt.Sprintf("diskutil backend has no command for %q", req.Action)}
	}
}

// windowsDriveLetter extracts the drive letter Repair-Volume addresses.
func windowsDriveLetter(devicePath string) (string, error) {
	device := strings.TrimSpace(devicePath)
	if len(device) >= 2 && device[1] == ':' {
		return strings.ToUpper(device[:1]), nil
	}
	return "", ErrInvalid{Field: "device.path", Reason: "Windows remediation needs a drive letter such as E:"}
}

func repairVolumeCommand(req Request, observed State) ([]string, error) {
	uniqueID := strings.TrimSpace(observed.Device.UUID)
	if uniqueID == "" {
		uniqueID = strings.TrimSpace(req.Device.UUID)
	}
	if uniqueID == "" && windowsVolumeGUIDPattern.MatchString(req.Device.Path) {
		uniqueID = req.Device.Path
	}
	if uniqueID != "" && !windowsVolumeGUIDPattern.MatchString(uniqueID) {
		return nil, ErrInvalid{Field: "device.uuid", Reason: "Windows remediation requires a stable volume GUID"}
	}
	if uniqueID == "" {
		if _, err := windowsDriveLetter(req.Device.Path); err != nil {
			return nil, err
		}
	}
	selector := ""
	if uniqueID != "" {
		selector = fmt.Sprintf("$v = Get-Volume -UniqueId '%s' -ErrorAction Stop; if (@($v).Count -ne 1) { throw 'volume GUID did not resolve to exactly one volume' }; ", uniqueID)
	}
	switch req.Action {
	case ActionCheck:
		if uniqueID != "" {
			return []string{"powershell", "-NoProfile", "-NonInteractive", "-Command", selector + "$result = Repair-Volume -InputObject $v -Scan -ErrorAction Stop; [Console]::Out.WriteLine('DBM_CHECK_RESULT=' + [string]$result)"}, nil
		}
		letter, _ := windowsDriveLetter(req.Device.Path)
		return []string{"powershell", "-NoProfile", "-NonInteractive", "-Command", "$result = Repair-Volume -DriveLetter " + letter + " -Scan -ErrorAction Stop; [Console]::Out.WriteLine('DBM_CHECK_RESULT=' + [string]$result)"}, nil
	case ActionRepair:
		if uniqueID != "" {
			return []string{"powershell", "-NoProfile", "-NonInteractive", "-Command", selector + "Repair-Volume -InputObject $v -OfflineScanAndFix -ErrorAction Stop"}, nil
		}
		letter, _ := windowsDriveLetter(req.Device.Path)
		return []string{"powershell", "-NoProfile", "-NonInteractive", "-Command", "Repair-Volume -DriveLetter " + letter + " -OfflineScanAndFix -ErrorAction Stop"}, nil
	case ActionUnmount:
		// Windows Repair-Volume takes a target volume offline for its own repair
		// operation; explicit unmount is not part of this adapter.
		return nil, ErrUnsupported{
			Reason: "Windows manages volume detachment itself; there is no unmount step to drive",
		}
	case ActionMountReadWrite:
		return windowsMountCommand(req, observed)
	default:
		return nil, ErrUnsupported{Reason: fmt.Sprintf("Windows backend has no command for %q", req.Action)}
	}
}

func windowsMountCommand(req Request, observed State) ([]string, error) {
	if err := validateWindowsMountpoint(req.DesiredMountpoint); err != nil {
		return nil, err
	}
	uniqueID := strings.TrimSpace(observed.Device.UUID)
	if uniqueID == "" {
		uniqueID = strings.TrimSpace(req.Device.UUID)
	}
	if uniqueID == "" && windowsVolumeGUIDPattern.MatchString(req.Device.Path) {
		uniqueID = req.Device.Path
	}
	if !windowsVolumeGUIDPattern.MatchString(uniqueID) {
		return nil, ErrInvalid{Field: "device.uuid", Reason: "Windows remount requires a stable volume GUID"}
	}
	letter := strings.ToUpper(strings.TrimSpace(req.DesiredMountpoint)[:1])
	// Resolve the volume by its stable UniqueId, then restore the prior drive
	// letter only when Windows has not assigned it another letter and the
	// requested letter is still unused. This changes attachment only.
	script := fmt.Sprintf("$v = Get-Volume -UniqueId '%s' -ErrorAction Stop; if (@($v).Count -ne 1) { throw 'volume GUID did not resolve to exactly one volume' }; if ($v.DriveLetter) { throw 'volume already has a drive letter; re-inspect before mounting' }; if (Get-Volume -DriveLetter '%s' -ErrorAction SilentlyContinue) { throw 'requested drive letter is already assigned' }; $p = @(Get-Partition -Volume $v -ErrorAction Stop); if ($p.Count -ne 1) { throw 'volume did not resolve to exactly one partition' }; $paths = @($p[0].AccessPaths | Where-Object { $_ -and -not $_.StartsWith('\\\\?\\Volume{', [StringComparison]::OrdinalIgnoreCase) }); if ($paths.Count -gt 0) { throw 'volume already has a mounted-folder access path; refusing to add another' }; Add-PartitionAccessPath -InputObject $p[0] -AccessPath '%s:' -ErrorAction Stop", uniqueID, letter, letter)
	return []string{"powershell", "-NoProfile", "-NonInteractive", "-Command", script}, nil
}
