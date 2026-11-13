package corpusgen

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

// CallerEvent is the minimal projection of an authenticated Agent Manager
// event needed to preserve a real retrieval request. It deliberately contains
// no conversation transcript or tool result body.
type CallerEvent struct {
	RunID        string
	EventID      string
	Sequence     uint64
	Harness      string
	ContentClass string
	EvidenceRef  string
	Query        string
	// SourceVerified must only be set by the Agent Manager owner adapter after
	// fetching and matching the exact run/event/sequence and message body.
	SourceVerified bool
}

// CallerLabelReview is a human audit of the expected documents for one actual
// caller query. Search results from the system under evaluation must not be
// used as label evidence.
type CallerLabelReview struct {
	Reviewer               string
	EvidenceURI            string
	ExpectedIDs            []string
	IndependentlyValidated bool
}

// GatewayRouteProof mirrors the privacy-safe join fields emitted by an
// authenticated Agent Manager replay with a durable Gateway route receipt.
type GatewayRouteProof struct {
	SourceRunID, SourceEventID  string
	SourceSequence              uint64
	GatewayEventID              string
	GatewayRequestID            string
	GatewayRoleEvidenceURI      string
	TaskRole                    string
	Role                        string
	Status                      string
	SelectedProvider            string
	SelectedModel               string
	Locality                    string
	CapacityClaimID             string
	CapacityRequestedBytes      int64
	CapacityRequiredBytes       int64
	CapacityGrantedBytes        int64
	CapacityVerdict             string
	CapacityEstimateEvidenceURI string
}

// AuditedCallerCase converts a real Agent Manager event into a Search Hub
// corpus case. Without an independent label review it remains a candidate and
// cannot enter an acceptance denominator. Role assignment and the exact
// caller-source join come from an executed Gateway route receipt; query text
// alone does not infer a model policy role.
func AuditedCallerCase(event CallerEvent, route GatewayRouteProof, review CallerLabelReview) (*evalv1.EvalCase, error) {
	if err := event.validate(); err != nil {
		return nil, err
	}
	if route.SourceRunID != event.RunID || route.SourceEventID != event.EventID || route.SourceSequence != event.Sequence {
		return nil, fmt.Errorf("Gateway receipt source identity does not match the Agent Manager event")
	}
	if route.GatewayEventID == "" || route.GatewayRequestID == "" || route.GatewayRoleEvidenceURI != "gateway-ledger://events/"+route.GatewayEventID {
		return nil, fmt.Errorf("caller case requires a durable Gateway event/request identity and role evidence URI")
	}
	if route.Status != "succeeded" || route.Locality != "local" || route.SelectedProvider == "" || route.SelectedModel == "" {
		return nil, fmt.Errorf("caller case route must have succeeded locally with a known provider and model")
	}
	if route.CapacityVerdict != "fit" || route.CapacityClaimID == "" || route.CapacityRequestedBytes <= 0 || route.CapacityRequiredBytes != route.CapacityRequestedBytes || route.CapacityGrantedBytes < route.CapacityRequiredBytes || !strings.HasPrefix(route.CapacityEstimateEvidenceURI, "capacity-footprint://") {
		return nil, fmt.Errorf("caller case route lacks an admitted local capacity claim")
	}
	if !isModelPolicyRole(route.Role) {
		return nil, fmt.Errorf("caller case uses unknown model policy role %q", route.Role)
	}
	if strings.TrimSpace(route.TaskRole) == "" {
		return nil, fmt.Errorf("caller case requires the distinct AI Gateway task role from its source-linked route")
	}
	caseID := callerCaseID(event.RunID, event.EventID, route.Role, route.GatewayEventID)
	result := &evalv1.EvalCase{
		CaseId: caseID, Query: strings.TrimSpace(event.Query), Status: "candidate",
		Tags: []string{"caller-audit", "agent-manager", "role:" + route.Role},
		Note: fmt.Sprintf("source=agent-manager://runs/%s/events/%s seq=%d gateway_event=%s gateway_request=%s role=%s role_evidence=%s model=%s capacity_estimate=%s", event.RunID, event.EventID, event.Sequence, route.GatewayEventID, route.GatewayRequestID, route.Role, route.GatewayRoleEvidenceURI, route.SelectedModel, route.CapacityEstimateEvidenceURI),
	}
	if strings.TrimSpace(route.TaskRole) != "" {
		result.Tags = append(result.Tags, "task-role:"+strings.TrimSpace(route.TaskRole))
		result.Note += " task_role=" + strings.TrimSpace(route.TaskRole)
	}
	if !review.IndependentlyValidated {
		return result, nil
	}
	if review.Reviewer == "" || review.EvidenceURI == "" || len(review.ExpectedIDs) == 0 {
		return nil, fmt.Errorf("independently validated caller labels require reviewer, evidence URI, and expected IDs")
	}
	seen := make(map[string]struct{}, len(review.ExpectedIDs))
	for _, expectedID := range review.ExpectedIDs {
		if strings.TrimSpace(expectedID) == "" {
			return nil, fmt.Errorf("independently validated caller labels cannot contain empty expected IDs")
		}
		if _, exists := seen[expectedID]; exists {
			return nil, fmt.Errorf("independently validated caller labels contain duplicate expected ID %q", expectedID)
		}
		seen[expectedID] = struct{}{}
	}
	result.Status = "reviewed"
	result.ExpectIds = append([]string(nil), review.ExpectedIDs...)
	result.Tags = append(result.Tags, "independently-reviewed")
	result.Note += " label_review=" + review.EvidenceURI + " reviewer=" + review.Reviewer
	return result, nil
}

func callerCaseID(runID, eventID, role, gatewayEventID string) string {
	idMaterial := runID + "/" + eventID + "/" + role + "/" + gatewayEventID
	sum := sha256.Sum256([]byte(idMaterial))
	return "caller-" + hex.EncodeToString(sum[:12])
}

func (e CallerEvent) validate() error {
	if e.RunID == "" || e.EventID == "" {
		return fmt.Errorf("caller event requires authenticated run and event identity")
	}
	if !e.SourceVerified {
		return fmt.Errorf("caller event has not been verified against Agent Manager")
	}
	wantRef := fmt.Sprintf("agent-manager://runs/%s/events/%s", e.RunID, e.EventID)
	if e.Harness == "" || e.ContentClass != "user-message" || e.EvidenceRef != wantRef {
		return fmt.Errorf("caller event must be a typed Agent Manager user-message record")
	}
	if strings.TrimSpace(e.Query) == "" {
		return fmt.Errorf("caller event has no recoverable retrieval query")
	}
	return nil
}
