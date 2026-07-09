package operatingmode

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	PlanRefProviderPlanManager      = "plan-manager"
	PlanRefRoleOperatingModePlan    = "operating_mode_plan"
	planContextSourceResume         = "plan-manager.resume"
	planContextSourceMissingPrepare = "prepare_plan.pending"
)

type PlanRef struct {
	Provider string `json:"provider,omitempty"`
	PlanID   string `json:"plan_id,omitempty"`
	Slug     string `json:"slug,omitempty"`
	Role     string `json:"role,omitempty"`
}

type PlanExecutionClient interface {
	Resume(context.Context, *executionv1.ResumeRequest) (*executionv1.ResumeResponse, error)
	GetNext(context.Context, *executionv1.GetNextRequest) (*executionv1.GetNextResponse, error)
	GetStatus(context.Context, *executionv1.GetStatusRequest) (*executionv1.GetStatusResponse, error)
}

type PlanExecutionContext struct {
	Required     bool            `json:"required"`
	Missing      bool            `json:"missing,omitempty"`
	Source       string          `json:"source,omitempty"`
	PlanRef      *PlanRef        `json:"plan_ref,omitempty"`
	ExecutionID  string          `json:"execution_id,omitempty"`
	PlanID       string          `json:"plan_id,omitempty"`
	Complete     bool            `json:"complete,omitempty"`
	PhaseContext json.RawMessage `json:"phase_context,omitempty"`
	Step         json.RawMessage `json:"step,omitempty"`
}

func normalizePlanRef(ref *PlanRef) *PlanRef {
	if ref == nil {
		return nil
	}
	out := &PlanRef{
		Provider: strings.TrimSpace(ref.Provider),
		PlanID:   strings.TrimSpace(ref.PlanID),
		Slug:     strings.TrimSpace(ref.Slug),
		Role:     strings.TrimSpace(ref.Role),
	}
	if out.Provider == "" {
		out.Provider = PlanRefProviderPlanManager
	}
	return out
}

func validateOperatingModePlanRef(ref *PlanRef) error {
	ref = normalizePlanRef(ref)
	if ref == nil {
		return fmt.Errorf("plan_ref is required")
	}
	if ref.Provider != PlanRefProviderPlanManager {
		return fmt.Errorf("plan_ref provider must be %q (got %q)", PlanRefProviderPlanManager, ref.Provider)
	}
	if strings.TrimSpace(ref.PlanID) == "" && strings.TrimSpace(ref.Slug) == "" {
		return fmt.Errorf("plan_ref requires plan_id or slug")
	}
	if ref.Role != PlanRefRoleOperatingModePlan {
		return fmt.Errorf("plan_ref role must be %q (got %q)", PlanRefRoleOperatingModePlan, ref.Role)
	}
	return nil
}

func (s *Service) collectPlanContext(ctx context.Context, init InitiativeSnapshot, def Definition, phaseDef PhaseDefinition) (*PlanExecutionContext, error) {
	if !def.PlanRef.Required {
		return nil, nil
	}
	ref := normalizePlanRef(init.PlanRef)
	if ref == nil {
		if phaseDef.Phase == def.PhaseGraph.StartPhase {
			return &PlanExecutionContext{
				Required: true,
				Missing:  true,
				Source:   planContextSourceMissingPrepare,
			}, nil
		}
		return nil, fmt.Errorf("mode %q phase %q requires initiative plan_ref", def.Mode, phaseDef.Phase)
	}
	if err := validateOperatingModePlanRef(ref); err != nil {
		return nil, fmt.Errorf("mode %q phase %q invalid initiative plan_ref: %w", def.Mode, phaseDef.Phase, err)
	}
	if s.planExecution == nil {
		return nil, fmt.Errorf("mode %q phase %q requires plan-manager execution client", def.Mode, phaseDef.Phase)
	}
	resp, err := s.planExecution.Resume(ctx, &executionv1.ResumeRequest{
		PlanOrExecution: firstNonEmpty(ref.PlanID, ref.Slug),
	})
	if err != nil {
		return nil, err
	}
	out := &PlanExecutionContext{
		Required: true,
		Source:   planContextSourceResume,
		PlanRef:  ref,
	}
	if exec := resp.GetExecution(); exec != nil {
		out.ExecutionID = strings.TrimSpace(exec.GetId())
		out.PlanID = strings.TrimSpace(exec.GetPlanId())
		out.Complete = exec.GetComplete()
	}
	out.PhaseContext = marshalProtoJSON(resp.GetContext())
	out.Step = marshalProtoJSON(resp.GetStep())
	return out, nil
}

func marshalProtoJSON(msg proto.Message) json.RawMessage {
	if msg == nil {
		return nil
	}
	data, err := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
	}.Marshal(msg)
	if err != nil {
		return nil
	}
	return data
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
