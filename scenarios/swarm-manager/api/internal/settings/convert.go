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

func settingsToProto(s Settings) *domainpb.Settings {
	return &domainpb.Settings{
		Theme:                  s.Theme,
		DefaultMode:            s.DefaultMode,
		AutoFixup:              s.AutoFixup,
		MaxFixupAttempts:       int32(s.MaxFixupAttempts),
		ReviewAgentEnabled:     s.ReviewAgentEnabled,
		MaxAutoRounds:          int32(s.MaxAutoRounds),
		AutoInitializeWorkshop: s.AutoInitializeWorkshop,
		AutoAdvanceWorkshop:    s.AutoAdvanceWorkshop,
		AutoCascadeWorkshop:    s.AutoCascadeWorkshop,
		AgentMaxTurns:          int32(s.AgentMaxTurns),
		AgentTimeoutSeconds:    int32(s.AgentTimeoutSeconds),
		SearchDebounceMs:       int32(s.SearchDebounceMs),
		ToastDurationMs:        int32(s.ToastDurationMs),
		DeleteConfirmation: &domainpb.DeleteConfirmationSettings{
			Backlog:    deleteConfirmLevelToProto(s.DeleteConfirmation.Backlog),
			Initiative: deleteConfirmLevelToProto(s.DeleteConfirmation.Initiative),
			Capture:    deleteConfirmLevelToProto(s.DeleteConfirmation.Capture),
		},
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
	if req.AgentMaxTurns != nil {
		v := int(*req.AgentMaxTurns)
		patch.AgentMaxTurns = &v
	}
	if req.AgentTimeoutSeconds != nil {
		v := int(*req.AgentTimeoutSeconds)
		patch.AgentTimeoutSeconds = &v
	}
	if req.SearchDebounceMs != nil {
		v := int(*req.SearchDebounceMs)
		patch.SearchDebounceMs = &v
	}
	if req.ToastDurationMs != nil {
		v := int(*req.ToastDurationMs)
		patch.ToastDurationMs = &v
	}
	if req.DeleteConfirmation != nil {
		dc := &DeleteConfirmationSettingsPatch{}
		b := deleteConfirmLevelFromProto(req.DeleteConfirmation.Backlog)
		dc.Backlog = &b
		i := deleteConfirmLevelFromProto(req.DeleteConfirmation.Initiative)
		dc.Initiative = &i
		c := deleteConfirmLevelFromProto(req.DeleteConfirmation.Capture)
		dc.Capture = &c
		patch.DeleteConfirmation = dc
	}
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
	return patch
}

func isEmptyUpdateSettingsRequest(req *apipb.UpdateSettingsRequest) bool {
	if req == nil {
		return true
	}
	return req.Theme == nil &&
		req.DefaultMode == nil &&
		req.AutoFixup == nil &&
		req.MaxFixupAttempts == nil &&
		req.ReviewAgentEnabled == nil &&
		req.MaxAutoRounds == nil &&
		req.AutoInitializeWorkshop == nil &&
		req.AutoAdvanceWorkshop == nil &&
		req.AutoCascadeWorkshop == nil &&
		req.AgentMaxTurns == nil &&
		req.AgentTimeoutSeconds == nil &&
		req.SearchDebounceMs == nil &&
		req.ToastDurationMs == nil &&
		req.DeleteConfirmation == nil &&
		req.ReviewCodeQualityMinScore == nil &&
		req.ReviewTestMinPassRate == nil &&
		req.ReviewMaxBlockingViolations == nil &&
		req.ReviewMaxWarnings == nil &&
		req.ReviewRequireScreenshots == nil &&
		req.ReviewRequireTests == nil &&
		req.LaneConcurrencyLimits == nil &&
		req.MaxQueueDepth == nil &&
		req.CircuitBreakerThreshold == nil &&
		req.CircuitBreakerCooldownMinutes == nil &&
		req.ExecutionCostCapPerRun == nil &&
		req.CostPerTurnEstimate == nil
}
