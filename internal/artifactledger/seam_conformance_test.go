package artifactledger

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// removalPrimitives are the calls that actually delete something.
//
// The set is about effect, not about name: reading, statting, and locking are
// all fine anywhere. Only the operations that destroy an artifact have to go
// through the seam.
var removalPrimitives = map[string]bool{
	"Remove":    true,
	"RemoveAll": true,
}

// guardedSources are the directories and files that remove install-root
// artifacts. Each entry may be a directory (all of its Go files are scanned) or
// a single file.
//
// The scope is deliberately narrow. storage-manager's other cleanup providers
// remove trash, temp files, and image layers -- different roots with different
// lifecycles, none of them installed artifacts under an ownership protocol.
// Sweeping them in would force unrelated code through a seam built for a
// problem they do not have, and the exemption list would then be doing the real
// work instead of the rule.
//
// Paths are relative to this package. storage-manager lives in its own module
// and cannot be reached by an import, so it is scanned from disk -- the same
// approach the Inspect purity guard takes.
var guardedSources = []string{
	filepath.Join("..", "cliinstall"),
	filepath.Join("..", "..", "scenarios", "storage-manager", "api", "internal", "providers", "scenario_binaries_provider.go"),
}

// nonArtifactReceivers are packages whose Remove operations act on ownership
// metadata rather than on an installed artifact.
//
// artifactlease.Remove deletes a lease sidecar. A lease is a record *about* an
// artifact, not an artifact under the protocol: it has no lease of its own to
// lock, and it is only ever removed as a consequence of a guarded removal that
// already took the family lock and wrote a receipt. Routing it through Guard
// would lock the wrong subject and produce a receipt for deleting a receipt's
// subject-matter.
//
// This is a receiver exclusion rather than a function exemption on purpose:
// exempting the enclosing function would also excuse any future direct
// os.Remove added beside it.
var nonArtifactReceivers = map[string]bool{
	"artifactlease": true,
}

// seamExemptions are the call sites permitted to remove directly, keyed by
// "<file>:<function>" and carrying the reason each is not an install-root
// artifact removal.
//
// Keying by file as well as function is deliberate: a bare function name would
// exempt every method of that name in the repository, and "Apply" alone would
// have silently exempted the storage-manager reaper -- the single most
// important caller this check exists to hold.
//
// An exemption is a claim that the call cannot destroy an installed artifact.
// Adding one is a design decision; if a future exemption really means "routing
// this is inconvenient", it belongs in the seam instead.
var seamExemptions = map[string]string{
	"uninstall_plan.go:Apply": "dispatches to the Remover interface; the production implementation (fileRemover) routes through the seam, and tests inject a recorder that touches no disk",
}

// TestRemovalsGoThroughTheSeam keeps the mutation seam a seam.
//
// The ownership design rests on every destructive operation taking the artifact
// lock and re-validating its predicate under it. That guarantee is worth
// exactly as much as the number of code paths that honour it, and nothing in
// the compiler stops a new caller from reaching for os.RemoveAll directly --
// which is how the install root came to have several removers and only one
// locker in the first place.
//
// Limitation worth stating rather than hiding: this checks direct calls. A
// removal reached through a function value (cliinstall stores os.Remove in a
// field) is invisible here, and is covered instead by the ledger's own tests at
// the seam. The check is a floor, not a proof.
func TestRemovalsGoThroughTheSeam(t *testing.T) {
	scanned := 0
	for _, source := range guardedSources {
		for _, path := range goSourcesUnder(t, source) {
			scanned++
			for _, violation := range unguardedRemovals(t, path) {
				t.Errorf("%s", violation)
			}
		}
	}
	// A directory rename must not turn this suite into a no-op that still
	// reports success.
	if scanned == 0 {
		t.Fatal("no guarded source files were scanned; the seam check is not running")
	}
}

// TestSeamCheckerCatchesAnUnguardedRemoval proves the check can fail. A
// conformance test that scans real code and finds nothing is indistinguishable
// from one whose matcher is broken.
func TestSeamCheckerCatchesAnUnguardedRemoval(t *testing.T) {
	dir := t.TempDir()
	source := `package probe

import "os"

func reclaim(path string) error {
	return os.RemoveAll(path)
}
`
	path := filepath.Join(dir, "probe.go")
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}

	violations := unguardedRemovals(t, path)
	if len(violations) == 0 {
		t.Fatal("the checker found no violation in a knowingly unguarded removal")
	}
	if !strings.Contains(violations[0], "os.RemoveAll") {
		t.Fatalf("violation does not name the call: %q", violations[0])
	}
}

// TestSeamCheckerAcceptsAGuardedRemoval keeps the check from being so strict
// that the sanctioned pattern fails it.
func TestSeamCheckerAcceptsAGuardedRemoval(t *testing.T) {
	dir := t.TempDir()
	source := `package probe

import "os"

type ledger interface {
	Guard(removal any, remove func() error) error
}

func reclaim(l ledger, path string) error {
	return l.Guard(nil, func() error { return os.RemoveAll(path) })
}
`
	path := filepath.Join(dir, "probe.go")
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}

	if violations := unguardedRemovals(t, path); len(violations) != 0 {
		t.Fatalf("a removal inside Guard was flagged: %v", violations)
	}
}

// seamViolation is one direct removal, tagged with the exemption key that would
// silence it.
type seamViolation struct {
	key     string
	message string
}

// unguardedRemovals reports removal primitives called outside a Guard closure,
// with exemptions applied.
func unguardedRemovals(t *testing.T, path string) []string {
	t.Helper()
	var kept []string
	for _, violation := range allDirectRemovals(t, path) {
		if _, exempt := seamExemptions[violation.key]; exempt {
			continue
		}
		kept = append(kept, violation.message)
	}
	return kept
}

// allDirectRemovals reports every direct removal, exemptions ignored.
func allDirectRemovals(t *testing.T, path string) []seamViolation {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	// Every closure handed to a Guard call is a sanctioned region.
	type span struct{ from, to token.Pos }
	var guarded []span
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Guard" {
			return true
		}
		for _, arg := range call.Args {
			if literal, isFunc := arg.(*ast.FuncLit); isFunc {
				guarded = append(guarded, span{literal.Pos(), literal.End()})
			}
		}
		return true
	})
	inGuard := func(pos token.Pos) bool {
		for _, region := range guarded {
			if pos >= region.from && pos <= region.to {
				return true
			}
		}
		return false
	}

	var violations []seamViolation
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !removalPrimitives[selector.Sel.Name] {
				return true
			}
			receiver := renderReceiver(selector.X)
			if receiver == "" || nonArtifactReceivers[receiver] {
				return true
			}
			if inGuard(call.Pos()) {
				return true
			}
			violations = append(violations, seamViolation{
				key:     filepath.Base(path) + ":" + fn.Name.Name,
				message: describeSeamViolation(fset, call, receiver+"."+selector.Sel.Name, fn.Name.Name, path),
			})
			return true
		})
	}
	return violations
}

func describeSeamViolation(fset *token.FileSet, call *ast.CallExpr, name, enclosing, path string) string {
	return fset.Position(call.Pos()).String() +
		": " + enclosing + " calls " + name + " directly. Install-root removals must go through" +
		" artifactledger.Guard so they take the artifact lock, re-validate their predicate under it," +
		" and leave a receipt. If this call cannot touch an installed artifact, add it to seamExemptions" +
		" with the reason."
}

// goSourcesUnder expands a guarded entry into the Go files it covers.
func goSourcesUnder(t *testing.T, source string) []string {
	t.Helper()
	info, err := os.Stat(source)
	if err != nil {
		t.Fatalf("resolve guarded source %s: %v", source, err)
	}
	if !info.IsDir() {
		return []string{source}
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		t.Fatalf("read %s: %v", source, err)
	}
	var paths []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		paths = append(paths, filepath.Join(source, name))
	}
	return paths
}

// renderReceiver names the value a removal was called on.
//
// It handles a field receiver (p.files.RemoveAll) as well as a package one
// (os.RemoveAll). Missing the field case is how the first draft of this check
// silently passed over a provider that removes through an injected filesystem
// seam -- the exact shape the storage-manager reaper uses.
func renderReceiver(expr ast.Expr) string {
	switch node := expr.(type) {
	case *ast.Ident:
		return node.Name
	case *ast.SelectorExpr:
		if base := renderReceiver(node.X); base != "" {
			return base + "." + node.Sel.Name
		}
	}
	return ""
}

// TestEverySeamExemptionIsLoadBearing deletes stale exemptions before they
// become holes.
//
// An exemption that no longer matches any call site is worse than useless: it
// reads as a reviewed decision while silencing nothing, and the next time a
// function of that name appears in that file it is exempt by accident. This
// also proves the scan actually reaches the exempted files, which a passing
// conformance test on its own does not.
func TestEverySeamExemptionIsLoadBearing(t *testing.T) {
	found := map[string]bool{}
	for _, source := range guardedSources {
		for _, path := range goSourcesUnder(t, source) {
			for _, violation := range allDirectRemovals(t, path) {
				found[violation.key] = true
			}
		}
	}
	for key := range seamExemptions {
		if !found[key] {
			t.Errorf("seam exemption %q matches no call site; delete it rather than leaving a standing exception", key)
		}
	}
}
