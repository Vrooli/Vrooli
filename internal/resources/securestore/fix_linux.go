//go:build linux

package securestore

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
)

func absentBackendFix() string {
	return "install libsecret's command client (Debian/Ubuntu: `sudo apt install libsecret-tools`; Fedora: `sudo dnf install libsecret`) and run a Secret Service such as gnome-keyring-daemon --components=pkcs11,secrets"
}

func nativeStorageStrength() (string, string) {
	dir, err := DefaultKeyringDir()
	if err != nil {
		return "", ""
	}
	paths, err := filepath.Glob(filepath.Join(dir, "*.keyring"))
	if err != nil || len(paths) == 0 {
		return "", ""
	}
	for _, path := range paths {
		contents, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if strings.HasPrefix(string(contents), "[keyring]") {
			return "unencrypted-keyring", "values are readable with a text editor; file mode is the only protection"
		}
	}
	return "encrypted-keyring", "the keyring file is not readable as a plaintext GKeyFile"
}

// tpmResourceManagers are the TPM device nodes systemd-creds uses, in the order
// it prefers them. The resource manager is the one an unprivileged process can
// be granted, so it is checked first.
var tpmResourceManagers = []string{"/dev/tpmrm0", "/dev/tpm0"}

// hostBoundFix explains why systemd-creds could not protect a key, in terms of
// something the operator can change.
//
// This exists because the common failure is not "no TPM" but "a TPM this
// process may not open": distributions ship /dev/tpmrm0 owned by a tss group,
// and the systemd host secret is root-only. Both then fail with a bare
// "Permission denied", which reads as a broken host rather than as one group
// membership away from working. Getting this wrong costs an operator the
// unattended reboot that the host-bound wrap exists to provide.
//
// It names `vrooli setup` and never a raw privileged command. Setup is the one
// place this project performs privileged host changes, so an operator who is
// told to run usermod by hand is being handed work the product owes them — and
// a host repaired that way is a host whose state no config describes.
//
// It returns "" when it cannot identify a concrete action, so a caller never
// prints a guess.
func hostBoundFix() string {
	for _, device := range tpmResourceManagers {
		info, err := os.Stat(device)
		if err != nil {
			continue
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			continue
		}
		// Only a group-readable device can be fixed by joining a group. If the
		// mode grants the group nothing, membership would change nothing and
		// saying otherwise would waste the operator's time.
		if info.Mode().Perm()&0o060 == 0 {
			continue
		}
		if slices.Contains(currentGroupIDs(), int(stat.Gid)) {
			// Already a member, so group access is not what is blocking this.
			continue
		}
		return fmt.Sprintf(
			"%s is readable only by group %s, and %s is not in it; run `vrooli setup` to grant it. Until then the passphrase wrap is the only way to open the store, and an unattended reboot cannot unlock it",
			device, groupLabel(int(stat.Gid)), userLabel())
	}
	return ""
}

func currentGroupIDs() []int {
	groups, err := os.Getgroups()
	if err != nil {
		return nil
	}
	return append(groups, os.Getgid())
}

// groupLabel prefers the group name because that is what usermod takes; the
// numeric id is the honest fallback on a host with no matching group entry.
func groupLabel(gid int) string {
	if group, err := user.LookupGroupId(strconv.Itoa(gid)); err == nil && group.Name != "" {
		return group.Name
	}
	return strconv.Itoa(gid)
}

func userLabel() string {
	if current, err := user.Current(); err == nil && current.Username != "" {
		return current.Username
	}
	return "$USER"
}
