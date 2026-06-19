package validation

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/vrooli/packages/proto/genmanifest"
)

func TestManifestVerifierReportsCleanAndDrift(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, repoRoot string)
		assert func(t *testing.T, status GenSyncStatus)
	}{
		{
			name: "clean",
			assert: func(t *testing.T, status GenSyncStatus) {
				require.True(t, status.InSync)
				require.Empty(t, status.Drift)
			},
		},
		{
			name: "stale input digest",
			mutate: func(t *testing.T, repoRoot string) {
				path := filepath.Join(repoRoot, "packages", "proto", "schemas", "demo", "v1", "core", "core.proto")
				require.NoError(t, os.WriteFile(path, []byte("syntax = \"proto3\";\npackage demo.v1.core;\nmessage Changed {}\n"), 0o644))
			},
			assert: func(t *testing.T, status GenSyncStatus) {
				require.False(t, status.InSync)
				require.Contains(t, status.Drift, "packages/proto/schemas/demo/v1/core/core.proto")
			},
		},
		{
			name: "output hash mismatch ignored when inputs are unchanged",
			mutate: func(t *testing.T, repoRoot string) {
				path := filepath.Join(repoRoot, "packages", "proto", "gen", "go", "demo", "v1", "core", "core.pb.go")
				require.NoError(t, os.WriteFile(path, []byte("changed"), 0o644))
			},
			assert: func(t *testing.T, status GenSyncStatus) {
				require.True(t, status.InSync)
				require.Empty(t, status.Drift)
			},
		},
		{
			name: "orphan output ignored when inputs are unchanged",
			mutate: func(t *testing.T, repoRoot string) {
				path := filepath.Join(repoRoot, "packages", "proto", "gen", "python", "demo", "v1", "old", "py.typed")
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, []byte{}, 0o644))
			},
			assert: func(t *testing.T, status GenSyncStatus) {
				require.True(t, status.InSync)
				require.Empty(t, status.Drift)
			},
		},
		{
			name: "output hash checked when input digest is stale",
			mutate: func(t *testing.T, repoRoot string) {
				inputPath := filepath.Join(repoRoot, "packages", "proto", "schemas", "demo", "v1", "core", "core.proto")
				require.NoError(t, os.WriteFile(inputPath, []byte("syntax = \"proto3\";\npackage demo.v1.core;\nmessage Changed {}\n"), 0o644))
				outputPath := filepath.Join(repoRoot, "packages", "proto", "gen", "go", "demo", "v1", "core", "core.pb.go")
				require.NoError(t, os.WriteFile(outputPath, []byte("changed"), 0o644))
			},
			assert: func(t *testing.T, status GenSyncStatus) {
				require.False(t, status.InSync)
				require.Contains(t, status.Drift, "packages/proto/schemas/demo/v1/core/core.proto")
				require.Contains(t, status.Drift, "packages/proto/gen/go/demo/v1/core/core.pb.go")
			},
		},
		{
			name: "toolchain drift only",
			mutate: func(t *testing.T, repoRoot string) {
				writeTool(t, repoRoot, "buf", "9.9.9")
			},
			assert: func(t *testing.T, status GenSyncStatus) {
				require.True(t, status.InSync)
				require.Empty(t, status.Drift)
				require.True(t, status.ToolchainDrift)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoRoot := makeManifestRepo(t)
			if tt.mutate != nil {
				tt.mutate(t, repoRoot)
			}

			status, err := NewManifestVerifier(repoRoot).CheckScenario(context.Background(), "demo")

			require.NoError(t, err)
			tt.assert(t, status)
		})
	}
}

func TestManifestVerifierReportsMissingManifest(t *testing.T) {
	repoRoot := makeManifestRepo(t)
	require.NoError(t, os.Remove(filepath.Join(repoRoot, "packages", "proto", "gen", "manifests", "demo.lock.json")))

	status, err := NewManifestVerifier(repoRoot).CheckScenario(context.Background(), "demo")

	require.NoError(t, err)
	require.True(t, status.ManifestMissing)
	require.Contains(t, status.Drift, "packages/proto/gen/manifests/demo.lock.json")
}

func makeManifestRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	protoRoot := filepath.Join(root, "packages", "proto")
	require.NoError(t, os.MkdirAll(filepath.Join(protoRoot, "schemas", "demo", "v1", "core"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(protoRoot, "schemas", "demo", "v1", "core", "core.proto"), []byte("syntax = \"proto3\";\npackage demo.v1.core;\nmessage Demo {}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(protoRoot, "buf.gen.yaml"), []byte("version: v2\n"), 0o644))
	for _, path := range []string{
		filepath.Join(protoRoot, "gen", "go", "demo", "v1", "core", "core.pb.go"),
		filepath.Join(protoRoot, "gen", "typescript", "demo", "v1", "core", "core_pb.ts"),
		filepath.Join(protoRoot, "gen", "typescript", "js", "demo", "v1", "core", "core_pb.js"),
		filepath.Join(protoRoot, "gen", "python", "demo", "v1", "core", "core_pb2.py"),
		filepath.Join(protoRoot, "gen", "python", "demo", "v1", "core", "py.typed"),
	} {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(filepath.Base(path)), 0o644))
	}
	for _, tool := range []string{"buf", "protoc-gen-go", "protoc-gen-connect-go", "protoc-gen-es", "protoc"} {
		writeTool(t, root, tool, "1.0.0")
	}
	manifest, err := genmanifest.BuildManifest(genmanifest.Options{RepoRoot: root, ProtoRoot: protoRoot}, "demo")
	require.NoError(t, err)
	require.NoError(t, genmanifest.WriteManifest(genmanifest.ManifestPath(protoRoot, "demo"), manifest))
	return root
}

func writeTool(t *testing.T, repoRoot, name, version string) {
	t.Helper()
	path := filepath.Join(repoRoot, "internal", "tools", name, "tool.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	raw, err := json.Marshal(map[string]string{"name": name, "version": version})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o644))
}
