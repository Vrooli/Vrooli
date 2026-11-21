package conversations

// The --form authoring shape: a YAML or JSON file that declares the typed
// questions of an answer form. See DECISION_PACKETS.md for what the Hub does
// with them; this file only turns the friendly shape into AskQuestion messages
// and rejects authoring mistakes with errors that name the question.
//
//	questions:
//	  - id: layout
//	    type: single_choice          # single_choice | multi_choice | variant | section_review | text
//	    prompt: Which layout do you prefer?
//	    help: Optional plain-text hint under the prompt.
//	    required: true
//	    choices:
//	      - {key: cards, label: Card stack, detail: One line of detail, attachment: cards.png}
//	      - {key: table, label: Table}              # label defaults to the key
//	    recommended: cards                          # key | [keys] | free text | full response mapping
//	    recommendation_reason: Reads best on a phone.
//	  - id: plan
//	    type: section_review
//	    prompt: Review each section of the plan.
//	    sections_from: plan.md                      # attachment name or path; the Hub follows its ## and ### headings
//	    unanswered_verdict: approve                 # what a section the operator leaves alone means: approve (default) | change | drop
//	  - id: notes
//	    type: text
//	    prompt: Anything else?
//
// The file may also be a bare list of questions. Keys are the proto's
// snake_case names. Limits (12 questions, 12 choices, 40 sections, text
// lengths) are enforced by the Hub, which reports what it dropped.

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

// maxReviewSections is the Hub's limit on the sections of one review.
const maxReviewSections = 40

// Question types the CLI treats specially; anything else is passed through and
// the Hub shows it as free text.
const (
	typeSingleChoice  = "single_choice"
	typeMultiChoice   = "multi_choice"
	typeVariant       = "variant"
	typeSectionReview = "section_review"
	typeText          = "text"
)

type formFile struct {
	Questions []formQuestion `yaml:"questions"`
}

type formQuestion struct {
	ID                   string         `yaml:"id"`
	Type                 string         `yaml:"type"`
	Prompt               string         `yaml:"prompt"`
	Help                 string         `yaml:"help"`
	Required             bool           `yaml:"required"`
	Choices              []formChoice   `yaml:"choices"`
	Sections             []formSection  `yaml:"sections"`
	SectionsFrom         string         `yaml:"sections_from"`
	Recommended          *formRecommend `yaml:"recommended"`
	RecommendationReason string         `yaml:"recommendation_reason"`
	UnansweredVerdict    string         `yaml:"unanswered_verdict"`
}

type formChoice struct {
	Key        string `yaml:"key"`
	Label      string `yaml:"label"`
	Detail     string `yaml:"detail"`
	Attachment string `yaml:"attachment"`
}

type formSection struct {
	Key        string `yaml:"key"`
	Title      string `yaml:"title"`
	Summary    string `yaml:"summary"`
	Attachment string `yaml:"attachment"`
	Anchor     string `yaml:"anchor"`
}

// formRecommend keeps the raw node: its meaning depends on the question type.
type formRecommend struct{ node yaml.Node }

func (r *formRecommend) UnmarshalYAML(value *yaml.Node) error {
	r.node = *value
	return nil
}

// questionKeys are the keys a question may have.
var questionKeys = []string{
	"id", "type", "prompt", "help", "required", "choices", "sections", "sections_from",
	"recommended", "recommendation_reason", "unanswered_verdict",
}

// readForm loads a form file and builds its questions. It never fails: a form
// is one packet part, so whatever cannot be used is left out and named in the
// packet's warnings, and the ask still goes out. A file that cannot be read or
// parsed leaves out every question; a broken question leaves out only itself.
// Attachment references and sections_from resolve against the packet.
func readForm(path string, pkt *packet) []*v1.AskQuestion {
	data, err := os.ReadFile(path)
	if err != nil {
		pkt.warnf("Left out the whole --form %q: %v. The ask is sent without its form.", path, unwrapPathError(err))
		return nil
	}
	return buildQuestions(data, path, pkt)
}

func buildQuestions(data []byte, path string, pkt *packet) []*v1.AskQuestion {
	dropForm := func(reason string, args ...any) []*v1.AskQuestion {
		pkt.warnf("Left out the whole --form %q: %s. The ask is sent without its form.", path, fmt.Sprintf(reason, args...))
		return nil
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return dropForm("not valid YAML or JSON: %v", err)
	}
	if root.Kind == 0 || len(root.Content) == 0 {
		return dropForm("the file is empty")
	}
	top := root.Content[0]
	var items []*yaml.Node
	switch top.Kind {
	case yaml.SequenceNode:
		items = top.Content
	case yaml.MappingNode:
		for i := 0; i+1 < len(top.Content); i += 2 {
			switch key := top.Content[i].Value; key {
			case "questions":
				if top.Content[i+1].Kind != yaml.SequenceNode {
					return dropForm("line %d: `questions` must be a list", top.Content[i].Line)
				}
				items = top.Content[i+1].Content
			default:
				pkt.warnf("Ignored unknown key %q at line %d of --form %q (expected questions).", key, top.Content[i].Line, path)
			}
		}
	default:
		return dropForm("expected a `questions:` list or a list of questions")
	}
	if len(items) == 0 {
		return dropForm("it declares no questions")
	}
	seen := make(map[string]bool, len(items))
	out := make([]*v1.AskQuestion, 0, len(items))
	for i, item := range items {
		q, err := decodeQuestion(item)
		if err == nil {
			var built *v1.AskQuestion
			if built, err = q.build(pkt); err == nil && seen[built.GetId()] {
				err = fmt.Errorf("id %q is already used by an earlier question", built.GetId())
			} else if err == nil {
				seen[built.GetId()] = true
				out = append(out, built)
			}
		}
		if err != nil {
			pkt.warnf("Left out form question %s: %v.", q.label(i), err)
		}
	}
	return out
}

// decodeQuestion reads one question node, rejecting keys the shape does not
// define so a typo such as `choice:` is reported instead of silently losing
// the question's choices.
func decodeQuestion(node *yaml.Node) (formQuestion, error) {
	var q formQuestion
	if node.Kind != yaml.MappingNode {
		return q, fmt.Errorf("line %d: a question is a mapping with id, type and prompt", node.Line)
	}
	// The id is read first so even a rejected question can be named.
	_ = node.Decode(&struct {
		ID *string `yaml:"id"`
	}{ID: &q.ID})
	if err := rejectUnknownKeys(node, questionKeys...); err != nil {
		return q, err
	}
	if err := node.Decode(&q); err != nil {
		return q, err
	}
	return q, nil
}

func (q formQuestion) label(index int) string {
	if id := strings.TrimSpace(q.ID); id != "" {
		return fmt.Sprintf("%d (%q)", index+1, id)
	}
	return fmt.Sprintf("%d", index+1)
}

func (q *formQuestion) build(pkt *packet) (*v1.AskQuestion, error) {
	id, kind, prompt := strings.TrimSpace(q.ID), strings.TrimSpace(q.Type), strings.TrimSpace(q.Prompt)
	switch {
	case id == "":
		return nil, errors.New("needs an id")
	case prompt == "":
		return nil, errors.New("needs a prompt")
	case kind == "":
		return nil, fmt.Errorf("needs a type (%s)", strings.Join([]string{typeSingleChoice, typeMultiChoice, typeVariant, typeSectionReview, typeText}, ", "))
	}
	// note reports something left off a question that is still sent.
	note := func(format string, args ...any) {
		pkt.warnf("Form question %q: %s", id, fmt.Sprintf(format, args...))
	}
	out := &v1.AskQuestion{
		Id:                   id,
		Type:                 kind,
		Prompt:               prompt,
		Help:                 strings.TrimSpace(q.Help),
		Required:             q.Required,
		RecommendationReason: strings.TrimSpace(q.RecommendationReason),
	}
	choiceKeys := map[string]bool{}
	if isChoiceType(kind) {
		if len(q.Choices) == 0 {
			return nil, fmt.Errorf("a %s question needs choices", kind)
		}
		for j, c := range q.Choices {
			choice, err := c.build(pkt, note)
			if err != nil {
				return nil, fmt.Errorf("choice %d: %w", j+1, err)
			}
			if choiceKeys[choice.Key] {
				return nil, fmt.Errorf("choice key %q is used twice", choice.Key)
			}
			choiceKeys[choice.Key] = true
			out.Choices = append(out.Choices, choice)
		}
	} else if len(q.Choices) > 0 {
		return nil, fmt.Errorf("choices belong to single_choice, multi_choice and variant questions, not %s", kind)
	}

	sectionKeys := map[string]bool{}
	switch {
	case kind != typeSectionReview && (len(q.Sections) > 0 || q.SectionsFrom != "" || q.UnansweredVerdict != ""):
		return nil, fmt.Errorf("sections, sections_from and unanswered_verdict belong to section_review questions, not %s", kind)
	case kind == typeSectionReview:
		sections, from, err := q.buildSections(pkt, note)
		if err != nil {
			return nil, err
		}
		for _, s := range sections {
			if sectionKeys[s.Key] {
				return nil, fmt.Errorf("section key %q is used twice", s.Key)
			}
			sectionKeys[s.Key] = true
		}
		out.Sections, out.SectionsFrom = sections, from
		if from != "" {
			// The Hub reads the headings, so only it knows the section keys; it
			// checks the recommendation against them.
			sectionKeys = nil
		}
		if verdict := strings.TrimSpace(q.UnansweredVerdict); verdict != "" {
			if !sectionVerdicts[verdict] {
				return nil, fmt.Errorf("unanswered_verdict %q must be approve, change, or drop", verdict)
			}
			out.UnansweredVerdict = verdict
		}
	}

	if q.Recommended != nil {
		// A recommendation is optional: a bad one is dropped, not the question.
		if rec, err := q.Recommended.response(kind, choiceKeys, sectionKeys); err != nil {
			note("left out the recommendation: %v.", err)
		} else {
			rec.QuestionId = id
			out.Recommended = rec
		}
	}
	return out, nil
}

func isChoiceType(kind string) bool {
	return kind == typeSingleChoice || kind == typeMultiChoice || kind == typeVariant
}

func (c formChoice) build(pkt *packet, note func(string, ...any)) (*v1.AskQuestionChoice, error) {
	key := strings.TrimSpace(c.Key)
	if key == "" {
		return nil, errors.New("needs a key")
	}
	label := strings.TrimSpace(c.Label)
	if label == "" {
		label = key
	}
	choice := &v1.AskQuestionChoice{Key: key, Label: label, Detail: strings.TrimSpace(c.Detail)}
	if ref := strings.TrimSpace(c.Attachment); ref != "" {
		name, err := pkt.attachmentRef(ref)
		if err != nil {
			note("choice %q shows no attachment: %v.", key, err)
		}
		choice.Attachment = name
	}
	return choice, nil
}

// buildSections returns the sections a section_review question states, or the
// attachment it follows (sections_from), whose headings the Hub reads.
func (q *formQuestion) buildSections(pkt *packet, note func(string, ...any)) (sections []*v1.AskQuestionSection, from string, err error) {
	explicit, ref := len(q.Sections) > 0, strings.TrimSpace(q.SectionsFrom)
	switch {
	case explicit && ref != "":
		return nil, "", errors.New("give sections or sections_from, not both")
	case !explicit && ref == "":
		return nil, "", errors.New("a section_review question needs sections or sections_from")
	case ref != "":
		name, err := pkt.attachmentRef(ref)
		if err != nil {
			return nil, "", fmt.Errorf("sections_from %w", err)
		}
		return nil, name, nil
	}
	out := make([]*v1.AskQuestionSection, 0, len(q.Sections))
	for j, s := range q.Sections {
		key, title := strings.TrimSpace(s.Key), strings.TrimSpace(s.Title)
		if key == "" || title == "" {
			return nil, "", fmt.Errorf("section %d needs a key and a title", j+1)
		}
		section := &v1.AskQuestionSection{Key: key, Title: title, Summary: strings.TrimSpace(s.Summary), Anchor: strings.TrimSpace(s.Anchor)}
		if ref := strings.TrimSpace(s.Attachment); ref != "" {
			name, err := pkt.attachmentRef(ref)
			if err != nil {
				note("section %q links no attachment: %v.", key, err)
			}
			section.Attachment = name
		}
		out = append(out, section)
	}
	if len(out) > maxReviewSections {
		note("it has %d sections; the review shows the first %d.", len(out), maxReviewSections)
		out = out[:maxReviewSections]
	}
	return out, "", nil
}

// recommendedMapping is the full response shape of `recommended:`.
type recommendedMapping struct {
	QuestionID string        `yaml:"question_id"`
	Selected   stringList    `yaml:"selected"`
	Text       string        `yaml:"text"`
	Sections   []formVerdict `yaml:"sections"`
}

type formVerdict struct {
	SectionKey string `yaml:"section_key"`
	Verdict    string `yaml:"verdict"`
	Note       string `yaml:"note"`
}

// stringList accepts one string or a list of strings.
type stringList []string

func (l *stringList) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		*l = stringList{value.Value}
		return nil
	case yaml.SequenceNode:
		var items []string
		if err := value.Decode(&items); err != nil {
			return err
		}
		*l = items
		return nil
	}
	return errors.New("expected a key or a list of keys")
}

// response expands the shorthand into an AskQuestionResponse and checks it
// against the question: a key string or list selects choices, a text string
// answers a text question, a mapping is the full response.
func (r *formRecommend) response(kind string, choiceKeys, sectionKeys map[string]bool) (*v1.AskQuestionResponse, error) {
	var resp *v1.AskQuestionResponse
	switch r.node.Kind {
	case yaml.ScalarNode:
		switch {
		case isChoiceType(kind):
			resp = &v1.AskQuestionResponse{Selected: []string{r.node.Value}}
		case kind == typeSectionReview:
			return nil, errors.New("a section_review recommendation is a mapping with sections: [{section_key, verdict}]")
		default:
			resp = &v1.AskQuestionResponse{Text: r.node.Value}
		}
	case yaml.SequenceNode:
		if !isChoiceType(kind) {
			return nil, fmt.Errorf("a list of keys fits choice questions, not %s", kind)
		}
		var keys []string
		if err := r.node.Decode(&keys); err != nil {
			return nil, fmt.Errorf("expected a list of choice keys: %w", err)
		}
		resp = &v1.AskQuestionResponse{Selected: keys}
	case yaml.MappingNode:
		if err := rejectUnknownKeys(&r.node, "question_id", "selected", "text", "sections"); err != nil {
			return nil, err
		}
		var m recommendedMapping
		if err := r.node.Decode(&m); err != nil {
			return nil, err
		}
		resp = &v1.AskQuestionResponse{QuestionId: m.QuestionID, Selected: m.Selected, Text: m.Text}
		for _, v := range m.Sections {
			resp.Sections = append(resp.Sections, &v1.AskSectionVerdict{SectionKey: v.SectionKey, Verdict: v.Verdict, Note: v.Note})
		}
	default:
		return nil, errors.New("expected a key, a list of keys, a text, or a response mapping")
	}
	return resp, checkRecommended(kind, resp, choiceKeys, sectionKeys)
}

// rejectUnknownKeys fails on a mapping key outside the allowed set, as the
// strict decoder does for the rest of the file.
func rejectUnknownKeys(n *yaml.Node, allowed ...string) error {
	for i := 0; i+1 < len(n.Content); i += 2 {
		key := n.Content[i].Value
		if !slices.Contains(allowed, key) {
			return fmt.Errorf("line %d: unknown key %q (expected %s)", n.Content[i].Line, key, strings.Join(allowed, ", "))
		}
	}
	return nil
}

var sectionVerdicts = map[string]bool{"approve": true, "change": true, "drop": true}

func checkRecommended(kind string, resp *v1.AskQuestionResponse, choiceKeys, sectionKeys map[string]bool) error {
	for _, key := range resp.Selected {
		if !choiceKeys[key] {
			return fmt.Errorf("%q is not one of this question's choice keys", key)
		}
	}
	switch kind {
	case typeSingleChoice, typeVariant:
		if len(resp.Selected) != 1 {
			return fmt.Errorf("a %s recommendation names exactly one choice key", kind)
		}
	case typeMultiChoice:
		if len(resp.Selected) == 0 {
			return errors.New("a multi_choice recommendation names at least one choice key")
		}
	case typeSectionReview:
		if len(resp.Sections) == 0 {
			return errors.New("a section_review recommendation needs sections: [{section_key, verdict}]")
		}
		for _, v := range resp.Sections {
			if sectionKeys != nil && !sectionKeys[v.SectionKey] {
				return fmt.Errorf("section_key %q is not a section of this question", v.SectionKey)
			}
			if !sectionVerdicts[v.Verdict] {
				return fmt.Errorf("verdict %q for section %q must be approve, change, or drop", v.Verdict, v.SectionKey)
			}
		}
	default:
		if strings.TrimSpace(resp.Text) == "" {
			return errors.New("a text recommendation needs text")
		}
	}
	return nil
}
