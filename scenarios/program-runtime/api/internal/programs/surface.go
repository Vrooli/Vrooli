package programs

// runtimeVerbNames mirrors `_RUNTIME_VERB_NAMES` in kernel/host/engine.py: the
// names the kernel binds at the top level before any scenario namespace.
var runtimeVerbNames = []string{
	"discover", "recall", "guide", "validate", "capture", "ai", "agent",
	"gather", "describe", "reachable", "lib", "tasks", "program",
}

// ProgramHelperMembers mirrors `PUBLIC_MEMBERS` in kernel/host/program_helper.py.
// Preflight validates `program.<member>` attribute paths against this list, so
// a typo such as `program.clasify` is refused before the kernel runs it.
// TestProgramHelperMembersMatchKernel keeps the two lists identical.
var ProgramHelperMembers = []string{
	"VERSION", "inputs", "envelope", "attach", "current", "fail", "guarded",
	"classify", "run", "report", "BindingError", "ScenarioUnreachable",
	"Refused", "InvalidInput", "AmbiguousResponse", "DeadlineExceeded",
	"RemoteError", "NoGovernedBinding",
}

// RuntimeSurfaceNames is the single authority for the names a program may
// reference that are neither builtins nor scenario bindings. Before it existed
// the same list was written out in three files, and adding a runtime name meant
// finding all three or shipping a preflight that refused a correct program.
func RuntimeSurfaceNames() []string {
	names := make([]string, 0, len(runtimeVerbNames)+len(ProgramHelperMembers)+3)
	names = append(names, runtimeVerbNames...)
	names = append(names, "vrooli", "__vrooli__", "Handle")
	for _, member := range ProgramHelperMembers {
		names = append(names, "program."+member)
	}
	return names
}

// ProtectedRuntimeNames reports the top-level names a program may not assign.
func ProtectedRuntimeNames() map[string]struct{} {
	protected := make(map[string]struct{}, len(runtimeVerbNames)+2)
	for _, name := range runtimeVerbNames {
		protected[name] = struct{}{}
	}
	protected["vrooli"] = struct{}{}
	protected["__vrooli__"] = struct{}{}
	return protected
}
