package inference

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

// Decision request bounds. They are the intersection of what every decision
// candidate accepts (Cloudflare Clef takes at most 64 questions; System One
// takes 2-255 choice options and 2-10 score levels), so a request valid here is
// never refused by one candidate's schema and accepted by the next.
const (
	MaxDecisionQuestions = 64
	MinChoiceOptions     = 2
	MaxChoiceOptions     = 255
	MinScoreLevels       = 2
	MaxScoreLevels       = 10
	// MaxDecisionStateBytes bounds the state well inside System One's 65,536
	// token ceiling and the daemon's 8 MiB body limit.
	MaxDecisionStateBytes = 256 << 10
	maxDecisionTextBytes  = 4096
	// probabilityTolerance absorbs providers that round to two decimals; a
	// distribution further from 1 than this is not a distribution.
	probabilityTolerance = 0.02
)

var questionIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,100}$`)

// DecisionRequest is the provider-neutral decision payload. Questions keep the
// proto shape so their order and option order survive to every candidate.
type DecisionRequest struct {
	Role      string
	State     string
	StateJSON string
	Questions []*inferencev1.DecisionQuestion
	Profile   sharedv1.Profile
}

// DecisionError is a request defect, reported with the construct to fix.
type DecisionError struct {
	Construct string
	Message   string
}

func (e *DecisionError) Error() string { return e.Construct + ": " + e.Message }

// ValidateQuestions checks the question set shared by every item of a batch.
func ValidateQuestions(questions []*inferencev1.DecisionQuestion) error {
	if len(questions) == 0 || len(questions) > MaxDecisionQuestions {
		return &DecisionError{"questions", fmt.Sprintf("1 to %d questions are required", MaxDecisionQuestions)}
	}
	seen := map[string]struct{}{}
	for index, question := range questions {
		where := fmt.Sprintf("questions[%d]", index)
		if question == nil || !questionIDPattern.MatchString(question.GetId()) {
			return &DecisionError{where + ".id", "id must be 1-100 letters, digits, '_', '.' or '-'"}
		}
		if _, dup := seen[question.GetId()]; dup {
			return &DecisionError{where + ".id", fmt.Sprintf("id %q is repeated", question.GetId())}
		}
		seen[question.GetId()] = struct{}{}
		if len(question.GetInstructions()) > maxDecisionTextBytes {
			return &DecisionError{where + ".instructions", fmt.Sprintf("instructions exceed %d bytes", maxDecisionTextBytes)}
		}
		if err := validateQuestionShape(where, question); err != nil {
			return err
		}
	}
	return nil
}

func validateQuestionShape(where string, question *inferencev1.DecisionQuestion) error {
	options, levels := question.GetOptions(), question.GetLevels()
	noulText := question.GetTrueDescription() != "" || question.GetFalseDescription() != ""
	switch question.GetType() {
	case inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE:
		if len(levels) > 0 || noulText {
			return &DecisionError{where, "a choice question takes options only"}
		}
		if len(options) < MinChoiceOptions || len(options) > MaxChoiceOptions {
			return &DecisionError{where + ".options", fmt.Sprintf("a choice question needs %d to %d options", MinChoiceOptions, MaxChoiceOptions)}
		}
		labels := map[string]struct{}{}
		for _, option := range options {
			label := option.GetLabel()
			if strings.TrimSpace(label) == "" || label != strings.TrimSpace(label) || len(label) > 100 || len(option.GetDescription()) > maxDecisionTextBytes {
				return &DecisionError{where + ".options", "labels must be 1-100 bytes without surrounding space; descriptions at most 4096 bytes"}
			}
			if _, dup := labels[label]; dup {
				return &DecisionError{where + ".options", fmt.Sprintf("label %q is repeated", label)}
			}
			labels[label] = struct{}{}
		}
	case inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_SCORE:
		if len(options) > 0 || noulText {
			return &DecisionError{where, "a score question takes levels only"}
		}
		if len(levels) < MinScoreLevels || len(levels) > MaxScoreLevels {
			return &DecisionError{where + ".levels", fmt.Sprintf("a score question needs %d to %d levels", MinScoreLevels, MaxScoreLevels)}
		}
		for _, level := range levels {
			if strings.TrimSpace(level) == "" || len(level) > maxDecisionTextBytes {
				return &DecisionError{where + ".levels", "every level needs a description of at most 4096 bytes"}
			}
		}
	case inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_NOUL:
		if len(options) > 0 || len(levels) > 0 {
			return &DecisionError{where, "a noul question takes only optional true/false descriptions"}
		}
	default:
		return &DecisionError{where + ".type", "type must be choice, score or noul"}
	}
	return nil
}

// ValidateState checks one item's state: exactly one of text or JSON, and JSON
// that is an object or an array (System One rejects scalars).
func ValidateState(state, stateJSON string) error {
	switch {
	case state != "" && stateJSON != "":
		return &DecisionError{"state", "set state or state_json, not both"}
	case strings.TrimSpace(state) == "" && strings.TrimSpace(stateJSON) == "":
		return &DecisionError{"state", "state or state_json is required"}
	case len(state)+len(stateJSON) > MaxDecisionStateBytes:
		return &DecisionError{"state", fmt.Sprintf("state exceeds %d bytes", MaxDecisionStateBytes)}
	}
	if stateJSON != "" {
		trimmed := bytes.TrimSpace([]byte(stateJSON))
		if !json.Valid(trimmed) || (trimmed[0] != '{' && trimmed[0] != '[') {
			return &DecisionError{"state_json", "state_json must be a JSON object or array"}
		}
	}
	return nil
}

// systemOneBody renders the request in TypeSafe's System One wire format with
// question and option order preserved; encoding/json maps would reorder both.
func systemOneBody(request DecisionRequest) (string, error) {
	var out bytes.Buffer
	out.WriteString(`{"state":`)
	if request.StateJSON != "" {
		out.Write(bytes.TrimSpace([]byte(request.StateJSON)))
	} else {
		writeJSON(&out, request.State)
	}
	out.WriteString(`,"questions":{`)
	for index, question := range request.Questions {
		if index > 0 {
			out.WriteByte(',')
		}
		writeJSON(&out, question.GetId())
		out.WriteString(`:{"type":`)
		writeJSON(&out, wireType(question.GetType()))
		if question.GetInstructions() != "" {
			out.WriteString(`,"instructions":`)
			writeJSON(&out, question.GetInstructions())
		}
		switch question.GetType() {
		case inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE:
			out.WriteString(`,"criteria":{`)
			for i, option := range question.GetOptions() {
				if i > 0 {
					out.WriteByte(',')
				}
				writeJSON(&out, option.GetLabel())
				out.WriteByte(':')
				writeJSON(&out, option.GetDescription())
			}
			out.WriteByte('}')
		case inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_SCORE:
			out.WriteString(`,"criteria":`)
			writeJSON(&out, question.GetLevels())
		case inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_NOUL:
			if question.GetTrueDescription() != "" || question.GetFalseDescription() != "" {
				criteria := map[string]string{}
				if question.GetTrueDescription() != "" {
					criteria["true"] = question.GetTrueDescription()
				}
				if question.GetFalseDescription() != "" {
					criteria["false"] = question.GetFalseDescription()
				}
				out.WriteString(`,"criteria":`)
				writeJSON(&out, criteria)
			}
		}
		out.WriteByte('}')
	}
	out.WriteString(`}}`)
	if !json.Valid(out.Bytes()) {
		return "", fmt.Errorf("decision request did not encode to valid JSON")
	}
	return out.String(), nil
}

func writeJSON(out *bytes.Buffer, value any) {
	raw, _ := json.Marshal(value)
	out.Write(raw)
}

func wireType(kind inferencev1.DecisionQuestionType) string {
	switch kind {
	case inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE:
		return "choice"
	case inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_SCORE:
		return "score"
	default:
		return "noul"
	}
}

// decisionOutput is the decision-model resource's normalized answer set.
type decisionOutput struct {
	Model     string                     `json:"model"`
	Locality  string                     `json:"locality"`
	Answers   map[string]json.RawMessage `json:"answers"`
	CostUSD   *float64                   `json:"cost_usd"`
	LatencyMS int64                      `json:"latency_ms"`
	Usage     struct {
		InputTokens  *int64 `json:"input_tokens"`
		OutputTokens *int64 `json:"output_tokens"`
	} `json:"usage"`
}

type wireAnswer struct {
	Type          string             `json:"type"`
	Choice        *string            `json:"choice"`
	Score         *float64           `json:"score"`
	Noul          *float64           `json:"noul"`
	Confidence    *float64           `json:"confidence"`
	Probabilities map[string]float64 `json:"probabilities"`
}

// mapAnswers turns a provider answer set into ordered proto answers, refusing
// any set that does not answer every question with an in-range value of the
// declared type. A provider is never trusted to have honored the closed set.
func mapAnswers(questions []*inferencev1.DecisionQuestion, raw map[string]json.RawMessage) ([]*inferencev1.DecisionAnswer, error) {
	answers := make([]*inferencev1.DecisionAnswer, 0, len(questions))
	for _, question := range questions {
		encoded, ok := raw[question.GetId()]
		if !ok {
			return nil, fmt.Errorf("question %q has no answer", question.GetId())
		}
		var wire wireAnswer
		if err := json.Unmarshal(encoded, &wire); err != nil {
			return nil, fmt.Errorf("question %q answer: %w", question.GetId(), err)
		}
		if wire.Type != wireType(question.GetType()) {
			return nil, fmt.Errorf("question %q answered as %q, asked as %q", question.GetId(), wire.Type, wireType(question.GetType()))
		}
		answer := &inferencev1.DecisionAnswer{QuestionId: question.GetId(), Type: question.GetType()}
		var err error
		switch question.GetType() {
		case inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE:
			err = mapChoice(question, wire, answer)
		case inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_SCORE:
			err = mapScore(question, wire, answer)
		default:
			if wire.Noul == nil || !unit(*wire.Noul) {
				err = fmt.Errorf("noul must be a probability")
			}
			answer.Noul = wire.Noul
		}
		if err != nil {
			return nil, fmt.Errorf("question %q: %w", question.GetId(), err)
		}
		answers = append(answers, answer)
	}
	return answers, nil
}

func mapChoice(question *inferencev1.DecisionQuestion, wire wireAnswer, answer *inferencev1.DecisionAnswer) error {
	labels := make([]string, 0, len(question.GetOptions()))
	for _, option := range question.GetOptions() {
		labels = append(labels, option.GetLabel())
	}
	if wire.Choice == nil || !contains(labels, *wire.Choice) {
		return fmt.Errorf("choice is not one of the declared labels")
	}
	answer.Choice = *wire.Choice
	return mapDistribution(labels, wire, answer)
}

func mapScore(question *inferencev1.DecisionQuestion, wire wireAnswer, answer *inferencev1.DecisionAnswer) error {
	top := float64(len(question.GetLevels()) - 1)
	if wire.Score == nil || math.IsNaN(*wire.Score) || *wire.Score < 0 || *wire.Score > top {
		return fmt.Errorf("score must lie between 0 and %g", top)
	}
	answer.Score = wire.Score
	levels := make([]string, 0, len(question.GetLevels()))
	for index := range question.GetLevels() {
		levels = append(levels, strconv.Itoa(index))
	}
	return mapDistribution(levels, wire, answer)
}

func mapDistribution(options []string, wire wireAnswer, answer *inferencev1.DecisionAnswer) error {
	if wire.Confidence != nil && !unit(*wire.Confidence) {
		return fmt.Errorf("confidence must be in [0,1]")
	}
	if len(wire.Probabilities) != len(options) {
		return fmt.Errorf("probabilities must cover exactly the %d declared options", len(options))
	}
	var sum float64
	for _, option := range options {
		probability, ok := wire.Probabilities[option]
		if !ok || !unit(probability) {
			return fmt.Errorf("probability for %q is missing or outside [0,1]", option)
		}
		sum += probability
		answer.Probabilities = append(answer.Probabilities, &inferencev1.DecisionProbability{Option: option, Probability: probability})
	}
	if math.Abs(sum-1) > probabilityTolerance {
		return fmt.Errorf("probabilities sum to %.4f, not 1", sum)
	}
	answer.Confidence = wire.Confidence
	if answer.Confidence == nil {
		value := normalizedConfidence(answer.Probabilities)
		answer.Confidence = &value
	}
	return nil
}

// normalizedConfidence is System One's confidence definition,
// (K * p_max - 1) / (K - 1) clamped to [0,1], used when a provider reports the
// distribution but not the confidence derived from it.
func normalizedConfidence(probabilities []*inferencev1.DecisionProbability) float64 {
	var top float64
	for _, probability := range probabilities {
		top = math.Max(top, probability.GetProbability())
	}
	k := float64(len(probabilities))
	return math.Min(1, math.Max(0, (k*top-1)/(k-1)))
}

func unit(value float64) bool { return !math.IsNaN(value) && value >= 0 && value <= 1 }

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// enumFallback expresses a choice-only question set as the typed classify
// path's object-of-enums schema plus an instruction naming every option. It
// reports false when any question is not a choice: a generative label is not a
// score or a probability, and inventing one would misstate the answer.
func enumFallback(request DecisionRequest) (ProviderRequest, bool) {
	properties := map[string]any{}
	required := make([]string, 0, len(request.Questions))
	var instruction strings.Builder
	instruction.WriteString("Answer every question about the source by choosing exactly one allowed label. Return a JSON object keyed by question id.\n")
	for _, question := range request.Questions {
		if question.GetType() != inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE {
			return ProviderRequest{}, false
		}
		labels := make([]string, 0, len(question.GetOptions()))
		fmt.Fprintf(&instruction, "\n%s: %s\n", question.GetId(), firstNonBlank(question.GetInstructions(), question.GetId()))
		for _, option := range question.GetOptions() {
			labels = append(labels, option.GetLabel())
			fmt.Fprintf(&instruction, "  - %s: %s\n", option.GetLabel(), firstNonBlank(option.GetDescription(), option.GetLabel()))
		}
		properties[question.GetId()] = map[string]any{"type": "string", "enum": labels}
		required = append(required, question.GetId())
	}
	schema, _ := json.Marshal(map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false})
	return ProviderRequest{
		Source: firstNonBlank(request.State, request.StateJSON), SchemaJSON: string(schema),
		Instruction: instruction.String(), Profile: request.Profile,
	}, true
}

// enumAnswers maps a validated classify value onto choice answers that carry
// no confidence and no probabilities.
func enumAnswers(questions []*inferencev1.DecisionQuestion, valueJSON string) ([]*inferencev1.DecisionAnswer, error) {
	var value map[string]string
	if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
		return nil, fmt.Errorf("decode enum fallback value: %w", err)
	}
	answers := make([]*inferencev1.DecisionAnswer, 0, len(questions))
	for _, question := range questions {
		label, ok := value[question.GetId()]
		if !ok {
			return nil, fmt.Errorf("enum fallback left %q unanswered", question.GetId())
		}
		answers = append(answers, &inferencev1.DecisionAnswer{QuestionId: question.GetId(), Type: question.GetType(), Choice: label})
	}
	return answers, nil
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
