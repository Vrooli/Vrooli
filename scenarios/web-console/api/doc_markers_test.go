package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode"
)

var docMarkerRE = regexp.MustCompile(`(?m)^\s*(?://|/\*|\*|#)\s*DOC:\s*([^\s]+)`)

func githubSlug(s string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-':
			b.WriteRune(r)
			lastDash = false
		case unicode.IsSpace(r):
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func docHeadingSlugs(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	headings := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		text := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		if text == "" {
			continue
		}
		headings[githubSlug(text)] = text
	}
	return headings, nil
}

func TestDocMarkers(t *testing.T) {
	scenarioRoot := filepath.Clean("..")
	var sources []string
	for _, sourceRoot := range []string{"api", "ui/src", "cli"} {
		err := filepath.WalkDir(filepath.Join(scenarioRoot, sourceRoot), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if entry.Name() == "node_modules" || entry.Name() == "dist" {
					return filepath.SkipDir
				}
				return nil
			}
			ext := filepath.Ext(entry.Name())
			if ext == ".go" || ext == ".ts" || ext == ".tsx" {
				sources = append(sources, path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, source := range sources {
		data, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range docMarkerRE.FindAllStringSubmatch(string(data), -1) {
			marker := match[1]
			parts := strings.SplitN(marker, "#", 2)
			docPath := filepath.Join(scenarioRoot, filepath.FromSlash(parts[0]))
			if _, err := os.Stat(docPath); err != nil {
				t.Errorf("DOC marker %q in %s names missing document: %v", marker, source, err)
				continue
			}
			if len(parts) != 2 {
				continue
			}
			headings, err := docHeadingSlugs(docPath)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := headings[strings.ToLower(parts[1])]; !ok {
				t.Errorf("DOC marker %q in %s names missing heading", marker, source)
			}
		}
	}
}
