package rewrite

import (
	"reflect"
	"testing"
)

func TestNormalizeDedupes(t *testing.T) {
	t.Parallel()
	in := []Operation{
		FileMove{From: "a.go", To: "b.go"},
		FileMove{From: "a.go", To: "b.go"},   // exact dup
		FileMove{From: "./a.go", To: "b.go"}, // canonicalizes to dup
	}
	out := Normalize(in)
	if len(out) != 1 {
		t.Fatalf("want 1 normalized op, got %d (%+v)", len(out), out)
	}
}

func TestNormalizeStableOrdering(t *testing.T) {
	t.Parallel()
	a := []Operation{
		FileMove{From: "z.go", To: "y.go"},
		ImportRewrite{Old: "x", New: "w"},
		FileMove{From: "a.go", To: "b.go"},
	}
	b := []Operation{
		ImportRewrite{Old: "x", New: "w"},
		FileMove{From: "a.go", To: "b.go"},
		FileMove{From: "z.go", To: "y.go"},
	}
	if got, want := Normalize(a), Normalize(b); !reflect.DeepEqual(got, want) {
		t.Fatalf("permutations did not normalize to same order:\n got %+v\nwant %+v", got, want)
	}
}

func TestValidateOperationsRejectsMalformed(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		ops  []Operation
	}{
		{"empty from", []Operation{FileMove{From: "", To: "b.go"}}},
		{"absolute from", []Operation{FileMove{From: "/etc/passwd", To: "b.go"}}},
		{"traversal", []Operation{FileMove{From: "../escape", To: "b.go"}}},
		{"identical", []Operation{FileMove{From: "a.go", To: "a.go"}}},
		{"empty import", []Operation{ImportRewrite{Old: "", New: "y"}}},
		{"identical import", []Operation{ImportRewrite{Old: "x", New: "x"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateOperations(tc.ops)
			if err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
			var rerr RewriteError
			if !errAs(err, &rerr) || rerr.Kind != RewriteErrorMalformedOperation {
				t.Fatalf("want RewriteError{MalformedOperation}, got %v", err)
			}
		})
	}
}

func TestValidateOperationsAcceptsWellFormed(t *testing.T) {
	t.Parallel()
	ops := []Operation{
		FileMove{From: "a/b.go", To: "c/d.go"},
		ImportRewrite{Old: "example.com/old", New: "example.com/new"},
	}
	if err := ValidateOperations(ops); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// errAs is a tiny shim so we don't import errors in every test file.
func errAs(err error, target any) bool {
	if err == nil {
		return false
	}
	if rt, ok := target.(*RewriteError); ok {
		if re, ok := err.(RewriteError); ok {
			*rt = re
			return true
		}
	}
	return false
}
