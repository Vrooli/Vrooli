package registry

import (
	"time"

	integrationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/integrations"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/shared"
)

type IntegrationID string

const (
	IntegrationSearchHub    IntegrationID = "search-hub"
	IntegrationOpenRouter   IntegrationID = "openrouter"
	IntegrationAgentManager IntegrationID = "agent-manager"
	IntegrationPromptMgr    IntegrationID = "prompt-manager"
)

type Override string

const (
	OverrideAuto         Override = "auto"
	OverrideForceOff     Override = "force-off"
	OverrideForcePassive Override = "force-passive"
)

type ProbeResult struct {
	OK       bool
	Degraded bool
	Reason   string
}

type Sample struct {
	At       time.Time
	Latency  time.Duration
	OK       bool
	Degraded bool
	Reason   string
}

type RollingStats struct {
	LatencyP50    time.Duration
	LatencyP95    time.Duration
	ErrorRate     float64
	DegradedRate  float64
	LastOKAt      time.Time
	SampleCount   int64
	ConsecutiveOK int
	LastReason    string
}

type IntegrationStatus struct {
	ID          IntegrationID
	DisplayName string
	State       sharedv1.IntegrationState
	Stats       RollingStats
	Reason      string
	Required    bool
}

type Status struct {
	Integrations []IntegrationStatus
	ActiveMode   sharedv1.BehaviorMode
	Override     Override
	Reason       string
	EvaluatedAt  time.Time
}

func OverrideFromProto(value integrationsv1.BehaviorOverride) Override {
	switch value {
	case integrationsv1.BehaviorOverride_BEHAVIOR_OVERRIDE_FORCE_OFF:
		return OverrideForceOff
	case integrationsv1.BehaviorOverride_BEHAVIOR_OVERRIDE_FORCE_PASSIVE:
		return OverrideForcePassive
	default:
		return OverrideAuto
	}
}

func OverrideToProto(value Override) integrationsv1.BehaviorOverride {
	switch value {
	case OverrideForceOff:
		return integrationsv1.BehaviorOverride_BEHAVIOR_OVERRIDE_FORCE_OFF
	case OverrideForcePassive:
		return integrationsv1.BehaviorOverride_BEHAVIOR_OVERRIDE_FORCE_PASSIVE
	default:
		return integrationsv1.BehaviorOverride_BEHAVIOR_OVERRIDE_AUTO
	}
}
