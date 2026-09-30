package validation

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Phase 5 finding codes for the coverage, architecture, quality, and diagnostics
// analyzers. Every code maps to an entry in the `maturity` block of `.vrooli/test-genie.json`; the anti-drift
// test in maturity_spec_test.go proves codeSeverity and the spec agree.
const (
	codeLowCoverage             = "LOW_COVERAGE"
	codeCoverageAbsent          = "COVERAGE_ABSENT"
	codeTestNotColocated        = "TEST_NOT_COLOCATED"
	codeTestUtilMissing         = "TEST_UTIL_MISSING"
	codeTestHelperFromProd      = "TEST_HELPER_FROM_PRODUCTION"
	codeMissingInjectableSeam   = "MISSING_INJECTABLE_SEAM"
	codeTestSkippedOrOnly       = "TEST_SKIPPED_OR_ONLY"
	codeTestExcessiveSnapshots  = "TEST_EXCESSIVE_SNAPSHOTS"
	codeTestFlakeSuspected      = "TEST_FLAKE_SUSPECTED"
	codeTestRuntimeGrowth       = "TEST_RUNTIME_GROWTH"
	codeTestUntaggedRequirement = "TEST_UNTAGGED_REQUIREMENT"
	codeTestQualityViolation    = "TEST_QUALITY_RULE_VIOLATION"
	codeSeamDuplicatedInPackage = "SEAM_DUPLICATED_IN_PACKAGE"
	codeSeamReimplemented       = "SEAM_REIMPLEMENTED"
	codeCompanionReimplemented  = "COMPANION_REIMPLEMENTED"
	codeCompanionAvailable      = "COMPANION_AVAILABLE"
)

// analyzerSkipDirs are directories the static analyzers never descend into:
// dependency trees, build outputs, coverage artifacts, and test fixtures
// (testdata holds intentionally-shaped sample code, not real tests).
var analyzerSkipDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	".git":         true,
	"dist":         true,
	"build":        true,
	"coverage":     true,
	".next":        true,
	".turbo":       true,
	"out":          true,
	"testdata":     true,
}

// EmittedCodes returns the validation finding vocabulary owned by this
// package. Callers use the copy to compare authored specifications without
// reaching into the package's severity table.
func EmittedCodes() map[string]struct{} {
	out := make(map[string]struct{}, len(codeSeverity))
	for code := range codeSeverity {
		out[code] = struct{}{}
	}
	return out
}

// walkSourceFiles invokes fn for every file under root, skipping dependency,
// build, and fixture directories. Errors from the walk are swallowed so a
// partially-readable tree still yields findings.
func walkSourceFiles(root string, fn func(path string)) {
	if root == "" {
		return
	}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // best-effort scan: skip unreadable entries.
		}
		if d.IsDir() {
			if path != root && analyzerSkipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		fn(path)
		return nil
	})
}

// readFileString reads a file best-effort, returning "" on any error.
func readFileString(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(raw)
}

// isGoTestFile reports whether path is a Go test file.
func isGoTestFile(path string) bool { return strings.HasSuffix(path, "_test.go") }

// isGoSourceFile reports whether path is a non-test Go source file.
func isGoSourceFile(path string) bool {
	return strings.HasSuffix(path, ".go") && !isGoTestFile(path)
}

// isTSTestFile reports whether path is a TypeScript/JavaScript test/spec file.
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

func isTSSourceFile(path string) bool {
	switch filepath.Ext(path) {
	case ".ts", ".tsx", ".js", ".jsx", ".mts", ".cts":
		return true
	default:
		return false
	}
}
