package operatingmode

import (
	"context"
	"fmt"
	"strings"
)

// This file implements the target-adapter seam of the v2 unit-of-work
// decoupling (EXECUTION-MODES.md, D1). A mode declares a target kind; one
// adapter per kind supplies the target-specific parts of a run — instance
// resolution, typed capability values, and the exclusive-ownership key. The
// shared engine consumes adapters only through the generic RunContext; it
// never branches on the target kind.

// Stable prompt aliases for target inputs. Their logical ids and capability
// sources are authored in each mode input_contract.
const (
	ReadInitiativeName        = "INITIATIVE_NAME"
	ReadInitiativeTitle       = "INITIATIVE_TITLE"
	ReadInitiativeDescription = "INITIATIVE_DESCRIPTION"
	ReadAcceptanceCriteria    = "ACCEPTANCE_CRITERIA"
	ReadMemberItemsJSON       = "MEMBER_ITEMS_JSON"
	ReadPlanContextJSON       = "PLAN_CONTEXT_JSON"
	ReadPlanID                = "PLAN_ID"
	ReadPlanPath              = "PLAN_PATH"
)

// TargetInstance is the resolved unit of work one mode run operates on. It is
// plain data: adapters fill the fields for their kind and read them back when
// supplying reads; the generic engine only touches Kind, ID, Title, and
// Description.
type TargetInstance struct {
	Kind TargetKind
	// ID is the stable identity of the target instance: the initiative name,
	// the plan-manager execution id (or slug), or the plan-ref path. It is the
	// round-store scope id and the input to the adapter's ownership key.
	ID          string
	Title       string
	Description string
	// Initiative and Items are populated by the initiative adapter.
	Initiative InitiativeSnapshot
	Items      []RoundItem
	// Plan is the resolved plan execution context. The plan adapters populate
	// it from their unit of work; the initiative adapter populates it from the
	// initiative's bound plan_ref when the mode's target policy requires one.
	Plan *PlanExecutionContext
	// PlanPath is populated by the plan-ref adapter.
	PlanPath string
}

// TargetAdapter supplies the target-specific parts of a mode run for one
// target kind. Capability compatibility is declared once in
// InputProviderCapabilities; Values performs only behavioral projection from
// a resolved target instance.
type TargetAdapter interface {
	Kind() TargetKind
	// Resolve loads the target instance identified by ref (initiative name,
	// plan execution id/slug, or plan path).
	Resolve(ctx context.Context, s *Service, def Definition, phaseDef PhaseDefinition, ref string) (TargetInstance, error)
	// Values returns typed values keyed by target capability id. Its key set is
	// checked against InputProviderCapabilities in tests.
	Values(t TargetInstance) map[string]any
	// OwnershipKey is the exclusive-run lock identity for a target instance
	// id. Initiative targets keep the initiative name; plan targets get an
	// equivalent plan-scoped key so a plan run never requires an initiative.
	OwnershipKey(id string) string
}

var targetAdapters = map[TargetKind]TargetAdapter{
	TargetInitiative:      initiativeTargetAdapter{},
	TargetPlanManagerPlan: planManagerPlanTargetAdapter{},
	TargetPlanRef:         planRefTargetAdapter{},
}

// AdapterFor returns the target adapter for the given kind, or a typed error
// for an unknown/invalid kind.
func AdapterFor(kind TargetKind) (TargetAdapter, error) {
	adapter, ok := targetAdapters[kind]
	if !ok {
		return nil, fmt.Errorf("unknown operating-mode target kind %q (want one of %s|%s|%s)", kind, TargetPlanManagerPlan, TargetPlanRef, TargetInitiative)
	}
	return adapter, nil
}

// OwnershipKeyFor returns the exclusive-run lock identity for a target kind +
// instance id, without needing a resolved instance (round refresh/cancel paths
// derive it from the persisted round's target kind and scope id).
func OwnershipKeyFor(kind TargetKind, id string) (string, error) {
	adapter, err := AdapterFor(kind)
	if err != nil {
		return "", err
	}
	return adapter.OwnershipKey(id), nil
}

// --- initiative adapter ---

type initiativeTargetAdapter struct{}

func (initiativeTargetAdapter) Kind() TargetKind { return TargetInitiative }

func (initiativeTargetAdapter) Resolve(ctx context.Context, s *Service, def Definition, phaseDef PhaseDefinition, ref string) (TargetInstance, error) {
	init, err := s.initiatives.LoadInitiative(strings.TrimSpace(ref))
	if err != nil {
		return TargetInstance{}, err
	}
	plan, err := s.collectPlanContext(ctx, init, def, phaseDef)
	if err != nil {
		return TargetInstance{}, err
	}
	return TargetInstance{
		Kind:        TargetInitiative,
		ID:          init.Name,
		Title:       init.Title,
		Description: init.Description,
		Initiative:  init,
		Items:       s.collectItems(init.Items),
		Plan:        plan,
	}, nil
}

func (initiativeTargetAdapter) Values(t TargetInstance) map[string]any {
	return map[string]any{
		"target.initiative_name":        t.Initiative.Name,
		"target.initiative_title":       t.Initiative.Title,
		"target.initiative_description": t.Initiative.Description,
		"target.acceptance_criteria":    strings.Join(t.Initiative.AcceptanceCriteria, "\n"),
		"target.member_items":           append([]RoundItem{}, t.Items...),
		"target.plan_context":           targetPlanValue(t.Plan),
	}
}

func (initiativeTargetAdapter) OwnershipKey(id string) string { return id }

// --- plan-manager-plan adapter ---

type planManagerPlanTargetAdapter struct{}

func (planManagerPlanTargetAdapter) Kind() TargetKind { return TargetPlanManagerPlan }

func (planManagerPlanTargetAdapter) Resolve(ctx context.Context, s *Service, def Definition, phaseDef PhaseDefinition, ref string) (TargetInstance, error) {
	handle := strings.TrimSpace(ref)
	if handle == "" {
		return TargetInstance{}, fmt.Errorf("mode %q targets a plan-manager plan: a plan execution id or slug is required", def.Mode)
	}
	plan, err := s.resolvePlanManagerPlan(ctx, def, phaseDef, handle)
	if err != nil {
		return TargetInstance{}, err
	}
	id := firstNonEmpty(plan.ExecutionID, handle)
	return TargetInstance{
		Kind:  TargetPlanManagerPlan,
		ID:    id,
		Title: fmt.Sprintf("Plan %s", id),
		Plan:  plan,
	}, nil
}

func (planManagerPlanTargetAdapter) Values(t TargetInstance) map[string]any {
	return map[string]any{
		"target.plan_id":      t.ID,
		"target.plan_context": targetPlanValue(t.Plan),
	}
}

func (planManagerPlanTargetAdapter) OwnershipKey(id string) string {
	return "plan--" + sanitizeOwnershipToken(id)
}

// --- plan-ref adapter ---

type planRefTargetAdapter struct{}

func (planRefTargetAdapter) Kind() TargetKind { return TargetPlanRef }

func (planRefTargetAdapter) Resolve(_ context.Context, _ *Service, def Definition, _ PhaseDefinition, ref string) (TargetInstance, error) {
	path := strings.TrimSpace(ref)
	if path == "" {
		return TargetInstance{}, fmt.Errorf("mode %q targets a plan reference: a plan path is required", def.Mode)
	}
	return TargetInstance{
		Kind:     TargetPlanRef,
		ID:       path,
		Title:    fmt.Sprintf("Plan ref %s", path),
		PlanPath: path,
		Plan: &PlanExecutionContext{
			Required: true,
			Source:   planContextSourcePlanRef,
		},
	}, nil
}

func (planRefTargetAdapter) Values(t TargetInstance) map[string]any {
	return map[string]any{
		"target.plan_path":    t.PlanPath,
		"target.plan_context": targetPlanValue(t.Plan),
	}
}

func (planRefTargetAdapter) OwnershipKey(id string) string {
	return "plan-ref--" + sanitizeOwnershipToken(id)
}

func targetPlanValue(plan *PlanExecutionContext) any {
	if plan == nil {
		return nil
	}
	return plan
}

// ParseTargetOwnershipKey recognizes a non-initiative ownership key
// ("plan--<token>" / "plan-ref--<token>") and returns its target kind and
// sanitized instance token. ok=false for initiative keys (a bare initiative
// name).
func ParseTargetOwnershipKey(key string) (TargetKind, string, bool) {
	switch {
	case strings.HasPrefix(key, "plan-ref--"):
		return TargetPlanRef, strings.TrimPrefix(key, "plan-ref--"), true
	case strings.HasPrefix(key, "plan--"):
		return TargetPlanManagerPlan, strings.TrimPrefix(key, "plan--"), true
	default:
		return "", "", false
	}
}

// roundOwnershipKey derives the exclusive-run lock key from a persisted
// round's target kind and scope id, so refresh/cancel paths release the same
// lock the phase start acquired. Rounds persisted before the target substrate
// carry an unknown kind; their scope id (the initiative name) is the key.
func roundOwnershipKey(round RoundEnvelope) string {
	scopeID := strings.TrimSpace(round.ScopeID)
	if scopeID == "" {
		scopeID = strings.TrimSpace(round.InitiativeName)
	}
	if key, err := OwnershipKeyFor(TargetKind(round.ScopeKind), scopeID); err == nil {
		return key
	}
	return scopeID
}

// sanitizeOwnershipToken maps an arbitrary target instance id (a plan slug, an
// execution UUID, a repo-relative path) onto a filesystem-safe lock key token.
func sanitizeOwnershipToken(id string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(id) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return b.String()
}
