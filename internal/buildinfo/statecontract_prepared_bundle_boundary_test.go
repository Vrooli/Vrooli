package buildinfo

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/statecontract"
)

// The CLI command owns routing and presentation. Durable bundle locators,
// progress readers, and recovery writes live in root_update_recovery.go and
// participate in the prepared-root-bundle source owner.
func TestRootUpdateHandlerDoesNotAccessPreparedRecoveryState(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "internal/cli/runtimehandlers/root_updates.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	imports := map[string]string{}
	allowedImports := map[string]bool{}
	for _, importPath := range []string{
		"bufio", "context", "encoding/json", "errors", "fmt", "io", "os", "os/signal", "path/filepath", "strings", "syscall", "time",
		"github.com/vrooli/cli-core/cliutil", "github.com/vrooli/repo-contract-go",
		"github.com/vrooli/vrooli/internal/agentscope", "github.com/vrooli/vrooli/internal/app/runtime",
		"github.com/vrooli/vrooli/internal/buildinfo", "github.com/vrooli/vrooli/internal/cliinstall",
		"github.com/vrooli/vrooli/internal/config", "github.com/vrooli/vrooli/internal/coreactivation",
		"github.com/vrooli/vrooli/internal/rootmetrics",
	} {
		allowedImports[importPath] = true
	}
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.Base(importPath)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		if name == "." {
			t.Errorf("root_updates.go uses dot import %s; recovery access must remain explicit", importPath)
		}
		imports[name] = importPath
		if !allowedImports[importPath] {
			t.Errorf("root_updates.go adds unaudited import %s; review its state access before extending the call boundary", importPath)
		}
		if importPath == "github.com/vrooli/vrooli/internal/rebaselineprogress" || importPath == "github.com/vrooli/vrooli/internal/rootdecision" {
			t.Errorf("root_updates.go imports prepared recovery state owner %s", importPath)
		}
		if importPath == "os/exec" {
			t.Error("root_updates.go may not launch subprocesses; the worker boundary is isolated and audited separately")
		}
	}
	fileIO := map[string]bool{
		"Open": true, "OpenFile": true, "ReadFile": true, "WriteFile": true,
		"Create": true, "CreateTemp": true, "Rename": true, "Remove": true,
		"RemoveAll": true, "Mkdir": true, "MkdirAll": true, "Chmod": true,
	}
	preparedReads := map[string]map[string]bool{
		"github.com/vrooli/vrooli/internal/cliinstall": {
			"VerifyPreparedBundle":  true,
			"PreparedBundleRootSHA": true,
			"RebaselineBundlePath":  true,
		},
		"github.com/vrooli/vrooli/internal/coreactivation": {
			"NewStore":   true,
			"NewStoreAt": true,
		},
	}
	selectorInventory := map[string]map[string]bool{
		"github.com/vrooli/vrooli/internal/buildinfo": {
			"PinCompiledRuntimeHome": true, "RuntimeHomeSnapshotBase64": true,
		},
		"github.com/vrooli/vrooli/internal/cliinstall": {
			"InstallPreparedBundleSameContractWithSourceChecked": true,
		},
		"github.com/vrooli/vrooli/internal/config": {"HomeDir": true},
		"github.com/vrooli/vrooli/internal/coreactivation": {
			"Enrollment": true, "InstalledWatchdogPath": true, "Record": true,
		},
		"github.com/vrooli/vrooli/internal/rootmetrics": {"Summarize": true, "Summary": true},
		"github.com/vrooli/repo-contract-go": {
			"HomeKeyBin": true, "HomeKeyState": true, "RuntimeHomeEntryPath": true,
		},
	}
	knownPaths := map[string]bool{
		"SHA256SUMS":       true,
		"progress.json":    true,
		"core-activation":  true,
		"snapshots":        true,
		"watchdog-pending": true,
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		function := fn.Name.Name
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.BasicLit:
				if node.Kind == token.STRING {
					value, err := strconv.Unquote(node.Value)
					if err == nil && knownPaths[value] {
						t.Errorf("root_updates.go embeds prepared recovery path %q in %s", value, function)
					}
					if value == "root-rebaseline" && function != "RunRootUpdateReadCommand" {
						t.Errorf("root_updates.go embeds a recovery state directory in %s", function)
					}
				}
			case *ast.SelectorExpr:
				selector := node
				base, ok := selector.X.(*ast.Ident)
				if !ok {
					return true
				}
				importPath := imports[base.Name]
				if importPath == "os" && fileIO[selector.Sel.Name] {
					t.Errorf("root_updates.go references file I/O selector os.%s in %s", selector.Sel.Name, function)
				}
				if importPath == "github.com/vrooli/vrooli/internal/config" && strings.Contains(selector.Sel.Name, "OwnedFile") {
					t.Errorf("root_updates.go references durable file helper config.%s in %s", selector.Sel.Name, function)
				}
				if preparedReads[importPath][selector.Sel.Name] {
					t.Errorf("root_updates.go references prepared recovery state through %s.%s in %s", importPath, selector.Sel.Name, function)
				}
				if known := selectorInventory[importPath]; known != nil && !known[selector.Sel.Name] {
					t.Errorf("root_updates.go adds unaudited state helper %s.%s in %s", importPath, selector.Sel.Name, function)
				}
			}
			return true
		})
	}

	patterns := statecontract.WholeSources["prepared-root-bundle"]
	contains := func(want string) bool {
		for _, pattern := range patterns {
			if pattern == want {
				return true
			}
		}
		return false
	}
	if !contains("internal/cli/runtimehandlers/root_update_recovery.go*") {
		t.Error("prepared-root-bundle owner omits the durable root update recovery owner")
	}
	if !contains("internal/app/runtime/prepared_rebaseline.go*") {
		t.Error("prepared-root-bundle owner omits the durable app-level progress and recovery lifecycle")
	}
	if contains("internal/cli/runtimehandlers/root_updates.go*") {
		t.Error("prepared-root-bundle owner still hashes command routing and presentation")
	}
	if contains("internal/cli/runtimehandlers/root_update_status.go*") {
		t.Error("prepared-root-bundle owner includes status-only enrollment/current-record reads")
	}
	for _, excluded := range []string{
		"internal/cli/runtimehandlers/root_update_admission.go*",
		"internal/cli/runtimehandlers/root_update_worker.go*",
		"internal/cli/runtimehandlers/root_update_status.go*",
	} {
		if contains(excluded) {
			t.Errorf("prepared-root-bundle owner includes unrelated helper source %s", excluded)
		}
	}
}

func TestRootDecisionLocatorAccessStaysInPreparedOwner(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	owner := filepath.Join(root, "internal/cli/runtimehandlers/root_update_recovery.go")
	violations, loads, saves, err := rootDecisionLocatorAccessAudit(root, owner)
	if err != nil {
		t.Fatal(err)
	}
	for _, violation := range violations {
		t.Error(violation)
	}
	if loads != 1 || saves != 1 {
		t.Errorf("prepared recovery owner should be the sole Load/Save route, got Load=%d Save=%d", loads, saves)
	}

	hubPath := filepath.Join(root, "internal/cli/runtimehandlers/root_rebaseline_hub.go")
	hub, err := parser.ParseFile(token.NewFileSet(), hubPath, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	var confirm *ast.FuncDecl
	for _, decl := range hub.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "confirmRootViaHub" {
			confirm = fn
			break
		}
	}
	if confirm == nil {
		t.Fatal("confirmRootViaHub missing")
	}
	loadAt, saveAt, publishAt := token.NoPos, token.NoPos, token.NoPos
	ast.Inspect(confirm.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if ident, ok := call.Fun.(*ast.Ident); ok {
			switch ident.Name {
			case "loadPreparedHubDecisionLocator":
				loadAt = call.Pos()
			case "savePreparedHubDecisionLocator":
				saveAt = call.Pos()
			}
		}
		if selector, ok := call.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "post" && len(call.Args) >= 2 {
			if basic, ok := call.Args[1].(*ast.BasicLit); ok && basic.Kind == token.STRING {
				if value, err := strconv.Unquote(basic.Value); err == nil && value == "/api/v1/internal/root-rebaseline/publish" {
					publishAt = call.Pos()
				}
			}
		}
		return true
	})
	if !loadAt.IsValid() || !saveAt.IsValid() || !publishAt.IsValid() || !(loadAt < saveAt && saveAt < publishAt) {
		t.Errorf("Hub locator flow must read, persist its exact locator, then publish; got load=%v save=%v publish=%v", loadAt, saveAt, publishAt)
	}
}

func rootDecisionLocatorAccessAudit(root, owner string) ([]string, int, int, error) {
	rootModule, err := goModulePath(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil, 0, 0, err
	}
	var violations []string
	var loads, saves int
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			topLevelNonSource := relative == "docs" || relative == "evidence"
			if path != root && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "vendor" || entry.Name() == "node_modules") || topLevelNonSource {
				return filepath.SkipDir
			}
			if path != root {
				nestedModule, err := goModulePath(filepath.Join(path, "go.mod"))
				if err == nil && nestedModule != rootModule {
					return filepath.SkipDir
				}
				if err != nil && !os.IsNotExist(err) {
					return err
				}
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		if err != nil {
			// Go snippets with template actions are not compiled package sources.
			// Valid Go files under templates are still scanned; only malformed
			// .go templates are skipped.
			if strings.Contains(filepath.ToSlash(path), "/templates/") {
				return nil
			}
			return err
		}
		imports, dotImport := rootDecisionImportAliases(file)
		if dotImport {
			violations = append(violations, fmt.Sprintf("dot import in %s bypasses the prepared locator access inventory", path))
		}
		ast.Inspect(file, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "Load" && selector.Sel.Name != "Save" {
				return true
			}
			base, ok := selector.X.(*ast.Ident)
			if !ok || imports[base.Name] != "github.com/vrooli/vrooli/internal/rootdecision" {
				return true
			}
			if filepath.Clean(path) != filepath.Clean(owner) {
				violations = append(violations, fmt.Sprintf("rootdecision.%s appears outside prepared recovery owner in %s", selector.Sel.Name, path))
			} else {
				if selector.Sel.Name == "Load" {
					loads++
				} else {
					saves++
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		return violations, loads, saves, err
	}
	return violations, loads, saves, nil
}

func rootDecisionImportAliases(file *ast.File) (map[string]string, bool) {
	const rootDecisionImport = "github.com/vrooli/vrooli/internal/rootdecision"
	imports := map[string]string{}
	dotImport := false
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		name := filepath.Base(importPath)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		if name == "." && importPath == rootDecisionImport {
			dotImport = true
		}
		if importPath == rootDecisionImport && name != "." && name != "_" {
			imports[name] = importPath
		}
	}
	return imports, dotImport
}

func TestRootDecisionAccessGuardRejectsDotImport(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", `package fixture
import . "github.com/vrooli/vrooli/internal/rootdecision"
func read(path string) { _, _ = Load(path) }
func write(path string) { _ = Save(path, nil) }
`, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	aliases, dotImport := rootDecisionImportAliases(file)
	if !dotImport {
		t.Fatal("dot-import fixture was not rejected")
	}
	if len(aliases) != 0 {
		t.Fatalf("dot import unexpectedly produced explicit aliases: %#v", aliases)
	}
}

func TestRootDecisionAccessAuditScansRootModuleScenariosAndTemplates(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/root\n\ngo 1.26\n")
	owner := filepath.Join(root, "internal/cli/runtimehandlers/root_update_recovery.go")
	writeTestFile(t, root, "internal/cli/runtimehandlers/root_update_recovery.go", `package runtimehandlers
import "github.com/vrooli/vrooli/internal/rootdecision"
func load(path string) { _, _ = rootdecision.Load(path) }
func save(path string) { _ = rootdecision.Save(path, nil) }
`)
	writeTestFile(t, root, "scenarios/example/handler.go", `package example
import rd "github.com/vrooli/vrooli/internal/rootdecision"
func read(path string) { _, _ = rd.Load(path) }
`)
	writeTestFile(t, root, "templates/example/handler.go", `package example
import . "github.com/vrooli/vrooli/internal/rootdecision"
func read(path string) { _, _ = Load(path) }
`)
	writeTestFile(t, root, "internal/example/docs/handler.go", `package docs
import rd "github.com/vrooli/vrooli/internal/rootdecision"
func read(path string) { _, _ = rd.Load(path) }
`)
	writeTestFile(t, root, "docs/handler.go", `package docs
import rd "github.com/vrooli/vrooli/internal/rootdecision"
func read(path string) { _, _ = rd.Load(path) }
`)
	writeTestFile(t, root, "evidence/handler.go", `package evidence
import rd "github.com/vrooli/vrooli/internal/rootdecision"
func read(path string) { _, _ = rd.Load(path) }
`)
	writeTestFile(t, root, "scenarios/external/go.mod", "module example.com/external\n")
	writeTestFile(t, root, "scenarios/external/handler.go", `package external
import rd "github.com/vrooli/vrooli/internal/rootdecision"
func read(path string) { _, _ = rd.Load(path) }
`)
	violations, loads, saves, err := rootDecisionLocatorAccessAudit(root, owner)
	if err != nil {
		t.Fatal(err)
	}
	if loads != 1 || saves != 1 {
		t.Fatalf("owner locator calls = %d/%d, want one Load and Save", loads, saves)
	}
	if len(violations) != 3 {
		t.Fatalf("visible scenarios/templates/nested-docs violations = %v, want findings in root-module sources; external nested module must be pruned", violations)
	}
}

func TestRootUpdateHandlerSamePackageCallGraphIsAudited(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	pattern := filepath.Join(root, "internal/cli/runtimehandlers/*.go")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatal(err)
	}
	declarations := map[string]string{}
	files := map[string]*ast.File{}
	fset := token.NewFileSet()
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		files[path] = file
		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.FuncDecl:
				declarations[decl.Name.Name] = filepath.Base(path)
			case *ast.GenDecl:
				if decl.Tok == token.VAR {
					for _, spec := range decl.Specs {
						values := spec.(*ast.ValueSpec)
						for _, name := range values.Names {
							declarations[name.Name] = filepath.Base(path)
						}
					}
				}
			}
		}
	}
	const commandFile = "root_updates.go"
	var command *ast.File
	for path, file := range files {
		if filepath.Base(path) == commandFile {
			command = file
			break
		}
	}
	if command == nil {
		t.Fatal("root_updates.go missing from production package")
	}
	allowed := map[string]string{
		"confirmRootViaHub":                   "root_rebaseline_hub.go",
		"callerCgroupFn":                      "root_updates.go",
		"callerKindFn":                        "root_updates.go",
		"installPreparedSameContractBundleFn": "root_updates.go",
		"installSameContractBundle":           "root_updates.go",
		"newRootDecisionClient":               "root_rebaseline_hub.go",
		"newPreparedRootRecovery":             "root_update_recovery.go",
		"pinnedRebaselineSource":              "root_update_admission.go",
		"pinnedRuntimeSnapshotSource":         "root_update_admission.go",
		"readPreparedRootBundle":              "root_update_recovery.go",
		"readRootUpdateStatus":                "root_update_status.go",
		"rebaselineRootFn":                    "root_updates.go",
		"retireSnapshotDecision":              "root_rebaseline_retire.go",
		"refuseAgentSession":                  "root_updates.go",
		"retainedRebaselineSource":            "root_update_recovery.go",
		"rootDecisionClientFactory":           "root_updates.go",
		"rootUpdateCommand":                   "root_updates.go",
		"rootUpdateHelpRequested":             "root_updates.go",
		"runRootRebaseline":                   "root_updates.go",
		"runRootUpdateWorker":                 "root_update_worker.go",
		"writeRebaselinePlan":                 "root_updates.go",
	}
	seen := map[string]bool{}
	ast.Inspect(command, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		ident, ok := call.Fun.(*ast.Ident)
		if !ok {
			return true
		}
		actual, local := declarations[ident.Name]
		if !local {
			return true
		}
		want, ok := allowed[ident.Name]
		if !ok {
			t.Errorf("root_updates.go calls unaudited same-package target %s from %s", ident.Name, actual)
			return true
		}
		if actual != want {
			t.Errorf("root_updates.go helper %s resolves to %s, want audited source %s", ident.Name, actual, want)
		}
		seen[ident.Name] = true
		return true
	})
	for name := range allowed {
		if !seen[name] {
			t.Errorf("root_updates.go no longer calls audited helper %s; review its boundary", name)
		}
	}
}

func TestPreparedRecoveryOwnerExcludesStatusReaders(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "internal/cli/runtimehandlers/root_update_recovery.go")
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && (fn.Name.Name == "rootUpdateStore" || fn.Name.Name == "readRootUpdateStatus") {
			t.Errorf("status-only helper %s remains in prepared recovery owner", fn.Name.Name)
		}
	}
}

func TestRuntimeRebaselineDelegatesPreparedPersistenceToOwner(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "internal/app/runtime/rebaseline.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	var rebaselineRun *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "run" || fn.Recv == nil || fn.Body == nil || len(fn.Recv.List) != 1 {
			continue
		}
		if exprName(fn.Recv.List[0].Type) == "rebaselineEffects" {
			rebaselineRun = fn
			break
		}
	}
	if rebaselineRun == nil {
		t.Fatal("rebaselineEffects.run missing")
	}
	statePathCalls, lockPathBindings := 0, 0
	statePathsStatement, lockStatement := -1, -1
	ast.Inspect(rebaselineRun.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch exprName(call.Fun) {
		case "preparedRebaselineStatePathsForHome":
			statePathCalls++
			if len(call.Args) != 1 || exprName(call.Args[0]) != "home" {
				t.Errorf("state-path owner must derive lock paths from the runtime home")
			}
			parent, ok := directParent(rebaselineRun, call)
			assign, isAssign := parent.(*ast.AssignStmt)
			if !ok || !isAssign || len(assign.Lhs) != 2 || exprName(assign.Lhs[0]) != "statePaths" || !statementIsDirectChild(rebaselineRun.Body, parent) {
				t.Errorf("state-path result must be bound to statePaths in the transition runner")
			} else {
				statePathsStatement = statementIndex(rebaselineRun.Body, parent)
			}
		case "artifactlock.AcquireWithPathContext":
			if len(call.Args) != 3 {
				return true
			}
			parent, ok := directParent(rebaselineRun, call)
			assign, isAssign := parent.(*ast.AssignStmt)
			if !ok || !isAssign || len(assign.Lhs) != 2 || exprName(assign.Lhs[0]) != "release" || !statementIsDirectChild(rebaselineRun.Body, parent) {
				t.Errorf("transition lock must be acquired in the rebaseline runner")
			} else {
				lockStatement = statementIndex(rebaselineRun.Body, parent)
			}
			resolver, ok := call.Args[2].(*ast.FuncLit)
			if !ok {
				return true
			}
			ast.Inspect(resolver.Body, func(node ast.Node) bool {
				ret, ok := node.(*ast.ReturnStmt)
				if ok && len(ret.Results) == 2 && exprName(ret.Results[0]) == "statePaths.lock" {
					lockPathBindings++
				}
				return true
			})
		}
		return true
	})
	if statePathCalls != 1 || lockPathBindings != 1 || lockStatement <= statePathsStatement {
		t.Errorf("rebaseline must derive and use the hashed transition-lock path once; state-path calls=%d lock bindings=%d", statePathCalls, lockPathBindings)
	}
	imports := map[string]string{}
	allowedImports := map[string]bool{}
	for _, importPath := range []string{
		"context", "crypto/rand", "crypto/sha256", "encoding/hex", "encoding/json", "errors", "fmt", "io", "os", "path/filepath", "runtime", "strings", "time",
		"github.com/vrooli/repo-contract-go", "github.com/vrooli/vrooli/internal/artifactlock",
		"github.com/vrooli/vrooli/internal/buildinfo", "github.com/vrooli/vrooli/internal/cliinstall",
		"github.com/vrooli/vrooli/internal/config", "github.com/vrooli/vrooli/internal/coreactivation",
		"github.com/vrooli/vrooli/internal/rebaselineprogress", "github.com/vrooli/vrooli/internal/safeguards/emergency-watchdog",
		"github.com/vrooli/vrooli/internal/safeguards/runtime-supervisor", "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1",
	} {
		allowedImports[importPath] = true
	}
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.Base(importPath)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		if name == "." {
			t.Errorf("runtime rebaseline uses dot import %s", importPath)
		}
		imports[name] = importPath
		if !allowedImports[importPath] {
			t.Errorf("runtime rebaseline adds unaudited import %s; review its persistent-state access", importPath)
		}
	}
	sensitiveSelectors := map[string]map[string]bool{
		"github.com/vrooli/vrooli/internal/buildinfo": {
			"AcquireBinaryInstallLockContext": true, "AcquireRootBundleWorkflow": true, "GuardPreparedBundle": true,
		},
		"github.com/vrooli/vrooli/internal/cliinstall":         {"AtomicInstallRebaselineLocked": true},
		"github.com/vrooli/vrooli/internal/rebaselineprogress": {"Progress": true},
	}
	fileIO := map[string]map[string]bool{
		"MkdirTemp": {"run": true},
		"RemoveAll": {"run": true},
		"ReadFile":  {"legacyPauseMatches": true},
		"Remove":    {"removeIfPresent": true},
		"Lstat":     {"retainRebaselineFile": true},
		"Open":      {"retainRebaselineFile": true},
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		callCallees := map[*ast.SelectorExpr]bool{}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
					callCallees[selector] = true
				}
			}
			return true
		})
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			base, ok := selector.X.(*ast.Ident)
			if !ok {
				return true
			}
			pkg := imports[base.Name]
			if pkg == "os" {
				if allowedFunctions, ok := fileIO[selector.Sel.Name]; ok && (!allowedFunctions[fn.Name.Name] || !callCallees[selector]) {
					t.Errorf("rebaseline.go references os.%s outside its audited file operation in %s", selector.Sel.Name, fn.Name.Name)
				}
			}
			if pkg == "github.com/vrooli/vrooli/internal/rebaselineprogress" && (selector.Sel.Name == "Read" || selector.Sel.Name == "Write") {
				t.Errorf("rebaseline.go accesses durable progress via rebaselineprogress.%s instead of the owner", selector.Sel.Name)
			}
			if pkg == "github.com/vrooli/vrooli/internal/cliinstall" && (selector.Sel.Name == "RetainRebaselineBundle" || selector.Sel.Name == "RemoveRebaselineBundle") {
				t.Errorf("rebaseline.go mutates prepared recovery bundles via cliinstall.%s instead of the owner", selector.Sel.Name)
			}
			if known := sensitiveSelectors[pkg]; known != nil && !known[selector.Sel.Name] {
				t.Errorf("rebaseline.go adds unaudited persistent-state helper %s.%s", pkg, selector.Sel.Name)
			}
			return true
		})
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ownerPath := filepath.Join(root, "internal/app/runtime/prepared_rebaseline.go")
	ownerFile, err := parser.ParseFile(token.NewFileSet(), ownerPath, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	for _, violation := range preparedProgressOrchestrationViolations(ownerFile) {
		t.Errorf("prepared progress orchestration: %s", violation)
	}
	for _, violation := range preparedRebaselinePlanEntryViolations(ownerFile) {
		t.Errorf("prepared recovery plan entry: %s", violation)
	}
	for _, violation := range runtimeRebaselineProgressBoundaryViolations(file) {
		t.Errorf("rebaseline caller boundary: %s", violation)
	}
	for _, literal := range []string{`"progress.json"`, `"watchdog-pending"`} {
		if strings.Contains(string(data), literal) {
			t.Errorf("rebaseline.go embeds durable prepared recovery locator %s outside owner", literal)
		}
	}
}

func TestRootUpdateWorkerExecCallIsPinnedToApprovedWorker(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "internal/cli/runtimehandlers/root_update_worker.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	imports := map[string]string{}
	allowedImports := map[string]bool{
		"context": true, "io": true, "os/exec": true, "time": true,
		"github.com/vrooli/vrooli/internal/hostreqkit": true,
	}
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.Base(importPath)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		if name == "." || name == "_" {
			t.Errorf("worker helper uses non-explicit import %s", importPath)
		}
		imports[name] = importPath
		if !allowedImports[importPath] {
			t.Errorf("worker helper adds unaudited import %s", importPath)
		}
	}
	var commandCalls, workerResolveCalls, runCalls int
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		calls := map[*ast.SelectorExpr]bool{}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
					calls[selector] = true
				}
			}
			return true
		})
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.BasicLit:
				if node.Kind == token.STRING {
					value, err := strconv.Unquote(node.Value)
					if err == nil && (value == "sh" || value == "bash" || value == "cat" || value == "sed") {
						t.Errorf("worker helper embeds shell or file-reading command %q", value)
					}
				}
			case *ast.SelectorExpr:
				if node.Sel.Name == "Output" || node.Sel.Name == "CombinedOutput" || node.Sel.Name == "StdoutPipe" || node.Sel.Name == "StderrPipe" {
					t.Errorf("worker helper may not capture subprocess output via %s", node.Sel.Name)
				}
				base, ok := node.X.(*ast.Ident)
				if !ok {
					return true
				}
				pkg := imports[base.Name]
				if pkg == "os/exec" {
					if node.Sel.Name != "CommandContext" || !calls[node] {
						t.Errorf("worker helper uses unaudited exec selector %s", node.Sel.Name)
					}
				}
				if pkg == "github.com/vrooli/vrooli/internal/hostreqkit" && node.Sel.Name != "InvokingUserCommand" {
					t.Errorf("worker helper uses unaudited worker resolver %s", node.Sel.Name)
				}
			case *ast.CallExpr:
				selector, ok := node.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				base, ok := selector.X.(*ast.Ident)
				if ok && base.Name == "cmd" {
					if selector.Sel.Name == "Run" {
						runCalls++
					} else {
						t.Errorf("worker helper may invoke only cmd.Run, got cmd.%s", selector.Sel.Name)
					}
					return true
				}
				if !ok {
					return true
				}
				pkg := imports[base.Name]
				if pkg == "os/exec" {
					commandCalls++
					if fn.Name.Name != "runRootUpdateWorker" || len(node.Args) != 3 || !identIs(node.Args[0], "ctx") || !identIs(node.Args[1], "name") || !callArgIsEllipsis(node, 2, "wrapped") {
						t.Errorf("exec.CommandContext must run the hostreqkit-resolved worker name and wrapped arguments")
					}
				}
				if pkg == "github.com/vrooli/vrooli/internal/hostreqkit" {
					workerResolveCalls++
					if fn.Name.Name != "runRootUpdateWorker" || len(node.Args) != 2 || !identIs(node.Args[0], "worker") || !callArgIsEllipsis(node, 1, "args") {
						t.Errorf("worker command must resolve the passed worker and its arguments")
					}
				}
			}
			return true
		})
	}
	if commandCalls != 1 || workerResolveCalls != 1 || runCalls != 1 {
		t.Errorf("expected one resolved-worker command and one cmd.Run, got exec=%d resolver=%d run=%d", commandCalls, workerResolveCalls, runCalls)
	}
}

func identIs(expr ast.Expr, name string) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == name
}

func callArgIsEllipsis(call *ast.CallExpr, index int, name string) bool {
	return call.Ellipsis.IsValid() && index < len(call.Args) && identIs(call.Args[index], name)
}

func TestPreparedRecoveryDurabilityOrdering(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "internal/app/runtime/prepared_rebaseline.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	callNames := func(name string) []string {
		t.Helper()
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != name || fn.Body == nil {
				continue
			}
			var names []string
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
					names = append(names, selector.Sel.Name)
				} else if ident, ok := call.Fun.(*ast.Ident); ok {
					names = append(names, ident.Name)
				}
				return true
			})
			return names
		}
		t.Fatalf("owner function %s missing", name)
		return nil
	}
	ordered := func(name string, required ...string) {
		t.Helper()
		names := callNames(name)
		at := -1
		for _, want := range required {
			found := -1
			for i := at + 1; i < len(names); i++ {
				if names[i] == want {
					found = i
					break
				}
			}
			if found < 0 {
				t.Fatalf("%s must call %s after %v; observed calls %v", name, want, required[:indexOf(required, want)], names)
			}
			at = found
		}
	}
	ordered("preparePreparedRebaselineRecovery", "retainBundle", "PrepareRecovery", "retainRebaselineFile", "Write")
	ordered("completePreparedRebaselineRecovery", "Write", "removeBundle")
}

func indexOf(values []string, target string) int {
	for i, value := range values {
		if value == target {
			return i
		}
	}
	return len(values)
}

func TestPreparedRootRecoveryPersistenceSitesRemainInOwner(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	const path = "internal/cli/runtimehandlers/root_update_recovery.go"
	data, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	fixture := t.TempDir()
	for _, sourcePath := range []string{
		"internal/cliinstall/companions.go",
		"internal/cliinstall/ordinary_bundle_transaction.go",
		"internal/cliinstall/rebaseline_bundle.go",
		"internal/app/runtime/prepared_rebaseline.go",
		"internal/rootdecision/pending.go",
		path,
		"internal/buildinfo/prepared_bundle_guard.go",
		"internal/buildinfo/root_activation_guard.go",
		"internal/buildinfo/source_root_pointer.go",
	} {
		source, err := os.ReadFile(filepath.Join(root, sourcePath))
		if err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, fixture, sourcePath, string(source))
	}
	digest := func() string {
		t.Helper()
		got, err := wholeSourceDigest(fixture, statecontract.WholeSources["prepared-root-bundle"])
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	baseline := digest()
	for _, tc := range []struct{ name, old, replacement string }{
		{"bundle manifest name", `"SHA256SUMS"`, `"bundle-manifest"`},
		{"Hub progress reader", `progress, err := rebaselineprogress.Read`, `progress, err := rebaselineprogress.ReadV2`},
		{"retained progress reader", `p, err := rebaselineprogress.Read`, `p, err := rebaselineprogress.ReadV2`},
		{"pending decision reader", `loadPreparedHubDecisionLocator(workflow.pendingPath)`, `loadPreparedHubDecisionLocatorV2(workflow.pendingPath)`},
		{"pending locator bytes reader", `os.ReadFile(r.pendingPath)`, `os.ReadFile(r.candidate)`},
		{"pending locator codec read", `return rootdecision.Load(path)`, `return rootdecision.LoadV2(path)`},
		{"pending locator codec write", `return rootdecision.Save(path, terms)`, `return rootdecision.SaveV2(path, terms)`},
		{"pending decision locator", `rootdecision.PendingFilename`, `"pending-v2.json"`},
		{"snapshot locator path", `filepath.Join(snapshot, rootdecision.PendingFilename)`, `filepath.Join(snapshot, "locator-v2.json")`},
		{"snapshot manifest path", `filepath.Join(snapshot, "SHA256SUMS")`, `filepath.Join(snapshot, "bundle-manifest")`},
		{"recovery locator write mode", `retainedPending, data, 0600`, `retainedPending, data, 0640`},
		{"watchdog recovery path", `"watchdog-pending"`, `"watchdog-recovery"`},
		{"companion publication", `InstallPreparedCompanionsLocked`, `InstallPreparedCompanionsAfterRecovery`},
		{"bundle source publication", `PublishPreparedBundleSourceLocked`, `PublishPreparedBundleSourceAfterRecovery`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := strings.Replace(string(data), tc.old, tc.replacement, 1)
			if changed == string(data) {
				t.Fatalf("mutation anchor %q missing", tc.old)
			}
			writeTestFile(t, fixture, path, changed)
			if digest() == baseline {
				t.Fatalf("%s mutation escaped prepared-root-bundle owner", tc.name)
			}
			writeTestFile(t, fixture, path, string(data))
		})
	}

	const lifecyclePath = "internal/app/runtime/prepared_rebaseline.go"
	lifecycleData, err := os.ReadFile(filepath.Join(root, lifecyclePath))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, old, replacement string }{
		{"shared transition lock path", `filepath.Join(transitions, "transition.lock")`, `filepath.Join(transitions, "transition-v2.lock")`},
		{"legacy pause codec version", `rebaselinePauseSchemaVersion   = 1`, `rebaselinePauseSchemaVersion   = 2`},
		{"legacy pause file reader", `data, err := os.ReadFile(path)`, `data, err := os.ReadFile(home)`},
		{"rollback pause enrollment write", `e.compareEnrollment(ctx, home, enrollment, false, true)`, `e.compareEnrollment(ctx, home, enrollment, false, false)`},
		{"rollback receipt codec", `json.MarshalIndent(receipt, "", "  ")`, `json.Marshal(receipt)`},
		{"progress path", `"progress.json"`, `"progress-v2.json"`},
		{"progress reader", `return rebaselineprogress.Read(path)`, `return rebaselineprogress.ReadV2(path)`},
		{"watchdog path", `"watchdog-pending"`, `"watchdog-recovery"`},
		{"pending watchdog reader", `os.Stat(pending)`, `os.Stat(snapshot)`},
		{"progress candidate binding", `progress.Candidate != candidateSHA`, `progress.Candidate == candidateSHA`},
		{"progress authority mode binding", `progress.Mode == "hub"`, `progress.Mode != "hub"`},
		{"progress watchdog binding", `progress.Watchdog == watchdogSHA`, `progress.Watchdog != watchdogSHA`},
		{"prepared watchdog manifest reader", `cliinstall.PreparedBundleWatchdogSHA(bundleDir, manifest)`, `cliinstall.PreparedBundleWatchdogSHA(manifest, bundleDir)`},
		{"bundle retained before recovery authority", `e.retainBundle(home, filepath.Dir(originalCandidate), bundleSHA)`, `e.retainBundle(home, filepath.Dir(originalCandidate), "other-bundle")`},
		{"recovery authority binding", `request.PrepareRecovery(ctx, RebaselineRecovery{Admission: admission, BundleDir: prepared.BundleDir})`, `request.PrepareRecovery(ctx, RebaselineRecovery{Admission: admission, BundleDir: originalCandidate})`},
		{"pending watchdog restoration", `retainRebaselineFile(pending, staged)`, `retainRebaselineFile(snapshot, staged)`},
		{"intent progress write", `rebaselineprogress.Write(progressPath, record)`, `rebaselineprogress.Write(record, progressPath)`},
		{"completion state", `record.Complete = true`, `record.Complete = false`},
		{"bundle retirement", `e.removeBundle(home, bundleSHA)`, `e.removeBundle(home, "other-bundle")`},
		{"production bundle retention wiring", `return cliinstall.RetainRebaselineBundle, cliinstall.RemoveRebaselineBundle`, `return cliinstall.RemoveRebaselineBundle, cliinstall.RetainRebaselineBundle`},
	} {
		t.Run("lifecycle/"+tc.name, func(t *testing.T) {
			changed := strings.Replace(string(lifecycleData), tc.old, tc.replacement, 1)
			if changed == string(lifecycleData) {
				t.Fatalf("mutation anchor %q missing", tc.old)
			}
			writeTestFile(t, fixture, lifecyclePath, changed)
			if digest() == baseline {
				t.Fatalf("%s mutation escaped prepared-root-bundle owner", tc.name)
			}
			writeTestFile(t, fixture, lifecyclePath, string(lifecycleData))
		})
	}

	const locatorCodecPath = "internal/rootdecision/pending.go"
	locatorCodec, err := os.ReadFile(filepath.Join(root, locatorCodecPath))
	if err != nil {
		t.Fatal(err)
	}
	decisionDigest := func() string {
		t.Helper()
		got, err := wholeSourceDigest(fixture, statecontract.WholeSources["root-rebaseline-hub-decision"])
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	decisionBaseline := decisionDigest()
	for _, tc := range []struct{ name, old, replacement string }{
		{"missing locator compatibility", `if os.IsNotExist(err)`, `if err == nil`},
		{"durable locator mode", `config.WriteOwnedFileDurable(path, data, 0600)`, `config.WriteOwnedFileDurable(path, data, 0640)`},
	} {
		t.Run("locator codec/"+tc.name, func(t *testing.T) {
			changed := strings.Replace(string(locatorCodec), tc.old, tc.replacement, 1)
			if changed == string(locatorCodec) {
				t.Fatalf("mutation anchor %q missing", tc.old)
			}
			writeTestFile(t, fixture, locatorCodecPath, changed)
			if decisionDigest() == decisionBaseline {
				t.Fatalf("%s mutation escaped root decision owner", tc.name)
			}
			writeTestFile(t, fixture, locatorCodecPath, string(locatorCodec))
		})
	}
	writeTestFile(t, fixture, "internal/rootdecision/new_reader.go", "package rootdecision\nfunc newReader() {}\n")
	if decisionDigest() == decisionBaseline {
		t.Fatal("new rootdecision production source escaped whole-source owner")
	}
}
