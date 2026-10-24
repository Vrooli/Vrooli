// Package reconcile walks a scenario's TypeScript source tree and
// cross-checks <Route path>/<Link to>/useNavigate target paths against
// the navigation spec. The walker uses regex matching rather than a
// full TS AST — the patterns are narrow enough and the false-positive
// cost is captured in the Finding (we report file/line so a human can
// disambiguate). Phase 4 owns the mechanism; Phase 5 (self-adoption)
// will exercise it against flow-verifier's own UI.
package reconcile

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"flow-verifier/internal/flows/kinds/navigation/compile"
)

// Finding is one reported discrepancy between spec and source.
type Finding struct {
	ID         string
	Severity   string // "error" | "warning" | "info"
	Message    string
	SourceFile string
	SourceLine int
}

// Result is the aggregate of one reconcile pass.
type Result struct {
	Passed       bool
	Findings     []Finding
	FilesScanned int
}

// Run walks scenarioRoot's `ui/src` tree and reconciles against g.
// scenarioRoot is an absolute or repo-relative path. If `ui/src` does
// not exist under it, the walker returns an explicit "ui/src not found"
// finding and a zero-pass result.
func Run(g compile.Graph, scenarioRoot string) (Result, error) {
	srcDir := filepath.Join(scenarioRoot, "ui", "src")
	info, err := os.Stat(srcDir)
	if err != nil || !info.IsDir() {
		return Result{
			Passed: false,
			Findings: []Finding{{
				ID:       "missing_ui_src",
				Severity: "error",
				Message:  fmt.Sprintf("ui/src not found under %s", scenarioRoot),
			}},
		}, nil
	}

	specPaths := map[string]bool{}
	for _, r := range g.Contract.Routes {
		specPaths[r.Path] = true
	}
	specPathReferences := map[string]bool{} // paths referenced anywhere in source
	codeRoutePaths := map[string]bool{}     // paths declared via <Route path=>

	var files []string
	if err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "node_modules" || strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		name := d.Name()
		ext := strings.ToLower(filepath.Ext(name))
		switch ext {
		case ".ts", ".tsx", ".js", ".jsx":
			if isTestFile(name) {
				return nil
			}
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return Result{}, err
	}

	var findings []Finding
	literalPathReferences := map[string]bool{} // any JS string literal that exactly equals a spec path
	for _, fpath := range files {
		raw, err := os.ReadFile(fpath)
		if err != nil {
			return Result{}, err
		}
		rel, _ := filepath.Rel(scenarioRoot, fpath)
		body := string(raw)
		for p := range specPaths {
			for _, q := range []string{`"` + p + `"`, `'` + p + `'`, "`" + p + "`"} {
				if strings.Contains(body, q) {
					literalPathReferences[p] = true
					break
				}
			}
		}
		lines := strings.Split(body, "\n")
		for i, line := range lines {
			lineNo := i + 1
			for _, m := range routePathRE.FindAllStringSubmatch(line, -1) {
				p := m[1]
				codeRoutePaths[p] = true
				if !specPaths[p] {
					findings = append(findings, Finding{
						ID:         "route_in_code_not_in_spec:" + p,
						Severity:   "error",
						Message:    fmt.Sprintf("<Route path=%q> declared in code but not in spec", p),
						SourceFile: rel,
						SourceLine: lineNo,
					})
				}
			}
			for _, m := range navTargetRE.FindAllStringSubmatch(line, -1) {
				p := m[1]
				if !looksLikePath(p) {
					continue
				}
				specPathReferences[p] = true
				if !specPaths[p] && !matchesParameterised(p, g) {
					findings = append(findings, Finding{
						ID:         "nav_target_not_in_spec:" + p,
						Severity:   "error",
						Message:    fmt.Sprintf("navigation target %q does not resolve to a spec route", p),
						SourceFile: rel,
						SourceLine: lineNo,
					})
				}
			}
		}
	}

	for p := range specPaths {
		if !codeRoutePaths[p] && !specPathReferences[p] && !literalPathReferences[p] && !matchesParameterisedCode(p, codeRoutePaths) {
			findings = append(findings, Finding{
				ID:       "spec_route_orphan:" + p,
				Severity: "warning",
				Message:  fmt.Sprintf("spec route %q is not referenced anywhere in ui/src", p),
			})
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].ID != findings[j].ID {
			return findings[i].ID < findings[j].ID
		}
		return findings[i].SourceFile < findings[j].SourceFile
	})

	passed := true
	for _, f := range findings {
		if f.Severity == "error" {
			passed = false
			break
		}
	}
	return Result{Passed: passed, Findings: findings, FilesScanned: len(files)}, nil
}

// routePathRE captures `<Route path="/foo">` and `path: "/foo"` route
// declarations.
var routePathRE = regexp.MustCompile(`(?:<Route[^>]*\bpath=|\bpath:\s*)["'` + "`" + `]([^"'` + "`" + `]+)["'` + "`" + `]`)

// navTargetRE captures `to="..."`, `to: "..."`, `navigate("...")`,
// `useNavigate()(...)` callsites.
var navTargetRE = regexp.MustCompile(`(?:\bto=|\bto:\s*|navigate\(\s*)["'` + "`" + `]([^"'` + "`" + `]+)["'` + "`" + `]`)

func looksLikePath(s string) bool {
	return strings.HasPrefix(s, "/")
}

// isTestFile reports whether name looks like a test or spec source file.
// Reconcile skips these because component/unit tests routinely declare
// fake <Route path> and <Link to> targets that aren't part of the real
// navigation graph.
func isTestFile(name string) bool {
	lower := strings.ToLower(name)
	for _, suf := range []string{".test.ts", ".test.tsx", ".test.js", ".test.jsx", ".spec.ts", ".spec.tsx", ".spec.js", ".spec.jsx"} {
		if strings.HasSuffix(lower, suf) {
			return true
		}
	}
	return false
}

// matchesParameterised reports whether `p` (a literal target path) is
// produced by one of the spec's parameterised routes (e.g. `/tasks/42`
// matches spec `/tasks/:id`). Strict prefix + segment-count check.
func matchesParameterised(p string, g compile.Graph) bool {
	for _, r := range g.Contract.Routes {
		if pathMatchesPattern(p, r.Path) {
			return true
		}
	}
	return false
}

// matchesParameterisedCode reports whether the spec path `specPath` is
// covered by any code-declared <Route path> (which itself may be
// parameterised). Used to suppress orphan warnings for parameterised
// spec routes that the code declares verbatim.
func matchesParameterisedCode(specPath string, codePaths map[string]bool) bool {
	for cp := range codePaths {
		if cp == specPath {
			return true
		}
	}
	return false
}

func pathMatchesPattern(literal, pattern string) bool {
	lp := strings.Split(strings.Trim(literal, "/"), "/")
	pp := strings.Split(strings.Trim(pattern, "/"), "/")
	if len(lp) != len(pp) {
		return false
	}
	for i := range pp {
		if strings.HasPrefix(pp[i], ":") {
			continue
		}
		if pp[i] != lp[i] {
			return false
		}
	}
	return true
}
