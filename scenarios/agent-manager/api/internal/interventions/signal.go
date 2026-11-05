package interventions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

// Signal is what an operator message reports about the agent's work: five
// broad labels, not moods. Labels, rules and the gate:
// docs/reference/operational-stats.md "Operator signals".
type Signal string

const (
	ReportedFailure Signal = "reported_failure"
	Correction      Signal = "correction"
	Frustration     Signal = "frustration"
	Satisfaction    Signal = "satisfaction"
	Neutral         Signal = "neutral"
)

// ExplicitNegative is the Gate key of the pooled bucket reported_failure plus
// correction, by probability mass. It is not a label.
const ExplicitNegative Signal = "explicit_negative"

type signalSpec struct {
	signal      Signal
	description string // read by the decision model; tuned on split=train only
}

// signalTaxonomy is the closed label set in choice-option order.
var signalTaxonomy = []signalSpec{
	{ReportedFailure, "The operator says something the agent built, fixed, ran or claimed is broken, failing, not working, still wrong, missing or not visible: \"it didn't work\", \"still broken\", \"I don't see the change\", \"that fails with this error\", pasted error output, \"nothing happens\"."},
	{Correction, "The operator says the agent was wrong, misunderstood, ignored an instruction or preference, stopped or got stuck, or rejects or redirects the agent's approach or proposal: \"that's not what I meant\", \"I told you not to\", \"you got stuck\", \"I don't want X\", \"don't treat that as a blocker\". Not a plain new request."},
	{Frustration, "Explicit irritation or exasperation with no specific failure or correction named: \"bruh\", \"why in the world\", insults, profanity, repeated complaint tone."},
	{Satisfaction, "Explicit praise or confirmation that something works or is good as the main content: \"great\", \"love it\", \"awesome\", \"that works now\", \"perfect\". A bare thanks before a plain request is neutral."},
	{Neutral, "Everything else: approvals, instructions, answers, questions, new tasks, status requests, information."},
}

// Signals returns every label in choice-option order.
func Signals() []Signal {
	out := make([]Signal, len(signalTaxonomy))
	for i, s := range signalTaxonomy {
		out[i] = s.signal
	}
	return out
}

func knownSignal(label Signal) bool {
	for _, s := range signalTaxonomy {
		if s.signal == label {
			return true
		}
	}
	return false
}

const signalQuestionID = "signal"

// SignalBatch is the most states in one signal call: small enough that a
// decision model running on CPU answers inside the role's 120 s timeout instead
// of failing the batch over to a smaller model.
const SignalBatch = 8

// Signal state bounds: the agent's last text gives the context, the operator's
// opening carries the signal.
const (
	signalOperatorChars = 1500
	signalAgentChars    = 800
)

func signalQuestions() []*inferencev1.DecisionQuestion {
	options := make([]*inferencev1.DecisionOption, len(signalTaxonomy))
	for i, s := range signalTaxonomy {
		options[i] = &inferencev1.DecisionOption{Label: string(s.signal), Description: s.description}
	}
	return []*inferencev1.DecisionQuestion{{
		Id:           signalQuestionID,
		Type:         inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE,
		Instructions: "The state is an operator_message a human sent to an AI coding agent, and the agent's preceding_agent_text. What does operator_message report about the agent's work? Judge it in the context of preceding_agent_text.",
		Options:      options,
	}}
}

// SignalVersion identifies the signal question set and state bounds: a stored
// signal is valid only for the questions that produced it, and a gate measured
// on one version does not transfer to another.
var SignalVersion = func() string {
	body, _ := json.Marshal(signalQuestions())
	sum := sha256.Sum256(append(body, fmt.Sprintf("|%d|%d", signalOperatorChars, signalAgentChars)...))
	return "signal-" + hex.EncodeToString(sum[:])[:12]
}()

// SignalAnswer is the decision for one message.
type SignalAnswer struct {
	Status        Status
	Signal        Signal
	Probabilities map[Signal]float64 // empty when the method reports no distribution
	Method        string
	Provider      string
	Model         string
	ResourceRole  string
	Reason        string
}

// Probability is the answer's probability for its label, nil when the method
// reports no distribution.
func (a SignalAnswer) Probability() *float64 {
	p, ok := a.Probabilities[a.Signal]
	if !ok {
		return nil
	}
	return &p
}

// ExplicitNegativeMass is the probability on reported_failure and correction,
// nil without a distribution.
func (a SignalAnswer) ExplicitNegativeMass() *float64 {
	if len(a.Probabilities) == 0 {
		return nil
	}
	mass := a.Probabilities[ReportedFailure] + a.Probabilities[Correction]
	return &mass
}

// SignalGate lists, per answering model, the labels proven on the hold-out
// (see the gate in docs/reference/operational-stats.md) and the minimum
// probability each was proven at. A message counts as a label when the model's
// probability for that label reaches the threshold, whether or not it is the
// most likely label. A label absent for a model is stored but never reported as
// a rate.
type SignalGate map[string]map[Signal]float64

// Shown reports whether a message may be counted as that label, given the
// model's probability for the label. Answers without a distribution pass nil and
// count only for a model proven at 0.
func (g SignalGate) Shown(model string, label Signal, probability *float64) bool {
	threshold, ok := g[model][label]
	if !ok {
		return false
	}
	if probability == nil {
		return threshold == 0
	}
	return *probability >= threshold
}

// Proven reports whether any model is proven for the label.
func (g SignalGate) Proven(label Signal) bool {
	for _, labels := range g {
		if _, ok := labels[label]; ok {
			return true
		}
	}
	return false
}

// ClassifySignals answers the signal question for each message, in order. The
// request carries the local-only profile: message text never goes to a hosted
// decision model.
func (c *Classifier) ClassifySignals(ctx context.Context, messages []Message) ([]SignalAnswer, error) {
	answers := make([]SignalAnswer, len(messages))
	client, err := c.resolve(ctx)
	if err != nil {
		for i := range answers {
			answers[i] = SignalAnswer{Status: StatusUnclassified, Reason: err.Error()}
		}
		return answers, fmt.Errorf("%w: %v", ErrGatewayUnavailable, err)
	}
	return answers, c.runBatches(len(messages), SignalBatch, func(start, end int) error {
		return c.signalBatch(ctx, client, messages[start:end], answers[start:end])
	})
}

func (c *Classifier) signalBatch(ctx context.Context, client DecideClient, messages []Message, out []SignalAnswer) error {
	results, err := c.decideSignals(ctx, client, messages, 1)
	if err != nil {
		for i := range out {
			out[i] = SignalAnswer{Status: StatusUnclassified, Reason: err.Error()}
		}
		return err
	}
	var shrink []int
	for i, result := range results {
		out[i] = signalAnswerFrom(result)
		if overflowed(result) {
			shrink = append(shrink, i)
		}
	}
	if len(shrink) == 0 {
		return nil
	}
	retry := make([]Message, len(shrink))
	for j, i := range shrink {
		retry[j] = messages[i]
	}
	retried, err := c.decideSignals(ctx, client, retry, shrinkScale)
	if err != nil {
		return nil // the first answers stand
	}
	for j, i := range shrink {
		if answer := signalAnswerFrom(retried[j]); answer.Status == StatusClassified {
			out[i] = answer
		}
	}
	return nil
}

func (c *Classifier) decideSignals(ctx context.Context, client DecideClient, messages []Message, scale float64) ([]*inferencev1.DecideResponse, error) {
	states := make([]string, len(messages))
	for i, message := range messages {
		body, _ := json.Marshal(struct {
			PrecedingAgentText string `json:"preceding_agent_text"`
			OperatorMessage    string `json:"operator_message"`
		}{tail(message.PrecedingAgentText, int(signalAgentChars*scale)), head(message.OperatorText, int(signalOperatorChars*scale))})
		states[i] = string(body)
	}
	return c.decideStates(ctx, client, signalQuestions(), sharedv1.Profile_PROFILE_LOCAL_ONLY, states)
}

func signalAnswerFrom(result *inferencev1.DecideResponse) SignalAnswer {
	if e := result.GetError(); e != nil && e.GetCode() != inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNSPECIFIED {
		return SignalAnswer{Status: StatusUnclassified, Reason: strings.ToLower(strings.TrimPrefix(e.GetCode().String(), "INFERENCE_ERROR_CODE_")) + ": " + e.GetMessage()}
	}
	var choice *inferencev1.DecisionAnswer
	for _, answer := range result.GetAnswers() {
		if answer.GetQuestionId() == signalQuestionID {
			choice = answer
		}
	}
	label := Signal(choice.GetChoice())
	if !knownSignal(label) {
		return SignalAnswer{Status: StatusUnclassified, Reason: fmt.Sprintf("invalid_answer: %q is not a signal", choice.GetChoice())}
	}
	answer := SignalAnswer{
		Status: StatusClassified, Signal: label,
		Method:   strings.TrimPrefix(result.GetMethod().String(), "DECISION_METHOD_"),
		Provider: result.GetProvider(), Model: result.GetModel(), ResourceRole: result.GetResourceRole(),
	}
	if len(choice.GetProbabilities()) > 0 {
		answer.Probabilities = make(map[Signal]float64, len(choice.GetProbabilities()))
		for _, p := range choice.GetProbabilities() {
			answer.Probabilities[Signal(p.GetOption())] = p.GetProbability()
		}
	}
	return answer
}

// ProvenSignalGate is the gate measured for SignalVersion
// (docs/reference/operational-stats.md "Operator signals", scored 2026-10-30,
// thresholds picked on split=train and bars fixed before scoring). Winnow-E4B
// (winnow:e4b through decide.accurate), hold-out of 137 turns from 128 sessions
// disjoint from the 116 sessions of the 133 train turns:
//   - explicit_negative (reported_failure + correction probability mass >= 0.40):
//     33/41 predicted, 35 gold: P 0.80 [0.66,0.90], R 0.94 [0.81,0.98]. PROVEN,
//     precision by a thin margin (33/41 = 0.805 against the 0.80 bar).
//   - reported_failure (p >= 0.15): P 0.80 [0.58,0.92], R 1.00, but the Wilson
//     lower bound 0.58 is under the 0.60 bar: not proven.
//   - correction (p >= 0.20): P 0.64, R 0.84: not proven.
//   - satisfaction (9 gold) and frustration (2 gold) have too few hold-out rows.
//   - neutral passes (P 0.99, R 0.77 at p >= 0.85) but is the complement, not a
//     signal, so it is not published.
//
// Laya (decide.fast) fails every label (explicit_negative P 0.28). Nothing but
// explicit_negative is shown as a rate.
//
// What this shows is limited. Winnow-E4B was the better of two models scored on
// this hold-out, and precision met the 0.80 bar by one message of 41. See the
// selection note in the accuracy report.
var ProvenSignalGate = SignalGate{"winnow:e4b": {ExplicitNegative: 0.40}}

// SignalOperatingPoints are the train-picked probabilities at which SignalHoldout
// measured each single label: the only points at which its precision is known. A
// consumer of that precision counts a message at these thresholds, on the label's
// own probability, whether or not it is the most likely label. Laya's thresholds
// went unrecorded and its precision is under the ledger's floor, so it has none.
var SignalOperatingPoints = SignalGate{"winnow:e4b": {ReportedFailure: 0.15, Correction: 0.20}}

// SignalHoldout is the hold-out accuracy measured for SignalVersion, per
// answering model, at each label's train-picked threshold (the rule of
// signalPredicted in signal_benchmark_test.go). The outcome ledger reads the
// precision of reported_failure and correction from here.
var SignalHoldout = SignalMeasurements{
	"winnow:e4b": {
		ReportedFailure: {TP: 16, Pred: 20, Gold: 16}, Correction: {TP: 16, Pred: 25, Gold: 19}, Frustration: {TP: 2, Pred: 2, Gold: 2},
		Satisfaction: {TP: 9, Pred: 12, Gold: 9}, Neutral: {TP: 70, Pred: 71, Gold: 91}, ExplicitNegative: {TP: 33, Pred: 41, Gold: 35},
	},
	"laya:en": {
		ReportedFailure: {TP: 12, Pred: 101, Gold: 16}, Correction: {TP: 14, Pred: 98, Gold: 19}, Frustration: {TP: 1, Pred: 77, Gold: 2},
		Satisfaction: {TP: 9, Pred: 52, Gold: 9}, Neutral: {TP: 61, Pred: 75, Gold: 91}, ExplicitNegative: {TP: 30, Pred: 109, Gold: 35},
	},
}

// SignalsSwitchEnv is the deployment off switch: set to off, 0, false, no or
// disabled, the operator-signal job reads no run events and sends nothing to a
// model, and the measure reports signals as switched off. The operator's runtime
// switch (SignalControl) does the same without a restart. Stored labels stay
// until Store.PurgeSignals removes them.
const SignalsSwitchEnv = "AGENT_MANAGER_OPERATOR_SIGNALS"
