package harnessfileownership

import (
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/resources/agentinstall"
)

// Seams. Setup runs as root, and the files belong to the invoking operator.
var (
	homeDir       = config.HomeDir
	rootOwnedList = agentinstall.RootOwnedHarnessPaths
	invokingIDs   = func() (int, int) {
		if uid, gid, ok := hostreqkit.InvokingUserIDs(); ok {
			return uid, gid
		}
		return os.Getuid(), os.Getgid()
	}
	runPrivileged = hostreqkit.RunPrivilegedCommand
)

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

// NewHandler is wired into the runtime registry as "harness_file_ownership".
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
	if host.OS == string(hostreqspec.PlatformWindows) {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "not applicable: Windows installs have no root-owned file case")
		return status
	}
	home, err := homeDir()
	if err != nil {
		return hostreqkit.InvalidConfigStatus(requirement, err.Error())
	}
	owned, err := rootOwnedList(home)
	if err != nil {
		return hostreqkit.InvalidConfigStatus(requirement, err.Error())
	}
	if len(owned) > 0 {
		status.Notes = append(status.Notes, "root-owned coding-agent files block updates: "+describe(owned))
		return status
	}
	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
	status.Notes = append(status.Notes, "coding-agent files belong to the operator")
	return status
}

func (h handler) Apply(_ hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	switch status.SupportClass {
	case hostreqkit.SupportUnsupported:
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		return status, nil
	case hostreqkit.SupportNotApplicable:
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		return status, nil
	case hostreqkit.SupportManualOnly:
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status, nil
	}
	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}
	home, err := homeDir()
	if err != nil {
		return status, err
	}
	owned, err := rootOwnedList(home)
	if err != nil {
		return status, err
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "would change ownership of: "+describe(owned))
		return status, nil
	}
	uid, gid := invokingIDs()
	if uid == 0 {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "refusing to hand files to root: no invoking user is known; run through sudo from your own account")
		return status, nil
	}
	for _, path := range owned {
		// -h changes a symlink itself, so a link can never redirect the chown.
		if err := runPrivileged(opts.SudoMode, "chown", []string{"-h", fmt.Sprintf("%d:%d", uid, gid), path}, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, fmt.Sprintf("chown %s failed: %v", path, err))
			return status, nil
		}
	}
	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	if len(owned) > 0 {
		status.Notes = append(status.Notes, "handed back to the operator: "+describe(owned))
	}
	return status, nil
}

func describe(paths []string) string {
	const shown = 8
	if len(paths) <= shown {
		return strings.Join(paths, ", ")
	}
	return fmt.Sprintf("%s and %d more", strings.Join(paths[:shown], ", "), len(paths)-shown)
}
