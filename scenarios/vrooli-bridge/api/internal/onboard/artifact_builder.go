package onboard

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// controlPlaneArtifactBuilder builds all node executables from one repository
// root. The Vrooli binary is delegated to cmd/vrooli-dist, the shared primitive
// also used by release packaging; bridge-specific modules are cross-built with
// the same target environment.
type controlPlaneArtifactBuilder struct {
	lookPath  func(string) (string, error)
	run       func(context.Context, string, []string, string, []string) error
	cacheRoot string
	mu        sync.Mutex
}

// NewArtifactBuilder constructs the production control-plane cross-builder.
// The optional stateDir argument is the Bridge-owned state directory. Keeping
// the argument optional preserves the small constructor used by isolated
// tests, while production keeps cached executables beside the other Bridge
// state instead of in an operator-wide cache shared by unrelated services.
func NewArtifactBuilder(stateDirs ...string) ArtifactBuilder {
	root := ""
	if len(stateDirs) > 0 {
		root = strings.TrimSpace(stateDirs[0])
	}
	if root == "" {
		var err error
		root, err = os.UserCacheDir()
		if err != nil || strings.TrimSpace(root) == "" {
			root = os.TempDir()
		}
	}
	return &controlPlaneArtifactBuilder{
		lookPath:  exec.LookPath,
		run:       runArtifactCommand,
		cacheRoot: filepath.Join(root, "artifacts"),
	}
}

var _ ArtifactBuilder = (*controlPlaneArtifactBuilder)(nil)

func (b *controlPlaneArtifactBuilder) Build(ctx context.Context, p ArtifactBuildParams) (PrebuiltArtifacts, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	root := strings.TrimSpace(p.RepoDir)
	if root == "" {
		return PrebuiltArtifacts{}, fmt.Errorf("artifact build repository root is required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return PrebuiltArtifacts{}, fmt.Errorf("resolve artifact build root: %w", err)
	}
	if !supportedBridgeTarget(p.Target) {
		return PrebuiltArtifacts{}, fmt.Errorf("unsupported bridge node target %s/%s", p.Target.OS, p.Target.Arch)
	}
	if _, err := b.lookPath("go"); err != nil {
		return PrebuiltArtifacts{}, fmt.Errorf("control plane cannot cross-build node artifacts: Go is not installed or not on PATH")
	}
	if strings.TrimSpace(p.CacheKey) != "" && b.cacheRoot != "" {
		if cached, ok := b.loadCache(p.CacheKey, p.Target); ok {
			return cached, nil
		}
	}

	dir, err := os.MkdirTemp("", "vrooli-bridge-node-artifacts-*")
	if err != nil {
		return PrebuiltArtifacts{}, fmt.Errorf("create artifact staging directory: %w", err)
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.RemoveAll(dir)
		}
	}()

	ext := ""
	if p.Target.OS == "windows" {
		ext = ".exe"
	}
	vrooliPath := filepath.Join(dir, "vrooli"+ext)
	bridgePath := filepath.Join(dir, "vrooli-bridge"+ext)
	agentPath := filepath.Join(dir, "vrooli-bridge-agent"+ext)
	env := crossBuildEnv(p.Target)

	vrooliArgs := []string{
		"run", "./cmd/vrooli-dist", "--root", root,
		"--goos", p.Target.OS, "--goarch", p.Target.Arch, "--output", vrooliPath,
	}
	if p.Target.OS == "darwin" {
		// A Linux control plane cannot link the macOS Security framework. The
		// resulting Darwin binary is explicitly bootstrap-only: the remote
		// bootstrap applies host requirements, then rebuilds the final Vrooli
		// CLI natively with CGO and the macOS SDK before installing the agent.
		vrooliArgs = append(vrooliArgs, "--allow-missing-darwin-keychain")
	}
	buildCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	type buildJob struct {
		dir   string
		args  []string
		name  string
		env   []string
		label string
	}
	jobs := []buildJob{
		{dir: root, args: vrooliArgs, name: "go", label: "vrooli with shared distribution primitive"},
		{dir: filepath.Join(root, "scenarios", "vrooli-bridge", "cli"), args: []string{"build", "-trimpath", "-o", bridgePath, "."}, name: "go", env: env, label: "vrooli-bridge CLI"},
		{dir: filepath.Join(root, "scenarios", "vrooli-bridge", "agent"), args: []string{"build", "-trimpath", "-o", agentPath, "."}, name: "go", env: env, label: "node agent"},
	}
	errs := make(chan error, len(jobs))
	var wg sync.WaitGroup
	for _, job := range jobs {
		job := job
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := b.run(buildCtx, job.dir, job.args, job.name, job.env); err != nil {
				errs <- fmt.Errorf("cross-build %s: %w", job.label, err)
				cancel()
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		return PrebuiltArtifacts{}, err
	}

	vrooliSidecar := vrooliPath + ".fp"
	fingerprintBytes, err := os.ReadFile(vrooliSidecar)
	if err != nil {
		return PrebuiltArtifacts{}, fmt.Errorf("read shared Vrooli freshness sidecar: %w", err)
	}
	fingerprint := strings.TrimSpace(string(fingerprintBytes))
	if fingerprint == "" {
		return PrebuiltArtifacts{}, fmt.Errorf("shared Vrooli freshness sidecar is empty")
	}
	bridgeSidecar := bridgePath + ".fp"
	agentSidecar := agentPath + ".fp"
	for _, sidecar := range []string{bridgeSidecar, agentSidecar} {
		if err := os.WriteFile(sidecar, []byte(fingerprint+"\n"), 0o644); err != nil {
			return PrebuiltArtifacts{}, fmt.Errorf("write artifact sidecar %s: %w", filepath.Base(sidecar), err)
		}
	}

	keep = true
	result := PrebuiltArtifacts{
		Directory: dir, Vrooli: vrooliPath, VrooliSidecar: vrooliSidecar,
		BridgeCLI: bridgePath, BridgeSidecar: bridgeSidecar,
		Agent: agentPath, AgentSidecar: agentSidecar,
		Fingerprint: fingerprint, Target: p.Target,
		VrooliBootstrapOnly: p.Target.OS == "darwin",
	}
	if strings.TrimSpace(p.CacheKey) != "" && b.cacheRoot != "" {
		b.saveCache(p.CacheKey, p.Target, result)
	}
	return result, nil
}

func (b *controlPlaneArtifactBuilder) cacheDir(key string, target NodePlatform) string {
	clean := strings.NewReplacer("/", "_", "\\", "_", " ", "_").Replace(strings.TrimSpace(key))
	return filepath.Join(b.cacheRoot, clean+"-"+target.OS+"-"+target.Arch)
}

func (b *controlPlaneArtifactBuilder) loadCache(key string, target NodePlatform) (PrebuiltArtifacts, bool) {
	dir := b.cacheDir(key, target)
	ext := ""
	if target.OS == "windows" {
		ext = ".exe"
	}
	paths := []string{"vrooli" + ext, "vrooli" + ext + ".fp", "vrooli-bridge" + ext, "vrooli-bridge" + ext + ".fp", "vrooli-bridge-agent" + ext, "vrooli-bridge-agent" + ext + ".fp"}
	for _, name := range paths {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			return PrebuiltArtifacts{}, false
		}
	}
	tmp, err := os.MkdirTemp("", "vrooli-bridge-cached-artifacts-*")
	if err != nil {
		return PrebuiltArtifacts{}, false
	}
	for _, name := range paths {
		if err := copyArtifact(filepath.Join(dir, name), filepath.Join(tmp, name)); err != nil {
			_ = os.RemoveAll(tmp)
			return PrebuiltArtifacts{}, false
		}
	}
	fingerprint, err := os.ReadFile(filepath.Join(tmp, "vrooli"+ext+".fp"))
	if err != nil || strings.TrimSpace(string(fingerprint)) == "" {
		_ = os.RemoveAll(tmp)
		return PrebuiltArtifacts{}, false
	}
	return PrebuiltArtifacts{Directory: tmp, Vrooli: filepath.Join(tmp, "vrooli"+ext), VrooliSidecar: filepath.Join(tmp, "vrooli"+ext+".fp"), BridgeCLI: filepath.Join(tmp, "vrooli-bridge"+ext), BridgeSidecar: filepath.Join(tmp, "vrooli-bridge"+ext+".fp"), Agent: filepath.Join(tmp, "vrooli-bridge-agent"+ext), AgentSidecar: filepath.Join(tmp, "vrooli-bridge-agent"+ext+".fp"), Fingerprint: strings.TrimSpace(string(fingerprint)), Target: target, VrooliBootstrapOnly: target.OS == "darwin"}, true
}

func (b *controlPlaneArtifactBuilder) saveCache(key string, target NodePlatform, artifacts PrebuiltArtifacts) {
	if err := os.MkdirAll(b.cacheRoot, 0o700); err != nil {
		return
	}
	stage, err := os.MkdirTemp(b.cacheRoot, ".artifacts-")
	if err != nil {
		return
	}
	defer os.RemoveAll(stage)
	for _, path := range []string{artifacts.Vrooli, artifacts.VrooliSidecar, artifacts.BridgeCLI, artifacts.BridgeSidecar, artifacts.Agent, artifacts.AgentSidecar} {
		if err := copyArtifact(path, filepath.Join(stage, filepath.Base(path))); err != nil {
			return
		}
	}
	dest := b.cacheDir(key, target)
	if err := os.RemoveAll(dest); err != nil {
		return
	}
	_ = os.Rename(stage, dest)
}

func copyArtifact(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func supportedBridgeTarget(target NodePlatform) bool {
	if target.OS != "linux" && target.OS != "darwin" && target.OS != "windows" {
		return false
	}
	return target.Arch == "amd64" || target.Arch == "arm64"
}

func crossBuildEnv(target NodePlatform) []string {
	env := append([]string(nil), os.Environ()...)
	env = setArtifactEnv(env, "CGO_ENABLED", "0")
	env = setArtifactEnv(env, "GOOS", target.OS)
	env = setArtifactEnv(env, "GOARCH", target.Arch)
	env = setArtifactEnv(env, "GOWORK", "off")
	return env
}

func setArtifactEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i := range env {
		if strings.HasPrefix(env[i], prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func runArtifactCommand(ctx context.Context, dir string, args []string, name string, env []string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = env
	}
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(output.String())
		if detail != "" {
			return fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), detail)
		}
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}
