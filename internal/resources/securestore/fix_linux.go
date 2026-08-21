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
	grant, found := tpmDeviceGrant()
	if !found || grant.processMember {
		return ""
	}
	if grant.accountMember {
		// This is a different condition from "not in the group", and conflating
		// the two costs the operator the fix. Supplementary groups are attached
		// by the kernel at login and never change for a running process, so a
		// session that started before the grant reports a live membership as
		// missing. Telling that operator to run setup "to grant it" sends them
		// to a command that correctly reports the grant already present and
		// changes nothing, which is a loop with no exit.
		return fmt.Sprintf(
			"%s is readable only by group %s; %s is a member, but this login session started before the grant and cannot use it. Re-run `vrooli setup`, which applies the wrap from a session that can, or log out and back in. Until then the passphrase wrap is the only way to open the store, and an unattended reboot cannot unlock it",
			grant.device, grant.group, userLabel())
	}
	return fmt.Sprintf(
		"%s is readable only by group %s, and %s is not in it; run `vrooli setup` to grant it and add the unattended wrap in the same run. Until then the passphrase wrap is the only way to open the store, and an unattended reboot cannot unlock it",
		grant.device, grant.group, userLabel())
}

// PendingGroupGrant names a group this operator account already holds that this
// process cannot use, or "" when there is none.
//
// It exists so a caller that is about to run the credential store in a
// subprocess can pick up a grant `vrooli setup` made moments earlier, instead
// of telling the operator to log out. It reports only the TPM device group, so
// there is no input that can aim it at an unrelated group.
func PendingGroupGrant() string {
	grant, found := tpmDeviceGrant()
	if !found || grant.processMember || !grant.accountMember {
		return ""
	}
	return grant.group
}

// tpmGrant is the membership picture for one grantable TPM device. The account
// answer and the process answer are kept apart on purpose: `usermod -aG`
// changes the first immediately and the second never.
type tpmGrant struct {
	device        string
	group         string
	accountMember bool
	processMember bool
}

func tpmDeviceGrant() (tpmGrant, bool) {
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
		gid := int(stat.Gid)
		return tpmGrant{
			device:        device,
			group:         groupLabel(gid),
			accountMember: accountInGroupID(gid),
			processMember: slices.Contains(currentGroupIDs(), gid),
		}, true
	}
	return tpmGrant{}, false
}

// accountInGroupID asks the operating system about this account rather than
// reading this process's own group set, which is what os.Getgroups() returns
// and which is fixed at login.
func accountInGroupID(gid int) bool {
	current, err := user.Current()
	if err != nil {
		return false
	}
	ids, err := current.GroupIds()
	if err != nil {
		return false
	}
	return slices.Contains(ids, strconv.Itoa(gid))
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
