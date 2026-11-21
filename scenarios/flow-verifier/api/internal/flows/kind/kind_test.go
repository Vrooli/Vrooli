package kind_test

import (
	"context"
	"testing"

	"flow-verifier/internal/flows/kind"
)

type fakeKind struct {
	name string
}

func (f *fakeKind) Name() string                           { return f.name }
func (f *fakeKind) SchemaJSON() []byte                     { return []byte(`{}`) }
func (f *fakeKind) FilenameGlobs() []string                { return []string{"*.json"} }
func (f *fakeKind) Load([]byte, string) (kind.Spec, error) { return nil, nil }
func (f *fakeKind) Verify(context.Context, kind.Spec) (kind.VerifyResult, error) {
	return kind.VerifyResult{}, nil
}
func (f *fakeKind) Scaffold(kind.ScaffoldOptions) (string, error) { return "", nil }
func (f *fakeKind) Codegen(kind.Spec, kind.Language) (kind.Artifacts, error) {
	return kind.Artifacts{}, nil
}
func (f *fakeKind) StudioDescriptor(kind.Spec) kind.StudioDescriptor { return kind.StudioDescriptor{} }

// Registration is a global side-effect, so the registry can't be reset
// between subtests. Use unique names per subtest.

func TestRegisterAndGet(t *testing.T) {
	k := &fakeKind{name: "kind-test-register"}
	kind.Register(k)
	got, ok := kind.Get("kind-test-register")
	if !ok {
		t.Fatal("Get returned ok=false for a registered kind")
	}
	if got.Name() != k.Name() {
		t.Fatalf("Get returned %q, want %q", got.Name(), k.Name())
	}
}

func TestGetUnknownReturnsFalse(t *testing.T) {
	if _, ok := kind.Get("kind-test-never-registered"); ok {
		t.Fatal("Get returned ok=true for an unregistered kind")
	}
}

func TestRegisterDuplicatePanics(t *testing.T) {
	kind.Register(&fakeKind{name: "kind-test-duplicate"})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()
	kind.Register(&fakeKind{name: "kind-test-duplicate"})
}

func TestRegisterEmptyNamePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on empty name")
		}
	}()
	kind.Register(&fakeKind{name: ""})
}

func TestAllAndNamesAreSorted(t *testing.T) {
	kind.Register(&fakeKind{name: "kind-test-zzz"})
	kind.Register(&fakeKind{name: "kind-test-aaa"})
	names := kind.Names()
	aaaIdx, zzzIdx := -1, -1
	for i, n := range names {
		if n == "kind-test-aaa" {
			aaaIdx = i
		}
		if n == "kind-test-zzz" {
			zzzIdx = i
		}
	}
	if aaaIdx == -1 || zzzIdx == -1 {
		t.Fatalf("expected both test kinds in Names(); got %v", names)
	}
	if aaaIdx >= zzzIdx {
		t.Fatalf("Names() not sorted: aaa at %d, zzz at %d", aaaIdx, zzzIdx)
	}
}
