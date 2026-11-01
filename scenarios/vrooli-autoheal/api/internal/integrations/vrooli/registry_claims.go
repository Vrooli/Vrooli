package vrooli

import "context"

// Port-claim recommendation codes mirror the canonical port-evidence
// classifications in the root CLI's internal/scenarioruntime package (which is
// internal to the root module and cannot be imported from a scenario module).
// They are part of the `vrooli locks --json` wire contract, carried on
// ProjectRuntimeClaim.recommendation_code.
const (
	RecommendationOK                        = "port-ok"
	RecommendationStaleClaimExpire          = "stale-claim-expire"
	RecommendationOrphanListenerInvestigate = "orphan-listener-investigate"
)

// RegistryClaim is one runtime port-registry claim — the authoritative (and
// only) record of who owns a port. The legacy `.port_*.lock` file layer is
// fully retired.
type RegistryClaim struct {
	Port               int    `json:"port"`
	Scenario           string `json:"scenario"`
	ClaimID            string `json:"claim_id"`
	InstanceID         string `json:"instance_id"`
	ClaimStatus        string `json:"claim_status"`
	InstanceStatus     string `json:"instance_status"`
	Reconciliation     string `json:"reconciliation"`
	RecommendationCode string `json:"recommendation_code"`
	Authoritative      *bool  `json:"authoritative,omitempty"`
}

// StaleExpireRecommended reports whether core maintenance recommends expiring
// this claim (non-authoritative per runtime reconciliation). This is exactly
// the set that `vrooli cleanup locks` acts on, so autoheal reuses the producer's
// recommendation rather than re-deriving staleness.
func (rc RegistryClaim) StaleExpireRecommended() bool {
	return rc.RecommendationCode == RecommendationStaleClaimExpire
}

// OrphanListenerSuspected reports a listener holding the port with no
// authoritative claim — a different problem from a stale claim that
// `vrooli cleanup locks` will not resolve.
func (rc RegistryClaim) OrphanListenerSuspected() bool {
	return rc.RecommendationCode == RecommendationOrphanListenerInvestigate
}

// ListRegistryClaims runs `vrooli locks --json` and returns the runtime
// registry's port claims — the authoritative allocation records that allocation
// hygiene must reason over (the legacy locks array is near-always empty now).
func (c *Client) ListRegistryClaims(ctx context.Context) ([]RegistryClaim, []byte, error) {
	resp, err := c.cli.Locks(ctx)
	if err != nil {
		return nil, nil, err
	}
	claims := make([]RegistryClaim, 0, len(resp.GetRegistryClaims()))
	for _, rc := range resp.GetRegistryClaims() {
		claims = append(claims, RegistryClaim{
			Port:               int(rc.GetPort()),
			Scenario:           rc.GetScenario(),
			ClaimID:            rc.GetClaimId(),
			InstanceID:         rc.GetInstanceId(),
			ClaimStatus:        rc.GetClaimStatus(),
			InstanceStatus:     rc.GetInstanceStatus(),
			Reconciliation:     rc.GetReconciliation(),
			RecommendationCode: rc.GetRecommendationCode(),
			Authoritative:      valueToBoolPtr(rc.GetAuthoritative()),
		})
	}
	return claims, diagnosticJSON(resp), nil
}
