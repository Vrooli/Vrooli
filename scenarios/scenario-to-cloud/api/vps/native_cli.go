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
		rel, err := release.LoadDir(filepath.Dir(local))
		if err != nil {
			return "", err
		}
		caps, err := e.reach.Negotiate(ctx, e.rt.Target)
		if err != nil && !reach.IsKind(err, reach.KindProtocolUnsupported) {
			return "", err
		}
		declared := rel.Manifest.NativeCLI.GOOS + "/" + rel.Manifest.NativeCLI.GOARCH
		if caps.Platform != "" && caps.Platform != declared {
			return "", apierrors.Newf(apierrors.CodeUnsupportedCapability, "release %s carries a %s control plane; the target is %s", rel.Digest, declared, caps.Platform).
				WithDetail("artifact", filepath.Base(rel.NativeCLIPath())).WithDetail("declared", declared).WithDetail("required", caps.Platform)
		}
		repoRoot, err := repocontract.FindRepoRootFromEnvOrCWD()
		if err != nil {
			return "", err
		}
		delivery.Files = append(delivery.Files,
			reach.ArtifactFile{Role: "release_manifest", LocalPath: rel.ManifestPath(), RemotePath: in["release_manifest"]},
			reach.ArtifactFile{Role: "native_cli", LocalPath: rel.NativeCLIPath(), RemotePath: in["native_cli"], Mode: 0o755, SHA256: rel.Manifest.NativeCLI.SHA256},
			reach.ArtifactFile{Role: "repo_contract", LocalPath: filepath.Join(repoRoot, ".vrooli", "repo-contract.json"), RemotePath: filepath.Join(e.rt.Target.Locator.Workdir, ".vrooli", "repo-contract.json")},
			reach.ArtifactFile{Role: "repo_service_manifest", LocalPath: filepath.Join(repoRoot, ".vrooli", "service.json"), RemotePath: filepath.Join(e.rt.Target.Locator.Workdir, ".vrooli", "service.json")},
			reach.ArtifactFile{Role: "repo_go_mod", LocalPath: filepath.Join(repoRoot, "go.mod"), RemotePath: filepath.Join(e.rt.Target.Locator.Workdir, "go.mod")},
		)
		autohealService := filepath.Join(repoRoot, "scenarios", "vrooli-autoheal", ".vrooli", "service.json")
		if fileExists(autohealService) {
			delivery.Files = append(delivery.Files, reach.ArtifactFile{
				Role: "autoheal_service_manifest", LocalPath: autohealService,
				RemotePath: filepath.Join(e.rt.Target.Locator.Workdir, "scenarios", "vrooli-autoheal", ".vrooli", "service.json"),
			})
		}
		for _, dir := range []string{"templates", "cmd", "internal"} {
			delivery.Files = append(delivery.Files, reach.ArtifactFile{
				Role:       "repo_marker_" + dir,
				LocalPath:  filepath.Join(repoRoot, "go.mod"),
				RemotePath: filepath.Join(e.rt.Target.Locator.Workdir, dir, ".cloud-bootstrap"),
			})
		}
		for _, resource := range e.manifest.Dependencies.Resources {
			resourceManifest := filepath.Join(repoRoot, "resources", resource, "resource.json")
			if _, statErr := os.Stat(resourceManifest); statErr != nil {
				continue
			}
			delivery.Files = append(delivery.Files, reach.ArtifactFile{
				Role:       "resource_manifest_" + resource,
				LocalPath:  resourceManifest,
				RemotePath: filepath.Join(e.rt.Target.Locator.Workdir, "resources", resource, "resource.json"),
			})
		}
		goWorkPath, err := appendBundleFile(&delivery, local, "go.work", filepath.Join(e.rt.Target.Locator.Workdir, "go.work"))
		if err != nil {
			return "", err
		}
		if goWorkPath != "" {
			defer os.Remove(goWorkPath)
			if err := appendWorkspaceModuleManifests(&delivery, goWorkPath, repoRoot, e.rt.Target.Locator.Workdir); err != nil {
				return "", err
			}
			if err := appendDirectoryFiles(&delivery, filepath.Join(repoRoot, "scenarios", "scenario-to-desktop", "runtime"), filepath.Join(e.rt.Target.Locator.Workdir, "scenarios", "scenario-to-desktop", "runtime"), "repo_watchdog_dependency"); err != nil {
				return "", err
			}
			if err := appendDirectoryFiles(&delivery, filepath.Join(repoRoot, "scenarios", "vrooli-autoheal", "cli"), filepath.Join(e.rt.Target.Locator.Workdir, "scenarios", "vrooli-autoheal", "cli"), "repo_autoheal_cli_source"); err != nil {
				return "", err
			}
			if err := appendDirectoryFiles(&delivery, filepath.Join(repoRoot, "packages", "cli-core"), filepath.Join(e.rt.Target.Locator.Workdir, "packages", "cli-core"), "repo_cli_core_source"); err != nil {
				return "", err
			}
			if err := appendDirectoryFiles(&delivery, filepath.Join(repoRoot, "packages", "api-core"), filepath.Join(e.rt.Target.Locator.Workdir, "packages", "api-core"), "repo_api_core_source"); err != nil {
				return "", err
			}
			watchdog, err := prepareWatchdogArtifact(repoRoot)
			if err != nil {
				return "", err
			}
			defer os.Remove(watchdog)
			delivery.Files = append(delivery.Files, reach.ArtifactFile{
				Role: "emergency_watchdog_release_build", LocalPath: watchdog,
				RemotePath: filepath.Join(remoteUserHome(e.rt.Target.Locator.User), ".vrooli", "libexec", "vrooli-watchdog"), Mode: 0o755,
			})
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
	addPackageFiles := func(moduleDir, pattern string) error {
		if _, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(moduleDir))); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		cmd := exec.Command("go", "list", "-deps", "-json", pattern)
		cmd.Dir = filepath.Join(repoRoot, filepath.FromSlash(moduleDir))
		out, err := cmd.Output()
		if err != nil {
			return err
		}
		dec := json.NewDecoder(strings.NewReader(string(out)))
		for dec.More() {
			var pkg struct {
				Dir        string   `json:"Dir"`
				GoFiles    []string `json:"GoFiles"`
				CgoFiles   []string `json:"CgoFiles"`
				SFiles     []string `json:"SFiles"`
				CFiles     []string `json:"CFiles"`
				HFiles     []string `json:"HFiles"`
				EmbedFiles []string `json:"EmbedFiles"`
			}
			err := dec.Decode(&pkg)
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			if !strings.HasPrefix(filepath.Clean(pkg.Dir)+string(os.PathSeparator), filepath.Clean(repoRoot)+string(os.PathSeparator)) {
				continue
			}
			for _, name := range append(append(append(append(append(append([]string{}, pkg.GoFiles...), pkg.CgoFiles...), pkg.SFiles...), pkg.CFiles...), pkg.HFiles...), pkg.EmbedFiles...) {
				if name == "" {
					continue
				}
				path := filepath.Join(pkg.Dir, name)
				if _, err := os.Stat(path); err == nil {
					rel, relErr := filepath.Rel(repoRoot, path)
					if relErr != nil {
						return relErr
					}
					paths[filepath.ToSlash(rel)] = struct{}{}
				}
			}
		}
		return nil
	}
	if err := addPackageFiles(".", "./cmd/vrooli-watchdog"); err != nil {
		return err
	}
	for rel := range paths {
		local := filepath.Join(repoRoot, filepath.FromSlash(rel))
		remote := filepath.Join(workdir, filepath.FromSlash(rel))
		delivery.Files = append(delivery.Files, reach.ArtifactFile{Role: "maintenance_source_" + strings.ReplaceAll(strings.ReplaceAll(rel, "/", "_"), ".", "_"), LocalPath: local, RemotePath: remote})
	}
	return nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
