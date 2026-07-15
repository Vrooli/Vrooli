package agentops

import (
	"encoding/json"
	"fmt"
)

// OperationID is a named operation-contract identity. The vocabulary below
// enumerates the operations seeded from the Phase-1 agent cutover ledger's 14
// target-bound behaviors; full contract authoring (prompts, richer inputs)
// happens in Phase 4. Ids are noun-ish behavior names, never noun+verb target
// composites.
type OperationID string

const (
	OpResearchRefine        OperationID = "research-refine"
	OpWorkshopRound         OperationID = "workshop-round"
	OpWorkshopFinalize      OperationID = "workshop-finalize"
	OpClarificationStart    OperationID = "clarification-start"
	OpClarificationContinue OperationID = "clarification-continue"
	OpExecutionRun          OperationID = "execution-run"
	OpExecutionRetry        OperationID = "execution-retry"
	OpExecutionFollowup     OperationID = "execution-followup"
	OpExecutionFixup        OperationID = "execution-fixup"
	OpReviewRound           OperationID = "review-round"
	OpEvidenceRequest       OperationID = "evidence-request"
	OpRevision              OperationID = "revision"
	OpInitiativeReview      OperationID = "initiative-review"
	OpResearchConclude      OperationID = "research-conclude"
	OpSpecSync              OperationID = "spec-sync"
)

// AllOperationIDs is the canonical ordered operation vocabulary.
var AllOperationIDs = []OperationID{
	OpResearchRefine, OpWorkshopRound, OpWorkshopFinalize,
	OpClarificationStart, OpClarificationContinue, OpExecutionRun,
	OpExecutionRetry, OpExecutionFollowup, OpExecutionFixup,
	OpReviewRound, OpEvidenceRequest, OpRevision, OpInitiativeReview,
	OpResearchConclude, OpSpecSync,
}

// IsValidOperationID reports whether id is a registered operation.
func IsValidOperationID(id OperationID) bool {
	for _, o := range AllOperationIDs {
		if o == id {
			return true
		}
	}
	return false
}

// OperationContract is the typed shape of operation-contract.schema.json.
type OperationContract struct {
	Kind               string                `json:"kind"`
	ID                 OperationID           `json:"id"`
	Version            string                `json:"version"`
	Summary            string                `json:"summary"`
	Description        string                `json:"description,omitempty"`
	TargetRequirements TargetRequirements    `json:"target_requirements"`
	Inputs             []CallerInput         `json:"inputs,omitempty"`
	Result             ResultSchema          `json:"result"`
	Outcomes           []Outcome             `json:"outcomes"`
	EvidenceExpects    []EvidenceExpectation `json:"evidence_expectations,omitempty"`
	Cancellation       *CancellationPolicy   `json:"cancellation,omitempty"`
	Retry              *RetryPolicy          `json:"retry,omitempty"`
}

type TargetRequirements struct {
	Capabilities []CapabilityID `json:"capabilities"`
}

type CallerInput struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required,omitempty"`
	Sensitivity string `json:"sensitivity,omitempty"`
	Retention   string `json:"retention"`
	Description string `json:"description,omitempty"`
}

type ResultSchema struct {
	Fields []ResultField `json:"fields"`
}

type ResultField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required,omitempty"`
	Enum        []any  `json:"enum,omitempty"`
	Description string `json:"description,omitempty"`
}

type Outcome struct {
	Name        string `json:"name"`
	Disposition string `json:"disposition"`
	Description string `json:"description,omitempty"`
}

type EvidenceExpectation struct {
	SubjectKind   string `json:"subject_kind"`
	Action        string `json:"action"`
	Producer      string `json:"producer,omitempty"`
	MinConfidence string `json:"min_confidence"`
	MinCount      int    `json:"min_count,omitempty"`
}

type CancellationPolicy struct {
	Cancelable          *bool  `json:"cancelable,omitempty"`
	CancelIsCooperative *bool  `json:"cancel_is_cooperative,omitempty"`
	InFlightDisposition string `json:"in_flight_disposition,omitempty"`
}

type RetryPolicy struct {
	Retryable         *bool `json:"retryable,omitempty"`
	RetryIsNewAttempt *bool `json:"retry_is_new_attempt,omitempty"`
	MaxAttempts       int   `json:"max_attempts,omitempty"`
}

// handoffResult is the common result shape most operations emit: the structured
// handoff object a round produces plus a routing `progress` field bounded to the
// continue/complete/blocked vocabulary the operating-mode classifier derives
// from the handoff. It is the operation-contract twin of the operating-mode
// handoff declared output.
func handoffResult() ResultSchema {
	return ResultSchema{Fields: []ResultField{
		{Name: "handoff", Type: "object", Required: true, Description: "Structured handoff: summary, blockers, next step, changed files, tests. The single artifact the next round consumes and the classifier derives progress from."},
		{
			Name: "progress", Type: "string", Required: true, Enum: []any{"continue", "complete", "blocked"},
			Description: "Routing signal derived from the handoff: continue (another round is warranted), complete (the operation's unit of work is done), or blocked (needs operator attention).",
		},
	}}
}

// workshopRoundResult is the enriched result a synthesis/research round emits
// (decision B): the common handoff + routing progress, PLUS the two structured
// artifacts the legacy workshop.Round carries — the lettered-option decisions the
// operator answers, and the 5-dimension readiness self-assessment. The
// save-workshop-round action handler writes round-NNN.json deterministically from
// this validated result (the mode never writes the domain folder). The contract
// result schema is intentionally flat — the runner validates top-level
// presence/enums — so the per-decision element shape is documented here and
// parsed by the handler, mirroring internal/workshop.Round (Items + Readiness).
func workshopRoundResult() ResultSchema {
	fields := append([]ResultField(nil), handoffResult().Fields...)
	fields = append(fields,
		ResultField{
			Name: "decisions", Type: "array", Required: true,
			Description: "The lettered-option decisions this round surfaces for the operator, each an object {id, topic, text, context?, options:[{key, label, rationale, recommended?}]}. An empty array when the round surfaces no new decisions; the operator answers each by selecting an option key or providing freeform.",
		},
		ResultField{
			Name: "self_assessment", Type: "object", Required: true,
			Description: "The agent's 5-dimension readiness self-assessment {problem_clarity, scope_defined, approach_solid, testable, risk_awareness}, each an integer 0-3. An INPUT to the workshop service's readiness computation (which applies round-count boosts), not the final readiness verdict.",
		},
	)
	return ResultSchema{Fields: fields}
}

// workshopFinalizeResult is the enriched result the terminal finalize round
// emits: the common handoff + routing progress, PLUS the canonical readiness
// plan_ref the finalize round authored for the domain bind-plan action to bind.
// plan_ref is declared but NOT required: the runner's result-schema validation
// (validateDeliveredResult) enforces required fields on EVERY non-abstain outcome
// — including a `continue` finalize round that has not authored a plan yet — so
// forcing presence would wrongly fail an in-progress finalize. Presence is
// enforced where it is meaningful: the bind-plan domain handler, which fires only
// on the terminal `completed` outcome and fails closed when the ref is absent or
// invalid. Declaring it here makes the ref a first-class top-level result field
// the bridge forwards verbatim, so bind-plan reads a validated canonical ref
// instead of digging it out of the freeform handoff.
func workshopFinalizeResult() ResultSchema {
	fields := append([]ResultField(nil), handoffResult().Fields...)
	fields = append(fields,
		ResultField{
			Name: "plan_ref", Type: "object", Required: false,
			Description: "The canonical readiness plan reference the finalize round authored and hands off for the domain bind-plan action: {provider:'plan-manager', plan_id, slug, role:'execution_spec'}. Present on the terminal complete round; the domain bind-plan action reads and validates it (fail-closed).",
		},
	)
	return ResultSchema{Fields: fields}
}

// reviewResult is the result shape a review-style operation emits: a normalized
// acceptance verdict plus the review handoff. The implementing operating mode
// emits the raw review classification (ready/ready_with_notes/needs_work/
// not_assessable) which the cutover driver maps onto this normalized verdict.
func reviewResult() ResultSchema {
	return ResultSchema{Fields: []ResultField{
		{
			Name: "verdict", Type: "string", Required: true, Enum: []any{"accepted", "changes-requested", "failed"},
			Description: "Normalized acceptance verdict. The review mode's raw classification (ready/ready_with_notes -> accepted, needs_work -> changes-requested, not_assessable -> needs-attention, invalid -> failed) maps onto this closed set.",
		},
		{Name: "handoff", Type: "object", Required: true, Description: "The review handoff: assessment, classification, evidence gathered, and improvement suggestions."},
	}}
}

func standardOutcomes() []Outcome {
	return []Outcome{
		{Name: "completed", Disposition: "success", Description: "The operation's unit of work finished; the workflow advances (e.g. opens review)."},
		{Name: "continue", Disposition: "continue", Description: "Another round of the same operation is warranted; the loop continues."},
		{Name: "blocked", Disposition: "blocked", Description: "Progress is blocked; the round parks for operator attention."},
		{Name: "needs-attention", Disposition: "abstain", Description: "The classifier could not derive an honest outcome; the round abstains to operator attention."},
	}
}

func reviewOutcomes() []Outcome {
	return []Outcome{
		{Name: "accepted", Disposition: "success", Description: "The reviewed work meets its criteria; the workflow completes."},
		{Name: "changes-requested", Disposition: "needs-decision", Description: "Specific fixable gaps were found; the workflow requests a revision/fixup."},
		{Name: "failed", Disposition: "failed", Description: "The review round itself failed (unassessable or invalid)."},
		{Name: "needs-attention", Disposition: "abstain", Description: "The verdict could not be derived honestly; the round abstains to operator attention."},
	}
}

func in(name, typ string, required bool, desc string) CallerInput {
	return CallerInput{Name: name, Type: typ, Required: required, Sensitivity: "internal", Retention: "value", Description: desc}
}

func ev(subject, action, confidence string) EvidenceExpectation {
	return EvidenceExpectation{SubjectKind: subject, Action: action, MinConfidence: confidence, MinCount: 1}
}

func boolPtr(b bool) *bool { return &b }

// SeedOperationContracts returns the authored operation catalog: every ledger
// target-bound behavior as a provider-neutral contract declaring its required
// target capabilities, typed caller inputs, typed result + closed outcomes,
// evidence expectations, and cancellation/retry semantics — grounded in what the
// legacy backlog/execution/review services pass and parse today (see
// docs/internal/AGENT-CUTOVER-LEDGER.md and docs/operations/migration/). The
// on-disk operation-contracts/*.json are materialized from this SSOT by
// `go run ./api/cmd/genopscatalog <scenario-root>`.
func SeedOperationContracts() []OperationContract {
	full := func(id OperationID, summary, description string, caps []CapabilityID, inputs []CallerInput, result ResultSchema, outcomes []Outcome, evidence []EvidenceExpectation, cancel *CancellationPolicy, retry *RetryPolicy) OperationContract {
		return OperationContract{
			Kind: "agentops-operation-contract", ID: id, Version: "1.0.0",
			Summary: summary, Description: description,
			TargetRequirements: TargetRequirements{Capabilities: caps},
			Inputs:             inputs, Result: result, Outcomes: outcomes,
			EvidenceExpects: evidence, Cancellation: cancel, Retry: retry,
		}
	}
	cooperative := &CancellationPolicy{Cancelable: boolPtr(true), CancelIsCooperative: boolPtr(true), InFlightDisposition: "canceled"}
	freshRetry := &RetryPolicy{Retryable: boolPtr(true), RetryIsNewAttempt: boolPtr(true)}
	noRetry := &RetryPolicy{Retryable: boolPtr(false)}

	return []OperationContract{
		full(OpResearchRefine,
			"Autonomous research pass refining a backlog item's spec.",
			"One agent round that investigates the item's kind, deliverable, prior workshop history and attached context, and refines the spec/research toward readiness. Mirrors internal/backlog/research.go: it produces a workshop round envelope and either continues (another round is warranted), completes (the spec is ready for synthesis/finalize), or blocks.",
			[]CapabilityID{CapSpecDocument},
			[]CallerInput{
				in("OPERATOR_NOTE", "string", false, "Optional operator guidance for this research round."),
				in("USER_PROMPT", "string", false, "The operator's research prompt/instruction driving this refinement pass (legacy research request text)."),
				in("CONTEXT_PATHS", "string", false, "Repository paths/files the operator attached as research context, one per line."),
				in("CONTEXT_TARGETS", "string", false, "Named targets (scenarios, resources, components) the operator scoped the research to, one per line."),
				in("CONTEXT_REQUIREMENTS", "string", false, "Explicit requirements or acceptance notes the operator attached to fold into the spec."),
				in("GAP_REPORT", "string", false, "A prior readiness/gap report identifying what the spec still lacks, to steer this round."),
			},
			workshopRoundResult(), standardOutcomes(),
			[]EvidenceExpectation{ev("backlog-item", "workshop-round-recorded", "observed")},
			cooperative, freshRetry),
		full(OpWorkshopRound,
			"One workshop synthesis round over an item spec.",
			"One agent round that synthesizes the workshop: it produces/updates the round envelope (decisions, readiness scores, plan updates) mirroring internal/backlog/workshop_save.go. continue -> another synthesis round; complete -> the item is ready to finalize; blocked -> a dependency or missing input parks the round.",
			[]CapabilityID{CapSpecDocument},
			[]CallerInput{in("OPERATOR_NOTE", "string", false, "Optional operator guidance for this workshop round.")},
			workshopRoundResult(), standardOutcomes(),
			[]EvidenceExpectation{ev("backlog-item", "workshop-round-recorded", "observed")},
			cooperative, freshRetry),
		full(OpWorkshopFinalize,
			"Finalize the workshop and bind the readiness plan_ref.",
			"The workshop's terminal round: all decisions answered and readiness reached, the agent authors the implementation plan and binds it as the item's plan_ref (mirrors the finalize mode of internal/backlog/research.go, which requires >=1 round, no pending decisions, and readiness). complete -> plan_ref bound and the item is execution-ready.",
			[]CapabilityID{CapSpecDocument, CapPlanRef},
			[]CallerInput{in("OPERATOR_NOTE", "string", false, "Optional operator guidance for finalization.")},
			workshopFinalizeResult(), standardOutcomes(),
			[]EvidenceExpectation{ev("backlog-item", "readiness-plan-bound", "observed")},
			cooperative, freshRetry),
		full(OpClarificationStart,
			"Open a workshop clarification thread on an item.",
			"Opens a clarification thread against a specific workshop decision and runs the first agent turn (mirrors internal/backlog/clarification.go CreateClarification). The agent answers the operator's question and assesses impact (none/decision/round). continue -> the thread stays active for follow-up; complete -> the clarification resolved; blocked -> the item is busy.",
			[]CapabilityID{CapClarificationThread, CapSpecDocument},
			[]CallerInput{
				in("USER_QUESTION", "string", false, "The operator's question about the workshop decision."),
				in("DECISION_TOPIC", "string", false, "The workshop decision topic being clarified."),
			},
			handoffResult(), standardOutcomes(),
			[]EvidenceExpectation{ev("backlog-item", "clarification-thread-updated", "observed")},
			cooperative, freshRetry),
		full(OpClarificationContinue,
			"Continue an item's clarification thread.",
			"A follow-up turn on an active clarification thread (mirrors internal/backlog/clarification.go ContinueClarification, a ContinueRun on the existing clarification run). continue -> more turns expected; complete -> resolved; blocked -> thread inactive/expired.",
			[]CapabilityID{CapClarificationThread},
			[]CallerInput{in("USER_MESSAGE", "string", true, "The operator's follow-up message on the active thread.")},
			handoffResult(), standardOutcomes(),
			[]EvidenceExpectation{ev("backlog-item", "clarification-thread-updated", "observed")},
			cooperative, noRetry),
		full(OpExecutionRun,
			"Drain the item's bound plan as the primary execution run.",
			"The primary execution of a unit of work: drain the bound plan slice-by-slice via the generic phased-plan-drain (the implementing mode targets plan-execution and delegates executed_by phased-plan-drain). Mirrors internal/execution/service_control.go startLocked. complete -> the plan is fully drained and review opens; blocked -> a slice cannot proceed.",
			[]CapabilityID{CapExecutionWorkspace, CapPlanRef},
			[]CallerInput{in("OPERATOR_NOTE", "string", false, "Optional operator guidance for the execution run.")},
			handoffResult(), standardOutcomes(),
			[]EvidenceExpectation{ev("plan-execution", "execution-slice-completed", "observed")},
			cooperative, freshRetry),
		full(OpExecutionRetry,
			"Fresh execution attempt parented to a failed run.",
			"A retry re-drains the plan as a brand-new execution parented to the failed one (Retry-as-New-Attempt; mirrors internal/execution/retry.go). It is never an in-place mutation of the prior run.",
			[]CapabilityID{CapExecutionWorkspace, CapPlanRef},
			[]CallerInput{in("RETRY_NOTE", "string", false, "Optional operator note carried into the retry as follow-up context.")},
			handoffResult(), standardOutcomes(),
			[]EvidenceExpectation{ev("plan-execution", "execution-slice-completed", "observed")},
			cooperative, &RetryPolicy{Retryable: boolPtr(true), RetryIsNewAttempt: boolPtr(true), MaxAttempts: 5}),
		full(OpExecutionFollowup,
			"Follow-up run after an item completes.",
			"An additional agent run after an item's execution completed, to extend or amend the delivered work (mirrors internal/execution/followup.go FollowUp, both the fresh-run and continue paths). Reads the completed deliverable and any review feedback.",
			[]CapabilityID{CapExecutionWorkspace},
			[]CallerInput{
				in("FOLLOWUP_NOTE", "string", false, "Operator context describing the follow-up work."),
				in("FOLLOWUP_TYPE", "string", false, "The follow-up flavor (followup/custom/fixup)."),
			},
			handoffResult(), standardOutcomes(),
			[]EvidenceExpectation{ev("backlog-item", "execution-followup-completed", "observed")},
			cooperative, freshRetry),
		full(OpExecutionFixup,
			"Autonomous fixup run after a review found remediable issues.",
			"A remediation run that reads the review feedback (GCT dimensions, baseline diff, changed paths) and addresses the specific gaps before re-review (mirrors internal/execution/followup.go spawnFixupRun). Requires both an execution workspace and review artifacts, so it only runs against a backlog item.",
			[]CapabilityID{CapExecutionWorkspace, CapReviewArtifacts},
			[]CallerInput{in("OPERATOR_NOTE", "string", false, "Optional operator guidance for the fixup.")},
			handoffResult(), standardOutcomes(),
			[]EvidenceExpectation{ev("backlog-item", "execution-fixup-completed", "observed")},
			cooperative, &RetryPolicy{Retryable: boolPtr(true), RetryIsNewAttempt: boolPtr(true), MaxAttempts: 3}),
		full(OpReviewRound,
			"One review round over completed work.",
			"One agent round that reviews completed execution work: gathers evidence (screenshots, api tests, cli output, config diffs, recordings) and returns a classification and assessment (mirrors internal/review/service.go). The mode emits the raw classification {ready, ready_with_notes, needs_work, not_assessable}; this contract normalizes it to accepted/changes-requested/failed. Requires review artifacts, which both backlog-item and initiative provide.",
			[]CapabilityID{CapReviewArtifacts},
			nil,
			reviewResult(), reviewOutcomes(),
			[]EvidenceExpectation{ev("backlog-item", "review-round-recorded", "observed")},
			cooperative, freshRetry),
		full(OpEvidenceRequest,
			"Gather additional evidence for a review round.",
			"A targeted sub-round of a review that gathers the specific additional evidence an operator requested, appending it to the open review round (mirrors internal/review/rounds.go RequestMoreEvidence). continue -> the request thread stays open; complete -> the requested evidence was gathered.",
			[]CapabilityID{CapReviewArtifacts, CapEvidenceLedger},
			[]CallerInput{in("EVIDENCE_REQUEST", "string", true, "The specific evidence the operator is asking the agent to gather.")},
			handoffResult(), standardOutcomes(),
			[]EvidenceExpectation{ev("backlog-item", "review-evidence-gathered", "observed")},
			cooperative, freshRetry),
		full(OpRevision,
			"Request a targeted revision of an item's spec/plan.",
			"A targeted revision round that re-opens the spec/plan to address a specific requested change before re-execution (the changes-requested path of the review/decision loop). continue -> further revision; complete -> the revision is applied and the item re-enters its pipeline.",
			[]CapabilityID{CapSpecDocument},
			[]CallerInput{in("REVISION_REQUEST", "string", true, "The specific change the operator wants applied to the spec/plan.")},
			handoffResult(), standardOutcomes(),
			[]EvidenceExpectation{ev("backlog-item", "revision-recorded", "observed")},
			cooperative, freshRetry),
		full(OpInitiativeReview,
			"Autonomous initiative-level review across member items and criteria.",
			"One agent round that reviews a whole initiative against its acceptance criteria once all member items are terminal (mirrors internal/initiativereview/trigger.go). Reuses the review round schema and classification vocabulary at the initiative scope. Requires member items + acceptance criteria + review artifacts, so it only runs against an initiative.",
			[]CapabilityID{CapMemberItems, CapAcceptanceCriteria, CapReviewArtifacts},
			nil,
			reviewResult(), reviewOutcomes(),
			[]EvidenceExpectation{ev("initiative", "initiative-review-recorded", "observed")},
			cooperative, freshRetry),
		full(OpResearchConclude,
			"Execute a plan-less item to produce its conclusion deliverable.",
			"The primary execution of a research/idea item that has no execution plan_ref: the agent works the item to produce its conclusion deliverable directly (mirrors internal/execution/service_control.go startConclusionSpawnLocked and the plan-less path of internal/execution/retry.go). Distinct from execution-run, which drains a bound plan; a research conclusion has no plan to drain. complete -> the deliverable is produced and review opens; blocked -> the work cannot proceed.",
			[]CapabilityID{CapExecutionWorkspace, CapSpecDocument},
			[]CallerInput{in("OPERATOR_NOTE", "string", false, "Optional operator guidance for the conclusion run (a retry-as-new-attempt carries its note here too).")},
			handoffResult(), standardOutcomes(),
			[]EvidenceExpectation{ev("backlog-item", "execution-conclusion-recorded", "observed")},
			cooperative, freshRetry),
		full(OpSpecSync,
			"Sync a scenario's spec artifacts to match its implementation.",
			"One or more agent rounds that read a scenario's implementation code and update its spec artifacts (PRD.md, requirements/, README.md, docs/) to match the actual behavior (mirrors internal/execution/service_queue.go QueueSpecSyncArchive). Targets a scenario workspace directly — there is no backlog item, plan, or review state; the deliverable is the scenario's own synced spec. On completion the execution service archives + removes the scenario directory. complete -> the spec is synced; blocked -> the sync cannot proceed.",
			[]CapabilityID{CapExecutionWorkspace, CapSpecDocument},
			[]CallerInput{in("OPERATOR_NOTE", "string", false, "Optional operator guidance for the spec-sync run.")},
			handoffResult(), standardOutcomes(),
			[]EvidenceExpectation{ev("scenario", "spec-sync-recorded", "observed")},
			cooperative, freshRetry),
	}
}

// ValidateOperationContract validates a contract document against the schema
// and the semantic rules JSON Schema cannot express: the id is a registered
// operation, required capabilities are in the closed vocabulary, at least one
// registered target kind can satisfy the requirements (an unrunnable contract
// is rejected), and outcome dispositions are internally consistent.
func ValidateOperationContract(raw []byte) error {
	if err := ValidateDocument(SchemaOperationContract, raw); err != nil {
		return err
	}
	var oc OperationContract
	if err := json.Unmarshal(raw, &oc); err != nil {
		return fmt.Errorf("decode operation contract: %w", err)
	}
	if !IsValidOperationID(oc.ID) {
		return fmt.Errorf("operation contract id %q is not a registered operation", oc.ID)
	}
	for _, cap := range oc.TargetRequirements.Capabilities {
		if !IsValidCapabilityID(cap) {
			return fmt.Errorf("operation %q requires unknown capability %q", oc.ID, cap)
		}
	}
	if len(CompatibleTargets(oc.TargetRequirements.Capabilities)) == 0 {
		return fmt.Errorf("operation %q requires a capability set no registered target kind can satisfy", oc.ID)
	}
	seen := map[string]bool{}
	for _, o := range oc.Outcomes {
		if seen[o.Name] {
			return fmt.Errorf("operation %q declares duplicate outcome %q", oc.ID, o.Name)
		}
		seen[o.Name] = true
	}
	return nil
}
