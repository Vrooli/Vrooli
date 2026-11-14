package programs

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// pythonTuple extracts the string members of a module-level tuple assignment
// such as `PUBLIC_MEMBERS = ("a", "b", ...)` from a kernel source file.
func pythonTuple(t *testing.T, relativePath, name string) []string {
	t.Helper()
	path := filepath.Join("..", "..", "..", "kernel", "host", relativePath)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	block := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(name) + `\s*=\s*\((.*?)\)\n`).FindStringSubmatch(string(data))
	if block == nil {
		t.Fatalf("%s does not declare %s", path, name)
	}
	var members []string
	for _, match := range regexp.MustCompile(`"([A-Za-z_][A-Za-z0-9_]*)"`).FindAllStringSubmatch(block[1], -1) {
		members = append(members, match[1])
	}
	return members
}

func TestRuntimeVerbNamesMatchKernel(t *testing.T) {
	kernel := pythonTuple(t, "engine.py", "_RUNTIME_VERB_NAMES")
	if strings.Join(kernel, ",") != strings.Join(runtimeVerbNames, ",") {
		t.Fatalf("runtime verb surface drifted: kernel=%v go=%v", kernel, runtimeVerbNames)
	}
}

func TestProgramHelperMembersMatchKernel(t *testing.T) {
	kernel := pythonTuple(t, "program_helper.py", "PUBLIC_MEMBERS")
	if strings.Join(kernel, ",") != strings.Join(ProgramHelperMembers, ",") {
		t.Fatalf("program helper surface drifted: kernel=%v go=%v", kernel, ProgramHelperMembers)
	}
}

func TestRuntimeSurfaceNamesCarryHelperPaths(t *testing.T) {
	names := RuntimeSurfaceNames()
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		set[name] = struct{}{}
	}
	for _, want := range []string{"program", "program.classify", "program.report", "Handle", "vrooli", "lib"} {
		if _, ok := set[want]; !ok {
			t.Fatalf("RuntimeSurfaceNames lacks %q: %v", want, names)
		}
	}
	if _, ok := ProtectedRuntimeNames()["program"]; !ok {
		t.Fatal("program is not protected")
	}
	if _, ok := ProtectedRuntimeNames()["Handle"]; ok {
		t.Fatal("Handle is a type a program may alias, not a protected verb")
	}
}
