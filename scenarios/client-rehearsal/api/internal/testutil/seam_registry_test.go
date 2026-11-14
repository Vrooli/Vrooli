package testutil_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var seamTag = regexp.MustCompile(`// seam: ([A-Za-z0-9_.]+)`)
var registryRow = regexp.MustCompile("^\\| `([^`]+)` \\|")

func TestSeamRegistryMatchesCodeTags(t *testing.T) {
	apiRoot := filepath.Clean(filepath.Join("..", ".."))
	internalRoot := filepath.Join(apiRoot, "internal")
	codeSeams := map[string]bool{}
	err := filepath.WalkDir(internalRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, match := range seamTag.FindAllSubmatch(body, -1) {
			codeSeams[string(match[1])] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(apiRoot, "..", "docs", "internal", "SEAMS.md")
	doc, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	docSeams := map[string]bool{}
	for _, line := range strings.Split(string(doc), "\n") {
		match := registryRow.FindStringSubmatch(line)
		if match != nil {
			docSeams[match[1]] = true
		}
	}
	for seam := range codeSeams {
		if !docSeams[seam] {
			t.Errorf("code seam %q is missing from docs/internal/SEAMS.md", seam)
		}
	}
	for seam := range docSeams {
		if !codeSeams[seam] {
			t.Errorf("registry seam %q has no // seam: tag in production code", seam)
		}
	}
}
