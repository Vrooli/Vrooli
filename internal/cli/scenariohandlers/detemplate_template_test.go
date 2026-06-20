package scenariohandlers

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/vrooli/vrooli/internal/cli/scenariocli" //nolint:revive // mirror runtime dot-import for fixture construction.
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
)

// findReactViteTemplate walks up from the test's working directory to locate
// the real react-vite template, so this integration test runs against the
// actual marked template content. Returns "" if not found (test skips).
func findReactViteTemplate(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, "templates", "scenarios", "react-vite", "template.json")
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Dir(candidate)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// copyTreeSkippingHeavy copies src→dst, skipping vendored/build directories.
func copyTreeSkippingHeavy(t *testing.T, src, dst string) {
	t.Helper()
	skip := map[string]struct{}{"node_modules": {}, ".git": {}, "dist": {}, "build": {}, ".next": {}, "coverage": {}}
	err := filepath.Walk(src, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, _ := filepath.Rel(src, path)
		if info.IsDir() {
			if _, s := skip[info.Name()]; s {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(filepath.Join(dst, rel))
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestReactViteTemplateDetemplatesClean is the round-trip proof: it runs the
// detemplate strip/prune/delete logic over a copy of the REAL marked
// react-vite template and asserts zero EXAMPLE-DOMAIN markers survive, no
// dangling references remain, and the pruned locale JSON is valid and
// notes-free. This keeps the template's fences and markers honest as it
// evolves — if someone adds unfenced notes content, this fails.
func TestReactViteTemplateDetemplatesClean(t *testing.T) {
	templateDir := findReactViteTemplate(t)
	if templateDir == "" {
		t.Skip("react-vite template not found from test working directory")
	}
	manifestBytes, err := os.ReadFile(filepath.Join(templateDir, "template.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest TemplateManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("parse template.json: %v", err)
	}
	if manifest.ExampleDomain == nil || manifest.ExampleDomain.Marker == "" {
		t.Fatal("template declares no exampleDomain")
	}
	marker := manifest.ExampleDomain.Marker

	// Copy the template into a temp "scenario" layout.
	root := t.TempDir()
	scenarioPath := filepath.Join(root, "scenarios", "react-vite-detemplate-proof")
	copyTreeSkippingHeavy(t, templateDir, scenarioPath)

	// Mirror what generation actually ships: copyExcludes files and the
	// manifest itself are not part of a generated scenario, and they
	// legitimately name the EXAMPLE-DOMAIN marker as documentation/config.
	for _, rel := range append([]string{"template.json"}, manifest.CopyExcludes...) {
		_ = os.Remove(filepath.Join(scenarioPath, filepath.FromSlash(rel)))
	}

	item := scenariomodel.Scenario{Slug: "react-vite-detemplate-proof", Path: scenarioPath}
	info := TemplateInfo{Name: "react-vite", Manifest: manifest}

	// Simulate the generator's out-of-tree proto relocation. The scenario copy
	// does not contain proto/, so this is the load-bearing proof that
	// detemplate also removes relocated example-domain protos while preserving
	// conventional shared health/errors contracts.
	values := map[string]string{
		"SCENARIO_ID":       item.Slug,
		"SCENARIO_ID_SNAKE": strings.ReplaceAll(item.Slug, "-", "_"),
	}
	for _, reloc := range manifest.Relocations {
		if reloc.From != "proto/" {
			continue
		}
		to := filepath.Join(root, filepath.FromSlash(renderTemplateString(reloc.To, values)))
		if err := copyRelocationTree(filepath.Join(templateDir, filepath.FromSlash(reloc.From)), to, values); err != nil {
			t.Fatal(err)
		}
	}

	// Resolve scenario-local deletions plus relocated proto schema deletions.
	deletions := resolveDetemplateDeletions(root, item, info, manifest.ExampleDomain.Paths)
	if len(deletions) == 0 {
		t.Fatal("expected example-domain paths to resolve for deletion")
	}

	edits, dangling, err := planDetemplateEdits(scenarioPath, marker, deletions)
	if err != nil {
		t.Fatal(err)
	}
	if len(dangling) > 0 {
		t.Fatalf("marked template still has dangling references after strip: %+v", dangling)
	}
	jsonEdits, err := planDetemplateJSONPrunes(scenarioPath, manifest.ExampleDomain.JSONPrune)
	if err != nil {
		t.Fatal(err)
	}
	edits = append(edits, jsonEdits...)

	// Apply everything.
	for _, e := range edits {
		if err := os.WriteFile(e.Abs, e.Content, e.Mode); err != nil {
			t.Fatal(err)
		}
	}
	for _, d := range deletions {
		if err := os.RemoveAll(d.Abs); err != nil {
			t.Fatal(err)
		}
	}

	// Proof 1: zero EXAMPLE-DOMAIN:notes markers survive anywhere (the same
	// token the orient residue gate scans for).
	hits, err := scanTreeForText(scenarioPath, "EXAMPLE-DOMAIN:"+marker)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) > 0 {
		t.Fatalf("EXAMPLE-DOMAIN:%s markers survived detemplate in %d file(s): %v", marker, len(hits), hits)
	}

	// Proof 2: pruned locale JSON is valid and notes-free.
	for _, e := range manifest.ExampleDomain.JSONPrune {
		if !strings.Contains(e.File, "locales") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(scenarioPath, filepath.FromSlash(e.File)))
		if err != nil {
			continue // file may be among deleted paths
		}
		var v any
		if err := json.Unmarshal(data, &v); err != nil {
			t.Errorf("%s invalid JSON after prune: %v", e.File, err)
		}
		if m, ok := v.(map[string]any); ok {
			if _, present := m["notes"]; present {
				t.Errorf("%s still has top-level notes key after prune", e.File)
			}
		}
	}

	// Proof 3: the example code directories are gone.
	for _, p := range []string{"api/internal/notes", "cli/domains/notes", "ui/src/features/notes"} {
		if _, err := os.Stat(filepath.Join(scenarioPath, filepath.FromSlash(p))); !os.IsNotExist(err) {
			t.Errorf("expected %s to be deleted", p)
		}
	}

	// Proof 4: relocated example-domain protos are gone, but conventional
	// shared health/errors protos remain.
	relocatedProtoRoot := filepath.Join(root, "packages", "proto", "schemas", item.Slug, "v1")
	if _, err := os.Stat(filepath.Join(relocatedProtoRoot, "notes")); !os.IsNotExist(err) {
		t.Errorf("expected relocated notes protos to be deleted")
	}
	for _, p := range []string{"shared/health.proto", "shared/errors.proto"} {
		if _, err := os.Stat(filepath.Join(relocatedProtoRoot, filepath.FromSlash(p))); err != nil {
			t.Errorf("expected relocated %s to remain: %v", p, err)
		}
	}
}
