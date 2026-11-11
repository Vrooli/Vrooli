package generatedtreeownership

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

// Seams. Setup runs as root, and the files belong to the invoking operator.
var (
	repoRoot    = cliutil.ResolveRepoRoot
	rootOwned   = rootOwnedTrees
	invokingIDs = func() (int, int) {
		if uid, gid, ok := hostreqkit.InvokingUserIDs(); ok {
			return uid, gid
		}
		return os.Getuid(), os.Getgid()
	}
	runPrivileged = hostreqkit.RunPrivilegedCommand
)

// generatedTrees are the repository-relative trees a Proto materialization
// publishes or replaces. Globs cover staging directories left beside them.
var generatedTrees = []string{
	"packages/proto/gen",
	"packages/proto/gen.previous",
	"packages/proto/.proto-compatibility-*",
}

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

// NewHandler is wired into the runtime registry as "generated_tree_ownership".
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
		status.Notes = append(status.Notes, "not applicable: Windows has no root-owned file case")
		return status
	}
	trees, err := rootOwned(repoRoot())
	if err != nil {
		return hostreqkit.InvalidConfigStatus(requirement, err.Error())
	}
	if len(trees) > 0 {
		status.Notes = append(status.Notes, "root-owned generated files block builds: "+strings.Join(trees, ", "))
		return status
	}
	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
	status.Notes = append(status.Notes, "generated Proto trees belong to the operator")
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
	trees, err := rootOwned(repoRoot())
	if err != nil {
		return status, err
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "would change ownership of: "+strings.Join(trees, ", "))
		return status, nil
	}
	uid, gid := invokingIDs()
	if uid == 0 {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "refusing to hand files to root: no invoking user is known; run through sudo from your own account")
		return status, nil
	}
	for _, tree := range trees {
		// -h changes a symlink itself, so a link can never redirect the chown.
		if err := runPrivileged(opts.SudoMode, "chown", []string{"-hR", fmt.Sprintf("%d:%d", uid, gid), tree}, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, fmt.Sprintf("chown %s failed: %v", tree, err))
			return status, nil
		}
	}
	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	if len(trees) > 0 {
		status.Notes = append(status.Notes, "handed back to the operator: "+strings.Join(trees, ", "))
	}
	return status, nil
}

// rootOwnedTrees returns each generated tree under root that holds at least one
// root-owned entry. A symlink is judged as itself and never entered.
func rootOwnedTrees(root string) ([]string, error) {
	var trees []string
	for _, pattern := range generatedTrees {
		matches, err := filepath.Glob(filepath.Join(root, pattern))
		if err != nil {
			return nil, err
		}
		for _, tree := range matches {
			found, err := containsRootOwned(tree)
			if err != nil {
				return nil, err
			}
			if found {
				trees = append(trees, tree)
			}
		}
	}
	return trees, nil
}

var errFound = errors.New("root-owned entry found")

func containsRootOwned(tree string) (bool, error) {
	err := filepath.WalkDir(tree, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if ownedByRoot(info) {
			return errFound
		}
		return nil
	})
	if errors.Is(err, errFound) {
		return true, nil
	}
	return false, err
}
