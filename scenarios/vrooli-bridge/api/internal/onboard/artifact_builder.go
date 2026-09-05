package onboard

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// controlPlaneArtifactBuilder builds all node executables from one repository
// root. The Vrooli binary is delegated to cmd/vrooli-dist, the shared primitive
// also used by release packaging; bridge-specific modules are cross-built with
// the same target environment.
type controlPlaneArtifactBuilder struct {
	lookPath func(string) (string, error)
	run      func(context.Context, string, []string, string, []string) error
}

// NewArtifactBuilder constructs the production control-plane cross-builder.
func NewArtifactBuilder() ArtifactBuilder {
	return &controlPlaneArtifactBuilder{lookPath: exec.LookPath, run: runArtifactCommand}
}

var _ ArtifactBuilder = (*controlPlaneArtifactBuilder)(nil)

func (b *controlPlaneArtifactBuilder) Build(ctx context.Context, p ArtifactBuildParams) (PrebuiltArtifacts, error) {
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
	if err := b.run(ctx, root, vrooliArgs, "go", nil); err != nil {
		return PrebuiltArtifacts{}, fmt.Errorf("build vrooli with shared distribution primitive: %w", err)
	}
	if err := b.run(ctx, filepath.Join(root, "scenarios", "vrooli-bridge", "cli"), []string{
		"build", "-trimpath", "-o", bridgePath, ".",
	}, "go", env); err != nil {
		return PrebuiltArtifacts{}, fmt.Errorf("cross-build vrooli-bridge CLI: %w", err)
	}
	if err := b.run(ctx, filepath.Join(root, "scenarios", "vrooli-bridge", "agent"), []string{
		"build", "-trimpath", "-o", agentPath, ".",
	}, "go", env); err != nil {
		return PrebuiltArtifacts{}, fmt.Errorf("cross-build node agent: %w", err)
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
	return PrebuiltArtifacts{
		Directory: dir, Vrooli: vrooliPath, VrooliSidecar: vrooliSidecar,
		BridgeCLI: bridgePath, BridgeSidecar: bridgeSidecar,
		Agent: agentPath, AgentSidecar: agentSidecar,
		Fingerprint: fingerprint, Target: p.Target,
		VrooliBootstrapOnly: p.Target.OS == "darwin",
	}, nil
}

func supportedBridgeTarget(target NodePlatform) bool {
	if target.OS != "linux" && target.OS != "darwin" {
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
