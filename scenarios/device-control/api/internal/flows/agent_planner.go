package flows

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"device-control/strategy"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

const AgentPlanRole = "device-control.plan"

type AgentWorld struct {
	Goal          string                         `json:"goal"`
	Capabilities  map[string]strategy.Capability `json:"capabilities"`
	StepKinds     []string                       `json:"step_kinds"`
	State         strategy.DeviceState           `json:"state"`
	FrameOptional bool                           `json:"frame_optional"`
}

type AgentPlan struct {
	// GoalMet lets a planner terminate a bounded loop without inventing a
	// no-op step. A plan that is not complete must still name a declared step.
	GoalMet  bool   `json:"goal_met,omitempty"`
	StepKind string `json:"step_kind"`
	Action   string `json:"action,omitempty"`
	Value    any    `json:"value,omitempty"`
}

type AgentPlanner interface {
	Plan(context.Context, AgentWorld) (AgentPlan, error)
}

// GatewayPlanner keeps agent planning on the same generated ai-gateway seam as
// visual resolution. It never selects a provider SDK or silently falls back to
// a second model route.
type GatewayPlanner struct{ Gateway InferenceRunner }

func NewGatewayPlanner(gateway InferenceRunner) *GatewayPlanner {
	return &GatewayPlanner{Gateway: gateway}
}

func (p *GatewayPlanner) Plan(ctx context.Context, world AgentWorld) (AgentPlan, error) {
	if p == nil || p.Gateway == nil {
		return AgentPlan{}, &UnavailableError{Reason: "ai_gateway_client_not_configured"}
	}
	encoded, err := json.Marshal(world)
	if err != nil {
		return AgentPlan{}, fmt.Errorf("encode agent world model: %w", err)
	}
	request := connect.NewRequest(&inferencev1.RunRequest{
		Role:        AgentPlanRole,
		Instruction: "Either set goal_met=true when the goal is satisfied, or choose exactly one declared step_kind that advances the goal. Return JSON with goal_met, optional step_kind, optional action, and optional value. Do not invent capabilities, coordinates, or device state.",
		SchemaJson:  `{"type":"object","required":["goal_met"],"properties":{"goal_met":{"type":"boolean"},"step_kind":{"type":"string"},"action":{"type":"string"},"value":{}}}`,
		Turns:       []*inferencev1.Turn{{Role: "user", Text: string(encoded)}},
	})
	response, err := p.Gateway.Run(ctx, request)
	if err != nil {
		return AgentPlan{}, err
	}
	if response == nil || response.Msg == nil {
		return AgentPlan{}, &UnavailableError{Reason: "empty_gateway_response"}
	}
	if response.Msg.GetError() != nil {
		return AgentPlan{}, fmt.Errorf("agent gateway inference failed: %s", response.Msg.GetError().GetMessage())
	}
	var plan AgentPlan
	if err := json.Unmarshal([]byte(response.Msg.GetValueJson()), &plan); err != nil {
		return AgentPlan{}, fmt.Errorf("decode agent plan: %w", err)
	}
	if !plan.GoalMet && strings.TrimSpace(plan.StepKind) == "" {
		return AgentPlan{}, fmt.Errorf("agent gateway returned an empty step_kind")
	}
	return plan, nil
}
