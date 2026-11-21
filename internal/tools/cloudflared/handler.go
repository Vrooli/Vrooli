package cloudflared

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/resources"
)

// NewHandler projects the resource-owned pinned artifact into host requirements.
// Dependency policy admits writes; this adapter never installs through APT/PATH.
func NewHandler(manifest hostreqkit.ToolManifest) hostreqkit.Handler {
	return newHandler(manifest, selectedArtifact, hostreqkit.RunAsInvokingUser)
}

func selectedArtifact(ctx context.Context) (resources.ManagedServiceArtifact, error) {
	root, err := buildinfo.ResolveSourceRoot()
	if err != nil {
		return resources.ManagedServiceArtifact{}, err
	}
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return resources.ManagedServiceArtifact{}, err
	}
	return resources.NewController(root, home).InspectManagedServiceArtifact(ctx, "cloudflared")
}

func newHandler(manifest hostreqkit.ToolManifest, inspect func(context.Context) (resources.ManagedServiceArtifact, error), run func(string, []string, hostreqkit.EnsureOptions) error) hostreqkit.Handler {
	return hostreqkit.InstallerHandler{
		Manifest: manifest, KindValue: hostreqspec.KindTool,
		InspectFunc: func(host hostreqkit.Host, req hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
			status := hostreqkit.BaseStatus(req)
			switch host.OS {
			case "linux", "darwin", "macos", "windows":
			default:
				return hostreqkit.UnsupportedRequirementStatus(req, "cloudflared has no pinned artifact for this host")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			info, err := inspect(ctx)
			status.Version = info.Version
			status.InstallSupported = info.Version != ""
			if err != nil {
				status.Notes = append(status.Notes, err.Error())
				return status
			}
			status.Command, status.Installed, status.ExecutionState = info.Path, true, hostreqkit.ExecutionAlreadyPresent
			return status
		},
		ApplyFunc: func(status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
			if status, done := hostreqkit.ApplyStatus(status); done {
				return status, nil
			}
			if status.Version == "" {
				return status, fmt.Errorf("cloudflared pinned resource target could not be resolved; inspect its acquisition contract")
			}
			args := []string{"deps", "install", "source/cloudflared@" + status.Version, "--scenario", "cloudflared", "--surface", "host-tool", "--exact", "--json"}
			if opts.DryRun {
				status.ExecutionState = hostreqkit.ExecutionWouldInstall
				status.Notes = append(status.Notes, "scenario-dependency-analyzer "+strings.Join(args, " "))
				return status, nil
			}
			if err := run("scenario-dependency-analyzer", append(args, "--apply"), opts); err != nil {
				return status, err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			info, err := inspect(ctx)
			if err != nil {
				return status, err
			}
			status.Command, status.Version, status.Installed, status.ExecutionState = info.Path, info.Version, true, hostreqkit.ExecutionInstalled
			return status, nil
		},
	}
}
