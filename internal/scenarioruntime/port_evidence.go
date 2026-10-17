package scenarioruntime

const (
	PortRecommendationOK                        = "port-ok"
	PortRecommendationUnboundWatch              = "port-unbound-watch"
	PortRecommendationLikelyManifestDrift       = "port-unbound-likely-manifest-drift"
	PortRecommendationLikelyRuntimeFailure      = "port-unbound-likely-runtime-failure"
	PortRecommendationInspectionUnavailable     = "port-inspection-unavailable"
	PortRecommendationStaleClaimExpire          = "stale-claim-expire"
	PortRecommendationOrphanListenerInvestigate = "orphan-listener-investigate"
	PortRecommendationClaimResolved             = "claim-resolved"
)

type PortEvidenceInput struct {
	Claim              PortClaim
	Instance           Instance
	Health             HealthSnapshot
	Reconciliation     ReconcileClassification
	Authoritative      bool
	HasAuthoritative   bool
	HostListenerInUse  bool
	StaticEnvReference string
}

type PortEvidenceRecommendation struct {
	Code       string `json:"code"`
	Confidence string `json:"confidence"`
	Rationale  string `json:"rationale"`
}

func ClassifyPortEvidence(in PortEvidenceInput) PortEvidenceRecommendation {
	claim := in.Claim
	if !IsActivePortClaimStatus(claim.Status) {
		// Expired/released rows are resolved history awaiting retention pruning.
		// Recommending expiry for them would make every stale-claim consumer
		// count tombstones forever.
		return PortEvidenceRecommendation{
			Code:       PortRecommendationClaimResolved,
			Confidence: "high",
			Rationale:  "claim is already " + claim.Status + "; no action needed",
		}
	}
	if !in.HasAuthoritative || !in.Authoritative {
		if in.Reconciliation == ReconcileStaleClaim || in.Reconciliation == ReconcileStaleInstance {
			return PortEvidenceRecommendation{
				Code:       PortRecommendationStaleClaimExpire,
				Confidence: "high",
				Rationale:  "claim is non-authoritative according to runtime reconciliation",
			}
		}
	}
	switch claim.ListenerStatus {
	case ListenerStatusListening:
		return PortEvidenceRecommendation{
			Code:       PortRecommendationOK,
			Confidence: "high",
			Rationale:  "claimed port has listener evidence",
		}
	case ListenerStatusInspectionUnavailable:
		return PortEvidenceRecommendation{
			Code:       PortRecommendationInspectionUnavailable,
			Confidence: "low",
			Rationale:  "host listener inspection was unavailable",
		}
	case ListenerStatusNotListening:
		if in.HostListenerInUse {
			return PortEvidenceRecommendation{
				Code:       PortRecommendationOrphanListenerInvestigate,
				Confidence: "medium",
				Rationale:  "registry evidence says the claim was unbound but a listener is currently present",
			}
		}
		if in.Instance.Status == StatusRunning && in.Health.Status == HealthStatusHealthy && claim.ConsecutiveListenerMisses >= 2 {
			return PortEvidenceRecommendation{
				Code:       PortRecommendationLikelyManifestDrift,
				Confidence: "medium",
				Rationale:  "scenario is healthy while declared listener port remains unbound",
			}
		}
		if in.Instance.Status == StatusStarting || in.Health.Status == HealthStatusUnhealthy || in.Health.Status == HealthStatusDegraded {
			return PortEvidenceRecommendation{
				Code:       PortRecommendationLikelyRuntimeFailure,
				Confidence: "medium",
				Rationale:  "declared listener port is unbound while runtime health is not healthy",
			}
		}
		return PortEvidenceRecommendation{
			Code:       PortRecommendationUnboundWatch,
			Confidence: "low",
			Rationale:  "declared listener port is currently unbound but evidence is not conclusive",
		}
	default:
		return PortEvidenceRecommendation{
			Code:       PortRecommendationUnboundWatch,
			Confidence: "low",
			Rationale:  "no durable listener evidence has been recorded yet",
		}
	}
}
