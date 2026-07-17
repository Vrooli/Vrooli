package onboard

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// AdmissionResult is durable, redacted evidence that the selected candidate
// reached the exact endpoint it will later use for pairing and dial-out.
type AdmissionResult struct {
	Endpoint  string
	SourceIP  string
	Category  string
	Detail    string
	Duration  time.Duration
	Retryable bool
}

const (
	AdmissionPassed                  = "passed"
	AdmissionControlPlaneUnreachable = "control_plane_unreachable"
	AdmissionNameUnresolvable        = "endpoint_name_unresolvable"
	AdmissionEndpointInvalid         = "endpoint_invalid"
	AdmissionControlPlaneUnhealthy   = "control_plane_unhealthy"
	AdmissionDependencyUnavailable   = "dependency_unavailable"
)

// CandidateProber is deliberately optional while old SSH-driver fakes remain
// useful. Production always implements it; absence fails closed before pairing.
type CandidateProber interface {
	ProbeEndpoint(context.Context, Conn, string) (AdmissionResult, error)
}

func admissionFailureReason(category string) FailureReason {
	switch category {
	case AdmissionEndpointInvalid:
		return FailureEndpointInvalid
	case AdmissionNameUnresolvable:
		return FailureEndpointNameUnresolvable
	case AdmissionControlPlaneUnhealthy:
		return FailureControlPlaneUnhealthy
	case AdmissionDependencyUnavailable:
		return FailureDependencyUnavailable
	default:
		return FailureControlPlaneUnreachable
	}
}

func (s *service) admitCandidate(ctx context.Context, conn Conn, endpoint string) AdmissionResult {
	prober, ok := s.driver.(CandidateProber)
	if !ok {
		return AdmissionResult{Endpoint: endpoint, Category: AdmissionControlPlaneUnreachable, Detail: "candidate admission probe is unavailable on this control plane", Retryable: false}
	}
	result, err := prober.ProbeEndpoint(ctx, conn, endpoint)
	if result.Endpoint == "" {
		result.Endpoint = endpoint
	}
	if err != nil && result.Detail == "" {
		result.Detail = err.Error()
	}
	if result.Category == "" {
		result.Category = AdmissionControlPlaneUnreachable
	}
	result.Detail = strings.TrimSpace(result.Detail)
	return result
}

func admissionDetail(result AdmissionResult) string {
	parts := []string{fmt.Sprintf("endpoint %s", result.Endpoint)}
	if result.SourceIP != "" {
		parts = append(parts, "candidate source "+result.SourceIP)
	}
	if result.Detail != "" {
		parts = append(parts, result.Detail)
	}
	return strings.Join(parts, "; ")
}
