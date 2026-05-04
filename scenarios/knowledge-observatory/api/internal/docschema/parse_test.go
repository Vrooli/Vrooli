package docschema

import "testing"

func TestParseDocType(t *testing.T) {
	cases := []struct {
		input string
		want  DocType
	}{
		{"problems", DocTypeProblems},
		{"PROBLEMS", DocTypeProblems},
		{"problem", DocTypeProblems},
		{"progress", DocTypeProgress},
		{"seams", DocTypeSeams},
		{"seam", DocTypeSeams},
		{"invariants", DocTypeInvariants},
		{"assumptions", DocTypeAssumptions},
		{"error-semantics", DocTypeErrorSemantics},
		{"error_semantics", DocTypeErrorSemantics},
		{"errorsemantics", DocTypeErrorSemantics},
		{"security-posture", DocTypeSecurityPosture},
		{"temporal-flows", DocTypeTemporalFlows},
		{"coherence-notes", DocTypeCoherenceNotes},
		{"experience-audit", DocTypeExperienceAudit},
		{"perf-audit", DocTypePerfAudit},
		{"perf_audit", DocTypePerfAudit},
		{"perfaudit", DocTypePerfAudit},
		{"PERF-AUDIT", DocTypePerfAudit},
		{"quickstart", DocTypeQuickstart},
		{"architecture", DocTypeArchitecture},
		{"glossary", DocTypeGlossary},
		{"prd", DocTypePRD},
		{"readme", DocTypeReadme},
		{"manifest", DocTypeManifest},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := ParseDocType(tc.input)
			if err != nil {
				t.Fatalf("ParseDocType(%q) returned error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("ParseDocType(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseDocTypeUnknown(t *testing.T) {
	if _, err := ParseDocType("totally-not-a-type"); err == nil {
		t.Fatal("expected error for unknown doc type, got nil")
	}
}
