package teamconfig

import "testing"

func TestFiniteLeaderRequiresExactBoundedConfiguration(t *testing.T) {
	for _, field := range []string{"effort", "revision", "prompt", "source", "profile", "source-limit", "whitespace", "supervision"} {
		t.Run(field, func(t *testing.T) {
			b := &FiniteLeader{EffortRef: "any:effort", AcceptedRevision: "r7", CoordinatorPromptRef: "pm:coordinator", SourceRefs: []string{"owner:assignment"}}
			profile := "qualified"
			var supervision *Supervision
			switch field {
			case "effort":
				b.EffortRef = ""
			case "revision":
				b.AcceptedRevision = ""
			case "prompt":
				b.CoordinatorPromptRef = ""
			case "source":
				b.SourceRefs = nil
			case "profile":
				profile = ""
			case "source-limit":
				b.SourceRefs = make([]string, 9)
			case "whitespace":
				b.AcceptedRevision = " r7"
			case "supervision":
				supervision = &Supervision{}
			}
			if err := b.Validate(profile, supervision); err == nil {
				t.Fatal("incomplete or conflicting binding accepted")
			}
		})
	}
}
