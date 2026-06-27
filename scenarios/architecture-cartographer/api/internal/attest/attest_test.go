package attest

import (
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestValidate_CitationRequiredForCodeBases(t *testing.T) {
	codeBases := []commonv1.Basis{
		commonv1.Basis_BASIS_DERIVED,
		commonv1.Basis_BASIS_VALIDATED,
		commonv1.Basis_BASIS_CONTRADICTED,
	}
	for _, basis := range codeBases {
		uncited, err := New("claim").Basis(basis).BuildValidated()
		if err == nil {
			t.Fatalf("basis %s without citation should be invalid", basis)
		}
		_ = uncited
		cited, err := New("claim").Basis(basis).CiteCode("api/internal/x/x.go:1", "").BuildValidated()
		if err != nil {
			t.Fatalf("basis %s with citation should be valid: %v", basis, err)
		}
		if len(cited.GetCitations()) != 1 {
			t.Fatalf("expected 1 citation, got %d", len(cited.GetCitations()))
		}
	}
}

func TestValidate_UncitedBasesAllowed(t *testing.T) {
	for _, basis := range []commonv1.Basis{commonv1.Basis_BASIS_DECLARED_UNVERIFIED, commonv1.Basis_BASIS_ABSENT} {
		if _, err := New("claim").Basis(basis).BuildValidated(); err != nil {
			t.Fatalf("basis %s may be uncited: %v", basis, err)
		}
	}
}

func TestValidate_EmptyClaim(t *testing.T) {
	if err := Validate(New("   ").Basis(commonv1.Basis_BASIS_ABSENT).Build()); err == nil {
		t.Fatal("empty claim should be invalid")
	}
}

func TestConvergenceBasis(t *testing.T) {
	cases := []struct {
		hasCode, hasDoc, agree bool
		want                   commonv1.Basis
	}{
		{true, true, true, commonv1.Basis_BASIS_VALIDATED},
		{true, true, false, commonv1.Basis_BASIS_CONTRADICTED},
		{true, false, false, commonv1.Basis_BASIS_DERIVED},
		{false, true, false, commonv1.Basis_BASIS_DECLARED_UNVERIFIED},
		{false, false, false, commonv1.Basis_BASIS_ABSENT},
	}
	for _, c := range cases {
		if got := ConvergenceBasis(c.hasCode, c.hasDoc, c.agree); got != c.want {
			t.Fatalf("ConvergenceBasis(%v,%v,%v) = %s, want %s", c.hasCode, c.hasDoc, c.agree, got, c.want)
		}
	}
}

func TestBuilder_GapsAndFollowUps(t *testing.T) {
	a := New("claim").
		Basis(commonv1.Basis_BASIS_DERIVED).
		CiteCode("x.go:1", "note").
		Gap("symbol-level links pending").
		FollowUp("run: architecture-cartographer slice show x").
		Build()
	if len(a.GetGaps()) != 1 || len(a.GetSuggestedFollowUps()) != 1 {
		t.Fatalf("gaps/followups not recorded: %+v", a)
	}
}
