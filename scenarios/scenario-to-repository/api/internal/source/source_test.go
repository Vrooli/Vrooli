package source

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnalyzeClosureExcludesPrivateAndHistoryFiles(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".git", "objects"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".git", "objects", "secret"), []byte("history"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".env"), []byte("TOKEN=private"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "README.md"), []byte("public"), 0o644))
	closure, err := AnalyzeClosure(root, "demo", "sha256:source")
	require.NoError(t, err)
	require.Len(t, closure.Files, 1)
	require.Equal(t, "README.md", closure.Files[0].ExportPath)
	require.NotEmpty(t, closure.ClosureDigest)
}

func TestAnalyzeClosureRefusesEscapingSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "private"), []byte("no"), 0o600))
	require.NoError(t, os.Symlink(filepath.Join(outside, "private"), filepath.Join(root, "link")))
	_, err := AnalyzeClosure(root, "demo", "sha256:source")
	require.ErrorContains(t, err, "escapes source root")
}

func TestAssembleIsByteDeterministic(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "b.txt"), []byte("b"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.txt"), []byte("a"), 0o755))
	closure, err := AnalyzeClosure(root, "demo", "sha256:source")
	require.NoError(t, err)
	recipe := Recipe{SchemaVersion: 1, Scenario: "demo", Mode: "buildable_source", ArchiveFormat: "tar_gzip", Deterministic: true}
	a, err := Assemble(root, filepath.Join(t.TempDir(), "one.tar.gz"), closure, recipe)
	require.NoError(t, err)
	b, err := Assemble(root, filepath.Join(t.TempDir(), "two.tar.gz"), closure, recipe)
	require.NoError(t, err)
	require.Equal(t, a.ArchiveDigest, b.ArchiveDigest)
	require.Equal(t, a.ManifestDigest, b.ManifestDigest)
}

func TestPublishabilityRefusesSecretLikeGeneratedContent(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "README.md"), []byte("api_key = '"+strings.Repeat("1", 16)+"'"), 0o644))
	closure, err := AnalyzeClosure(root, "demo", "sha256:source")
	require.NoError(t, err)
	decision, err := CheckPublishability(root, closure)
	require.NoError(t, err)
	require.False(t, decision.Allowed)
	require.Contains(t, decision.Violations[0], "secret-like")
}

func TestPublishPolicyRejectsUnlicensedBinaryAssetAndUnboundedLimits(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "assets"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "assets", "logo.png"), []byte("image"), 0o644))
	closure, err := AnalyzeClosure(root, "demo", "sha256:source")
	require.NoError(t, err)
	decision, err := CheckPublishabilityWithPolicy(root, closure, PublishPolicy{Version: 1, Limits: ScanLimits{MaxFiles: 1, MaxBytes: 2}, RequireAssetLicense: true})
	require.NoError(t, err)
	require.False(t, decision.Allowed)
	require.Contains(t, decision.Violations, "unknown asset license: assets/logo.png")
	require.Contains(t, decision.Violations, "source bytes exceed scan budget")
}

func TestVerifyArchiveDetectsChangedArtifact(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "README.md"), []byte("public"), 0o644))
	closure, err := AnalyzeClosure(root, "demo", "sha256:source")
	require.NoError(t, err)
	artifact, err := Assemble(root, filepath.Join(t.TempDir(), "one.tar.gz"), closure, Recipe{SchemaVersion: 1, Scenario: "demo", ArchiveFormat: "tar_gzip", Deterministic: true})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(artifact.ArchivePath, []byte("changed"), 0o644))
	verification, err := VerifyArchive(artifact)
	require.NoError(t, err)
	require.Equal(t, "failed", verification.Status)
	require.Contains(t, verification.Failures, "archive digest mismatch")
}

func TestParseRecipeRequiresVersionedDeterministicMode(t *testing.T) {
	recipe, digest, err := ParseRecipe([]byte("schema_version: 1\nscenario: demo\nmode: buildable_source\narchive: tar_gzip\ndeterministic: true\n"))
	require.NoError(t, err)
	require.Equal(t, "demo", recipe.Scenario)
	require.True(t, len(digest) > len("sha256:"))
	_, _, err = ParseRecipe([]byte("schema_version: 1\nscenario: demo\nmode: mirror\narchive: tar_gzip\ndeterministic: true\n"))
	require.ErrorContains(t, err, "unsupported recipe mode")
}

func TestCaptureSnapshotHasStableSourceDigest(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0o644))
	snapshot, err := CaptureSnapshot(root, "demo")
	require.NoError(t, err)
	require.Equal(t, "demo", snapshot.Scenario)
	require.NotEmpty(t, snapshot.SourceDigest)
	require.NotEmpty(t, snapshot.Closure.ClosureDigest)
}

func TestVerifyCleanInputsRequiresDeclaredIndependentInputs(t *testing.T) {
	closure := Closure{Files: []File{{ExportPath: "main.go"}}, Nodes: []Node{{ID: "postgres", Kind: "runtime_requirement"}}}
	report, err := VerifyCleanInputs(t.TempDir(), closure)
	require.NoError(t, err)
	require.Equal(t, "passed", report.Status)
	require.Contains(t, report.GatesPassed, "no-git-mutation")
	require.Equal(t, []string{"postgres"}, report.ExternalRequirements)
}

func TestVerifyCleanInputsRejectsPrivateEnvironmentInputs(t *testing.T) {
	report, err := VerifyCleanInputs(t.TempDir(), Closure{Files: []File{{ExportPath: ".git/config"}}})
	require.NoError(t, err)
	require.Equal(t, "failed", report.Status)
	require.Contains(t, report.Failures[0], "private environment input")
}

func TestCompareUpdateSeparatesSourceAndDestinationDrift(t *testing.T) {
	preview := CompareUpdate("source-new", "source-old", "recipe-new", "recipe-old", "policy-v1", "policy-v1", "archive-new", "archive-old", "archive-old", "manifest-new")
	require.True(t, preview.SourceChanged)
	require.True(t, preview.RecipeChanged)
	require.True(t, preview.DestinationDrift)
	require.Equal(t, "review_required", preview.Disposition)
	require.Contains(t, preview.Actions, "require a new human publication decision")
}
