package volumeremediation

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

const (
	safetyParameterA = 256
)

// supportedFilesystems is the allowlist of filesystems this package will act
// on. Membership means a repair adapter exists and its behaviour is understood
// — not merely that the name is spelled correctly. An unlisted filesystem is
// refused rather than handed to a generic fsck that may do the wrong thing.
var supportedFilesystems = map[string]string{
	"ntfs":  "ntfs",
	"ntfs3": "ntfs",
	"exfat": "exfat",
	"vfat":  "vfat",
	"fat":   "vfat",
	"fat32": "vfat",
	"msdos": "vfat",
	"ext2":  "ext",
	"ext3":  "ext",
	"ext4":  "ext",
	"apfs":  "apfs",
	"hfs":   "hfs",
	"hfs+":  "hfs",
}

// filesystemFamily maps a reported filesystem type onto the adapter family
// that repairs it.
func filesystemFamily(fs string) (string, bool) {
	family, ok := supportedFilesystems[strings.ToLower(strings.TrimSpace(fs))]
	return family, ok
}

// SupportedFilesystems lists the accepted filesystem names, for error messages
// and documentation that must not drift from the code.
func SupportedFilesystems() []string {
	out := make([]string, 0, len(supportedFilesystems))
	for name := range supportedFilesystems {
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}

// devicePathPattern accepts the device spellings the supported platforms use
// and nothing else. It exists so a device string can never smuggle an argument,
// a path traversal, or a shell fragment into a command line.
var devicePathPattern = regexp.MustCompile(`^(/dev/[A-Za-z0-9._/-]+|[A-Za-z]:|\\\\\?\\Volume\{[0-9A-Fa-f-]{36}\}\\)$`)
var windowsVolumeGUIDPattern = regexp.MustCompile(`^\\\\\?\\Volume\{[0-9A-Fa-f-]{36}\}\\$`)
var windowsDriveRootPattern = regexp.MustCompile(`^[D-Zd-z]:\\?$`)

// validateDevicePath rejects anything that is not a plain device specifier.
func validateDevicePath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return ErrInvalid{Field: "device.path", Reason: "required"}
	}
	if len(path) > safetyParameterA {
		return ErrInvalid{Field: "device.path", Reason: "implausibly long"}
	}
	if strings.Contains(path, "..") {
		return ErrInvalid{Field: "device.path", Reason: "must not contain a path traversal"}
	}
	if !devicePathPattern.MatchString(path) {
		return ErrInvalid{Field: "device.path", Reason: "must be a plain device path such as /dev/sda1, a drive letter such as E:, or a stable Windows volume GUID"}
	}
	return nil
}

// systemMountpoints are paths whose volume keeps the host running. Unmounting
// or repairing one of these is never remediation — it is an outage. The list is
// matched by prefix so /boot/efi is covered by /boot.
var systemMountpoints = []string{"/", "/boot", "/usr", "/etc", "/var", "/home", "/root", "/nix", "/System", "/Applications", "/Library", "C:"}

// isSystemMountpoint reports whether a mountpoint belongs to the running system.
func isSystemMountpoint(mountpoint string) bool {
	mountpoint = strings.TrimSpace(mountpoint)
	if mountpoint == "" {
		return false
	}
	// filepath.Clean follows the host running the control plane, which means a
	// Windows C:\ path must still be protected when fixtures or shared callers
	// evaluate it from another build target.
	windowsPath := strings.ReplaceAll(mountpoint, `\`, "/")
	if len(windowsPath) >= 2 && windowsPath[1] == ':' && strings.EqualFold(windowsPath[:2], "C:") {
		if len(windowsPath) == 2 || windowsPath[2] == '/' {
			return true
		}
	}
	clean := filepath.Clean(mountpoint)
	if clean == "/" {
		return true
	}
	for _, sys := range systemMountpoints {
		if sys == "/" {
			continue
		}
		if strings.EqualFold(clean, sys) || strings.HasPrefix(strings.ToLower(clean), strings.ToLower(sys)+"/") {
			return true
		}
	}
	return false
}

// permittedMountRoots are the directories a remediated volume may be mounted
// under. Restricting the target keeps a mount action from being redirected to
// shadow a system path.
var permittedMountRoots = []string{"/media", "/run/media", "/mnt", "/Volumes"}

// validateWindowsMountpoint only permits assigning a non-system drive letter.
// Directory mount points have a separate ownership and ACL contract and are
// intentionally not created by automatic recovery.
func validateWindowsMountpoint(mountpoint string) error {
	mountpoint = strings.TrimSpace(mountpoint)
	if !windowsDriveRootPattern.MatchString(mountpoint) {
		return ErrInvalid{Field: "mountpoint", Reason: "Windows recovery requires an unused drive-letter root from D: through Z:"}
	}
	return nil
}

// validateMountpoint rejects a mount target outside the permitted media roots.
func validateMountpoint(mountpoint string) error {
	mountpoint = strings.TrimSpace(mountpoint)
	if mountpoint == "" {
		return ErrInvalid{Field: "mountpoint", Reason: "required"}
	}
	if strings.Contains(mountpoint, "..") {
		return ErrInvalid{Field: "mountpoint", Reason: "must not contain a path traversal"}
	}
	if !filepath.IsAbs(mountpoint) {
		return ErrInvalid{Field: "mountpoint", Reason: "must be an absolute path"}
	}
	clean := filepath.Clean(mountpoint)
	if isSystemMountpoint(clean) {
		return ErrRefused{Reason: "refusing to mount over a system path: " + clean}
	}
	for _, root := range permittedMountRoots {
		if clean == root || strings.HasPrefix(clean, root+"/") {
			return nil
		}
	}
	return ErrRefused{Reason: fmt.Sprintf("mountpoint %s is outside the permitted media roots %s", clean, strings.Join(permittedMountRoots, ", "))}
}

// validateRequest applies every gate that does not need a fresh observation of
// the host. Observation-dependent gates live in the service, which re-reads
// state immediately before acting.
func validateRequest(req Request) error {
	if !req.Action.Known() {
		return ErrInvalid{Field: "action", Reason: "not in the remediation registry"}
	}
	if err := validateDevicePath(req.Device.Path); err != nil {
		return err
	}
	if isSystemMountpoint(req.Device.Mountpoint) {
		return ErrRefused{Reason: "refusing to act on a system volume mounted at " + req.Device.Mountpoint}
	}
	if req.Action == ActionInspect {
		return nil
	}
	if _, ok := filesystemFamily(req.Device.Filesystem); !ok {
		return ErrRefused{Reason: fmt.Sprintf("filesystem %q has no repair adapter; supported: %s", req.Device.Filesystem, strings.Join(SupportedFilesystems(), ", "))}
	}
	if req.Action.Destructive() {
		if !req.Device.StableIdentity() {
			// Without a UUID or serial, a device path alone cannot prove the
			// approved disk is still the one present. Refusing is the only
			// safe answer for an action that writes.
			return ErrRefused{Reason: "a destructive action requires a device UUID or serial to bind the approval to this disk"}
		}
		if !req.AcknowledgeDataLoss {
			return ErrRefused{Reason: "repair can discard inconsistent filesystem metadata and requires an explicit data-loss acknowledgement"}
		}
	}
	if req.Action == ActionMountReadWrite && strings.TrimSpace(req.DesiredMountpoint) != "" {
		var err error
		if windowsDriveRootPattern.MatchString(req.DesiredMountpoint) {
			err = validateWindowsMountpoint(req.DesiredMountpoint)
		} else {
			err = validateMountpoint(req.DesiredMountpoint)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
