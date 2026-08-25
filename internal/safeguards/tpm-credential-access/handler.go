// Package tpmcredentialaccess grants the operator account use of the host TPM,
// so the encrypted credential store can open itself after a reboot.
//
// Why this is a setup safeguard and not advice printed by the credential layer:
// distributions ship /dev/tpmrm0 readable only by a tss group, and an account
// outside that group is refused. The credential store then falls back to the
// passphrase wrap, which means a human types a passphrase after every boot —
// exactly what the host-bound wrap exists to avoid. Fixing that is a privileged
// change to host state, and `vrooli setup` is the one place this project makes
// those. An operator who is told to run `usermod` by hand is being handed work
// the product owes them, and leaves the host in a state no config describes.
package tpmcredentialaccess

import (
	"fmt"
	"os"
	"os/user"
	"slices"
	"strconv"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

// tpmDevices are the TPM character devices systemd-creds can use, preferred
// first. The resource manager is the one a non-root account can be granted;
// the raw device is the fallback a host may expose instead.
var tpmDevices = []string{"/dev/tpmrm0", "/dev/tpm0"}

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

// These seams keep the safeguard matrix testable without manufacturing device
// nodes or changing the process supplementary groups. In production they
// point at the host probes below; tests replace them with deterministic host
// states and still observe the exact privileged argv.
var (
	grantableDeviceFn = grantableDevice
	accountInGroupFn  = accountInGroup
)

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.SupportClass = hostreqkit.SupportSupported

	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}

	if host.OS != "linux" {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "TPM credential access is a Linux concern; other platforms use a native credential store")
		return status
	}

	// A host with no TPM is not a broken host. The encrypted store still works
	// through the passphrase wrap, and there is nothing here to grant.
	device, group, found := grantableDeviceFn()
	if !found {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes,
			"no TPM device this host can grant by group; the encrypted credential store uses its passphrase wrap")
		return status
	}

	account := hostreqkit.InvokingUser()
	if account == "" {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.Notes = append(status.Notes,
			"cannot identify the operator account to grant; re-run setup as that account rather than as root")
		return status
	}

	member, err := accountInGroupFn(account, group)
	if err != nil {
		status.Notes = append(status.Notes, "could not read group membership for "+account+": "+err.Error())
		return status
	}
	if member {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes,
			fmt.Sprintf("%s is in group %s, so %s opens the credential store unattended", account, group, device))
		return status
	}

	status.Notes = append(status.Notes, fmt.Sprintf(
		"pending: add %s to group %s so %s opens the credential store without a passphrase at every boot",
		account, group, device))
	return status
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	switch status.SupportClass {
	case hostreqkit.SupportUnsupported:
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		return status, nil
	case hostreqkit.SupportNotApplicable:
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		return status, nil
	case hostreqkit.SupportManualOnly:
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.Notes = append(status.Notes, "manual safeguard action required by manifest declaration")
		return status, nil
	}

	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}

	device, group, found := grantableDeviceFn()
	if !found {
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		return status, nil
	}
	account := hostreqkit.InvokingUser()
	if account == "" {
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.Notes = append(status.Notes, "no operator account to grant")
		return status, nil
	}

	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, fmt.Sprintf("dry-run: would add %s to group %s for %s", account, group, device))
		return status, nil
	}

	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "usermod", []string{"-aG", group, account}, opts); err != nil {
		// A refused privilege and a failed usermod are different conditions
		// with different operator actions, and collapsing them into one
		// "failed" sends an operator who chose --sudo-mode=skip looking for a
		// broken host. This safeguard is required, so either way setup reports
		// that it did not reach the unattended state — but it names the reason
		// the operator can act on.
		if hostreqkit.IsSudoSkipped(err) {
			status.ExecutionState = hostreqkit.ExecutionManualActionRequired
			status.BlockingReason = hostreqkit.BlockingNeedsSudo
			status.Notes = append(status.Notes, fmt.Sprintf(
				"granting %s access to %s needs privilege; re-run as `vrooli setup --sudo-mode=ask` to finish unattended credential access", account, device))
			return status, nil
		}
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, fmt.Sprintf("failed to add %s to group %s: %s", account, group, err.Error()))
		return status, nil
	}

	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	// The grant is real but this process cannot see it: supplementary groups
	// are fixed at login. That is a fact about this process, not a task for the
	// operator — setup's credential stage runs later in the same invocation and
	// enters a group set that does include the grant, so it adds the unattended
	// wrap without anyone logging out.
	status.Notes = append(status.Notes, fmt.Sprintf(
		"added %s to group %s; setup's credential stage picks the grant up later in this same run and adds the unattended wrap",
		account, group))
	return status, nil
}

// grantableDevice reports the first TPM device whose access can actually be
// granted by joining a group, with that group's name.
//
// A device whose mode gives the group nothing is deliberately not grantable:
// adding an account to its group would change nothing, and reporting otherwise
// would send an operator after a fix that cannot work.
func grantableDevice() (device string, group string, found bool) {
	for _, candidate := range tpmDevices {
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}
		gid, ok := fileInfoGroupID(info)
		if !ok || info.Mode().Perm()&0o060 == 0 {
			continue
		}
		return candidate, groupLabel(int(gid)), true
	}
	return "", "", false
}

// accountInGroup asks the OS about the named account rather than reading this
// process's own group set. Under `sudo vrooli setup` the process is root, whose
// membership says nothing about the operator's.
func accountInGroup(account, group string) (bool, error) {
	target, err := user.Lookup(account)
	if err != nil {
		return false, err
	}
	groupIDs, err := target.GroupIds()
	if err != nil {
		return false, err
	}
	wanted, err := user.LookupGroup(group)
	if err != nil {
		return false, err
	}
	return slices.Contains(groupIDs, wanted.Gid), nil
}

func groupLabel(gid int) string {
	if group, err := user.LookupGroupId(strconv.Itoa(gid)); err == nil && group.Name != "" {
		return group.Name
	}
	return strconv.Itoa(gid)
}
