package registry

import (
	"time"

	integrationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/integrations"
)

func ToProtoStatus(status Status) *integrationsv1.StatusResponse {
	out := &integrationsv1.StatusResponse{
		Integrations: make([]*integrationsv1.IntegrationStatus, 0, len(status.Integrations)),
		ActiveMode:   status.ActiveMode,
		Override:     OverrideToProto(status.Override),
		Reason:       status.Reason,
		EvaluatedAt:  formatTime(status.EvaluatedAt),
	}
	for _, integration := range status.Integrations {
		out.Integrations = append(out.Integrations, ToProtoIntegrationStatus(integration))
	}
	return out
}

func ToProtoIntegrationStatus(status IntegrationStatus) *integrationsv1.IntegrationStatus {
	return &integrationsv1.IntegrationStatus{
		Id:          string(status.ID),
		DisplayName: status.DisplayName,
		State:       status.State,
		Stats: &integrationsv1.RollingStats{
			LatencyP50Ms: float64(status.Stats.LatencyP50.Microseconds()) / 1000,
			LatencyP95Ms: float64(status.Stats.LatencyP95.Microseconds()) / 1000,
			ErrorRate:    status.Stats.ErrorRate,
			DegradedRate: status.Stats.DegradedRate,
			LastOkAt:     formatTime(status.Stats.LastOKAt),
			SampleCount:  status.Stats.SampleCount,
		},
		Reason:   status.Reason,
		Required: status.Required,
	}
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
