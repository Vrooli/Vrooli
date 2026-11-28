package agentinstall

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/vrooli/cli-core/agentcatalog"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/internal/cliinstall"
)

// ManageHarness is the shared local lifecycle boundary for the control plane
// and desktop consumers. Authority belongs to the caller; filesystem ownership,
// dependency admission, receipts and rollback stay with the existing owners.
func ManageHarness(ctx context.Context, root, resource, action, actor, cleanupPlanID string) (HarnessActionResult, error) {
	return ManageHarnessWithChannel(ctx, root, resource, "", action, actor, cleanupPlanID)
}

// ManageHarnessWithChannel is the channel-aware lifecycle boundary used by the
// update command. An empty channel follows the installed major, so a host that
// has already moved to a non-default line is never silently downgraded by a
// plain `update`.
func ManageHarnessWithChannel(ctx context.Context, root, resource, channel, action, actor, cleanupPlanID string) (HarnessActionResult, error) {
	deps := defaultHarnessDeps()
	deps.record = recordHarnessArtifact
	return manageHarnessWithChannel(ctx, root, resource, channel, action, actor, cleanupPlanID, deps)
}

func manageHarness(ctx context.Context, root, resource, action, actor, cleanupPlanID string, deps harnessDeps) (HarnessActionResult, error) {
	return manageHarnessWithChannel(ctx, root, resource, "", action, actor, cleanupPlanID, deps)
}

func manageHarnessWithChannel(ctx context.Context, root, resource, channel, action, actor, cleanupPlanID string, deps harnessDeps) (HarnessActionResult, error) {
	h, err := resolveHarnessForAction(ctx, root, resource, channel, deps)
	if err != nil {
		return HarnessActionResult{}, err
	}
	deps.actor = actor
	switch action {
	case "check", "doctor":
		report := inspectHarness(ctx, h, deps)
		return HarnessActionResult{Action: action, Success: report.State != StateCheckFailed, Message: report.Reason, Report: report}, nil
	case "refresh_catalog", "refresh-catalog":
		result, err := agentcatalog.RefreshCodingRoleCatalogBy(ctx, resource, filepath.Join(root, "resources", resource, "model-policy.json"), time.Now(), actor)
		if err != nil {
			return HarnessActionResult{Action: action, Message: err.Error()}, nil
		}
		return HarnessActionResult{Action: action, Success: true, Message: fmt.Sprintf("Model list refreshed from %s.", result.Evidence)}, nil
	case "install":
		unlock, err := deps.platform.acquireLock(ctx, "resource-update-"+resource)
		if err != nil {
			return HarnessActionResult{}, err
		}
		defer unlock()
		_, version, err := latestHarnessReleaseWithDeps(ctx, h, deps.platform)
		if err != nil {
			return HarnessActionResult{Action: action, Message: err.Error()}, nil
		}
		change := &harnessChange{}
		if err := h.installRelease(ctx, version, deps, change); err != nil {
			_ = change.revert(context.WithoutCancel(ctx))
			return HarnessActionResult{Action: action, Message: err.Error()}, nil
		}
		change.commit()
		return HarnessActionResult{Action: action, Success: true, Message: "Harness installed in your user-owned prefix.", Report: inspectHarness(ctx, h, deps)}, nil
	case "preview_cleanup", "uninstall":
		plan, err := PreviewCleanup(root, resource)
		if err != nil {
			return HarnessActionResult{}, err
		}
		if action == "preview_cleanup" {
			return HarnessActionResult{Action: action, Success: true, CleanupPlan: &plan}, nil
		}
		if cleanupPlanID == "" || cleanupPlanID != plan.ID {
			return HarnessActionResult{Action: action, Message: "Cleanup preview changed; review the managed paths again."}, nil
		}
		if err := ApplyCleanup(ctx, root, plan); err != nil {
			return HarnessActionResult{Action: action, Message: err.Error()}, nil
		}
		return HarnessActionResult{Action: action, Success: true, Message: "Removed the previewed managed copies. Independently installed copies were preserved.", CleanupPlan: &plan}, nil

	default:
		return runHarnessAction(ctx, h, action, deps), nil
	}
}

// resolveHarnessForAction selects the declared channel. An explicit channel is
// authoritative. Without one, the installed major decides: a V2 host stays on
// its channel so a plain `update` cannot downgrade it.
func resolveHarnessForAction(ctx context.Context, root, resource, channel string, deps harnessDeps) (Harness, error) {
	if channel != "" {
		return LoadHarnessChannel(root, resource, channel)
	}
	h, err := LoadHarness(root, resource)
	if err != nil {
		return Harness{}, err
	}
	selected, _ := h.selectedCopy(deps)
	if selected == "" {
		return h, nil
	}
	version := deps.version
	if version == nil {
		version = func(ctx context.Context, path string) (string, error) {
			return installedVersion(ctx, path, deps.platform)
		}
	}
	raw, err := version(ctx, selected)
	if err != nil {
		return h, nil
	}
	major, ok := MajorVersion(raw)
	if !ok || major < 2 {
		return h, nil
	}
	selectedChannel, err := ChannelForMajor(root, resource, major)
	if err != nil {
		// The installed copy may be unmanaged or newer than every declared
		// channel. Reporting the default channel keeps the failure visible
		// without inventing a release line.
		return h, nil
	}
	return LoadHarnessChannel(root, resource, selectedChannel)
}

func recordHarnessArtifact(home, resource, path string, directory bool) error {
	entry := cliinstall.InstallEntry{Scope: cliinstall.ScopeRuntime, Kind: cliinstall.EntryFile, Path: path, Prefix: home, Resource: resource}
	if directory {
		entry.Kind, entry.Prefix = cliinstall.EntryDirectory, path
	}
	return cliinstall.RecordToolArtifacts(home, entry)
}

// DirectInstallCommand exposes the shell-free installer to a resource CLI.
func DirectInstallCommand(spec Spec) cliapp.Command {
	return cliapp.Command{
		Name: "install-direct", Description: "Install the upstream CLI into the user-owned prefix",
		Run: func(args []string) error {
			if err := requireNoInstallArgs("install-direct", args); err != nil {
				return err
			}
			if spec.Acquisition == nil {
				acquisition, err := acquisitionFromSourceManifest(spec.ResourceName)
				if err != nil {
					return err
				}
				spec.Acquisition = acquisition
			}
			return Install(context.Background(), spec)
		},
	}
}

func requireNoInstallArgs(command string, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("%s accepts no arguments; preview through scenario-dependency-analyzer deps install", command)
	}
	return nil
}
