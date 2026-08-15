package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func init() {
	register(&isoRoutedSeams{})
	register(&isoFileRoutedSeams{})
	register(&isoUnverified{})
}

// The four api-core seams whose presence, taken together, statically PROVE that
// a Go scenario's destructive E2E playbooks land in an isolated test pool rather
// than the real database. Each is (import path, exported function).
//
// Static proof of all four ⟹ the routed path is eligible ⟹ test-genie installs
// a test pool on the live process ⟹ isolation by construction. Any missing seam
// means the scenario silently falls back to its real DB during mutating
// playbooks — exactly the real-data risk storage-manager exists to close.
var isoSeams = []isoSeam{
	{
		key: "database.Open", importPath: "github.com/vrooli/api-core/database", fn: "Open",
		why: "routes every connection through *database.RoutedDB",
	},
	{
		key: "database.EnsureSchemas", importPath: "github.com/vrooli/api-core/database", fn: "EnsureSchemas",
		why: "creates the per-domain schema in whichever pool is active (test or live)",
	},
	{
		key: "apihttp.TestModeMiddleware", importPath: "github.com/vrooli/api-core/apihttp", fn: "TestModeMiddleware",
		why: "reads the X-Vrooli-Test-Mode header and marks the request context for routing",
	},
	{
		key: "devrouting.Register", importPath: "github.com/vrooli/api-core/devrouting", fn: "Register",
		why: "exposes the dev-only RoutingService test-genie calls to install the test pool without a restart",
	},
}

type isoSeam struct {
	key        string
	importPath string
	fn         string
	why        string
}

// isoRoutedSeams emits ROUTED_SEAMS_UNWIRED when a Go scenario with a relational
// store does not wire all four routed-isolation seams. This is the L2 safety
// gate: an unwired scenario cannot be proven isolation-safe, so test-genie
// fail-closes its destructive playbooks.
type isoRoutedSeams struct{}

func (isoRoutedSeams) Name() string { return "isolation.routed-seams" }

func (isoRoutedSeams) Applies(ac AnalyzerContext) bool {
	// Only Go SQL scenarios install a routed test-DB pool. A Go scenario with
	// only Redis/Qdrant has no relational pool to route, and a non-Go scenario
	// is handled by isoUnverified.
	return ac.IsGo() && ac.HasRelationalStore() && ac.APIDir != ""
}

func (isoRoutedSeams) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	present := map[string]bool{}
	for _, gf := range CollectGoFiles(ac) {
		isoCollectSeams(gf.AbsPath, present)
	}

	var missing []isoSeam
	for _, seam := range isoSeams {
		if !present[seam.key] {
			missing = append(missing, seam)
		}
	}
	if len(missing) == 0 {
		return nil, nil
	}

	var b strings.Builder
	b.WriteString("Test-isolation is NOT statically proven: destructive E2E playbooks would run against this scenario's REAL database, risking production data.\n")
	b.WriteString("Missing routed-isolation seam(s):\n")
	for _, seam := range missing {
		fmt.Fprintf(&b, "  - %s — %s\n", seam.key, seam.why)
	}
	b.WriteString("Because isolation cannot be proven, test-genie will fail-closed and refuse this scenario's destructive playbooks until every seam is wired.")

	remediation := "Wire the missing seam(s) in api/main.go: open the DB via database.Open (returns *RoutedDB), call database.EnsureSchemas(ctx, db.Primary(), modules.AllSchemas()...), wrap the handler with apihttp.TestModeMiddleware, and devrouting.RegisterWithFileRoots(rootMux, db, roots). See scenarios/storage-manager/docs/concepts/test-isolation-contract.md. (Autofixable: run `storage-manager` apply-fix for ROUTED_SEAMS_UNWIRED once the registry is wired.)"

	return []Finding{{
		Code:        "ROUTED_SEAMS_UNWIRED",
		Severity:    SeverityError,
		Title:       "Routed test-isolation seams unwired",
		Message:     b.String(),
		Location:    isoMainLocation(ac),
		Remediation: remediation,
		Analyzer:    "isolation.routed-seams",
	}}, nil
}

// isoCollectSeams parses one Go file and records which routed seams it calls,
// resolving import aliases so a renamed import (e.g. `db "…/api-core/database"`)
// is still recognized. A parse error is non-fatal: the file simply contributes
// no seam evidence.
func isoCollectSeams(absPath string, present map[string]bool) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, absPath, nil, parser.SkipObjectResolution)
	if err != nil {
		return
	}
	// local import identifier -> import path
	aliasToPath := map[string]string{}
	for _, imp := range file.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		name := importLocalName(imp, path)
		aliasToPath[name] = path
	}
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkgIdent, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		path := aliasToPath[pkgIdent.Name]
		if path == "" {
			return true
		}
		for _, seam := range isoSeams {
			if path == seam.importPath && sel.Sel.Name == seam.fn {
				present[seam.key] = true
			}
		}
		// Some Go services deliberately own a richer startup schema authority
		// (migrations and optional seed handling) instead of calling the
		// api-core helper directly. A qualified persistence.ApplySchema call
		// still proves the active RoutedDB is used for schema setup; recognize
		// that equivalent without treating an arbitrary function name as proof.
		if sel.Sel.Name == "ApplySchema" && strings.HasSuffix(path, "/persistence") {
			present["database.EnsureSchemas"] = true
		}
		// The combined registration is the stricter successor to the
		// database-only seam: it mounts the same RoutingService plus leased
		// file roots, so it satisfies the DB registration proof as well.
		if path == "github.com/vrooli/api-core/devrouting" && (sel.Sel.Name == "RegisterWithFileRoots" || sel.Sel.Name == "RegisterWithFileRootsService") {
			present["devrouting.Register"] = true
		}
		return true
	})
}

// importLocalName returns the identifier a Go file uses to reference an import:
// the explicit alias when present, otherwise the package's importable name,
// which defaults to the last path segment.
func importLocalName(imp *ast.ImportSpec, path string) string {
	if imp.Name != nil && imp.Name.Name != "" && imp.Name.Name != "_" && imp.Name.Name != "." {
		return imp.Name.Name
	}
	seg := path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		seg = path[idx+1:]
	}
	return seg
}

// isoMainLocation points the finding at api/main.go when present (where the
// seams are wired), else the api directory.
func isoMainLocation(ac AnalyzerContext) string {
	for _, gf := range CollectGoFiles(ac) {
		if strings.HasSuffix(gf.RelPath, "api/main.go") {
			return gf.RelPath
		}
	}
	return "api"
}

// isoFileRoutedSeams verifies the corresponding file-storage leg for Go APIs
// that persist files. It intentionally requires both construction of
// RoutedRoots and registration through RegisterWithFileRoots: either fact by
// itself cannot route a live test request safely.
type isoFileRoutedSeams struct{}

func (isoFileRoutedSeams) Name() string { return "isolation.file-routed-seams" }

func (isoFileRoutedSeams) Applies(ac AnalyzerContext) bool {
	if !ac.IsGo() || ac.APIDir == "" {
		return false
	}
	// Observer-mode suites never install or execute destructive playbooks. The
	// scenario may still write its own durable evidence and therefore remains
	// subject to storage census/schema checks, but file test-isolation seams are
	// not an applicable safety gate when no mutating E2E path exists.
	if isoObserverOnly(ac) {
		return false
	}
	for _, engine := range ac.Engines {
		if engine == EngineFile {
			return true
		}
	}
	return isoUsesFilePersistence(ac)
}

func isoObserverOnly(ac AnalyzerContext) bool {
	registry := ReadFile(filepath.Join(ac.ScenarioDir, "bas", "registry.json"))
	if strings.Contains(registry, `"execution_mode": "observer"`) {
		return true
	}
	// test-genie generates a metadata-only registry when a scenario declares no
	// BAS workflows. That is equivalent to observer-only for this analyzer: no
	// mutating browser path exists that could write an application file. Keep
	// malformed or missing registries fail-closed rather than inferring safety.
	var document map[string]json.RawMessage
	if strings.TrimSpace(registry) == "" || json.Unmarshal([]byte(registry), &document) != nil {
		return false
	}
	for _, key := range []string{"workflows", "cases", "flows"} {
		value, present := document[key]
		if !present {
			continue
		}
		var entries []json.RawMessage
		if json.Unmarshal(value, &entries) != nil || len(entries) > 0 {
			return false
		}
	}
	return true
}

// isoUsesFilePersistence catches Go APIs whose storage declaration is absent
// or incomplete. It requires an actual application file-content operation;
// importing os (or merely creating the directory that holds an already-routed
// SQLite database) is not a separate file-storage surface. Validation evidence
// writers are excluded because coverage artifacts are owned by the validation
// system, not scenario application data.
func isoUsesFilePersistence(ac AnalyzerContext) bool {
	for _, gf := range CollectGoFiles(ac) {
		if strings.Contains(filepath.ToSlash(gf.RelPath), "/internal/artifacts/") {
			continue
		}
		// Signing keys are durable security identity, not request-scoped
		// application files. They are already resolved through the storage seam
		// and must remain stable across test leases so JWKS consumers can verify
		// tokens issued before and during a test run.
		if strings.Contains(filepath.ToSlash(gf.RelPath), "/internal/authcrypto/") {
			continue
		}
		// Bridge control-plane keys are durable security identity, not
		// request-scoped application files. They are already resolved through
		// the storage seam and must remain stable across test leases; routing
		// them through a test lease would rotate the identity and invalidate
		// every pinned node.
		if strings.Contains(filepath.ToSlash(gf.RelPath), "/internal/cpkeys/") {
			continue
		}
		// Bridge SSH onboarding state (the owner-only keypair and pinned
		// known_hosts) is likewise durable trust state resolved through the
		// storage seam. It is not a request-scoped application file and must
		// survive test leases unchanged.
		if strings.Contains(filepath.ToSlash(gf.RelPath), "/internal/onboard/ssh/") {
			continue
		}
		// Onboarding's artifact builder writes only to an os.MkdirTemp staging
		// directory whose lifetime ends with the build. It is not scenario state
		// and cannot be routed into a request test lease.
		if strings.HasSuffix(filepath.ToSlash(gf.RelPath), "/internal/onboard/artifact_builder.go") {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, gf.AbsPath, nil, parser.SkipObjectResolution)
		if err != nil {
			continue
		}
		aliases := map[string]string{}
		for _, imp := range file.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err == nil {
				aliases[importLocalName(imp, path)] = path
			}
		}
		found := false
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			if aliases[pkg.Name] == "github.com/vrooli/api-core/filerouting" {
				found = true
				return false
			}
			if aliases[pkg.Name] == "os" {
				switch sel.Sel.Name {
				case "WriteFile", "Create", "OpenFile", "Rename", "Truncate":
					found = true
					return false
				}
			}
			return true
		})
		if found {
			return true
		}
	}
	return false
}

func (isoFileRoutedSeams) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	var hasRoots, hasContextPick, hasRegistration bool
	for _, gf := range CollectGoFiles(ac) {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, gf.AbsPath, nil, parser.SkipObjectResolution)
		if err != nil {
			continue
		}
		aliases := map[string]string{}
		for _, imp := range file.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err == nil {
				aliases[importLocalName(imp, path)] = path
			}
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if sel.Sel.Name == "Pick" && isoCallUsesContext(call) {
				hasContextPick = true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			switch aliases[pkg.Name] {
			case "github.com/vrooli/api-core/filerouting":
				hasRoots = hasRoots || sel.Sel.Name == "New"
			case "github.com/vrooli/api-core/devrouting":
				hasRegistration = hasRegistration || sel.Sel.Name == "RegisterWithFileRoots" || sel.Sel.Name == "RegisterWithFileRootsService" || sel.Sel.Name == "RegisterFileRoots"
			}
			return true
		})
	}
	if hasRoots && hasContextPick && hasRegistration {
		return nil, nil
	}
	missing := make([]string, 0, 3)
	if !hasRoots {
		missing = append(missing, "filerouting.New(primaryPaths)")
	}
	if !hasContextPick {
		missing = append(missing, "RoutedRoots.Pick(ctx, class) in file-store paths")
	}
	if !hasRegistration {
		missing = append(missing, "devrouting.RegisterWithFileRoots(rootMux, db, roots)")
	}
	return []Finding{{
		Code:        "FILE_ROUTED_SEAMS_UNWIRED",
		Severity:    SeverityError,
		Title:       "File test-isolation seams unwired",
		Message:     "File-persisting API lacks required routed file seams: " + strings.Join(missing, ", ") + ". Mutating E2E cannot be proven safe.",
		Location:    isoMainLocation(ac),
		Remediation: "Construct filerouting.RoutedRoots from startup storage paths, mount devrouting.RegisterFileRoots (or RegisterWithFileRoots when SQL is also present), and route file-store paths through RoutedRoots.Pick(ctx, class). See scenarios/storage-manager/docs/concepts/test-isolation-contract.md.",
		Analyzer:    "isolation.file-routed-seams",
	}}, nil
}

// isoCallUsesContext accepts a Pick call only when its first argument is
// visibly a context-bearing value. This avoids treating unrelated Pick()
// methods as proof and makes the intended per-request routing surface clear.
func isoCallUsesContext(call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}
	ident, ok := call.Args[0].(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "ctx" || ident.Name == "context"
}

// isoUnverified emits STORAGE_ISOLATION_UNVERIFIED when a NON-Go scenario
// persists data. The four routed seams are Go/api-core-specific, so isolation
// cannot be statically proven for a non-Go API. Per the plan this is
// advisory-but-visible (a WARNING, not an error rung) — it never silently
// passes, and it drives the same fail-closed playbooks gate as an unwired Go
// scenario until an equivalent non-Go mechanism exists.
type isoUnverified struct{}

func (isoUnverified) Name() string { return "isolation.unverified-nongo" }

func (isoUnverified) Applies(ac AnalyzerContext) bool {
	// A non-Go API surface that persists data: nothing to verify the isolation
	// of a stateless surface, so require at least one declared engine.
	return ac.APIDir != "" && !ac.IsGo() && len(ac.Engines) > 0
}

func (isoUnverified) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	lang := ac.Language
	if lang == "" {
		lang = "non-Go (undetermined)"
	}
	engines := make([]string, 0, len(ac.Engines))
	for _, e := range ac.Engines {
		engines = append(engines, string(e))
	}
	sort.Strings(engines)
	msg := fmt.Sprintf(
		"This scenario's API surface is %s and persists data (%s), but test-isolation is proven only for Go scenarios via the api-core routed-DB seams. Isolation CANNOT be statically verified here, so destructive playbooks could hit the real datastore. Test-genie treats unverified isolation as fail-closed and will refuse destructive playbooks.",
		lang, strings.Join(engines, ", "),
	)
	return []Finding{{
		Code:        "STORAGE_ISOLATION_UNVERIFIED",
		Severity:    SeverityWarning,
		Title:       "Test-isolation unverified (non-Go API)",
		Message:     msg,
		Location:    "api",
		Remediation: "Port the API surface to Go and wire the routed SQL and file seams (database.Open/EnsureSchemas, apihttp.TestModeMiddleware, devrouting.RegisterWithFileRoots, context-aware RoutedRoots), or provide an equivalent test-pool routing mechanism. Until then, declare read-only playbooks only. See scenarios/storage-manager/docs/concepts/test-isolation-contract.md.",
		Analyzer:    "isolation.unverified-nongo",
	}}, nil
}
