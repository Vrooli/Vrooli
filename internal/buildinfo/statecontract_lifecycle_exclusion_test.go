package buildinfo

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	pathpkg "path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/statecontract"
)

func TestScenarioLocatorsHaveNoRootControlPlaneCallers(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	const lifecycleImport = "github.com/vrooli/vrooli/internal/lifecycle"
	locators := map[string]bool{"LocateTestGenieCLI": true, "LocateScenarioCompletenessCLI": true}
	for dir := range rootControlPlanePackageDirs(t, root) {
		files, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range files {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			filename := filepath.Join(dir, entry.Name())
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, filename, nil, parser.SkipObjectResolution)
			if err != nil {
				t.Fatal(err)
			}
			imports := map[string]string{}
			dotLifecycleImport := false
			for _, spec := range file.Imports {
				importPath, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					t.Fatal(err)
				}
				name := pathpkg.Base(importPath)
				if spec.Name != nil {
					name = spec.Name.Name
				}
				imports[name] = importPath
				dotLifecycleImport = dotLifecycleImport || name == "." && importPath == lifecycleImport
			}
			ast.Inspect(file, func(node ast.Node) bool {
				switch node := node.(type) {
				case *ast.SelectorExpr:
					if receiver, ok := node.X.(*ast.Ident); ok && imports[receiver.Name] == lifecycleImport && locators[node.Sel.Name] {
						t.Errorf("root control-plane source %s calls lifecycle.%s", filename, node.Sel.Name)
					}
				case *ast.Ident:
					if dotLifecycleImport && locators[node.Name] {
						t.Errorf("root control-plane source %s refers to dot-imported lifecycle.%s", filename, node.Name)
					}
				}
				return true
			})
		}
	}
}

func TestExcludedScenarioLocatorRemainsStateless(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	path := "internal/lifecycle/scenario_locator.go"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join(root, path), nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	allowedImports := map[string]map[string]bool{
		"fmt":                                    {"Errorf": true},
		"os":                                     {"Getenv": true, "Stat": true},
		"path/filepath":                          {"Join": true},
		"strings":                                {"TrimSpace": true},
		"github.com/vrooli/api-core/scenariocli": {"ResolveExecutable": true},
		"github.com/vrooli/repo-contract-go":     {"RuntimeHomeEntryPath": true},
		"github.com/vrooli/vrooli/internal/config": {"HomeDir": true},
	}
	imports := map[string]string{}
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		name := pathpkg.Base(importPath)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		if _, ok := allowedImports[importPath]; !ok {
			t.Errorf("excluded locator imports unreviewed package %q", importPath)
		}
		imports[name] = importPath
	}
	for _, declaration := range file.Decls {
		switch declaration := declaration.(type) {
		case *ast.GenDecl:
			if declaration.Tok == token.VAR {
				t.Error("excluded locator declares package state")
			}
		case *ast.FuncDecl:
			if declaration.Name.Name == "init" {
				t.Error("excluded locator declares init behavior")
			}
			ast.Inspect(declaration.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				switch function := call.Fun.(type) {
				case *ast.SelectorExpr:
					receiver, ok := function.X.(*ast.Ident)
					if !ok {
						t.Errorf("excluded locator has an unreviewed selector call at %s", fset.Position(call.Pos()))
						return true
					}
					importPath := imports[receiver.Name]
					localMethod := receiver.Name == "info" && (function.Sel.Name == "IsDir" || function.Sel.Name == "Mode")
					if !localMethod && !allowedImports[importPath][function.Sel.Name] {
						t.Errorf("excluded locator calls unreviewed %s.%s at %s", importPath, function.Sel.Name, fset.Position(call.Pos()))
					}
				case *ast.Ident:
					if function.Name != "isExecutable" {
						t.Errorf("excluded locator calls unreviewed local or builtin %s at %s", function.Name, fset.Position(call.Pos()))
					}
				default:
					t.Errorf("excluded locator has an unreviewed call at %s", fset.Position(call.Pos()))
				}
				return true
			})
		}
	}
}

func rootControlPlanePackageDirs(t *testing.T, root string) map[string]bool {
	t.Helper()
	dirs := map[string]bool{}
	targets := []struct{ goos, goarch string }{{"", ""}, {"linux", "amd64"}, {"linux", "arm64"}, {"darwin", "amd64"}, {"darwin", "arm64"}, {"windows", "amd64"}, {"windows", "arm64"}}
	for _, target := range targets {
		cmd := exec.Command("go", "list", "-deps", "-f", "{{if not .Standard}}{{.Dir}}{{end}}", "./cmd/vrooli", "./cmd/vrooli-watchdog")
		cmd.Dir = root
		if target.goos != "" {
			cmd.Env = append(os.Environ(), "GOOS="+target.goos, "GOARCH="+target.goarch, "CGO_ENABLED=0")
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go list root control-plane dependencies for %s/%s: %v: %s", target.goos, target.goarch, err, out)
		}
		for _, dir := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if dir != "" {
				dirs[dir] = true
			}
		}
	}
	return dirs
}

func TestLifecycleCacheOwnerExcludesUnreferencedScenarioLocator(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	locatorPath := "internal/lifecycle/scenario_locator.go"
	locator, err := os.ReadFile(filepath.Join(root, locatorPath))
	if err != nil {
		t.Fatal(err)
	}
	cachePath := "internal/lifecycle/closure_cache.go"
	cache, err := os.ReadFile(filepath.Join(root, cachePath))
	if err != nil {
		t.Fatal(err)
	}
	fixture := t.TempDir()
	writeTestFile(t, fixture, locatorPath, string(locator))
	writeTestFile(t, fixture, cachePath, string(cache))
	digest := func() string {
		t.Helper()
		got, err := wholeSourceDigest(fixture, statecontract.WholeSources["lifecycle-cache"])
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	baseline := digest()

	// The durable closure-cache schema/version remains covered by the owner.
	cacheMutation := strings.Replace(string(cache), "const closureCacheVersion = 3", "const closureCacheVersion = 4", 1)
	if cacheMutation == string(cache) {
		t.Fatal("closure-cache version mutation anchor missing")
	}
	writeTestFile(t, fixture, cachePath, cacheMutation)
	if digest() == baseline {
		t.Fatal("persistent closure-cache version change escaped lifecycle owner")
	}

	// These executable-path helpers are outside the root compatibility path;
	// changing their diagnostics cannot affect records that a prior release must read.
	locatorMutation := strings.Replace(string(locator), "test-genie CLI not found", "test-genie executable not found", 1)
	if locatorMutation == string(locator) {
		t.Fatal("scenario locator mutation anchor missing")
	}
	writeTestFile(t, fixture, cachePath, string(cache))
	writeTestFile(t, fixture, locatorPath, locatorMutation)
	if got := digest(); got != baseline {
		t.Fatalf("nonpersistent scenario locator edit changed lifecycle owner digest: got %s, want %s", got, baseline)
	}
}

func TestWholeSourceExclusionsAreExactAndMustMatchSelectedSources(t *testing.T) {
	fixture := t.TempDir()
	writeTestFile(t, fixture, "internal/lifecycle/fixture.go", "package lifecycle\nfunc retained() {}\n")
	for _, exclusion := range []string{
		"!../outside.go",
		"!/absolute/file.go",
		"!internal/lifecycle/*.go",
		"!internal/lifecycle/missing.go",
		"!internal/lifecycle/fixture.go",
	} {
		t.Run(exclusion, func(t *testing.T) {
			if _, err := wholeSourceDigest(fixture, []string{"internal/lifecycle", exclusion}); err == nil {
				t.Fatalf("unsafe or unmatched exclusion %q unexpectedly qualified", exclusion)
			}
		})
	}
}
