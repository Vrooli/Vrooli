package hub

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// The answer form: typed questions under an ask's main options, and the
// operator's responses to them. The rules are documented in
// docs/concepts/DECISION_PACKETS.md ("Answer form").

// Question types the Hub renders. Any other type is kept and shown as a
// free-text question, so a newer sender never breaks the Hub.
const (
	QuestionSingleChoice  = "single_choice"
	QuestionMultiChoice   = "multi_choice"
	QuestionVariant       = "variant"
	QuestionSectionReview = "section_review"
	QuestionText          = "text"
)

// Section review verdicts.
const (
	VerdictApprove = "approve"
	VerdictChange  = "change"
	VerdictDrop    = "drop"
)

// Warning codes of the answer form. They extend the packet check.
const (
	WarnQuestionInvalid           = "question_invalid"
	WarnQuestionTypeUnknown       = "question_type_unknown"
	WarnQuestionAttachmentUnknown = "question_attachment_unknown"
	WarnQuestionRemoveUnknown     = "question_remove_unknown"
)

const (
	maxAskQuestions      = 12
	maxQuestionChoices   = 12
	maxReviewSections    = 40
	maxQuestionPrompt    = 500
	maxQuestionHelp      = 1_000
	maxQuestionReason    = 300
	maxChoiceDetail      = 300
	maxSectionTitle      = 200
	maxSectionSummary    = 300
	maxSectionAnchor     = 200
	maxResponseTextRunes = 4_000
)

// AskQuestion is one field of the answer form.
type AskQuestion struct {
	ID                   string               `json:"id"`
	Type                 string               `json:"type"`
	Prompt               string               `json:"prompt"`
	Help                 string               `json:"help,omitempty"`
	Required             bool                 `json:"required,omitempty"`
	Choices              []AskQuestionChoice  `json:"choices,omitempty"`
	Sections             []AskQuestionSection `json:"sections,omitempty"`
	Recommended          *AskQuestionResponse `json:"recommended,omitempty"`
	RecommendationReason string               `json:"recommendation_reason,omitempty"`
	// UnansweredVerdict is the verdict a section_review gives a section the
	// operator leaves alone: approve (the default), change, or drop.
	UnansweredVerdict string `json:"unanswered_verdict,omitempty"`
	// SectionsFrom names the attachment a section_review follows (its name, or
	// "name/path" for a file in a folder). The Hub derives Sections from that
	// file's Markdown headings; see followSections.
	SectionsFrom string `json:"sections_from,omitempty"`
}

// AskQuestionChoice is one selectable answer of a choice question.
type AskQuestionChoice struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	Detail     string `json:"detail,omitempty"`
	Attachment string `json:"attachment,omitempty"`
}

// AskQuestionSection is one section of a section review.
type AskQuestionSection struct {
	Key        string `json:"key"`
	Title      string `json:"title"`
	Summary    string `json:"summary,omitempty"`
	Attachment string `json:"attachment,omitempty"`
	Anchor     string `json:"anchor,omitempty"`
}

// AskQuestionResponse is the operator's answer to one question.
type AskQuestionResponse struct {
	QuestionID string              `json:"question_id"`
	Selected   []string            `json:"selected,omitempty"`
	Text       string              `json:"text,omitempty"`
	Sections   []AskSectionVerdict `json:"sections,omitempty"`
}

// AskSectionVerdict is the operator's verdict on one section of a review.
type AskSectionVerdict struct {
	SectionKey string `json:"section_key"`
	Verdict    string `json:"verdict"`
	Note       string `json:"note,omitempty"`
}

func isChoiceType(questionType string) bool {
	switch questionType {
	case QuestionSingleChoice, QuestionMultiChoice, QuestionVariant:
		return true
	}
	return false
}

func isKnownQuestionType(questionType string) bool {
	return isChoiceType(questionType) || questionType == QuestionSectionReview || questionType == QuestionText
}

func (q AskQuestion) choice(key string) (AskQuestionChoice, bool) {
	for _, choice := range q.Choices {
		if choice.Key == key {
			return choice, true
		}
	}
	return AskQuestionChoice{}, false
}

func (q AskQuestion) section(key string) (AskQuestionSection, bool) {
	for _, section := range q.Sections {
		if section.Key == key {
			return section, true
		}
	}
	return AskQuestionSection{}, false
}

// normalizeQuestions validates the form a requester sent. A question that
// breaks a rule is dropped and reported; one of an unknown type is kept as a
// free-text question. attached holds the names of the ask's accepted
// attachments, which choices and sections may point at. Neither outcome fails
// the ask.
func normalizeQuestions(in []AskQuestion, attached map[string]bool) ([]AskQuestion, []PacketWarning) {
	var (
		out      []AskQuestion
		warnings []PacketWarning
		seen     = make(map[string]bool, len(in))
	)
	for index, raw := range in {
		field := fmt.Sprintf("questions[%d]", index)
		if len(out) >= maxAskQuestions {
			warnings = append(warnings, droppedQuestion(field, fmt.Sprintf("an ask shows at most %d questions; this one was left out", maxAskQuestions)))
			continue
		}
		question, questionWarnings, err := normalizeQuestion(raw, field, attached)
		if err != nil {
			warnings = append(warnings, droppedQuestion(field, err.Error()))
			continue
		}
		if seen[question.ID] {
			warnings = append(warnings, droppedQuestion(field, fmt.Sprintf("question id %q is already used by an earlier question; this one was left out", question.ID)))
			continue
		}
		seen[question.ID] = true
		out = append(out, question)
		warnings = append(warnings, questionWarnings...)
	}
	return out, warnings
}

func droppedQuestion(field, message string) PacketWarning {
	return PacketWarning{Code: WarnQuestionInvalid, Field: field, Severity: WarningDropped, Message: message}
}

// normalizeQuestion returns the question as the Hub stores it. The error text
// is a complete sentence about the rule it breaks.
func normalizeQuestion(raw AskQuestion, field string, attached map[string]bool) (AskQuestion, []PacketWarning, error) {
	var warnings []PacketWarning
	question := AskQuestion{
		ID:                   strings.TrimSpace(raw.ID),
		Type:                 strings.ToLower(strings.TrimSpace(raw.Type)),
		Prompt:               strings.TrimSpace(raw.Prompt),
		Help:                 strings.TrimSpace(raw.Help),
		Required:             raw.Required,
		RecommendationReason: strings.TrimSpace(raw.RecommendationReason),
	}
	if !askOptionKeyRule.MatchString(question.ID) {
		return AskQuestion{}, nil, fmt.Errorf("question id %q must be 1-64 letters, digits, '.', '_', ':' or '-'; the question was left out", question.ID)
	}
	if err := checkText("prompt", question.Prompt, 1, maxQuestionPrompt); err != nil {
		return AskQuestion{}, nil, fmt.Errorf("question %q: %w; the question was left out", question.ID, err)
	}
	if err := checkText("help", question.Help, 0, maxQuestionHelp); err != nil {
		return AskQuestion{}, nil, fmt.Errorf("question %q: %w; the question was left out", question.ID, err)
	}
	if err := checkText("recommendation_reason", question.RecommendationReason, 0, maxQuestionReason); err != nil {
		return AskQuestion{}, nil, fmt.Errorf("question %q: %w; the question was left out", question.ID, err)
	}
	if question.Type == "" {
		return AskQuestion{}, nil, fmt.Errorf("question %q has no type; the question was left out", question.ID)
	}
	if from := strings.TrimSpace(raw.SectionsFrom); from != "" {
		if question.Type != QuestionSectionReview {
			return AskQuestion{}, nil, fmt.Errorf("question %q is a %s question, and only a section_review follows a file with sections_from; the question was left out", question.ID, question.Type)
		}
		if err := checkText("sections_from", from, 1, maxSectionAnchor); err != nil {
			return AskQuestion{}, nil, fmt.Errorf("question %q: %w; the question was left out", question.ID, err)
		}
		question.SectionsFrom = from
	}
	var err error
	switch {
	case isChoiceType(question.Type):
		question.Choices, warnings, err = normalizeChoices(question.ID, raw.Choices, field, attached)
	case question.Type == QuestionSectionReview:
		question.Sections, warnings, err = normalizeSections(question.ID, raw.Sections, field, attached)
		if err == nil {
			question.UnansweredVerdict, err = normalizeUnansweredVerdict(question.ID, raw.UnansweredVerdict)
		}
	case question.Type == QuestionText:
	default:
		warnings = append(warnings, PacketWarning{
			Code: WarnQuestionTypeUnknown, Field: field, Severity: WarningAdvice,
			Message: fmt.Sprintf("question %q has type %q, which the Hub does not know; it is shown as a free-text question (known types: single_choice, multi_choice, variant, section_review, text)", question.ID, question.Type),
		})
	}
	if err != nil {
		return AskQuestion{}, nil, err
	}
	if raw.Recommended != nil {
		recommended, recommendErr := normalizeResponse(question, *raw.Recommended)
		switch {
		case recommendErr != nil:
			warnings = append(warnings, droppedQuestion(field+".recommended", fmt.Sprintf("question %q: the recommendation %s; the recommendation was left out", question.ID, recommendErr.Error())))
		case !recommended.isEmpty():
			question.Recommended = &recommended
		}
	}
	return question, warnings, nil
}

// normalizeUnansweredVerdict returns the verdict untouched sections take.
func normalizeUnansweredVerdict(questionID, raw string) (string, error) {
	switch verdict := strings.ToLower(strings.TrimSpace(raw)); verdict {
	case "":
		return VerdictApprove, nil
	case VerdictApprove, VerdictChange, VerdictDrop:
		return verdict, nil
	default:
		return "", fmt.Errorf("question %q: unanswered_verdict %q must be approve, change, or drop; the question was left out", questionID, raw)
	}
}

func normalizeChoices(questionID string, raw []AskQuestionChoice, field string, attached map[string]bool) ([]AskQuestionChoice, []PacketWarning, error) {
	if len(raw) == 0 {
		return nil, nil, fmt.Errorf("question %q needs at least one choice; the question was left out", questionID)
	}
	if len(raw) > maxQuestionChoices {
		return nil, nil, fmt.Errorf("question %q has %d choices; at most %d are allowed; the question was left out", questionID, len(raw), maxQuestionChoices)
	}
	var (
		out      = make([]AskQuestionChoice, 0, len(raw))
		warnings []PacketWarning
		seen     = make(map[string]bool, len(raw))
	)
	for index, choice := range raw {
		key := strings.TrimSpace(choice.Key)
		if !askOptionKeyRule.MatchString(key) {
			return nil, nil, fmt.Errorf("question %q: choice key %q must be 1-64 letters, digits, '.', '_', ':' or '-'; the question was left out", questionID, key)
		}
		if seen[key] {
			return nil, nil, fmt.Errorf("question %q: choice key %q is repeated; the question was left out", questionID, key)
		}
		seen[key] = true
		label := strings.TrimSpace(choice.Label)
		if label == "" {
			label = key
		}
		detail := strings.TrimSpace(choice.Detail)
		if err := checkText("label of choice "+key, label, 1, maxAskOptionLabel); err != nil {
			return nil, nil, fmt.Errorf("question %q: %w; the question was left out", questionID, err)
		}
		if err := checkText("detail of choice "+key, detail, 0, maxChoiceDetail); err != nil {
			return nil, nil, fmt.Errorf("question %q: %w; the question was left out", questionID, err)
		}
		attachment := strings.TrimSpace(choice.Attachment)
		if attachment != "" && !isAttached(attached, attachment) {
			warnings = append(warnings, unknownQuestionAttachment(fmt.Sprintf("%s.choices[%d]", field, index), questionID, "choice "+key, attachment))
		}
		out = append(out, AskQuestionChoice{Key: key, Label: label, Detail: detail, Attachment: attachment})
	}
	return out, warnings, nil
}

func normalizeSections(questionID string, raw []AskQuestionSection, field string, attached map[string]bool) ([]AskQuestionSection, []PacketWarning, error) {
	if len(raw) == 0 {
		return nil, nil, fmt.Errorf("question %q needs at least one section; the question was left out", questionID)
	}
	if len(raw) > maxReviewSections {
		return nil, nil, fmt.Errorf("question %q has %d sections; at most %d are allowed; the question was left out", questionID, len(raw), maxReviewSections)
	}
	var (
		out      = make([]AskQuestionSection, 0, len(raw))
		warnings []PacketWarning
		seen     = make(map[string]bool, len(raw))
	)
	for index, section := range raw {
		key := strings.TrimSpace(section.Key)
		if !askOptionKeyRule.MatchString(key) {
			return nil, nil, fmt.Errorf("question %q: section key %q must be 1-64 letters, digits, '.', '_', ':' or '-'; the question was left out", questionID, key)
		}
		if seen[key] {
			return nil, nil, fmt.Errorf("question %q: section key %q is repeated; the question was left out", questionID, key)
		}
		seen[key] = true
		title := strings.TrimSpace(section.Title)
		summary := strings.TrimSpace(section.Summary)
		anchor := strings.TrimSpace(section.Anchor)
		for _, check := range []struct {
			name, value string
			min, max    int
		}{
			{"title of section " + key, title, 1, maxSectionTitle},
			{"summary of section " + key, summary, 0, maxSectionSummary},
			{"anchor of section " + key, anchor, 0, maxSectionAnchor},
		} {
			if err := checkText(check.name, check.value, check.min, check.max); err != nil {
				return nil, nil, fmt.Errorf("question %q: %w; the question was left out", questionID, err)
			}
		}
		attachment := strings.TrimSpace(section.Attachment)
		if attachment != "" && !isAttached(attached, attachment) {
			warnings = append(warnings, unknownQuestionAttachment(fmt.Sprintf("%s.sections[%d]", field, index), questionID, "section "+key, attachment))
		}
		out = append(out, AskQuestionSection{Key: key, Title: title, Summary: summary, Attachment: attachment, Anchor: anchor})
	}
	return out, warnings, nil
}

// isAttached reports whether a reference names an attachment of the ask: its
// name, or "name/path" for a file inside a folder attachment (names hold no "/").
func isAttached(attached map[string]bool, reference string) bool {
	name, _, _ := strings.Cut(reference, "/")
	return attached[name]
}

func unknownQuestionAttachment(field, questionID, where, attachment string) PacketWarning {
	return PacketWarning{
		Code: WarnQuestionAttachmentUnknown, Field: field, Severity: WarningAdvice,
		Message: fmt.Sprintf("question %q, %s points at attachment %q, which is not attached to this ask; the operator will not see it (attach it, or fix the name)", questionID, where, attachment),
	}
}

// checkText reports a value that is not valid UTF-8 or whose length in code
// points is outside [min, max]. The error names the field and the rule.
func checkText(name, value string, min, max int) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s must be valid UTF-8", name)
	}
	length := utf8.RuneCountInString(value)
	if length < min {
		return fmt.Errorf("%s is required", name)
	}
	if length > max {
		return fmt.Errorf("%s is %d characters; at most %d are allowed", name, length, max)
	}
	return nil
}

// encodeQuestions renders the stored form of an ask's questions. It is the
// empty string for none, so asks without a form keep their rows unchanged.
func encodeQuestions(questions []AskQuestion) string {
	if len(questions) == 0 {
		return ""
	}
	encoded, _ := json.Marshal(questions)
	return string(encoded)
}

func decodeQuestions(encoded string) []AskQuestion {
	if strings.TrimSpace(encoded) == "" {
		return nil
	}
	var questions []AskQuestion
	if json.Unmarshal([]byte(encoded), &questions) != nil {
		return nil
	}
	return questions
}

// sameQuestions reports whether two normalized forms are identical.
func sameQuestions(a, b []AskQuestion) bool {
	return encodeQuestions(a) == encodeQuestions(b)
}

func (r AskQuestionResponse) isEmpty() bool {
	return len(r.Selected) == 0 && len(r.Sections) == 0 && r.Text == ""
}

// normalizeResponse validates one response against its question and returns
// it with trimmed values. The error is a clause that completes "response to
// question X <error>" style messages; callers prefix the question.
func normalizeResponse(question AskQuestion, raw AskQuestionResponse) (AskQuestionResponse, error) {
	response := AskQuestionResponse{QuestionID: question.ID, Text: strings.TrimSpace(raw.Text)}
	if id := strings.TrimSpace(raw.QuestionID); id != "" && id != question.ID {
		return AskQuestionResponse{}, fmt.Errorf("names question %q, not %q", id, question.ID)
	}
	if err := checkText("text", response.Text, 0, maxResponseTextRunes); err != nil {
		return AskQuestionResponse{}, err
	}
	switch {
	case isChoiceType(question.Type):
		if len(raw.Sections) > 0 {
			return AskQuestionResponse{}, fmt.Errorf("has section verdicts, but a %s question takes selected choices", question.Type)
		}
		selected, err := normalizeSelection(question, raw.Selected)
		if err != nil {
			return AskQuestionResponse{}, err
		}
		response.Selected = selected
	case question.Type == QuestionSectionReview:
		if len(raw.Selected) > 0 {
			return AskQuestionResponse{}, fmt.Errorf("has selected choices, but a section_review question takes section verdicts")
		}
		verdicts, err := normalizeVerdicts(question, raw.Sections)
		if err != nil {
			return AskQuestionResponse{}, err
		}
		response.Sections = verdicts
	default:
		// text, and any type the Hub does not know, answer with text.
		if len(raw.Selected) > 0 || len(raw.Sections) > 0 {
			return AskQuestionResponse{}, fmt.Errorf("has choices or section verdicts, but a %s question takes text", question.Type)
		}
	}
	return response, nil
}

func normalizeSelection(question AskQuestion, raw []string) ([]string, error) {
	var out []string
	seen := make(map[string]bool, len(raw))
	for _, value := range raw {
		key := strings.TrimSpace(value)
		if _, ok := question.choice(key); !ok {
			return nil, fmt.Errorf("selects %q, which is not one of its choices", key)
		}
		if !seen[key] {
			seen[key] = true
			out = append(out, key)
		}
	}
	if question.Type != QuestionMultiChoice && len(out) > 1 {
		return nil, fmt.Errorf("selects %d choices, but a %s question takes one", len(out), question.Type)
	}
	return out, nil
}

// normalizeVerdicts validates the verdicts the operator gave and returns one
// verdict for every section, in the review's order: a section the operator left
// alone takes the question's unanswered verdict. A requester therefore reads a
// complete review and never has to apply the default itself. Without any given
// verdict there is no review to return.
func normalizeVerdicts(question AskQuestion, raw []AskSectionVerdict) ([]AskSectionVerdict, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	given := make(map[string]AskSectionVerdict, len(raw))
	seen := make(map[string]bool, len(raw))
	for _, verdict := range raw {
		key := strings.TrimSpace(verdict.SectionKey)
		if _, ok := question.section(key); !ok {
			return nil, fmt.Errorf("reviews section %q, which is not one of its sections", key)
		}
		if seen[key] {
			return nil, fmt.Errorf("reviews section %q twice", key)
		}
		seen[key] = true
		decision := strings.ToLower(strings.TrimSpace(verdict.Verdict))
		switch decision {
		case VerdictApprove, VerdictChange, VerdictDrop:
		default:
			return nil, fmt.Errorf("gives section %q the verdict %q; use approve, change, or drop", key, verdict.Verdict)
		}
		note := strings.TrimSpace(verdict.Note)
		if err := checkText("note of section "+key, note, 0, maxResponseTextRunes); err != nil {
			return nil, err
		}
		given[key] = AskSectionVerdict{SectionKey: key, Verdict: decision, Note: note}
	}
	out := make([]AskSectionVerdict, 0, len(question.Sections))
	for _, section := range question.Sections {
		verdict, ok := given[section.Key]
		if !ok {
			verdict = AskSectionVerdict{SectionKey: section.Key, Verdict: question.unansweredVerdict()}
		}
		out = append(out, verdict)
	}
	return out, nil
}

// unansweredVerdict is the verdict an untouched section takes, approve for a
// question stored before the field existed.
func (q AskQuestion) unansweredVerdict() string {
	if q.UnansweredVerdict == "" {
		return VerdictApprove
	}
	return q.UnansweredVerdict
}

// answersQuestion reports whether a normalized response counts as answering
// its question for the required-question rule: a choice was made, a section was
// given a verdict (the rest take the unanswered verdict), or text was written.
// A note alone does not answer a choice question.
func answersQuestion(question AskQuestion, response AskQuestionResponse) bool {
	switch {
	case isChoiceType(question.Type):
		return len(response.Selected) > 0
	case question.Type == QuestionSectionReview:
		return len(response.Sections) > 0
	default:
		return response.Text != ""
	}
}

// normalizeResponses validates the operator's responses against the ask's
// questions and returns them in question order, without empty ones. A
// required question must be answered unless the operator answered the whole
// decision with "other": an alternative the requester's form cannot express
// must never be blocked by that form (docs/concepts/DECISION_PACKETS.md,
// "Robustness rules").
func normalizeResponses(questions []AskQuestion, answer string, raw []AskQuestionResponse) ([]AskQuestionResponse, error) {
	byQuestion := make(map[string]AskQuestion, len(questions))
	for _, question := range questions {
		byQuestion[question.ID] = question
	}
	given := make(map[string]AskQuestionResponse, len(raw))
	for _, response := range raw {
		id := strings.TrimSpace(response.QuestionID)
		question, known := byQuestion[id]
		if !known {
			return nil, fmt.Errorf("%w: response names question %q, which this ask does not have", ErrInvalidArgument, id)
		}
		if _, repeated := given[id]; repeated {
			return nil, fmt.Errorf("%w: question %q has more than one response", ErrInvalidArgument, id)
		}
		normalized, err := normalizeResponse(question, response)
		if err != nil {
			return nil, fmt.Errorf("%w: response to question %q %s", ErrInvalidArgument, id, err.Error())
		}
		given[id] = normalized
	}
	var out []AskQuestionResponse
	for _, question := range questions {
		response, ok := given[question.ID]
		if question.Required && answer != "other" && !(ok && answersQuestion(question, response)) {
			return nil, fmt.Errorf("%w: question %q is required: %s", ErrInvalidArgument, question.ID, requiredHint(question))
		}
		if ok && !response.isEmpty() {
			out = append(out, response)
		}
	}
	return out, nil
}

func requiredHint(question AskQuestion) string {
	switch {
	case isChoiceType(question.Type):
		return "select a choice, or answer the decision with Other"
	case question.Type == QuestionSectionReview:
		return "give at least one section a verdict, or answer the decision with Other"
	default:
		return "write an answer, or answer the decision with Other"
	}
}

func encodeResponses(responses []AskQuestionResponse) string {
	if len(responses) == 0 {
		return ""
	}
	encoded, _ := json.Marshal(responses)
	return string(encoded)
}

func decodeResponses(encoded string) []AskQuestionResponse {
	if strings.TrimSpace(encoded) == "" {
		return nil
	}
	var responses []AskQuestionResponse
	if json.Unmarshal([]byte(encoded), &responses) != nil {
		return nil
	}
	return responses
}

// ResponsePayload renders responses as plain JSON-able values with snake_case
// keys, for event payloads that must encode through the Vrooli Events
// serializer (which accepts []any and map[string]any only).
func ResponsePayload(responses []AskQuestionResponse) []any {
	out := make([]any, 0, len(responses))
	for _, response := range responses {
		selected := make([]any, 0, len(response.Selected))
		for _, key := range response.Selected {
			selected = append(selected, key)
		}
		sections := make([]any, 0, len(response.Sections))
		for _, verdict := range response.Sections {
			sections = append(sections, map[string]any{"section_key": verdict.SectionKey, "verdict": verdict.Verdict, "note": verdict.Note})
		}
		out = append(out, map[string]any{"question_id": response.QuestionID, "selected": selected, "text": response.Text, "sections": sections})
	}
	return out
}
