package reactvitest

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"unit-health/internal/adapters"
)

type ArchitectureDrift = adapters.ArchitectureDrift

// Analyzer exposes the React/Vitest static capabilities through the generic
// adapter contract. Framework-specific parsing remains in this package.
type Analyzer struct{}

func (Analyzer) Identity() adapters.Identity {
	return adapters.Identity{ID: "react-vitest", Version: "1.0.0"}
}

func (Analyzer) Matches(m adapters.Match) bool {
	return strings.EqualFold(m.Language, "typescript") && (strings.EqualFold(m.Framework, "vite") || strings.EqualFold(m.Framework, "vitest"))
}

func (Analyzer) ValidatePolicySettings(settings map[string]json.RawMessage) error {
	return ValidatePolicySettings(settings)
}
func (Analyzer) DefaultProjectionPolicy() adapters.ProjectionPolicy { return DefaultPolicy() }
func (Analyzer) ProjectionPolicyFromSettings(settings map[string]json.RawMessage) adapters.ProjectionPolicy {
	return PolicyFromSettings(settings)
}

func (Analyzer) AnalyzeArchitecture(root string) []adapters.ArchitectureDrift {
	return AnalyzeArchitecture(root)
}

func (Analyzer) AnalyzeProjection(input adapters.ProjectionInput) []adapters.ProjectionDrift {
	return AnalyzeProjection(input)
}

func (Analyzer) AnalyzeProjectionChecks(input adapters.ProjectionInput) []adapters.ProjectionCheck {
	return AnalyzeProjectionChecks(input)
}

var testingLibraryRenderImport = regexp.MustCompile(`(?s)import\s*\{([^}]*)\}\s*from\s*["']@testing-library/react["']`)

var skipDirs = map[string]bool{".git": true, "node_modules": true, "dist": true, "build": true, ".cache": true, "coverage": true, "vendor": true}

// AnalyzeArchitecture owns React/Vitest-specific test-architecture rules.
func AnalyzeArchitecture(root string) []ArchitectureDrift {
	testFiles := 0
	hasTestUtils, hasRenderHelper, hasSharedRenderHelper := false, false, false
	directRenderFiles := map[string]bool{}
	rogue := map[string]bool{}
	walkSourceFiles(root, func(path string, source string) {
		if isTSTestFile(path) {
			testFiles++
			if importsTestingLibraryRender(source) && !strings.Contains(source, "provider-free-exception:") {
				directRenderFiles[path] = true
			}
			parent := filepath.Base(filepath.Dir(path))
			if (parent == "__tests__" || parent == "tests" || parent == "test") && !strings.Contains(path, string(filepath.Separator)+"src"+string(filepath.Separator)) {
				rogue[filepath.Dir(path)] = true
			}
		}
		base, dir := filepath.Base(path), filepath.Base(filepath.Dir(path))
		if dir == "test-utils" || dir == "test-util" || strings.HasPrefix(base, "setupTests.") || strings.HasPrefix(base, "test-utils.") {
			hasTestUtils = true
		}
		if strings.HasPrefix(base, "renderWithProviders.") && dir == "test-utils" {
			hasRenderHelper = true
		}
		if importsSharedRenderHelper(source) {
			hasSharedRenderHelper = true
		}
	})
	hasRenderHelper = hasRenderHelper || hasSharedRenderHelper
	if hasSharedRenderHelper {
		hasTestUtils = true
	}
	var drifts []ArchitectureDrift
	add := func(code, file, message, evidence, expected, observed, why, remediation string) {
		drifts = append(drifts, ArchitectureDrift{Code: code, File: file, Message: message, Evidence: evidence, Expected: expected, Observed: observed, WhyItMatters: why, Remediation: remediation})
	}
	if testFiles >= 3 && !hasTestUtils {
		add("TEST_UTIL_MISSING", root, "UI workspace has several test files but no shared test-utils module.", formatCount(testFiles)+" test files; no src/test-utils or setupTests found", "A shared src/test-utils (render wrapper, fixtures) and a setupTests file.", "no shared test-utils module", "Without shared render/util helpers, component tests duplicate provider setup and drift.", "Add a src/test-utils module with a custom render and shared fixtures.")
	}
	if hasTestUtils && !hasRenderHelper {
		add("UNIT_POLICY_PROJECTION_DRIFT", filepath.Join(root, "src", "test-utils"), "UI test-utils projection is missing the canonical render helper.", "src/test-utils exists; no renderWithProviders helper found", "src/test-utils/renderWithProviders.tsx exports the canonical provider-aware render helper.", "missing renderWithProviders helper", "Component tests need one provider-aware render path so provider setup does not drift.", "Add src/test-utils/renderWithProviders.tsx and re-export it from src/test-utils/index.ts.")
	}
	if hasRenderHelper && len(directRenderFiles) > 0 {
		files := make([]string, 0, len(directRenderFiles))
		for path := range directRenderFiles {
			files = append(files, relative(root, path))
		}
		sort.Strings(files)
		if len(files) > 10 {
			files = files[:10]
		}
		add("UNIT_POLICY_PROJECTION_DRIFT", root, "UI tests bypass the canonical renderWithProviders helper.", "direct Testing Library render imports: "+strings.Join(files, ", "), "Component tests import renderWithProviders from src/test-utils so provider setup stays centralized.", "direct Testing Library render import", "Bypassing the canonical render helper lets provider setup drift between tests.", "Replace direct Testing Library render imports with renderWithProviders, or document a provider-free exception with a reason.")
	}
	if len(rogue) > 0 {
		dirs := make([]string, 0, len(rogue))
		for dir := range rogue {
			dirs = append(dirs, relative(root, dir))
		}
		sort.Strings(dirs)
		add("TEST_NOT_COLOCATED", root, "UI test files live outside src/, separated from the code they test.", "non-colocated test dirs: "+strings.Join(dirs, ", "), "Test files co-located with components under src/.", "tests outside src/", "Separated tests drift from their components and obscure which code is covered.", "Co-locate component tests beside their source under src/.")
	}
	return drifts
}

func walkSourceFiles(root string, fn func(string, string)) {
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if path != root && skipDirs[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		fn(path, readFile(path))
		return nil
	})
}

func isTSTestFile(path string) bool {
	base := filepath.Base(path)
	for _, infix := range []string{".test.", ".spec."} {
		if strings.Contains(base, infix) {
			switch filepath.Ext(base) {
			case ".ts", ".tsx", ".js", ".jsx", ".mts", ".cts":
				return true
			}
		}
	}
	return false
}

func importsTestingLibraryRender(src string) bool {
	for _, match := range testingLibraryRenderImport.FindAllStringSubmatch(src, -1) {
		for _, part := range strings.Split(match[1], ",") {
			fields := strings.Fields(strings.TrimSpace(part))
			if len(fields) > 0 && fields[0] == "render" {
				return true
			}
		}
	}
	return false
}

func importsSharedRenderHelper(src string) bool {
	return strings.Contains(src, "@vrooli/api-base/testing") && strings.Contains(src, "renderWithProviders") && strings.Contains(src, "import")
}

// HasSharedRenderHelper reports whether source imports the package-owned
// provider-aware render helper.
func HasSharedRenderHelper(src string) bool {
	return importsSharedRenderHelper(src)
}

func readFile(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(raw)
}

func relative(root, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil {
		return rel
	}
	return path
}

func formatCount(n int) string {
	return fmt.Sprintf("%d", n)
}
