// Package interventions classifies operator messages to agents (P4-W4): a
// correction that names an agent failure (friction) or new work and replies
// (not friction). Classification runs on ai-gateway's decision capability; the
// package stores one row per operator message and classifier version, with the
// agent work that preceded the message and a repeated-report signal.
package interventions

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

// Category is what an operator message does. The first six are friction.
type Category string

const (
	EarlyStop          Category = "early_stop"
	FalseInability     Category = "false_inability"
	WrongClaim         Category = "wrong_claim"
	QualityMiss        Category = "quality_miss"
	TooDense           Category = "too_dense"
	InstructionIgnored Category = "instruction_ignored"
	ScopeChange        Category = "scope_change"
	ApprovalOrInfo     Category = "approval_or_info"
)

type categorySpec struct {
	category    Category
	friction    bool
	description string // read by the decision model; tuned on split=train only
	owner       string // who can fix the recurring cause
}

// taxonomy is the closed label set in choice-option order. Labels and rules:
// api/internal/runsignal/testdata/interventions/README.md. Owners:
// docs/efforts/usage-intelligence/p4-w4-interventions.md.
var taxonomy = []categorySpec{
	{EarlyStop, true, "Complaint or nudge that the agent stopped, paused, waited for a go-ahead or declared itself done before the job was finished: \"keep going\", \"don't stop\", \"why did you stop\", \"are you stuck\", \"you stop too early\", \"I didn't tell you to stop\", \"did you finish?\", \"finish the rest\", \"do all the next steps at once\". A bare \"continue\" or resuming after a crash or limit, with no complaint and no sign the agent stopped early, is approval_or_info.", "harness-goal-authoring and large-effort-orchestration continuation contract; Agent Manager park/wake"},
	{FalseInability, true, "The operator tells the agent it can or may act on its own, or should do a step itself instead of leaving it to the operator: \"restart it yourself\", \"test it yourself\", \"run the tests yourself\", \"you're allowed to ...\", \"you have permission to ...\", \"you don't need my approval\", \"you should be able to do that\", \"request the approval yourself\", \"why do you need me\". Includes a standing permission added to another request. A plain yes or choice answering the agent's question is approval_or_info.", "docs/concepts/AUTHORIZATION.md and AGENTS.md rule 9; privilege and approval skills"},
	{WrongClaim, true, "Complaint that the agent stated something untrue or misunderstood the operator: \"that's not true\", \"that's not what I meant\", \"I meant X\", \"wrong one\", \"you said Y\", \"not what I'm referring to\". About what the agent said or understood, not about broken work.", "recall before answering: search-hub query, conversation search (AGENTS.md rule 6)"},
	{QualityMiss, true, "The operator reports that something the agent built, fixed or claimed is not right: broken, still failing, incomplete, missing, wrong-looking, cut off, nothing happens, never arrived, or below expectation (\"still broken\", \"not fixed\", \"it doesn't work\", \"I haven't received any notification\", \"it looks odd\", \"you forgot to\", \"unprofessional\", \"I only see X\"), even when the message also says thanks or asks for a follow-up. A report about a symptom is quality_miss unless the message is clearly a brand new unrelated request.", "proof before claim: scientific-debugging, test-genie and ui-health evidence"},
	{TooDense, true, "The operator cannot follow the agent's explanation: \"I'm confused\", \"I don't understand\", \"dumb this down\", \"simpler words\", \"give me the tldr\", \"what does that term mean\". An ordinary question about facts or status is approval_or_info.", "writing-standards operator-facing reports"},
	{InstructionIgnored, true, "The operator objects that the agent did something it was told not to, did something that must not be done, or forgot a standing rule or preference: \"I told you not to\", \"I said\", \"as I said before\", \"you should NEVER\", \"you keep doing X\", \"I don't want that\", \"that is a mistake, it shouldn't be there\", \"don't forget to\", \"this is greenfield\". A fresh instruction about what to do next is scope_change.", "standing-preference hand-off: harness-goal-authoring, report-bug defaults, never-use-git rule"},
	{ScopeChange, false, "A new task, request, feature, bug report about existing behaviour, design idea or direction for the agent to work on next. Not a complaint about the agent.", ""},
	{ApprovalOrInfo, false, "A plain reply that carries no complaint: approving or choosing an option the agent offered, granting permission it asked for, answering its question, thanks, status ping, a question about facts, a command, or resuming after an interruption.", ""},
}

// Categories returns every category in choice-option order.
func Categories() []Category {
	out := make([]Category, len(taxonomy))
	for i, spec := range taxonomy {
		out[i] = spec.category
	}
	return out
}

func spec(category Category) (categorySpec, bool) {
	for _, s := range taxonomy {
		if s.category == category {
			return s, true
		}
	}
	return categorySpec{}, false
}

// Friction reports whether the category names an agent failure.
func (c Category) Friction() bool { s, _ := spec(c); return s.friction }

// Owner names who can remove the category's recurring cause; empty for the
// categories that are not friction.
func (c Category) Owner() string { s, _ := spec(c); return s.owner }

// questionID is the id of the single choice question.
const questionID = "category"

// questions is the decision question set. One choice over every category gives
// a calibrated distribution from decision models and stays answerable by the
// role's choice-only LLM fallback. Per-category yes/no questions were measured
// and were worse (docs/efforts/usage-intelligence/p4-w4-interventions.md, "Classifier benchmark").
func questions() []*inferencev1.DecisionQuestion {
	options := make([]*inferencev1.DecisionOption, len(taxonomy))
	for i, s := range taxonomy {
		options[i] = &inferencev1.DecisionOption{Label: string(s.category), Description: s.description}
	}
	return []*inferencev1.DecisionQuestion{{
		Id:           questionID,
		Type:         inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE,
		Instructions: "The state is an operator_message a human sent to an AI coding agent, and the agent's preceding_agent_text. What is the operator doing in operator_message?",
		Options:      options,
	}}
}

// ClassifierVersion identifies the question set: a stored classification is
// valid only for the questions that produced it, and a gate measured on one
// version does not transfer to another.
var ClassifierVersion = func() string {
	body, _ := json.Marshal(questions())
	sum := sha256.Sum256(body)
	return "choice-" + hex.EncodeToString(sum[:])[:12]
}()

// PooledFriction is the Gate key of the pooled "operator friction (category
// unproven)" bucket. It is not a classification category: its threshold is the
// probability mass on all friction categories (see ShownPooled), never a single
// category's probability.
const PooledFriction Category = "operator_friction"

// Gate lists, per answering model, the categories proven to meet P4-A4
// (precision >= 0.8 and recall >= 0.7 on the held-out split and on the
// untouched validation set, which must hold >= 10 gold rows of the category)
// and the minimum probability each was proven at. A category absent for a
// model is classified and stored but must not be shown. The PooledFriction key,
// when present, proves "some friction, category unproven" at that friction mass.
type Gate map[string]map[Category]float64

// Shown reports whether a classification may be shown as that category.
// probability is the answer's probability for the category; answers without a
// distribution (the LLM enum fallback) pass nil and are shown only if their
// model was proven with threshold 0.
func (g Gate) Shown(model string, category Category, probability *float64) bool {
	threshold, ok := g[model][category]
	if !ok {
		return false
	}
	if probability == nil {
		return threshold == 0
	}
	return *probability >= threshold
}

// ShownPooled reports whether a classification may be counted in the pooled
// "operator friction (category unproven)" bucket. mass is the total probability
// the answer puts on friction categories; answers without a distribution pass
// nil and are shown only if the model was proven with threshold 0. The bucket
// may carry the answer's category as a hint only, never as a count.
func (g Gate) ShownPooled(model string, mass *float64) bool {
	return g.Shown(model, PooledFriction, mass)
}

// FrictionMass is the probability the answer puts on friction categories, nil
// when the answer has no distribution. Pass it to ShownPooled.
func (a Answer) FrictionMass() *float64 {
	if len(a.Probabilities) == 0 {
		return nil
	}
	mass := 0.0
	for category, p := range a.Probabilities {
		if category.Friction() {
			mass += p
		}
	}
	return &mass
}

// Proven reports whether any model is proven for the category, so its count
// (zero included) may be shown.
func (g Gate) Proven(category Category) bool {
	for _, categories := range g {
		if _, ok := categories[category]; ok {
			return true
		}
	}
	return false
}

// ProvenGate is the gate measured for ClassifierVersion. It is empty: on
// 2026-10-30 Winnow-E4B (winnow:e4b through decide.accurate, 1,262/1,262 rows
// answered by it), tuned on split=train only, was scored on the hold-out and the
// untouched validation set (315 operator rows, 67 friction).
//   - Pooled friction (mass >= 0.50, train-picked): hold-out P 0.92 / R 0.75
//     (99/108/132) passes; validation P 0.73 [0.61,0.82] / R 0.72 [0.60,0.81]
//     (48/66/67) fails the precision bar, so the pooled bucket is not shown.
//   - quality_miss (the only category with >= 10 validation gold rows): hold-out
//     P 0.63 / R 0.63, validation P 0.72 / R 0.81; fails precision on both.
//   - Every other friction category has < 10 validation gold rows and cannot be
//     shown individually.
//
// Nothing is shown (docs/efforts/usage-intelligence/p4-w4-interventions.md,
// "Classifier benchmark"). To show the pooled bucket for a model add
// PooledFriction: <friction-mass threshold>. Re-run TestLiveInterventionBenchmark
// after changing the questions, the model or the corpus.
var ProvenGate = Gate{}
