package registry

import (
	"time"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/shared"
)

const (
	enterOffP95       = 15 * time.Second
	recoverPassiveP95 = 8 * time.Second
	enterOffErrorRate = 0.60
	recoverErrorRate  = 0.20
	recoveryOKSamples = 2
)

type PolicyInput struct {
	PreviousMode sharedv1.BehaviorMode
	Override     Override
	Search       IntegrationStatus
}

type PolicyDecision struct {
	Mode   sharedv1.BehaviorMode
	Reason string
}

func DecideMode(input PolicyInput) PolicyDecision {
	switch input.Override {
	case OverrideForceOff:
		return PolicyDecision{Mode: sharedv1.BehaviorMode_BEHAVIOR_MODE_OFF, Reason: "operator override forces search off"}
	case OverrideForcePassive:
		return PolicyDecision{Mode: sharedv1.BehaviorMode_BEHAVIOR_MODE_PASSIVE, Reason: "operator override forces passive search"}
	}

	search := input.Search
	stats := search.Stats
	if search.State == sharedv1.IntegrationState_INTEGRATION_STATE_UNAVAILABLE {
		return PolicyDecision{Mode: sharedv1.BehaviorMode_BEHAVIOR_MODE_OFF, Reason: defaultReason(search.Reason, "search-hub is unavailable")}
	}
	if stats.SampleCount == 0 || stats.ConsecutiveOK == 0 {
		return PolicyDecision{Mode: sharedv1.BehaviorMode_BEHAVIOR_MODE_OFF, Reason: "waiting for a successful search-hub measurement"}
	}
	if input.PreviousMode == sharedv1.BehaviorMode_BEHAVIOR_MODE_OFF && stats.ConsecutiveOK < recoveryOKSamples {
		return PolicyDecision{Mode: sharedv1.BehaviorMode_BEHAVIOR_MODE_OFF, Reason: "waiting for search-hub recovery window"}
	}
	if stats.ErrorRate >= enterOffErrorRate {
		return PolicyDecision{Mode: sharedv1.BehaviorMode_BEHAVIOR_MODE_OFF, Reason: "search-hub error rate is too high"}
	}
	if stats.LatencyP95 >= enterOffP95 {
		return PolicyDecision{Mode: sharedv1.BehaviorMode_BEHAVIOR_MODE_OFF, Reason: "search-hub latency is too high"}
	}
	if input.PreviousMode == sharedv1.BehaviorMode_BEHAVIOR_MODE_OFF {
		if stats.ErrorRate > recoverErrorRate || stats.LatencyP95 > recoverPassiveP95 {
			return PolicyDecision{Mode: sharedv1.BehaviorMode_BEHAVIOR_MODE_OFF, Reason: "search-hub has not met recovery thresholds"}
		}
	}
	return PolicyDecision{Mode: sharedv1.BehaviorMode_BEHAVIOR_MODE_PASSIVE, Reason: "search-hub is ready for passive search"}
}

func defaultReason(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
