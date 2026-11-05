package wizard

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"business-health/internal/checks"
	"business-health/internal/extraction"

	intent "intent-go"

	"github.com/stretchr/testify/require"
)

// answersFor builds a complete, anchor-conformant answer set. Variations
// come from the target counts per tier and a text seed.
func answersFor(seed string, p0, p1, p2 int) []Answer {
	mkTargets := func(n int, tier string) []OTAnswer {
		out := make([]OTAnswer, 0, n)
		for i := 0; i < n; i++ {
			out = append(out, OTAnswer{
				Title:       fmt.Sprintf("%s outcome %s-%d", seed, tier, i+1),
				Description: fmt.Sprintf("Delivers the %s capability number %d.", seed, i+1),
			})
		}
		return out
	}
	return []Answer{
		{QuestionID: "section_overview", Text: "- **Purpose**: " + seed + " does things.\n- **Value promise**: it helps."},
		{QuestionID: "targets_p0", Targets: mkTargets(p0, "P0")},
		{QuestionID: "targets_p1", Targets: mkTargets(p1, "P1")},
		{QuestionID: "targets_p2", Targets: mkTargets(p2, "P2")},
		{QuestionID: "section_tech_direction_snapshot", Text: "- Preferred stacks: Go, React."},
		{QuestionID: "section_dependencies_launch_plan", Text: "- Required resources: none."},
		{QuestionID: "section_ux_branding", Text: "- Accessibility: WCAG AA.\n- Look & feel: calm."},
	}
}

// [REQ:BH-WIZ-002] Round-trip property: for a corpus of generated answer
// sets, wizard output validates with ZERO findings — clean by
// construction, not by luck.
func TestRoundTripCleanByConstruction(t *testing.T) {
	corpus := []struct {
		seed       string
		p0, p1, p2 int
	}{
		{"alpha", 1, 1, 1},
		{"beta", 3, 2, 1},
		{"gamma-thing", 2, 1, 4},
		{"x9", 5, 5, 5},
	}
	for _, tc := range corpus {
		tc := tc
		t.Run(tc.seed, func(t *testing.T) {
			target := t.TempDir()
			e := NewEngine(t.TempDir(), nil, nil)
			s, err := e.StartSession("wizard-fixture", target, false)
			require.NoError(t, err)
			invalid, err := e.SubmitAnswers(&s, answersFor(tc.seed, tc.p0, tc.p1, tc.p2))
			require.NoError(t, err)
			require.Empty(t, invalid)
			require.True(t, Complete(s), "remaining: %v", Remaining(s))

			written, err := e.Apply(s)
			require.NoError(t, err)
			require.Contains(t, written, "PRD.md")
			require.Contains(t, written, "requirements/index.json")
			require.Contains(t, written, "requirements/README.md")

			contract, err := extraction.NewFileExtractor().Load("wizard-fixture", target)
			require.NoError(t, err)
			var findings []intent.Finding
			for _, chk := range checks.Registry() {
				findings = append(findings, chk.Run(context.Background(), contract)...)
			}
			require.Empty(t, findings, "wizard output must validate clean, got %+v", findings)
		})
	}
}

// [REQ:BH-WIZ-001] The question model covers exactly the validator's
// section model: every required template section maps to a question, and
// required content anchors are enforced at answer time.
func TestQuestionsMirrorTemplate(t *testing.T) {
	qs := questionIndex()
	for _, section := range intent.DefaultPRDTemplate() {
		key := intent.NormalizeSectionTitle(section.Title)
		if key == "operational targets" {
			for _, tier := range []string{"p0", "p1", "p2"} {
				require.Contains(t, qs, "targets_"+tier)
			}
			continue
		}
		if !section.Required {
			continue
		}
		q, ok := qs["section_"+slugify(key)]
		require.True(t, ok, "no question for required section %q", section.Title)
		for _, rule := range section.Rules {
			if rule.Kind == "contains" && rule.Required {
				require.Contains(t, q.RequiredAnchors, rule.Pattern)
			}
		}
	}

	// Anchor enforcement: an Overview answer without "Purpose" is invalid.
	reason := ValidateAnswer(qs["section_overview"], Answer{QuestionID: "section_overview", Text: "no anchor here"})
	require.NotEmpty(t, reason)
}

// [REQ:BH-WIZ-003] Sessions are resumable and dry-run is structural:
// Scaffold never writes; Apply refuses while required questions are open.
func TestSessionResumeAndDryRun(t *testing.T) {
	dataDir := t.TempDir()
	target := t.TempDir()
	e := NewEngine(dataDir, nil, nil)
	s, err := e.StartSession("resume-fixture", target, false)
	require.NoError(t, err)
	_, err = e.SubmitAnswers(&s, []Answer{{QuestionID: "section_overview", Text: "- **Purpose**: partial."}})
	require.NoError(t, err)

	// Scaffold on an incomplete session renders diffs but blocks apply.
	files, blocking, err := e.Scaffold(s)
	require.NoError(t, err)
	require.NotEmpty(t, files)
	require.NotEmpty(t, blocking)
	entries, err := osReadDirNames(target)
	require.NoError(t, err)
	require.Empty(t, entries, "Scaffold must not write")
	_, err = e.Apply(s)
	require.Error(t, err)

	// A fresh engine over the same data dir resumes the session.
	resumed, err := NewEngine(dataDir, nil, nil).StartSession("resume-fixture", target, false)
	require.NoError(t, err)
	require.Equal(t, s.ID, resumed.ID)
	require.Contains(t, resumed.Answers, "section_overview")

	// Reset discards it.
	fresh, err := NewEngine(dataDir, nil, nil).StartSession("resume-fixture", target, true)
	require.NoError(t, err)
	require.NotEqual(t, s.ID, fresh.ID)
	require.Empty(t, fresh.Answers)
}

// [REQ:BH-WIZ-003] No network seams exist in the wizard path: the only
// outward interface is Hinter, and the default is a silent no-op.
func TestDedupHookDegradesSilently(t *testing.T) {
	e := NewEngine(t.TempDir(), nil, nil)
	s, err := e.StartSession("hint-fixture", t.TempDir(), false)
	require.NoError(t, err)
	_, err = e.SubmitAnswers(&s, []Answer{{QuestionID: "targets_p0", Targets: []OTAnswer{{Title: "Thing", Description: "Does thing."}}}})
	require.NoError(t, err)
	require.Nil(t, e.Hints(s))
}

// [REQ:BH-WIZ-005] A wired hinter surfaces hints; a failing one stays
// silent (fake client, no network).
func TestDedupHookWithHinter(t *testing.T) {
	e := NewEngine(t.TempDir(), fakeHinter{hints: []CapabilityHint{{Scenario: "other", Capability: "Thing", Anchor: "scenarios/other/PRD.md#OT-P0-001", Score: 0.91}}}, nil)
	s, err := e.StartSession("hint-fixture", t.TempDir(), false)
	require.NoError(t, err)
	_, err = e.SubmitAnswers(&s, []Answer{{QuestionID: "targets_p0", Targets: []OTAnswer{{Title: "Thing", Description: "Does thing."}}}})
	require.NoError(t, err)
	hints := e.Hints(s)
	require.Len(t, hints, 1)
	require.Equal(t, "other", hints[0].Scenario)
}

type fakeHinter struct{ hints []CapabilityHint }

func (f fakeHinter) Hints(string, []OTAnswer) []CapabilityHint { return f.hints }

// [REQ:BH-WIZ-004] Identical inputs produce identical artifacts —
// determinism across engines and sessions.
func TestScaffoldDeterministic(t *testing.T) {
	render := func() map[string]string {
		e := NewEngine(t.TempDir(), nil, nil)
		s, err := e.StartSession("det-fixture", t.TempDir(), false)
		require.NoError(t, err)
		_, err = e.SubmitAnswers(&s, answersFor("det", 2, 1, 1))
		require.NoError(t, err)
		files, _, err := e.Scaffold(s)
		require.NoError(t, err)
		out := map[string]string{}
		for _, f := range files {
			out[f.Path] = f.After
		}
		return out
	}
	require.Equal(t, render(), render())
}

func osReadDirNames(dir string) ([]string, error) {
	entries, err := osReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}

// Prefix trims: an unusable slug fails loudly instead of minting a bad ID.
func TestRequirementPrefix(t *testing.T) {
	s := Session{Scenario: "image-tools", Answers: map[string]Answer{}}
	p, err := requirementPrefix(s)
	require.NoError(t, err)
	require.Equal(t, "IMAGETOO", p)

	s.Answers["requirement_prefix"] = Answer{QuestionID: "requirement_prefix", Text: "img"}
	p, err = requirementPrefix(s)
	require.NoError(t, err)
	require.Equal(t, "IMG", p)

	s.Answers["requirement_prefix"] = Answer{QuestionID: "requirement_prefix", Text: "9bad prefix"}
	_, err = requirementPrefix(s)
	require.Error(t, err)

	require.Contains(t, strings.Join(sortedAnswerIDs(s), ","), "requirement_prefix")
}
