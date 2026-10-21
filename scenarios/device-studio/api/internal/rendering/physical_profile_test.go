package rendering

import (
	"os"
	"path/filepath"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/repo-contract-go/cliinvoke"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

func TestValidatePhysicalBlenderProfilePinsManagedArtifact(t *testing.T) {
	profile := validPhysicalBlenderProfile()
	if err := validatePhysicalBlenderProfile(profile); err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*cliv1.BlenderToolProfile){
		"wrong artifact": func(p *cliv1.BlenderToolProfile) { p.ArtifactIdentity = "other" },
		"wrong tree":     func(p *cliv1.BlenderToolProfile) { p.ExpectedTreeSha256 = testDigest('f') },
		"wrong platform": func(p *cliv1.BlenderToolProfile) { p.Architecture = "arm64" },
		"not present": func(p *cliv1.BlenderToolProfile) {
			p.State = cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_MISSING
		},
		"wrong version":   func(p *cliv1.BlenderToolProfile) { p.Blender.Version = "Blender 5.2.2" },
		"wrong binary":    func(p *cliv1.BlenderToolProfile) { p.Blender.Sha256 = testDigest('f') },
		"outside archive": func(p *cliv1.BlenderToolProfile) { p.Blender.Path = "/tmp/blender" },
		"relative path": func(p *cliv1.BlenderToolProfile) {
			p.Blender.Path = filepath.Join("tools", PhysicalBlenderArchive, "blender")
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			candidate := validPhysicalBlenderProfile()
			mutate(candidate)
			if err := validatePhysicalBlenderProfile(candidate); err == nil {
				t.Fatal("untrusted profile was accepted")
			}
		})
	}
}

func validPhysicalBlenderProfile() *cliv1.BlenderToolProfile {
	return &cliv1.BlenderToolProfile{
		SchemaVersion: 1, ProfileId: PhysicalBlenderProfileID,
		State:            cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_ARTIFACT_PRESENT,
		ArtifactIdentity: PhysicalBlenderArtifact, ExpectedTreeSha256: PhysicalBlenderTreeSHA,
		Os: "linux", Architecture: "amd64",
		Blender: &cliv1.RendererToolBinary{
			Path:   filepath.Join("/home/test/.vrooli/tools/blender", PhysicalBlenderArchive, "blender"),
			Sha256: PhysicalBlenderBinarySHA, SizeBytes: 300_000_000,
			Version: PhysicalBlenderVersion, ArchiveMemberPath: PhysicalBlenderArchive + "/blender",
		},
	}
}

func TestPhysicalBlenderCLILookupUsesOSHomeRuntimeEntryWithoutPATH(t *testing.T) {
	home := t.TempDir()
	binDir, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(binDir, "vrooli")
	if err := os.WriteFile(want, []byte("fixture"), 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := resolvePhysicalBlenderCLI(func(string) string { return "" }, home)
	if err != nil {
		t.Fatalf("resolve CLI from contract runtime root with PATH disabled: %v", err)
	}
	if got != want {
		t.Fatalf("resolved CLI = %q, want OS-home runtime entry binary %q", got, want)
	}
}

func TestPhysicalBlenderCLILookupAcceptsExplicitBinaryOutsideRepository(t *testing.T) {
	workspace := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(workspace); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("VROOLI_SOURCE_ROOT", "")
	bin := filepath.Join(workspace, "tools", "vrooli")
	if err := os.MkdirAll(filepath.Dir(bin), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("fixture"), 0o700); err != nil {
		t.Fatal(err)
	}
	getenv := func(key string) string {
		if key == cliinvoke.BinaryEnvVar {
			return bin
		}
		return ""
	}
	got, err := resolvePhysicalBlenderCLI(getenv, filepath.Join(workspace, "non-repo-home"))
	if err != nil {
		t.Fatalf("resolve explicit CLI without repository context or PATH: %v", err)
	}
	if got != bin {
		t.Fatalf("resolved CLI = %q, want explicit binary %q", got, bin)
	}
}
