package tuning

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"testing"
)

func TestPermissionDefaults(t *testing.T) {
	checks := map[string]uint32{
		"dir": uint32(PermDir), "private dir": uint32(PermPrivateDir),
		"file": uint32(PermFile), "secret": uint32(PermSecret),
		"group dir": uint32(PermGroupDir), "executable": uint32(PermExecutable),
		"read-only group": uint32(PermReadOnlyGroup), "sudoers": uint32(PermSudoers),
		"socket": uint32(PermSocket), "lock": uint32(PermLock),
		"execute mask": uint32(PermExecuteMask), "group and other mask": uint32(PermGroupAndOtherMask),
		"group read-write": uint32(PermGroupReadWrite), "owner write": uint32(PermOwnerWrite),
		"none": uint32(PermNone),
	}
	want := map[string]uint32{
		"dir": 0o755, "private dir": 0o700, "file": 0o644, "secret": 0o600,
		"group dir": 0o750, "executable": 0o755, "read-only group": 0o640,
		"sudoers": 0o440, "socket": 0o660, "lock": 0o666, "execute mask": 0o111,
		"group and other mask": 0o077, "group read-write": 0o060, "owner write": 0o200, "none": 0o000,
	}
	for name, got := range checks {
		if got != want[name] {
			t.Errorf("%s = %o, want %o", name, got, want[name])
		}
	}
}

func TestTimingConstantsUsePurposeNames(t *testing.T) {
	source, err := os.ReadFile("timing.go")
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "timing.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	valueNamed := regexp.MustCompile(`^Duration[0-9]+(ms|s|m|h|d)$`)
	for _, declaration := range file.Decls {
		gen, ok := declaration.(*ast.GenDecl)
		if !ok || gen.Tok.String() != "const" {
			continue
		}
		for _, spec := range gen.Specs {
			valueSpec := spec.(*ast.ValueSpec)
			for _, name := range valueSpec.Names {
				if valueNamed.MatchString(name.Name) {
					t.Errorf("timing constant %q is named after its value", name.Name)
				}
			}
		}
	}
}
