package onboard

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
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
	var callsMu sync.Mutex
	b := &controlPlaneArtifactBuilder{
		lookPath: func(string) (string, error) { return "/usr/bin/go", nil },
		run: func(_ context.Context, dir string, args []string, _ string, env []string) error {
			call := struct {
				dir  string
				args []string
				env  []string
			}{dir: dir, args: append([]string(nil), args...), env: append([]string(nil), env...)}
			callsMu.Lock()
			calls = append(calls, call)
			callsMu.Unlock()
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
	var vrooliCall *struct {
		dir  string
		args []string
		env  []string
	}
	for i := range calls {
		if strings.Contains(strings.Join(calls[i].args, " "), "./cmd/vrooli-dist") {
			call := calls[i]
			vrooliCall = &call
			break
		}
	}
	require.NotNil(t, vrooliCall)
	require.Contains(t, strings.Join(vrooliCall.args, " "), "run ./cmd/vrooli-dist")
	require.Contains(t, strings.Join(vrooliCall.args, " "), "--goos darwin --goarch arm64")
	require.Contains(t, vrooliCall.args, "--allow-missing-darwin-keychain")
	for _, call := range calls {
		if strings.Contains(strings.Join(call.args, " "), "./cmd/vrooli-dist") {
			continue
		}
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
	require.True(t, got.VrooliBootstrapOnly, "Darwin artifacts built on the control plane must be bootstrap-only")
}

func TestArtifactBuilderCachesBySnapshotAndBuildsExecutablesInParallel(t *testing.T) {
	root := t.TempDir()
	cacheRoot := t.TempDir()
	var callsMu sync.Mutex
	var calls int
	active := 0
	maxActive := 0
	b := &controlPlaneArtifactBuilder{
		cacheRoot: cacheRoot,
		lookPath:  func(string) (string, error) { return "/usr/bin/go", nil },
		run: func(_ context.Context, _ string, args []string, _ string, _ []string) error {
			callsMu.Lock()
			calls++
			active++
			if active > maxActive {
				maxActive = active
			}
			callsMu.Unlock()
			defer func() {
				callsMu.Lock()
				active--
				callsMu.Unlock()
			}()
			var output string
			for i, arg := range args {
				if (arg == "--output" || arg == "-o") && i+1 < len(args) {
					output = args[i+1]
				}
			}
			if output == "" {
				return fmt.Errorf("test build did not receive an output path")
			}
			if err := os.WriteFile(output, []byte("binary"), 0o755); err != nil {
				return err
			}
			if strings.Contains(strings.Join(args, " "), "./cmd/vrooli-dist") {
				return os.WriteFile(output+".fp", []byte("snapshot-1\n"), 0o644)
			}
			return nil
		},
	}
	first, err := b.Build(context.Background(), ArtifactBuildParams{RepoDir: root, Target: NodePlatform{OS: "linux", Arch: "amd64"}, CacheKey: "snapshot-1"})
	require.NoError(t, err)
	defer os.RemoveAll(first.Directory)
	second, err := b.Build(context.Background(), ArtifactBuildParams{RepoDir: root, Target: NodePlatform{OS: "linux", Arch: "amd64"}, CacheKey: "snapshot-1"})
	require.NoError(t, err)
	defer os.RemoveAll(second.Directory)
	callsMu.Lock()
	gotCalls, gotMaxActive := calls, maxActive
	callsMu.Unlock()
	require.Equal(t, 3, gotCalls, "the second request should use the immutable snapshot cache")
	require.GreaterOrEqual(t, gotMaxActive, 2, "the three executable builds should overlap")
	require.Equal(t, "snapshot-1", second.Fingerprint)
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
