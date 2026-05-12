package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// docs_structure_test.go enforces the web-console docs tree invariants set by
// the react-vite template cutover:
//
//   1. manifestResolves       — every manifest path exists on disk.
//   2. noOrphanDocs           — every .md under docs/ is manifest-listed or
//                               whitelisted (PROGRESS/QUICKSTART/internal/plans).
//   3. codeDocRefsResolve     — every `// DOC: docs/...` marker in api/ ui/ cli/
//                               points at a real file.
//   4. noStaleOldPaths        — the old pre-cutover paths no longer appear.

const scenarioRoot = "../" // tests run from the api/ directory

type manifestDoc struct {
	Path string `json:"path"`
}

type manifestSection struct {
	Documents []manifestDoc `json:"documents"`
}

type docsManifest struct {
	Sections []manifestSection `json:"sections"`
}

func loadManifest(t *testing.T) docsManifest {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(scenarioRoot, "docs", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var m docsManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	return m
}

func TestDocsManifestResolves(t *testing.T) {
	m := loadManifest(t)
	docsDir := filepath.Join(scenarioRoot, "docs")
	for _, sec := range m.Sections {
		for _, doc := range sec.Documents {
			full := filepath.Join(docsDir, doc.Path)
			if _, err := os.Stat(full); err != nil {
				t.Errorf("manifest path does not resolve: %s (%v)", doc.Path, err)
			}
		}
	}
}

func TestDocsNoOrphans(t *testing.T) {
	m := loadManifest(t)
	listed := map[string]bool{}
	for _, sec := range m.Sections {
		for _, doc := range sec.Documents {
			listed[doc.Path] = true
		}
	}
	whitelist := func(rel string) bool {
		switch rel {
		case "QUICKSTART.md", "PROGRESS.md":
			return true
		}
		return strings.HasPrefix(rel, "internal/plans/")
	}
	docsDir := filepath.Join(scenarioRoot, "docs")
	err := filepath.WalkDir(docsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		rel, _ := filepath.Rel(docsDir, path)
		if listed[rel] || whitelist(rel) {
			return nil
		}
		t.Errorf("orphan doc not in manifest and not whitelisted: %s", rel)
		return nil
	})
	if err != nil {
		t.Fatalf("walk docs: %v", err)
	}
}

var docMarkerRe = regexp.MustCompile(`docs/[A-Za-z0-9_/-]+\.md`)
var commentLineRe = regexp.MustCompile(`(?m)^\s*(?://|/\*|\*|#)?\s*(?://\s*)?DOC:\s*(docs/[A-Za-z0-9_/.#-]+)`)

func TestCodeDocRefsResolve(t *testing.T) {
	roots := []string{
		filepath.Join(scenarioRoot, "api"),
		filepath.Join(scenarioRoot, "ui", "src"),
		filepath.Join(scenarioRoot, "cli"),
	}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				name := d.Name()
				if name == "node_modules" || name == "dist" || name == "coverage" {
					return filepath.SkipDir
				}
				return nil
			}
			ext := filepath.Ext(path)
			switch ext {
			case ".go", ".ts", ".tsx", ".js", ".jsx":
			default:
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			for _, m := range commentLineRe.FindAllSubmatch(b, -1) {
				raw := string(m[1])
				// Strip anchor (#section) before resolving on disk.
				docPath := raw
				if i := strings.Index(docPath, "#"); i >= 0 {
					docPath = docPath[:i]
				}
				full := filepath.Join(scenarioRoot, docPath)
				if _, err := os.Stat(full); err != nil {
					t.Errorf("// DOC: marker does not resolve: %s in %s", raw, path)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
}

func TestDocsNoStaleOldPaths(t *testing.T) {
	stale := []*regexp.Regexp{
		regexp.MustCompile(`ERROR-SEMANTICS\.md`),
		regexp.MustCompile(`internal/PROGRESS\.md`),
		// "docs/plans/" without the "internal/" prefix is the stale form.
		regexp.MustCompile(`(^|[^l/])docs/plans/`),
	}
	roots := []string{
		filepath.Join(scenarioRoot, "api"),
		filepath.Join(scenarioRoot, "ui", "src"),
		filepath.Join(scenarioRoot, "cli"),
		filepath.Join(scenarioRoot, "docs"),
	}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				name := d.Name()
				if name == "node_modules" || name == "dist" || name == "coverage" {
					return filepath.SkipDir
				}
				return nil
			}
			// Skip this very test file (it names the stale paths intentionally).
			if filepath.Base(path) == "docs_structure_test.go" {
				return nil
			}
			ext := filepath.Ext(path)
			switch ext {
			case ".go", ".ts", ".tsx", ".js", ".jsx", ".md", ".json":
			default:
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			for _, re := range stale {
				if re.Match(b) {
					t.Errorf("stale pre-cutover path %s found in %s", re.String(), path)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	_ = docMarkerRe // referenced for future expansion
}
