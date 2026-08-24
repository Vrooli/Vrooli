package store

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	got, err := fs.ImportSkill(ImportRequest{SourceDir: source, SourceURL: "https://example.test/repo", Commit: "4f2c1ab", License: "Apache-2.0", Checksum: checksum, ImportedBy: "alice", ID: "imported-demo"})
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
	if err := fs.ReviewImportedSkill("imported-demo", "alice", ReviewVerdictPassed); err == nil {
		t.Fatal("importer must not self-approve")
	}
	if err := fs.ReviewImportedSkill("imported-demo", "bob", ReviewVerdictPassed); err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Get(t.Context(), "imported-demo"); err != nil {
		t.Fatalf("passed vendor skill should be active: %v", err)
	}
	if err := fs.Update(t.Context(), "imported-demo", &Skill{}, func() *string { v := "edited"; return &v }()); err == nil || !strings.Contains(err.Error(), "overlay") {
		t.Fatalf("in-place vendor edit should name overlay path, got %v", err)
	}
	path, err := fs.WriteImportedSkillOverlay("imported-demo", "local.patch", "patch")
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
	_, err := fs.ImportSkill(ImportRequest{SourceDir: filepath.Join(root, "source"), SourceURL: "url", Commit: "abcdef1", License: "MIT", Checksum: "sha256:" + strings.Repeat("0", 64), ImportedBy: "alice", ID: "demo"})
	if err == nil {
		t.Fatal("expected frontmatter rejection")
	}
}
