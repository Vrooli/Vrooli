package validation

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"

	"unit-health/internal/adapterregistry"
	"unit-health/internal/adapters"
)

// analyzeArchitecture runs the static, source-fact-driven test-architecture
// checks for each workspace. It needs no execution: it reads source files under
// the workspace root and emits co-location, shared-test-utility,
// production-helper-import, and injectable-seam findings.
func analyzeArchitecture(scenario string, workspaces []Workspace, now string) []Finding {
	return analyzeArchitectureWithClosure(scenario, workspaces, now, DependencyClosure{})
}

func analyzeArchitectureWithClosure(scenario string, workspaces []Workspace, now string, closure DependencyClosure) []Finding {
	registry := adapterregistry.Default()
	var findings []Finding
	for _, ws := range workspaces {
		switch ws.Language {
		case "go":
			findings = append(findings, analyzeGoArchitectureWithClosure(scenario, ws, now, closure)...)
		case "typescript":
			if analyzer, ok := registry.Resolve(ws.AdapterID, ws.Language, ws.Framework); ok {
				findings = append(findings, analyzeAdapterArchitecture(scenario, ws, now, analyzer)...)
			}
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
	testImports    map[string]bool
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
	return analyzeGoArchitectureWithClosure(scenario, ws, now, DependencyClosure{})
}

func analyzeGoArchitectureWithClosure(scenario string, ws Workspace, now string, closure DependencyClosure) []Finding {
	modulePath := goModulePath(ws.RootPath)
	scan := goWorkspaceScan{seamBypass: map[string]bool{}, seamDeclared: map[string]bool{}, rogueTestDirs: map[string]bool{}, testImports: map[string]bool{}}

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
			for _, imp := range goImports(path) {
				scan.testImports[imp] = true
			}
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
				// Temporal flow replay helpers are generated under flow/generated
				// and are only invoked by their sibling flow_test.go. Flow Verifier
				// deliberately emits them there so they can load the generated
				// artifact beside the runtime table. They are verification bridges,
				// not application behavior, despite Go requiring a .go suffix.
				if isTestHelperImport(imp, modulePath) && !isGeneratedFlowReplay(path) {
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

	registry, hasRegistry := loadCompanionRegistry(ws.RootPath)
	hasAdoptedCompanion := hasRegistry && importsRegisteredCompanion(scan.testImports, registry)
	if scan.testFiles >= 3 && !scan.hasTestUtilPkg && !hasAdoptedCompanion {
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

	if hasRegistry {
		findings = append(findings, analyzeCompanionDeclarations(scenario, ws, now, registry, closure)...)
	}

	return findings
}

func isGeneratedFlowReplay(path string) bool {
	clean := filepath.ToSlash(path)
	return strings.HasSuffix(clean, "/flow/generated/replay.go")
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

func analyzeAdapterArchitecture(scenario string, ws Workspace, now string, analyzer adapters.Analyzer) []Finding {
	var findings []Finding
	drifts := analyzer.AnalyzeArchitecture(ws.RootPath)
	for _, drift := range drifts {
		if drift.Code == codeUnitProjectionDrift {
			findings = append(findings, projectionFinding(scenario, ws, drift.File, drift.Message, drift.Evidence, drift.Expected, drift.Observed, drift.WhyItMatters, drift.Remediation, now))
			continue
		}
		findings = append(findings, Finding{
			ID:           drift.Code + "-" + ws.ID,
			Scenario:     scenario,
			WorkspaceID:  ws.ID,
			Language:     "typescript",
			Framework:    ws.Framework,
			Code:         drift.Code,
			Category:     "architecture",
			Severity:     codeSeverity[drift.Code],
			FilePath:     drift.File,
			Message:      drift.Message,
			Evidence:     drift.Evidence,
			Expected:     drift.Expected,
			Observed:     drift.Observed,
			WhyItMatters: drift.WhyItMatters,
			Remediation:  drift.Remediation,
			CreatedAt:    now,
		})
	}
	findings = append(findings, analyzeAdapterProjection(scenario, ws, now, analyzer)...)
	return findings
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

func resolveProjectionPolicy(ws Workspace) adapters.ProjectionPolicy {
	analyzer, ok := adapterregistry.Default().Resolve(ws.AdapterID, ws.Language, ws.Framework)
	if !ok {
		return adapters.ProjectionPolicy{}
	}
	policy := analyzer.DefaultProjectionPolicy()
	root := scenarioRootForWorkspace(ws.RootPath)
	if root == "" {
		return policy
	}
	profile, _, ok, _ := loadUnitPolicyProfile("", root, "")
	if !ok {
		return policy
	}
	for _, role := range profile.RequiredRoles {
		class, exists := profile.PolicyClasses[role.PolicyClass]
		if !exists || !pathMatches(role.Match.Path, ws.RootPath, root) {
			continue
		}
		if class.Coverage.MinimumPercent > 0 {
			policy.CoverageFloor = class.Coverage.MinimumPercent
		}
		if class.Coverage.Provider != "" {
			policy.CoverageProvider = class.Coverage.Provider
		}
		if len(class.Coverage.Reporters) > 0 {
			policy.CoverageReporters = class.Coverage.Reporters
		}
		settings := analyzer.ProjectionPolicyFromSettings(class.Projection.Settings)
		if settings.Environment != "" {
			policy.Environment = settings.Environment
		}
		if len(settings.SetupFiles) > 0 {
			policy.SetupFiles = settings.SetupFiles
		}
		return policy
	}
	return policy
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

func analyzeAdapterProjection(scenario string, ws Workspace, now string, analyzer adapters.Analyzer) []Finding {
	policy := resolveProjectionPolicy(ws)
	drifts := analyzer.AnalyzeProjection(adapters.ProjectionInput{
		RootPath: ws.RootPath,
		Policy:   policy,
	})
	findings := make([]Finding, 0, len(drifts))
	for _, drift := range drifts {
		findings = append(findings, projectionFinding(scenario, ws, drift.File, drift.Message, drift.Evidence, drift.Expected, drift.Observed, "Native config has to project the declared unit policy so scenario-local edits cannot silently weaken the test contract.", drift.Remediation, now))
	}
	return findings
}
