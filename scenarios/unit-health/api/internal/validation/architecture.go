package validation

import (
	"fmt"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
)

// analyzeArchitecture runs the static, source-fact-driven test-architecture
// checks for each workspace. It needs no execution: it reads source files under
// the workspace root and emits co-location, shared-test-utility,
// production-helper-import, and injectable-seam findings.
func analyzeArchitecture(scenario string, workspaces []Workspace, now string) []Finding {
	var findings []Finding
	for _, ws := range workspaces {
		switch ws.Language {
		case "go":
			findings = append(findings, analyzeGoArchitecture(scenario, ws, now)...)
		case "typescript":
			findings = append(findings, analyzeTSArchitecture(scenario, ws, now)...)
		}
	}
	return findings
}

// goWorkspaceScan collects the source facts the Go architecture checks need in a
// single walk of the workspace.
type goWorkspaceScan struct {
	testFiles      int
	hasTestUtilPkg bool
	prodSeamUsers  map[string]bool // seam name -> used directly in production
	prodHelperUses []helperImport
	rogueTestDirs  map[string]bool
}

type helperImport struct {
	file       string
	importPath string
}

func analyzeGoArchitecture(scenario string, ws Workspace, now string) []Finding {
	modulePath := goModulePath(ws.RootPath)
	scan := goWorkspaceScan{prodSeamUsers: map[string]bool{}, rogueTestDirs: map[string]bool{}}

	walkSourceFiles(ws.RootPath, func(path string) {
		switch {
		case isGoTestFile(path):
			scan.testFiles++
			// A Go test file living under a directory named tests/ or test/ is
			// not co-located with its package.
			parent := filepath.Base(filepath.Dir(path))
			if parent == "tests" || parent == "test" {
				scan.rogueTestDirs[filepath.Dir(path)] = true
			}
		case isGoSourceFile(path):
			rel := filepath.Dir(path)
			base := filepath.Base(rel)
			if base == "testutil" || base == "testutils" || base == "testkit" || base == "testhelpers" {
				scan.hasTestUtilPkg = true
			}
			imports := goImports(path)
			for _, imp := range imports {
				if isTestHelperImport(imp, modulePath) {
					scan.prodHelperUses = append(scan.prodHelperUses, helperImport{file: path, importPath: imp})
				}
			}
			detectSeamUsage(readFileString(path), scan.prodSeamUsers)
		}
	})

	var findings []Finding
	mk := func(code, file, message, evidence, expected, observed, why, remediation string) Finding {
		return Finding{
			ID:           code + "-" + ws.ID,
			Scenario:     scenario,
			WorkspaceID:  ws.ID,
			Language:     "go",
			Code:         code,
			Category:     "architecture",
			Severity:     codeSeverity[code],
			FilePath:     file,
			Message:      message,
			Evidence:     evidence,
			Expected:     expected,
			Observed:     observed,
			WhyItMatters: why,
			Remediation:  remediation,
			CreatedAt:    now,
		}
	}

	if len(scan.prodHelperUses) > 0 {
		ex := make([]string, 0, len(scan.prodHelperUses))
		for _, h := range scan.prodHelperUses {
			ex = append(ex, fmt.Sprintf("%s imports %s", relTo(ws.RootPath, h.file), h.importPath))
		}
		sort.Strings(ex)
		findings = append(findings, mk(codeTestHelperFromProd, scan.prodHelperUses[0].file,
			"Production code imports a test-helper package.",
			strings.Join(ex, "; "),
			"Test helpers (testutil/mocks/fixtures) are imported only from _test.go files.",
			fmt.Sprintf("%d production file(s) import test helpers", len(scan.prodHelperUses)),
			"Importing test helpers from production ships test-only code in the binary and inverts the dependency direction.",
			"Move the shared logic into a production package, or restrict the helper import to _test.go files (see the no_prod_import_test convention).",
		))
	}

	if scan.testFiles >= 3 && !scan.hasTestUtilPkg {
		findings = append(findings, mk(codeTestUtilMissing, ws.RootPath,
			"Workspace has several test files but no shared test-utility package.",
			fmt.Sprintf("%d test files; no testutil/ package found", scan.testFiles),
			"A shared internal/testutil (or testkit) package for fixtures, fakes, and assertions.",
			"no shared test-utility package",
			"Without shared test utilities, setup is duplicated across tests and fakes drift out of sync.",
			"Extract shared fixtures/fakes into an internal/testutil package.",
		))
	}

	if len(scan.rogueTestDirs) > 0 {
		dirs := make([]string, 0, len(scan.rogueTestDirs))
		for d := range scan.rogueTestDirs {
			dirs = append(dirs, relTo(ws.RootPath, d))
		}
		sort.Strings(dirs)
		findings = append(findings, mk(codeTestNotColocated, ws.RootPath,
			"Go test files live in a separate tests/ directory instead of beside their package.",
			"non-colocated test dirs: "+strings.Join(dirs, ", "),
			"Go test files co-located with the package they test (same directory).",
			"tests in a separate directory",
			"Non-co-located Go tests cannot access unexported symbols and drift away from the code they cover.",
			"Move the _test.go files into the package directory they exercise.",
		))
	}

	if scan.testFiles > 0 {
		missing := missingSeams(scan.prodSeamUsers)
		if len(missing) > 0 {
			findings = append(findings, mk(codeMissingInjectableSeam, ws.RootPath,
				"Production code uses non-deterministic dependencies directly without an injectable seam.",
				"direct use of: "+strings.Join(missing, ", "),
				"Time, env, and HTTP dependencies injected through a seam (e.g. a clock func, env reader, or http.Client interface) so tests can substitute them.",
				"direct calls to "+strings.Join(missing, ", "),
				"Hard-wired time/env/HTTP makes behavior non-deterministic and forces tests to touch the real world.",
				"Introduce injectable seams (clock, env, http doer) and pass fakes in tests (see seam-discovery-and-enforcement).",
			))
		}
	}

	return findings
}

// detectSeamUsage records direct, non-injectable dependency calls in source.
func detectSeamUsage(src string, into map[string]bool) {
	if strings.Contains(src, "time.Now(") {
		into["time.Now"] = true
	}
	if strings.Contains(src, "os.Getenv(") || strings.Contains(src, "os.LookupEnv(") {
		into["os.Getenv"] = true
	}
	if strings.Contains(src, "http.Get(") || strings.Contains(src, "http.Post(") || strings.Contains(src, "http.DefaultClient") {
		into["net/http default client"] = true
	}
}

// missingSeams returns the sorted list of directly-used dependencies that lack a
// seam. A workspace that provides a clock/env/http seam still trips this when
// production code bypasses it; the heuristic is intentionally conservative and
// only flags the direct-use forms detectSeamUsage recognizes.
func missingSeams(used map[string]bool) []string {
	out := make([]string, 0, len(used))
	for k := range used {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// isTestHelperImport reports whether an import path points at a module-local
// test-helper package (testutil/mocks/fixtures), which production code must not
// import.
func isTestHelperImport(importPath, modulePath string) bool {
	if modulePath == "" || !strings.HasPrefix(importPath, modulePath) {
		return false
	}
	last := importPath
	if i := strings.LastIndex(importPath, "/"); i >= 0 {
		last = importPath[i+1:]
	}
	switch last {
	case "testutil", "testutils", "testkit", "testhelpers", "mocks", "fixtures":
		return true
	}
	return strings.Contains(importPath, "/testutil/")
}

// goImports returns the import paths of a Go file, best-effort.
func goImports(path string) []string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil || f == nil {
		return nil
	}
	out := make([]string, 0, len(f.Imports))
	for _, imp := range f.Imports {
		out = append(out, strings.Trim(imp.Path.Value, `"`))
	}
	return out
}

// goModulePath reads the module path from the nearest go.mod at or above root.
func goModulePath(root string) string {
	dir := root
	for i := 0; i < 8 && dir != "" && dir != string(filepath.Separator); i++ {
		raw := readFileString(filepath.Join(dir, "go.mod"))
		if raw != "" {
			for _, line := range strings.Split(raw, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module ") {
					return strings.TrimSpace(strings.TrimPrefix(line, "module "))
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func relTo(base, path string) string {
	if rel, err := filepath.Rel(base, path); err == nil {
		return rel
	}
	return path
}

// analyzeTSArchitecture runs the TypeScript/Vite test-architecture checks.
func analyzeTSArchitecture(scenario string, ws Workspace, now string) []Finding {
	var testFiles int
	var hasTestUtils bool
	rogue := map[string]bool{}

	walkSourceFiles(ws.RootPath, func(path string) {
		if isTSTestFile(path) {
			testFiles++
			parent := filepath.Base(filepath.Dir(path))
			if parent == "__tests__" || parent == "tests" || parent == "test" {
				if !strings.Contains(path, string(filepath.Separator)+"src"+string(filepath.Separator)) {
					rogue[filepath.Dir(path)] = true
				}
			}
		}
		base := filepath.Base(path)
		dir := filepath.Base(filepath.Dir(path))
		if dir == "test-utils" || dir == "test-util" || strings.HasPrefix(base, "setupTests.") || strings.HasPrefix(base, "test-utils.") {
			hasTestUtils = true
		}
	})

	var findings []Finding
	mk := func(code, file, message, evidence, expected, observed, why, remediation string) Finding {
		return Finding{
			ID:           code + "-" + ws.ID,
			Scenario:     scenario,
			WorkspaceID:  ws.ID,
			Language:     "typescript",
			Code:         code,
			Category:     "architecture",
			Severity:     codeSeverity[code],
			FilePath:     file,
			Message:      message,
			Evidence:     evidence,
			Expected:     expected,
			Observed:     observed,
			WhyItMatters: why,
			Remediation:  remediation,
			CreatedAt:    now,
		}
	}

	if testFiles >= 3 && !hasTestUtils {
		findings = append(findings, mk(codeTestUtilMissing, ws.RootPath,
			"UI workspace has several test files but no shared test-utils module.",
			fmt.Sprintf("%d test files; no src/test-utils or setupTests found", testFiles),
			"A shared src/test-utils (render wrapper, fixtures) and a setupTests file.",
			"no shared test-utils module",
			"Without shared render/util helpers, component tests duplicate provider setup and drift.",
			"Add a src/test-utils module with a custom render and shared fixtures.",
		))
	}

	if len(rogue) > 0 {
		dirs := make([]string, 0, len(rogue))
		for d := range rogue {
			dirs = append(dirs, relTo(ws.RootPath, d))
		}
		sort.Strings(dirs)
		findings = append(findings, mk(codeTestNotColocated, ws.RootPath,
			"UI test files live outside src/, separated from the code they test.",
			"non-colocated test dirs: "+strings.Join(dirs, ", "),
			"Test files co-located with components under src/ (e.g. Button.test.tsx beside Button.tsx).",
			"tests outside src/",
			"Separated tests drift from their components and obscure which code is covered.",
			"Co-locate component tests beside their source under src/.",
		))
	}

	return findings
}
