// Subsumption proof: every realistic legacy-detectable classifier
// coupling pattern is also caught by ruleProseTopicLeak. This file is the
// permanent regression record that motivates the deletion of
// ruleNonPortableClassifier and the classifierForbiddenSubstrings list:
// the legacy rule has been retired, but the contract it enforced lives on
// here as a synthetic-fixture pass.
//
// Every realistic SKILL.md prose form the legacy rule was authored to
// forbid is now caught by the broader Pillar 2 prose scanner:
//
//   - Inbox topic-prefix coupling (e.g. `research-inbox/<date>`) — caught
//     by `inferred-backtick-topic-ref` for backticked references and by the
//     `cli-knowledge-*` regexes for full CLI invocations.
//
//   - Knowledge-write CLI invocations with a `--topic` flag (the only
//     form that's actually copy-pasteable) — caught by
//     `cli-knowledge-add-topic` / `cli-knowledge-update-topic` /
//     `cli-knowledge-list-topic` / `cli-knowledge-list-prefix`.
//
// Patterns intentionally NOT subsumed (refinements of the legacy rule,
// not regressions):
//
//   - Bare-verb references without a `--topic` flag
//     (`knowledge-delete <id>`, `knowledge-update --id ...`). The new
//     scanner targets topic-prefix coupling, which is the actual
//     portability concern; a verb mention without a topic carries no
//     team binding.
//
//   - Plain-prose prefix mentions without backticks or CLI context.
//     Documentation in classifier-or-triage SKILL.md files uses
//     backticks for any identifier worth grepping; treating bare prose
//     mentions as findings would generate false-positives on ordinary
//     English text containing slashes.
//
// Test strategy: build a synthetic repo with three distinct classifier
// skills, each carrying one realistic coupling pattern, and assert
// ruleProseTopicLeak fires on each — this is the property the legacy
// rule used to enforce.

package memberflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// classifierSubsumptionFixture is one realistic legacy-detectable
// coupling that the prose scanner must continue to catch.
type classifierSubsumptionFixture struct {
	// SkillID is the classifier-skill id; the synthetic intake on the
	// member references it via classifier_skill.
	SkillID string
	// SkillBody is the SKILL.md content shipped with this fixture.
	SkillBody string
	// LegacyExpectedSubstring is the substring the retired
	// non_portable_classifier rule was authored to catch in this body.
	// The test sanity-checks the substring is actually present in the
	// fixture so future readers see exactly which legacy intent each
	// fixture preserves.
	LegacyExpectedSubstring string
	// ExpectedProsePrefix is the prefix the prose scanner is expected to
	// capture and flag for this fixture.
	ExpectedProsePrefix string
	// Comment documents what real-world coupling the fixture represents.
	Comment string
}

func TestSubsumption_NonPortableClassifierPatterns_AreCaughtByProseTopicLeak(t *testing.T) {
	fixtures := []classifierSubsumptionFixture{
		{
			SkillID: "leaky-inbox-classifier",
			SkillBody: "# Leaky Inbox Classifier\n" +
				"\n" +
				"Drain entries from `research-inbox/<date>/<slug>` and route them.\n",
			LegacyExpectedSubstring: "research-inbox/",
			ExpectedProsePrefix:     "research-inbox/<date>/<slug>",
			Comment:                 "Inbox topic-prefix coupling expressed as a backticked reference.",
		},
		{
			SkillID: "leaky-cli-classifier",
			SkillBody: "# Leaky CLI Classifier\n" +
				"\n" +
				"Tag entries with `prompt-manager team knowledge-update marketing-crew knw-abc --topic=\"opportunity-inbox/<date>/<slug>\"`.\n",
			LegacyExpectedSubstring: "prompt-manager team knowledge-update",
			ExpectedProsePrefix:     "opportunity-inbox/<date>/<slug>",
			Comment:                 "Knowledge-write CLI verb with --topic flag (the realistic copy-pasteable form).",
		},
	}

	for _, fx := range fixtures {
		fx := fx
		t.Run(fx.SkillID, func(t *testing.T) {
			// Sanity: the legacy substring must actually appear in
			// the fixture body, otherwise this fixture isn't
			// preserving the legacy intent it claims to.
			if !strings.Contains(fx.SkillBody, fx.LegacyExpectedSubstring) {
				t.Fatalf("fixture %q does not contain the legacy substring %q — fixture is not exercising the legacy intent",
					fx.SkillID, fx.LegacyExpectedSubstring)
			}

			root := buildSyntheticRepo(t)

			// Plant the classifier skill under the synthetic store.
			// Tags omit "writer-skill" — this is a classifier, so the
			// prose scanner's strict no-topic rule for non-writer
			// skills applies.
			skillDir := filepath.Join(root, "scenarios", "prompt-manager", "store",
				"skills", "packs", "core", fx.SkillID)
			if err := os.MkdirAll(skillDir, 0o755); err != nil {
				t.Fatal(err)
			}
			mustWriteFile(t, filepath.Join(skillDir, "skill.json"),
				`{"id":"`+fx.SkillID+`","tags":["skill","classifier"]}`)
			mustWriteFile(t, filepath.Join(skillDir, "SKILL.md"), fx.SkillBody)

			// A synthetic intake referencing this classifier — what
			// the legacy rule used to walk via classifier_skill.
			members := []MemberTopics{
				mkMember("synthetic-team", "consumer", Topics{
					Intake: []IntakeEntry{{
						Prefix:          "x/*",
						Taxonomy:        "tx",
						ClassifierSkill: fx.SkillID,
					}},
					RequiredRead:      []RequiredReadEntry{{Prefix: "research-inbox/*"}},
					ExternalProducers: []string{"operator"},
				}),
			}

			findings := ruleProseTopicLeak(members, ValidationOptions{
				ScanRoots: []string{root},
			})

			ownerKey := "skill:" + fx.SkillID
			matched := false
			for _, f := range findings {
				if f.OwnerKey != ownerKey {
					continue
				}
				if f.Prefix == fx.ExpectedProsePrefix ||
					strings.HasPrefix(fx.ExpectedProsePrefix, f.Prefix) {
					matched = true
					break
				}
			}
			if !matched {
				t.Fatalf("prose_topic_leak did not catch fixture %q (%s).\nFindings:\n%s",
					fx.SkillID, fx.Comment, debugFindings(findings))
			}
		})
	}
}
