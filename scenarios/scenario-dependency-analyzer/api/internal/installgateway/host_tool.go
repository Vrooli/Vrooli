package installgateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/cli-core/upstreamcheck"
	"github.com/vrooli/envkit-go"
	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/resources/agentinstall"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/tools/akt"
	"github.com/vrooli/vrooli/internal/tools/hyperframes"
	"github.com/vrooli/vrooli/internal/tools/kubo"
	triposrsource "github.com/vrooli/vrooli/internal/tools/triposr-source"
)

// resolveHostTool is deliberately a closed owner surface, not an arbitrary path
// or another interpretation of tools/<shared-package>.
func resolveHostTool(root, name, ecosystem, pkg, version string) (Resolution, error) {
	switch name {
	case "codex", "claude-code", "opencode", "grok":
		return resolveAgentResource(root, name, ecosystem, pkg, version)
	}

	if name == kubo.ToolName && ecosystem == "source" && pkg == kubo.ToolName && version == kubo.Version {
		source := filepath.Join(root, "internal", "tools", kubo.ToolName)
		if err := kubo.ValidateSource(source); err != nil {
			return Resolution{}, err
		}
		return Resolution{RepositoryRoot: root, SurfaceRoot: source, ManifestPath: filepath.Join(source, "tool.json"), PackageManager: "native-binaryfetch", PackageName: pkg, ExactVersion: true, Argv: []string{"native-acquire", pkg, version}, Profile: InstallProfile{Governance: "scenario-dependency-analyzer", LifecycleMode: "pinned-release", FrozenLockfile: true, ScriptsDisabled: true}}, nil
	}
	if name == akt.ToolName && ecosystem == "source" && pkg == akt.ToolName && version == akt.Version {
		source := filepath.Join(root, "internal", "tools", akt.ToolName)
		if err := akt.ValidateSource(source); err != nil {
			return Resolution{}, err
		}
		return Resolution{RepositoryRoot: root, SurfaceRoot: source, ManifestPath: filepath.Join(source, "tool.json"), PackageManager: "native-binaryfetch", PackageName: pkg, ExactVersion: true, Argv: []string{"native-acquire", pkg, version}, Profile: InstallProfile{Governance: "scenario-dependency-analyzer", LifecycleMode: "pinned-release", FrozenLockfile: true, ScriptsDisabled: true}}, nil
	}
	if name == triposrsource.ToolName && ecosystem == "source" && pkg == triposrsource.ToolName && version == triposrsource.Commit {
		source := filepath.Join(root, "internal", "tools", triposrsource.ToolName)
		return Resolution{HostSource: true, RepositoryRoot: root, SurfaceRoot: source, ManifestPath: filepath.Join(source, "tool.json"), PackageManager: "native-binaryfetch", PackageName: pkg, ExactVersion: true, Argv: []string{"native-acquire", pkg, version}, Profile: InstallProfile{Governance: "scenario-dependency-analyzer", LifecycleMode: "pinned-source-tree", FrozenLockfile: true}}, nil
	}
	if name != "hyperframes" || ecosystem != "npm" || pkg != "hyperframes" || version != hyperframes.Version {
		return Resolution{}, fmt.Errorf("host-tool requires npm/hyperframes@%s and owner hyperframes", hyperframes.Version)
	}
	source := filepath.Join(root, "internal", "tools", "hyperframes")
	if err := hyperframes.ValidateSource(source); err != nil {
		return Resolution{}, err
	}
	argv := []string{"npm", "ci", "--ignore-scripts", "--no-audit", "--no-fund"}
	return Resolution{HostTool: true, RepositoryRoot: root, SurfaceRoot: source, PackageManager: "npm", ManifestPath: filepath.Join(source, "package.json"), PackageName: pkg, ExactVersion: true, Argv: argv, Profile: SafeProfileFor("npm", argv)}, nil
}

func provisionHostRelease(ctx context.Context, r Resolution) (string, error) {
	var name, version string
	var provision func(context.Context) (string, error)
	switch r.PackageName {
	case akt.ToolName:
		name, version, provision = akt.ToolName, akt.Version, akt.Provision
	case kubo.ToolName:
		name, version, provision = kubo.ToolName, kubo.Version, kubo.Provision
	default:
		return "", fmt.Errorf("pinned release resolution changed: unknown owner %q", r.PackageName)
	}
	want, err := resolveHostTool(r.RepositoryRoot, name, "source", name, version)
	if err != nil {
		return "", err
	}
	if r.SurfaceRoot != want.SurfaceRoot || r.ManifestPath != want.ManifestPath || r.PackageName != want.PackageName || r.PackageManager != want.PackageManager || !r.ExactVersion || r.Command() != want.Command() || r.Profile != want.Profile || r.HostTool || r.HostSource || r.HostToolRefresh {
		return "", fmt.Errorf("%s release resolution changed before acquisition", name)
	}
	path, err := provision(ctx)
	if err != nil {
		return "", err
	}
	return "Installed verified " + name + " " + version + " through native binaryfetch at " + path, nil
}

// ResolveHostToolRefresh is an explicit source-only operation. Frozen runtime
// reproduction remains closed to the currently committed version.
func ResolveHostToolRefresh(root, name, ecosystem, pkg, version string) (Resolution, error) {
	if name != "hyperframes" || ecosystem != "npm" || pkg != "hyperframes" || !exactNpmVersion.MatchString(version) {
		return Resolution{}, fmt.Errorf("host-tool source refresh requires npm/hyperframes and an exact semantic version")
	}
	source := filepath.Join(root, "internal", "tools", "hyperframes")
	if err := hyperframes.ValidateSource(source); err != nil {
		return Resolution{}, fmt.Errorf("host-tool source closure requires recovery/review before refresh: %w", err)
	}
	argv := []string{"npm", "install", "--package-lock-only", "--save-exact", "--ignore-scripts", "--no-audit", "--no-fund", "hyperframes@" + version}
	return Resolution{HostToolRefresh: true, HostToolTargetVersion: version, RepositoryRoot: root, SurfaceRoot: source, PackageManager: "npm", ManifestPath: filepath.Join(source, "package.json"), PackageName: pkg, ExactVersion: true, Argv: argv, Profile: SafeProfileFor("npm", argv)}, nil
}

func refreshHostToolSource(ctx context.Context, r Resolution) (string, error) {
	want, err := ResolveHostToolRefresh(r.RepositoryRoot, "hyperframes", "npm", "hyperframes", r.HostToolTargetVersion)
	if err != nil {
		return "", err
	}
	if r.SurfaceRoot != want.SurfaceRoot || r.Command() != want.Command() || !r.ExactVersion {
		return "", fmt.Errorf("host-tool source refresh resolution changed")
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	return hyperframes.RefreshSource(ctx, r.SurfaceRoot, r.HostToolTargetVersion, func(ctx context.Context, dir string) (string, error) { return hostToolInstall(ctx, dir, want.Argv) })
}

// hostToolInstall is an injected command seam; governance remains in the API
// caller and all package-manager execution stays inside this owner.
var hostToolInstall = func(ctx context.Context, dir string, argv []string) (string, error) {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Env = envkit.Toolchain(envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, nil), envkit.ToolchainOptions{})
	cmd.WaitDelay = 5 * time.Second
	var output hostToolOutput
	cmd.Stdout, cmd.Stderr = &output, &output
	err := cmd.Run()
	return output.String(), err
}

func provisionHostTool(ctx context.Context, r Resolution) (string, error) {
	// Re-resolve so a manually constructed Resolution cannot change argv or source.
	want, err := resolveHostTool(r.RepositoryRoot, "hyperframes", "npm", "hyperframes", hyperframes.Version)
	if err != nil {
		return "", err
	}
	if r.SurfaceRoot != want.SurfaceRoot || r.Command() != want.Command() {
		return "", fmt.Errorf("host tool provision resolution changed")
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	return hyperframes.Provision(ctx, r.SurfaceRoot, func(ctx context.Context, dir string) (string, error) { return hostToolInstall(ctx, dir, want.Argv) })
}

// The existing harness manifests own package identity; the registry separately
// admits an exact version. No caller supplies a path, prefix, command or scripts.
func resolveAgentResource(root, name, ecosystem, pkg, version string) (Resolution, error) {
	if ecosystem != "npm" || !exactNpmVersion.MatchString(version) {
		return Resolution{}, fmt.Errorf("harness install requires npm and an exact semantic version")
	}
	source := filepath.Join(root, "resources", name)
	manifestPath := filepath.Join(source, "resource.json")
	manifest, err := manifestpkg.Load(manifestPath)
	if err != nil {
		return Resolution{}, err
	}
	if manifest.Name != name || manifest.Acquisition == nil {
		return Resolution{}, fmt.Errorf("harness resource acquisition is not declared")
	}
	target, err := manifest.Acquisition.Resolve(binaryfetch.Facts{"os": runtime.GOOS, "arch": runtime.GOARCH})
	if err != nil {
		return Resolution{}, err
	}
	if target.Unsupported != "" || target.Package == "" || target.Package != pkg {
		return Resolution{}, fmt.Errorf("harness package differs from the resource acquisition contract")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return Resolution{}, err
	}
	prefix, err := agentinstall.NPMInstallPrefix(home, name, version)
	if err != nil {
		return Resolution{}, err
	}
	executable := target.BinPath
	if executable == "" {
		binary := name
		if name == "claude-code" {
			binary = "claude"
		}
		executable = "node_modules/.bin/" + binary
	}
	if filepath.IsAbs(executable) || filepath.ToSlash(filepath.Clean(executable)) != executable || !strings.HasPrefix(executable, "node_modules/") {
		return Resolution{}, fmt.Errorf("harness executable must be a clean path within node_modules")
	}
	argv := agentinstall.NPMInstallArgs(prefix, pkg+"@"+version)
	return Resolution{HostTool: true, HostToolTargetVersion: version, HostToolExecutable: executable, RepositoryRoot: root, SurfaceRoot: source, ManifestPath: manifestPath, PackageManager: "npm", PackageName: pkg, ExactVersion: true, Argv: argv, Profile: InstallProfile{Governance: "scenario-dependency-analyzer", LifecycleMode: "resource-agent", ScriptsDisabled: true}}, nil
}

func provisionAgentResource(ctx context.Context, r Resolution) (string, error) {
	name := filepath.Base(r.SurfaceRoot)
	want, err := resolveHostTool(r.RepositoryRoot, name, "npm", r.PackageName, r.HostToolTargetVersion)
	if err != nil {
		return "", err
	}
	if want.Profile.LifecycleMode != "resource-agent" || r.Command() != want.Command() || r.HostToolExecutable != want.HostToolExecutable || r.ManifestPath != want.ManifestPath || r.SurfaceRoot != want.SurfaceRoot || r.Profile != want.Profile || !r.HostTool || !r.ExactVersion || r.HostSource || r.HostToolRefresh {
		return "", fmt.Errorf("harness resolution changed before installation")
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	prefix, err := agentinstall.NPMInstallPrefix(home, name, r.HostToolTargetVersion)
	if err != nil {
		return "", err
	}
	// This lock serializes cooperating owners; it is not a filesystem CAS
	// against arbitrary external writers.
	release, err := artifactlock.AcquireContext(ctx, "resource-agent-"+name)
	if err != nil {
		return "", err
	}
	defer release()
	root := filepath.Dir(prefix)
	ledger, err := cliinstall.LoadInstallRecord(home)
	if err != nil {
		return "", err
	}
	owned := false
	for _, entry := range ledger.Entries {
		if entry.Scope == cliinstall.ScopeRuntime && entry.Kind == cliinstall.EntryDirectory && filepath.Clean(entry.Path) == root {
			owned = true
		}
	}
	if entries, err := os.ReadDir(root); err == nil && len(entries) > 0 && !owned {
		return "", fmt.Errorf("preserve unattributed harness artifact directory %s", root)
	} else if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if _, err := config.EnsureOwnedDir(root); err != nil {
		return "", err
	}
	if err := cliinstall.RecordToolArtifacts(home, cliinstall.InstallEntry{Scope: cliinstall.ScopeRuntime, Kind: cliinstall.EntryDirectory, Path: root, Prefix: root, Resource: name}); err != nil {
		return "", err
	}
	binary := name
	if name == "claude-code" {
		binary = "claude"
	}
	type installation struct{ Resource, Package, Version, TreeSHA256 string }
	verify := func(path string) error {
		b, err := os.ReadFile(filepath.Join(path, "installation.json"))
		if err != nil {
			return err
		}
		var record installation
		if err := json.Unmarshal(b, &record); err != nil {
			return err
		}
		if record.Resource != name || record.Package != r.PackageName || record.Version != r.HostToolTargetVersion {
			return fmt.Errorf("harness installation identity mismatch")
		}
		if err := binaryfetch.VerifyArtifactContext(ctx, filepath.Join(path, "node_modules"), "dir", record.TreeSHA256); err != nil {
			return err
		}
		return validateAgentBinary(path, r.HostToolExecutable, r.PackageName, r.HostToolTargetVersion)
	}
	if _, err := os.Lstat(prefix); err == nil {
		if err := verify(prefix); err != nil {
			return "", fmt.Errorf("preserve invalid existing harness candidate: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return "", err
	} else {
		candidate, err := os.MkdirTemp(root, ".candidate-")
		if err != nil {
			return "", err
		}
		defer os.RemoveAll(candidate)
		output, err := hostToolInstall(ctx, r.SurfaceRoot, agentinstall.NPMInstallArgs(candidate, r.PackageName+"@"+r.HostToolTargetVersion))
		if err != nil {
			return output, err
		}
		packageJSON, err := os.ReadFile(filepath.Join(candidate, "node_modules", filepath.FromSlash(r.PackageName), "package.json"))
		if err != nil {
			return output, err
		}
		var installed struct{ Name, Version string }
		if err := json.Unmarshal(packageJSON, &installed); err != nil {
			return output, err
		}
		if installed.Name != r.PackageName || installed.Version != r.HostToolTargetVersion {
			return output, fmt.Errorf("candidate package identity mismatch")
		}
		tree, err := binaryfetch.TreeDigestContext(ctx, filepath.Join(candidate, "node_modules"))
		if err != nil {
			return output, err
		}
		data, err := json.Marshal(installation{name, r.PackageName, r.HostToolTargetVersion, tree})
		if err != nil {
			return output, err
		}
		if err := os.WriteFile(filepath.Join(candidate, "installation.json"), data, 0600); err != nil {
			return output, err
		}
		latest, err := resolveHostTool(r.RepositoryRoot, name, "npm", r.PackageName, r.HostToolTargetVersion)
		if err != nil {
			return output, fmt.Errorf("harness source changed during preparation: %w", err)
		}
		if latest.Command() != want.Command() || latest.HostToolExecutable != want.HostToolExecutable {
			return output, fmt.Errorf("harness source changed during preparation")
		}
		if err := ctx.Err(); err != nil {
			return output, err
		}
		if err := config.CommitOwnedPathAtomic(candidate, prefix); err != nil {
			return output, err
		}
	}
	if err := verify(prefix); err != nil {
		return "", err
	}
	// Qualify at the final artifact path before changing the selected link:
	// relocation can break an otherwise byte-identical executable.
	if err := agentBinaryProbe(ctx, name, filepath.Join(prefix, r.HostToolExecutable), r.HostToolTargetVersion, root); err != nil {
		return "", fmt.Errorf("candidate unsupported without installer scripts; previous selection retained: %w", err)
	}
	if err := verify(prefix); err != nil {
		return "", err
	}
	latest, err := resolveHostTool(r.RepositoryRoot, name, "npm", r.PackageName, r.HostToolTargetVersion)
	if err != nil {
		return "", fmt.Errorf("harness source changed before selection: %w", err)
	}
	if latest.Command() != want.Command() || latest.HostToolExecutable != want.HostToolExecutable {
		return "", fmt.Errorf("harness source changed before selection")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	link := filepath.Join(home, ".local", "bin", binary)
	if st, err := os.Lstat(link); err == nil && st.Mode()&os.ModeSymlink == 0 {
		return "", fmt.Errorf("harness selection is an unowned regular file")
	}
	if err := cliinstall.RecordToolArtifacts(home, cliinstall.InstallEntry{Scope: cliinstall.ScopeRuntime, Kind: cliinstall.EntryFile, Path: link, Prefix: home, Resource: name}); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := agentinstall.LinkNPMBinary(filepath.Join(prefix, r.HostToolExecutable), binary, filepath.Dir(link)); err != nil {
		return "", err
	}
	return "Verified script-disabled candidate selected at " + prefix, nil
}

// The executable must resolve into the artifact actually verified above.
// In particular, a matching external --version is not candidate qualification.
func validateAgentBinary(prefix, executable, mainPackage, version string) error {
	root, err := filepath.EvalSymlinks(filepath.Join(prefix, "node_modules"))
	if err != nil {
		return err
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(prefix, executable))
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("harness executable escapes verified node_modules")
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("harness executable is not a regular file")
	}
	if strings.HasPrefix(executable, "node_modules/.bin/") {
		return nil
	}
	parts := strings.Split(strings.TrimPrefix(executable, "node_modules/"), "/")
	count := 1
	if strings.HasPrefix(parts[0], "@") {
		count = 2
	}
	if len(parts) <= count {
		return fmt.Errorf("harness native package path is incomplete")
	}
	nativePackage := strings.Join(parts[:count], "/")
	var main struct {
		Name, Version        string
		OptionalDependencies map[string]string
	}
	var native struct{ Name, Version string }
	for _, source := range []struct {
		path  string
		value any
	}{
		{filepath.Join(root, filepath.FromSlash(mainPackage), "package.json"), &main},
		{filepath.Join(root, filepath.FromSlash(nativePackage), "package.json"), &native},
	} {
		data, err := os.ReadFile(source.path)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, source.value); err != nil {
			return err
		}
	}
	if main.Name != mainPackage || main.Version != version || main.OptionalDependencies[nativePackage] != version || native.Name != nativePackage || native.Version != version {
		return fmt.Errorf("harness native package identity/version does not match exact main optional dependency")
	}
	packageRoot, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(nativePackage)))
	if err != nil {
		return err
	}
	rel, err = filepath.Rel(packageRoot, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("harness executable escapes declared native package")
	}
	return nil
}

var agentBinaryProbe = qualifyAgentBinary

func agentVersionProbeSpec(bwrap, binary, receiptDir string) platform.OperationSpec {
	args := []string{"--unshare-user", "--unshare-pid", "--unshare-net", "--die-with-parent", "--ro-bind", "/", "/", "--proc", "/proc", "--dev", "/dev", "--tmpfs", "/tmp", "--bind", receiptDir, "/tmp/probe-output", "--clearenv", "--setenv", "PATH", "/usr/local/bin:/usr/bin:/bin", "--setenv", "HOME", "/tmp", "--chdir", "/tmp", "--", "/bin/sh", "-c", `ulimit -f 128; exec "$@" > /tmp/probe-output/version.txt 2>&1`, "probe", binary, "--version"}
	return platform.OperationSpec{Path: bwrap, Args: args, Env: []string{}, Dir: "/", Limits: platform.OperationLimits{MemoryMaxBytes: 1 << 30, TasksMax: 64, CPUWeight: 20, WallTime: 20 * time.Second}}
}

// Version qualification isolates IP networking under the native operation owner.
// Host AF_UNIX sockets remain visible through the read-only host mount.
// The child gets a private HOME and bounded output file. This hides ordinary
// account discovery, not absolute host files or AF_UNIX access from hostile code.
func qualifyAgentBinary(ctx context.Context, name, binary, version, root string) (resultErr error) {
	bwrap, err := exec.LookPath("bwrap")
	if err != nil {
		return err
	}
	if _, err := os.Stat(binary); err != nil {
		return err
	}
	receiptDir, err := os.MkdirTemp(root, ".qualification-")
	if err != nil {
		return err
	}
	spec := agentVersionProbeSpec(bwrap, binary, receiptDir)
	intent, err := platform.NewOperationIntent("resource-agent", fmt.Sprintf("%s-%d", name, time.Now().UnixNano()), spec)
	if err != nil {
		return err
	}
	data, err := json.Marshal(intent)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(receiptDir, "intent.json"), data, 0600); err != nil {
		return err
	}
	operation, err := platform.StartOwnedOperation(ctx, intent, spec)
	if err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		stopped, err := operation.Stop(cleanup)
		if err == nil {
			if !stopped.NativeQuiescent {
				err = platform.ErrOperationNotQuiescent
			} else {
				err = platform.ReleaseOwnedOperation(cleanup, operation.Ref)
			}
		}
		resultErr = errors.Join(resultErr, err)
	}()
	if err := operation.Wait(ctx); err != nil {
		return err
	}
	observation, err := platform.InspectOwnedOperation(ctx, operation.Ref)
	if err != nil {
		return err
	}
	data, err = json.Marshal(observation)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(receiptDir, "result.json"), data, 0600); err != nil {
		return err
	}
	if !observation.ExitKnown || observation.Populated || observation.ExecMainCode != 1 || observation.ExecMainStatus != 0 || observation.Result != "success" {
		return fmt.Errorf("offline version command failed: %s/%d", observation.Result, observation.ExecMainStatus)
	}
	output, err := os.ReadFile(filepath.Join(receiptDir, "version.txt"))
	if err != nil {
		return err
	}
	if upstreamcheck.ExtractSemver(string(output)) != version {
		return fmt.Errorf("offline candidate version mismatch: %q", string(output))
	}
	return nil
}
