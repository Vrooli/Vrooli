package sysmounts

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// mountInfo is the gopsutil-free shape the classifier operates on, so the
// classification logic is unit-testable without importing gopsutil.
type mountInfo struct {
	Device     string
	Mountpoint string
	Fstype     string
	Opts       []string
}

// classifier owns removable/network classification. Its two injectable fields
// (goos and sysBlockRoot) let tests exercise the Linux removable path on any
// host without touching real sysfs.
type classifier struct {
	goos         string
	sysBlockRoot string
}

func newClassifier() *classifier {
	return &classifier{goos: runtime.GOOS, sysBlockRoot: "/sys/block"}
}

// pseudoFS is the set of non-storage filesystem types we never surface as
// volumes. Anything here is kernel/virtual plumbing, not a place to back up to.
var pseudoFS = map[string]struct{}{
	"proc": {}, "sysfs": {}, "tmpfs": {}, "devtmpfs": {}, "devpts": {},
	"cgroup": {}, "cgroup2": {}, "overlay": {}, "squashfs": {}, "mqueue": {},
	"debugfs": {}, "tracefs": {}, "securityfs": {}, "pstore": {}, "bpf": {},
	"configfs": {}, "fusectl": {}, "hugetlbfs": {}, "autofs": {}, "binfmt_misc": {},
	"ramfs": {}, "rpc_pipefs": {}, "nsfs": {}, "none": {}, "fuse.gvfsd-fuse": {},
	"fuse.portal": {}, "efivarfs": {}, "selinuxfs": {},
}

// networkFSPrefixes classify a mount as network by filesystem type prefix.
var networkFSPrefixes = []string{"nfs", "cifs", "smb", "afpfs", "fuse.sshfs", "fuse.rclone", "9p", "ceph", "glusterfs", "webdav"}

func (c *classifier) isPseudoFS(fstype string) bool {
	_, ok := pseudoFS[strings.ToLower(strings.TrimSpace(fstype))]
	return ok
}

// classify returns the drive class and removable flag for a mount.
func (c *classifier) classify(m mountInfo) (DriveClass, bool) {
	fstype := strings.ToLower(m.Fstype)
	for _, p := range networkFSPrefixes {
		if strings.HasPrefix(fstype, p) {
			return ClassNetwork, false
		}
	}
	if c.isRemovable(m) {
		return ClassRemovable, true
	}
	return ClassFixed, false
}

// isRemovable applies per-OS heuristics. gopsutil reports neither removability
// nor drive type portably, so this is the OS-specific seam.
func (c *classifier) isRemovable(m mountInfo) bool {
	switch c.goos {
	case "linux":
		return c.linuxRemovable(m)
	case "darwin":
		// External media mounts under /Volumes; the root volume ("/") and the
		// system-managed data volume are not removable.
		return strings.HasPrefix(m.Mountpoint, "/Volumes/")
	case "windows":
		// Without a portable drive-type API here, treat the system drive as
		// fixed and leave others fixed too; Windows removable detection is a
		// documented follow-up. Conservative: never falsely flag fixed disks.
		return false
	default:
		return false
	}
}

// linuxRemovable reads /sys/block/<base>/removable for the partition's backing
// block device, unioned with the conventional removable mountpoints. The
// sysBlockRoot is injectable so this is testable against a fixture tree.
func (c *classifier) linuxRemovable(m mountInfo) bool {
	if strings.HasPrefix(m.Mountpoint, "/media/") || strings.HasPrefix(m.Mountpoint, "/run/media/") {
		return true
	}
	base := c.baseBlockDevice(m.Device)
	if base == "" {
		return false
	}
	data, err := os.ReadFile(filepath.Join(c.sysBlockRoot, base, "removable"))
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == "1"
}

// baseBlockDevice maps a partition device (e.g. /dev/sda1, /dev/nvme0n1p2) to
// its backing whole-disk device name (sda, nvme0n1) by matching against the
// entries actually present under sysBlockRoot — the longest matching prefix
// wins, which handles both sdaN and nvme0n1pN naming without regex guesswork.
func (c *classifier) baseBlockDevice(device string) string {
	dev := filepath.Base(strings.TrimSpace(device))
	if dev == "" || dev == "." || dev == "/" {
		return ""
	}
	entries, err := os.ReadDir(c.sysBlockRoot)
	if err != nil {
		return ""
	}
	best := ""
	for _, e := range entries {
		name := e.Name()
		if dev == name || strings.HasPrefix(dev, name) {
			if len(name) > len(best) {
				best = name
			}
		}
	}
	return best
}

// isReadOnly reports whether the mount options include `ro`.
func isReadOnly(opts []string) bool {
	for _, o := range opts {
		if strings.EqualFold(strings.TrimSpace(o), "ro") {
			return true
		}
	}
	return false
}
