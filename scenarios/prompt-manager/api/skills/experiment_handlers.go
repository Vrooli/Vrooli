package skills

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"prompt-manager/store"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/receiptsigning"
)

// ExperimentHandlers provides HTTP handlers for experiment operations.
type ExperimentHandlers struct {
	experiments              store.ExperimentStore
	variants                 store.VariantStore
	skills                   store.SkillStore
	decisions                decisionPublisher
	receiptSigner            receiptsigning.ReceiptSigner
	productionReceiptSigning bool
}

type decisionPublisher interface {
	AppendDecision(context.Context, string, *store.DecisionEntry) error
}

// decisionReader is deliberately separate from publishing. Promotion must
// inspect the durable team decision log rather than trust a caller-provided
// assertion that an operator approved it.
type decisionReader interface {
	GetDecisions(context.Context, string, string, string, int) ([]store.DecisionEntry, int, error)
}

// NewExperimentHandlers creates experiment handlers.
func NewExperimentHandlers(experiments store.ExperimentStore, variants store.VariantStore, skills store.SkillStore) *ExperimentHandlers {
	return &ExperimentHandlers{experiments: experiments, variants: variants, skills: skills}
}

// SetDecisionPublisher wires experiment recommendations into prompt-manager's
// existing human decision queue. It is intentionally optional for isolated
// handler tests, but production configures it during API startup.
func (h *ExperimentHandlers) SetDecisionPublisher(p decisionPublisher) { h.decisions = p }

// SetReceiptSigner provides the sole receipt-signing boundary. Production
// passes a lifecycle-bound provider; raw signing material never enters here.
func (h *ExperimentHandlers) SetReceiptSigner(signer receiptsigning.ReceiptSigner) {
	h.receiptSigner = signer
}

func (h *ExperimentHandlers) SetProductionReceiptSigningRequired(required bool) {
	h.productionReceiptSigning = required
}

// ListExperiments handles GET /experiments
func (h *ExperimentHandlers) ListExperiments(w http.ResponseWriter, r *http.Request) {
	exps, err := h.experiments.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := make([]ExperimentResponse, 0, len(exps))
	for _, e := range exps {
		resp = append(resp, h.experimentToResponse(r, e, false))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// ListExperimentsBySkill handles GET /skills/{id}/experiments
func (h *ExperimentHandlers) ListExperimentsBySkill(w http.ResponseWriter, r *http.Request) {
	skillID := mux.Vars(r)["id"]

	exps, err := h.experiments.ListBySkill(r.Context(), skillID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := make([]ExperimentResponse, 0, len(exps))
	for _, e := range exps {
		resp = append(resp, h.experimentToResponse(r, e, false))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// GetExperiment handles GET /experiments/{eid}
func (h *ExperimentHandlers) GetExperiment(w http.ResponseWriter, r *http.Request) {
	eid := mux.Vars(r)["eid"]

	exp, err := h.experiments.Get(r.Context(), eid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.experimentToResponse(r, *exp, true))
}

// CreateExperiment handles POST /experiments
func (h *ExperimentHandlers) CreateExperiment(w http.ResponseWriter, r *http.Request) {
	var req CreateExperimentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.ID == "" || req.SkillID == "" || req.Name == "" || req.Hypothesis == "" {
		http.Error(w, "id, skillId, name, and hypothesis are required", http.StatusBadRequest)
		return
	}
	if len(req.Arms) < 2 {
		http.Error(w, "at least 2 arms are required", http.StatusBadRequest)
		return
	}
	normalizeProtocol(&req.Protocol)
	if err := validateProtocol(req.Protocol); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validateWeights(req.Arms); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Verify skill exists
	if _, err := h.skills.Get(r.Context(), req.SkillID); err != nil {
		http.Error(w, "skill not found: "+req.SkillID, http.StatusBadRequest)
		return
	}

	// Verify all non-control variants exist
	for _, arm := range req.Arms {
		if arm.VariantID != store.ControlVariantID {
			if _, err := h.variants.Get(r.Context(), req.SkillID, arm.VariantID); err != nil {
				http.Error(w, "variant not found: "+arm.VariantID, http.StatusBadRequest)
				return
			}
		}
	}

	arms := make([]store.ExperimentArm, len(req.Arms))
	for i, a := range req.Arms {
		arms[i] = store.ExperimentArm{VariantID: a.VariantID, Weight: a.Weight}
	}

	exp := &store.Experiment{
		ID:         req.ID,
		SkillID:    req.SkillID,
		Name:       req.Name,
		Hypothesis: req.Hypothesis,
		Protocol:   req.Protocol,
		Arms:       arms,
	}

	if err := h.experiments.Create(r.Context(), exp); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	created, err := h.experiments.Get(r.Context(), req.ID)
	if err != nil {
		http.Error(w, "created but failed to read back: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(h.experimentToResponse(r, *created, false))
}

// UpdateExperiment handles PUT /experiments/{eid}
func (h *ExperimentHandlers) UpdateExperiment(w http.ResponseWriter, r *http.Request) {
	eid := mux.Vars(r)["eid"]

	existing, err := h.experiments.Get(r.Context(), eid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if existing.Status != store.ExperimentStatusDraft {
		http.Error(w, "can only update draft experiments", http.StatusConflict)
		return
	}

	var req UpdateExperimentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Hypothesis != nil {
		existing.Hypothesis = *req.Hypothesis
	}
	if req.Protocol != nil {
		normalizeProtocol(req.Protocol)
		if err := validateProtocol(*req.Protocol); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		existing.Protocol = *req.Protocol
	}
	if len(req.Arms) > 0 {
		if len(req.Arms) < 2 {
			http.Error(w, "at least 2 arms are required", http.StatusBadRequest)
			return
		}
		if err := validateWeights(req.Arms); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		arms := make([]store.ExperimentArm, len(req.Arms))
		for i, a := range req.Arms {
			arms[i] = store.ExperimentArm{VariantID: a.VariantID, Weight: a.Weight}
		}
		existing.Arms = arms
	}

	if err := h.experiments.Update(r.Context(), eid, existing); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	updated, _ := h.experiments.Get(r.Context(), eid)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.experimentToResponse(r, *updated, false))
}

// DeleteExperiment handles DELETE /experiments/{eid}
func (h *ExperimentHandlers) DeleteExperiment(w http.ResponseWriter, r *http.Request) {
	eid := mux.Vars(r)["eid"]

	if err := h.experiments.Delete(r.Context(), eid); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// StartExperiment handles POST /experiments/{eid}/start
func (h *ExperimentHandlers) StartExperiment(w http.ResponseWriter, r *http.Request) {
	eid := mux.Vars(r)["eid"]

	exp, err := h.experiments.Get(r.Context(), eid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if exp.Status != store.ExperimentStatusDraft {
		http.Error(w, "can only start draft experiments", http.StatusConflict)
		return
	}
	normalizeProtocol(&exp.Protocol)
	if err := validateProtocol(exp.Protocol); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	exp.Protocol.ProtocolHash = protocolHash(exp.Protocol)

	now := time.Now().UTC().Format(time.RFC3339)
	exp.Status = store.ExperimentStatusRunning
	exp.StartedAt = &now

	if err := h.experiments.Update(r.Context(), eid, exp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	updated, _ := h.experiments.Get(r.Context(), eid)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.experimentToResponse(r, *updated, false))
}

// ConcludeExperiment handles POST /experiments/{eid}/conclude
func (h *ExperimentHandlers) ConcludeExperiment(w http.ResponseWriter, r *http.Request) {
	eid := mux.Vars(r)["eid"]

	exp, err := h.experiments.Get(r.Context(), eid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if exp.Status != store.ExperimentStatusRunning {
		http.Error(w, "can only conclude running experiments", http.StatusConflict)
		return
	}

	var req ConcludeExperimentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.WinnerVariantID == "" {
		http.Error(w, "winnerVariantId is required", http.StatusBadRequest)
		return
	}

	// Verify winner is one of the arms
	validWinner := false
	for _, arm := range exp.Arms {
		if arm.VariantID == req.WinnerVariantID {
			validWinner = true
			break
		}
	}
	if !validWinner {
		http.Error(w, "winnerVariantId must be one of the experiment's arms", http.StatusBadRequest)
		return
	}
	gateErr := h.validateRecommendationEvidence(r.Context(), exp, req.WinnerVariantID)
	if gateErr != nil && (!req.Override || strings.TrimSpace(req.OverrideJustification) == "") {
		http.Error(w, gateErr.Error(), http.StatusConflict)
		return
	}
	if err := h.requireReceiptSigner(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	audits, ok := h.experiments.(store.ExperimentAuditStore)
	if !ok {
		http.Error(w, "durable audit store is unavailable", http.StatusServiceUnavailable)
		return
	}
	receipt, err := audits.GetAuditReceipt(r.Context(), eid)
	if err != nil || !h.verifyAuditReceipt(r.Context(), receipt) || receipt.ProtocolHash != exp.Protocol.ProtocolHash || receipt.ChallengeState != "clear" {
		http.Error(w, "a valid clear audit receipt is required before recommendation", http.StatusConflict)
		return
	}
	if h.decisions == nil {
		http.Error(w, "decision publishing is required before an experiment can conclude", http.StatusServiceUnavailable)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	decisionID := fmt.Sprintf("dec-experiment-%x", sha256.Sum256([]byte(eid+"|"+req.WinnerVariantID+"|"+now)))
	rationale := fmt.Sprintf("Experiment %q recommends this variant. Review the protocol, outcome evidence, audit receipts, and holdout validation before accepting.", eid)
	if summary := h.promotionEvidenceSummary(r.Context(), exp, req.WinnerVariantID); summary != "" {
		rationale = rationale + "\n" + summary
	}
	if gateErr != nil {
		rationale = fmt.Sprintf("GATE OVERRIDE: the pre-registered recommendation gate failed (%s) and was overridden with justification: %s. %s", gateErr.Error(), strings.TrimSpace(req.OverrideJustification), rationale)
	}
	decision := &store.DecisionEntry{ID: decisionID, At: now, By: "skill-optimizer", Status: store.DecisionStatusPending, Context: "skill-experiment-promotion", Decision: fmt.Sprintf("Adopt variant %q for skill %q", req.WinnerVariantID, exp.SkillID), Rationale: rationale}
	if err := h.decisions.AppendDecision(r.Context(), "meta-optimization", decision); err != nil {
		http.Error(w, "publish promotion decision: "+err.Error(), http.StatusInternalServerError)
		return
	}
	exp.Status = store.ExperimentStatusConcluded
	exp.ConcludedAt = &now
	exp.WinnerVariantID = &req.WinnerVariantID
	exp.Notes = req.Notes
	if gateErr != nil {
		exp.Notes = strings.TrimSpace(fmt.Sprintf("[gate-override] %s — justification: %s\n%s", gateErr.Error(), strings.TrimSpace(req.OverrideJustification), req.Notes))
	}
	exp.PromotionDecisionID = decisionID

	if err := h.experiments.Update(r.Context(), eid, exp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	updated, _ := h.experiments.Get(r.Context(), eid)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.experimentToResponse(r, *updated, true))
}

// validateRecommendationEvidence makes the frozen controlled-lane contract a
// real gate. Observational serves, terminal statuses, and audit prose are
// purposefully absent from this calculation.
func (h *ExperimentHandlers) validateRecommendationEvidence(ctx context.Context, exp *store.Experiment, winner string) error {
	assignments, assignmentsOK := h.experiments.(store.ExperimentAssignmentStore)
	exposures, exposuresOK := h.experiments.(store.ExperimentExposureStore)
	if !assignmentsOK || !exposuresOK {
		return fmt.Errorf("durable controlled evidence store is unavailable")
	}
	assignmentRows, err := assignments.ListAssignments(ctx, exp.ID)
	if err != nil {
		return fmt.Errorf("read controlled assignments: %w", err)
	}
	exposureRows, err := exposures.ListExposures(ctx, exp.ID)
	if err != nil {
		return fmt.Errorf("read controlled exposures: %w", err)
	}
	outcomes, err := h.experiments.ListOutcomes(ctx, exp.ID)
	if err != nil {
		return fmt.Errorf("read controlled outcomes: %w", err)
	}
	report := buildControlledReport(exp, assignmentRows, exposureRows, outcomes)
	if report.Assignments == 0 || report.OutcomeCompleteness < exp.Protocol.OutcomeCompletenessThreshold {
		return fmt.Errorf("outcome completeness %.3f does not meet frozen threshold %.3f", report.OutcomeCompleteness, exp.Protocol.OutcomeCompletenessThreshold)
	}
	var winnerArm *ControlledArmReport
	for i := range report.Arms {
		if report.Arms[i].VariantID == winner {
			winnerArm = &report.Arms[i]
			break
		}
	}
	if winnerArm == nil || winnerArm.Eligible == 0 || winnerArm.PosteriorMean == nil {
		return fmt.Errorf("winner has no eligible controlled evaluator evidence")
	}
	if winner != store.ControlVariantID && (winnerArm.EffectVsControl == nil || *winnerArm.EffectVsControl < exp.Protocol.EffectThreshold) {
		return fmt.Errorf("winner effect does not meet frozen threshold %.3f", exp.Protocol.EffectThreshold)
	}
	if winner != store.ControlVariantID {
		if winnerArm.ProbBeatsControl == nil || *winnerArm.ProbBeatsControl < exp.Protocol.ProbabilityThreshold {
			observed := 0.0
			if winnerArm.ProbBeatsControl != nil {
				observed = *winnerArm.ProbBeatsControl
			}
			return fmt.Errorf("winner posterior probability of beating control %.3f does not meet frozen threshold %.3f", observed, exp.Protocol.ProbabilityThreshold)
		}
		if err := validateCostNonInferiority(exp, outcomes, winner); err != nil {
			return err
		}
	}
	return nil
}

// promotionEvidenceSummary renders the evidence a reviewer needs directly into
// the promotion decision, so accepting it never requires re-running the report
// command. Best-effort: an unreadable lane degrades to the pointer sentence in
// the base rationale rather than blocking the conclude.
func (h *ExperimentHandlers) promotionEvidenceSummary(ctx context.Context, exp *store.Experiment, winner string) string {
	assignments, assignmentsOK := h.experiments.(store.ExperimentAssignmentStore)
	exposures, exposuresOK := h.experiments.(store.ExperimentExposureStore)
	if !assignmentsOK || !exposuresOK {
		return ""
	}
	assignmentRows, err := assignments.ListAssignments(ctx, exp.ID)
	if err != nil {
		return ""
	}
	exposureRows, err := exposures.ListExposures(ctx, exp.ID)
	if err != nil {
		return ""
	}
	outcomes, err := h.experiments.ListOutcomes(ctx, exp.ID)
	if err != nil {
		return ""
	}
	report := buildControlledReport(exp, assignmentRows, exposureRows, outcomes)
	sums := map[string]float64{}
	counts := map[string]int{}
	for _, o := range outcomes {
		var payload outcomePayload
		if err := json.Unmarshal(o.Data, &payload); err != nil || payload.TokensUsed == nil {
			continue
		}
		sums[o.VariantID] += *payload.TokensUsed
		counts[o.VariantID]++
	}

	var winnerRow, controlRow *ControlledArmReport
	for i := range report.Arms {
		if report.Arms[i].VariantID == winner {
			winnerRow = &report.Arms[i]
		}
		if report.Arms[i].VariantID == store.ControlVariantID {
			controlRow = &report.Arms[i]
		}
	}

	var b strings.Builder
	// Operator-legibility contract (docs/agent-system/DECISIONS.md § Operator
	// legibility): plain summary before the metric block.
	if winnerRow != nil {
		if winner == store.ControlVariantID {
			fmt.Fprintf(&b, "In plain terms: no variant beat the current skill content — control stays (%d/%d controlled runs succeeded).\n", winnerRow.Successes, winnerRow.Eligible)
		} else {
			fmt.Fprintf(&b, "In plain terms: the winning variant succeeded in %d/%d controlled runs", winnerRow.Successes, winnerRow.Eligible)
			if controlRow != nil {
				fmt.Fprintf(&b, " vs the current content's %d/%d", controlRow.Successes, controlRow.Eligible)
			}
			if winnerRow.ProbBeatsControl != nil {
				fmt.Fprintf(&b, "; the chance it is genuinely better (not luck) is %.0f%%", *winnerRow.ProbBeatsControl*100)
			}
			if counts[winner] > 0 && counts[store.ControlVariantID] > 0 {
				winnerMean := sums[winner] / float64(counts[winner])
				controlMean := sums[store.ControlVariantID] / float64(counts[store.ControlVariantID])
				if controlMean > 0 {
					fmt.Fprintf(&b, "; it used %.2fx the tokens of the current content (limit %.2fx)", winnerMean/controlMean, exp.Protocol.CostNonInferiorityRatio)
				}
			}
			b.WriteString(".\n")
		}
	}
	fmt.Fprintf(&b, "Controlled lane: assignments=%d eligible=%d completeness=%.3f (threshold %.3f).",
		report.Assignments, report.EligibleAssignments, report.OutcomeCompleteness, exp.Protocol.OutcomeCompletenessThreshold)
	for _, arm := range report.Arms {
		label := arm.VariantID
		if arm.VariantID == winner {
			label += " (winner)"
		}
		fmt.Fprintf(&b, "\n- %s: eligible=%d successes=%d", label, arm.Eligible, arm.Successes)
		if arm.PosteriorMean != nil {
			fmt.Fprintf(&b, " posterior=%.3f", *arm.PosteriorMean)
		}
		if arm.EffectVsControl != nil {
			fmt.Fprintf(&b, " effect=%+.3f (threshold %.3f)", *arm.EffectVsControl, exp.Protocol.EffectThreshold)
		}
		if arm.ProbBeatsControl != nil {
			fmt.Fprintf(&b, " P(beats control)=%.3f (threshold %.3f)", *arm.ProbBeatsControl, exp.Protocol.ProbabilityThreshold)
		}
	}
	if counts[winner] > 0 && counts[store.ControlVariantID] > 0 {
		winnerMean := sums[winner] / float64(counts[winner])
		controlMean := sums[store.ControlVariantID] / float64(counts[store.ControlVariantID])
		fmt.Fprintf(&b, "\nGuardrail lane: mean tokens winner=%.0f control=%.0f", winnerMean, controlMean)
		if controlMean > 0 {
			fmt.Fprintf(&b, " ratio=%.2f (limit %.2fx)", winnerMean/controlMean, exp.Protocol.CostNonInferiorityRatio)
		}
		b.WriteString(".")
	}
	if serves, err := h.experiments.ListServes(ctx, exp.ID); err == nil {
		fmt.Fprintf(&b, "\nObservational: serves=%d outcomes=%d.", len(serves), len(outcomes))
	}
	return b.String()
}

// validateCostNonInferiority enforces the frozen equal-budget rule from the
// guardrail lane: a winner whose mean token cost exceeds the pre-registered
// ratio of control's is not "better", it is more expensive. Token data is
// required — a comparison at unknown budget is not a comparison.
func validateCostNonInferiority(exp *store.Experiment, outcomes []store.ExperimentOutcome, winner string) error {
	sums := map[string]float64{}
	counts := map[string]int{}
	for _, o := range outcomes {
		var payload outcomePayload
		if err := json.Unmarshal(o.Data, &payload); err != nil || payload.TokensUsed == nil {
			continue
		}
		sums[o.VariantID] += *payload.TokensUsed
		counts[o.VariantID]++
	}
	if counts[winner] == 0 || counts[store.ControlVariantID] == 0 {
		return fmt.Errorf("cost non-inferiority cannot be verified: token guardrail data is missing for %q or control", winner)
	}
	winnerMean := sums[winner] / float64(counts[winner])
	controlMean := sums[store.ControlVariantID] / float64(counts[store.ControlVariantID])
	if controlMean > 0 && winnerMean > exp.Protocol.CostNonInferiorityRatio*controlMean {
		return fmt.Errorf("winner mean token cost %.0f exceeds %.2fx control mean %.0f", winnerMean, exp.Protocol.CostNonInferiorityRatio, controlMean)
	}
	return nil
}

func (h *ExperimentHandlers) RecordAuditReceipt(w http.ResponseWriter, r *http.Request) {
	eid := mux.Vars(r)["eid"]
	exp, err := h.experiments.Get(r.Context(), eid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if exp.Status != store.ExperimentStatusRunning {
		http.Error(w, "audit receipt requires a running experiment", http.StatusConflict)
		return
	}
	if err := h.requireReceiptSigner(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	audits, ok := h.experiments.(store.ExperimentAuditStore)
	if !ok {
		http.Error(w, "durable audit store is unavailable", http.StatusServiceUnavailable)
		return
	}
	var req RecordAuditReceiptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.SampledAssignmentIDs) == 0 || req.FindingsHash == "" || req.ChallengeState == "" || req.IdempotencyKey == "" || req.AnomalyCount < 0 || req.GamingCount < 0 {
		http.Error(w, "audit receipt requires samples, findings hash, challenge state, nonnegative counts, and idempotency key", http.StatusBadRequest)
		return
	}
	receipt := store.ExperimentAuditReceipt{ExperimentID: eid, ProtocolHash: exp.Protocol.ProtocolHash, SampledAssignmentIDs: append([]string(nil), req.SampledAssignmentIDs...), FindingsHash: req.FindingsHash, ChallengeState: req.ChallengeState, AnomalyCount: req.AnomalyCount, GamingCount: req.GamingCount, CompletedAt: time.Now().UTC().Format(time.RFC3339), IdempotencyKey: req.IdempotencyKey}
	envelope, err := h.signAuditReceipt(r.Context(), &receipt)
	if err != nil {
		http.Error(w, "sign audit receipt: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	receipt.SignatureEnvelope, _ = json.Marshal(envelope)
	if err := audits.RecordAuditReceipt(r.Context(), receipt); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(receipt)
}

func canonicalAuditReceipt(receipt *store.ExperimentAuditReceipt) ([]byte, error) {
	samples := append([]string(nil), receipt.SampledAssignmentIDs...)
	sort.Strings(samples)
	return json.Marshal(struct {
		ExperimentID, ProtocolHash, FindingsHash, ChallengeState, CompletedAt, IdempotencyKey string
		Samples                                                                               []string
		AnomalyCount, GamingCount                                                             int
	}{receipt.ExperimentID, receipt.ProtocolHash, receipt.FindingsHash, receipt.ChallengeState, receipt.CompletedAt, receipt.IdempotencyKey, samples, receipt.AnomalyCount, receipt.GamingCount})
}

func (h *ExperimentHandlers) signAuditReceipt(ctx context.Context, receipt *store.ExperimentAuditReceipt) (receiptsigning.SignatureEnvelope, error) {
	if err := h.requireReceiptSigner(ctx); err != nil {
		return receiptsigning.SignatureEnvelope{}, err
	}
	canonical, err := canonicalAuditReceipt(receipt)
	if err != nil {
		return receiptsigning.SignatureEnvelope{}, err
	}
	return h.receiptSigner.Sign(ctx, receiptsigning.PurposeExperimentAuditReceipt, canonical)
}

func (h *ExperimentHandlers) verifyAuditReceipt(ctx context.Context, receipt *store.ExperimentAuditReceipt) bool {
	if receipt == nil || len(receipt.SignatureEnvelope) == 0 {
		return false
	}
	var envelope receiptsigning.SignatureEnvelope
	if err := json.Unmarshal(receipt.SignatureEnvelope, &envelope); err != nil {
		return false
	}
	canonical, err := canonicalAuditReceipt(receipt)
	if err != nil {
		return false
	}
	if err := h.requireReceiptSigner(ctx); err != nil {
		return false
	}
	return h.receiptSigner.Verify(ctx, envelope, canonical) == nil
}

// RecordHoldoutReceipt handles POST /experiments/{eid}/holdout-receipt. The
// holdout finding is sealed after the experiment recommendation, so it cannot
// be silently swapped between operator review and promotion.
func (h *ExperimentHandlers) RecordHoldoutReceipt(w http.ResponseWriter, r *http.Request) {
	eid := mux.Vars(r)["eid"]
	exp, err := h.experiments.Get(r.Context(), eid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if exp.Status != store.ExperimentStatusConcluded || exp.PromotionDecisionID == "" {
		http.Error(w, "holdout receipt requires a concluded experiment with a published recommendation", http.StatusConflict)
		return
	}
	if err := h.requireReceiptSigner(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	var req RecordHoldoutReceiptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.FindingsHash == "" || req.IdempotencyKey == "" {
		http.Error(w, "findingsHash and idempotencyKey are required", http.StatusBadRequest)
		return
	}
	// A completed receipt is immutable. Exact retries are safe; any other
	// attempt is a conflict because a holdout is a one-shot confirmation.
	if exp.HoldoutCompletedAt != "" {
		if exp.HoldoutFindingsHash == req.FindingsHash {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(h.experimentToResponse(r, *exp, true))
			return
		}
		http.Error(w, "holdout receipt is already recorded", http.StatusConflict)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	exp.HoldoutFindingsHash = req.FindingsHash
	exp.HoldoutCompletedAt = now
	exp.HoldoutIdempotencyKey = req.IdempotencyKey
	envelope, err := h.signHoldoutReceipt(r.Context(), exp, req.IdempotencyKey)
	if err != nil {
		http.Error(w, "sign holdout receipt: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	exp.HoldoutSignatureEnvelope, _ = json.Marshal(envelope)
	if err := h.experiments.Update(r.Context(), eid, exp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	updated, _ := h.experiments.Get(r.Context(), eid)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.experimentToResponse(r, *updated, true))
}

func canonicalHoldoutReceipt(exp *store.Experiment, idempotencyKey string) ([]byte, error) {
	return json.Marshal(struct {
		ExperimentID, ProtocolHash, DecisionID, FindingsHash, CompletedAt, IdempotencyKey string
	}{exp.ID, exp.Protocol.ProtocolHash, exp.PromotionDecisionID, exp.HoldoutFindingsHash, exp.HoldoutCompletedAt, idempotencyKey})
}

func (h *ExperimentHandlers) signHoldoutReceipt(ctx context.Context, exp *store.Experiment, idempotencyKey string) (receiptsigning.SignatureEnvelope, error) {
	if err := h.requireReceiptSigner(ctx); err != nil {
		return receiptsigning.SignatureEnvelope{}, err
	}
	canonical, err := canonicalHoldoutReceipt(exp, idempotencyKey)
	if err != nil {
		return receiptsigning.SignatureEnvelope{}, err
	}
	return h.receiptSigner.Sign(ctx, receiptsigning.PurposeExperimentHoldoutReceipt, canonical)
}

// PromoteExperiment handles POST /experiments/{eid}/promote. It is the only
// path that can apply a winner to SKILL.md and requires both sealed holdout
// evidence and the exact decision accepted by the human operator.
func (h *ExperimentHandlers) PromoteExperiment(w http.ResponseWriter, r *http.Request) {
	eid := mux.Vars(r)["eid"]
	exp, err := h.experiments.Get(r.Context(), eid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if exp.Status != store.ExperimentStatusConcluded || exp.WinnerVariantID == nil || exp.PromotionDecisionID == "" {
		http.Error(w, "promotion requires a concluded recommendation", http.StatusConflict)
		return
	}
	if exp.PromotedAt != "" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(h.experimentToResponse(r, *exp, true))
		return
	}
	if err := h.requireReceiptSigner(r.Context()); err != nil || exp.HoldoutFindingsHash == "" || exp.HoldoutCompletedAt == "" || len(exp.HoldoutSignatureEnvelope) == 0 {
		http.Error(w, "a signed holdout receipt is required before promotion", http.StatusConflict)
		return
	}
	var req PromoteExperimentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.DecisionID == "" || req.DecisionID != exp.PromotionDecisionID {
		http.Error(w, "decisionId must match the experiment's published promotion decision", http.StatusForbidden)
		return
	}
	var envelope receiptsigning.SignatureEnvelope
	canonical, canonicalErr := canonicalHoldoutReceipt(exp, exp.HoldoutIdempotencyKey)
	if json.Unmarshal(exp.HoldoutSignatureEnvelope, &envelope) != nil || canonicalErr != nil || h.receiptSigner.Verify(r.Context(), envelope, canonical) != nil {
		http.Error(w, "holdout receipt signature is invalid", http.StatusConflict)
		return
	}
	reader, ok := h.decisions.(decisionReader)
	if !ok {
		http.Error(w, "durable decision reader is unavailable", http.StatusServiceUnavailable)
		return
	}
	entries, _, err := reader.GetDecisions(r.Context(), "meta-optimization", "skill-experiment-promotion", store.DecisionStatusAccepted, 0)
	if err != nil {
		http.Error(w, "read operator decision: "+err.Error(), http.StatusInternalServerError)
		return
	}
	accepted := false
	for _, entry := range entries {
		if entry.ID == exp.PromotionDecisionID {
			accepted = true
			break
		}
	}
	if !accepted {
		http.Error(w, "the published promotion decision has not been accepted by the operator", http.StatusForbidden)
		return
	}
	if *exp.WinnerVariantID != store.ControlVariantID {
		_, content, err := h.variants.GetWithContent(r.Context(), exp.SkillID, *exp.WinnerVariantID)
		if err != nil {
			http.Error(w, "read winning variant: "+err.Error(), http.StatusConflict)
			return
		}
		skill, err := h.skills.Get(r.Context(), exp.SkillID)
		if err != nil {
			http.Error(w, "read skill for promotion: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := h.skills.Update(r.Context(), exp.SkillID, skill, &content); err != nil {
			http.Error(w, "apply accepted promotion: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	exp.PromotedAt = time.Now().UTC().Format(time.RFC3339)
	if err := h.experiments.Update(r.Context(), eid, exp); err != nil {
		http.Error(w, "record promotion: "+err.Error(), http.StatusInternalServerError)
		return
	}
	updated, _ := h.experiments.Get(r.Context(), eid)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.experimentToResponse(r, *updated, true))
}

func (h *ExperimentHandlers) requireReceiptSigner(ctx context.Context) error {
	_, err := receiptsigning.RequireHealthy(ctx, h.receiptSigner, h.productionReceiptSigning)
	return err
}

// RecordOutcome handles POST /experiments/{eid}/outcomes
func (h *ExperimentHandlers) RecordOutcome(w http.ResponseWriter, r *http.Request) {
	eid := mux.Vars(r)["eid"]

	exp, err := h.experiments.Get(r.Context(), eid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if exp.Status != store.ExperimentStatusRunning {
		http.Error(w, "can only record outcomes for running experiments", http.StatusConflict)
		return
	}

	var req RecordOutcomeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.VariantID == "" || req.Source == "" {
		http.Error(w, "variantId and source are required", http.StatusBadRequest)
		return
	}
	if req.Controlled != nil {
		if err := validateControlledOutcome(req.Controlled); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	outcome := store.ExperimentOutcome{
		IdempotencyKey: req.IdempotencyKey,
		VariantID:      req.VariantID,
		Source:         req.Source,
		SchemaVersion:  req.SchemaVersion,
		RecordedAt:     time.Now().UTC().Format(time.RFC3339),
		Data:           req.Data,
		Controlled:     req.Controlled,
	}

	if err := h.experiments.RecordOutcome(r.Context(), eid, outcome); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func validateControlledOutcome(c *store.ControlledExperimentOutcome) error {
	if c.AssignmentID == "" || c.ExecutionID == "" || c.EvaluatorAttemptID == "" || c.EvaluatorRunID == "" || c.OutcomeStatus == "" || c.RubricHash == "" || c.EvaluatorPromptHash == "" || c.StructuredSchemaHash == "" {
		return fmt.Errorf("controlled outcome requires assignment, evaluator, status, and provenance fields")
	}
	if c.OutcomeStatus != "complete" && c.OutcomeStatus != "incomplete" {
		return fmt.Errorf("controlled outcome status must be complete or incomplete")
	}
	if c.OutcomeStatus == "complete" && (c.Verdict == "" || c.Success == nil) {
		return fmt.Errorf("complete controlled outcome requires verdict and success mapping")
	}
	return nil
}

// AssignExperiment handles POST /experiments/{eid}/assignments. Unlike a
// normal skill read, this is the only treatment-selection path for workflows:
// it atomically preserves one prompt snapshot per dispatch idempotency key.
func (h *ExperimentHandlers) AssignExperiment(w http.ResponseWriter, r *http.Request) {
	eid := mux.Vars(r)["eid"]
	exp, err := h.experiments.Get(r.Context(), eid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if exp.Status != store.ExperimentStatusRunning {
		http.Error(w, "can only assign running experiments", http.StatusConflict)
		return
	}
	assignments, ok := h.experiments.(store.ExperimentAssignmentStore)
	if !ok {
		http.Error(w, "durable experiment assignment store is unavailable", http.StatusServiceUnavailable)
		return
	}
	var req AssignExperimentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.ExecutionID == "" || req.NodeID == "" || req.AttemptKey == "" || req.IdempotencyKey == "" {
		http.Error(w, "executionId, nodeId, attemptKey, and idempotencyKey are required", http.StatusBadRequest)
		return
	}
	if existing, err := assignments.GetAssignment(r.Context(), eid, req.IdempotencyKey); err == nil {
		h.writeAssignment(w, existing)
		return
	}
	variantID := assignmentVariant(exp.Arms, req.IdempotencyKey)
	var content string
	if variantID == store.ControlVariantID {
		_, content, err = h.skills.GetWithContent(r.Context(), exp.SkillID)
	} else {
		_, content, err = h.variants.GetWithContent(r.Context(), exp.SkillID, variantID)
	}
	if err != nil {
		http.Error(w, "read assignment content: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Variables) > 0 {
		content = SubstituteVariables(content, req.Variables)
	}
	a := store.ExperimentAssignment{ExperimentID: eid, SkillID: exp.SkillID, VariantID: variantID, ExecutionID: req.ExecutionID, NodeID: req.NodeID, AttemptKey: req.AttemptKey, IdempotencyKey: req.IdempotencyKey, Content: content, ContentHash: contentSHA256(content)}
	if err := assignments.CreateAssignment(r.Context(), a); err != nil {
		// A concurrent retry may have won; its snapshot is authoritative.
		if existing, getErr := assignments.GetAssignment(r.Context(), eid, req.IdempotencyKey); getErr == nil {
			h.writeAssignment(w, existing)
			return
		}
		http.Error(w, "record assignment: "+err.Error(), http.StatusInternalServerError)
		return
	}
	stored, err := assignments.GetAssignment(r.Context(), eid, req.IdempotencyKey)
	if err != nil {
		http.Error(w, "assignment recorded but unavailable: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.writeAssignment(w, stored)
}

func (h *ExperimentHandlers) writeAssignment(w http.ResponseWriter, a *store.ExperimentAssignment) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ExperimentAssignmentResponse{ExperimentID: a.ExperimentID, SkillID: a.SkillID, VariantID: a.VariantID, Content: a.Content, ContentHash: a.ContentHash, AssignedAt: a.AssignedAt})
}

func assignmentVariant(arms []store.ExperimentArm, key string) string {
	// A hash makes the choice stable even if a caller reaches us after a crash;
	// the persisted receipt remains the authority once it is written.
	digest := sha256.Sum256([]byte(key))
	unit := float64(uint64(digest[0])<<56|uint64(digest[1])<<48|uint64(digest[2])<<40|uint64(digest[3])<<32|uint64(digest[4])<<24|uint64(digest[5])<<16|uint64(digest[6])<<8|uint64(digest[7])) / float64(^uint64(0))
	acc := 0.0
	for _, arm := range arms {
		acc += arm.Weight
		if unit < acc {
			return arm.VariantID
		}
	}
	return arms[len(arms)-1].VariantID
}

// ListOutcomes handles GET /experiments/{eid}/outcomes
func (h *ExperimentHandlers) ListOutcomes(w http.ResponseWriter, r *http.Request) {
	eid := mux.Vars(r)["eid"]

	outcomes, err := h.experiments.ListOutcomes(r.Context(), eid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	resp := make([]ExperimentOutcomeResponse, 0, len(outcomes))
	for _, o := range outcomes {
		resp = append(resp, ExperimentOutcomeResponse{
			IdempotencyKey: o.IdempotencyKey,
			VariantID:      o.VariantID,
			Source:         o.Source,
			SchemaVersion:  o.SchemaVersion,
			RecordedAt:     o.RecordedAt,
			Data:           o.Data,
			Controlled:     o.Controlled,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// GetExperimentReport handles GET /experiments/{eid}/report
func (h *ExperimentHandlers) GetExperimentReport(w http.ResponseWriter, r *http.Request) {
	eid := mux.Vars(r)["eid"]

	exp, err := h.experiments.Get(r.Context(), eid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	serves, err := h.experiments.ListServes(r.Context(), eid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	outcomes, err := h.experiments.ListOutcomes(r.Context(), eid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	report := buildExperimentReport(exp, serves, outcomes)
	if assignments, ok := h.experiments.(store.ExperimentAssignmentStore); ok {
		if exposures, exposuresOK := h.experiments.(store.ExperimentExposureStore); exposuresOK {
			assigned, assignmentErr := assignments.ListAssignments(r.Context(), eid)
			exposureRows, exposureErr := exposures.ListExposures(r.Context(), eid)
			if assignmentErr != nil || exposureErr != nil {
				http.Error(w, "read controlled evidence", http.StatusInternalServerError)
				return
			}
			report.Controlled = buildControlledReport(exp, assigned, exposureRows, outcomes)
		}
	}
	for i, arm := range report.Arms {
		name := arm.VariantID
		if arm.VariantID == store.ControlVariantID {
			name = "control (original)"
		} else if v, err := h.variants.Get(r.Context(), exp.SkillID, arm.VariantID); err == nil {
			name = v.Name
		}
		report.Arms[i].VariantName = name
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}

// experimentToResponse converts a store experiment to API response.
func (h *ExperimentHandlers) experimentToResponse(r *http.Request, exp store.Experiment, includeCounts bool) ExperimentResponse {
	arms := make([]ExperimentArmResponse, len(exp.Arms))
	for i, a := range exp.Arms {
		name := a.VariantID
		if a.VariantID == store.ControlVariantID {
			name = "control (original)"
		} else if v, err := h.variants.Get(r.Context(), exp.SkillID, a.VariantID); err == nil {
			name = v.Name
		}
		arms[i] = ExperimentArmResponse{
			VariantID:   a.VariantID,
			VariantName: name,
			Weight:      a.Weight,
		}
	}

	resp := ExperimentResponse{
		ID:                  exp.ID,
		SkillID:             exp.SkillID,
		Name:                exp.Name,
		Hypothesis:          exp.Hypothesis,
		Protocol:            exp.Protocol,
		Status:              exp.Status,
		Arms:                arms,
		StartedAt:           exp.StartedAt,
		ConcludedAt:         exp.ConcludedAt,
		WinnerVariantID:     exp.WinnerVariantID,
		PromotionDecisionID: exp.PromotionDecisionID,
		HoldoutFindingsHash: exp.HoldoutFindingsHash,
		HoldoutCompletedAt:  exp.HoldoutCompletedAt,
		PromotedAt:          exp.PromotedAt,
		Notes:               exp.Notes,
		CreatedAt:           exp.CreatedAt,
		UpdatedAt:           exp.UpdatedAt,
		Revision:            exp.Revision,
	}

	if includeCounts {
		if counts, err := h.experiments.CountOutcomesByVariant(r.Context(), exp.ID); err == nil {
			resp.OutcomeCounts = counts
		}
	}

	return resp
}

// validateWeights checks that arm weights sum to approximately 1.0.
func validateWeights(arms []ExperimentArmInput) error {
	var sum float64
	for _, a := range arms {
		if a.Weight <= 0 || a.Weight > 1 {
			return experimentError("each weight must be in (0, 1], got %f for %s", a.Weight, a.VariantID)
		}
		sum += a.Weight
	}
	if math.Abs(sum-1.0) > 0.01 {
		return experimentError("arm weights must sum to 1.0 (got %f)", sum)
	}
	return nil
}

func experimentError(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

// normalizeProtocol fills the pre-registered statistical gates with their
// doctrine defaults when the author left them unset. It runs before the
// protocol is validated and hashed at start, so the effective thresholds are
// always part of the frozen contract.
func normalizeProtocol(p *store.ExperimentProtocol) {
	if p.ProbabilityThreshold == 0 {
		p.ProbabilityThreshold = 0.95
	}
	if p.CostNonInferiorityRatio == 0 {
		p.CostNonInferiorityRatio = 1.10
	}
}

func validateProtocol(p store.ExperimentProtocol) error {
	if p.Population == "" || p.RandomizationUnit != "workflow-node-per-execution" || p.PrimaryMetric == "" || p.EffectThreshold < 0 || p.ExposurePolicy == "" || p.OutcomeCompletenessThreshold <= 0 || p.OutcomeCompletenessThreshold > 1 || p.Budget == "" || p.StoppingRule == "" || !p.HoldoutRequired || p.HoldoutPopulationHash == "" || p.PromotionAuthority != "operator" || p.EvaluatorRubricHash == "" || p.EvaluatorAuthor == "" {
		return fmt.Errorf("protocol requires population, workflow-node-per-execution randomization, primary metric, non-negative effect threshold, exposure policy, completeness threshold in (0,1], budget, stopping rule, holdout requirement and population hash, operator promotion authority, evaluator rubric hash, and evaluator author")
	}
	if p.ProbabilityThreshold < 0.5 || p.ProbabilityThreshold >= 1 {
		return fmt.Errorf("protocol probability threshold must be in [0.5, 1)")
	}
	if p.CostNonInferiorityRatio < 1 {
		return fmt.Errorf("protocol cost non-inferiority ratio must be >= 1")
	}
	return nil
}

func protocolHash(p store.ExperimentProtocol) string {
	p.ProtocolHash = ""
	b, _ := json.Marshal(p)
	digest := sha256.Sum256(b)
	return fmt.Sprintf("sha256:%x", digest[:])
}
