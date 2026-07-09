package planimport

import (
	"context"
	"fmt"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/identity"
	"swarm-manager/internal/operatingmode"
	"swarm-manager/internal/planclient"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// ImportedRef is a created backlog item the bridge reports back.
type ImportedRef struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Title  string `json:"title"`
	Action string `json:"action"`
}

// ImportedInitiative is the plan-bound initiative the bridge reports back.
type ImportedInitiative struct {
	Name   string `json:"name"`
	Title  string `json:"title"`
	Mode   string `json:"mode,omitempty"`
	Action string `json:"action"`
}

// BatchLander lands or links a translated plan batch and returns item refs.
// *backlog.Handler satisfies it via an adapter, so this package need not import
// backlog.
type BatchLander interface {
	LandBatch(ctx context.Context, payload BatchPayload, prov identity.Provenance) ([]ImportedRef, error)
}

// InitiativeLander lands the optional initiative container around imported
// phase items. It lives outside BatchLander because item-only imports remain
// the default API behavior.
type InitiativeLander interface {
	LandInitiative(ctx context.Context, spec InitiativeSpec, itemRefs []ImportedRef, prov identity.Provenance) (ImportedInitiative, error)
}

// Service fetches an authored plan and lands its phases as a provenance-stamped
// linear depends_on chain via the existing atomic batch-create.
type Service struct {
	fetcher    planclient.PlanReader
	lander     BatchLander
	initLander InitiativeLander
}

// NewService wires the plan fetcher and the batch lander.
func NewService(fetcher planclient.PlanReader, lander BatchLander, initLander InitiativeLander) *Service {
	return &Service{fetcher: fetcher, lander: lander, initLander: initLander}
}

// Request describes a Create-Work-From-Plan backend operation. PlanID preserves
// the original item-import contract; SourcePath/Markdown adopt external plans
// through plan-manager before binding work in swarm-manager.
type Request struct {
	PlanID     string
	SourcePath string
	Markdown   string
	Title      string
	Slug       string
	Container  ContainerSpec
}

type ContainerSpec struct {
	Type        string
	Name        string
	Title       string
	Description string
	Mode        string
}

// Result reports what an import produced.
type Result struct {
	Slug       string              `json:"slug"`
	PlanID     string              `json:"plan_id"`
	Container  string              `json:"container"`
	Items      []ImportedRef       `json:"items"`
	Initiative *ImportedInitiative `json:"initiative,omitempty"`
	Count      int                 `json:"count"`
	Created    int                 `json:"created"`
	Linked     int                 `json:"linked"`
	Updated    int                 `json:"updated"`
}

// PlanSummary is the small picker-facing shape used by the UI. Swarm-manager
// binds plans by reference; full structure stays owned by plan-manager.
type PlanSummary struct {
	ID         string `json:"id"`
	Slug       string `json:"slug"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	UpdatedAt  string `json:"updated_at,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	PhaseCount int    `json:"phase_count"`
}

// ListPlans returns canonical plan-manager plans for Create-Work-From-Plan.
func (s *Service) ListPlans(ctx context.Context) ([]PlanSummary, error) {
	plans, err := s.fetcher.ListPlans(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PlanSummary, 0, len(plans))
	for _, plan := range plans {
		if plan == nil {
			continue
		}
		out = append(out, PlanSummary{
			ID:         strings.TrimSpace(plan.GetId()),
			Slug:       strings.TrimSpace(plan.GetSlug()),
			Title:      strings.TrimSpace(plan.GetTitle()),
			Status:     strings.TrimPrefix(plan.GetStatus().String(), "PLAN_STATUS_"),
			UpdatedAt:  strings.TrimSpace(plan.GetUpdatedAt()),
			CreatedAt:  strings.TrimSpace(plan.GetCreatedAt()),
			PhaseCount: len(plan.GetPhases()),
		})
	}
	return out, nil
}

// Import fetches the plan, translates its phases, and lands them atomically.
func (s *Service) Import(ctx context.Context, req Request, prov identity.Provenance) (Result, error) {
	containerType := normalizeContainerType(req.Container.Type)
	if containerType == "" {
		containerType = "items"
	}
	if containerType != "items" && containerType != "initiative" {
		return Result{}, apierr.BadRequest("container must be items or initiative")
	}
	if containerType == "initiative" && s.initLander == nil {
		return Result{}, fmt.Errorf("planimport: initiative container support is not configured")
	}

	plan, err := s.resolvePlan(ctx, req)
	if err != nil {
		return Result{}, err
	}
	payload, err := Translate(plan)
	if err != nil {
		return Result{}, err
	}
	var preparedInitiative *ImportedInitiative
	if containerType == "initiative" {
		if err := validateInitiativeMode(req.Container.Mode); err != nil {
			return Result{}, err
		}
		initSpec := initiativeSpec(req.Container, plan)
		landed, err := s.initLander.LandInitiative(ctx, initSpec, nil, prov)
		if err != nil {
			return Result{}, err
		}
		preparedInitiative = &landed
		initName := strings.TrimSpace(initSpec.Name)
		if initName == "" {
			initName = plan.GetSlug()
		}
		for i := range payload.Items {
			payload.Items[i].Initiative = initName
		}
	}
	refs, err := s.lander.LandBatch(ctx, payload, prov)
	if err != nil {
		return Result{}, err
	}
	result := Result{Slug: plan.GetSlug(), PlanID: plan.GetId(), Container: containerType, Items: refs, Count: len(refs)}
	for _, ref := range refs {
		switch ref.Action {
		case "created":
			result.Created++
		case "updated":
			result.Updated++
		default:
			result.Linked++
		}
	}
	if containerType == "initiative" {
		landed, err := s.initLander.LandInitiative(ctx, initiativeSpec(req.Container, plan), refs, prov)
		if err != nil {
			return Result{}, err
		}
		if preparedInitiative != nil && preparedInitiative.Action == "created" {
			landed.Action = "created"
		}
		result.Initiative = &landed
	}
	return result, nil
}

func (s *Service) resolvePlan(ctx context.Context, req Request) (*sharedv1.Plan, error) {
	if strings.TrimSpace(req.Markdown) != "" || strings.TrimSpace(req.SourcePath) != "" {
		importer, ok := s.fetcher.(planclient.PlanImporter)
		if !ok {
			return nil, fmt.Errorf("planimport: plan adoption is not configured")
		}
		return importer.ImportPlan(ctx, planclient.ImportPlanInput{
			SourcePath: strings.TrimSpace(req.SourcePath),
			Markdown:   req.Markdown,
			Title:      strings.TrimSpace(req.Title),
			Slug:       strings.TrimSpace(req.Slug),
		})
	}
	planID := strings.TrimSpace(req.PlanID)
	if planID == "" {
		return nil, apierr.BadRequest("plan_id is required unless source_path or markdown is provided")
	}
	return s.fetcher.GetPlan(ctx, planID)
}

// validateInitiativeMode rejects an import that would stamp an initiative
// with an unknown mode or a mode that does not target initiatives (e.g. the
// generic plan-target phased-plan-drain): initiative-keyed surfaces reject
// plan-target modes with typed errors, so accepting one here would create a
// wedged initiative. Empty means the server default and is always valid.
func validateInitiativeMode(raw string) error {
	mode := strings.TrimSpace(raw)
	if mode == "" {
		return nil
	}
	def, err := operatingmode.DefinitionFor(operatingmode.Mode(mode))
	if err != nil {
		return apierr.BadRequest("unknown operating mode %q (registered: %s)", mode, operatingmode.ModeList())
	}
	if def.Target.Kind != operatingmode.TargetInitiative {
		return apierr.BadRequest("mode %q targets %s, not an initiative; start it directly on the plan (operating-mode start / StartTargetPhase) or pick an initiative-target mode", def.Mode, def.Target.Kind)
	}
	return nil
}

func initiativeSpec(container ContainerSpec, plan *sharedv1.Plan) InitiativeSpec {
	name := strings.TrimSpace(container.Name)
	if name == "" {
		name = strings.TrimSpace(plan.GetSlug())
	}
	title := strings.TrimSpace(container.Title)
	if title == "" {
		title = strings.TrimSpace(plan.GetTitle())
	}
	if title == "" {
		title = name
	}
	return InitiativeSpec{
		Name:        name,
		Title:       title,
		Description: strings.TrimSpace(container.Description),
		Mode:        strings.TrimSpace(container.Mode),
		PlanRef: PlanRef{
			Provider: "plan-manager",
			PlanID:   strings.TrimSpace(plan.GetId()),
			Slug:     strings.TrimSpace(plan.GetSlug()),
			Role:     "operating_mode_plan",
		},
	}
}

func normalizeContainerType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "item", "items", "backlog":
		return "items"
	case "initiative", "initiatives":
		return "initiative"
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}
