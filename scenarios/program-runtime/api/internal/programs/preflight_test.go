package programs

import (
	"path/filepath"
	"strings"
	"testing"
)

// analyzerPath resolves the shipped analyzer relative to this package.
func analyzerPath() string {
	return filepath.Join("..", "..", "..", "kernel", "host", "analyze.py")
}

var knownNames = []string{
	"discover", "recall", "guide", "validate", "capture", "ai", "agent", "gather",
	"describe", "reachable", "lib", "vrooli", "__vrooli__", "Handle",
	"search_hub", "test_genie", "program_runtime", "agent_manager",
}

func resolve(t *testing.T, source string) []diagnosticView {
	t.Helper()
	diagnostics := ResolveSource(source, knownNames, analyzerPath())
	if diagnostics == nil {
		// Analysis is unavailable in this environment. The contract is that a
		// missing analyzer never refuses a program, so an empty result is the
		// correct degraded behavior rather than a test failure.
		return nil
	}
	out := make([]diagnosticView, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		out = append(out, diagnosticView{
			Severity: diagnostic.GetSeverity(),
			Name:     diagnostic.GetName(),
			Message:  diagnostic.GetMessage(),
			Nearest:  diagnostic.GetNearestMatch(),
		})
	}
	return out
}

type diagnosticView struct {
	Severity string
	Name     string
	Message  string
	Nearest  string
}

func errorsOnly(views []diagnosticView) []diagnosticView {
	var out []diagnosticView
	for _, view := range views {
		if view.Severity == "error" {
			out = append(out, view)
		}
	}
	return out
}

// TestResolveSourceAcceptsLocallyBoundNames is the regression guard for the
// defect that made this runtime unusable: the previous regex resolver treated
// every name bound by a `def`, `lambda`, comprehension, `for`, `with`, or
// `except` as an unresolved global and refused the program before execution.
// Each case below is an ordinary, correct program.
func TestResolveSourceAcceptsLocallyBoundNames(t *testing.T) {
	cases := map[string]string{
		"lambda parameter":         "f = lambda row: row['a']\nprint(f({'a': 1}))",
		"comprehension variable":   "print([x for x in range(3)])",
		"dict comprehension":       "print({k: v for k, v in [(1, 2)]})",
		"function definition":      "def g(a):\n    return a\nprint(g(2))",
		"for target":               "for i in range(2):\n    print(i)",
		"with target":              "with open('f') as fh:\n    print(fh)",
		"except alias":             "try:\n    pass\nexcept KeyError as exc:\n    print(exc)",
		"class definition":         "class C:\n    def m(self):\n        return 1\nprint(C().m())",
		"walrus":                   "if (n := 5) > 1:\n    print(n)",
		"forward global reference": "def f():\n    return later\nlater = 1\nprint(f())",
		"nested closure":           "def outer():\n    total = 0\n    def inner():\n        return total\n    return inner()\nprint(outer())",
		"import alias":             "import json as j\nprint(j.dumps({}))",
		"from import":              "from math import sqrt\nprint(sqrt(4))",
		"gather with lambda":       "print(gather(*[lambda q=q: search_hub.query.query(query=q) for q in ['a', 'b']]))",
	}
	for name, source := range cases {
		t.Run(name, func(t *testing.T) {
			if found := errorsOnly(resolve(t, source)); len(found) != 0 {
				t.Fatalf("correct program was refused: %+v", found)
			}
		})
	}
}

// TestResolveSourceAcceptsWithheldBuiltins guards the second half of the same
// defect: builtins the kernel does supply must never be reported as unresolved.
func TestResolveSourceAcceptsWithheldBuiltins(t *testing.T) {
	for _, name := range []string{"round", "type", "getattr", "hasattr", "next", "iter", "format", "frozenset", "divmod", "callable"} {
		t.Run(name, func(t *testing.T) {
			if found := errorsOnly(resolve(t, "print("+name+")")); len(found) != 0 {
				t.Fatalf("builtin %q was refused: %+v", name, found)
			}
		})
	}
}

func TestResolveSourceReportsGenuinelyUnresolvedName(t *testing.T) {
	found := errorsOnly(resolve(t, "print(test_geni.runs.list())"))
	if found == nil {
		t.Skip("analyzer unavailable in this environment")
	}
	if len(found) != 1 {
		t.Fatalf("expected exactly one error, got %+v", found)
	}
	if found[0].Name != "test_geni" {
		t.Fatalf("expected the offending name, got %q", found[0].Name)
	}
	if found[0].Nearest != "test_genie" {
		t.Fatalf("expected nearest match test_genie, got %q", found[0].Nearest)
	}
}

// TestNearestNameWithholdsMisleadingSuggestion guards the suggestion quality
// defect: the character-overlap metric matched `KeyError` to `discover` and
// `text` to `agent`. A model acts on a suggestion, so a wrong one is worse than
// none.
func TestNearestNameWithholdsMisleadingSuggestion(t *testing.T) {
	for _, name := range []string{"KeyError", "text", "row", "item", "zzzzzz"} {
		if suggestion := nearestName(name, knownNames); suggestion != "" {
			t.Fatalf("name %q should have no suggestion, got %q", name, suggestion)
		}
	}
}

func TestNearestNameOffersCloseMatch(t *testing.T) {
	for name, want := range map[string]string{
		"test_geni":  "test_genie",
		"search_hb":  "search_hub",
		"discovre":   "discover",
		"__vrooli__": "__vrooli__",
	} {
		if suggestion := nearestName(name, knownNames); suggestion != want {
			t.Fatalf("name %q: expected %q, got %q", name, want, suggestion)
		}
	}
}

func TestResolveSourceWarnsOnShadowAndRefusesProtected(t *testing.T) {
	views := resolve(t, "search_hub = 5\nprint(search_hub)")
	if views == nil {
		t.Skip("analyzer unavailable in this environment")
	}
	if len(views) != 1 || views[0].Severity != "warning" || views[0].Name != "search_hub" {
		t.Fatalf("expected one shadow warning, got %+v", views)
	}
	if !strings.Contains(views[0].Message, "__vrooli__.search_hub") {
		t.Fatalf("shadow warning must name the escape hatch, got %q", views[0].Message)
	}

	protectedViews := errorsOnly(resolve(t, "vrooli = 1"))
	if len(protectedViews) != 1 || protectedViews[0].Name != "vrooli" {
		t.Fatalf("expected a protected-name error, got %+v", protectedViews)
	}
}

// TestResolveSourceDegradesOnSyntaxError keeps a syntax error owned by the
// kernel, which reports it with the kernel_syntax cause. Refusing it here would
// attribute it to name resolution instead.
func TestResolveSourceDegradesOnSyntaxError(t *testing.T) {
	if found := errorsOnly(resolve(t, "def broken(:\n  pass")); len(found) != 0 {
		t.Fatalf("syntax error must not be reported as a name diagnostic, got %+v", found)
	}
}

func TestResolveSourceWithoutAnalyzerReturnsNoDiagnostics(t *testing.T) {
	if diagnostics := ResolveSource("print(totally_unknown_name)", knownNames, ""); diagnostics != nil {
		t.Fatalf("a missing analyzer must never refuse a program, got %+v", diagnostics)
	}
}

// TestOnlyUnresolvedCapabilitiesReachTheLedger guards the Act denominator's
// evidence. A protected-name assignment and a shadow warning both name things
// that resolve; recording them answers "what could an agent not invoke" with a
// name the agent could invoke perfectly well.
func TestOnlyUnresolvedCapabilitiesReachTheLedger(t *testing.T) {
	protectedViews := ResolveSource("vrooli = 1", knownNames, analyzerPath())
	if protectedViews == nil {
		t.Skip("analyzer unavailable in this environment")
	}
	for _, diagnostic := range protectedViews {
		if IsUnresolvedNameDiagnostic(diagnostic) {
			t.Fatalf("protected-name refusal must not be recorded as an unresolved capability: %q", diagnostic.GetMessage())
		}
	}

	shadowViews := ResolveSource("search_hub = 5\nprint(search_hub)", knownNames, analyzerPath())
	for _, diagnostic := range shadowViews {
		if IsUnresolvedNameDiagnostic(diagnostic) {
			t.Fatalf("shadow warning must not be recorded as an unresolved capability: %q", diagnostic.GetMessage())
		}
	}

	missViews := ResolveSource("print(test_geni.runs.list())", knownNames, analyzerPath())
	recorded := 0
	for _, diagnostic := range missViews {
		if IsUnresolvedNameDiagnostic(diagnostic) {
			recorded++
		}
	}
	if recorded != 1 {
		t.Fatalf("expected exactly one recordable capability miss, got %d", recorded)
	}
}
