//go:build linux

package securestore

import (
	"os"
	"slices"
	"strings"
	"syscall"
	"testing"
)

// The host-bound wrap is what makes an unattended reboot possible, so the
// failure that costs an operator that capability must name the action that
// restores it. A bare "Permission denied" reads as a host that cannot do this;
// the truth is usually a host one group membership away.
func TestHostBoundFixNamesTheGroupBlockingTheTPM(t *testing.T) {
	device, gid := groupAccessibleTPM(t)
	if slices.Contains(currentGroupIDs(), gid) {
		t.Skipf("this process is already in group %s, so group access is not blocking the TPM", groupLabel(gid))
	}

	fix := hostBoundFix()
	if fix == "" {
		t.Fatalf("%s is group-accessible and this process is not in its group, but no fix was reported", device)
	}
	for _, want := range []string{device, groupLabel(gid), "vrooli setup"} {
		if !strings.Contains(fix, want) {
			t.Fatalf("fix = %q, want it to name %q", fix, want)
		}
	}
	// An operator who reads only the fix must still learn what it costs them
	// until they act, or they will not know the reboot story is affected.
	if !strings.Contains(fix, "passphrase") {
		t.Fatalf("fix = %q, want it to say the passphrase wrap is the only option until the group changes", fix)
	}
}

// The two ways a process can be outside the TPM group need two different
// answers, and giving the wrong one costs the operator the fix. An account that
// was never granted the group is told to run setup, which grants it. An account
// that already holds the grant is in a running process that cannot see it,
// because supplementary groups are attached at login — telling that operator to
// run setup "to grant it" sends them to a command that correctly reports the
// membership already present and changes nothing.
func TestHostBoundFixSeparatesAStaleSessionFromAMissingGrant(t *testing.T) {
	_, gid := groupAccessibleTPM(t)
	if slices.Contains(currentGroupIDs(), gid) {
		t.Skipf("this process is already in group %s, so no grant is outstanding", groupLabel(gid))
	}

	fix := hostBoundFix()
	if accountInGroupID(gid) {
		if !strings.Contains(fix, "log out") {
			t.Fatalf("fix = %q, want it to tell an account that already holds the grant to start a new session", fix)
		}
		if strings.Contains(fix, "to grant it") {
			t.Fatalf("fix = %q, want it not to ask for a grant this account already holds", fix)
		}
		return
	}
	if !strings.Contains(fix, "to grant it") {
		t.Fatalf("fix = %q, want it to ask setup to grant a membership this account does not hold", fix)
	}
}

// The grant an operator holds but a running process cannot use is the one setup
// can act on within the same run, and it is the only group this ever names.
func TestPendingGroupGrantReportsOnlyAnUnusableTPMGrant(t *testing.T) {
	_, gid := groupAccessibleTPM(t)
	pending := PendingGroupGrant()
	switch {
	case slices.Contains(currentGroupIDs(), gid):
		if pending != "" {
			t.Fatalf("PendingGroupGrant() = %q, want \"\" when this process can already use the group", pending)
		}
	case accountInGroupID(gid):
		if pending != groupLabel(gid) {
			t.Fatalf("PendingGroupGrant() = %q, want %q", pending, groupLabel(gid))
		}
	default:
		if pending != "" {
			t.Fatalf("PendingGroupGrant() = %q, want \"\" when the account holds no grant to pick up", pending)
		}
	}
}

// A fix is only honest when membership would actually change the outcome. On a
// host with no TPM at all there is nothing to join, and the caller must print
// nothing rather than send the operator after a group that will not help.
func TestHostBoundFixStaysSilentWithoutAFixableDevice(t *testing.T) {
	for _, device := range tpmResourceManagers {
		if _, err := os.Stat(device); err == nil {
			t.Skip("this host has a TPM device, so the no-device case cannot be observed here")
		}
	}
	if fix := hostBoundFix(); fix != "" {
		t.Fatalf("hostBoundFix() = %q on a host with no TPM device, want silence", fix)
	}
}

func groupAccessibleTPM(t *testing.T) (string, int) {
	t.Helper()
	for _, device := range tpmResourceManagers {
		info, err := os.Stat(device)
		if err != nil {
			continue
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || info.Mode().Perm()&0o060 == 0 {
			continue
		}
		return device, int(stat.Gid)
	}
	t.Skip("no group-accessible TPM device on this host")
	return "", 0
}
