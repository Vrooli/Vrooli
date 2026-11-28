package validation

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	corestorage "github.com/vrooli/api-core/storage"
)

// filesystemContract is the shared owner-neutral filesystem contract checker.
// It uses Go's parser rather than grep so comments, strings, and unrelated
// identifiers cannot masquerade as filesystem operations.
type filesystemContract struct{}

func init() { register(&filesystemContract{}) }

func (filesystemContract) Name() string                    { return "filesystem.contract" }
func (filesystemContract) Kinds() []corestorage.OwnerKind  { return nil }
func (filesystemContract) Applies(ac AnalyzerContext) bool { return ac.Owner != nil }

func (filesystemContract) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	findings := make([]Finding, 0)
	for _, file := range ownerGoFiles(ac) {
		data, err := os.ReadFile(file.path)
		if err != nil {
			continue
		}
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, file.path, data, 0)
		if err != nil {
			continue
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.X == nil {
				return true
			}
			ident, identOK := selector.X.(*ast.Ident)
			if !identOK || ident.Name != "os" || !directWriterNames[selector.Sel.Name] {
				return true
			}
			pos := fset.Position(call.Pos())
			findings = append(findings, Finding{
				Code: "FILESYSTEM_DIRECT_WRITER", Severity: SeverityWarning,
				Title: "Direct filesystem writer", Message: "managed filesystem mutation bypasses the reviewed ownership-safe seam",
				Location:    fmt.Sprintf("%s:%d", file.rel, pos.Line),
				Remediation: "Use api-core/storage or the control-plane owned-write seam with explicit mode, containment, and atomic replacement.",
				Analyzer:    "filesystem.contract", CapabilityID: "filesystem_safety", FixClass: "manual",
				Metrics: map[string]float64{"line": float64(pos.Line)},
			})
			if modeArg, modeExpected := filesystemModeArgument(selector.Sel.Name, call.Args); modeExpected && !explicitModeLiteral(modeArg) {
				findings = append(findings, Finding{
					Code: "FILESYSTEM_MODE_UNPROVEN", Severity: SeverityWarning,
					Title: "Filesystem mode is not explicit", Message: "filesystem mutation does not prove its creation mode at the writer boundary",
					Location:    fmt.Sprintf("%s:%d", file.rel, pos.Line),
					Remediation: "Pass an explicit restrictive mode or use the reviewed ownership-safe writer seam.",
					Analyzer:    "filesystem.contract", CapabilityID: "filesystem_safety", FixClass: "manual",
					Metrics: map[string]float64{"line": float64(pos.Line)},
				})
			}
			for _, arg := range call.Args {
				literal, ok := arg.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					continue
				}
				value, _ := strconv.Unquote(literal.Value)
				if filepath.IsAbs(value) || strings.Contains(value, `\`) && strings.Contains(value, ":") {
					findings = append(findings, Finding{
						Code: "FILESYSTEM_NONPORTABLE_PATH", Severity: SeverityWarning,
						Title: "Non-portable filesystem path", Message: "filesystem writer contains a platform-specific absolute path",
						Location:    fmt.Sprintf("%s:%d", file.rel, pos.Line),
						Remediation: "Resolve the path through the portable storage or runtime-home contract.",
						Analyzer:    "filesystem.contract", CapabilityID: "filesystem_portability", FixClass: "manual",
						Metrics: map[string]float64{"line": float64(pos.Line)},
					})
				}
			}
			return true
		})
	}
	for _, entry := range ac.Owner.StorageEntries {
		if entry.Regenerable && entry.Budget == nil && entry.Reclaim == nil {
			findings = append(findings, Finding{
				Code: "RETENTION_AUTHORITY_MISSING", Severity: SeverityWarning,
				Title: "Retention authority missing", Message: "regenerable managed storage has no retention budget or cleanup authority",
				Location: filepath.ToSlash(ac.Owner.ManifestPath), Remediation: "Declare a bounded budget or an owner-approved reclaim provider.",
				Analyzer: "filesystem.contract", CapabilityID: "retention_governance", FixClass: "manual",
				Metrics: map[string]float64{"regenerable": 1},
			})
		}
	}
	if !ac.Owner.StorageDeclared {
		findings = append(findings, Finding{
			Code: "RETENTION_POLICY_MISSING", Severity: SeverityInfo,
			Title: "Retention policy not declared", Message: "this aggregate owner has no storage declaration; conservative defaults protect it from cleanup",
			Location: filepath.ToSlash(ac.Owner.ManifestPath), Remediation: "Declare the managed filesystem surface or explicitly record that it owns no runtime data.",
			Analyzer: "filesystem.contract", CapabilityID: "retention_governance", FixClass: "manual",
			Metrics: map[string]float64{"declared": 0},
		})
	}
	return findings, nil
}

func filesystemModeArgument(name string, args []ast.Expr) (ast.Expr, bool) {
	switch name {
	case "Create":
		return nil, true
	case "OpenFile", "WriteFile":
		if len(args) >= 3 {
			return args[2], true
		}
	case "MkdirAll":
		if len(args) >= 2 {
			return args[1], true
		}
	}
	return nil, false
}

func explicitModeLiteral(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)
	return ok && lit.Kind == token.INT && strings.TrimSpace(lit.Value) != "0"
}

var directWriterNames = map[string]bool{
	"Create": true, "OpenFile": true, "WriteFile": true, "MkdirAll": true,
	"Rename": true, "Remove": true, "RemoveAll": true, "Truncate": true,
}

type ownerGoFile struct{ path, rel string }

func ownerGoFiles(ac AnalyzerContext) []ownerGoFile {
	root := ac.ScenarioDir
	if root == "" && ac.Owner != nil {
		switch ac.Owner.Kind {
		case corestorage.OwnerPackage:
			root = filepath.Dir(ac.Owner.ManifestPath)
		case corestorage.OwnerControlPlane:
			root = filepath.Join(ac.RepoRoot, "internal")
		case corestorage.OwnerProject:
			root = ac.RepoRoot
		default:
			root = filepath.Dir(ac.Owner.ManifestPath)
		}
	}
	if ac.Owner != nil && ac.Owner.Kind == corestorage.OwnerScenario && ac.APIDir != "" {
		root = ac.APIDir
	}
	var out []ownerGoFile
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, relErr := filepath.Rel(ac.RepoRoot, path)
		if relErr == nil && !isExemptPath(rel) {
			out = append(out, ownerGoFile{path: path, rel: filepath.ToSlash(rel)})
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].rel < out[j].rel })
	return out
}
