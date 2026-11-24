// Package clianatomy derives bounded source evidence for ScenarioApp and
// command-group composition. It reports declared wiring, not runtime dispatch.
package clianatomy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var scenarioSlug = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

type Evidence struct {
	Path   string `json:"path"`
	Line   int    `json:"line"`
	SHA256 string `json:"sha256"`
}
type Result struct {
	ID          string     `json:"id"`
	Path        string     `json:"path"`
	Title       string     `json:"title"`
	Text        string     `json:"text"`
	Score       float64    `json:"score"`
	Evidence    []Evidence `json:"evidence"`
	Attestation string     `json:"attestation"`
}
type Degradation struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type Response struct {
	Results      []Result      `json:"results"`
	Degradations []Degradation `json:"degradations"`
	Scope        string        `json:"scope"`
}
type registration struct {
	Package string
	Line    int
}
type manifest struct {
	Groups []struct {
		Name     string `json:"name"`
		Commands []struct {
			Name string `json:"name"`
		} `json:"commands"`
	} `json:"groups"`
}

// Search inspects one scenario's CLI app, domain aggregator, registered group
// constructors and manifest. It reads no more than 11 files, each <=1 MiB.
func Search(repoRoot, scenario, query string) Response {
	out := Response{Results: []Result{}, Degradations: []Degradation{}, Scope: "one scenario cli/app.go, cli/domains/domains.go, at most eight registered group constructors, and cli/manifest.json; source declarations only"}
	scenario, query = strings.TrimSpace(scenario), strings.TrimSpace(query)
	if !scenarioSlug.MatchString(scenario) || query == "" {
		out.Degradations = append(out.Degradations, Degradation{"invalid_scope_or_query", "A scenario slug and non-empty query are required."})
		return out
	}
	scenariosRoot := filepath.Join(repoRoot, "scenarios")
	scenarioRoot, rootErr := filepath.EvalSymlinks(filepath.Join(scenariosRoot, scenario))
	if rootErr != nil {
		degrade(&out, "source_unavailable", "Scenario source directory is unavailable.")
		return out
	}
	realScenarios, rootErr := filepath.EvalSymlinks(scenariosRoot)
	if rootErr != nil {
		degrade(&out, "source_unavailable", "Scenario source directory is unavailable.")
		return out
	}
	relScenario, rootErr := filepath.Rel(realScenarios, scenarioRoot)
	if rootErr != nil || relScenario == ".." || strings.HasPrefix(relScenario, ".."+string(filepath.Separator)) {
		degrade(&out, "invalid_scope_or_query", "Scenario source resolves outside the repository scenarios directory.")
		return out
	}
	base := filepath.Join(scenarioRoot, "cli")
	app, err := readScenarioFile(base, "app.go", 1<<20)
	if err != nil {
		degrade(&out, "source_unavailable", "CLI app source is unavailable.")
		return out
	}
	domains, err := readScenarioFile(base, "domains/domains.go", 1<<20)
	if err != nil {
		degrade(&out, "source_unavailable", "CLI domain aggregator is unavailable.")
		return out
	}
	appWiring, appLine, ok := parseApp(app)
	if !ok {
		degrade(&out, "scenario_app_not_found", "No cliapp.NewStandardScenarioApp wiring with group constructors was established.")
		return out
	}
	regs, domainLine, ok := parseGroups(domains)
	if !ok || len(regs) == 0 {
		degrade(&out, "group_registration_unresolved", "Subcommand-group constructor registrations were not resolved.")
		return out
	}
	if len(regs) > 8 {
		degrade(&out, "group_scope_limit", "More than eight group constructors exceed the bounded source scope.")
		return out
	}
	manifestBytes, err := readScenarioFile(base, "manifest.json", 1<<20)
	if err != nil {
		degrade(&out, "manifest_unavailable", "CLI command manifest is unavailable.")
		return out
	}
	var m manifest
	if json.Unmarshal(manifestBytes, &m) != nil {
		degrade(&out, "manifest_invalid", "CLI command manifest could not be parsed.")
		return out
	}
	manifestGroups := map[string]int{}
	manifestNames := make([]string, 0, len(m.Groups))
	for _, g := range m.Groups {
		manifestGroups[g.Name] = len(g.Commands)
		manifestNames = append(manifestNames, g.Name)
	}
	sort.Strings(manifestNames)
	groupEvidence := make([]Evidence, 0, len(regs))
	groupMappings := make([]string, 0, len(regs))
	partial := false
	for _, reg := range regs {
		rel := filepath.ToSlash(filepath.Join("domains", reg.Package, "register.go"))
		src, readErr := readScenarioFile(base, rel, 1<<20)
		if readErr != nil {
			partial = true
			groupMappings = append(groupMappings, reg.Package+".Register: source unavailable")
			continue
		}
		groupName, line, loaded := parseGroupRegistration(src)
		if !loaded {
			partial = true
			groupMappings = append(groupMappings, reg.Package+".Register: manifest binding unresolved")
			continue
		}
		commands, exists := manifestGroups[groupName]
		if !exists {
			partial = true
			groupMappings = append(groupMappings, reg.Package+".Register → "+groupName+": no matching manifest group")
		} else {
			groupMappings = append(groupMappings, reg.Package+".Register → "+groupName+" ("+fmt.Sprint(commands)+" declared commands)")
		}
		groupEvidence = append(groupEvidence, Evidence{Path: rel, Line: line, SHA256: digest(src)})
	}
	if len(groupEvidence) == 0 {
		degrade(&out, "group_binding_unresolved", "No registered group could be joined to a manifest-loading constructor.")
		return out
	}
	text := fmt.Sprintf("%s CLI wires cliapp.NewStandardScenarioApp with %s. domains.SubcommandGroups registers %s. The registered constructors resolve as %s. Manifest groups: %s. This is source/manifest composition evidence; it does not establish runtime dispatch or installed CLI availability.", scenario, appWiring, strings.Join(packageNames(regs), ", "), strings.Join(groupMappings, "; "), strings.Join(manifestNames, ", "))
	score := lexicalScore(query, text)
	if score < 0.2 {
		degrade(&out, "no_matching_composition", "No strong source-composition match for this query.")
		return out
	}
	evidence := []Evidence{{Path: "cli/app.go", Line: appLine, SHA256: digest(app)}, {Path: "cli/domains/domains.go", Line: domainLine, SHA256: digest(domains)}}
	evidence = append(evidence, groupEvidence...)
	evidence = append(evidence, Evidence{Path: "cli/manifest.json", Line: jsonKeyLine(manifestBytes, "groups"), SHA256: digest(manifestBytes)})
	attestation := "DERIVED"
	if partial {
		attestation = "PARTIAL"
		out.Degradations = append(out.Degradations, Degradation{"partial_group_join", "One or more registered constructors could not be joined to manifest groups."})
	}
	out.Results = append(out.Results, Result{ID: scenario + ":cli-composition:scenario-app", Path: "cli/app.go", Title: scenario + " ScenarioApp and command groups", Text: text, Score: score, Evidence: evidence, Attestation: attestation})
	return out
}

func degrade(out *Response, code, message string) {
	out.Degradations = append(out.Degradations, Degradation{code, message})
}
func readScenarioFile(base, relative string, max int64) ([]byte, error) {
	realBase, err := filepath.EvalSymlinks(base)
	if err != nil {
		return nil, err
	}
	candidate := filepath.Join(realBase, filepath.Clean(relative))
	realPath, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(realBase, realPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("source escapes scenario CLI root")
	}
	info, err := os.Stat(realPath)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > max {
		return nil, fmt.Errorf("source exceeds bound")
	}
	return os.ReadFile(realPath)
}
func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func parseApp(src []byte) (string, int, bool) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "cli/app.go", src, 0)
	if err != nil {
		return "", 0, false
	}
	found := false
	line := 0
	options := []string{}
	ast.Inspect(f, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := c.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "NewStandardScenarioApp" {
			return true
		}
		x, ok := sel.X.(*ast.Ident)
		if !ok || x.Name != "cliapp" {
			return true
		}
		found = true
		line = fset.Position(c.Pos()).Line
		for _, a := range c.Args {
			if kvs, ok := a.(*ast.CompositeLit); ok {
				for _, e := range kvs.Elts {
					if pair, ok := e.(*ast.KeyValueExpr); ok {
						if id, ok := pair.Key.(*ast.Ident); ok && (id.Name == "CommandGroups" || id.Name == "SubcommandGroups") {
							options = append(options, id.Name+"="+exprName(pair.Value))
						}
					}
				}
			}
		}
		return true
	})
	if !found {
		return "", 0, false
	}
	sort.Strings(options)
	return strings.Join(options, " and "), line, len(options) > 0
}
func parseGroups(src []byte) ([]registration, int, bool) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "cli/domains/domains.go", src, 0)
	if err != nil {
		return nil, 0, false
	}
	var out []registration
	line := 0
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "SubcommandGroups" {
			continue
		}
		line = fset.Position(fn.Pos()).Line
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			s, ok := n.(*ast.SelectorExpr)
			if ok && s.Sel.Name == "Register" {
				if x, ok := s.X.(*ast.Ident); ok {
					out = append(out, registration{x.Name, fset.Position(s.Pos()).Line})
				}
			}
			return true
		})
		break
	}
	seen := map[string]bool{}
	unique := out[:0]
	for _, r := range out {
		if !seen[r.Package] {
			seen[r.Package] = true
			unique = append(unique, r)
		}
	}
	sort.Slice(unique, func(i, j int) bool { return unique[i].Package < unique[j].Package })
	return unique, line, line > 0
}
func parseGroupRegistration(src []byte) (string, int, bool) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "register.go", src, 0)
	if err != nil {
		return "", 0, false
	}
	name, line := "", 0
	consts := map[string]string{}
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			v, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, id := range v.Names {
				if i >= len(v.Values) {
					continue
				}
				lit, ok := v.Values[i].(*ast.BasicLit)
				if !ok {
					continue
				}
				value, e := strconv.Unquote(lit.Value)
				if e == nil {
					consts[id.Name] = value
				}
			}
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := c.Fun.(*ast.SelectorExpr)
		if !ok || !strings.HasPrefix(sel.Sel.Name, "LoadFromManifest") {
			return true
		}
		if len(c.Args) < 2 {
			return true
		}
		switch value := c.Args[1].(type) {
		case *ast.BasicLit:
			name, _ = strconv.Unquote(value.Value)
		case *ast.Ident:
			name = consts[value.Name]
		}
		if name != "" {
			line = fset.Position(c.Pos()).Line
		}
		return true
	})
	return name, line, name != ""
}
func exprName(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.SelectorExpr:
		return exprName(x.X) + "." + x.Sel.Name
	case *ast.Ident:
		return x.Name
	case *ast.FuncLit:
		return "factory"
	default:
		return "expression"
	}
}
func packageNames(regs []registration) []string {
	out := make([]string, len(regs))
	for i, r := range regs {
		out[i] = r.Package
	}
	return out
}
func jsonKeyLine(b []byte, key string) int {
	needle := `"` + key + `"`
	idx := strings.Index(string(b), needle)
	if idx < 0 {
		return 1
	}
	return 1 + strings.Count(string(b[:idx]), "\n")
}
func lexicalScore(query, text string) float64 {
	q := strings.Fields(strings.ToLower(query))
	if len(q) == 0 {
		return 0
	}
	text = strings.ToLower(text)
	seen := map[string]bool{}
	hit := 0
	for _, w := range q {
		w = strings.Trim(w, ".,:;()\"'`/")
		if len(w) < 3 || seen[w] {
			continue
		}
		seen[w] = true
		if strings.Contains(text, w) {
			hit++
		}
	}
	if len(seen) == 0 {
		return 0
	}
	return float64(hit) / float64(len(seen))
}
