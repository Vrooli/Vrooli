package reconcile

import "testing"

func TestVerdictVocabularyIsClosed(t *testing.T) {
	want := []Verdict{"matches", "drifted", "missing", "extra", "unverifiable"}
	if len(VerdictVocabulary) != len(want) {
		t.Fatalf("got %d verdicts", len(VerdictVocabulary))
	}
	for i := range want {
		if VerdictVocabulary[i] != want[i] {
			t.Fatalf("verdict[%d] = %q", i, VerdictVocabulary[i])
		}
	}
}

func TestFindingCarriesAllFiveTypedFields(t *testing.T) {
	v := Verify("/repo/scenarios", "demo", "page", []Result{{Region: "r", Required: true, Reason: "none"}, {FilePath: "ui/src/Extra.tsx", Extra: true}}, nil)
	for _, row := range v.Regions {
		f := row.Finding
		if f.CatalogID == "" || f.Scope == "" || f.Owner == "" || f.Severity == "" {
			t.Fatalf("incomplete finding: %#v", f)
		}
		if f.Blocking {
			t.Fatal("phase-four findings must remain advisory")
		}
	}
}

func TestExitCodeFollowsVerdicts(t *testing.T) {
	passing := Verify("/repo/scenarios", "demo", "page", []Result{{Region: "r", FilePath: "x", Provenance: ProvenanceCustom}}, nil)
	if !passing.Passes {
		t.Fatal("custom resolved region should pass")
	}
	failing := Verify("/repo/scenarios", "demo", "page", []Result{{Region: "r", Required: true, Reason: "none"}}, nil)
	if failing.Passes {
		t.Fatal("required missing region should fail")
	}
}
