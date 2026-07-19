// Package archtest holds permanent architecture guardrails for swarm-manager.
//
// The spawn-boundary guardrail ensures a target-bound domain package never
// spawns or continues an Agent Manager Run directly. Programmatic work starts
// a declared workflow through the typed workflow boundary; interactive,
// human-led conversation uses the agentsessions boundary.
//
// The test is AST-based (it inspects real call expressions, not text), so it is
// robust to comments, string literals, and formatting. It fails on any NEW direct
// spawn/continuation call in a guarded package, and it also fails when a
// documented legacy survivor disappears — forcing the allowlist to shrink as the
// cutover completes rather than silently rotting.
package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// guardedPackages are the target-bound domain packages that must not spawn or
// continue Agent Manager runs directly. Relative to this test's directory
// (internal/archtest).
var guardedPackages = []string{
	"../backlog",
	"../workshop",
	"../execution",
	"../review",
	"../initiativereview",
}

// spawnMethods are the Agent Manager spawn/continuation seam method names. A call
// to any of them from a guarded package is a boundary violation: these launch or
// resume an autonomous agent run, which must instead use a declared workflow.
// (ApproveRun / StopRun / GetRunState are run-management, not launch, and are not
// listed — they do not create autonomous work.)
var spawnMethods = map[string]bool{
	"SpawnBacklog":    true,
	"SpawnInitiative": true,
	"SpawnSession":    true,
	"ContinueRun":     true,
}

// spawnSite identifies a direct-spawn call by the file basename that contains it
// and the seam method it calls. File+method (not line) granularity keeps the
// allowlist stable across unrelated edits while still pinning each survivor to a
// concrete location.
type spawnSite struct {
	file   string
	method string
}

// allowedLegacySpawns is the closed allowlist of direct-spawn calls permitted in
// the guarded domain packages. It is EMPTY — the Phase 6 reroutes landed and the
// Phase 9 cutover deleted every legacy spawn site — and it must stay empty: the
// boundary is fully closed, and every new autonomous launch is expressed as a
// declared workflow instead. The guardrail fails on any spawn call that is
// not listed here (a regression) and on any listed entry with no matching call
// (a stale exception).
var allowedLegacySpawns = map[spawnSite]string{}

// [REQ:REQ-P0-009-OPERATION-SPAWN-BOUNDARY]
func TestNoDirectAgentSpawnInDomainPackages(t *testing.T) {
	found := map[spawnSite][]string{} // site -> "file:line" occurrences
	for _, pkgDir := range guardedPackages {
		abs, err := filepath.Abs(pkgDir)
		if err != nil {
			t.Fatalf("resolve %s: %v", pkgDir, err)
		}
		entries, err := os.ReadDir(abs)
		if err != nil {
			t.Fatalf("read package dir %s: %v", abs, err)
		}
		fset := token.NewFileSet()
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			path := filepath.Join(abs, name)
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			label := filepath.Base(filepath.Dir(path)) + "/" + name
			for site, locs := range spawnCallsInFile(fset, file, name, label) {
				found[site] = append(found[site], locs...)
			}
		}
	}

	// (1) Every detected spawn must be a documented legacy exception.
	var violations []string
	for site, locs := range found {
		if _, allowed := allowedLegacySpawns[site]; !allowed {
			violations = append(violations, "NEW direct spawn "+site.method+" in "+site.file+" at "+strings.Join(locs, ", "))
		}
	}
	// (2) Every documented exception must still exist (force the allowlist to shrink).
	var stale []string
	for site, reason := range allowedLegacySpawns {
		if len(found[site]) == 0 {
			stale = append(stale, site.file+" "+site.method+" (reroute landed — delete this allowlist entry; was: "+reason+")")
		}
	}

	sort.Strings(violations)
	sort.Strings(stale)
	if len(violations) > 0 {
		t.Errorf("domain packages must not spawn/continue Agent Manager runs directly — route programmatic work through a declared workflow or conversation through agentsessions:\n  %s", strings.Join(violations, "\n  "))
	}
	if len(stale) > 0 {
		t.Errorf("stale spawn-boundary allowlist entries (their reroute is done):\n  %s", strings.Join(stale, "\n  "))
	}
}

// spawnCallsInFile returns the spawn-seam call sites in one parsed file, keyed by
// (basename, method) with "label:line" occurrences. It is the single detection
// primitive both the real guardrail and the red-proof test exercise.
func spawnCallsInFile(fset *token.FileSet, file *ast.File, basename, label string) map[spawnSite][]string {
	out := map[spawnSite][]string{}
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !spawnMethods[sel.Sel.Name] {
			return true
		}
		site := spawnSite{file: basename, method: sel.Sel.Name}
		pos := fset.Position(call.Pos())
		out[site] = append(out[site], label+":"+itoa(pos.Line))
		return true
	})
	return out
}

// TestSpawnBoundaryDetectorFiresOnViolation red-proofs the guardrail: it runs the
// exact detection primitive against a synthetic source that contains a direct
// SpawnBacklog call and asserts the call is detected, and against a clean source
// and asserts nothing is detected. This proves the guardrail would fire on a real
// regression without committing a violation to any production package.
func TestSpawnBoundaryDetectorFiresOnViolation(t *testing.T) {
	const violating = `package sample
type spawner interface{ SpawnBacklog(x int) }
func run(s spawner) { s.SpawnBacklog(1) }`
	const clean = `package sample
func run() { _ = 1 }`

	fset := token.NewFileSet()
	vf, err := parser.ParseFile(fset, "violating.go", violating, 0)
	if err != nil {
		t.Fatalf("parse violating: %v", err)
	}
	got := spawnCallsInFile(fset, vf, "violating.go", "sample/violating.go")
	if len(got[spawnSite{"violating.go", "SpawnBacklog"}]) != 1 {
		t.Fatalf("detector must flag the synthetic SpawnBacklog call, got %v", got)
	}

	cf, err := parser.ParseFile(fset, "clean.go", clean, 0)
	if err != nil {
		t.Fatalf("parse clean: %v", err)
	}
	if len(spawnCallsInFile(fset, cf, "clean.go", "sample/clean.go")) != 0 {
		t.Fatalf("detector must not flag a clean source")
	}
}

// itoa is a tiny int->string to avoid importing strconv for one call.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
