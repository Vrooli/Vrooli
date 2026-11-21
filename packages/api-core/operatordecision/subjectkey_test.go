package operatordecision

import (
	"strings"
	"testing"
)

func TestSubjectKeyFromAlwaysSatisfiesTheHubRule(t *testing.T) {
	for _, name := range []string{
		"backup-failure:vol-1", "scenarios/foo:BAS-D-070", "has spaces/and#marks", strings.Repeat("c", 90) + "/" + strings.Repeat("I", 90),
		"prog/" + strings.Repeat("é", 80), "/leading/slash", "..", "-",
	} {
		if got := SubjectKeyFrom(name); !SubjectKeyRule.MatchString(got) {
			t.Errorf("SubjectKeyFrom(%q) = %q breaks the Hub's rule", name, got)
		}
	}
}

func TestSubjectKeyFromKeepsAnAcceptableNameAndNeverSharesASubject(t *testing.T) {
	if got := SubjectKeyFrom("backup-failure:vol-1"); got != "backup-failure:vol-1" {
		t.Errorf("an acceptable name is used as it is, got %q", got)
	}
	if SubjectKeyFrom("prog 1/A#4") == SubjectKeyFrom("prog_1/A_4") {
		t.Error("two different names that clean to the same text must not share a subject")
	}
	if SubjectKeyFrom("prog 1/A#4") != SubjectKeyFrom("prog 1/A#4") {
		t.Error("the same name always gives the same key")
	}
}
