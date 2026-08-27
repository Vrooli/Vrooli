package volumeremediation

import (
	"fmt"
	"path/filepath"
	"strings"
)

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
		return repairVolumeCommand(req)
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
		// udisks selects the driver and per-filesystem options that make the
		// volume usable by its owner. Hand-rolling those options here would
		// duplicate policy udisks already owns and gets right.
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
		return []string{"mount", "-t", strings.ToLower(strings.TrimSpace(req.Device.Filesystem)), "-o", "rw", device, mountpoint}, nil
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
	case "exfat":
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

func repairVolumeCommand(req Request) ([]string, error) {
	letter, err := windowsDriveLetter(req.Device.Path)
	if err != nil {
		return nil, err
	}
	switch req.Action {
	case ActionCheck:
		return []string{"powershell", "-NoProfile", "-NonInteractive", "-Command", "Repair-Volume -DriveLetter " + letter + " -Scan"}, nil
	case ActionRepair:
		return []string{"powershell", "-NoProfile", "-NonInteractive", "-Command", "Repair-Volume -DriveLetter " + letter + " -OfflineScanAndFix"}, nil
	case ActionUnmount, ActionMountReadWrite:
		// Windows attaches and detaches volumes itself, and Repair-Volume
		// operates on an online volume. Exposing a no-op mount action here
		// would imply control the platform does not give us.
		return nil, ErrUnsupported{
			Reason:          "Windows manages volume attachment itself; there is no mount step to drive",
			OperatorCommand: "Repair-Volume -DriveLetter " + letter + " -OfflineScanAndFix",
		}
	default:
		return nil, ErrUnsupported{Reason: fmt.Sprintf("Windows backend has no command for %q", req.Action)}
	}
}
