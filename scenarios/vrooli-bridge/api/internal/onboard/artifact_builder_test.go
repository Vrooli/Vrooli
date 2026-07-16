package onboard

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArtifactBuilderRejectsMissingControlPlaneGo(t *testing.T) {
	b := &controlPlaneArtifactBuilder{
		lookPath: func(string) (string, error) { return "", os.ErrNotExist },
		run:      runArtifactCommand,
	}
	_, err := b.Build(context.Background(), ArtifactBuildParams{
		RepoDir: t.TempDir(), Target: NodePlatform{OS: "linux", Arch: "amd64"},
	})
	require.ErrorContains(t, err, "control plane cannot cross-build")
	require.ErrorContains(t, err, "Go is not installed")
}

func TestArtifactBuilderRejectsUnsupportedNodeTarget(t *testing.T) {
	b := &controlPlaneArtifactBuilder{lookPath: func(string) (string, error) { return "/usr/bin/go", nil }}
	_, err := b.Build(context.Background(), ArtifactBuildParams{
		RepoDir: t.TempDir(), Target: NodePlatform{OS: "plan9", Arch: "amd64"},
	})
	require.ErrorContains(t, err, "unsupported bridge node target")
}

func TestArtifactBuilderBuildsExactlyOneTargetWithSharedSidecars(t *testing.T) {
	root := t.TempDir()
	var calls []struct {
		dir  string
		args []string
		env  []string
	}
	b := &controlPlaneArtifactBuilder{
		lookPath: func(string) (string, error) { return "/usr/bin/go", nil },
		run: func(_ context.Context, dir string, args []string, _ string, env []string) error {
			calls = append(calls, struct {
				dir  string
				args []string
				env  []string
			}{dir: dir, args: append([]string(nil), args...), env: append([]string(nil), env...)})
			var output string
			for i, arg := range args {
				if (arg == "--output" || arg == "-o") && i+1 < len(args) {
					output = args[i+1]
				}
			}
			require.NotEmpty(t, output)
			require.NoError(t, os.WriteFile(output, []byte("binary"), 0o755))
			if strings.Contains(strings.Join(args, " "), "./cmd/vrooli-dist") {
				require.NoError(t, os.WriteFile(output+".fp", []byte("same-live-tree\n"), 0o644))
			}
			return nil
		},
	}

	got, err := b.Build(context.Background(), ArtifactBuildParams{
		RepoDir: root, Target: NodePlatform{OS: "darwin", Arch: "arm64"},
	})
	require.NoError(t, err)
	defer os.RemoveAll(got.Directory)
	require.Len(t, calls, 3)
	require.Contains(t, strings.Join(calls[0].args, " "), "run ./cmd/vrooli-dist")
	require.Contains(t, strings.Join(calls[0].args, " "), "--goos darwin --goarch arm64")
	for _, call := range calls[1:] {
		env := strings.Join(call.env, "\n")
		require.Contains(t, env, "CGO_ENABLED=0")
		require.Contains(t, env, "GOOS=darwin")
		require.Contains(t, env, "GOARCH=arm64")
		require.Contains(t, env, "GOWORK=off")
	}
	for _, sidecar := range []string{got.VrooliSidecar, got.BridgeSidecar, got.AgentSidecar} {
		contents, readErr := os.ReadFile(sidecar)
		require.NoError(t, readErr)
		require.Equal(t, "same-live-tree", strings.TrimSpace(string(contents)))
	}
	require.Equal(t, "same-live-tree", got.Fingerprint)
}

func TestArtifactBuilderRealTarget(t *testing.T) {
	if os.Getenv("BRIDGE_REAL_CROSS_BUILD") != "1" {
		t.Skip("set BRIDGE_REAL_CROSS_BUILD=1 for the control-plane cross-build gate")
	}
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	require.NoError(t, err)
	got, err := NewArtifactBuilder().Build(context.Background(), ArtifactBuildParams{
		RepoDir: root, Target: NodePlatform{OS: "linux", Arch: "amd64"},
	})
	require.NoError(t, err)
	defer os.RemoveAll(got.Directory)
	for _, path := range []string{
		got.Vrooli, got.VrooliSidecar, got.BridgeCLI,
		got.BridgeSidecar, got.Agent, got.AgentSidecar,
	} {
		info, statErr := os.Stat(path)
		require.NoError(t, statErr, path)
		require.Positive(t, info.Size(), path)
	}
	// Run with a PATH that contains no Go. If the transferred sidecar does not
	// match this exact root, freshness handling attempts a rebuild and this fails.
	cmd := exec.Command(got.Vrooli, "version")
	cmd.Env = []string{
		"HOME=" + t.TempDir(),
		"PATH=" + t.TempDir(),
		"VROOLI_SOURCE_ROOT=" + root,
	}
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	require.Contains(t, string(out), "Vrooli CLI")
}
