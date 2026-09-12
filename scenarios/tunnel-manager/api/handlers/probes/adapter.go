package probes

import (
	"tunnel-manager/internal/probes"

	probesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/probes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// resultToProto converts an internal probes.ProbeResult into the wire
// shape the probes proto declares. Lives in the handler package by
// intent — the conversion is mechanical and only used at the transport
// edge.
func resultToProto(r probes.ProbeResult) *probesv1.ProbeResult {
	return &probesv1.ProbeResult{
		Id:         r.ID,
		Subdomain:  r.Subdomain,
		Kind:       kindToProto(r.Kind),
		Status:     statusToProto(r.Status),
		LatencyMs:  int32(r.LatencyMS),
		StatusCode: int32(r.StatusCode),
		ErrorMsg:   r.ErrorMsg,
		CreatedAt:  timestamppb.New(r.CreatedAt.UTC()),
	}
}

func classificationToProto(c probes.RouteClassification) *probesv1.RouteClassification {
	return &probesv1.RouteClassification{
		Subdomain:      c.Subdomain,
		Classification: failureClassToProto(c.Classification),
		Internal:       statusToProto(c.Internal),
		External:       statusToProto(c.External),
		Assessment:     c.Assessment,
	}
}

func kindToProto(k probes.ProbeKind) probesv1.ProbeKind {
	switch k {
	case probes.ProbeKindInternal:
		return probesv1.ProbeKind_PROBE_KIND_INTERNAL
	case probes.ProbeKindExternal:
		return probesv1.ProbeKind_PROBE_KIND_EXTERNAL
	default:
		return probesv1.ProbeKind_PROBE_KIND_UNSPECIFIED
	}
}

func statusToProto(s probes.ProbeStatus) probesv1.ProbeStatus {
	switch s {
	case probes.ProbeStatusUp:
		return probesv1.ProbeStatus_PROBE_STATUS_UP
	case probes.ProbeStatusDown:
		return probesv1.ProbeStatus_PROBE_STATUS_DOWN
	case probes.ProbeStatusTimeout:
		return probesv1.ProbeStatus_PROBE_STATUS_TIMEOUT
	case probes.ProbeStatusError:
		return probesv1.ProbeStatus_PROBE_STATUS_ERROR
	default:
		return probesv1.ProbeStatus_PROBE_STATUS_UNSPECIFIED
	}
}

func failureClassToProto(f probes.FailureClass) probesv1.FailureClass {
	switch f {
	case probes.FailureClassHealthy:
		return probesv1.FailureClass_FAILURE_CLASS_HEALTHY
	case probes.FailureClassTunnelDown:
		return probesv1.FailureClass_FAILURE_CLASS_TUNNEL_DOWN
	case probes.FailureClassScenarioDown:
		return probesv1.FailureClass_FAILURE_CLASS_SCENARIO_DOWN
	case probes.FailureClassCloudflareOutage:
		return probesv1.FailureClass_FAILURE_CLASS_CLOUDFLARE_OUTAGE
	case probes.FailureClassDNSFailure:
		return probesv1.FailureClass_FAILURE_CLASS_DNS_FAILURE
	case probes.FailureClassConfigDrift:
		return probesv1.FailureClass_FAILURE_CLASS_CONFIG_DRIFT
	default:
		return probesv1.FailureClass_FAILURE_CLASS_UNSPECIFIED
	}
}
