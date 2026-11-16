package vps

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/release"
)

// The native control plane is never compiled at deploy time. The release
// build owns compilation (reproducible flags, recorded provenance) and binds
// the binary's sha256 and platform into the release manifest; delivery copies
// the release's bundle, manifest and binary through reach.Deliver, which
// proves the remote bytes match before the target owner verifies them again.

// runReleaseDeliver places the bundle and, for a built release, the release
// manifest and the native control plane for the negotiated platform.
func runReleaseDeliver(ctx context.Context, e *executor, action execplan.Action) (string, error) {
	in := action.Inputs
	local := in["artifact_path"]
	if local == "" {
		local = e.bundlePath
	}
	delivery := reach.Delivery{Files: []reach.ArtifactFile{{Role: "bundle", LocalPath: local, RemotePath: in["destination"], SHA256: in["bundle_sha256"]}}}
	if in["release_id"] != "" {
		if err := appendBuiltReleaseDelivery(ctx, e, in, local, &delivery); err != nil {
			return "", err
		}
	}
	receipt, err := e.reach.Deliver(ctx, e.rt.Target, delivery)
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(receipt.Files))
	for _, f := range receipt.Files {
		parts = append(parts, f.Role+"="+f.SHA256[:minInt(12, len(f.SHA256))])
	}
	return "delivered " + strings.Join(parts, ","), nil
}

func appendBuiltReleaseDelivery(ctx context.Context, e *executor, in map[string]string, bundlePath string, delivery *reach.Delivery) error {
	rel, err := release.LoadDir(filepath.Dir(bundlePath))
	if err != nil {
		return err
	}
	caps, err := e.reach.Negotiate(ctx, e.rt.Target)
	if err != nil && !reach.IsKind(err, reach.KindProtocolUnsupported) {
		return err
	}
	declared := rel.Manifest.NativeCLI.GOOS + "/" + rel.Manifest.NativeCLI.GOARCH
	if caps.Platform != "" && caps.Platform != declared {
		return apierrors.Newf(apierrors.CodeUnsupportedCapability, "release %s carries a %s control plane; the target is %s", rel.Digest, declared, caps.Platform).
			WithDetail("artifact", filepath.Base(rel.NativeCLIPath())).WithDetail("declared", declared).WithDetail("required", caps.Platform)
	}
	repoRoot, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return err
	}
	appendReleaseFiles(e, in, rel, repoRoot, delivery)
	if err := appendScenarioManifests(e, repoRoot, delivery); err != nil {
		return err
	}
	if err := appendWatchdogSource(e, repoRoot, delivery); err != nil {
		return err
	}
	if err := appendResourceFiles(e, repoRoot, delivery); err != nil {
		return err
	}
	return appendWorkspaceDelivery(e, bundlePath, repoRoot, delivery)
}

func appendReleaseFiles(e *executor, in map[string]string, rel release.Release, repoRoot string, delivery *reach.Delivery) {
	workdir := e.rt.Target.Locator.Workdir
	delivery.Files = append(delivery.Files,
		reach.ArtifactFile{Role: "release_manifest", LocalPath: rel.ManifestPath(), RemotePath: in["release_manifest"]},
		reach.ArtifactFile{Role: "native_cli", LocalPath: rel.NativeCLIPath(), RemotePath: in["native_cli"], Mode: 0o755, SHA256: rel.Manifest.NativeCLI.SHA256},
		reach.ArtifactFile{Role: "native_cli_managed_launcher", LocalPath: rel.NativeCLIPath(), RemotePath: filepath.Join(remoteUserHome(e.rt.Target.Locator.User), ".vrooli", "bin", "vrooli"), Mode: 0o755, SHA256: rel.Manifest.NativeCLI.SHA256},
		reach.ArtifactFile{Role: "repo_contract", LocalPath: filepath.Join(repoRoot, ".vrooli", "repo-contract.json"), RemotePath: filepath.Join(workdir, ".vrooli", "repo-contract.json")},
		reach.ArtifactFile{Role: "repo_service_manifest", LocalPath: filepath.Join(repoRoot, ".vrooli", "service.json"), RemotePath: filepath.Join(workdir, ".vrooli", "service.json")},
		reach.ArtifactFile{Role: "repo_go_mod", LocalPath: filepath.Join(repoRoot, "go.mod"), RemotePath: filepath.Join(workdir, "go.mod")},
		reach.ArtifactFile{Role: "repo_go_sum", LocalPath: filepath.Join(repoRoot, "go.sum"), RemotePath: filepath.Join(workdir, "go.sum")},
	)
}

func appendScenarioManifests(e *executor, repoRoot string, delivery *reach.Delivery) error {
	workdir := e.rt.Target.Locator.Workdir
	autohealService := filepath.Join(repoRoot, "scenarios", "vrooli-autoheal", ".vrooli", "service.json")
	if fileExists(autohealService) {
		delivery.Files = append(delivery.Files, reach.ArtifactFile{Role: "autoheal_service_manifest", LocalPath: autohealService, RemotePath: filepath.Join(workdir, "scenarios", "vrooli-autoheal", ".vrooli", "service.json")})
	}
	for _, scenarioID := range append([]string{e.manifest.Scenario.ID}, e.manifest.Dependencies.Scenarios...) {
		scenarioID = strings.TrimSpace(scenarioID)
		if scenarioID == "" || scenarioID == "vrooli-autoheal" {
			continue
		}
		serviceManifest := filepath.Join(repoRoot, "scenarios", scenarioID, ".vrooli", "service.json")
		if !fileExists(serviceManifest) {
			continue
		}
		delivery.Files = append(delivery.Files, reach.ArtifactFile{Role: "scenario_service_manifest_" + scenarioID, LocalPath: serviceManifest, RemotePath: filepath.Join(workdir, "scenarios", scenarioID, ".vrooli", "service.json")})
		for _, name := range []string{"go.mod", "go.sum"} {
			local := filepath.Join(repoRoot, "scenarios", scenarioID, "api", name)
			if !fileExists(local) {
				continue
			}
			delivery.Files = append(delivery.Files, reach.ArtifactFile{Role: "scenario_api_" + scenarioID + "_" + strings.ReplaceAll(name, ".", "_"), LocalPath: local, RemotePath: filepath.Join(workdir, "scenarios", scenarioID, "api", name)})
		}
	}
	return nil
}

func appendWatchdogSource(e *executor, repoRoot string, delivery *reach.Delivery) error {
	workdir := e.rt.Target.Locator.Workdir
	for _, dir := range []string{"templates", "cmd", "internal"} {
		delivery.Files = append(delivery.Files, reach.ArtifactFile{Role: "repo_marker_" + dir, LocalPath: filepath.Join(repoRoot, "go.mod"), RemotePath: filepath.Join(workdir, dir, ".cloud-bootstrap")})
	}
	for _, dir := range []string{"cmd/vrooli-watchdog", "internal/buildinfo", "internal/hostpressure", "internal/hostinventory", "internal/setpoint", "internal/workloadowner", "internal/tuning", "internal/logx", "packages/platform-go"} {
		if err := appendDirectoryFiles(delivery, filepath.Join(repoRoot, dir), filepath.Join(workdir, dir), "repo_watchdog_source"); err != nil {
			return err
		}
	}
	return nil
}

func appendResourceFiles(e *executor, repoRoot string, delivery *reach.Delivery) error {
	workdir := e.rt.Target.Locator.Workdir
	for _, resource := range e.manifest.Dependencies.Resources {
		resourceRoot := filepath.Join(repoRoot, "resources", resource)
		resourceManifest := filepath.Join(resourceRoot, "resource.json")
		if _, err := os.Stat(resourceManifest); err != nil {
			continue
		}
		delivery.Files = append(delivery.Files, reach.ArtifactFile{Role: "resource_manifest_" + resource, LocalPath: resourceManifest, RemotePath: filepath.Join(workdir, "resources", resource, "resource.json")})
		if err := appendDirectoryFiles(delivery, resourceRoot, filepath.Join(workdir, "resources", resource), "resource_source"); err != nil {
			return err
		}
	}
	return nil
}

func appendWorkspaceDelivery(e *executor, bundlePath, repoRoot string, delivery *reach.Delivery) error {
	workdir := e.rt.Target.Locator.Workdir
	goWorkPath, err := appendBundleFile(delivery, bundlePath, "go.work", filepath.Join(workdir, "go.work"))
	if err != nil || goWorkPath == "" {
		return err
	}
	defer os.Remove(goWorkPath)
	goWorkSumPath, err := appendBundleFile(delivery, bundlePath, "go.work.sum", filepath.Join(workdir, "go.work.sum"))
	if err != nil {
		return err
	}
	if goWorkSumPath != "" {
		defer os.Remove(goWorkSumPath)
	}
	if err := appendWorkspaceModuleManifests(delivery, goWorkPath, repoRoot, workdir); err != nil {
		return err
	}
	if err := appendWorkspaceSources(delivery, repoRoot, workdir); err != nil {
		return err
	}
	watchdog, err := prepareWatchdogArtifact(repoRoot)
	if err != nil {
		return err
	}
	defer os.Remove(watchdog)
	delivery.Files = append(delivery.Files, reach.ArtifactFile{Role: "emergency_watchdog_release_build", LocalPath: watchdog, RemotePath: filepath.Join(remoteUserHome(e.rt.Target.Locator.User), ".vrooli", "libexec", "vrooli-watchdog"), Mode: 0o755})
	return nil
}

func appendWorkspaceSources(delivery *reach.Delivery, repoRoot, workdir string) error {
	for _, source := range []struct{ local, remote, role string }{
		{"scenarios/scenario-to-desktop/runtime", "scenarios/scenario-to-desktop/runtime", "repo_watchdog_dependency"},
		{"scenarios/vrooli-autoheal/cli", "scenarios/vrooli-autoheal/cli", "repo_autoheal_cli_source"},
		{"scenarios/vrooli-autoheal/api", "scenarios/vrooli-autoheal/api", "repo_autoheal_api_source"},
		{"packages/cli-core", "packages/cli-core", "repo_cli_core_source"},
		{"packages/api-core", "packages/api-core", "repo_api_core_source"},
	} {
		if err := appendDirectoryFiles(delivery, filepath.Join(repoRoot, source.local), filepath.Join(workdir, source.remote), source.role); err != nil {
			return err
		}
	}
	for _, module := range []string{"envkit-go", "repo-contract-go"} {
		if err := appendDirectoryFiles(delivery, filepath.Join(repoRoot, "packages", module), filepath.Join(workdir, "packages", module), "repo_shared_module_source"); err != nil {
			return err
		}
	}
	return nil
}

// prepareWatchdogArtifact builds the small host safeguard on the control
// plane. The target can then verify and reuse it without compiling the root
// workspace under VPS memory constraints.
func prepareWatchdogArtifact(repoRoot string) (string, error) {
	// Go ignores dot-prefixed files when resolving a package, so keep this
	// temporary source name visible to `go run` while it exists.
	versionSource, err := os.CreateTemp(repoRoot, "watchdog-version-*.go")
	if err != nil {
		return "", err
	}
	versionPath := versionSource.Name()
	defer os.Remove(versionPath)
	if _, err := io.WriteString(versionSource, `package main
import (
  "fmt"
  "github.com/vrooli/vrooli/internal/buildinfo"
)
func main() {
  v, err := buildinfo.ComputeSourceFingerprintForPaths(".", "cmd/vrooli-watchdog", "internal/hostpressure", "internal/setpoint", "internal/workloadowner", "internal/hostinventory", "packages/platform-go")
  if err != nil { panic(err) }
  fmt.Print("managed:" + v)
}`); err != nil {
		versionSource.Close()
		return "", err
	}
	if err := versionSource.Close(); err != nil {
		return "", err
	}
	versionCmd := exec.Command("go", "run", versionPath)
	versionCmd.Dir = repoRoot
	version, err := versionCmd.Output()
	if err != nil {
		return "", fmt.Errorf("compute watchdog version: %w", err)
	}
	artifact, err := os.CreateTemp("", "vrooli-watchdog-release-")
	if err != nil {
		return "", err
	}
	artifactPath := artifact.Name()
	if err := artifact.Close(); err != nil {
		os.Remove(artifactPath)
		return "", err
	}
	ldflags := "-X main.buildVersion=" + strings.TrimSpace(string(version))
	cmd := exec.Command("go", "-C", repoRoot, "build", "-ldflags", ldflags, "-o", artifactPath, "./cmd/vrooli-watchdog")
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64")
	if output, err := cmd.CombinedOutput(); err != nil {
		os.Remove(artifactPath)
		return "", fmt.Errorf("build release watchdog: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return artifactPath, nil
}

func appendDirectoryFiles(delivery *reach.Delivery, localRoot, remoteRoot, rolePrefix string) error {
	if info, err := os.Stat(localRoot); err != nil || !info.IsDir() {
		return nil
	}
	return filepath.Walk(localRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || strings.HasSuffix(info.Name(), "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(localRoot, path)
		if err != nil {
			return err
		}
		delivery.Files = append(delivery.Files, reach.ArtifactFile{
			Role:      rolePrefix + "_" + strings.ReplaceAll(strings.ReplaceAll(filepath.ToSlash(rel), "/", "_"), ".", "_"),
			LocalPath: path, RemotePath: filepath.Join(remoteRoot, rel),
		})
		return nil
	})
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func appendBundleFile(delivery *reach.Delivery, archivePath, name, remotePath string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		// Unit fixtures and older callers may use a placeholder bundle. The
		// target's release verification remains the authority for archive
		// validity; optional workspace extraction must not mask that check.
		return "", nil
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return "", fmt.Errorf("release archive does not contain %s", name)
		}
		if err != nil {
			return "", fmt.Errorf("read release archive: %w", err)
		}
		if filepath.ToSlash(header.Name) != filepath.ToSlash(name) {
			continue
		}
		tmp, err := os.CreateTemp("", "scenario-to-cloud-artifact-")
		if err != nil {
			return "", err
		}
		tmpPath := tmp.Name()
		if _, err := io.Copy(tmp, tr); err != nil {
			tmp.Close()
			os.Remove(tmpPath)
			return "", err
		}
		if err := tmp.Close(); err != nil {
			os.Remove(tmpPath)
			return "", err
		}
		delivery.Files = append(delivery.Files, reach.ArtifactFile{Role: "release_" + strings.ReplaceAll(name, "/", "_"), LocalPath: tmpPath, RemotePath: remotePath})
		return tmpPath, nil
	}
}

func appendWorkspaceModuleManifests(delivery *reach.Delivery, goWorkPath, repoRoot, workdir string) error {
	contents, err := os.ReadFile(goWorkPath)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(contents), "\n") {
		module := strings.TrimSpace(line)
		module = strings.TrimPrefix(module, "./")
		if module == "" || module == "use (" || module == ")" || strings.HasPrefix(module, "go ") || strings.HasPrefix(module, "//") {
			continue
		}
		local := filepath.Join(repoRoot, filepath.FromSlash(module), "go.mod")
		if !fileExists(local) {
			continue
		}
		delivery.Files = append(delivery.Files, reach.ArtifactFile{
			Role:      "repo_module_manifest_" + strings.ReplaceAll(module, "/", "_"),
			LocalPath: local, RemotePath: filepath.Join(workdir, filepath.FromSlash(module), "go.mod"),
		})
	}
	return nil
}

func remoteUserHome(user string) string {
	user = strings.TrimSpace(user)
	if user == "" || user == "root" {
		return "/root"
	}
	return filepath.Join("/home", user)
}

// appendMaintenanceSourceFiles delivers the local Go source closure needed by
// target-side setup to rebuild the emergency watchdog. The autoheal source is
// already part of the staged application bundle; this small, derived source
// closure keeps the mutable workdir able to repair its host safeguards after
// a deployment from a clean or partial VPS checkout.
func appendMaintenanceSourceFiles(delivery *reach.Delivery, repoRoot, workdir string) error {
	if delivery == nil {
		return nil
	}
	paths := map[string]struct{}{}
	files, err := maintenancePackageSourcePaths(repoRoot, ".", "./cmd/vrooli-watchdog")
	if err != nil {
		return err
	}
	for _, file := range files {
		paths[file] = struct{}{}
	}
	for rel := range paths {
		local := filepath.Join(repoRoot, filepath.FromSlash(rel))
		remote := filepath.Join(workdir, filepath.FromSlash(rel))
		delivery.Files = append(delivery.Files, reach.ArtifactFile{Role: "maintenance_source_" + strings.ReplaceAll(strings.ReplaceAll(rel, "/", "_"), ".", "_"), LocalPath: local, RemotePath: remote})
	}
	return nil
}

type maintenancePackage struct {
	Dir        string   `json:"Dir"`
	GoFiles    []string `json:"GoFiles"`
	CgoFiles   []string `json:"CgoFiles"`
	SFiles     []string `json:"SFiles"`
	CFiles     []string `json:"CFiles"`
	HFiles     []string `json:"HFiles"`
	EmbedFiles []string `json:"EmbedFiles"`
}

func maintenancePackageSourcePaths(repoRoot, moduleDir, pattern string) ([]string, error) {
	localDir := filepath.Join(repoRoot, filepath.FromSlash(moduleDir))
	if _, err := os.Stat(localDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	cmd := exec.Command("go", "list", "-deps", "-json", pattern)
	cmd.Dir = localDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return decodeMaintenanceSourcePaths(out, repoRoot)
}

func decodeMaintenanceSourcePaths(encoded []byte, repoRoot string) ([]string, error) {
	decoder := json.NewDecoder(strings.NewReader(string(encoded)))
	var paths []string
	for decoder.More() {
		var pkg maintenancePackage
		if err := decoder.Decode(&pkg); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if !maintenancePackageWithinRepo(pkg.Dir, repoRoot) {
			continue
		}
		files, err := existingMaintenanceFiles(pkg, repoRoot)
		if err != nil {
			return nil, err
		}
		paths = append(paths, files...)
	}
	return paths, nil
}

func maintenancePackageWithinRepo(packageDir, repoRoot string) bool {
	return strings.HasPrefix(filepath.Clean(packageDir)+string(os.PathSeparator), filepath.Clean(repoRoot)+string(os.PathSeparator))
}

func existingMaintenanceFiles(pkg maintenancePackage, repoRoot string) ([]string, error) {
	names := append(append(append(append(append(append([]string{}, pkg.GoFiles...), pkg.CgoFiles...), pkg.SFiles...), pkg.CFiles...), pkg.HFiles...), pkg.EmbedFiles...)
	var paths []string
	for _, name := range names {
		if name == "" {
			continue
		}
		path := filepath.Join(pkg.Dir, name)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return nil, err
		}
		paths = append(paths, filepath.ToSlash(rel))
	}
	return paths, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
