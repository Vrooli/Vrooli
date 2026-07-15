package validation

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
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
	hasImportBan   bool
	seamBypass     map[string]bool // seam category -> production bypasses it (AST)
	seamDeclared   map[string]bool // seam category -> a seam interface/pkg exists
	prodHelperUses []helperImport
	rogueTestDirs  map[string]bool
}

type helperImport struct {
	file       string
	importPath string
}

func analyzeGoArchitecture(scenario string, ws Workspace, now string) []Finding {
	modulePath := goModulePath(ws.RootPath)
	scan := goWorkspaceScan{seamBypass: map[string]bool{}, seamDeclared: map[string]bool{}, rogueTestDirs: map[string]bool{}}

	walkSourceFiles(ws.RootPath, func(path string) {
		// A seam-implementation package (clock/httpc/envx/logx) legitimately
		// calls the ambient primitives it wraps; its presence is evidence a
		// seam exists, and it must never count as a production bypass.
		if cat := seamDirCategory(path); cat != "" {
			scan.seamDeclared[cat] = true
		}
		switch {
		case isGoTestFile(path):
			scan.testFiles++
			if filepath.Base(path) == "no_prod_import_test.go" {
				scan.hasImportBan = true
			}
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
			inspectGoSeams(path, scan.seamBypass, scan.seamDeclared)
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

	if scan.hasTestUtilPkg && !scan.hasImportBan {
		findings = append(findings, projectionFinding(scenario, ws, ws.RootPath,
			"Go testutil projection is missing the production import-ban meta-test.",
			"testutil package present; no no_prod_import_test.go found",
			"api/cli workspaces with testutil include no_prod_import_test.go or an equivalent AST guard.",
			"missing production import-ban test",
			"Without a native import-ban projection, production code can accidentally ship test-only helpers.",
			"Add internal/testutil/no_prod_import_test.go to walk non-test Go files and reject imports from internal/testutil.",
			now,
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
		bypassed := bypassedSeams(scan.seamBypass, scan.seamDeclared)
		if len(bypassed) > 0 {
			labels := make([]string, 0, len(bypassed))
			for _, cat := range bypassed {
				labels = append(labels, seamAmbientLabel[cat])
			}
			findings = append(findings, mk(codeMissingInjectableSeam, ws.RootPath,
				"Production code calls a non-deterministic dependency directly and no injectable seam exists for it.",
				"bypassed (no seam declared) for: "+strings.Join(labels, ", "),
				"Time/env/HTTP/logger dependencies injected through a seam (clock, env reader, http.Doer, logger interface) so tests can substitute them.",
				"direct calls to "+strings.Join(labels, ", ")+" with no matching seam interface or package",
				"A hard-wired ambient dependency with no seam makes behavior non-deterministic and forces tests to touch the real world; it cannot be substituted.",
				"Introduce the missing seam (clock, env, http doer, or logger interface) and inject it (see seam-discovery-and-enforcement).",
			))
		}
	}

	return findings
}

// seamAmbientCallers maps an ambient package selector (pkg.Sel) to the seam
// category it bypasses when called directly in production code.
var seamAmbientCallers = map[string]map[string]string{
	"time": {"Now": "clock"},
	"os":   {"Getenv": "env", "LookupEnv": "env", "Environ": "env"},
	"http": {"DefaultClient": "http", "Get": "http", "Post": "http", "PostForm": "http", "Head": "http"},
	"slog": {"Default": "logger"},
}

// seamAmbientLabel renders a seam category for finding text.
var seamAmbientLabel = map[string]string{
	"clock":  "time.Now()",
	"env":    "os.Getenv()/LookupEnv()",
	"http":   "net/http default client",
	"logger": "package-level logger (log.*/slog.Default)",
}

// inspectGoSeams parses a production Go file and records (a) ambient calls that
// bypass a seam and (b) seam interfaces declared in the file. Using the AST
// instead of substring matching excludes ambient names that appear only in
// comments or string literals — the chief false-positive source of the old
// detector. A file inside a seam package was already excluded by the caller.
func inspectGoSeams(path string, bypass, declared map[string]bool) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil || f == nil {
		return
	}
	// The composition root (main.go) wires seams and reads config; its direct
	// ambient use is expected, not a bypass.
	isMain := filepath.Base(path) == "main.go"
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if isMain {
				break
			}
			if id, ok := x.X.(*ast.Ident); ok {
				if sels, ok := seamAmbientCallers[id.Name]; ok {
					if cat, ok := sels[x.Sel.Name]; ok {
						bypass[cat] = true
					}
				}
				// log.Print*/Fatal*/Panic* — package-level logger use.
				if id.Name == "log" && (strings.HasPrefix(x.Sel.Name, "Print") || strings.HasPrefix(x.Sel.Name, "Fatal") || strings.HasPrefix(x.Sel.Name, "Panic")) {
					bypass["logger"] = true
				}
			}
		case *ast.InterfaceType:
			classifySeamInterface(x, declared)
		}
		return true
	})
}

// classifySeamInterface marks a seam category as declared when an interface's
// method set matches a canonical seam shape (clock.Now, httpc.Doer.Do, an env
// reader, or a structured logger).
func classifySeamInterface(it *ast.InterfaceType, declared map[string]bool) {
	if it.Methods == nil {
		return
	}
	for _, m := range it.Methods.List {
		for _, name := range m.Names {
			switch name.Name {
			case "Now":
				declared["clock"] = true
			case "Do":
				declared["http"] = true
			case "Getenv", "LookupEnv", "Environ":
				declared["env"] = true
			case "Info", "Error", "Debug", "Warn", "Infof", "Errorf":
				declared["logger"] = true
			}
		}
	}
}

// seamDirCategory returns the seam category a file's directory provides, or ""
// when the file is not inside a recognized seam package.
func seamDirCategory(path string) string {
	for _, seg := range strings.Split(filepath.ToSlash(filepath.Dir(path)), "/") {
		switch seg {
		case "clock":
			return "clock"
		case "httpc", "httpx":
			return "http"
		case "envx", "envreader":
			return "env"
		case "logx", "logger", "logging":
			return "logger"
		}
	}
	return ""
}

// bypassedSeams returns the sorted seam categories that production code uses
// directly AND for which no seam is declared. A workspace that declares a seam
// (interface or package) for a category is not flagged even if some code still
// calls the ambient primitive — the seam exists to migrate toward.
func bypassedSeams(bypass, declared map[string]bool) []string {
	var out []string
	for cat := range bypass {
		if !declared[cat] {
			out = append(out, cat)
		}
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
	var hasRenderHelper bool
	directRenderFiles := map[string]bool{}
	rogue := map[string]bool{}

	walkSourceFiles(ws.RootPath, func(path string) {
		if isTSTestFile(path) {
			testFiles++
			if src := readFileString(path); importsTestingLibraryRender(src) && !documentsProviderFreeException(src) {
				directRenderFiles[path] = true
			}
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
		if strings.HasPrefix(base, "renderWithProviders.") && filepath.Base(filepath.Dir(path)) == "test-utils" {
			hasRenderHelper = true
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

	if hasTestUtils && !hasRenderHelper {
		findings = append(findings, projectionFinding(scenario, ws, filepath.Join(ws.RootPath, "src", "test-utils"),
			"UI test-utils projection is missing the canonical render helper.",
			"src/test-utils exists; no renderWithProviders helper found",
			"src/test-utils/renderWithProviders.tsx exports the canonical provider-aware render helper.",
			"missing renderWithProviders helper",
			"Component tests need one provider-aware render path so QueryClient, i18n, router, and theme setup do not drift.",
			"Add src/test-utils/renderWithProviders.tsx and re-export it from src/test-utils/index.ts.",
			now,
		))
	}
	if hasRenderHelper && len(directRenderFiles) > 0 {
		files := make([]string, 0, len(directRenderFiles))
		for path := range directRenderFiles {
			files = append(files, relTo(ws.RootPath, path))
		}
		sort.Strings(files)
		findings = append(findings, projectionFinding(scenario, ws, ws.RootPath,
			"UI tests bypass the canonical renderWithProviders helper.",
			"direct Testing Library render imports: "+strings.Join(truncateList(files, 10), ", "),
			"Component tests import renderWithProviders from src/test-utils so provider setup stays centralized.",
			"direct Testing Library render import",
			"Bypassing the canonical render helper lets QueryClient, i18n, router, and theme setup drift between tests.",
			"Replace direct Testing Library render imports with renderWithProviders from src/test-utils, or document a narrow exception with a \"provider-free-exception: <reason>\" comment in the test file.",
			now,
		))
	}

	findings = append(findings, analyzeVitestProjection(scenario, ws, now)...)

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

var testingLibraryRenderImportRe = regexp.MustCompile(`(?s)import\s*\{([^}]*)\}\s*from\s*["']@testing-library/react["']`)

// providerFreeExceptionMarker is the documented escape hatch for tests that
// genuinely must render without the canonical provider stack — e.g. a test of
// the theme/query provider itself, where wrapping it in renderWithProviders
// would double-mount the very provider under test. The marker must appear with
// a reason, e.g.:
//
//	// provider-free-exception: this test mounts ThemeProvider itself; the
//	// canonical wrapper would double-provide and fight over documentElement.
const providerFreeExceptionMarker = "provider-free-exception:"

func documentsProviderFreeException(src string) bool {
	return strings.Contains(src, providerFreeExceptionMarker)
}

func importsTestingLibraryRender(src string) bool {
	for _, m := range testingLibraryRenderImportRe.FindAllStringSubmatch(src, -1) {
		for _, part := range strings.Split(m[1], ",") {
			name := strings.TrimSpace(part)
			if strings.HasPrefix(name, "type ") {
				continue
			}
			fields := strings.Fields(name)
			if len(fields) > 0 && fields[0] == "render" {
				return true
			}
		}
	}
	return false
}

func projectionFinding(scenario string, ws Workspace, file, message, evidence, expected, observed, why, remediation, now string) Finding {
	return Finding{
		ID:           codeUnitProjectionDrift + "-" + ws.ID + "-" + fileSlug(observed),
		Scenario:     scenario,
		WorkspaceID:  ws.ID,
		Language:     ws.Language,
		Framework:    ws.Framework,
		Code:         codeUnitProjectionDrift,
		Category:     "projection",
		Severity:     codeSeverity[codeUnitProjectionDrift],
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

type projectionExpectation struct {
	coverageFloor     float64
	vitestEnv         string
	setupFiles        []string
	coverageProvider  string
	coverageReporters []string
	coverageInclude   []string
	coverageExclude   []string
	reportOnFailure   bool
}

func resolveProjectionExpectation(ws Workspace) projectionExpectation {
	expect := projectionExpectation{
		coverageFloor:     minimumCoverageForPolicyClass("react_vite_ui", unitPolicyClass{Framework: "vitest"}),
		vitestEnv:         "jsdom",
		setupFiles:        []string{"./src/test-setup.ts"},
		coverageProvider:  "v8",
		coverageReporters: []string{"json-summary", "json"},
		coverageInclude:   []string{"src/**/*.{ts,tsx}"},
		coverageExclude: []string{
			"src/**/*.test.{ts,tsx}",
			"src/**/*.spec.{ts,tsx}",
			"src/**/*.d.ts",
			"src/main.tsx",
			"src/test-setup.ts",
			"src/test-utils/**",
			"src/consts/strings.generated.ts",
			"src/i18n/locales/**",
			"src/**/generated/**",
		},
		reportOnFailure: true,
	}
	root := scenarioRootForWorkspace(ws.RootPath)
	if root == "" {
		return expect
	}
	profile, _, ok, _ := loadUnitPolicyProfile("", root, "")
	if !ok {
		return expect
	}
	for _, role := range profile.RequiredRoles {
		class, exists := profile.PolicyClasses[role.PolicyClass]
		if !exists || !pathMatches(role.Match.Path, ws.RootPath, root) {
			continue
		}
		if class.Coverage.MinimumPercent > 0 {
			expect.coverageFloor = class.Coverage.MinimumPercent
		}
		if class.Coverage.Provider != "" {
			expect.coverageProvider = class.Coverage.Provider
		}
		if len(class.Coverage.Reporters) > 0 {
			expect.coverageReporters = class.Coverage.Reporters
		}
		if class.Projection.Vitest.Environment != "" {
			expect.vitestEnv = class.Projection.Vitest.Environment
		}
		if len(class.Projection.Vitest.SetupFiles) > 0 {
			expect.setupFiles = class.Projection.Vitest.SetupFiles
		}
		return expect
	}
	return expect
}

func scenarioRootForWorkspace(root string) string {
	dir := filepath.Clean(root)
	for i := 0; i < 8 && dir != "" && dir != string(filepath.Separator); i++ {
		if fileExists(filepath.Join(dir, ".vrooli", "testing.json")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

type viteProjection struct {
	hasVitestConfig    bool
	environment        string
	setupFiles         []string
	coverageProvider   string
	coverageReporters  []string
	coverageInclude    []string
	coverageExclude    []string
	reportOnFailure    bool
	hasReportOnFailure bool
	thresholds         map[string]float64
	hasImportBanRule   bool
}

func analyzeVitestProjection(scenario string, ws Workspace, now string) []Finding {
	if !fileExists(filepath.Join(ws.RootPath, "vite.config.ts")) && !fileExists(filepath.Join(ws.RootPath, "vite.config.js")) {
		return nil
	}

	expect := resolveProjectionExpectation(ws)
	manifest, _ := loadNodeManifest(ws.RootPath)
	cfg := readFileString(filepath.Join(ws.RootPath, "vite.config.ts"))
	if cfg == "" {
		cfg = readFileString(filepath.Join(ws.RootPath, "vite.config.js"))
	}
	eslint := readFileString(filepath.Join(ws.RootPath, "eslint.config.js"))
	proj := parseViteProjection(cfg, eslint)

	var findings []Finding
	add := func(file, message, evidence, expected, observed, remediation string) {
		findings = append(findings, projectionFinding(scenario, ws, file, message, evidence, expected, observed,
			"Native config has to project the declared unit policy so scenario-local edits cannot silently weaken the test contract.",
			remediation, now))
	}

	pkgPath := filepath.Join(ws.RootPath, "package.json")
	vitePath := filepath.Join(ws.RootPath, "vite.config.ts")
	if !manifest.hasDep("vitest") {
		add(pkgPath,
			"UI policy projection is missing the Vitest dependency.",
			"vitest dependency not found",
			"package.json declares vitest as the React/Vite unit-test runner.",
			"missing vitest dependency",
			"Add vitest through Scenario Dependency Analyzer and keep the test scripts on Vitest.")
	}
	if !manifest.hasScript("test") || !strings.Contains(manifest.Scripts["test"], "vitest") {
		add(pkgPath,
			"UI policy projection is missing a Vitest test script.",
			fmt.Sprintf("test=%q", manifest.Scripts["test"]),
			"package.json scripts.test runs vitest.",
			"missing or non-Vitest test script",
			"Set scripts.test to a Vitest command such as \"vitest run\".")
	}
	if !manifest.hasScript("test:coverage") || !strings.Contains(manifest.Scripts["test:coverage"], "vitest") || !strings.Contains(manifest.Scripts["test:coverage"], "coverage") {
		add(pkgPath,
			"UI policy projection is missing a coverage test script.",
			fmt.Sprintf("test:coverage=%q", manifest.Scripts["test:coverage"]),
			"package.json scripts.test:coverage runs Vitest coverage.",
			"missing or non-Vitest coverage script",
			"Set scripts.test:coverage to a Vitest coverage command such as \"vitest run --coverage\".")
	}
	if !proj.hasVitestConfig {
		add(vitePath, "UI policy projection is missing the Vite test block.", "no test: block detected", "vite.config declares a test block.", "missing Vitest config", "Add test configuration to vite.config.")
	}
	if proj.environment != expect.vitestEnv {
		add(vitePath, "UI policy projection is missing jsdom test environment.", "environment="+expect.vitestEnv+" not detected", "Vitest test.environment is "+expect.vitestEnv+".", "missing jsdom environment", "Set test.environment to "+fmt.Sprintf("%q", expect.vitestEnv)+".")
	}
	if !containsAllSetupFiles(proj.setupFiles, expect.setupFiles) {
		add(vitePath, "UI policy projection is missing setupFiles registration.", "setupFiles="+strings.Join(expect.setupFiles, ",")+" not detected", "Vitest setupFiles includes "+strings.Join(expect.setupFiles, ", ")+".", "missing setupFiles", "Register the policy-declared setup file(s) in test.setupFiles.")
	}
	if proj.coverageProvider != expect.coverageProvider {
		add(vitePath, "UI policy projection is missing V8 coverage provider.", "coverage.provider="+expect.coverageProvider+" not detected", "Vitest coverage.provider is "+expect.coverageProvider+".", "missing V8 coverage provider", "Set coverage.provider to "+fmt.Sprintf("%q", expect.coverageProvider)+".")
	}
	if !containsAllStrings(proj.coverageReporters, expect.coverageReporters) {
		add(vitePath, "UI policy projection is missing coverage reporters.", strings.Join(expect.coverageReporters, "/")+" reporters not all detected", "Coverage reporters include "+strings.Join(expect.coverageReporters, ", ")+".", "missing coverage reporters", "Include the policy-declared reporters in coverage.reporter.")
	}
	if !containsAllStrings(proj.coverageInclude, expect.coverageInclude) {
		add(vitePath, "UI policy projection is missing an explicit source coverage include set.", "coverage.include="+strings.Join(proj.coverageInclude, ", "), "Coverage include contains "+strings.Join(expect.coverageInclude, ", ")+".", "missing source coverage include", "Set coverage.include so coverage denominators stay scoped to production source files.")
	}
	if !containsAllStrings(proj.coverageExclude, expect.coverageExclude) {
		add(vitePath, "UI policy projection is missing canonical coverage exclusions.", "coverage.exclude="+strings.Join(proj.coverageExclude, ", "), "Coverage exclude contains test scaffolding, generated files, boot files, and locale catalogs.", "missing coverage exclusions", "Restore the canonical coverage.exclude entries without weakening production-source coverage.")
	}
	if expect.reportOnFailure && (!proj.hasReportOnFailure || !proj.reportOnFailure) {
		add(vitePath, "UI policy projection is missing coverage reporting on test failure.", "coverage.reportOnFailure=true not detected", "Vitest coverage.reportOnFailure is true.", "missing coverage reportOnFailure", "Set coverage.reportOnFailure to true so failed coverage runs remain interpretable.")
	}
	for _, key := range []string{"lines", "functions", "branches", "statements"} {
		if v, ok := proj.thresholds[key]; !ok || v < expect.coverageFloor {
			add(vitePath,
				"UI policy projection weakens Vitest coverage thresholds.",
				fmt.Sprintf("%s=%.1f", key, v),
				fmt.Sprintf("Vitest coverage thresholds are at least %.1f for lines, functions, branches, and statements.", expect.coverageFloor),
				"coverage threshold below policy",
				fmt.Sprintf("Restore the threshold to %.1f or higher.", expect.coverageFloor))
		}
	}
	if !proj.hasImportBanRule {
		add(filepath.Join(ws.RootPath, "eslint.config.js"),
			"UI policy projection is missing the production import ban for test helpers.",
			"no-restricted-imports/test-utils pattern not detected",
			"ESLint forbids production imports from src/test-utils and feature mocks.",
			"missing production import ban",
			"Restore the no-restricted-imports patterns for test-utils and feature mocks.")
	}
	return findings
}

func parseViteProjection(cfg, eslint string) viteProjection {
	p := viteProjection{thresholds: map[string]float64{}}
	clean := stripJSComments(cfg)
	testBlock, ok := objectValueBlock(clean, "test")
	p.hasVitestConfig = ok
	coverageBlock, _ := objectValueBlock(testBlock, "coverage")
	if values := stringArrayPropertyValues(testBlock, "environment"); len(values) == 1 {
		p.environment = values[0]
	}
	p.setupFiles = stringArrayPropertyValues(testBlock, "setupFiles")
	if values := stringArrayPropertyValues(coverageBlock, "provider"); len(values) == 1 {
		p.coverageProvider = values[0]
	}
	p.coverageReporters = stringArrayPropertyValues(coverageBlock, "reporter")
	p.coverageInclude = stringArrayPropertyValues(coverageBlock, "include")
	p.coverageExclude = stringArrayPropertyValues(coverageBlock, "exclude")
	p.reportOnFailure, p.hasReportOnFailure = booleanProperty(coverageBlock, "reportOnFailure")
	thresholdBlock, _ := objectValueBlock(coverageBlock, "thresholds")
	for _, key := range []string{"lines", "functions", "branches", "statements"} {
		if v, ok := numericProperty(thresholdBlock, key); ok {
			p.thresholds[key] = v
		}
	}
	p.hasImportBanRule = hasESLintImportBanProjection(eslint)
	return p
}

func containsAllStrings(have, want []string) bool {
	for _, value := range want {
		if !containsString(have, value) {
			return false
		}
	}
	return true
}

func containsAllSetupFiles(have, want []string) bool {
	for _, value := range want {
		found := false
		for _, candidate := range have {
			if normalizeSetupPath(candidate) == normalizeSetupPath(value) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func normalizeSetupPath(path string) string {
	path = strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(path)), "./")
	return path
}

func hasESLintImportBanProjection(src string) bool {
	lits := stringLiterals(stripJSComments(src))
	hasRule := false
	hasTestUtils := false
	hasFeatureMocks := false
	for _, lit := range lits {
		if lit == "no-restricted-imports" {
			hasRule = true
		}
		if strings.Contains(lit, "test-utils") {
			hasTestUtils = true
		}
		if strings.Contains(lit, "features/*/mocks") {
			hasFeatureMocks = true
		}
	}
	return hasRule && hasTestUtils && hasFeatureMocks
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func stripJSComments(src string) string {
	var b strings.Builder
	inLineComment := false
	inBlockComment := false
	var quote byte
	escaped := false
	for i := 0; i < len(src); i++ {
		c := src[i]
		if inLineComment {
			if c == '\n' {
				inLineComment = false
				b.WriteByte(c)
			}
			continue
		}
		if inBlockComment {
			if c == '*' && i+1 < len(src) && src[i+1] == '/' {
				inBlockComment = false
				i++
			}
			continue
		}
		if quote != 0 {
			b.WriteByte(c)
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '/' && i+1 < len(src) {
			switch src[i+1] {
			case '/':
				inLineComment = true
				i++
				continue
			case '*':
				inBlockComment = true
				i++
				continue
			}
		}
		if c == '\'' || c == '"' || c == '`' {
			quote = c
		}
		b.WriteByte(c)
	}
	return b.String()
}

func objectValueBlock(src, key string) (string, bool) {
	i, ok := propertyValueIndex(src, key)
	if !ok {
		return "", false
	}
	i = skipSpace(src, i)
	if i >= len(src) || src[i] != '{' {
		return "", false
	}
	end := matchingDelimiter(src, i, '{', '}')
	if end <= i {
		return "", false
	}
	return src[i+1 : end], true
}

func stringPropertyEquals(src, key, want string) bool {
	values := stringArrayPropertyValues(src, key)
	return len(values) == 1 && values[0] == want
}

func stringArrayPropertyContains(src, key, wantSubstr string) bool {
	for _, value := range stringArrayPropertyValues(src, key) {
		if strings.Contains(value, wantSubstr) {
			return true
		}
	}
	return false
}

func stringArrayPropertyValues(src, key string) []string {
	i, ok := propertyValueIndex(src, key)
	if !ok {
		return nil
	}
	i = skipSpace(src, i)
	if i >= len(src) {
		return nil
	}
	if src[i] == '[' {
		end := matchingDelimiter(src, i, '[', ']')
		if end <= i {
			return nil
		}
		return stringLiterals(src[i+1 : end])
	}
	if src[i] == '\'' || src[i] == '"' || src[i] == '`' {
		value, _, ok := readStringLiteral(src, i)
		if ok {
			return []string{value}
		}
	}
	return nil
}

func numericProperty(src, key string) (float64, bool) {
	i, ok := propertyValueIndex(src, key)
	if !ok {
		return 0, false
	}
	i = skipSpace(src, i)
	start := i
	for i < len(src) && ((src[i] >= '0' && src[i] <= '9') || src[i] == '.') {
		i++
	}
	if i == start {
		return 0, false
	}
	var v float64
	if _, err := fmt.Sscanf(src[start:i], "%f", &v); err != nil {
		return 0, false
	}
	return v, true
}

func booleanProperty(src, key string) (bool, bool) {
	i, ok := propertyValueIndex(src, key)
	if !ok {
		return false, false
	}
	i = skipSpace(src, i)
	if strings.HasPrefix(src[i:], "true") {
		return true, true
	}
	if strings.HasPrefix(src[i:], "false") {
		return false, true
	}
	return false, false
}

func propertyValueIndex(src, key string) (int, bool) {
	for i := 0; i < len(src); {
		i = skipSpaceAndCommas(src, i)
		if i >= len(src) {
			return 0, false
		}
		prop, next, ok := readPropertyName(src, i)
		if !ok {
			i++
			continue
		}
		next = skipSpace(src, next)
		if next >= len(src) || src[next] != ':' {
			i = next
			continue
		}
		if prop == key {
			return next + 1, true
		}
		i = next + 1
	}
	return 0, false
}

func readPropertyName(src string, i int) (string, int, bool) {
	if i >= len(src) {
		return "", i, false
	}
	if src[i] == '\'' || src[i] == '"' || src[i] == '`' {
		value, next, ok := readStringLiteral(src, i)
		return value, next, ok
	}
	if !isIdentStart(src[i]) {
		return "", i, false
	}
	start := i
	i++
	for i < len(src) && isIdentPart(src[i]) {
		i++
	}
	return src[start:i], i, true
}

func stringLiterals(src string) []string {
	var out []string
	for i := 0; i < len(src); i++ {
		if src[i] != '\'' && src[i] != '"' && src[i] != '`' {
			continue
		}
		value, next, ok := readStringLiteral(src, i)
		if ok {
			out = append(out, value)
			i = next - 1
		}
	}
	return out
}

func readStringLiteral(src string, i int) (string, int, bool) {
	if i >= len(src) {
		return "", i, false
	}
	quote := src[i]
	if quote != '\'' && quote != '"' && quote != '`' {
		return "", i, false
	}
	var b strings.Builder
	escaped := false
	for j := i + 1; j < len(src); j++ {
		c := src[j]
		if escaped {
			b.WriteByte(c)
			escaped = false
			continue
		}
		if c == '\\' {
			escaped = true
			continue
		}
		if c == quote {
			return b.String(), j + 1, true
		}
		b.WriteByte(c)
	}
	return "", i, false
}

func matchingDelimiter(src string, start int, open, close byte) int {
	if start >= len(src) || src[start] != open {
		return -1
	}
	depth := 0
	var quote byte
	escaped := false
	for i := start; i < len(src); i++ {
		c := src[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '\'' || c == '"' || c == '`' {
			quote = c
			continue
		}
		if c == open {
			depth++
			continue
		}
		if c == close {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func skipSpace(src string, i int) int {
	for i < len(src) && (src[i] == ' ' || src[i] == '\t' || src[i] == '\n' || src[i] == '\r') {
		i++
	}
	return i
}

func skipSpaceAndCommas(src string, i int) int {
	for i < len(src) && (src[i] == ' ' || src[i] == '\t' || src[i] == '\n' || src[i] == '\r' || src[i] == ',') {
		i++
	}
	return i
}

func isIdentStart(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '_' || c == '$'
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9') || c == '-'
}
