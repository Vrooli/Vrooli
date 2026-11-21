// Package seamregistry holds the drift gate that keeps the `// seam:` tags in
// the test-genie codebase reconciled with docs/internal/SEAMS.md. It has no
// production code — only the test — so it imports nothing and never ships in
// the binary.
package seamregistry

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// seamTagPattern matches a `// seam: <Name>` declaration and captures <Name>,
// the first identifier-ish token after the tag. Trailing punctuation (em dash,
// colon) is excluded by the character class.
var seamTagPattern = regexp.MustCompile(`//\s*seam:\s*([A-Za-z0-9_.]+)`)

// TestSeamRegistryReconciled walks every Go file under api/internal, collects
// the names of all `// seam:`-tagged declarations, and fails if any name is not
// documented in docs/internal/SEAMS.md. This is the L5 anti-drift gate: a new
// seam that is not registered breaks the build.
func TestSeamRegistryReconciled(t *testing.T) {
	internalRoot := mustAbs(t, "..") // api/internal
	seamsDoc := mustAbs(t, "../../../docs/internal/SEAMS.md")

	docBytes, err := os.ReadFile(seamsDoc)
	if err != nil {
		t.Fatalf("read SEAMS.md: %v", err)
	}
	doc := string(docBytes)

	tags := collectSeamTags(t, internalRoot)
	if len(tags) == 0 {
		t.Fatal("no // seam: tags found — the scanner is broken")
	}

	for name, locs := range tags {
		// A seam is documented when its name appears in the registry, by
		// convention wrapped in backticks (`Resolver`).
		if strings.Contains(doc, "`"+name+"`") || strings.Contains(doc, name) {
			continue
		}
		t.Errorf("seam %q (declared at %s) is not documented in docs/internal/SEAMS.md — add a registry row", name, strings.Join(locs, ", "))
	}
}

func collectSeamTags(t *testing.T, root string) map[string][]string {
	t.Helper()
	tags := make(map[string][]string)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Skip the registry test package itself (the pattern literal above
		// would otherwise self-match).
		if strings.Contains(path, "seamregistry") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, m := range seamTagPattern.FindAllStringSubmatch(string(data), -1) {
			name := strings.TrimRight(m[1], ".:")
			if name == "" {
				continue
			}
			rel, _ := filepath.Rel(root, path)
			tags[name] = append(tags[name], rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return tags
}

func mustAbs(t *testing.T, rel string) string {
	t.Helper()
	abs, err := filepath.Abs(rel)
	if err != nil {
		t.Fatalf("abs %s: %v", rel, err)
	}
	return abs
}
