package installgateway

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveWorkspace extends the existing governed installer to task-owned source
// outside the maintained repository. It never invents a scenario or follows a
// package manager's parent workspace. Explicit pnpm evidence permits bootstrap
// without manufacturing a lockfile. Registry policy remains the caller's gate.
func ResolveWorkspace(repoRoot, workspace, ecosystem, packageName, version string) (Resolution, error) {
	if ecosystem != "npm" {
		return Resolution{}, fmt.Errorf("external workspaces currently support the npm ecosystem only")
	}
	if err := validatePackageSpec(packageName, version); err != nil {
		return Resolution{}, err
	}
	if !exactNpmVersion.MatchString(version) {
		return Resolution{}, fmt.Errorf("external workspace installs require a concrete semantic version")
	}
	if !filepath.IsAbs(workspace) {
		return Resolution{}, fmt.Errorf("workspace must be an absolute existing directory")
	}
	root, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		return Resolution{}, fmt.Errorf("resolve workspace: %w", err)
	}
	repo, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		return Resolution{}, fmt.Errorf("resolve maintained repository: %w", err)
	}
	if rel, err := filepath.Rel(repo, root); err != nil || rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return Resolution{}, fmt.Errorf("maintained repository paths must use their scenario, resource, or package surface")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return Resolution{}, fmt.Errorf("workspace is not a directory")
	}
	for _, name := range []string{"package.json", "pnpm-lock.yaml", "pnpm-workspace.yaml", ".npmrc"} {
		info, err := os.Lstat(filepath.Join(root, name))
		if os.IsNotExist(err) && name != "package.json" {
			continue
		}
		if err != nil || !info.Mode().IsRegular() {
			return Resolution{}, fmt.Errorf("workspace %s must be a regular file when present", name)
		}
	}
	manifestPath := filepath.Join(root, "package.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return Resolution{}, err
	}
	var manifest struct {
		PackageManager       string            `json:"packageManager"`
		Workspaces           json.RawMessage   `json:"workspaces"`
		PNPM                 json.RawMessage   `json:"pnpm"`
		Dependencies         map[string]string `json:"dependencies"`
		DevDependencies      map[string]string `json:"devDependencies"`
		PeerDependencies     map[string]string `json:"peerDependencies"`
		OptionalDependencies map[string]string `json:"optionalDependencies"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Resolution{}, fmt.Errorf("parse workspace manifest: %w", err)
	}
	manager, pin, _ := strings.Cut(manifest.PackageManager, "@")
	if manager != "pnpm" || !exactNpmVersion.MatchString(pin) {
		return Resolution{}, fmt.Errorf("workspace package.json must declare packageManager as pnpm@<exact-version>")
	}
	if len(manifest.Workspaces) != 0 || len(manifest.PNPM) != 0 {
		return Resolution{}, fmt.Errorf("external install targets must be standalone manifests without workspace membership or pnpm resolver overrides")
	}
	for _, group := range []map[string]string{manifest.Dependencies, manifest.DevDependencies, manifest.PeerDependencies, manifest.OptionalDependencies} {
		for name, pin := range group {
			if name == packageName {
				continue
			}
			if err := validatePackageSpec(name, pin); err != nil || !exactNpmVersion.MatchString(pin) {
				return Resolution{}, fmt.Errorf("external dependency %s must have an exact registry version", name)
			}
		}
	}
	for _, name := range []string{"pnpm-workspace.yaml", "package-lock.json", "yarn.lock", "bun.lock", "bun.lockb", ".npmrc"} {
		if _, err := os.Lstat(filepath.Join(root, name)); !os.IsNotExist(err) {
			return Resolution{}, fmt.Errorf("external standalone workspace cannot contain %s", name)
		}
	}
	dev, err := existingJSDevDependency(manifestPath, packageName)
	if err != nil {
		return Resolution{}, err
	}
	argv := []string{"pnpm", "add", "--ignore-scripts", "--ignore-pnpmfile", "--ignore-workspace", "--save-exact", packageName + "@" + version}
	if dev {
		argv = append(argv, "-D")
	}
	return Resolution{RepositoryRoot: repo, SurfaceRoot: root, PackageManager: "pnpm", ManifestPath: manifestPath,
		PackageName: packageName, Argv: argv, Profile: SafeProfileFor("pnpm", argv), ExternalWorkspace: true, ExactVersion: true}, nil
}
