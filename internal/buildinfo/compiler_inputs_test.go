package buildinfo

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vrooli/repo-contract-go/repocontracttest"
)

func TestRootFingerprintExcludesProtoOwnerWorkTreesButKeepsPublishedAndHiddenSource(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")
	writeTestFile(t, root, "packages/proto/gen/go/types.pb.go", "package generated\nconst Value = 1\n")
	writeTestFile(t, root, "packages/.authored/source.go", "package hidden\nconst Value = 1\n")
	writeTestFile(t, root, "internal/.proto-gen-verify-authored/source.go", "package authored\nconst Value = 1\n")
	writeTestFile(t, root, "packages/proto/gen.previous-authored/source.go", "package authored\nconst Value = 1\n")
	before, err := ComputeSourceFingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"packages/.proto-gen-verify-123/gen/go/copy.pb.go",
		"packages/.proto-gen-stage-123/gen/go/copy.pb.go",
		"packages/.proto-descriptor-123/generated.go",
		"packages/proto/.proto-compatibility-123/go/copy.pb.go",
		"packages/proto/gen.previous/go/copy.pb.go",
	} {
		writeTestFile(t, root, path, "package disposable\n")
	}
	during, err := ComputeSourceFingerprint(root)
	if err != nil || during != before {
		t.Fatalf("Proto owner work changed root source identity: before=%s during=%s err=%v", before, during, err)
	}
	for _, path := range []string{
		"packages/proto/gen/go/types.pb.go",
		"packages/.authored/source.go",
		"internal/.proto-gen-verify-authored/source.go",
		"packages/proto/gen.previous-authored/source.go",
	} {
		writeTestFile(t, root, path, "package authored\nconst Value = 222\n")
		after, err := ComputeSourceFingerprint(root)
		if err != nil || after == before {
			t.Fatalf("real source %s was hidden: %v", path, err)
		}
		before = after
	}
}

func TestExplicitEmbeddedAssetInProtoWorkNamespaceRemainsCompilerInput(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.test/fixture\n\ngo 1.25.0\n")
	writeTestFile(t, root, "packages/embed.go", "package fixture\nimport _ \"embed\"\n//go:embed all:.proto-gen-verify-authored/policy.json\nvar Policy string\n")
	writeTestFile(t, root, "packages/.proto-gen-verify-authored/policy.json", "first authored policy")
	cmd := exec.Command("go", "test", "-mod=readonly", "./packages")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOWORK=off")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("installed compiler rejected embed fixture: %v\n%s", err, output)
	}
	before, err := ComputeSourceFingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, "packages/.proto-gen-verify-authored/policy.json", "second authored policy")
	after, err := ComputeSourceFingerprint(root)
	if err != nil || before == after {
		t.Fatalf("explicit authored compiler asset was hidden by directory classification: %v", err)
	}
}

func TestProjectFingerprintIncludesCompilerSourcesAndEmbeddedAssets(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\nimport _ \"embed\"\n//go:embed rules.json\nvar rules string\n")
	writeTestFile(t, root, "cmd/vrooli/rules.json", "version one")
	writeTestFile(t, root, "internal/native/native.go", "package native\n")
	previous, err := ComputeSourceFingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []struct{ file, data string }{
		{"cmd/vrooli/rules.json", "version two"},
		{"internal/native/implementation.c", "int value = 1;"},
		{"internal/native/implementation.h", "#define VERSION 2"},
		{"internal/native/implementation.hxx", "#define CXX_VERSION 3"},
		{"internal/native/implementation.S", "assembly source"},
	} {
		writeTestFile(t, root, change.file, change.data)
		next, err := ComputeSourceFingerprint(root)
		if err != nil || next == previous {
			t.Fatalf("compiler input %s did not change stamp: %v", change.file, err)
		}
		previous = next
	}
	if err := os.Remove(filepath.Join(root, "cmd/vrooli/rules.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := ComputeSourceFingerprint(root); err == nil || !strings.Contains(err.Error(), "matches no files") {
		t.Fatalf("missing embedded asset was accepted: %v", err)
	}
}

func TestProjectFingerprintIncludesCanonicalCompileFlagPolicy(t *testing.T) {
	root := t.TempDir()
	repocontracttest.WriteRepoContract(t, root, "scenarios")
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")
	contract := filepath.Join(root, ".vrooli/repo-contract.json")
	data, err := os.ReadFile(contract)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	set := func(flags []string) {
		t.Helper()
		document["build.go_flags"] = map[string]any{"distribution": flags}
		data, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(contract, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	set([]string{"-trimpath"})
	before, err := ComputeSourceFingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	set([]string{"-trimpath", "-buildvcs=false"})
	after, err := ComputeSourceFingerprint(root)
	if err != nil || after == before {
		t.Fatalf("canonical compile flag policy did not invalidate stamp: %v", err)
	}
}

// Compare the selected asset set to the installed compiler, including cases
// where public documentation and this Go version differ about empty globs.
// Fixtures use only the standard library and forbid module/toolchain fetching.
func TestEmbeddedInputsMatchInstalledCompilerDirectoryRules(t *testing.T) {
	for _, tc := range []struct{ name, pattern, special string }{
		{"glob empty directory", "assets/*", "empty"},
		{"recursive symlink", "assets", "symlink"},
		{"literal symlink", "assets/link", "symlink"},
		{"recursive nested module", "assets", "module"},
		{"glob nested module", "assets/*", "module"},
		{"recursive vendor", "assets", "vendor"},
		{"all excludes vcs directories", "all:assets", "vcs"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeTestFile(t, root, "go.mod", "module example.test/embedded\n\ngo 1.26\n")
			writeTestFile(t, root, "main.go", "package embedded\nimport \"embed\"\n//go:embed "+tc.pattern+"\nvar Files embed.FS\n")
			writeTestFile(t, root, "assets/data.json", "compiled")
			switch tc.special {
			case "empty":
				if err := os.Mkdir(filepath.Join(root, "assets/empty"), 0o755); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink("data.json", filepath.Join(root, "assets/link")); err != nil {
					t.Fatal(err)
				}
			case "module":
				writeTestFile(t, root, "assets/child/go.mod", "module example.test/child\n")
				writeTestFile(t, root, "assets/child/data.json", "excluded")
			case "vendor":
				writeTestFile(t, root, "assets/vendor/data.json", "compiled vendor bytes")
			case "vcs":
				writeTestFile(t, root, "assets/.git/config", "excluded vcs bytes")
			}
			command := exec.Command("go", "list", "-mod=readonly", "-json", ".")
			command.Dir = root
			command.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local")
			output, compilerErr := command.CombinedOutput()
			files, resolverErr := embeddedFiles(root, tc.pattern)
			if (compilerErr == nil) != (resolverErr == nil) {
				t.Fatalf("compiler error=%v output=%s; resolver error=%v", compilerErr, output, resolverErr)
			}
			if compilerErr != nil {
				t.Logf("compiler refused fixture: %s", output)
				return
			}
			var pkg struct{ EmbedFiles []string }
			if err := json.Unmarshal(output, &pkg); err != nil {
				t.Fatal(err)
			}
			var relative []string
			for _, file := range files {
				rel, err := filepath.Rel(root, file)
				if err != nil {
					t.Fatal(err)
				}
				relative = append(relative, filepath.ToSlash(rel))
			}
			slices.Sort(relative)
			slices.Sort(pkg.EmbedFiles)
			if !slices.Equal(relative, pkg.EmbedFiles) {
				t.Fatalf("assets=%v; compiler assets=%v", relative, pkg.EmbedFiles)
			}
		})
	}
}

func TestProjectFingerprintTracksCompiledRootCLIManifest(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")
	writeTestFile(t, root, "internal/example.go", "package example\n")
	writeTestFile(t, root, "packages/example/example.go", "package example\n")
	writeTestFile(t, root, "go.mod", "module example.test/root\n")
	writeTestFile(t, root, "go.sum", "")
	source := "package manifest\nimport _ \"embed\"\n//go:embed manifest.json\nvar Raw []byte\n"
	writeTestFile(t, root, "cli/manifest_embed.go", source)
	writeTestFile(t, root, "cli/manifest.json", `{"commands":["status"]}`)
	targets := FingerprintTargetsForExecutable("vrooli")
	before, err := ComputeSourceFingerprintReport(root, FingerprintOptions{RequireExistingTargets: true}, targets...)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []struct{ path, content string }{
		{"cli/manifest.json", `{"commands":["status","restart"]}`},
		{"cli/manifest_embed.go", source + "func Bytes() []byte { return append([]byte(nil), Raw...) }\n"},
	} {
		writeTestFile(t, root, change.path, change.content)
		after, err := ComputeSourceFingerprintReport(root, FingerprintOptions{RequireExistingTargets: true}, targets...)
		if err != nil || after.Fingerprint == before.Fingerprint {
			t.Fatalf("compiled command contract change %s did not invalidate stamp: %v", change.path, err)
		}
		before = after
	}
}

func TestProjectFingerprintTracksEmbeddedDirectoryMembershipAndAllPatterns(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "main.go", "package main\nimport _ \"embed\"\n//go:embed \"assets with spaces\" all:extra\nvar assets embed.FS\n")
	writeTestFile(t, root, "assets with spaces/one.json", "one")
	writeTestFile(t, root, "extra/_hidden", "hidden")
	previous, err := ComputeSourceFingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"assets with spaces/two.json", "extra/_hidden", "extra/.hidden"} {
		writeTestFile(t, root, file, "changed")
		next, err := ComputeSourceFingerprint(root)
		if err != nil || next == previous {
			t.Fatalf("embedded membership/content change %s ignored: %v", file, err)
		}
		previous = next
	}
	writeTestFile(t, root, "assets with spaces/_excluded", "ignored by directory embed")
	writeTestFile(t, root, "README.md", "not compiled")
	next, err := ComputeSourceFingerprint(root)
	if err != nil || next != previous {
		t.Fatalf("unrelated files invalidated stamp: %v", err)
	}
}

func TestProjectFingerprintDoesNotParseRawStringExamplesAsEmbedDirectives(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "main.go", "package main\nvar example = `\n//go:embed absent.json\n`\n")
	if _, err := ComputeSourceFingerprint(root); err != nil {
		t.Fatalf("raw string became an input declaration: %v", err)
	}
}

func TestProjectFingerprintRefusesEmbeddedAliasAndEscapingPatterns(t *testing.T) {
	for _, pattern := range []string{"../outside", "assets/link/data.json", "assets/leaf.json"} {
		t.Run(pattern, func(t *testing.T) {
			root := t.TempDir()
			writeTestFile(t, root, "main.go", "package main\nimport _ \"embed\"\n//go:embed "+pattern+"\nvar data string\n")
			outside := t.TempDir()
			writeTestFile(t, outside, "data.json", "outside bytes")
			if err := os.Mkdir(filepath.Join(root, "assets"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, filepath.Join(root, "assets/link")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(outside, "data.json"), filepath.Join(root, "assets/leaf.json")); err != nil {
				t.Fatal(err)
			}
			if _, err := ComputeSourceFingerprint(root); err == nil {
				t.Fatal("embedded alias or escaping pattern was accepted")
			}
		})
	}
}
