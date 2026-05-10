package scenarioruntime

import (
	"strings"
	"testing"
)

func TestClassifyPortEvidence(t *testing.T) {
	tests := []struct {
		name       string
		input      PortEvidenceInput
		wantCode   string
		wantConf   string
		wantReason string
	}{
		{
			name: "healthy running scenario with repeated unbound custom port suggests manifest drift",
			input: PortEvidenceInput{
				Claim: PortClaim{
					PortName:                  "metrics",
					EnvVar:                    "METRICS_PORT",
					ListenerStatus:            ListenerStatusNotListening,
					ConsecutiveListenerMisses: 3,
				},
				Instance: Instance{Status: StatusRunning},
				Health:   HealthSnapshot{Status: HealthStatusHealthy},
			},
			wantCode:   PortRecommendationLikelyManifestDrift,
			wantConf:   "medium",
			wantReason: "healthy",
		},
		{
			name: "unhealthy scenario with unbound declared port suggests runtime failure",
			input: PortEvidenceInput{
				Claim:    PortClaim{PortName: "control", EnvVar: "CONTROL_PORT", ListenerStatus: ListenerStatusNotListening},
				Instance: Instance{Status: StatusRunning},
				Health:   HealthSnapshot{Status: HealthStatusUnhealthy},
			},
			wantCode:   PortRecommendationLikelyRuntimeFailure,
			wantConf:   "medium",
			wantReason: "runtime health is not healthy",
		},
		{
			name: "inspection unavailable stays low confidence",
			input: PortEvidenceInput{
				Claim:    PortClaim{PortName: "api", EnvVar: "API_PORT", ListenerStatus: ListenerStatusInspectionUnavailable},
				Instance: Instance{Status: StatusRunning},
				Health:   HealthSnapshot{Status: HealthStatusHealthy},
			},
			wantCode:   PortRecommendationInspectionUnavailable,
			wantConf:   "low",
			wantReason: "inspection was unavailable",
		},
		{
			name: "listening claim is ok for arbitrary env var",
			input: PortEvidenceInput{
				Claim:    PortClaim{PortName: "playwright-driver", EnvVar: "DRIVER_PORT", ListenerStatus: ListenerStatusListening},
				Instance: Instance{Status: StatusRunning},
				Health:   HealthSnapshot{Status: HealthStatusHealthy},
			},
			wantCode:   PortRecommendationOK,
			wantConf:   "high",
			wantReason: "listener evidence",
		},
		{
			name: "stale reconciliation recommends expiration",
			input: PortEvidenceInput{
				Claim:             PortClaim{PortName: "api", EnvVar: "API_PORT", ListenerStatus: ListenerStatusNotListening},
				Reconciliation:    ReconcileStaleClaim,
				HasAuthoritative:  true,
				Authoritative:     false,
				HostListenerInUse: false,
			},
			wantCode:   PortRecommendationStaleClaimExpire,
			wantConf:   "high",
			wantReason: "non-authoritative",
		},
		{
			name: "host listener contradicting stored unbound evidence asks for orphan investigation",
			input: PortEvidenceInput{
				Claim:             PortClaim{PortName: "api", EnvVar: "API_PORT", ListenerStatus: ListenerStatusNotListening},
				Instance:          Instance{Status: StatusRunning},
				Health:            HealthSnapshot{Status: HealthStatusHealthy},
				HostListenerInUse: true,
			},
			wantCode:   PortRecommendationOrphanListenerInvestigate,
			wantConf:   "medium",
			wantReason: "listener is currently present",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyPortEvidence(tt.input)
			if got.Code != tt.wantCode || got.Confidence != tt.wantConf {
				t.Fatalf("ClassifyPortEvidence() = %#v, want code %q confidence %q", got, tt.wantCode, tt.wantConf)
			}
			if tt.wantReason != "" && !strings.Contains(got.Rationale, tt.wantReason) {
				t.Fatalf("Rationale = %q, want it to contain %q", got.Rationale, tt.wantReason)
			}
		})
	}
}
