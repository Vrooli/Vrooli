// Package coverage contains the checks that measure whether critical findings
// have a recovery path and whether their event entered delivery processing.
package coverage

import (
	"context"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

const (
	RemediationReachCheckID = "coverage-remediation-reach"
	DeliveryReachCheckID    = "coverage-delivery-reach"
)

type CriticalFinding struct {
	ID    string
	Check string
	Title string
}

type DeliveryAttempt struct {
	IncidentID string
	Outcome    string
	Channel    string
}

type DeliverySnapshot struct {
	Incidents []CriticalFinding
	Attempts  []DeliveryAttempt
}

type DeliveryReader func(context.Context) (DeliverySnapshot, error)

// UnavailableDeliveryReader is the honest default until a notification-hub
// delivery projection is configured. It prevents the coverage cell from
// silently treating a missing cross-scenario read as healthy.
func UnavailableDeliveryReader(context.Context) (DeliverySnapshot, error) {
	return DeliverySnapshot{}, fmt.Errorf("notification-hub delivery-attempt projection is not configured")
}

// EvaluateRemediationReach is the pure ordered-severity projection. A
// critical finding is covered when its registered check exposes at least one
// recovery action; availability is reported separately because an unavailable
// action is still a remediation path that an operator can review.
func EvaluateRemediationReach(findings []CriticalFinding, actionCounts map[string]int) checks.Result {
	result := checks.Result{CheckID: RemediationReachCheckID, Timestamp: time.Now().UTC(), Details: map[string]interface{}{
		"projection": "recovery",
		"question":   "what fraction of critical findings have any remediation path at all?",
	}}
	missing := make([]string, 0)
	covered := 0
	for _, finding := range findings {
		if actionCounts[finding.Check] > 0 {
			covered++
			continue
		}
		missing = append(missing, finding.ID)
	}
	result.Details["criticalFindings"] = len(findings)
	result.Details["coveredFindings"] = covered
	result.Details["missingFindingIds"] = missing
	if len(missing) > 0 {
		result.Status = checks.StatusCritical
		result.Message = fmt.Sprintf("%d of %d critical findings have no registered remediation path", len(missing), len(findings))
		return result
	}
	result.Status = checks.StatusOK
	result.Message = fmt.Sprintf("all %d critical findings have a registered remediation path", len(findings))
	return result
}

func NewRemediationReachCheck(registry *checks.Registry) checks.Check {
	return &remediationReachCheck{registry: registry}
}

type remediationReachCheck struct{ registry *checks.Registry }

func (c *remediationReachCheck) ID() string    { return RemediationReachCheckID }
func (c *remediationReachCheck) Title() string { return "Critical Finding Remediation Reach" }
func (c *remediationReachCheck) Description() string {
	return "Measures whether critical findings expose a registered recovery action."
}
func (c *remediationReachCheck) Importance() string {
	return "A detected critical finding without a recovery path is an actionable coverage gap."
}
func (c *remediationReachCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *remediationReachCheck) IntervalSeconds() int       { return 300 }
func (c *remediationReachCheck) Platforms() []platform.Type { return nil }
func (c *remediationReachCheck) Run(_ context.Context) checks.Result {
	if c.registry == nil {
		return unreadable(RemediationReachCheckID, "health registry is unavailable")
	}
	results := c.registry.GetAllResults()
	if len(results) == 0 {
		return unreadable(RemediationReachCheckID, "no registered check results are available")
	}
	findings := make([]CriticalFinding, 0)
	actions := make(map[string]int)
	for _, result := range results {
		if result.Status != checks.StatusCritical || result.CheckID == DeliveryReachCheckID || result.CheckID == RemediationReachCheckID {
			continue
		}
		check, exists := c.registry.GetCheck(result.CheckID)
		if !exists {
			return unreadable(RemediationReachCheckID, "registered check disappeared while reading coverage")
		}
		findings = append(findings, CriticalFinding{ID: result.CheckID, Check: result.CheckID, Title: check.Title()})
		if healable, ok := check.(checks.HealableCheck); ok {
			actions[result.CheckID] = len(healable.RecoveryActions(&result))
		}
	}
	return EvaluateRemediationReach(findings, actions)
}

func (c *remediationReachCheck) ExecuteAction(context.Context, string) checks.ActionResult {
	return checks.ActionResult{ActionID: "observe", CheckID: c.ID(), Success: false, Message: "coverage checks are read-only"}
}

// EvaluateDeliveryReach joins the incident identity emitted by autoheal to
// durable notification intake attempts. Any attempt proves the dispatch path
// ran; the attempt outcome remains visible in the returned details.
func EvaluateDeliveryReach(snapshot DeliverySnapshot) checks.Result {
	result := checks.Result{CheckID: DeliveryReachCheckID, Timestamp: time.Now().UTC(), Details: map[string]interface{}{
		"projection": "substrate",
		"question":   "did a critical finding reach a human channel?",
	}}
	attempted := make(map[string]DeliveryAttempt, len(snapshot.Attempts))
	for _, attempt := range snapshot.Attempts {
		if attempt.IncidentID != "" {
			attempted[attempt.IncidentID] = attempt
		}
	}
	missing := make([]string, 0)
	for _, incident := range snapshot.Incidents {
		if _, ok := attempted[incident.ID]; !ok {
			missing = append(missing, incident.ID)
		}
	}
	result.Details["criticalIncidents"] = len(snapshot.Incidents)
	result.Details["attemptedIncidents"] = len(snapshot.Incidents) - len(missing)
	result.Details["missingIncidentIds"] = missing
	if len(missing) > 0 {
		result.Status = checks.StatusCritical
		result.Message = fmt.Sprintf("%d of %d critical incidents have no delivery attempt", len(missing), len(snapshot.Incidents))
		return result
	}
	result.Status = checks.StatusOK
	result.Message = fmt.Sprintf("all %d critical incidents have a delivery attempt", len(snapshot.Incidents))
	return result
}

func NewDeliveryReachCheck(reader DeliveryReader) checks.Check {
	return &deliveryReachCheck{reader: reader}
}

type deliveryReachCheck struct{ reader DeliveryReader }

func (c *deliveryReachCheck) ID() string    { return DeliveryReachCheckID }
func (c *deliveryReachCheck) Title() string { return "Critical Finding Delivery Reach" }
func (c *deliveryReachCheck) Description() string {
	return "Measures whether critical incident IDs have entered notification delivery."
}
func (c *deliveryReachCheck) Importance() string {
	return "A critical finding without a delivery attempt is an unread alert gap."
}
func (c *deliveryReachCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *deliveryReachCheck) IntervalSeconds() int       { return 300 }
func (c *deliveryReachCheck) Platforms() []platform.Type { return nil }
func (c *deliveryReachCheck) Run(ctx context.Context) checks.Result {
	if c.reader == nil {
		return unreadable(DeliveryReachCheckID, "notification delivery projection is not configured")
	}
	snapshot, err := c.reader(ctx)
	if err != nil {
		return unreadable(DeliveryReachCheckID, err.Error())
	}
	return EvaluateDeliveryReach(snapshot)
}

func (c *deliveryReachCheck) ExecuteAction(context.Context, string) checks.ActionResult {
	return checks.ActionResult{ActionID: "observe", CheckID: c.ID(), Success: false, Message: "coverage checks are read-only"}
}

func unreadable(id, reason string) checks.Result {
	return checks.Result{CheckID: id, Status: checks.StatusUndetermined, Message: "coverage projection is unreadable: " + reason, Details: map[string]interface{}{"readable": false, "reason": reason}, Timestamp: time.Now().UTC()}
}
