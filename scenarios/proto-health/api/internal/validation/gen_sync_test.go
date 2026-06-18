package validation

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratedArtifactCheckerAttributesCompileFailures(t *testing.T) {
	tests := []struct {
		name        string
		stderr      string
		wantBlocked bool
		wantErr     bool
		wantFile    string
	}{
		{
			name:        "target proto compile failure is hard drift",
			stderr:      "schemas/demo/v1/core/core.proto:3:1: expected identifier",
			wantErr:     true,
			wantBlocked: false,
		},
		{
			name:        "foreign proto compile failure blocks verification",
			stderr:      "schemas/other/v1/core/core.proto:3:1: expected identifier",
			wantBlocked: true,
			wantFile:    "schemas/other/v1/core/core.proto",
		},
		{
			name:        "unattributable failure blocks verification",
			stderr:      "compiler failed before reporting a file",
			wantBlocked: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoRoot := makeGenSyncRepo(t)
			installFakeBuf(t, tt.stderr, "")

			status, err := NewGeneratedArtifactChecker(repoRoot).CheckScenario(context.Background(), "demo")

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantBlocked, status.Blocked)
			if tt.wantFile != "" {
				require.Equal(t, []string{tt.wantFile}, status.BlockedBy)
			}
		})
	}
}

func TestGeneratedArtifactCheckerReportsCleanAndDrift(t *testing.T) {
	tests := []struct {
		name       string
		generated  string
		wantInSync bool
		wantDrift  bool
	}{
		{name: "clean", generated: "current", wantInSync: true},
		{name: "drift", generated: "changed", wantDrift: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoRoot := makeGenSyncRepo(t)
			installFakeBuf(t, "", tt.generated)

			status, err := NewGeneratedArtifactChecker(repoRoot).CheckScenario(context.Background(), "demo")

			require.NoError(t, err)
			require.Equal(t, tt.wantInSync, status.InSync)
			if tt.wantDrift {
				require.Equal(t, []string{"packages/proto/gen/go/demo"}, status.Drift)
			}
		})
	}
}

func makeGenSyncRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	protoRoot := filepath.Join(root, "packages", "proto")
	require.NoError(t, os.MkdirAll(filepath.Join(protoRoot, "schemas", "demo", "v1", "core"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(protoRoot, "schemas", "demo", "v1", "core", "core.proto"), []byte("syntax = \"proto3\";\n"), 0o644))
	for _, name := range []string{"buf.yaml", "buf.gen.yaml"} {
		require.NoError(t, os.WriteFile(filepath.Join(protoRoot, name), []byte("version: v1\n"), 0o644))
	}
	require.NoError(t, os.MkdirAll(filepath.Join(protoRoot, "gen", "go", "demo"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(protoRoot, "gen", "go", "demo", "artifact.txt"), []byte("current"), 0o644))
	return root
}

func installFakeBuf(t *testing.T, stderr, generated string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake shell buf is POSIX-only")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "buf")
	script := "#!/usr/bin/env bash\n" +
		"if [[ \"$*\" != \"generate --path schemas/demo\" ]]; then echo unexpected args: \"$*\" >&2; exit 2; fi\n"
	if stderr != "" {
		script += "echo '" + stderr + "' >&2\nexit 1\n"
	} else {
		script += "mkdir -p gen/go/demo\n" +
			"printf '" + generated + "' > gen/go/demo/artifact.txt\n"
	}
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
