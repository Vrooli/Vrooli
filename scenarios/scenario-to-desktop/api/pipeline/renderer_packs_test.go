package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestRendererToolProfilesAreFiniteAndCanonical(t *testing.T) {
	got, err := normalizeRendererToolProfiles([]string{"ffmpeg-video", "blender"})
	if err != nil || strings.Join(got, ",") != "blender,ffmpeg-video" {
		t.Fatalf("profiles = %v, %v", got, err)
	}
	for _, input := range [][]string{{"other"}, {"blender", "blender"}} {
		if _, err := normalizeRendererToolProfiles(input); err == nil {
			t.Fatalf("accepted invalid selection %v", input)
		}
	}
}

func TestResolveRendererPacksBindsVerifiedReleaseMetadataAndOwnerTree(t *testing.T) {
	repo, release := t.TempDir(), t.TempDir()
	artifact := filepath.Join(release, "tool_fixture_linux_amd64")
	if err := os.Mkdir(artifact, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifact, "bin"), []byte("tool"), 0o755); err != nil {
		t.Fatal(err)
	}
	tree, err := binaryfetch.TreeDigest(artifact)
	if err != nil {
		t.Fatal(err)
	}
	pin := rendererPackPin{profileID: "fixture-profile-v1", treeSHA256: tree, role: "renderer-profile:blender", registryName: "fixture"}
	manifest := resourcedeployment.ReleaseManifest{SchemaVersion: "v1", Artifacts: []resourcedeployment.ReleaseArtifact{{Name: "tool_fixture_linux_amd64", SHA256: tree, Role: pin.role, OS: "linux", Arch: "amd64", UpstreamProvenance: "fixture"}}}
	data, err := manifest.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(release, "release-manifest.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	lookup := func(string) (rendererPackPin, bool) { return pin, true }
	got, err := resolveRendererPacksWithPins(repo, release, []resourcedeployment.Platform{{OS: "linux", Arch: "amd64"}}, resourcedeployment.ArtifactTrustDevelopmentLocal, []string{"blender"}, lookup)
	if err != nil || len(got) != 1 || got[0].TreeSHA256 != tree || got[0].Artifact != "tool_fixture_linux_amd64" {
		t.Fatalf("resolved renderer pack = %+v, %v", got, err)
	}
	manifest.Artifacts[0].Role = "managed-service"
	data, _ = manifest.CanonicalBytes()
	if err := os.WriteFile(filepath.Join(release, "release-manifest.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveRendererPacksWithPins(repo, release, []resourcedeployment.Platform{{OS: "linux", Arch: "amd64"}}, resourcedeployment.ArtifactTrustDevelopmentLocal, []string{"blender"}, lookup); err == nil {
		t.Fatal("release artifact with wrong role was accepted")
	}
}

func TestStageRendererDirectoryVerifiesAndNeverReplaces(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "tool"), []byte("pinned"), 0o755); err != nil {
		t.Fatal(err)
	}
	digest, err := binaryfetch.TreeDigest(source)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(root, "bundle", "tools", "blender")
	if err := stageRendererDirectory(source, destination, digest); err != nil {
		t.Fatal(err)
	}
	if got, err := binaryfetch.TreeDigest(destination); err != nil || got != digest {
		t.Fatalf("staged tree digest = %q, %v; want %q", got, err, digest)
	}
	if err := os.WriteFile(filepath.Join(destination, "tool"), []byte("keep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := stageRendererDirectory(source, destination, digest); err == nil {
		t.Fatal("existing destination was replaced")
	}
	got, err := os.ReadFile(filepath.Join(destination, "tool"))
	if err != nil || string(got) != "keep" {
		t.Fatalf("existing bytes = %q, %v", got, err)
	}
}

func TestStageRendererDirectoryRejectsSymlinkDestinationParentAndChangedSource(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "tool"), []byte("pinned"), 0o755); err != nil {
		t.Fatal(err)
	}
	digest, _ := binaryfetch.TreeDigest(source)
	external := filepath.Join(root, "external")
	if err := os.Mkdir(external, 0o755); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Join(root, "bundle", "tools")
	if err := os.MkdirAll(filepath.Dir(parent), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, parent); err != nil {
		t.Fatal(err)
	}
	if err := stageRendererDirectory(source, filepath.Join(parent, "blender"), digest); err == nil {
		t.Fatal("symlink destination parent accepted")
	}
	if err := os.WriteFile(filepath.Join(source, "tool"), []byte("changed"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := stageRendererDirectory(source, filepath.Join(root, "elsewhere", "blender"), digest); err == nil {
		t.Fatal("changed source tree accepted")
	}
}

func TestAddRendererDeclarationsPreservesManifestAndRejectsConflicts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":"v1","app":{"name":"demo"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	pack := RendererPackPlanItem{ToolID: "blender", ProfileID: "pinned", OS: "linux", Architecture: "amd64", TreeSHA256: strings.Repeat("a", 64)}
	if err := addRendererDeclarations(path, []RendererPackPlanItem{pack}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["app"]; !ok || len(doc["tool_profiles"]) == 0 {
		t.Fatalf("manifest fields were not preserved: %s", data)
	}
	pack.ProfileID = "conflicting"
	if err := addRendererDeclarations(path, []RendererPackPlanItem{pack}); err == nil {
		t.Fatal("conflicting renderer declaration accepted")
	}
}

func TestAddRendererDeclarationsRejectsMalformedUnselectedEntriesWithoutChangingManifest(t *testing.T) {
	for _, existing := range []string{
		`[{"tool_id":"ffmpeg-video","profile_id":"pinned","os":"linux","architecture":"amd64"}]`,
		`[{"tool_id":"ffmpeg-video","profile_id":"pinned","os":"","architecture":"amd64","tree_sha256":"` + strings.Repeat("a", 64) + `"}]`,
	} {
		t.Run(existing, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "bundle.json")
			before := []byte(`{"schema_version":"v1","tool_profiles":` + existing + `}`)
			if err := os.WriteFile(path, before, 0o644); err != nil {
				t.Fatal(err)
			}
			pack := RendererPackPlanItem{ToolID: "blender", ProfileID: "pinned", OS: "linux", Architecture: "amd64", TreeSHA256: strings.Repeat("b", 64)}
			if err := addRendererDeclarations(path, []RendererPackPlanItem{pack}); err == nil {
				t.Fatal("malformed unselected tool declaration was accepted")
			}
			after, err := os.ReadFile(path)
			if err != nil || string(after) != string(before) {
				t.Fatalf("failed validation changed packaged manifest: %q, %v", after, err)
			}
		})
	}
}

func TestStageSelectedRendererPacksRechecksReleaseMetadataBeforeMutation(t *testing.T) {
	root := t.TempDir()
	release := filepath.Join(root, "release")
	artifactName := "tool_fixture_linux_amd64"
	artifactPath := filepath.Join(release, artifactName)
	if err := os.MkdirAll(artifactPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactPath, "bin"), []byte("pinned"), 0o755); err != nil {
		t.Fatal(err)
	}
	tree, err := binaryfetch.TreeDigest(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	manifest := resourcedeployment.ReleaseManifest{SchemaVersion: "v1", Artifacts: []resourcedeployment.ReleaseArtifact{{Name: artifactName, SHA256: tree, Role: "changed-role", OS: "linux", Arch: "amd64", UpstreamProvenance: "fixture"}}}
	manifestBytes, err := manifest.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(release, "release-manifest.json"), manifestBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(root, "bundle")
	manifestPath := filepath.Join(bundle, "bundle.json")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatal(err)
	}
	before := []byte(`{"schema_version":"v1","app":{"name":"demo"}}`)
	if err := os.WriteFile(manifestPath, before, 0o644); err != nil {
		t.Fatal(err)
	}
	plan := &ResourceDeploymentPlan{
		ArtifactTrustMode: resourcedeployment.ArtifactTrustDevelopmentLocal,
		RendererPacks:     []RendererPackPlanItem{{ToolID: "blender", ProfileID: "pinned", Artifact: artifactName, Role: "renderer-profile:blender", OS: "linux", Architecture: "amd64", TreeSHA256: tree}},
	}
	pinFor := func(string) (rendererPackPin, bool) {
		return rendererPackPin{profileID: "pinned", treeSHA256: tree, role: "renderer-profile:blender", registryName: "fixture"}, true
	}
	if _, err := stageSelectedRendererPacksWithPins(bundle, manifestPath, release, "", plan, pinFor); err == nil {
		t.Fatal("changed release metadata was accepted")
	}
	if _, err := os.Lstat(filepath.Join(bundle, "tools")); !os.IsNotExist(err) {
		t.Fatalf("staging mutated bundle before rejecting release metadata: %v", err)
	}
	after, err := os.ReadFile(manifestPath)
	if err != nil || string(after) != string(before) {
		t.Fatalf("failed staging changed packaged manifest: %q, %v", after, err)
	}
}

func TestResolveRendererPacksRejectsUnsupportedTargetBeforeReleaseRead(t *testing.T) {
	_, err := resolveRendererPacks(t.TempDir(), t.TempDir(), []resourcedeployment.Platform{{OS: "windows", Arch: "amd64"}}, resourcedeployment.ArtifactTrustDevelopmentLocal, []string{"blender"})
	if err == nil || !strings.Contains(err.Error(), "supports linux-amd64 only") {
		t.Fatalf("unsupported target error = %v", err)
	}
}
