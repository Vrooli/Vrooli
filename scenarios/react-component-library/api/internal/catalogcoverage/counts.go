package catalogcoverage

// This file owns the corpus line-count measurements used by I27, the tracked
// non-blocking machinery-ratio invariant.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"react-component-library/internal/librarywalk"
)

func countRoot() string {
	_, file, _, _ := runtime.Caller(0)
	// counts.go lives at scenarios/react-component-library/api/internal/
	// catalogcoverage. Five parents reach the repository root.
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../.."))
}

func countLines(root string, include func(string) bool) int {
	lines := 0
	_ = librarywalk.WalkContext(context.Background(), root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".retired", "node_modules", "dist":
				return filepath.SkipDir
			}
			return nil
		}
		if !include(path) {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr == nil {
			lines += strings.Count(string(data), "\n")
		}
		return nil
	})
	return lines
}

func goFile(path string) bool {
	return filepath.Ext(path) == ".go" && !strings.HasSuffix(path, "_test.go")
}

func sourceFile(path string) bool {
	ext := filepath.Ext(path)
	lower := strings.ToLower(filepath.ToSlash(path))
	return (ext == ".ts" || ext == ".tsx") && !strings.Contains(lower, "test") &&
		!strings.Contains(lower, "/mocks/") && !strings.Contains(lower, "/fixtures/") &&
		!strings.HasSuffix(lower, "/strings.generated.ts") &&
		!strings.HasSuffix(lower, "/selectors.library.ts")
}

func CountGoInternals() int {
	return countLines(filepath.Join(countRoot(), "scenarios", "react-component-library", "api"), func(path string) bool {
		slashPath := filepath.ToSlash(path)
		// Mock and testutil packages are validation support imported by tests,
		// not shipped API machinery. Keep them out of the same non-test measure
		// that already excludes *_test.go files.
		return goFile(path) && !strings.Contains(slashPath, "/cmd/") &&
			!strings.Contains(slashPath, "/mocks/") &&
			!strings.Contains(slashPath, "/internal/testutil/")
	})
}

func CountCmdLines() int {
	return countLines(filepath.Join(countRoot(), "scenarios", "react-component-library", "api", "cmd"), goFile)
}

func CountCmdDirs() int {
	count := 0
	entries, _ := os.ReadDir(filepath.Join(countRoot(), "scenarios", "react-component-library", "api", "cmd"))
	for _, entry := range entries {
		if entry.IsDir() {
			count++
		}
	}
	return count
}

func CountCLILines() int {
	return countLines(filepath.Join(countRoot(), "scenarios", "react-component-library", "cli"), goFile)
}

func CountCLICommands() int {
	data, err := os.ReadFile(filepath.Join(countRoot(), "scenarios", "react-component-library", "cli", "manifest.json"))
	if err != nil {
		return 0
	}
	var value any
	if json.Unmarshal(data, &value) != nil {
		return 0
	}
	return ledgerBindingCount(value)
}

func ledgerBindingCount(value any) int {
	count := 0
	switch item := value.(type) {
	case []any:
		for _, child := range item {
			count += ledgerBindingCount(child)
		}
	case map[string]any:
		if _, ok := item["binding"]; ok {
			count++
		}
		for _, child := range item {
			count += ledgerBindingCount(child)
		}
	}
	return count
}

func CountUIScriptLines() int {
	return countLines(filepath.Join(countRoot(), "scenarios", "react-component-library", "ui", "scripts"), func(path string) bool {
		return filepath.Ext(path) == ".mjs" && !strings.HasSuffix(path, ".test.mjs")
	})
}

func CountToolingLines() int {
	return countLines(filepath.Join(countRoot(), "packages", "react-component-library", "tooling"), func(path string) bool {
		return filepath.Ext(path) == ".mjs" && !strings.HasSuffix(path, ".test.mjs")
	})
}

func CountWorkbenchLines() int {
	return countLines(filepath.Join(countRoot(), "scenarios", "react-component-library", "ui", "src"), sourceFile)
}

func CountMakefileTargets() int {
	data, _ := os.ReadFile(filepath.Join(countRoot(), "scenarios", "react-component-library", "Makefile"))
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, ":") && strings.Contains(line, "##") && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			count++
		}
	}
	return count
}

func CountUIConfigFiles() int {
	count := 0
	entries, _ := os.ReadDir(filepath.Join(countRoot(), "scenarios", "react-component-library", "ui"))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".config.js") || strings.HasSuffix(name, ".config.ts") || strings.HasPrefix(name, "tsconfig") || strings.HasPrefix(name, "tailwind") || strings.HasPrefix(name, "postcss") {
			count++
		}
	}
	return count
}

func CountDocsMarkdown() int {
	count := 0
	_ = librarywalk.WalkContext(context.Background(), filepath.Join(countRoot(), "scenarios", "react-component-library", "docs"), func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && filepath.Ext(path) == ".md" {
			count++
		}
		return err
	})
	return count
}

func CountAssetLines() int {
	return countLines(filepath.Join(countRoot(), "scenarios", "react-component-library", "library"), func(path string) bool {
		if !sourceFile(path) && filepath.Ext(path) != ".css" {
			return false
		}
		return strings.Contains(filepath.ToSlash(path), "/versions/")
	})
}

// CountMachineryLines is the tracked non-blocking ratio numerator used by
// corpus invariant I27. Tests and component source are intentionally excluded
// because they are validation evidence, not shipped machinery.
func CountMachineryLines() int {
	return CountGoInternals() + CountCmdLines() + CountCLILines() + CountUIScriptLines() + CountToolingLines()
}
