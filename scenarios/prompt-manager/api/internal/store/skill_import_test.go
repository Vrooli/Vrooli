package store

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

func TestImportSkillQuarantinesAndRequiresIndependentReview(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: imported-demo\ndescription: imported demo\n---\n\nUse safely.\n"
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(content))
	checksum := "sha256:" + hex.EncodeToString(digest[:])
	config := filepath.Join(root, "config")
	if err := os.MkdirAll(filepath.Join(config, "skills", "packs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SaveJSON(filepath.Join(config, "skills", "_pack-order.json"), &PackOrder{ActivePacks: []string{"local", "core"}, InactivePacks: []string{"drafts"}}); err != nil {
		t.Fatal(err)
	}
	fs := NewFileSkillStore(config)
	got, err := fs.ImportSkill(t.Context(), ImportRequest{SourceDir: source, SourceURL: "https://example.test/repo", Commit: "4f2c1ab", License: "Apache-2.0", Checksum: checksum, ImportedBy: "alice", ID: "imported-demo"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Origin == nil || got.Origin.Review.Verdict != ReviewVerdictPending {
		t.Fatalf("unexpected origin: %#v", got.Origin)
	}
	imported, err := os.ReadFile(filepath.Join(config, "skills", "packs", "vendor", "imported-demo", "SKILL.md"))
	if err != nil || !strings.Contains(string(imported), "source_url:") || !strings.Contains(string(imported), "verdict: pending") {
		t.Fatalf("imported origin block missing: %s %v", imported, err)
	}
	if _, err := fs.Get(t.Context(), "imported-demo"); err == nil {
		t.Fatal("pending vendor skill must not be discoverable")
	} else if !strings.Contains(err.Error(), "review verdict is pending") {
		t.Fatalf("pending refusal must name verdict: %v", err)
	}
	if err := fs.ReviewImportedSkill(t.Context(), "imported-demo", "alice", ReviewVerdictPassed); err == nil {
		t.Fatal("importer must not self-approve")
	}
	if err := fs.ReviewImportedSkill(t.Context(), "imported-demo", "bob", ReviewVerdictPassed); err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Get(t.Context(), "imported-demo"); err != nil {
		t.Fatalf("passed vendor skill should be active: %v", err)
	}
	if err := fs.Update(t.Context(), "imported-demo", &Skill{}, func() *string { v := "edited"; return &v }()); err == nil || !strings.Contains(err.Error(), "overlay") {
		t.Fatalf("in-place vendor edit should name overlay path, got %v", err)
	}
	path, err := fs.WriteImportedSkillOverlay(t.Context(), "imported-demo", "local.patch", "patch")
	if err != nil || !strings.HasSuffix(path, filepath.Join("overlays", "local.patch")) {
		t.Fatalf("write overlay: path=%s err=%v", path, err)
	}
}

func TestImportSkillRejectsChecksumAndInvalidFrontmatter(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "source"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "source", "SKILL.md"), []byte("plain body"), 0o600); err != nil {
		t.Fatal(err)
	}
	fs := NewFileSkillStore(filepath.Join(root, "config"))
	_, err := fs.ImportSkill(t.Context(), ImportRequest{SourceDir: filepath.Join(root, "source"), SourceURL: "url", Commit: "abcdef1", License: "MIT", Checksum: "sha256:" + strings.Repeat("0", 64), ImportedBy: "alice", ID: "demo"})
	if err == nil {
		t.Fatal("expected frontmatter rejection")
	}
}

func newImportFixture(t *testing.T) (string, string, string, *FileSkillStore) {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.MkdirAll(filepath.Join(source, "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: third-party-demo\ndescription: third-party demo\n---\n\nRead references/step.md.\n"
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "references", "step.md"), []byte("step one"), 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(content))
	config := filepath.Join(root, "config")
	if err := os.MkdirAll(filepath.Join(config, "skills", "packs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SaveJSON(filepath.Join(config, "skills", "_pack-order.json"), &PackOrder{ActivePacks: []string{"local", "core"}}); err != nil {
		t.Fatal(err)
	}
	return source, config, "sha256:" + hex.EncodeToString(digest[:]), NewFileSkillStore(config)
}

func TestImportSkillCopiesSupportingFilesAndRecordsThirdPartyFacts(t *testing.T) {
	source, config, checksum, fs := newImportFixture(t)
	tools := []ExternalTool{{Name: "hyperframes", URL: "https://hyperframes.heygen.com/", Purpose: "renders video"}}
	got, err := fs.ImportSkill(t.Context(), ImportRequest{SourceDir: source, SourceURL: "https://github.com/example/demo", Commit: "1f8d9ade", License: "MIT", Checksum: checksum, ImportedBy: "alice", ID: "third-party-demo", ExternalTools: tools})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got.Origin.TreeChecksum, "sha256:") || got.Origin.TreeChecksum == got.Origin.Checksum {
		t.Fatalf("tree checksum must cover the whole tree: %#v", got.Origin)
	}
	skillDir := filepath.Join(config, "skills", "packs", "vendor", "third-party-demo")
	if data, err := os.ReadFile(filepath.Join(skillDir, "references", "step.md")); err != nil || string(data) != "step one" {
		t.Fatalf("supporting file not copied: %q %v", data, err)
	}
	persisted, err := LoadJSON[Skill](filepath.Join(skillDir, "skill.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(persisted.ExternalTools) != 1 || persisted.ExternalTools[0].Name != "hyperframes" {
		t.Fatalf("external tools not persisted: %#v", persisted.ExternalTools)
	}
	if persisted.Origin == nil || persisted.Origin.TreeChecksum != got.Origin.TreeChecksum {
		t.Fatalf("tree checksum not persisted: %#v", persisted.Origin)
	}
}

func TestImportSkillRefusesSymlinksAndNestedManifests(t *testing.T) {
	source, config, checksum, fs := newImportFixture(t)
	if err := os.WriteFile(filepath.Join(source, "references", "SKILL.md"), []byte("nested"), 0o600); err != nil {
		t.Fatal(err)
	}
	req := ImportRequest{SourceDir: source, SourceURL: "https://github.com/example/demo", Commit: "1f8d9ade", License: "MIT", Checksum: checksum, ImportedBy: "alice", ID: "third-party-demo"}
	if _, err := fs.ImportSkill(t.Context(), req); err == nil || !strings.Contains(err.Error(), "nested skill manifest") {
		t.Fatalf("nested manifest must be refused, got %v", err)
	}
	if err := os.Remove(filepath.Join(source, "references", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/etc/hostname", filepath.Join(source, "references", "escape.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := fs.ImportSkill(t.Context(), req); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlink must be refused, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(config, "skills", "packs", "vendor", "third-party-demo")); !os.IsNotExist(err) {
		t.Fatalf("refused import must not leave a vendor directory: %v", err)
	}
}

func TestRoutedSkillStoreImportsIntoResolvedConfigRoot(t *testing.T) {
	source, config, checksum, _ := newImportFixture(t)
	roots := filerouting.New(storage.Paths{ConfigDir: config})
	fs := NewRoutedFileSkillStore(roots)
	ctx := t.Context()
	req := ImportRequest{SourceDir: source, SourceURL: "https://github.com/example/demo", Commit: "1f8d9ade", License: "MIT", Checksum: checksum, ImportedBy: "alice", ID: "third-party-demo"}
	if _, err := fs.ImportSkill(ctx, req); err != nil {
		t.Fatalf("routed import must resolve the config root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(config, "skills", "packs", "vendor", "third-party-demo", "skill.json")); err != nil {
		t.Fatalf("routed import did not land in the resolved root: %v", err)
	}
	if err := fs.ReviewImportedSkill(ctx, "third-party-demo", "bob", ReviewVerdictPassed); err != nil {
		t.Fatalf("routed review: %v", err)
	}
	order, err := LoadJSON[PackOrder](filepath.Join(config, "skills", "_pack-order.json"))
	if err != nil {
		t.Fatal(err)
	}
	if n := len(order.ActivePacks); n == 0 || order.ActivePacks[n-1] != "vendor" {
		t.Fatalf("vendor pack must have the lowest precedence, got %v", order.ActivePacks)
	}
	reviewed, err := os.ReadFile(filepath.Join(config, "skills", "packs", "vendor", "third-party-demo", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(reviewed), "verdict: pending") || !strings.Contains(string(reviewed), "verdict: passed") || !strings.Contains(string(reviewed), `reviewer: "bob"`) || !strings.Contains(string(reviewed), "Read references/step.md.") {
		t.Fatalf("review must be recorded in the origin block without touching the body:\n%s", reviewed)
	}
	if _, _, err := fs.ImportedSkillStaleness(ctx, "third-party-demo", "0.2.2"); err != nil {
		t.Fatalf("routed staleness: %v", err)
	}
	if skill, err := fs.Get(ctx, "third-party-demo"); err != nil || skill.Origin == nil {
		t.Fatalf("reviewed third-party skill should be readable with origin: %#v %v", skill, err)
	}
}
