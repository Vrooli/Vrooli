package execplan

import (
	"path"
	"strconv"
	"strings"
)

// layout resolves the target paths every scope shares.
type layout struct {
	workdir       string
	setupWorkdir  string
	bundleDir     string
	archive       string
	manifestPath  string
	nativeCLIPath string
	releaseDir    string
}

func layoutFor(in CompileInputs) layout {
	workdir := strings.TrimSpace(in.Manifest.Target.VPS.Workdir)
	setupRelease := strings.TrimSpace(in.ReleaseArtifacts.ID)
	if setupRelease == "" {
		setupRelease = in.Release.Digest
	}
	bundleDir := remoteJoin(workdir, ".vrooli", "cloud", "bundles", releaseDirName(in.Release.Digest))
	archiveName := path.Base(in.ArtifactPath)
	if strings.TrimSpace(in.ArtifactPath) == "" {
		archiveName = "bundle.tar.gz"
	}
	return layout{
		workdir:       workdir,
		setupWorkdir:  remoteJoin(path.Dir(workdir), ".vrooli", "cloud", "deployments", in.Deployment.ID, "releases", releaseDirName(setupRelease)),
		bundleDir:     bundleDir,
		archive:       remoteJoin(bundleDir, archiveName),
		manifestPath:  remoteJoin(bundleDir, "release-manifest.json"),
		nativeCLIPath: remoteJoin(workdir, ".vrooli", "bin", "vrooli"),
		releaseDir:    remoteJoin(workdir, ".vrooli", "cloud", "releases", releaseDirName(in.Release.Digest)),
	}
}

func installActions(in CompileInputs) []Action {
	lay := layoutFor(in)
	targetScenarios := targetScenarioIDs(in.Manifest)
	legacy := legacyInventory(in)

	actions := []Action{{
		// Release delivery is the bootstrap boundary. It places the
		// deployment-local vrooli binary before any target owner verb runs,
		// so a stopped or partially removed VPS can still be repaired.
		ID:                 OpReleaseDeliver,
		OwnerOperation:     OpReleaseDeliver,
		Effect:             EffectDeploymentWrite,
		RequiredCapability: "reach:deliver",
		Inputs: map[string]string{
			"artifact_ref":     in.Release.Digest,
			"release_id":       in.ReleaseArtifacts.ID,
			"bundle_sha256":    bundleSHA(in),
			"artifact_path":    in.ArtifactPath,
			"destination":      lay.archive,
			"release_manifest": lay.manifestPath,
			"native_cli":       lay.nativeCLIPath,
		},
		DependsOn:    []string{},
		Verification: "delivered_sha256_matches",
		Recovery:     "redeliver",
		Retry:        RetrySafeReplay,
		CancelPoint:  true,
	}, {
		ID:                 OpHostPrepare,
		OwnerOperation:     OpHostPrepare,
		Effect:             EffectHostWrite,
		RequiredCapability: "privilegebroker:apt.packages.ensure",
		Inputs: map[string]string{
			"packages":   strings.Join(hostPreparePackages(in.Manifest.Edge.Caddy.Enabled || strings.TrimSpace(in.Manifest.Edge.Domain) != ""), ","),
			"workdir":    lay.workdir,
			"bundle_dir": lay.bundleDir,
		},
		DependsOn:    []string{OpReleaseDeliver},
		Verification: "packages_present",
		Recovery:     "none_required",
		Retry:        RetrySafeReplay,
		CancelPoint:  true,
	}}
	if in.Manifest.Edge.Caddy.Enabled {
		actions = append(actions, Action{
			ID:                 OpEdgeFirewallAllow,
			OwnerOperation:     OpEdgeFirewallAllow,
			Effect:             EffectEdgeWrite,
			RequiredCapability: "privilegebroker:edge.ufw.allow",
			Inputs:             map[string]string{"ports": "80,443", "protocol": "tcp"},
			DependsOn:          []string{OpHostPrepare},
			Verification:       "rules_present_or_firewall_inactive",
			Recovery:           "none_required",
			Retry:              RetrySafeReplay,
			CancelPoint:        true,
		})
	}
	actions = append(actions,
		Action{
			ID:                 OpDataInventory,
			OwnerOperation:     OpDataInventory,
			Effect:             EffectNone,
			RequiredCapability: "cloud-target:data.inventory",
			Inputs: map[string]string{
				"workdir":         lay.workdir,
				"scenarios":       strings.Join(targetScenarios, ","),
				"bindings":        strings.Join(persistentBindings(in), ","),
				"data_bindings":   strings.Join(dataBindingSpecs(in), ","),
				"legacy_preserve": strings.Join(legacy, ","),
				"mutable_names":   strings.Join(LegacyMutableDirNames, ","),
			},
			DependsOn:    []string{OpHostPrepare},
			Verification: "inventory_recorded",
			Recovery:     "none_required",
			Retry:        RetrySafeReplay,
			CancelPoint:  true,
		},
		Action{
			ID:                 OpReleaseVerify,
			OwnerOperation:     OpReleaseVerify,
			Effect:             EffectNone,
			RequiredCapability: "cloud-target:release.verify",
			Inputs: map[string]string{
				"release_digest":   in.Release.Digest,
				"release_id":       in.ReleaseArtifacts.ID,
				"bundle_sha256":    bundleSHA(in),
				"archive":          lay.archive,
				"release_manifest": lay.manifestPath,
			},
			DependsOn:    []string{OpReleaseDeliver},
			Verification: "archive_sha256_matches_release",
			Recovery:     "none_required",
			Retry:        RetrySafeReplay,
			CancelPoint:  true,
		},
		Action{
			ID:                 OpReleaseStage,
			OwnerOperation:     OpReleaseStage,
			Effect:             EffectDeploymentWrite,
			RequiredCapability: "cloud-target:release.stage",
			Inputs: map[string]string{
				"release_digest":   in.Release.Digest,
				"release_id":       in.ReleaseArtifacts.ID,
				"archive":          lay.archive,
				"release_manifest": lay.manifestPath,
				"release_dir":      lay.releaseDir,
			},
			DependsOn:    []string{OpReleaseVerify},
			Verification: "staged_tree_complete",
			Recovery:     "discard_owned_inactive_stage",
			Retry:        RetrySafeReplay,
			CancelPoint:  true,
		},
		Action{
			ID:                 OpConfigApply,
			OwnerOperation:     OpConfigApply,
			Effect:             EffectHostWrite,
			RequiredCapability: "vrooli:setup",
			Inputs: map[string]string{
				"workdir":             lay.workdir,
				"setup_workdir":       lay.setupWorkdir,
				"setup_native_cli":    in.ReleaseArtifacts.NativeCLI,
				"environment":         "production",
				"selection":           "setup/v1",
				"selection_json_b64":  selectionJSON(in),
				"scenario_id":         in.Manifest.Scenario.ID,
				"resources":           strings.Join(sortedUnique(in.Manifest.Dependencies.Resources), ","),
				"scenarios":           strings.Join(sortedUnique(in.Manifest.Dependencies.Scenarios), ","),
				"autoheal":            strconv.FormatBool(in.Manifest.Bundle.IncludeAutoheal),
				"autoheal_scope_path": remoteJoin(lay.workdir, ".vrooli", "cloud", "autoheal-scope.json"),
			},
			DependsOn:    []string{OpReleaseStage},
			Verification: "vrooli_setup_completed",
			Recovery:     "rerun_setup",
			Retry:        RetrySafeReplay,
			CancelPoint:  true,
		},
	)
	return actions
}
