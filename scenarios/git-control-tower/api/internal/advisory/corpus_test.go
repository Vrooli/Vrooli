package advisory

import "testing"

func TestGroundingCorpusCoversRequiredAdversarialSubjects(t *testing.T) {
	corpus, err := LoadGroundingCorpus()
	if err != nil {
		t.Fatal(err)
	}
	if corpus.CorpusID != "git-control-tower-grounding-v1" {
		t.Fatalf("corpus id = %q", corpus.CorpusID)
	}
	requiredKinds := map[string]bool{
		string(SubjectCurrent):     false,
		string(SubjectStaged):      false,
		string(SubjectCommit):      false,
		string(SubjectRange):       false,
		string(SubjectPullRequest): false,
	}
	requiredLabels := map[string]bool{
		"merge_parent_ambiguity": false,
		"mixed_staged_current":   false,
		"rename":                 false,
		"binary":                 false,
		"untracked":              false,
		"private_work_reference": false,
		"prompt_injection":       false,
		"stale_revision":         false,
	}
	knownDefectCases := 0
	for _, item := range corpus.Cases {
		requiredKinds[item.SubjectKind] = true
		for _, label := range item.Labels {
			requiredLabels[label] = true
		}
		if len(item.KnownDefects) > 0 {
			knownDefectCases++
			if len(item.ExpectedFindings) == 0 {
				t.Errorf("known-defect case %q has no expected findings", item.ID)
			}
		}
	}
	for kind, covered := range requiredKinds {
		if !covered {
			t.Errorf("subject kind %q is not covered", kind)
		}
	}
	for label, covered := range requiredLabels {
		if !covered {
			t.Errorf("adversarial label %q is not covered", label)
		}
	}
	if knownDefectCases == 0 {
		t.Fatal("review corpus has no planted known-defect case")
	}
}
