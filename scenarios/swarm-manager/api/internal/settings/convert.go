package settings

import (
	"math"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
)

// laneLimitsToProto narrows widths from int (Go) to int32 (proto). Values
// that overflow int32 are clamped to math.MaxInt32 — settings normalize
// already keeps them in [1, 50] but the clamp keeps the conversion total.
func laneLimitsToProto(limits map[string]int) map[string]int32 {
	if limits == nil {
		return nil
	}
	out := make(map[string]int32, len(limits))
	for k, v := range limits {
		switch {
		case v > math.MaxInt32:
			out[k] = math.MaxInt32
		case v < math.MinInt32:
			out[k] = math.MinInt32
		default:
			out[k] = int32(v)
		}
	}
	return out
}

// laneLimitsFromProto widens int32 (proto) to int (Go) without loss.
func laneLimitsFromProto(limits map[string]int32) map[string]int {
	if limits == nil {
		return nil
	}
	out := make(map[string]int, len(limits))
	for k, v := range limits {
		out[k] = int(v)
	}
	return out
}

func deleteConfirmLevelToProto(level DeleteConfirmLevel) domainpb.DeleteConfirmLevel {
	switch level {
	case DeleteConfirmNone:
		return domainpb.DeleteConfirmLevel_DELETE_CONFIRM_LEVEL_NONE
	case DeleteConfirmStrong:
		return domainpb.DeleteConfirmLevel_DELETE_CONFIRM_LEVEL_STRONG
	default:
		return domainpb.DeleteConfirmLevel_DELETE_CONFIRM_LEVEL_SIMPLE
	}
}

func deleteConfirmLevelFromProto(level domainpb.DeleteConfirmLevel) DeleteConfirmLevel {
	switch level {
	case domainpb.DeleteConfirmLevel_DELETE_CONFIRM_LEVEL_SIMPLE:
		return DeleteConfirmSimple
	case domainpb.DeleteConfirmLevel_DELETE_CONFIRM_LEVEL_NONE:
		return DeleteConfirmNone
	case domainpb.DeleteConfirmLevel_DELETE_CONFIRM_LEVEL_STRONG:
		return DeleteConfirmStrong
	default:
		return DeleteConfirmSimple
	}
}

func deleteConfirmationLevelsToProto(levels map[string]DeleteConfirmLevel) map[string]domainpb.DeleteConfirmLevel {
	if levels == nil {
		return nil
	}
	out := make(map[string]domainpb.DeleteConfirmLevel, len(levels))
	for k, v := range levels {
		out[k] = deleteConfirmLevelToProto(v)
	}
	return out
}

func deleteConfirmationLevelsFromProto(levels map[string]domainpb.DeleteConfirmLevel) map[string]DeleteConfirmLevel {
	if levels == nil {
		return nil
	}
	out := make(map[string]DeleteConfirmLevel, len(levels))
	for k, v := range levels {
		out[k] = deleteConfirmLevelFromProto(v)
	}
	return out
}

func settingsToProto(s Settings) *domainpb.Settings {
	return &domainpb.Settings{
		Theme:                         s.Theme,
		DefaultMode:                   s.DefaultMode,
		AutoFixup:                     s.AutoFixup,
		MaxFixupAttempts:              int32(s.MaxFixupAttempts),
		ReviewAgentEnabled:            s.ReviewAgentEnabled,
		MaxAutoRounds:                 int32(s.MaxAutoRounds),
		AutoInitializeWorkshop:        s.AutoInitializeWorkshop,
		AutoAdvanceWorkshop:           s.AutoAdvanceWorkshop,
		AutoCascadeWorkshop:           s.AutoCascadeWorkshop,
		AgentMaxTurns:                 int32(s.AgentMaxTurns),
		AgentTimeoutSeconds:           int32(s.AgentTimeoutSeconds),
		SearchDebounceMs:              int32(s.SearchDebounceMs),
		ToastDurationMs:               int32(s.ToastDurationMs),
		DeleteConfirmationLevels:      deleteConfirmationLevelsToProto(s.DeleteConfirmationLevels),
		ReviewCodeQualityMinScore:     s.ReviewCodeQualityMinScore,
		ReviewTestMinPassRate:         s.ReviewTestMinPassRate,
		ReviewMaxBlockingViolations:   int32(s.ReviewMaxBlockingViolations),
		ReviewMaxWarnings:             int32(s.ReviewMaxWarnings),
		ReviewRequireScreenshots:      s.ReviewRequireScreenshots,
		ReviewRequireTests:            s.ReviewRequireTests,
		LaneConcurrencyLimits:         laneLimitsToProto(s.LaneConcurrencyLimits),
		MaxQueueDepth:                 int32(s.MaxQueueDepth),
		CircuitBreakerThreshold:       int32(s.CircuitBreakerThreshold),
		CircuitBreakerCooldownMinutes: int32(s.CircuitBreakerCooldownMinutes),
		ExecutionCostCapPerRun:        s.ExecutionCostCapPerRun,
		CostPerTurnEstimate:           s.CostPerTurnEstimate,
		FixBeforeFeature:              s.FixBeforeFeature,
		FixBeforeFeatureDiscovery:     s.FixBeforeFeatureDiscovery,
	}
}

func settingsPatchFromProto(req *apipb.UpdateSettingsRequest) SettingsPatch {
	patch := SettingsPatch{}
	if req == nil {
		return patch
	}
	if req.Theme != nil {
		patch.Theme = req.Theme
	}
	executionPatchFromProto(req, &patch)
	workshopPatchFromProto(req, &patch)
	agentPatchFromProto(req, &patch)
	uiPatchFromProto(req, &patch)
	reviewPatchFromProto(req, &patch)
	governancePatchFromProto(req, &patch)
	return patch
}

// executionPatchFromProto copies the execution-default request fields.
func executionPatchFromProto(req *apipb.UpdateSettingsRequest, patch *SettingsPatch) {
	if req.DefaultMode != nil {
		s := *req.DefaultMode
		patch.DefaultMode = &s
	}
	if req.AutoFixup != nil {
		v := *req.AutoFixup
		patch.AutoFixup = &v
	}
	if req.MaxFixupAttempts != nil {
		v := int(*req.MaxFixupAttempts)
		patch.MaxFixupAttempts = &v
	}
	if req.ReviewAgentEnabled != nil {
		v := *req.ReviewAgentEnabled
		patch.ReviewAgentEnabled = &v
	}
}

// workshopPatchFromProto copies the workshop request fields. The proto request
// has no AutoAdvanceDelaySeconds field, so it is intentionally absent here.
func workshopPatchFromProto(req *apipb.UpdateSettingsRequest, patch *SettingsPatch) {
	if req.MaxAutoRounds != nil {
		v := int(*req.MaxAutoRounds)
		patch.MaxAutoRounds = &v
	}
	if req.AutoInitializeWorkshop != nil {
		v := *req.AutoInitializeWorkshop
		patch.AutoInitializeWorkshop = &v
	}
	if req.AutoAdvanceWorkshop != nil {
		v := *req.AutoAdvanceWorkshop
		patch.AutoAdvanceWorkshop = &v
	}
	if req.AutoCascadeWorkshop != nil {
		v := *req.AutoCascadeWorkshop
		patch.AutoCascadeWorkshop = &v
	}
}

// agentPatchFromProto copies the agent-behavior request fields.
func agentPatchFromProto(req *apipb.UpdateSettingsRequest, patch *SettingsPatch) {
	if req.AgentMaxTurns != nil {
		v := int(*req.AgentMaxTurns)
		patch.AgentMaxTurns = &v
	}
	if req.AgentTimeoutSeconds != nil {
		v := int(*req.AgentTimeoutSeconds)
		patch.AgentTimeoutSeconds = &v
	}
}

// uiPatchFromProto copies the UI-preference request fields.
func uiPatchFromProto(req *apipb.UpdateSettingsRequest, patch *SettingsPatch) {
	if req.SearchDebounceMs != nil {
		v := int(*req.SearchDebounceMs)
		patch.SearchDebounceMs = &v
	}
	if req.ToastDurationMs != nil {
		v := int(*req.ToastDurationMs)
		patch.ToastDurationMs = &v
	}
	if req.DeleteConfirmationLevels != nil {
		patch.DeleteConfirmationLevels = deleteConfirmationLevelsFromProto(req.DeleteConfirmationLevels)
	}
}

// reviewPatchFromProto copies the review-threshold request fields.
func reviewPatchFromProto(req *apipb.UpdateSettingsRequest, patch *SettingsPatch) {
	if req.ReviewCodeQualityMinScore != nil {
		v := *req.ReviewCodeQualityMinScore
		patch.ReviewCodeQualityMinScore = &v
	}
	if req.ReviewTestMinPassRate != nil {
		v := *req.ReviewTestMinPassRate
		patch.ReviewTestMinPassRate = &v
	}
	if req.ReviewMaxBlockingViolations != nil {
		v := int(*req.ReviewMaxBlockingViolations)
		patch.ReviewMaxBlockingViolations = &v
	}
	if req.ReviewMaxWarnings != nil {
		v := int(*req.ReviewMaxWarnings)
		patch.ReviewMaxWarnings = &v
	}
	if req.ReviewRequireScreenshots != nil {
		v := *req.ReviewRequireScreenshots
		patch.ReviewRequireScreenshots = &v
	}
	if req.ReviewRequireTests != nil {
		v := *req.ReviewRequireTests
		patch.ReviewRequireTests = &v
	}
}

// governancePatchFromProto copies the concurrency/governance and
// fix-before-feature request fields.
func governancePatchFromProto(req *apipb.UpdateSettingsRequest, patch *SettingsPatch) {
	if req.LaneConcurrencyLimits != nil {
		patch.LaneConcurrencyLimits = laneLimitsFromProto(req.LaneConcurrencyLimits)
	}
	if req.MaxQueueDepth != nil {
		v := int(*req.MaxQueueDepth)
		patch.MaxQueueDepth = &v
	}
	if req.CircuitBreakerThreshold != nil {
		v := int(*req.CircuitBreakerThreshold)
		patch.CircuitBreakerThreshold = &v
	}
	if req.CircuitBreakerCooldownMinutes != nil {
		v := int(*req.CircuitBreakerCooldownMinutes)
		patch.CircuitBreakerCooldownMinutes = &v
	}
	if req.ExecutionCostCapPerRun != nil {
		v := *req.ExecutionCostCapPerRun
		patch.ExecutionCostCapPerRun = &v
	}
	if req.CostPerTurnEstimate != nil {
		v := *req.CostPerTurnEstimate
		patch.CostPerTurnEstimate = &v
	}
	if req.FixBeforeFeature != nil {
		v := *req.FixBeforeFeature
		patch.FixBeforeFeature = &v
	}
	if req.FixBeforeFeatureDiscovery != nil {
		v := *req.FixBeforeFeatureDiscovery
		patch.FixBeforeFeatureDiscovery = &v
	}
}

func isEmptyUpdateSettingsRequest(req *apipb.UpdateSettingsRequest) bool {
	if req == nil {
		return true
	}
	return req.Theme == nil &&
		isEmptyExecutionRequest(req) &&
		isEmptyWorkshopRequest(req) &&
		isEmptyAgentRequest(req) &&
		isEmptyUIRequest(req) &&
		isEmptyReviewRequest(req) &&
		isEmptyGovernanceRequest(req)
}

// isEmptyExecutionRequest reports whether no execution-default field is set.
func isEmptyExecutionRequest(req *apipb.UpdateSettingsRequest) bool {
	return req.DefaultMode == nil &&
		req.AutoFixup == nil &&
		req.MaxFixupAttempts == nil &&
		req.ReviewAgentEnabled == nil
}

// isEmptyWorkshopRequest reports whether no workshop field is set.
func isEmptyWorkshopRequest(req *apipb.UpdateSettingsRequest) bool {
	return req.MaxAutoRounds == nil &&
		req.AutoInitializeWorkshop == nil &&
		req.AutoAdvanceWorkshop == nil &&
		req.AutoCascadeWorkshop == nil
}

// isEmptyAgentRequest reports whether no agent-behavior field is set.
func isEmptyAgentRequest(req *apipb.UpdateSettingsRequest) bool {
	return req.AgentMaxTurns == nil &&
		req.AgentTimeoutSeconds == nil
}

// isEmptyUIRequest reports whether no UI-preference field is set.
func isEmptyUIRequest(req *apipb.UpdateSettingsRequest) bool {
	return req.SearchDebounceMs == nil &&
		req.ToastDurationMs == nil &&
		req.DeleteConfirmationLevels == nil
}

// isEmptyReviewRequest reports whether no review-threshold field is set.
func isEmptyReviewRequest(req *apipb.UpdateSettingsRequest) bool {
	return req.ReviewCodeQualityMinScore == nil &&
		req.ReviewTestMinPassRate == nil &&
		req.ReviewMaxBlockingViolations == nil &&
		req.ReviewMaxWarnings == nil &&
		req.ReviewRequireScreenshots == nil &&
		req.ReviewRequireTests == nil
}

// isEmptyGovernanceRequest reports whether no concurrency/governance or
// fix-before-feature field is set.
func isEmptyGovernanceRequest(req *apipb.UpdateSettingsRequest) bool {
	return req.LaneConcurrencyLimits == nil &&
		req.MaxQueueDepth == nil &&
		req.CircuitBreakerThreshold == nil &&
		req.CircuitBreakerCooldownMinutes == nil &&
		req.ExecutionCostCapPerRun == nil &&
		req.CostPerTurnEstimate == nil &&
		req.FixBeforeFeature == nil &&
		req.FixBeforeFeatureDiscovery == nil
}
