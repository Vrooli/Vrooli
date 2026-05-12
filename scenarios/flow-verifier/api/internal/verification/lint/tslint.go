package lint

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"flow-verifier/internal/codegen"
	"flow-verifier/internal/flows/model"
)

const tsShape = `    1. import { runFormalReplay } from "./generated/replay.helper";
    2. import { <transition> } from "./transition";
    3. import { <fixtures> } from "./fixtures";
    4. runFormalReplay({ transition, fixtures }); at module top level.
  Example:
    import { runFormalReplay } from "./generated/replay.helper";
    import { transitionFoo } from "./transition";
    import { fooFormalFixtures } from "./fixtures";
    runFormalReplay({ transition: transitionFoo, fixtures: fooFormalFixtures });
`

func checkTypeScript(root string, flow model.Flow) error {
	files, err := listFiles(root, flow.Layout.BaseDir, ".test.ts")
	if err != nil {
		return fmt.Errorf("%s: read %s: %w", flow.FlowID, flow.Layout.BaseDir, err)
	}
	expectedHelper := "./generated/replay.helper"
	expectedWrapper := "./transition"
	expectedFixtures := "./fixtures"
	wrapperFunc := flow.Replay.Transition.Function
	fixtureExport := codegen.TypeScriptFixturesExportName(flow)
	var failures []string
	matched := false
	for _, path := range files {
		ok, why, err := scanTSTestFile(path, expectedHelper, expectedWrapper, wrapperFunc, expectedFixtures, fixtureExport)
		if err != nil {
			failures = append(failures, fmt.Sprintf("    %s: %v", path, err))
			continue
		}
		if ok {
			matched = true
			continue
		}
		if why != "" {
			failures = append(failures, fmt.Sprintf("    %s: %s", path, why))
		}
	}
	if matched {
		return nil
	}
	msg := shape(flow, tsShape)
	if len(failures) > 0 {
		msg += "  Scanned files reported:\n" + strings.Join(failures, "\n") + "\n"
	} else {
		msg += "  Scanned files: none with .test.ts suffix found.\n"
	}
	return fmt.Errorf("%s", msg)
}

// scanTSTestFile reads the source of a TS test file and verifies the
// four-point contract documented in tsShape. The scan uses structured
// regex matching against the import section and the top-level call
// site; it explicitly rejects calls nested inside any block (so
// `if (false) { runFormalReplay({...}) }` does not satisfy the lint).
func scanTSTestFile(path string, expectedHelper string, wrapperModule string, wrapperFunc string, fixtureModule string, fixtureExport string) (bool, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, "", err
	}
	source := stripComments(string(data))

	imports := parseTSImports(source)
	helperImport, ok := imports[normalizeImport(expectedHelper)]
	if !ok {
		return false, "", nil
	}
	if !containsBinding(helperImport, "runFormalReplay") {
		return false, "imports replay.helper but does not bind runFormalReplay", nil
	}

	wrapperImport, ok := imports[normalizeImport(stripTSExtension(wrapperModule))]
	if !ok {
		return false, fmt.Sprintf("missing import of wrapper module %q", wrapperModule), nil
	}
	if !containsBinding(wrapperImport, wrapperFunc) {
		return false, fmt.Sprintf("wrapper import does not bind %s", wrapperFunc), nil
	}

	fixtureImport, ok := imports[normalizeImport(stripTSExtension(fixtureModule))]
	if !ok {
		return false, fmt.Sprintf("missing import of fixtures module %q", fixtureModule), nil
	}
	if !containsBinding(fixtureImport, fixtureExport) {
		return false, fmt.Sprintf("fixtures import does not bind %s", fixtureExport), nil
	}

	if !hasTopLevelCall(source, "runFormalReplay") {
		return false, "no top-level call to runFormalReplay (calls nested inside blocks are rejected)", nil
	}
	return true, "", nil
}

// importBinding captures the named bindings in an import declaration.
type importBinding struct {
	Named []string
}

func parseTSImports(source string) map[string]importBinding {
	out := map[string]importBinding{}
	importRE := regexp.MustCompile(`(?m)^\s*import\s+(.+?)\s+from\s+["']([^"']+)["']\s*;?\s*$`)
	for _, match := range importRE.FindAllStringSubmatch(source, -1) {
		clause := match[1]
		modulePath := normalizeImport(match[2])
		binding := out[modulePath]
		binding.Named = append(binding.Named, parseImportClause(clause)...)
		out[modulePath] = binding
	}
	return out
}

var importClauseBracesRE = regexp.MustCompile(`\{([^}]*)\}`)

func parseImportClause(clause string) []string {
	var bindings []string
	for _, match := range importClauseBracesRE.FindAllStringSubmatch(clause, -1) {
		for _, name := range strings.Split(match[1], ",") {
			name = strings.TrimSpace(name)
			name = strings.TrimPrefix(name, "type ")
			if idx := strings.Index(name, " as "); idx >= 0 {
				name = strings.TrimSpace(name[:idx])
			}
			if name != "" {
				bindings = append(bindings, name)
			}
		}
	}
	defaultRE := regexp.MustCompile(`^\s*([A-Za-z_$][A-Za-z0-9_$]*)`)
	if m := defaultRE.FindStringSubmatch(clause); m != nil {
		bindings = append(bindings, m[1])
	}
	return bindings
}

func containsBinding(imp importBinding, name string) bool {
	for _, b := range imp.Named {
		if b == name {
			return true
		}
	}
	return false
}

func normalizeImport(value string) string {
	return strings.TrimSuffix(value, ".ts")
}

func stripTSExtension(value string) string {
	return strings.TrimSuffix(value, ".ts")
}

// hasTopLevelCall returns true if `<name>(` appears at zero brace
// depth in the source (i.e. at module scope). This is the strict
// shape the lint requires: the call must execute when the module is
// imported, not lazily inside a conditional.
func hasTopLevelCall(source string, name string) bool {
	depth := 0
	inString := byte(0)
	i := 0
	prefix := name + "("
	for i < len(source) {
		ch := source[i]
		if inString != 0 {
			if ch == '\\' && i+1 < len(source) {
				i += 2
				continue
			}
			if ch == inString {
				inString = 0
			}
			i++
			continue
		}
		switch ch {
		case '"', '\'', '`':
			inString = ch
			i++
			continue
		case '{', '(', '[':
			depth++
			i++
			continue
		case '}', ')', ']':
			if depth > 0 {
				depth--
			}
			i++
			continue
		}
		if depth == 0 && strings.HasPrefix(source[i:], prefix) && isIdentBoundary(source, i) {
			return true
		}
		i++
	}
	return false
}

func isIdentBoundary(source string, i int) bool {
	if i == 0 {
		return true
	}
	prev := source[i-1]
	return !(prev == '_' || prev == '$' || (prev >= 'A' && prev <= 'Z') || (prev >= 'a' && prev <= 'z') || (prev >= '0' && prev <= '9'))
}

// stripComments removes // line comments and /* block */ comments
// from source so the scanner does not see calls inside commented-out
// code. String literals are preserved.
func stripComments(source string) string {
	var b strings.Builder
	inString := byte(0)
	i := 0
	for i < len(source) {
		ch := source[i]
		if inString != 0 {
			b.WriteByte(ch)
			if ch == '\\' && i+1 < len(source) {
				b.WriteByte(source[i+1])
				i += 2
				continue
			}
			if ch == inString {
				inString = 0
			}
			i++
			continue
		}
		switch ch {
		case '"', '\'', '`':
			inString = ch
			b.WriteByte(ch)
			i++
			continue
		}
		if i+1 < len(source) && ch == '/' && source[i+1] == '/' {
			for i < len(source) && source[i] != '\n' {
				i++
			}
			continue
		}
		if i+1 < len(source) && ch == '/' && source[i+1] == '*' {
			i += 2
			for i+1 < len(source) && !(source[i] == '*' && source[i+1] == '/') {
				i++
			}
			i += 2
			continue
		}
		b.WriteByte(ch)
		i++
	}
	return b.String()
}
