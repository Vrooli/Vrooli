package rewrite

import "testing"

func TestDerivePlanID_DeterministicAcrossRuns(t *testing.T) {
	ops := []Operation{
		{FileMove: &FileMove{FromPath: "src/a.ts", ToPath: "src/b.ts"}},
		{ImportRewrite: &ImportRewrite{OldPath: "./a", NewPath: "./b"}},
	}
	first := DerivePlanID(ops)
	for i := 0; i < 9; i++ {
		if got := DerivePlanID(ops); got != first {
			t.Fatalf("DerivePlanID drift on iter %d: %s != %s", i, got, first)
		}
	}
}

func TestDerivePlanID_DiffersForDifferentInputs(t *testing.T) {
	a := []Operation{{FileMove: &FileMove{FromPath: "src/a.ts", ToPath: "src/b.ts"}}}
	b := []Operation{{FileMove: &FileMove{FromPath: "src/a.ts", ToPath: "src/c.ts"}}}
	if DerivePlanID(a) == DerivePlanID(b) {
		t.Errorf("different operations should produce different plan IDs")
	}
}

func TestDerivePlanID_OrderIndependent_AfterNormalize(t *testing.T) {
	a := []Operation{
		{FileMove: &FileMove{FromPath: "src/a.ts", ToPath: "src/b.ts"}},
		{ImportRewrite: &ImportRewrite{OldPath: "./a", NewPath: "./b"}},
	}
	b := []Operation{
		{ImportRewrite: &ImportRewrite{OldPath: "./a", NewPath: "./b"}},
		{FileMove: &FileMove{FromPath: "src/a.ts", ToPath: "src/b.ts"}},
	}
	na, err := Normalize(a)
	if err != nil {
		t.Fatalf("normalize a: %v", err)
	}
	nb, err := Normalize(b)
	if err != nil {
		t.Fatalf("normalize b: %v", err)
	}
	if DerivePlanID(na) != DerivePlanID(nb) {
		t.Errorf("normalize+hash should be order-independent")
	}
}
