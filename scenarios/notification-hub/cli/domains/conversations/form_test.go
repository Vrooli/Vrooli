package conversations

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

// formPacket attaches a Markdown plan and a folder of variant images, the
// attachments the form examples refer to.
func formPacket(t *testing.T) *packet {
	t.Helper()
	dir := t.TempDir()
	p := newPacket()
	require.NoError(t, p.addAttachment(attachSpec{Path: writeFile(t, dir, "plan.md", "# T\n## Storage\n### Retention\n## Billing\n")}, v1.AskAttachment_ROLE_EVIDENCE))
	writeFile(t, dir, "mocks/a.png", "a")
	writeFile(t, dir, "mocks/b.png", "b")
	require.NoError(t, p.addAttachment(attachSpec{Path: dir + "/mocks"}, v1.AskAttachment_ROLE_EVIDENCE))
	return p
}

func TestFormBuildsEveryQuestionType(t *testing.T) {
	form := `
questions:
  - id: layout
    type: variant
    prompt: Which layout?
    help: Pick the clearest.
    required: true
    choices:
      - {key: cards, label: Card stack, detail: One card per item, attachment: mocks/a.png}
      - {key: table, attachment: mocks}
    recommended: cards
    recommendation_reason: Reads best on a phone.
  - id: scope
    type: multi_choice
    prompt: Which parts?
    choices: [{key: api}, {key: ui}, {key: docs}]
    recommended: [api, ui]
  - id: plan
    type: section_review
    prompt: Review each section.
    sections_from: plan.md
    recommended:
      sections:
        - {section_key: storage, verdict: approve}
        - {section_key: retention, verdict: change, note: keep 30 days}
  - id: notes
    type: text
    prompt: Anything else?
    recommended: No, ship it.
`
	questions := mustBuild(t, form, formPacket(t))
	require.Len(t, questions, 4)

	layout := questions[0]
	require.Equal(t, "variant", layout.GetType())
	require.True(t, layout.GetRequired())
	require.Equal(t, "Pick the clearest.", layout.GetHelp())
	require.Equal(t, "cards", layout.GetChoices()[0].GetKey())
	require.Equal(t, "mocks/a.png", layout.GetChoices()[0].GetAttachment(), "a folder entry is addressed as folder/entry")
	require.Equal(t, "table", layout.GetChoices()[1].GetLabel(), "label defaults to the key")
	require.Equal(t, "mocks", layout.GetChoices()[1].GetAttachment())
	require.Equal(t, &v1.AskQuestionResponse{QuestionId: "layout", Selected: []string{"cards"}}, layout.GetRecommended())
	require.Equal(t, "Reads best on a phone.", layout.GetRecommendationReason())

	require.Equal(t, []string{"api", "ui"}, questions[1].GetRecommended().GetSelected())

	plan := questions[2]
	require.Equal(t, "plan.md", plan.GetSectionsFrom(), "the Hub reads the headings, so a revision that replaces the plan changes the review")
	require.Empty(t, plan.GetSections(), "the CLI sends no sections of its own for a review that follows a file")
	rec := plan.GetRecommended()
	require.Equal(t, "plan", rec.GetQuestionId())
	require.Equal(t, "change", rec.GetSections()[1].GetVerdict())
	require.Equal(t, "keep 30 days", rec.GetSections()[1].GetNote())

	require.Equal(t, "No, ship it.", questions[3].GetRecommended().GetText())
}

func TestFormAcceptsJSONAndABareList(t *testing.T) {
	asMapping := mustBuild(t, `{"questions":[{"id":"q","type":"text","prompt":"Why?"}]}`, newPacket())
	require.Equal(t, "q", asMapping[0].GetId())

	asList := mustBuild(t, "- id: q\n  type: text\n  prompt: Why?\n", newPacket())
	require.Equal(t, "q", asList[0].GetId())
}

func TestFormRecommendedAsAFullResponseMapping(t *testing.T) {
	form := `
- id: scope
  type: multi_choice
  prompt: Which?
  choices: [{key: a}, {key: b}]
  recommended: {selected: b, text: because}
`
	questions := mustBuild(t, form, newPacket())
	require.Equal(t, []string{"b"}, questions[0].GetRecommended().GetSelected(), "a single string selects one key")
	require.Equal(t, "because", questions[0].GetRecommended().GetText())
}

func TestFormPassesAnUnknownTypeThroughAsTextForTheHubToDegrade(t *testing.T) {
	questions := mustBuild(t, "- {id: q, type: slider, prompt: 'How much?', recommended: lots}\n", newPacket())
	require.Equal(t, "slider", questions[0].GetType())
	require.Equal(t, "lots", questions[0].GetRecommended().GetText())
}

// buildWithWarnings builds a form against a packet and returns the questions
// and the local warnings it produced.
func buildWithWarnings(t *testing.T, form string, p *packet) ([]*v1.AskQuestion, string) {
	t.Helper()
	p.Warnings = nil
	questions := buildQuestions([]byte(form), "form.yaml", p)
	return questions, strings.Join(p.Warnings, "\n")
}

// mustBuild builds a form that must need no warnings.
func mustBuild(t *testing.T, form string, p *packet) []*v1.AskQuestion {
	t.Helper()
	questions, warnings := buildWithWarnings(t, form, p)
	require.Empty(t, warnings)
	return questions
}

func TestFormMistakesLeaveOutOnlyWhatIsBrokenAndSayWhat(t *testing.T) {
	const ok = "- {id: ok, type: text, prompt: Fine}\n"
	for name, tc := range map[string]struct {
		form string
		want string
		kept int // questions still sent
	}{
		"empty file":         {"", "the file is empty", 0},
		"no questions":       {"questions: []\n", "declares no questions", 0},
		"scalar root":        {"hello\n", "expected a `questions:` list", 0},
		"not yaml":           {"questions: [unclosed\n", "not valid YAML or JSON", 0},
		"questions not list": {"questions: nope\n", "`questions` must be a list", 0},
		"question not map":   {"- just text\n" + ok, "a question is a mapping", 1},
		"missing id":         {"- {type: text, prompt: P}\n" + ok, "question 1: needs an id", 1},
		"missing prompt":     {"- {id: q, type: text}\n" + ok, `question 1 ("q"): needs a prompt`, 1},
		"missing type":       {"- {id: q, prompt: P}\n" + ok, "needs a type", 1},
		"duplicate id":       {"- {id: q, type: text, prompt: P}\n- {id: q, type: text, prompt: P}\n", `id "q" is already used`, 1},
		"typo in a key":      {"- {id: q, type: single_choice, prompt: P, choice: [{key: a}]}\n" + ok, `unknown key "choice"`, 1},
		"unknown top key":    {"questions: [{id: q, type: text, prompt: P}]\nextra: 1\n", `Ignored unknown key "extra"`, 1},
		"choices missing":    {"- {id: q, type: single_choice, prompt: P}\n", "needs choices", 0},
		"choice without key": {"- {id: q, type: single_choice, prompt: P, choices: [{label: A}]}\n", "choice 1: needs a key", 0},
		"duplicate choice":   {"- {id: q, type: single_choice, prompt: P, choices: [{key: a}, {key: a}]}\n", `choice key "a" is used twice`, 0},
		"choices on text":    {"- {id: q, type: text, prompt: P, choices: [{key: a}]}\n", "not text", 0},
		"no sections":        {"- {id: q, type: section_review, prompt: P}\n", "sections or sections_from", 0},
		"both section kinds": {"- {id: q, type: section_review, prompt: P, sections_from: plan.md, sections: [{key: s, title: S}]}\n", "not both", 0},
		"sections on text":   {"- {id: q, type: text, prompt: P, sections_from: plan.md}\n", "section_review", 0},
		"section no title":   {"- {id: q, type: section_review, prompt: P, sections: [{key: s}]}\n", "needs a key and a title", 0},
		"bad unanswered":     {"- {id: q, type: section_review, prompt: P, unanswered_verdict: skip, sections: [{key: s, title: S}]}\n", "approve, change, or drop", 0},
		"unanswered on text": {"- {id: q, type: text, prompt: P, unanswered_verdict: drop}\n", "section_review", 0},
	} {
		questions, warnings := buildWithWarnings(t, tc.form, formPacket(t))
		require.Len(t, questions, tc.kept, name)
		require.Contains(t, warnings, tc.want, name)
		require.Regexp(t, "Left out|Ignored", warnings, name+": warnings must say what was dropped")
	}
}

func TestFormKeepsAQuestionWhenOnlyItsRecommendationIsBroken(t *testing.T) {
	const choices = "type: single_choice, prompt: P, choices: [{key: a}, {key: b}]"
	for name, tc := range map[string]struct{ question, want string }{
		"unknown key":        {"{id: q, " + choices + ", recommended: z}", `"z" is not one of`},
		"two for single":     {"{id: q, " + choices + ", recommended: [a, b]}", "exactly one"},
		"list for text":      {"{id: q, type: text, prompt: P, recommended: [a]}", "list of keys fits choice"},
		"scalar for section": {"{id: q, type: section_review, prompt: P, sections: [{key: s, title: S}], recommended: approve}", "mapping"},
		"unknown rec key":    {"{id: q, type: text, prompt: P, recommended: {txt: x}}", `unknown key "txt"`},
		"bad verdict":        {"{id: q, type: section_review, prompt: P, sections: [{key: s, title: S}], recommended: {sections: [{section_key: s, verdict: maybe}]}}", "approve, change, or drop"},
		"unknown section":    {"{id: q, type: section_review, prompt: P, sections: [{key: s, title: S}], recommended: {sections: [{section_key: t, verdict: drop}]}}", `"t" is not a section`},
	} {
		questions, warnings := buildWithWarnings(t, "- "+tc.question+"\n", newPacket())
		require.Len(t, questions, 1, name)
		require.Nil(t, questions[0].GetRecommended(), name)
		require.Contains(t, warnings, "left out the recommendation", name)
		require.Contains(t, warnings, tc.want, name)
	}
}

func TestFormPassesAnAttachmentNameThisSendDoesNotCarryToTheHub(t *testing.T) {
	// A revision's form may point at an attachment the open ask already holds;
	// only the Hub knows, and it reports a name it does not find.
	questions, warnings := buildWithWarnings(t, `
- {id: v, type: variant, prompt: P, choices: [{key: a, attachment: other.png}, {key: b, attachment: mocks/b.png}]}
- {id: s, type: section_review, prompt: P, sections: [{key: x, title: X, attachment: gone.md}]}
- {id: r, type: section_review, prompt: P, sections_from: earlier-plan.md}
`, formPacket(t))
	require.Empty(t, warnings)
	require.Len(t, questions, 3)
	require.Equal(t, "other.png", questions[0].GetChoices()[0].GetAttachment())
	require.Equal(t, "mocks/b.png", questions[0].GetChoices()[1].GetAttachment())
	require.Equal(t, "gone.md", questions[1].GetSections()[0].GetAttachment())
	require.Equal(t, "earlier-plan.md", questions[2].GetSectionsFrom())
}

func TestFormRefusesALocalFileThatIsNotAttached(t *testing.T) {
	stray := writeFile(t, t.TempDir(), "stray.md", "## A\n")
	questions, warnings := buildWithWarnings(t, "- {id: q, type: section_review, prompt: P, sections_from: "+stray+"}\n", newPacket())
	require.Empty(t, questions)
	require.Contains(t, warnings, "is a local file that is not attached; add --attach")

	questions, warnings = buildWithWarnings(t, "- {id: v, type: variant, prompt: P, choices: [{key: a, attachment: "+stray+"}]}\n", newPacket())
	require.Len(t, questions, 1, "a choice's picture is optional")
	require.Empty(t, questions[0].GetChoices()[0].GetAttachment(), "a host path is never sent as an attachment name")
	require.Contains(t, warnings, `choice "a" shows no attachment`)
}

func TestFormSectionsFromAnAttachedFileNamesItByAttachmentName(t *testing.T) {
	dir := t.TempDir()
	p := newPacket()
	require.NoError(t, p.addAttachment(attachSpec{Path: writeFile(t, dir, "My Plan.md", "## A\n")}, v1.AskAttachment_ROLE_EVIDENCE))
	questions := mustBuild(t, "- {id: q, type: section_review, prompt: P, sections_from: "+dir+"/My Plan.md}\n", p)
	require.Equal(t, "My-Plan.md", questions[0].GetSectionsFrom(), "a path is turned into the name the Hub gives the attachment")
}

func TestFormCapsExplicitSectionsAtFortyWithAWarning(t *testing.T) {
	var explicit strings.Builder
	explicit.WriteString("- {id: q, type: section_review, prompt: P, sections: [")
	for i := 0; i < 41; i++ {
		fmt.Fprintf(&explicit, "{key: s%d, title: S%d},", i, i)
	}
	explicit.WriteString("]}\n")
	questions, warnings := buildWithWarnings(t, explicit.String(), newPacket())
	require.Len(t, questions[0].GetSections(), maxReviewSections)
	require.Contains(t, warnings, "it has 41 sections")
}

func TestReadFormLeavesOutTheWholeFormWhenTheFileIsMissing(t *testing.T) {
	p := newPacket()
	require.Empty(t, readForm("/nonexistent/form.yaml", p))
	require.Len(t, p.Warnings, 1)
	require.Contains(t, p.Warnings[0], "Left out the whole --form")
	require.Contains(t, p.Warnings[0], "The ask is sent without its form")
}

func TestFormCarriesUnansweredVerdictOnSectionReviews(t *testing.T) {
	form := "- {id: plan, type: section_review, prompt: P, unanswered_verdict: drop, sections: [{key: s, title: S}]}\n"
	questions := mustBuild(t, form, newPacket())
	require.Equal(t, "drop", questions[0].GetUnansweredVerdict())

	questions = mustBuild(t, "- {id: plan, type: section_review, prompt: P, sections: [{key: s, title: S}]}\n", newPacket())
	require.Empty(t, questions[0].GetUnansweredVerdict(), "empty means approve; the Hub owns the default")
}
