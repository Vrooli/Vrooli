// Declarative agent-operations action handlers for the backlog-item workflow.
//
// These are the REAL domain handlers the operation runner's dispatcher fires
// when the backlog-item transition policy selects an action. Each is a thin,
// idempotent wrapper over the same domain writes the legacy HTTP flow performs
// (round-file persistence, plan-ref binding, terminal status), decoupled from
// net/http so the generic runner can own the coordination while the backlog
// package keeps owning the domain mutation and its invariants.
//
// The runner never branches on a named action; it only dispatches from data.
// So this file is the single place the closed action vocabulary
// (agentops.AllActionNames, as routed by policy/backlog-item-default.json) is
// bound to backlog domain code. Handlers not routed by that policy are left as
// the registry's pre-registered no-ops.
package backlog

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/opsrunner"
	"swarm-manager/internal/pathredact"
	"swarm-manager/internal/workshop"
)

// OpsItemStore is the subset of Store the operation handlers need. *FileStore
// (and any Store) satisfies it.
type OpsItemStore interface {
	ItemDir(kind BacklogKind, name string) string
	LoadItem(kind BacklogKind, name string) (BacklogItem, error)
	SaveItem(item BacklogItem) error
}

// OpsHandlerDeps are the collaborators the backlog operation handlers need.
type OpsHandlerDeps struct {
	Store  OpsItemStore
	Now    func() time.Time
	Logger *slog.Logger
}

func (d OpsHandlerDeps) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
}

func (d OpsHandlerDeps) log() *slog.Logger {
	if d.Logger != nil {
		return d.Logger
	}
	return slog.Default()
}

// RegisterOpsHandlers binds the backlog domain handlers onto the runner's action
// registry for every action the backlog-item transition policy routes to. It is
// the composition seam production wiring calls after NewActionRegistry so the
// runner dispatches real domain writes; unrouted actions keep their no-op
// default. Registration panics on an unregistered name (a compiled invariant),
// which can only happen if the closed vocabulary drifts.
func RegisterOpsHandlers(reg *opsrunner.ActionRegistry, deps OpsHandlerDeps) {
	reg.Register(agentops.ActionCommitWorkshopRound, deps.commitWorkshopRound)
	reg.Register(agentops.ActionOpenReview, deps.openReview)
	reg.Register(agentops.ActionBindPlan, deps.bindPlan)
	reg.Register(agentops.ActionCompleteItem, deps.setStatus(StatusCompleted))
	reg.Register(agentops.ActionFailItem, deps.setStatus(StatusFailed))
	reg.Register(agentops.ActionRequestRevision, deps.setStatus(StatusBacklog))
	reg.Register(agentops.ActionEscalateNeedsAttention, deps.escalateNeedsAttention)
	// Clarification thread-commit handlers: a clarification operation's completed
	// round writes its assistant turn (and parsed impact) into the thread the
	// operator opened. start-clarification commits the first turn, resolve-
	// clarification the follow-up turns (see opshandlers_clarification.go).
	reg.Register(agentops.ActionStartClarification, deps.commitClarificationTurn)
	reg.Register(agentops.ActionResolveClarification, deps.commitClarificationTurn)
}

// splitItemRef parses a target ref ("kind/name") into its backlog kind and name.
func splitItemRef(id string) (BacklogKind, string, error) {
	parts := strings.SplitN(strings.TrimSpace(id), "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("backlog ops: target ref %q is not a kind/name ref", id)
	}
	return BacklogKind(parts[0]), parts[1], nil
}

// ---------------------------------------------------------------------------
// commit-workshop-round
// ---------------------------------------------------------------------------

// workshopRoundResult is the workshop-round operation's declared output: the
// enriched result the mode produces and the engine validates (per decision B),
// carrying everything the round file needs. The handler materializes the round
// file from it deterministically — it never re-derives content.
type workshopRoundResult struct {
	Handoff        json.RawMessage `json:"handoff"`
	Progress       string          `json:"progress"`
	Decisions      []decisionInput `json:"decisions"`
	SelfAssessment map[string]int  `json:"self_assessment"`
}

type decisionInput struct {
	ID      string        `json:"id"`
	Topic   string        `json:"topic"`
	Text    string        `json:"text"`
	Context string        `json:"context,omitempty"`
	Options []optionInput `json:"options,omitempty"`
}

type optionInput struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Rationale   string `json:"rationale"`
	Recommended bool   `json:"recommended,omitempty"`
}

// opProvenance is the operation-execution provenance sidecar written next to the
// round file, so a materialized artifact records exactly which operation
// execution (and workflow) produced it and doubles as the idempotency marker.
type opProvenance struct {
	Kind        string `json:"kind"`
	ExecutionID string `json:"execution_id"`
	WorkflowID  string `json:"workflow_instance_id"`
	Operation   string `json:"operation"`
	Outcome     string `json:"outcome"`
	Artifact    string `json:"artifact"`
	RecordedAt  string `json:"recorded_at"`
}

// commitWorkshopRound writes the next round-NNN.json for a continuing workshop
// from the validated enriched result, plus its operation-execution provenance
// sidecar. It is idempotent: a re-fire for the same execution (a crash-recovery
// retry) detects the existing provenance sidecar and is a no-op, so a round is
// never duplicated. Readiness COMPUTATION stays in the workshop service — this
// handler only records the round's self-assessment inputs verbatim.
func (d OpsHandlerDeps) commitWorkshopRound(_ context.Context, ac opsrunner.ActionContext) error {
	kind, name, err := splitItemRef(ac.Target.ID)
	if err != nil {
		return err
	}
	if len(ac.Result) == 0 {
		return fmt.Errorf("commit-workshop-round: no operation result to persist for %s", ac.Target.ID)
	}
	var res workshopRoundResult
	if err := json.Unmarshal(ac.Result, &res); err != nil {
		return fmt.Errorf("commit-workshop-round: decode result for %s: %w", ac.Target.ID, err)
	}

	itemDir := d.Store.ItemDir(kind, name)
	workshopDir := filepath.Join(itemDir, "workshop")

	// Idempotent short-circuit: if a provenance sidecar already records this
	// execution, the round was committed on a prior fire; do nothing.
	if committed, err := executionAlreadyCommitted(workshopDir, ac.ExecutionID); err != nil {
		return err
	} else if committed {
		return nil
	}

	_, roundCount, err := workshop.LoadLatestRound(itemDir)
	if err != nil {
		return fmt.Errorf("commit-workshop-round: count rounds for %s: %w", ac.Target.ID, err)
	}
	roundNum := roundCount + 1

	round := workshop.Round{
		RoundNum:    roundNum,
		GeneratedAt: d.now().UTC().Format(time.RFC3339),
		Mode:        "workshop",
		Readiness:   readinessFromAssessment(res.SelfAssessment),
		Items:       itemsFromDecisions(res.Decisions),
		PlanUpdates: planUpdatesFromHandoff(res.Handoff),
	}
	round.PendingSynthesis = workshop.NeedsSynthesis(&round)

	if err := os.MkdirAll(workshopDir, 0o750); err != nil {
		return fmt.Errorf("commit-workshop-round: create workshop dir: %w", err)
	}
	roundFile := fmt.Sprintf("round-%03d.json", roundNum)
	if err := writeJSONRedacted(filepath.Join(workshopDir, roundFile), round); err != nil {
		return fmt.Errorf("commit-workshop-round: write %s: %w", roundFile, err)
	}

	prov := opProvenance{
		Kind:        "agentops-artifact-provenance",
		ExecutionID: ac.ExecutionID,
		WorkflowID:  ac.Workflow.InstanceID,
		Operation:   string(ac.Operation),
		Outcome:     ac.Outcome,
		Artifact:    filepath.Join("workshop", roundFile),
		RecordedAt:  d.now().UTC().Format(time.RFC3339Nano),
	}
	// Named so it does NOT match the workshop round glob (round-*.json), which the
	// readiness/round loader scans — otherwise the sidecar would be counted as a
	// round and inflate the next round number.
	provFile := fmt.Sprintf("provenance-%03d.json", roundNum)
	if err := writeJSONRedacted(filepath.Join(workshopDir, provFile), prov); err != nil {
		return fmt.Errorf("commit-workshop-round: write provenance: %w", err)
	}
	d.log().Info("backlog ops: workshop round committed", "kind", kind, "name", name, "round", roundNum, "execution", ac.ExecutionID)
	return nil
}

// executionAlreadyCommitted reports whether a provenance sidecar under the
// workshop dir already records the given execution id.
func executionAlreadyCommitted(workshopDir, executionID string) (bool, error) {
	if strings.TrimSpace(executionID) == "" {
		return false, nil
	}
	entries, err := os.ReadDir(workshopDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "provenance-") || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(workshopDir, e.Name()))
		if err != nil {
			return false, err
		}
		var p opProvenance
		if err := json.Unmarshal(raw, &p); err != nil {
			continue // an unrelated sidecar shape is not a match
		}
		if p.ExecutionID == executionID {
			return true, nil
		}
	}
	return false, nil
}

func readinessFromAssessment(assessment map[string]int) map[string]int {
	out := make(map[string]int, len(workshop.ReadinessDimensions))
	for _, dim := range workshop.ReadinessDimensions {
		out[dim] = assessment[dim]
	}
	return out
}

func itemsFromDecisions(decisions []decisionInput) []workshop.Item {
	if len(decisions) == 0 {
		return nil
	}
	items := make([]workshop.Item, 0, len(decisions))
	for _, dec := range decisions {
		item := workshop.Item{
			ID:      dec.ID,
			Type:    "decision",
			Topic:   dec.Topic,
			Text:    dec.Text,
			Context: dec.Context,
		}
		for _, o := range dec.Options {
			item.Options = append(item.Options, workshop.Option{
				Key: o.Key, Label: o.Label, Rationale: o.Rationale, Recommended: o.Recommended,
			})
		}
		items = append(items, item)
	}
	return items
}

// planUpdatesFromHandoff surfaces the handoff summary as the round's plan_updates
// note when present, so the round file keeps carrying the human-readable
// progress the legacy round did.
func planUpdatesFromHandoff(handoff json.RawMessage) string {
	if len(handoff) == 0 {
		return ""
	}
	var h struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(handoff, &h); err != nil {
		return ""
	}
	return strings.TrimSpace(h.Summary)
}

// ---------------------------------------------------------------------------
// open-review (workshop-completion review opener)
// ---------------------------------------------------------------------------

// openReview opens the operator review gate for a COMPLETED workshop operation.
// It is the workshop-completion review opener (distinct from the execution-scoped
// review of Phase 6): it records the completed workshop handoff as a review-open
// artifact the operator inspects before accepting/failing. It is idempotent —
// the artifact is written to a stable path, so a re-fire overwrites in place.
func (d OpsHandlerDeps) openReview(_ context.Context, ac opsrunner.ActionContext) error {
	kind, name, err := splitItemRef(ac.Target.ID)
	if err != nil {
		return err
	}
	itemDir := d.Store.ItemDir(kind, name)
	workshopDir := filepath.Join(itemDir, "workshop")
	if err := os.MkdirAll(workshopDir, 0o750); err != nil {
		return fmt.Errorf("open-review: create workshop dir: %w", err)
	}
	artifact := map[string]any{
		"kind":                 "agentops-workshop-review-open",
		"execution_id":         ac.ExecutionID,
		"workflow_instance_id": ac.Workflow.InstanceID,
		"operation":            string(ac.Operation),
		"outcome":              ac.Outcome,
		"opened_at":            d.now().UTC().Format(time.RFC3339),
	}
	if len(ac.Result) > 0 {
		artifact["result"] = json.RawMessage(ac.Result)
	}
	if err := writeJSONRedacted(filepath.Join(workshopDir, "review-open.json"), artifact); err != nil {
		return fmt.Errorf("open-review: write review-open artifact: %w", err)
	}
	d.log().Info("backlog ops: workshop review opened", "kind", kind, "name", name, "execution", ac.ExecutionID)
	return nil
}

// ---------------------------------------------------------------------------
// bind-plan
// ---------------------------------------------------------------------------

// bindPlan atomically writes the canonical plan_ref onto the item's spec.json
// from the finalize operation's validated result. The item stays active (the
// policy's to_state is running) with its plan bound. It is idempotent — binding
// the same ref twice is a no-op-equivalent overwrite.
//
// It is FAIL-CLOSED: the plan_ref MUST come from the finalize result itself (the
// first-class top-level plan_ref field the workshop-finalize contract now
// declares, or — for back-compat — one nested inside the freeform handoff). A
// finalize completion that carries no valid canonical ref binds nothing and the
// item is left untouched (the handler errors, the dispatcher persists no state).
// The ref is NOT resolved against the existing item ref: an unbound item finishing
// finalize with no authored plan is an honest failure, not a re-bind of a stale
// ref. Existence-resolution against plan-manager is intentionally not done here —
// the finalize agent is the authoring authority that just created the plan, and a
// network existence check in the completion handler would add a failure mode; the
// canonical shape is validated instead.
func (d OpsHandlerDeps) bindPlan(_ context.Context, ac opsrunner.ActionContext) error {
	kind, name, err := splitItemRef(ac.Target.ID)
	if err != nil {
		return err
	}
	ref, err := planRefFromResult(ac.Result)
	if err != nil {
		return fmt.Errorf("bind-plan: %w", err)
	}
	if ref == nil {
		return fmt.Errorf("bind-plan: finalize result carries no canonical plan_ref for %s (fail-closed; item left unbound)", ac.Target.ID)
	}
	if err := validatePlanRef(ref, PlanRefRoleExecutionSpec); err != nil {
		return fmt.Errorf("bind-plan: invalid plan_ref: %w", err)
	}
	// Load only after the ref is validated, so a missing/invalid ref never mutates
	// the item.
	item, err := d.Store.LoadItem(kind, name)
	if err != nil {
		return fmt.Errorf("bind-plan: load %s: %w", ac.Target.ID, err)
	}
	item.PlanRef = ref
	item.Updated = d.now().UTC().Format(time.RFC3339)
	if err := d.Store.SaveItem(item); err != nil {
		return fmt.Errorf("bind-plan: save %s: %w", ac.Target.ID, err)
	}
	d.log().Info("backlog ops: plan bound", "kind", kind, "name", name, "plan", firstNonBlank(ref.PlanID, ref.Slug))
	return nil
}

// planRefFromResult extracts a canonical plan_ref from a finalize result and ONLY
// from the result — it never falls back to an existing item ref. It reads the
// first-class top-level plan_ref field first, then (back-compat) a ref nested
// inside the freeform handoff object. A result with no ref returns (nil, nil), so
// bind-plan can fail closed on it. Reuses the same canonical parser+validator the
// legacy finalize path used, so both paths agree on what a bindable ref is.
func planRefFromResult(result json.RawMessage) (*PlanRef, error) {
	if len(result) == 0 {
		return nil, nil
	}
	if ref, err := finalizedPlanRefFromRound(string(result), nil); err == nil && ref != nil {
		return ref, nil
	}
	// Back-compat: a plan_ref nested inside the freeform handoff object.
	var envelope struct {
		Handoff json.RawMessage `json:"handoff"`
	}
	if err := json.Unmarshal(result, &envelope); err == nil && len(envelope.Handoff) > 0 {
		if ref, err := finalizedPlanRefFromRound(string(envelope.Handoff), nil); err == nil && ref != nil {
			return ref, nil
		}
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// terminal + needs-attention status handlers
// ---------------------------------------------------------------------------

// setStatus returns a handler that flips the item to a fixed terminal/coordination
// status. Setting the same status twice is idempotent.
func (d OpsHandlerDeps) setStatus(status BacklogStatus) opsrunner.ActionHandler {
	return func(_ context.Context, ac opsrunner.ActionContext) error {
		kind, name, err := splitItemRef(ac.Target.ID)
		if err != nil {
			return err
		}
		item, err := d.Store.LoadItem(kind, name)
		if err != nil {
			return fmt.Errorf("set-status %s: load %s: %w", status, ac.Target.ID, err)
		}
		if item.Status == status {
			return nil
		}
		item.Status = status
		item.Updated = d.now().UTC().Format(time.RFC3339)
		if err := d.Store.SaveItem(item); err != nil {
			return fmt.Errorf("set-status %s: save %s: %w", status, ac.Target.ID, err)
		}
		d.log().Info("backlog ops: status set", "kind", kind, "name", name, "status", status)
		return nil
	}
}

// escalateNeedsAttention parks the item for operator attention: it records a
// needs-attention marker (the blocking reason and the run that raised it) and
// flags the item as needing followup. It is idempotent — the marker is written
// to a stable path.
func (d OpsHandlerDeps) escalateNeedsAttention(_ context.Context, ac opsrunner.ActionContext) error {
	kind, name, err := splitItemRef(ac.Target.ID)
	if err != nil {
		return err
	}
	item, err := d.Store.LoadItem(kind, name)
	if err != nil {
		return fmt.Errorf("escalate-needs-attention: load %s: %w", ac.Target.ID, err)
	}
	itemDir := d.Store.ItemDir(kind, name)
	marker := map[string]any{
		"kind":                 "agentops-needs-attention",
		"execution_id":         ac.ExecutionID,
		"workflow_instance_id": ac.Workflow.InstanceID,
		"operation":            string(ac.Operation),
		"outcome":              ac.Outcome,
		"raised_at":            d.now().UTC().Format(time.RFC3339),
	}
	if len(ac.Result) > 0 {
		marker["result"] = json.RawMessage(ac.Result)
	}
	if err := os.MkdirAll(itemDir, 0o750); err != nil {
		return fmt.Errorf("escalate-needs-attention: create item dir: %w", err)
	}
	if err := writeJSONRedacted(filepath.Join(itemDir, "needs-attention.json"), marker); err != nil {
		return fmt.Errorf("escalate-needs-attention: write marker: %w", err)
	}
	if item.Status != StatusNeedsFollowup {
		item.Status = StatusNeedsFollowup
		item.Updated = d.now().UTC().Format(time.RFC3339)
		if err := d.Store.SaveItem(item); err != nil {
			return fmt.Errorf("escalate-needs-attention: save %s: %w", ac.Target.ID, err)
		}
	}
	d.log().Info("backlog ops: escalated needs-attention", "kind", kind, "name", name, "outcome", ac.Outcome)
	return nil
}

// writeJSONRedacted marshals v as indented JSON, applies artifact-path redaction
// (matching the legacy workshop-save write), and writes it 0600.
func writeJSONRedacted(path string, v any) error {
	encoded, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if redacted, changed := pathredact.NewForArtifactPath(path).RedactBytes(path, encoded); changed {
		encoded = redacted
	}
	return os.WriteFile(path, encoded, 0o600)
}
