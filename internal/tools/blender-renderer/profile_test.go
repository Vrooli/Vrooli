package blenderrenderer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
)

func TestPinnedArtifactMatchesManagedToolDeclaration(t *testing.T) {
	data, err := os.ReadFile("tool.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Name        string   `json:"name"`
		Commands    []string `json:"commands"`
		VersionArg  []string `json:"versionArgs"`
		Acquisition struct {
			Targets []struct {
				When struct {
					OS   string `json:"os"`
					Arch string `json:"arch"`
				} `json:"when"`
				URL     string `json:"url"`
				SHA256  string `json:"sha256"`
				Archive string `json:"archive"`
				Layout  string `json:"layout"`
				BinPath string `json:"bin_path"`
			} `json:"targets"`
		} `json:"acquisition"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	artifact := PinnedArtifact()
	if manifest.Name != ToolName || len(manifest.Commands) != 1 || manifest.Commands[0] != manifest.Name {
		t.Fatalf("managed tool name/command differs from profile pin: %+v", manifest)
	}
	if len(manifest.VersionArg) != 1 || manifest.VersionArg[0] != "--version" || len(manifest.Acquisition.Targets) != 1 {
		t.Fatalf("tool declaration is not one fixed version probe and one target: %+v", manifest)
	}
	target := manifest.Acquisition.Targets[0]
	if target.When.OS != "linux" || target.When.Arch != "amd64" || target.URL != artifact.ArchiveURL || target.SHA256 != artifact.ArchiveSHA256 || target.Archive != "tar.xz" || target.Layout != "dir" || target.BinPath != artifact.BinaryMember {
		t.Fatalf("managed acquisition differs from profile artifact: %+v", target)
	}
}

func TestResolveRequiresManagedPinnedTreeBeforeVersionProbe(t *testing.T) {
	root := filepath.Join(t.TempDir(), "opt", "blender-renderer")
	artifact := fixtureArtifact(t, root)
	probeCalls := 0
	probe := func(_ context.Context, path string, args ...string) ([]byte, error) {
		probeCalls++
		if path != filepath.Join(root, BinaryMember) || strings.Join(args, " ") != "--version" {
			t.Fatalf("probe received path=%q args=%q", path, args)
		}
		return []byte("Blender 5.2.2 LTS\n"), nil
	}

	for _, tc := range []struct {
		name   string
		owned  bool
		os     string
		arch   string
		state  State
		probeN int
	}{
		{name: "unmanaged", owned: false, os: "linux", arch: "amd64", state: StateUnmanaged},
		{name: "unsupported", owned: true, os: "darwin", arch: "arm64", state: StateUnsupported},
		{name: "managed", owned: true, os: "linux", arch: "amd64", state: StatePresent, probeN: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := probeCalls
			response, err := Resolve(context.Background(), root, tc.owned, artifact, tc.os, tc.arch, probe)
			if err != nil {
				t.Fatal(err)
			}
			if response.State != tc.state || probeCalls-before != tc.probeN {
				t.Fatalf("resolve state=%q probe calls=%d, want %q/%d", response.State, probeCalls-before, tc.state, tc.probeN)
			}
			if tc.state == StatePresent && (response.Version != "Blender 5.2.2 LTS" || response.ExpectedTreeSHA256 != artifact.TreeSHA256) {
				t.Fatalf("present response lost pin or version: %+v", response)
			}
		})
	}
}

func TestResolveRequiresExactPinnedLTSVersionLine(t *testing.T) {
	root := filepath.Join(t.TempDir(), "opt", "blender-renderer")
	artifact := fixtureArtifact(t, root)
	for _, tc := range []struct {
		name   string
		output string
		state  State
	}{
		{name: "actual official output", output: "Blender 5.2.2 LTS\n", state: StatePresent},
		{name: "version without pinned LTS suffix", output: "Blender 5.2.2\n", state: StateProbeFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			probe := func(context.Context, string, ...string) ([]byte, error) {
				return []byte(tc.output), nil
			}
			got, err := Resolve(context.Background(), root, true, artifact, "linux", "amd64", probe)
			if err != nil || got.State != tc.state {
				t.Fatalf("Resolve() = state %q, err %v; want %q", got.State, err, tc.state)
			}
			if tc.state == StatePresent && got.Version != "Blender 5.2.2 LTS" {
				t.Fatalf("present version %q does not retain exact official line", got.Version)
			}
		})
	}
}

func TestResolveClassifiesMissingPartialTamperedAndProbeFailure(t *testing.T) {
	artifact := fixtureArtifact(t, filepath.Join(t.TempDir(), "tree"))
	root := filepath.Join(t.TempDir(), "opt", "blender-renderer")
	missing, err := Resolve(context.Background(), root, false, artifact, "linux", "amd64", nil)
	if err != nil || missing.State != StateMissing {
		t.Fatalf("missing = %+v, %v", missing, err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	partial, err := Resolve(context.Background(), root, true, artifact, "linux", "amd64", nil)
	if err != nil || partial.State != StatePartial {
		t.Fatalf("ledger-only install = %+v, %v", partial, err)
	}

	artifact = fixtureArtifact(t, root)
	if err := os.WriteFile(filepath.Join(root, "unexpected"), []byte("extra"), 0o600); err != nil {
		t.Fatal(err)
	}
	tampered, err := Resolve(context.Background(), root, true, artifact, "linux", "amd64", func(context.Context, string, ...string) ([]byte, error) {
		t.Fatal("tree mismatch must not execute the version probe")
		return nil, nil
	})
	if err != nil || tampered.State != StateTampered {
		t.Fatalf("extra managed-tree member = %+v, %v", tampered, err)
	}

	root = filepath.Join(t.TempDir(), "opt", "blender-renderer")
	artifact = fixtureArtifact(t, root)
	probeFailed, err := Resolve(context.Background(), root, true, artifact, "linux", "amd64", func(context.Context, string, ...string) ([]byte, error) {
		return []byte("Blender 5.2.1\n"), nil
	})
	if err != nil || probeFailed.State != StateProbeFailed {
		t.Fatalf("version mismatch = %+v, %v", probeFailed, err)
	}
}

func TestResolveAcceptsOnlySafeRelativeTreeLinks(t *testing.T) {
	root := filepath.Join(t.TempDir(), "opt")
	if err := os.MkdirAll(filepath.Join(root, ArchiveRoot, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, BinaryMember), []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ArchiveRoot, "lib", "libactual.so"), []byte("library"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("libactual.so", filepath.Join(root, ArchiveRoot, "lib", "libalias.so")); err != nil {
		t.Fatal(err)
	}
	artifact := PinnedArtifact()
	artifact.BinarySize = int64(len("binary"))
	artifact.BinarySHA256 = digest([]byte("binary"))
	artifact.TreeSHA256, _ = binaryfetch.TreeDigest(root)
	valid, err := Resolve(context.Background(), root, true, artifact, "linux", "amd64", func(context.Context, string, ...string) ([]byte, error) {
		return []byte("Blender 5.2.2 LTS\n"), nil
	})
	if err != nil || valid.State != StatePresent {
		t.Fatalf("safe relative symlink was rejected: %+v, %v", valid, err)
	}
	if err := os.Remove(filepath.Join(root, ArchiveRoot, "lib", "libalias.so")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../../../outside", filepath.Join(root, ArchiveRoot, "lib", "libalias.so")); err != nil {
		t.Fatal(err)
	}
	escaping, err := Resolve(context.Background(), root, true, artifact, "linux", "amd64", func(context.Context, string, ...string) ([]byte, error) {
		t.Fatal("unsafe symlink tree must not execute version probe")
		return nil, nil
	})
	if err != nil || escaping.State != StateTampered {
		t.Fatalf("escaping symlink = %+v, %v", escaping, err)
	}
}

func TestArtifactIdentityIsContentBasedAndIncludesWholeTreePin(t *testing.T) {
	artifact := PinnedArtifact()
	first := ArtifactIdentity(artifact)
	artifact.TreeSHA256 = strings.Repeat("a", 64)
	if ArtifactIdentity(artifact) == first {
		t.Fatal("artifact identity omitted expected tree digest")
	}
	artifact = PinnedArtifact()
	artifact.BinaryMember = "other/path/blender"
	if ArtifactIdentity(artifact) == first {
		t.Fatal("artifact identity omitted executable member path")
	}
}

func TestVersionProbeUsesOnlyFixedArgAndCapsActualSubprocessOutput(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("shell-script process fixture is Linux-specific")
	}
	root := t.TempDir()
	good := filepath.Join(root, "probe")
	if err := os.WriteFile(good, []byte("#!/bin/sh\n[ \"$1\" = --version ] || exit 8\nprintf 'Blender 5.2.2 LTS\\n'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	output, err := VersionProbe(context.Background(), good, "--version")
	if err != nil || string(output) != "Blender 5.2.2 LTS\n" {
		t.Fatalf("bounded fixed probe = %q, %v", output, err)
	}
	if _, err := VersionProbe(context.Background(), good, "--python-expr"); err == nil {
		t.Fatal("non-version invocation was accepted")
	}
	overflow := filepath.Join(root, "overflow")
	if err := os.WriteFile(overflow, []byte("#!/bin/sh\nprintf '%40000s' x\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if output, err := VersionProbe(context.Background(), overflow, "--version"); err == nil || output != nil {
		t.Fatalf("oversized subprocess output returned %d bytes, err=%v", len(output), err)
	}
}

func fixtureArtifact(t *testing.T, root string) Artifact {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ArchiveRoot), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, LicenseMember)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, BinaryMember), []byte("fixture blender"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, LicenseMember), []byte("fixture license"), 0o644); err != nil {
		t.Fatal(err)
	}
	artifact := PinnedArtifact()
	artifact.BinarySize = int64(len("fixture blender"))
	artifact.BinarySHA256 = digest([]byte("fixture blender"))
	artifact.LicenseSize = int64(len("fixture license"))
	artifact.LicenseSHA256 = digest([]byte("fixture license"))
	var err error
	artifact.TreeSHA256, err = binaryfetch.TreeDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}
