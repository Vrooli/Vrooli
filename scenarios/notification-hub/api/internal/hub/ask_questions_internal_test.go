package hub

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func choiceQuestion(id, questionType string, keys ...string) AskQuestion {
	question := AskQuestion{ID: id, Type: questionType, Prompt: "Which " + id + "?"}
	for _, key := range keys {
		question.Choices = append(question.Choices, AskQuestionChoice{Key: key, Label: strings.ToUpper(key)})
	}
	return question
}

func sectionQuestion(id string, keys ...string) AskQuestion {
	question := AskQuestion{ID: id, Type: QuestionSectionReview, Prompt: "Plan sections"}
	for _, key := range keys {
		question.Sections = append(question.Sections, AskQuestionSection{Key: key, Title: strings.ToUpper(key[:1]) + key[1:]})
	}
	return question
}

func warningCodes(warnings []PacketWarning) []string {
	var codes []string
	for _, warning := range warnings {
		codes = append(codes, warning.Code+"@"+warning.Field)
	}
	return codes
}

func TestNormalizeQuestionsKeepsAValidFormAndTrimsIt(t *testing.T) {
	in := []AskQuestion{
		{
			ID: " layout ", Type: " Variant ", Prompt: "  Layout?  ", Help: " pick one ", Required: true,
			Choices:     []AskQuestionChoice{{Key: "cards", Label: " Cards ", Detail: " stacked ", Attachment: "cards-png"}, {Key: "table"}},
			Recommended: &AskQuestionResponse{Selected: []string{"cards"}}, RecommendationReason: " reads best ",
		},
		sectionQuestion("plan", "storage", "billing"),
		{ID: "why", Type: "text", Prompt: "Anything else?"},
	}
	got, warnings := normalizeQuestions(in, map[string]bool{"cards-png": true})
	require.Empty(t, warnings)
	require.Len(t, got, 3)
	require.Equal(t, AskQuestion{
		ID: "layout", Type: QuestionVariant, Prompt: "Layout?", Help: "pick one", Required: true,
		Choices:              []AskQuestionChoice{{Key: "cards", Label: "Cards", Detail: "stacked", Attachment: "cards-png"}, {Key: "table", Label: "table"}},
		Recommended:          &AskQuestionResponse{QuestionID: "layout", Selected: []string{"cards"}},
		RecommendationReason: "reads best",
	}, got[0], "keys and types are normalized, a missing label falls back to the key, the recommendation names its question")
	require.Equal(t, QuestionSectionReview, got[1].Type)
	require.Equal(t, QuestionText, got[2].Type)
}

func TestNormalizeQuestionsDropsEveryInvalidQuestionAndSaysWhy(t *testing.T) {
	manyChoices := make([]string, maxQuestionChoices+1)
	for i := range manyChoices {
		manyChoices[i] = fmt.Sprintf("c%d", i)
	}
	manySections := make([]string, maxReviewSections+1)
	for i := range manySections {
		manySections[i] = fmt.Sprintf("s%d", i)
	}
	for name, tc := range map[string]struct {
		question AskQuestion
		rule     string
	}{
		"bad id":                  {AskQuestion{ID: "has space", Type: "text", Prompt: "Why?"}, "question id"},
		"empty id":                {AskQuestion{Type: "text", Prompt: "Why?"}, "question id"},
		"missing prompt":          {AskQuestion{ID: "q", Type: "text"}, "prompt is required"},
		"prompt too long":         {AskQuestion{ID: "q", Type: "text", Prompt: strings.Repeat("é", maxQuestionPrompt+1)}, "at most 500"},
		"help too long":           {AskQuestion{ID: "q", Type: "text", Prompt: "Why?", Help: strings.Repeat("x", maxQuestionHelp+1)}, "at most 1000"},
		"invalid utf8 prompt":     {AskQuestion{ID: "q", Type: "text", Prompt: "bad \xff"}, "UTF-8"},
		"no type":                 {AskQuestion{ID: "q", Prompt: "Why?"}, "no type"},
		"choice without choices":  {choiceQuestion("q", QuestionSingleChoice), "at least one choice"},
		"too many choices":        {choiceQuestion("q", QuestionMultiChoice, manyChoices...), "at most 12"},
		"bad choice key":          {choiceQuestion("q", QuestionSingleChoice, "ok", "not ok"), "choice key"},
		"repeated choice key":     {choiceQuestion("q", QuestionSingleChoice, "a", "a"), "is repeated"},
		"review without sections": {AskQuestion{ID: "q", Type: QuestionSectionReview, Prompt: "Plan"}, "at least one section"},
		"too many sections":       {sectionQuestion("q", manySections...), "at most 40"},
		"repeated section key":    {sectionQuestion("q", "a", "a"), "is repeated"},
		"section without title":   {AskQuestion{ID: "q", Type: QuestionSectionReview, Prompt: "Plan", Sections: []AskQuestionSection{{Key: "a"}}}, "title of section a is required"},
	} {
		t.Run(name, func(t *testing.T) {
			got, warnings := normalizeQuestions([]AskQuestion{tc.question}, nil)
			require.Empty(t, got)
			require.Len(t, warnings, 1)
			require.Equal(t, WarnQuestionInvalid, warnings[0].Code)
			require.Equal(t, WarningDropped, warnings[0].Severity)
			require.Equal(t, "questions[0]", warnings[0].Field)
			require.Contains(t, warnings[0].Message, tc.rule)
		})
	}
}

func TestNormalizeQuestionsLimitsTheFormAndRefusesRepeatedIDs(t *testing.T) {
	var in []AskQuestion
	for i := 0; i < maxAskQuestions+2; i++ {
		in = append(in, AskQuestion{ID: fmt.Sprintf("q%d", i), Type: "text", Prompt: "Why?"})
	}
	in = append(in[:1], append([]AskQuestion{{ID: "q0", Type: "text", Prompt: "again"}}, in[1:]...)...)
	got, warnings := normalizeQuestions(in, nil)
	require.Len(t, got, maxAskQuestions)
	require.Equal(t, "q0", got[0].ID)
	require.Equal(t, "q1", got[1].ID, "the repeat of q0 is dropped, later questions keep their order")
	require.Len(t, warnings, 3, "one repeated id and two questions over the limit")
	require.Contains(t, warnings[0].Message, `"q0" is already used`)
	require.Contains(t, warnings[1].Message, "at most 12 questions")
}

func TestNormalizeQuestionsKeepsAnUnknownTypeAsFreeTextWithAdvice(t *testing.T) {
	got, warnings := normalizeQuestions([]AskQuestion{
		{ID: "when", Type: "date_picker", Prompt: "When?", Choices: []AskQuestionChoice{{Key: "x"}}, Recommended: &AskQuestionResponse{Text: "tomorrow"}},
	}, nil)
	require.Len(t, got, 1)
	require.Equal(t, "date_picker", got[0].Type)
	require.Equal(t, &AskQuestionResponse{QuestionID: "when", Text: "tomorrow"}, got[0].Recommended)
	require.Equal(t, []string{"question_type_unknown@questions[0]"}, warningCodes(warnings))
	require.Equal(t, WarningAdvice, warnings[0].Severity)
	require.Contains(t, warnings[0].Message, "free-text")

	// A question of an unknown type answers with text, like a text question.
	responses, err := normalizeResponses(got, "approve", []AskQuestionResponse{{QuestionID: "when", Text: " next week "}})
	require.NoError(t, err)
	require.Equal(t, []AskQuestionResponse{{QuestionID: "when", Text: "next week"}}, responses)
	_, err = normalizeResponses(got, "approve", []AskQuestionResponse{{QuestionID: "when", Selected: []string{"x"}}})
	require.ErrorIs(t, err, ErrInvalidArgument)
}

func TestNormalizeQuestionsDropsAnInvalidRecommendationButKeepsTheQuestion(t *testing.T) {
	question := choiceQuestion("layout", QuestionSingleChoice, "cards", "table")
	question.Recommended = &AskQuestionResponse{Selected: []string{"grid"}}
	got, warnings := normalizeQuestions([]AskQuestion{question}, nil)
	require.Len(t, got, 1)
	require.Nil(t, got[0].Recommended)
	require.Equal(t, []string{"question_invalid@questions[0].recommended"}, warningCodes(warnings))
	require.Equal(t, WarningDropped, warnings[0].Severity)
	require.Contains(t, warnings[0].Message, `"grid"`)
}

func TestNormalizeQuestionsAdvisesOnAttachmentNamesThatAreNotAttached(t *testing.T) {
	question := choiceQuestion("layout", QuestionVariant, "cards", "table")
	question.Choices[0].Attachment = "cards.png"
	question.Choices[1].Attachment = "table.png"
	review := sectionQuestion("plan", "storage")
	review.Sections[0].Attachment = "plan.md"
	got, warnings := normalizeQuestions([]AskQuestion{question, review}, map[string]bool{"cards.png": true})
	require.Len(t, got, 2, "an unknown attachment never drops the question")
	require.Equal(t, []string{
		"question_attachment_unknown@questions[0].choices[1]",
		"question_attachment_unknown@questions[1].sections[0]",
	}, warningCodes(warnings))
	for _, warning := range warnings {
		require.Equal(t, WarningAdvice, warning.Severity)
	}
}

func TestQuestionsRoundTripThroughTheirStoredForm(t *testing.T) {
	require.Empty(t, encodeQuestions(nil), "an ask without a form stores nothing")
	require.Nil(t, decodeQuestions(""))
	require.Nil(t, decodeQuestions("not json"), "a damaged column reads as no form instead of failing the read")
	in, _ := normalizeQuestions([]AskQuestion{
		{ID: "layout", Type: "single_choice", Prompt: "Layout?", Choices: []AskQuestionChoice{{Key: "a", Label: "A"}}, Recommended: &AskQuestionResponse{Selected: []string{"a"}}},
		sectionQuestion("plan", "storage"),
	}, nil)
	require.True(t, sameQuestions(in, decodeQuestions(encodeQuestions(in))))
	changed := decodeQuestions(encodeQuestions(in))
	changed[0].Choices[0].Label = "Renamed"
	require.False(t, sameQuestions(in, changed))
}

func TestNormalizeResponsesValidatesShapeAgainstTheQuestions(t *testing.T) {
	questions := []AskQuestion{
		choiceQuestion("layout", QuestionSingleChoice, "cards", "table"),
		choiceQuestion("extras", QuestionMultiChoice, "dark", "export", "share"),
		sectionQuestion("plan", "storage", "billing"),
		{ID: "why", Type: QuestionText, Prompt: "Why?"},
	}
	good, err := normalizeResponses(questions, "approve", []AskQuestionResponse{
		{QuestionID: "why", Text: "  because  "},
		{QuestionID: "extras", Selected: []string{"share", "dark", "share"}, Text: "both please"},
		{QuestionID: "plan", Sections: []AskSectionVerdict{{SectionKey: "storage", Verdict: " Approve "}, {SectionKey: "billing", Verdict: "change", Note: " cap it "}}},
		{QuestionID: "layout", Selected: []string{" cards "}},
	})
	require.NoError(t, err)
	require.Equal(t, []AskQuestionResponse{
		{QuestionID: "layout", Selected: []string{"cards"}},
		{QuestionID: "extras", Selected: []string{"share", "dark"}, Text: "both please"},
		{QuestionID: "plan", Sections: []AskSectionVerdict{{SectionKey: "storage", Verdict: "approve"}, {SectionKey: "billing", Verdict: "change", Note: "cap it"}}},
		{QuestionID: "why", Text: "because"},
	}, good, "responses come back in question order, trimmed, with repeated selections collapsed")

	empty, err := normalizeResponses(questions, "approve", []AskQuestionResponse{{QuestionID: "why"}})
	require.NoError(t, err)
	require.Empty(t, empty, "a response with no content is the same as none")

	for name, tc := range map[string]struct {
		responses []AskQuestionResponse
		contains  string
	}{
		"unknown question":       {[]AskQuestionResponse{{QuestionID: "nope", Text: "x"}}, `question "nope"`},
		"repeated question":      {[]AskQuestionResponse{{QuestionID: "why", Text: "a"}, {QuestionID: "why", Text: "b"}}, "more than one response"},
		"unknown choice":         {[]AskQuestionResponse{{QuestionID: "layout", Selected: []string{"grid"}}}, `"grid"`},
		"two choices for single": {[]AskQuestionResponse{{QuestionID: "layout", Selected: []string{"cards", "table"}}}, "takes one"},
		"sections on a choice":   {[]AskQuestionResponse{{QuestionID: "layout", Sections: []AskSectionVerdict{{SectionKey: "a", Verdict: "approve"}}}}, "takes selected choices"},
		"choices on a review":    {[]AskQuestionResponse{{QuestionID: "plan", Selected: []string{"storage"}}}, "takes section verdicts"},
		"choices on text":        {[]AskQuestionResponse{{QuestionID: "why", Selected: []string{"x"}}}, "takes text"},
		"unknown section":        {[]AskQuestionResponse{{QuestionID: "plan", Sections: []AskSectionVerdict{{SectionKey: "legal", Verdict: "approve"}}}}, `section "legal"`},
		"repeated section":       {[]AskQuestionResponse{{QuestionID: "plan", Sections: []AskSectionVerdict{{SectionKey: "storage", Verdict: "approve"}, {SectionKey: "storage", Verdict: "drop"}}}}, "twice"},
		"unknown verdict":        {[]AskQuestionResponse{{QuestionID: "plan", Sections: []AskSectionVerdict{{SectionKey: "storage", Verdict: "maybe"}}}}, "approve, change, or drop"},
		"text too long":          {[]AskQuestionResponse{{QuestionID: "why", Text: strings.Repeat("x", maxResponseTextRunes+1)}}, "at most 4000"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := normalizeResponses(questions, "approve", tc.responses)
			require.Error(t, err)
			require.True(t, errors.Is(err, ErrInvalidArgument), "%v", err)
			require.Contains(t, err.Error(), tc.contains)
		})
	}
}

func TestNormalizeResponsesEnforcesRequiredQuestionsOnlyForTheRequestersOptions(t *testing.T) {
	layout := choiceQuestion("layout", QuestionSingleChoice, "cards", "table")
	layout.Required = true
	plan := sectionQuestion("plan", "storage", "billing")
	plan.Required = true
	why := AskQuestion{ID: "why", Type: QuestionText, Prompt: "Why?", Required: true}
	questions := []AskQuestion{layout, plan, why}
	complete := []AskQuestionResponse{
		{QuestionID: "layout", Selected: []string{"cards"}},
		{QuestionID: "plan", Sections: []AskSectionVerdict{{SectionKey: "storage", Verdict: "approve"}, {SectionKey: "billing", Verdict: "drop"}}},
		{QuestionID: "why", Text: "because"},
	}
	_, err := normalizeResponses(questions, "approve", complete)
	require.NoError(t, err)

	layoutOnly := complete[:1]
	for name, incomplete := range map[string][]AskQuestionResponse{
		"missing choice":       complete[1:],
		"note instead of pick": {{QuestionID: "layout", Text: "dunno"}, complete[1], complete[2]},
		"blank text":           {complete[0], complete[1], {QuestionID: "why", Text: "  "}},
		"only one answered":    layoutOnly,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := normalizeResponses(questions, "approve", incomplete)
			require.ErrorIs(t, err, ErrInvalidArgument)
			require.Contains(t, err.Error(), "required")
			// "Other" is the operator's way out of a form that cannot express
			// their answer: it never binds.
			_, err = normalizeResponses(questions, "other", incomplete)
			require.NoError(t, err)
		})
	}
	none, err := normalizeResponses(questions, "other", nil)
	require.NoError(t, err)
	require.Empty(t, none)

	// Even with "other", a response that does not match its question is wrong.
	_, err = normalizeResponses(questions, "other", []AskQuestionResponse{{QuestionID: "layout", Selected: []string{"grid"}}})
	require.ErrorIs(t, err, ErrInvalidArgument)
}

func TestResponsePayloadIsPlainJSONWithSnakeCaseKeys(t *testing.T) {
	payload := ResponsePayload([]AskQuestionResponse{
		{QuestionID: "layout", Selected: []string{"cards"}},
		{QuestionID: "plan", Sections: []AskSectionVerdict{{SectionKey: "storage", Verdict: "change", Note: "less"}}},
	})
	require.Equal(t, []any{
		map[string]any{"question_id": "layout", "selected": []any{"cards"}, "text": "", "sections": []any{}},
		map[string]any{"question_id": "plan", "selected": []any{}, "text": "", "sections": []any{map[string]any{"section_key": "storage", "verdict": "change", "note": "less"}}},
	}, payload)
	require.Equal(t, []any{}, ResponsePayload(nil), "no responses is an empty list, so consumers can range over it")
}

func TestNormalizeResponsesGivesEverySectionAVerdictFromTheUnansweredVerdict(t *testing.T) {
	plan := sectionQuestion("plan", "storage", "retention", "billing")
	plan.UnansweredVerdict = VerdictDrop
	plan.Required = true
	given := []AskQuestionResponse{{QuestionID: "plan", Sections: []AskSectionVerdict{
		{SectionKey: "billing", Verdict: "approve", Note: " only if capped "},
	}, Text: " overall fine "}}
	got, err := normalizeResponses([]AskQuestion{plan}, "approve", given)
	require.NoError(t, err, "one verdict answers a required review; the rest take the default")
	require.Equal(t, []AskQuestionResponse{{QuestionID: "plan", Text: "overall fine", Sections: []AskSectionVerdict{
		{SectionKey: "storage", Verdict: "drop"},
		{SectionKey: "retention", Verdict: "drop"},
		{SectionKey: "billing", Verdict: "approve", Note: "only if capped"},
	}}}, got, "a requester reads a complete review in section order")

	// A review with no verdict at all is no answer, however many notes it has.
	_, err = normalizeResponses([]AskQuestion{plan}, "approve", []AskQuestionResponse{{QuestionID: "plan"}})
	require.ErrorIs(t, err, ErrInvalidArgument)
}

func TestNormalizeQuestionsStoresTheUnansweredVerdictOfAReview(t *testing.T) {
	for given, want := range map[string]string{"": VerdictApprove, " Drop ": VerdictDrop, "change": VerdictChange} {
		review := sectionQuestion("plan", "storage")
		review.UnansweredVerdict = given
		got, warnings := normalizeQuestions([]AskQuestion{review}, nil)
		require.Empty(t, warnings)
		require.Equal(t, want, got[0].UnansweredVerdict, "%q", given)
	}
	bad := sectionQuestion("plan", "storage")
	bad.UnansweredVerdict = "maybe"
	got, warnings := normalizeQuestions([]AskQuestion{bad}, nil)
	require.Empty(t, got)
	require.Contains(t, warnings[0].Message, "unanswered_verdict")

	// Only a review has one.
	text := AskQuestion{ID: "why", Type: QuestionText, Prompt: "Why?", UnansweredVerdict: "drop"}
	got, _ = normalizeQuestions([]AskQuestion{text}, nil)
	require.Empty(t, got[0].UnansweredVerdict)
}

func TestNormalizeAttachmentNotesValidatesAgainstTheAsksAttachments(t *testing.T) {
	attachments := []AskAttachment{
		{Name: "character.glb", Kind: AttachmentFile},
		{Name: "shots", Kind: AttachmentFolder, Entries: []AskAttachmentEntry{{Path: "day-1/intro.png"}, {Path: "day-2/outro.png"}}},
	}
	got, err := normalizeAttachmentNotes([]AskAttachmentNote{
		{Attachment: " character.glb ", Note: " Hair is too glossy. "},
		{Attachment: "shots", Path: "day-1/intro.png", Note: "Crop the top."},
		{Attachment: "shots", Note: "Too many."},
		{Attachment: "shots", Path: "day-2/outro.png", Note: "  "},
	}, attachments)
	require.NoError(t, err)
	require.Equal(t, []AskAttachmentNote{
		{Attachment: "character.glb", Note: "Hair is too glossy."},
		{Attachment: "shots", Path: "day-1/intro.png", Note: "Crop the top."},
		{Attachment: "shots", Note: "Too many."},
	}, got, "an empty note is dropped")

	for name, notes := range map[string][]AskAttachmentNote{
		"unknown attachment":  {{Attachment: "nope", Note: "x"}},
		"path on a file":      {{Attachment: "character.glb", Path: "a.png", Note: "x"}},
		"unknown entry":       {{Attachment: "shots", Path: "day-3/x.png", Note: "x"}},
		"repeated target":     {{Attachment: "character.glb", Note: "a"}, {Attachment: "character.glb", Note: "b"}},
		"note too long":       {{Attachment: "character.glb", Note: strings.Repeat("x", maxAttachmentNoteRune+1)}},
		"too many notes":      make([]AskAttachmentNote, maxAttachmentNotes+1),
		"empty attachment id": {{Note: "x"}},
	} {
		_, err := normalizeAttachmentNotes(notes, attachments)
		require.ErrorIs(t, err, ErrInvalidArgument, name)
	}
	_, err = normalizeAttachmentNotes([]AskAttachmentNote{{Attachment: "character.glb", Note: "x"}}, nil)
	require.ErrorIs(t, err, ErrInvalidArgument, "an ask without attachments takes no notes")

	require.Equal(t, []any{map[string]any{"attachment": "shots", "path": "a/b.png", "note": "n"}}, AttachmentNotePayload([]AskAttachmentNote{{Attachment: "shots", Path: "a/b.png", Note: "n"}}))
	require.Equal(t, []any{}, AttachmentNotePayload(nil))
}
