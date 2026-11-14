package rewrite

import (
	"errors"
	"testing"
)

func TestNormalize_RejectsEmpty(t *testing.T) {
	_, err := Normalize(nil)
	var re RewriteError
	if !errors.As(err, &re) || re.Kind != RewriteErrorInvalidInput {
		t.Fatalf("want RewriteErrorInvalidInput, got %v", err)
	}
}

func TestNormalize_CanonicalizesPaths(t *testing.T) {
	ops := []Operation{
		{FileMove: &FileMove{FromPath: "./src/a.ts", ToPath: "src/b.ts"}},
	}
	got, err := Normalize(ops)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if got[0].FileMove.FromPath != "src/a.ts" {
		t.Errorf("from_path should strip ./; got %q", got[0].FileMove.FromPath)
	}
}

func TestNormalize_RejectsAbsoluteAndParent(t *testing.T) {
	cases := []Operation{
		{FileMove: &FileMove{FromPath: "/abs/a.ts", ToPath: "b.ts"}},
		{FileMove: &FileMove{FromPath: "a.ts", ToPath: "/abs/b.ts"}},
		{FileMove: &FileMove{FromPath: "../outside.ts", ToPath: "b.ts"}},
		{ImportRewrite: &ImportRewrite{OldPath: "./a", NewPath: "/abs/b"}},
		{ImportRewrite: &ImportRewrite{OldPath: "../x", NewPath: "y"}},
	}
	for i, op := range cases {
		_, err := Normalize([]Operation{op})
		var re RewriteError
		if !errors.As(err, &re) || re.Kind != RewriteErrorInvalidOperation {
			t.Errorf("case %d: want RewriteErrorInvalidOperation, got %v", i, err)
		}
	}
}

func TestNormalize_RejectsSamePaths(t *testing.T) {
	cases := []Operation{
		{FileMove: &FileMove{FromPath: "a.ts", ToPath: "./a.ts"}},
		{ImportRewrite: &ImportRewrite{OldPath: "x", NewPath: "x"}},
	}
	for i, op := range cases {
		_, err := Normalize([]Operation{op})
		var re RewriteError
		if !errors.As(err, &re) || re.Kind != RewriteErrorInvalidOperation {
			t.Errorf("case %d: want RewriteErrorInvalidOperation, got %v", i, err)
		}
	}
}

func TestNormalize_RejectsBothOrNeitherOneof(t *testing.T) {
	cases := []Operation{
		{},
		{FileMove: &FileMove{FromPath: "a", ToPath: "b"}, ImportRewrite: &ImportRewrite{OldPath: "x", NewPath: "y"}},
	}
	for i, op := range cases {
		_, err := Normalize([]Operation{op})
		var re RewriteError
		if !errors.As(err, &re) || re.Kind != RewriteErrorInvalidOperation {
			t.Errorf("case %d: want RewriteErrorInvalidOperation, got %v", i, err)
		}
	}
}

func TestNormalize_StableSort(t *testing.T) {
	a := Operation{ImportRewrite: &ImportRewrite{OldPath: "z/a", NewPath: "z/b"}}
	b := Operation{FileMove: &FileMove{FromPath: "src/b.ts", ToPath: "src/c.ts"}}
	c := Operation{FileMove: &FileMove{FromPath: "src/a.ts", ToPath: "src/d.ts"}}

	got, err := Normalize([]Operation{a, b, c})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if got[0].OperationTag() != "file_move" || got[0].FileMove.FromPath != "src/a.ts" {
		t.Errorf("expected file_move src/a.ts first; got %+v", got[0])
	}
	if got[1].FileMove.FromPath != "src/b.ts" {
		t.Errorf("expected file_move src/b.ts second; got %+v", got[1])
	}
	if got[2].OperationTag() != "import_rewrite" {
		t.Errorf("expected import_rewrite last; got %+v", got[2])
	}
}

func TestNormalize_Dedup(t *testing.T) {
	a := Operation{FileMove: &FileMove{FromPath: "src/a.ts", ToPath: "src/b.ts"}}
	b := Operation{FileMove: &FileMove{FromPath: "./src/a.ts", ToPath: "src/b.ts"}}
	got, err := Normalize([]Operation{a, b})
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 op after dedup; got %d", len(got))
	}
}
