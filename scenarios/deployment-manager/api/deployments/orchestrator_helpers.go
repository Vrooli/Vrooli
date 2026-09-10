package deployments

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"deployment-manager/build"
	"deployment-manager/bundles"
	"deployment-manager/profiles"
	"deployment-manager/releases"
)

type publishPipelineRunner interface {
	RunPublishPipelineConnect(context.Context, *PublishPipelineRequest) (*PublishPipelineResponse, error)
	WaitForPipelineConnect(context.Context, string) (*PipelineStatus, error)
}

// publishToLPBS triggers a deploy-stage-only pipeline, extracts version data,
// and persists it. A commercial publication failure is terminal for the
// operation because a warning cannot establish which bytes reached LPBS.
func (o *Orchestrator) publishToLPBS(ctx context.Context, profile *profiles.Profile, req DeployDesktopRequest, response *DeployDesktopResponse, step *OrchestrationStep) {
	ownerEffectPossible := false
	fail := func(message string) {
		o.failStep(step, message)
		o.log("error", map[string]interface{}{"msg": "publish failed", "error": message})
		if !ownerEffectPossible {
			return
		}
		// The owner pipeline has started, so a later transport, receipt, or
		// ledger failure cannot establish whether the external effect happened.
		// Preserve that uncertainty for reconciliation instead of presenting a
		// retryable ordinary failure that could duplicate publication.
		response.Status = "ambiguous"
		if o.releasesRepo != nil && strings.TrimSpace(req.ReleaseID) != "" {
			if err := o.releasesRepo.UpdateStatus(ctx, req.ReleaseID, releases.StatusAmbiguous); err != nil {
				o.log("error", map[string]interface{}{"msg": "persist ambiguous publication status failed", "release_id": req.ReleaseID, "error": err.Error()})
			}
		}
	}
	if strings.TrimSpace(req.ReleaseID) == "" {
		step.Status = "skipped"
		step.Message = "local packaging has no governed release identity; publication was not attempted"
		return
	}
	pipelineReq := &PublishPipelineRequest{
		ScenarioName:            profile.Scenario,
		Platforms:               req.Platforms,
		Publish:                 true,
		ReleaseID:               req.ReleaseID,
		Channel:                 req.Channel,
		ReleaseVersion:          req.ReleaseVersion,
		ArtifactDigest:          req.ArtifactDigest,
		ExpectedArtifactDigests: req.ExpectedArtifactDigests,
		CandidateID:             req.CandidateID,
		DestinationRevisionID:   req.DestinationRevisionID,
		AuthorizationEpoch:      req.AuthorizationEpoch,
		ReadinessReviewKey:      req.ReadinessReviewKey,
	}

	// Pass full LPBS coords on the inline DeployConfig. ReleaseID + Channel
	// must ride on the deploy config so S2D's Config decoder forwards them
	// through to lpbs_client.go. A commercial publication without these exact
	// coordinates would be an unbound owner effect, so fail before launching
	// the pipeline.
	cfg := o.loadLPBSConfigForPublish(ctx, req.ProfileID)
	if cfg == nil || strings.TrimSpace(cfg.LPBSAppKey) == "" || strings.TrimSpace(cfg.LPBSRemoteProfile) == "" {
		fail("commercial publication requires an exact LPBS app key and remote profile")
		return
	}
	if o.releaseIdentity != nil {
		destination, err := o.releaseIdentity.GetDestinationRevision(ctx, req.DestinationRevisionID)
		if err != nil || destination == nil {
			fail(fmt.Sprintf("commercial publication requires the approved destination revision: %v", err))
			return
		}
		if destination.ID != req.DestinationRevisionID || strings.TrimSpace(destination.Revision.DestinationID) != strings.TrimSpace(cfg.LPBSRemoteProfile) {
			fail("commercial publication destination revision does not match the configured LPBS remote profile")
			return
		}
	}
	pipelineReq.DeployConfig = &PublishDeployConfig{
		ScenarioName:            profile.Scenario,
		RemoteProfile:           cfg.LPBSRemoteProfile,
		AppKey:                  cfg.LPBSAppKey,
		UpdateURL:               cfg.UpdateURL,
		ReleaseID:               req.ReleaseID,
		Channel:                 req.Channel,
		ArtifactDigest:          req.ArtifactDigest,
		ExpectedArtifactDigests: req.ExpectedArtifactDigests,
		CandidateID:             req.CandidateID,
		DestinationRevisionID:   req.DestinationRevisionID,
		AuthorizationEpoch:      req.AuthorizationEpoch,
		ReadinessReviewKey:      req.ReadinessReviewKey,
	}

	var client publishPipelineRunner = o.publishPipelineRunner
	if client == nil {
		var err error
		client, err = NewDesktopPackagerClient(o.log)
		if err != nil {
			fail(fmt.Sprintf("could not create desktop client: %v", err))
			return
		}
	}

	pipelineResp, err := client.RunPublishPipelineConnect(ctx, pipelineReq)
	if err != nil {
		fail(fmt.Sprintf("failed to trigger publish pipeline: %v", err))
		return
	}
	if pipelineResp == nil || pipelineResp.PipelineID == "" {
		fail("publish pipeline returned no operation id")
		return
	}
	ownerEffectPossible = true

	o.log("info", map[string]interface{}{
		"msg":         "publish pipeline started",
		"pipeline_id": pipelineResp.PipelineID,
	})

	status, err := client.WaitForPipelineConnect(ctx, pipelineResp.PipelineID)
	if err != nil {
		fail(fmt.Sprintf("publish pipeline failed: %v", err))
		return
	}

	deployResult, err := ExtractDeployResult(status)
	if err != nil {
		fail(fmt.Sprintf("could not extract deploy result: %v", err))
		return
	}
	if deployResult == nil || len(deployResult.Artifacts) == 0 {
		fail("publish pipeline returned no published artifacts")
		return
	}
	if !exactArtifactTargetSet(req.Platforms, deployResult.Artifacts) {
		fail(fmt.Sprintf("published target set does not match requested target set (%s)", strings.Join(req.Platforms, ",")))
		return
	}
	if o.publishedVersionsRepo == nil {
		fail("published-version repository is not configured")
		return
	}

	provenance, err := ExtractProvenance(status)
	if err != nil {
		fail(fmt.Sprintf("published pipeline returned invalid provenance: %v", err))
		return
	}

	version := strings.TrimSpace(provenance.Version)
	gitHash := strings.TrimSpace(provenance.GitCommitHash)
	if strings.TrimSpace(req.ReleaseVersion) != "" && version != strings.TrimSpace(req.ReleaseVersion) {
		fail(fmt.Sprintf("published pipeline provenance version %q does not match requested release version %q", version, req.ReleaseVersion))
		return
	}
	if strings.TrimSpace(req.GitCommitHash) != "" && gitHash != strings.TrimSpace(req.GitCommitHash) {
		fail(fmt.Sprintf("published pipeline provenance commit %q does not match requested commit %q", gitHash, req.GitCommitHash))
		return
	}

	var published []PublishedVersion
	receipts, hasReceipts := o.releasesRepo.(releases.PublicationReceiptRepository)
	if req.ReleaseID != "" && !hasReceipts {
		fail("commercial publication requires a durable publication-receipt repository")
		return
	}
	hasReceipts = hasReceipts && req.ReleaseID != ""
	for _, artifact := range deployResult.Artifacts {
		if hasReceipts && (artifact.ArtifactID <= 0 || artifact.SHA512 == "" || strings.TrimSpace(artifact.DestinationObject) == "") {
			fail(fmt.Sprintf("published artifact %s has incomplete receipt identity", artifact.Platform))
			return
		}
		record := &PublishedVersion{
			ProfileID:     req.ProfileID,
			Platform:      artifact.Platform,
			Version:       version,
			SHA512:        artifact.SHA512,
			GitCommitHash: gitHash,
			ArtifactID:    artifact.ArtifactID,
			ReleaseID:     req.ReleaseID,
		}
		if hasReceipts {
			if err := receipts.RecordPublicationReceipt(ctx, req.ReleaseID, releases.PublicationReceipt{
				CandidateID: req.CandidateID, DestinationRevisionID: req.DestinationRevisionID,
				TargetID: artifact.Platform, ArtifactDigest: "sha512:" + artifact.SHA512,
				DestinationObject: artifact.DestinationObject,
				Producer:          "scenario-to-desktop/lpbs", ExternalReceipt: fmt.Sprintf("%s/%d", pipelineResp.PipelineID, artifact.ArtifactID),
				Outcome: releases.ReceiptPublished, ObservedAt: time.Now().UTC(),
			}); err != nil {
				fail(fmt.Sprintf("failed to record publication receipt for %s: %v", artifact.Platform, err))
				return
			}
		}
		if err := o.publishedVersionsRepo.RecordPublish(ctx, record); err != nil {
			fail(fmt.Sprintf("failed to record published version for %s: %v", artifact.Platform, err))
			return
		}
		published = append(published, *record)

		// Advance the per-platform release row so verify sees published state.
		if o.releasesRepo != nil && req.ReleaseID != "" {
			if err := o.releasesRepo.MarkPlatformPublished(ctx, req.ReleaseID, artifact.Platform, artifact.ArtifactID); err != nil {
				fail(fmt.Sprintf("failed to persist published platform %s: %v", artifact.Platform, err))
				return
			}
		}
	}

	if o.releasesRepo != nil && req.ReleaseID != "" && len(published) > 0 {
		if err := o.releasesRepo.UpdateStatus(ctx, req.ReleaseID, releases.StatusPublishing); err != nil {
			fail(fmt.Sprintf("failed to persist publishing status: %v", err))
			return
		}
	}
	if len(published) != len(deployResult.Artifacts) {
		fail(fmt.Sprintf("persisted %d of %d published artifact receipts", len(published), len(deployResult.Artifacts)))
		return
	}

	response.PublishedVersions = published
	o.successStep(step, fmt.Sprintf("published %d artifact(s), version %s", len(published), version))
}

func exactArtifactTargetSet(requested []string, artifacts []PipelineDeployArtifact) bool {
	if len(requested) == 0 || len(requested) != len(artifacts) {
		return false
	}
	want := make(map[string]struct{}, len(requested))
	for _, target := range requested {
		target = strings.TrimSpace(target)
		if target == "" {
			return false
		}
		if _, exists := want[target]; exists {
			return false
		}
		want[target] = struct{}{}
	}
	for _, artifact := range artifacts {
		if _, exists := want[artifact.Platform]; !exists {
			return false
		}
		delete(want, artifact.Platform)
	}
	return len(want) == 0
}

// applySigningConfig applies the provided signing configuration to scenario-to-desktop.
func (o *Orchestrator) applySigningConfig(ctx context.Context, scenarioName string, config map[string]interface{}) error {
	desktopClient, err := NewDesktopPackagerClient(o.log)
	if err != nil {
		return fmt.Errorf("scenario-to-desktop unavailable: %w", err)
	}
	return desktopClient.SetSigningConfig(ctx, scenarioName, config)
}

// checkSigningReadiness checks if signing is configured for the scenario.
func (o *Orchestrator) checkSigningReadiness(ctx context.Context, scenarioName string) []string {
	var warnings []string

	desktopClient, err := NewDesktopPackagerClient(o.log)
	if err != nil {
		o.log("error", map[string]interface{}{
			"msg":   "could not check signing readiness - scenario-to-desktop unavailable",
			"error": err.Error(),
		})
		return []string{fmt.Sprintf("signing readiness unavailable: %v", err)}
	}

	readiness, err := desktopClient.CheckSigningReadiness(ctx, scenarioName)
	if err != nil {
		o.log("error", map[string]interface{}{
			"msg":      "signing readiness check failed",
			"scenario": scenarioName,
			"error":    err.Error(),
		})
		return []string{fmt.Sprintf("signing readiness unavailable: %v", err)}
	}
	if readiness == nil {
		return []string{"signing readiness returned no result"}
	}

	if !readiness.Ready {
		if len(readiness.Issues) > 0 {
			warnings = append(warnings, readiness.Issues...)
		} else {
			warnings = append(warnings, "Code signing not configured - installers will be unsigned")
		}
		for platform, status := range readiness.Platforms {
			if !status.Ready && status.Reason != "" {
				warnings = append(warnings, fmt.Sprintf("%s: %s", platform, status.Reason))
			}
		}
	}

	return warnings
}

// populateAssetMetadata fills in missing SHA256 and size metadata for assets that exist on disk.
func populateAssetMetadata(manifest *bundles.Manifest, scenarioDir string) error {
	if manifest == nil {
		return fmt.Errorf("manifest is nil")
	}

	var firstErr error
	if err := expandUIAssets(manifest, scenarioDir); err != nil {
		firstErr = err
	}
	for si := range manifest.Services {
		svc := &manifest.Services[si]

		// Ensure API services allow loopback origin in the bundled desktop.
		if strings.Contains(strings.ToLower(svc.ID), "-api") {
			if svc.Env == nil {
				svc.Env = make(map[string]string)
			}
			if _, ok := svc.Env["CORS_ALLOWED_ORIGINS"]; !ok {
				svc.Env["CORS_ALLOWED_ORIGINS"] = "*"
			}
			if _, ok := svc.Env["UI_PORT"]; !ok {
				svc.Env["UI_PORT"] = "${ui.ui}"
			}
		}

		for ai := range svc.Assets {
			asset := &svc.Assets[ai]
			if asset == nil || asset.Path == "" {
				continue
			}
			if asset.SHA256 != "" && asset.SHA256 != "pending" {
				continue
			}

			assetPath := asset.Path
			if !filepath.IsAbs(assetPath) {
				assetPath = filepath.Join(scenarioDir, assetPath)
			}

			info, err := os.Stat(assetPath)
			if err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("stat asset %s: %w", assetPath, err)
				}
				continue
			}

			hash, err := hashFileSHA256(assetPath)
			if err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("hash asset %s: %w", assetPath, err)
				}
				continue
			}

			asset.SHA256 = hash
			asset.SizeBytes = info.Size()
		}
	}

	return firstErr
}

func hashFileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// expandUIAssets ensures ui-bundle services enumerate all built assets under ui/dist.
func expandUIAssets(manifest *bundles.Manifest, scenarioDir string) error {
	if manifest == nil {
		return nil
	}
	var firstErr error
	for si := range manifest.Services {
		svc := &manifest.Services[si]
		if !strings.EqualFold(svc.Type, "ui-bundle") {
			continue
		}

		uiRoot := filepath.Join(scenarioDir, "ui", "dist")
		entries, err := os.ReadDir(uiRoot)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("read ui dist: %w", err)
			}
			continue
		}

		var assets []bundles.Asset
		for _, entry := range entries {
			err := filepath.WalkDir(filepath.Join(uiRoot, entry.Name()), func(path string, d os.DirEntry, werr error) error {
				if werr != nil {
					return werr
				}
				if d.IsDir() {
					return nil
				}
				rel, relErr := filepath.Rel(scenarioDir, path)
				if relErr != nil {
					if firstErr == nil {
						firstErr = fmt.Errorf("relativize ui asset %s: %w", path, relErr)
					}
					return nil
				}
				info, statErr := os.Stat(path)
				if statErr != nil {
					if firstErr == nil {
						firstErr = fmt.Errorf("stat ui asset %s: %w", path, statErr)
					}
					return nil
				}
				hash, herr := hashFileSHA256(path)
				if herr != nil {
					if firstErr == nil {
						firstErr = herr
					}
					return nil
				}
				assets = append(assets, bundles.Asset{
					Path:      filepath.ToSlash(rel),
					SHA256:    hash,
					SizeBytes: info.Size(),
				})
				return nil
			})
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}

		if len(assets) > 0 {
			svc.Assets = assets
		}
	}
	return firstErr
}

func resolveBuildPlatforms(inputs []string) []string {
	defaultPlatforms := []string{"linux-x64", "linux-arm64", "darwin-x64", "darwin-arm64", "win-x64"}
	if len(inputs) == 0 {
		return defaultPlatforms
	}

	allowed := make(map[string]bool)
	for _, p := range build.SupportedPlatforms {
		allowed[p.Name] = true
	}

	var result []string
	add := func(name string) {
		if !allowed[name] {
			return
		}
		for _, existing := range result {
			if existing == name {
				return
			}
		}
		result = append(result, name)
	}

	for _, raw := range inputs {
		switch strings.ToLower(raw) {
		case "win", "windows", "win-x64":
			add("win-x64")
		case "mac":
			add("darwin-x64")
			add("darwin-arm64")
		case "darwin-x64":
			add("darwin-x64")
		case "darwin-arm64":
			add("darwin-arm64")
		case "linux":
			add("linux-x64")
			add("linux-arm64")
		case "linux-x64":
			add("linux-x64")
		case "linux-arm64":
			add("linux-arm64")
		default:
			add(strings.ToLower(raw))
		}
	}

	if len(result) == 0 {
		return defaultPlatforms
	}
	return result
}

func resolveInstallerTargets(inputs []string) []string {
	if len(inputs) == 0 {
		return []string{"win", "mac", "linux"}
	}
	targetSet := map[string]bool{}
	add := func(name string) {
		targetSet[name] = true
	}
	for _, raw := range inputs {
		switch strings.ToLower(raw) {
		case "win", "windows", "win-x64":
			add("win")
		case "mac", "darwin", "darwin-arm64", "darwin-x64":
			add("mac")
		case "linux", "linux-x64", "linux-arm64":
			add("linux")
		}
	}
	if len(targetSet) == 0 {
		return []string{"win", "mac", "linux"}
	}
	var out []string
	for _, name := range []string{"win", "mac", "linux"} {
		if targetSet[name] {
			out = append(out, name)
		}
	}
	return out
}

func copyBuiltBinariesToBundle(manifest *bundles.Manifest, manifestDir, bundleDir string, platforms []string) ([]string, error) {
	if manifest == nil {
		return nil, fmt.Errorf("manifest is nil")
	}
	platformSet := make(map[string]bool)
	for _, p := range platforms {
		platformSet[p] = true
	}

	if err := os.MkdirAll(filepath.Join(bundleDir, "bin"), 0o755); err != nil {
		return nil, err
	}

	var missing []string
	for _, svc := range manifest.Services {
		for platform, bin := range svc.Binaries {
			if len(platformSet) > 0 && !platformSet[platform] {
				continue
			}

			src := bin.Path
			if !filepath.IsAbs(src) {
				src = filepath.Join(manifestDir, filepath.FromSlash(bin.Path))
			}

			if _, err := os.Stat(src); err != nil {
				missing = append(missing, fmt.Sprintf("%s:%s", svc.ID, platform))
				continue
			}

			destDir := filepath.Join(bundleDir, "bin", platform)
			if err := os.MkdirAll(destDir, 0o755); err != nil {
				return missing, err
			}

			dest := filepath.Join(destDir, filepath.Base(src))
			if err := copyFilePreserveMode(src, dest); err != nil {
				return missing, err
			}
		}
	}

	return missing, nil
}

func copyFilePreserveMode(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	mode := os.FileMode(0o755)
	if info, err := os.Stat(src); err == nil {
		mode = info.Mode().Perm()
	}
	return os.WriteFile(dst, data, mode)
}

func (o *Orchestrator) buildInstallersWithPnpm(ctx context.Context, desktopPath string, platforms []string) (map[string]string, error) {
	if len(platforms) == 0 {
		platforms = []string{"win", "mac", "linux"}
	}

	packageManager := "pnpm"
	if _, err := exec.LookPath(packageManager); err != nil {
		o.log("warn", map[string]interface{}{
			"msg":   "pnpm not found, falling back to npm",
			"error": err.Error(),
		})
		packageManager = "npm"
	}

	if err := o.runCommandLogged(ctx, packageManager, []string{"install"}, desktopPath); err != nil {
		return nil, err
	}

	distDir := filepath.Join(desktopPath, "dist-electron")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		return nil, err
	}

	installers := make(map[string]string)
	var missing []string
	for _, platform := range platforms {
		cmd := []string{"run", fmt.Sprintf("dist:%s", platform)}
		if err := o.runCommandLogged(ctx, packageManager, cmd, desktopPath); err != nil {
			return installers, fmt.Errorf("%s build failed: %w", platform, err)
		}

		artifact, err := findInstallerArtifact(distDir, platform)
		if err != nil {
			o.log("error", map[string]interface{}{
				"msg":      "installer built but artifact not located",
				"platform": platform,
				"error":    err.Error(),
			})
			missing = append(missing, platform)
			continue
		}
		installers[platform] = artifact
	}
	if len(missing) > 0 {
		return installers, fmt.Errorf("installer artifacts missing for platform(s): %s", strings.Join(missing, ", "))
	}

	return installers, nil
}

type commandRunner interface {
	Run(context.Context, string, []string, string) ([]byte, error)
}

type processCommandRunner struct{}

func (processCommandRunner) Run(ctx context.Context, bin string, args []string, dir string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

func (o *Orchestrator) runCommandLogged(ctx context.Context, bin string, args []string, dir string) error {
	runner := o.commandRunner
	if runner == nil {
		runner = processCommandRunner{}
	}
	return runCommandLoggedWith(runner, ctx, bin, args, dir, o.log)
}

func runCommandLogged(ctx context.Context, bin string, args []string, dir string, log func(string, map[string]interface{})) error {
	return runCommandLoggedWith(processCommandRunner{}, ctx, bin, args, dir, log)
}

func runCommandLoggedWith(runner commandRunner, ctx context.Context, bin string, args []string, dir string, log func(string, map[string]interface{})) error {
	output, err := runner.Run(ctx, bin, args, dir)
	log("info", map[string]interface{}{
		"msg":  "command completed",
		"cmd":  fmt.Sprintf("%s %s", bin, strings.Join(args, " ")),
		"dir":  dir,
		"ok":   err == nil,
		"logs": string(output),
	})
	if err != nil {
		return fmt.Errorf("%s %s failed: %v", bin, strings.Join(args, " "), err)
	}
	return nil
}

func findInstallerArtifact(distDir, platform string) (string, error) {
	var patterns []string
	switch platform {
	case "win":
		patterns = []string{"*.exe", "*.msi"}
	case "mac":
		patterns = []string{"*.dmg", "*.pkg", "*.zip"}
	case "linux":
		patterns = []string{"*.AppImage", "*.deb", "*.tar.gz"}
	default:
		return "", fmt.Errorf("unknown platform %s", platform)
	}

	var candidates []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(distDir, pattern))
		if err == nil {
			candidates = append(candidates, matches...)
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no installer artifacts matched for %s", platform)
	}

	sort.Slice(candidates, func(i, j int) bool {
		infoI, _ := os.Stat(candidates[i])
		infoJ, _ := os.Stat(candidates[j])
		if infoI == nil || infoJ == nil {
			return candidates[i] < candidates[j]
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})

	return candidates[0], nil
}

// pruneNonCrossPlatformCLIs removes CLI services that are not cross-platform.
func pruneNonCrossPlatformCLIs(manifest *bundles.Manifest, scenarioDir string) ([]string, error) {
	if manifest == nil {
		return nil, nil
	}

	var kept []bundles.ServiceEntry
	var pruned []string

	for _, svc := range manifest.Services {
		if !isCLIService(svc) {
			kept = append(kept, svc)
			continue
		}
		if isCrossPlatformCLIBuild(svc, scenarioDir) {
			kept = append(kept, svc)
			continue
		}
		pruned = append(pruned, svc.ID)
	}

	manifest.Services = kept
	return pruned, nil
}

func isCLIService(svc bundles.ServiceEntry) bool {
	id := strings.ToLower(svc.ID)
	if strings.Contains(id, "cli") {
		return true
	}
	if svc.Build != nil {
		base := strings.ToLower(filepath.Base(svc.Build.SourceDir))
		if base == "cli" {
			return true
		}
	}
	return false
}

func isCrossPlatformCLIBuild(svc bundles.ServiceEntry, scenarioDir string) bool {
	if svc.Build == nil {
		return false
	}

	switch svc.Build.Type {
	case "go":
		sourceDir := filepath.Join(scenarioDir, svc.Build.SourceDir)
		matches, _ := filepath.Glob(filepath.Join(sourceDir, "*.go"))
		return len(matches) > 0
	case "rust":
		sourceDir := filepath.Join(scenarioDir, svc.Build.SourceDir)
		if _, err := os.Stat(filepath.Join(sourceDir, "Cargo.toml")); err == nil {
			return true
		}
		matches, _ := filepath.Glob(filepath.Join(sourceDir, "*.rs"))
		return len(matches) > 0
	default:
		return false
	}
}

// updateManifestBinaryPaths updates manifest service binaries to point to actual build outputs.
func updateManifestBinaryPaths(manifest *bundles.Manifest, results []build.BuildResult, scenarioDir, manifestDir string) {
	platformPaths := make(map[string]string)
	for _, r := range results {
		if r.Success && r.OutputPath != "" {
			relPath, err := filepath.Rel(manifestDir, r.OutputPath)
			if err != nil {
				relPath = r.OutputPath
				if strings.HasPrefix(r.OutputPath, scenarioDir) {
					relPath = strings.TrimPrefix(r.OutputPath, scenarioDir)
					relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
				}
			}
			relPath = filepath.ToSlash(relPath)
			platformPaths[r.Platform] = relPath
		}
	}

	for i := range manifest.Services {
		svc := &manifest.Services[i]
		if svc.Build == nil {
			continue
		}

		for platform := range svc.Binaries {
			if newPath, ok := platformPaths[platform]; ok {
				svc.Binaries[platform] = bundles.ServiceBinary{
					Path: newPath,
					Args: svc.Binaries[platform].Args,
					Env:  svc.Binaries[platform].Env,
					Cwd:  svc.Binaries[platform].Cwd,
				}
			}
		}

		if newPath, ok := platformPaths["darwin-arm64"]; ok {
			svc.Binaries["darwin-arm64"] = bundles.ServiceBinary{Path: newPath}
		}
		if newPath, ok := platformPaths["linux-arm64"]; ok {
			svc.Binaries["linux-arm64"] = bundles.ServiceBinary{Path: newPath}
		}

		svc.Build = nil
	}
}
