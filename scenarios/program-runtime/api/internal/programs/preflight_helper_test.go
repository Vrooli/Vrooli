package programs

import "testing"

// The `program` helper is a runtime name like `gather` or `lib`: its members
// are validated by attribute path so a typo is refused before the kernel runs
// it, and it cannot be assigned at the top level.
func TestResolveSourceAcceptsProgramHelperMembers(t *testing.T) {
	source := "envelope = program.envelope('demo.read', '1')\n" +
		"def step_validate():\n    return 'report'\n" +
		"def step_report():\n    program.report()\n" +
		"try:\n    raise program.BindingError('x')\nexcept program.BindingError as exc:\n    status, klass = program.classify(exc)\n" +
		"program.run({'validate': step_validate, 'report': step_report}, 'validate')\n"
	if diagnostics := ResolveSource(source, RuntimeSurfaceNames(), analyzerPath()); len(diagnostics) != 0 {
		t.Fatalf("helper members rejected: %v", diagnostics)
	}
}

func TestResolveSourceReportsMisspelledProgramHelperMember(t *testing.T) {
	diagnostics := ResolveSource("status, klass = program.clasify(RuntimeError('x'))", RuntimeSurfaceNames(), analyzerPath())
	if len(diagnostics) != 1 {
		t.Fatalf("expected one attribute diagnostic, got %v", diagnostics)
	}
	if diagnostics[0].GetName() != "program.clasify" || diagnostics[0].GetNearestMatch() != "program.classify" {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
}

func TestResolveSourceRefusesAssigningProgram(t *testing.T) {
	diagnostics := ResolveSource("program = {}\nprint(program)", RuntimeSurfaceNames(), analyzerPath())
	if len(diagnostics) != 1 || !IsProtectedNameMisuseDiagnostic(diagnostics[0]) {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
}

// A function-local named `program` is ordinary Python: it shadows the helper
// inside that function only and is not a protected-name misuse.
func TestResolveSourceAllowsLocalProgramVariable(t *testing.T) {
	source := "def pick(rows):\n    program = rows[0]\n    return program\nprint(pick([1]))"
	if diagnostics := ResolveSource(source, RuntimeSurfaceNames(), analyzerPath()); len(diagnostics) != 0 {
		t.Fatalf("local variable refused: %v", diagnostics)
	}
}
