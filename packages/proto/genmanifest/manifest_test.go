package genmanifest

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildManifestUsesTransitiveSourceClosureAndOutputs(t *testing.T) {
	repoRoot, protoRoot := makeRepo(t)
	writeFile(t, filepath.Join(protoRoot, "schemas", "demo", "v1", "core", "core.proto"), `syntax = "proto3";
package demo.v1.core;
import "common/v1/shared/shared.proto";
message Demo { common.v1.shared.Shared value = 1; }
`)
	writeFile(t, filepath.Join(protoRoot, "schemas", "common", "v1", "shared", "shared.proto"), `syntax = "proto3";
package common.v1.shared;
message Shared {}
`)
	writeFile(t, filepath.Join(protoRoot, "gen", "go", "demo", "v1", "core", "core.pb.go"), "go")
	writeFile(t, filepath.Join(protoRoot, "gen", "typescript", "demo", "v1", "core", "core_pb.ts"), "ts")
	writeFile(t, filepath.Join(protoRoot, "gen", "python", "demo", "v1", "core", "core_pb2.py"), "py")
	writeFile(t, filepath.Join(protoRoot, "gen", "python", "demo", "v1", "core", "py.typed"), "")

	manifest, err := BuildManifest(Options{RepoRoot: repoRoot, ProtoRoot: protoRoot}, "demo")
	if err != nil {
		t.Fatal(err)
	}
	wantInputs := []string{
		"schemas/common/v1/shared/shared.proto",
		"schemas/demo/v1/core/core.proto",
	}
	if !equalStrings(manifest.InputFiles, wantInputs) {
		t.Fatalf("InputFiles = %#v, want %#v", manifest.InputFiles, wantInputs)
	}
	if _, ok := manifest.Outputs["gen/python/demo/v1/core/py.typed"]; !ok {
		t.Fatalf("manifest missing py.typed output: %#v", manifest.Outputs)
	}
}

func TestWriteManifestDeterministic(t *testing.T) {
	repoRoot, protoRoot := makeRepo(t)
	writeFile(t, filepath.Join(protoRoot, "schemas", "demo", "v1", "core", "core.proto"), "syntax = \"proto3\";\npackage demo.v1.core;\n")
	writeFile(t, filepath.Join(protoRoot, "gen", "go", "demo", "artifact.pb.go"), "go")

	manifest, err := BuildManifest(Options{RepoRoot: repoRoot, ProtoRoot: protoRoot}, "demo")
	if err != nil {
		t.Fatal(err)
	}
	path := ManifestPath(protoRoot, "demo")
	if err := WriteManifest(path, manifest); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteManifest(path, manifest); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("manifest write is not deterministic")
	}
	if !bytes.HasSuffix(first, []byte("\n")) {
		t.Fatal("manifest should end with newline")
	}
}

func TestChangedScenariosDetectsInputAndToolchainChanges(t *testing.T) {
	repoRoot, protoRoot := makeRepo(t)
	writeFile(t, filepath.Join(protoRoot, "schemas", "leaf", "v1", "leaf.proto"), "syntax = \"proto3\";\npackage leaf.v1;\nmessage Leaf {}\n")
	writeFile(t, filepath.Join(protoRoot, "schemas", "other", "v1", "other.proto"), "syntax = \"proto3\";\npackage other.v1;\nmessage Other {}\n")
	for _, name := range []string{"leaf", "other"} {
		manifest, err := BuildManifest(Options{RepoRoot: repoRoot, ProtoRoot: protoRoot}, name)
		if err != nil {
			t.Fatal(err)
		}
		if err := WriteManifest(ManifestPath(protoRoot, name), manifest); err != nil {
			t.Fatal(err)
		}
	}
	changed, err := ChangedScenarios(Options{RepoRoot: repoRoot, ProtoRoot: protoRoot})
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 0 {
		t.Fatalf("unchanged locks reported as changed: %v", changed)
	}
	writeFile(t, filepath.Join(protoRoot, "schemas", "leaf", "v1", "leaf.proto"), "syntax = \"proto3\";\npackage leaf.v1;\nmessage Leaf { string value = 1; }\n")
	changed, err = ChangedScenarios(Options{RepoRoot: repoRoot, ProtoRoot: protoRoot})
	if err != nil {
		t.Fatal(err)
	}
	if !equalStrings(changed, []string{"leaf"}) {
		t.Fatalf("input change reported %v, want [leaf]", changed)
	}
	writeFile(t, filepath.Join(repoRoot, "internal", "tools", "buf", "tool.json"), `{"version":"2.0.0"}`)
	changed, err = ChangedScenarios(Options{RepoRoot: repoRoot, ProtoRoot: protoRoot})
	if err != nil {
		t.Fatal(err)
	}
	if !equalStrings(changed, []string{"leaf", "other"}) {
		t.Fatalf("toolchain change reported %v, want both sets", changed)
	}
}

func TestChangedScenariosIncludesDependentsOfMissingLock(t *testing.T) {
	repoRoot, protoRoot := makeRepo(t)
	writeFile(t, filepath.Join(protoRoot, "schemas", "common", "v1", "common.proto"), "syntax = \"proto3\";\npackage common.v1;\nmessage Common {}\n")
	writeFile(t, filepath.Join(protoRoot, "schemas", "dependent", "v1", "dependent.proto"), "syntax = \"proto3\";\npackage dependent.v1;\nimport \"common/v1/common.proto\";\nmessage Dependent { common.v1.Common value = 1; }\n")
	dependent, err := BuildManifest(Options{RepoRoot: repoRoot, ProtoRoot: protoRoot}, "dependent")
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteManifest(ManifestPath(protoRoot, "dependent"), dependent); err != nil {
		t.Fatal(err)
	}
	changed, err := ChangedScenarios(Options{RepoRoot: repoRoot, ProtoRoot: protoRoot})
	if err != nil {
		t.Fatal(err)
	}
	if !equalStrings(changed, []string{"common", "dependent"}) {
		t.Fatalf("missing lock scope = %v, want [common dependent]", changed)
	}
}

func makeRepo(t *testing.T) (string, string) {
	t.Helper()
	repoRoot := t.TempDir()
	protoRoot := filepath.Join(repoRoot, "packages", "proto")
	writeFile(t, filepath.Join(protoRoot, "buf.gen.yaml"), "version: v2\n")
	for _, tool := range []string{"buf", "protoc-gen-go", "protoc-gen-connect-go", "protoc-gen-es", "protoc"} {
		writeFile(t, filepath.Join(repoRoot, "internal", "tools", tool, "tool.json"), `{"version":"1.0.0"}`)
	}
	return repoRoot, protoRoot
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
