package templateengine

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	scenariocli "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

func mustLoadRepoTemplate(t *testing.T, name string) (string, scenariocli.TemplateInfo) {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	repoRoot, err := repocontract.FindRepoRootFromPath(thisFile)
	if err != nil {
		t.Fatalf("FindRepoRootFromPath: %v", err)
	}
	info, err := loadTemplate(repoRoot, name)
	if err != nil {
		t.Fatalf("loadTemplate(%s): %v", name, err)
	}
	return repoRoot, info
}

// copyTemplateDirToTemp copies a real template into a temp directory so the
// test can mutate files without touching the canonical tree.
func copyTemplateDirToTemp(t *testing.T, src string) string {
	t.Helper()
	dest := t.TempDir()
	err := filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if fi.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, fi.Mode())
	})
	if err != nil {
		t.Fatalf("copy template tree: %v", err)
	}
	return dest
}

func TestComputeTemplateHashesDeterministic(t *testing.T) {
	_, info := mustLoadRepoTemplate(t, "react-vite")
	m1, c1, err := computeTemplateHashes(info)
	if err != nil {
		t.Fatalf("computeTemplateHashes #1: %v", err)
	}
	m2, c2, err := computeTemplateHashes(info)
	if err != nil {
		t.Fatalf("computeTemplateHashes #2: %v", err)
	}
	if m1 == "" || c1 == "" {
		t.Fatalf("expected non-empty hashes, got manifest=%q content=%q", m1, c1)
	}
	if m1 != m2 || c1 != c2 {
		t.Fatalf("hashes not deterministic: manifest %q vs %q, content %q vs %q", m1, m2, c1, c2)
	}
}

func TestComputeTemplateHashesContentFlipsOnEmittedFileChange(t *testing.T) {
	_, info := mustLoadRepoTemplate(t, "react-vite")
	tmpDir := copyTemplateDirToTemp(t, info.Path)
	tmpInfo := info
	tmpInfo.Path = tmpDir

	m1, c1, err := computeTemplateHashes(tmpInfo)
	if err != nil {
		t.Fatalf("computeTemplateHashes initial: %v", err)
	}

	// Append a marker to an inherited file. We pick something stable across
	// templates: every scenario template ships a top-level README.md or
	// api/go.mod. Try README.md first, fall back to any non-skipped file.
	target := filepath.Join(tmpDir, "README.md")
	if _, err := os.Stat(target); os.IsNotExist(err) {
		// Find any emitted file by walking.
		err := filepath.Walk(tmpDir, func(path string, fi os.FileInfo, err error) error {
			if err != nil || fi.IsDir() || target != filepath.Join(tmpDir, "README.md") {
				if target != filepath.Join(tmpDir, "README.md") {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Base(path) == "template.json" {
				return nil
			}
			target = path
			return filepath.SkipDir
		})
		if err != nil {
			t.Fatalf("find target: %v", err)
		}
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target %s: %v", target, err)
	}
	if err := os.WriteFile(target, append(data, []byte("\n// drift probe\n")...), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	m2, c2, err := computeTemplateHashes(tmpInfo)
	if err != nil {
		t.Fatalf("computeTemplateHashes post-edit: %v", err)
	}
	if m2 != m1 {
		t.Fatalf("manifest sha should not change when only content changes: was %q now %q", m1, m2)
	}
	if c2 == c1 {
		t.Fatalf("content sha did not change after editing %s", target)
	}
}

func TestComputeTemplateHashesManifestFlipsOnManifestEdit(t *testing.T) {
	_, info := mustLoadRepoTemplate(t, "react-vite")
	tmpDir := copyTemplateDirToTemp(t, info.Path)
	tmpInfo := info
	tmpInfo.Path = tmpDir

	m1, c1, err := computeTemplateHashes(tmpInfo)
	if err != nil {
		t.Fatalf("initial: %v", err)
	}

	manifestPath := filepath.Join(tmpDir, "template.json")
	original, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	// Append a trailing space-only edit inside a comment-safe location: just
	// append a newline. Hash is over raw bytes, so any byte change flips it.
	if err := os.WriteFile(manifestPath, append(original, '\n'), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	m2, c2, err := computeTemplateHashes(tmpInfo)
	if err != nil {
		t.Fatalf("post-edit: %v", err)
	}
	if m2 == m1 {
		t.Fatalf("manifest sha unchanged after editing template.json")
	}
	if c2 != c1 {
		t.Fatalf("content sha should not change when only template.json bytes change: was %q now %q", c1, c2)
	}
}

func TestHashTemplateContentRespectsCopyExcludes(t *testing.T) {
	_, info := mustLoadRepoTemplate(t, "react-vite")
	tmpDir := copyTemplateDirToTemp(t, info.Path)

	// Create an extra file at the root of the template tree, then add it to
	// CopyExcludes — the hash should be unchanged because excluded files are
	// never emitted to a scenario.
	noisePath := filepath.Join(tmpDir, "DRIFT_TEST_NOISE.txt")
	if err := os.WriteFile(noisePath, []byte("noise\n"), 0o644); err != nil {
		t.Fatalf("write noise: %v", err)
	}

	manifestWithoutExclude := info.Manifest
	manifestWithExclude := info.Manifest
	manifestWithExclude.CopyExcludes = append([]string{}, info.Manifest.CopyExcludes...)
	manifestWithExclude.CopyExcludes = append(manifestWithExclude.CopyExcludes, "DRIFT_TEST_NOISE.txt")

	cWith, err := hashTemplateContent(tmpDir, manifestWithoutExclude)
	if err != nil {
		t.Fatalf("hash without exclude: %v", err)
	}
	cExcluded, err := hashTemplateContent(tmpDir, manifestWithExclude)
	if err != nil {
		t.Fatalf("hash with exclude: %v", err)
	}
	if cWith == cExcluded {
		t.Fatalf("expected hashes to differ when a file is and isn't excluded — exclude path was ignored")
	}

	// And removing the noise file should make the unexcluded hash match the
	// excluded one, proving the exclude has the same effect as deletion from
	// the emission perspective.
	if err := os.Remove(noisePath); err != nil {
		t.Fatalf("remove noise: %v", err)
	}
	cAfter, err := hashTemplateContent(tmpDir, manifestWithoutExclude)
	if err != nil {
		t.Fatalf("hash after removal: %v", err)
	}
	if cAfter != cExcluded {
		t.Fatalf("after removing the noise file, hash differs from excluded-hash: %q vs %q", cAfter, cExcluded)
	}
}

func TestHashTemplateContentSkipsTemplateJsonAndHardSkippedDirs(t *testing.T) {
	_, info := mustLoadRepoTemplate(t, "react-vite")
	tmpDir := copyTemplateDirToTemp(t, info.Path)

	c1, err := hashTemplateContent(tmpDir, info.Manifest)
	if err != nil {
		t.Fatalf("initial: %v", err)
	}

	// Editing template.json bytes must not affect content_sha.
	manifestPath := filepath.Join(tmpDir, "template.json")
	if data, err := os.ReadFile(manifestPath); err == nil {
		if err := os.WriteFile(manifestPath, append(data, '\n'), 0o644); err != nil {
			t.Fatalf("write manifest: %v", err)
		}
	}

	// Creating files inside a hard-skipped directory must not affect content_sha.
	skipped := filepath.Join(tmpDir, "node_modules")
	if err := os.MkdirAll(skipped, 0o755); err != nil {
		t.Fatalf("mkdir node_modules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skipped, "junk.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write junk: %v", err)
	}

	c2, err := hashTemplateContent(tmpDir, info.Manifest)
	if err != nil {
		t.Fatalf("after edits: %v", err)
	}
	if c2 != c1 {
		t.Fatalf("content_sha changed after edits to template.json + node_modules — skip rules drifted")
	}
}
