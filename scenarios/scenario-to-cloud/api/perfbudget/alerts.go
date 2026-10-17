package perfbudget

import (
	"fmt"
	"sort"
	"time"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health"
)

// AlertEventType is the notification-hub domain event emitted for a
// deployment alert. Producers publish facts only (see Alert.Facts);
// notification-hub owns the rendered copy and the sensitivity label.
const AlertEventType = "scenario-to-cloud.deployment.alert.v1"

// Alert check ids. Each is one SLO condition of
// docs/reference/service-objectives.md.
const (
	CheckServiceUnavailable        = "service_unavailable"
	CheckHealthObservationStale    = "health_observation_stale"
	CheckCertificateExpiring       = "certificate_expiring"
	CheckBackupFreshness           = "backup_freshness"
	CheckCredentialRotationPending = "credential_rotation_pending" // #nosec G101 -- public alert identifier, not credential material.
)

// Alert severities use notification-hub's vocabulary so the server-owned
// sensitivity mapping applies (critical → critical, warning → sensitive,
// informational → public).
const (
	SeverityCritical      = "critical"
	SeverityWarning       = "warning"
	SeverityInformational = "informational"
)

// Alert statuses. "open" is a condition currently met; "resolved" is the
// recovery notification for a previously open condition and carries the
// same incident_id so notification-hub de-duplicates the pair.
const (
	AlertOpen     = "open"
	AlertResolved = "resolved"
)

// NextAction is the typed operator action attached to an alert.
type NextAction struct {
	Owner     string `json:"owner"`
	Kind      string `json:"kind"`
	Reference string `json:"reference"`
	Label     string `json:"label"`
}

// Alert is one evaluated condition. Facts() is the payload of the
// scenario-to-cloud.deployment.alert.v1 event.
type Alert struct {
	CheckID       string     `json:"check_id"`
	Severity      string     `json:"severity"`
	Status        string     `json:"status"`
	IncidentID    string     `json:"incident_id"`
	DeploymentID  string     `json:"deployment_id"`
	TargetID      string     `json:"target_id"`
	ReleaseDigest string     `json:"release_digest"`
	ObservedAt    time.Time  `json:"observed_at"`
	DetectedAt    time.Time  `json:"detected_at"`
	Reason        string     `json:"reason"`
	Message       string     `json:"message"`
	NextAction    NextAction `json:"next_action"`
}

// Facts is the event payload: flat, snake_case, no secrets, no rendered
// copy beyond message/reason (which notification-hub may template).
func (a Alert) Facts() map[string]any {
	return map[string]any{
		"schema_version": 1,
		"check_id":       a.CheckID,
		"severity":       a.Severity,
		"status":         a.Status,
		"incident_id":    a.IncidentID,
		"deployment_id":  a.DeploymentID,
		"target_id":      a.TargetID,
		"release_digest": a.ReleaseDigest,
		"observed_at":    a.ObservedAt.UTC().Format(time.RFC3339),
		"detected_at":    a.DetectedAt.UTC().Format(time.RFC3339),
		"reason":         a.Reason,
		"message":        a.Message,
		"next_action": map[string]any{
			"owner":     a.NextAction.Owner,
			"kind":      a.NextAction.Kind,
			"reference": a.NextAction.Reference,
			"label":     a.NextAction.Label,
		},
	}
}

// AlertInput is everything Evaluate needs. Every field is an observation
// someone else produced; Evaluate performs no I/O.
type AlertInput struct {
	Observation *healthv1.HealthObservation
	// CertificateNotAfter is the edge certificate expiry (zero: unobserved).
	CertificateNotAfter time.Time
	// LastRecoveryPointAt is the newest recovery point (zero: none). Only
	// consulted when HasProtectedWrites is true.
	LastRecoveryPointAt time.Time
	HasProtectedWrites  bool
	// RotationPending names credential bindings whose rotation is pending.
	RotationPending []string
	Now             time.Time
}

// Evaluate returns the open alerts for the input under the detection
// budget. A stale or unknown observation never contributes a "service
// healthy" conclusion: it raises health_observation_stale and suppresses
// nothing else.
func Evaluate(in AlertInput, d Detection) []Alert {
	now := in.Now
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()
	var out []Alert
	obs := in.Observation
	var deploymentID, targetID, release string
	observedAt := now
	if obs != nil {
		deploymentID, targetID, release = obs.GetDeploymentId(), obs.GetTargetId(), obs.GetObservedReleaseDigest()
		if obs.GetObservedAt() != nil {
			observedAt = obs.GetObservedAt().AsTime().UTC()
		}
	}
	base := func(check, severity, reason, message string, next NextAction) Alert {
		return Alert{
			CheckID: check, Severity: severity, Status: AlertOpen,
			IncidentID:   fmt.Sprintf("%s:%s", deploymentID, check),
			DeploymentID: deploymentID, TargetID: targetID, ReleaseDigest: release,
			ObservedAt: observedAt, DetectedAt: now, Reason: reason, Message: message, NextAction: next,
		}
	}
	healthRef := "scenario-to-cloud deployment health --deployment " + deploymentID

	staleAfter := time.Duration(d.StaleObservationAfterSeconds) * time.Second
	switch {
	case obs == nil:
		out = append(out, base(CheckHealthObservationStale, SeverityCritical, "no_observation",
			"No health observation exists for the deployment; service state is unknown.",
			NextAction{Owner: "scenario-to-cloud", Kind: "cli", Reference: healthRef, Label: "Produce a health observation"}))
	case obs.GetFreshness() != healthv1.Freshness_FRESHNESS_CURRENT || (staleAfter > 0 && now.Sub(observedAt) > staleAfter):
		out = append(out, base(CheckHealthObservationStale, SeverityCritical, "observation_stale",
			fmt.Sprintf("The last health observation is %s old (freshness %s); the service cannot be reported healthy.", now.Sub(observedAt).Truncate(time.Second), obs.GetFreshness().String()),
			NextAction{Owner: "scenario-to-cloud", Kind: "cli", Reference: healthRef, Label: "Refresh the health observation"}))
	case obs.GetStatus() == healthv1.HealthStatus_HEALTH_STATUS_UNHEALTHY:
		out = append(out, base(CheckServiceUnavailable, SeverityCritical, "status_unhealthy",
			"The deployment is observed unhealthy on a current observation.",
			NextAction{Owner: "scenario-to-cloud", Kind: "doc", Reference: "docs/guides/incident-runbook.md#diagnose", Label: "Open the incident runbook"}))
	case obs.GetStatus() == healthv1.HealthStatus_HEALTH_STATUS_DEGRADED:
		out = append(out, base(CheckServiceUnavailable, SeverityWarning, "status_degraded",
			"The deployment is observed degraded on a current observation.",
			NextAction{Owner: "scenario-to-cloud", Kind: "doc", Reference: "docs/guides/incident-runbook.md#diagnose", Label: "Open the incident runbook"}))
	case obs.GetStatus() != healthv1.HealthStatus_HEALTH_STATUS_HEALTHY:
		out = append(out, base(CheckHealthObservationStale, SeverityCritical, "status_unknown",
			"The current observation carries an unknown status; the service cannot be reported healthy.",
			NextAction{Owner: "scenario-to-cloud", Kind: "cli", Reference: healthRef, Label: "Inspect the observation checks"}))
	}

	if !in.CertificateNotAfter.IsZero() {
		days := in.CertificateNotAfter.Sub(now).Hours() / 24
		if days <= float64(d.CertificateExpiryWarningDays) {
			sev := SeverityWarning
			if days <= 0 {
				sev = SeverityCritical
			}
			out = append(out, base(CheckCertificateExpiring, sev, "certificate_expiring",
				fmt.Sprintf("The edge certificate expires in %.0f days (not_after %s).", days, in.CertificateNotAfter.UTC().Format(time.RFC3339)),
				NextAction{Owner: "scenario-to-cloud", Kind: "cli", Reference: "scenario-to-cloud edge status --deployment " + deploymentID, Label: "Inspect renewal state"}))
		}
	}

	if in.HasProtectedWrites {
		rpo := time.Duration(d.BackupRPOSecondsMax) * time.Second
		age := now.Sub(in.LastRecoveryPointAt)
		if in.LastRecoveryPointAt.IsZero() || age > rpo {
			reason := "recovery_point_older_than_rpo"
			msg := fmt.Sprintf("The newest recovery point is %s old; the RPO is %s.", age.Truncate(time.Second), rpo)
			if in.LastRecoveryPointAt.IsZero() {
				reason, msg = "no_recovery_point", "No recovery point exists for a deployment with protected writes."
			}
			out = append(out, base(CheckBackupFreshness, SeverityCritical, reason, msg,
				NextAction{Owner: "scenario-to-cloud", Kind: "cli", Reference: "scenario-to-cloud deployment recovery-points capture --deployment " + deploymentID, Label: "Capture a recovery point"}))
		}
	}

	if len(in.RotationPending) > 0 {
		names := append([]string(nil), in.RotationPending...)
		sort.Strings(names)
		out = append(out, base(CheckCredentialRotationPending, SeverityWarning, "rotation_pending",
			fmt.Sprintf("%d credential binding(s) have a pending rotation.", len(names)),
			NextAction{Owner: "scenario-to-cloud", Kind: "cli", Reference: "scenario-to-cloud credential list --deployment " + deploymentID, Label: "Complete the rotation"}))
	}
	return out
}

// ServiceHealthy reports whether alerts allow the service to be called
// healthy: no open service_unavailable and no open
// health_observation_stale. A stale observation is never healthy.
func ServiceHealthy(alerts []Alert) bool {
	for _, a := range alerts {
		if a.Status != AlertOpen {
			continue
		}
		if a.CheckID == CheckServiceUnavailable || a.CheckID == CheckHealthObservationStale {
			return false
		}
	}
	return true
}

// Transitions compares the previous open set with the current evaluation
// and returns the events to publish: every newly open alert, and a
// resolved alert (same incident_id) for every previously open condition
// that is no longer met. Unchanged open alerts are not re-published; the
// producer re-evaluates on every observation and the detection budget is
// the observation interval plus delivery.
func Transitions(previous, current []Alert, now time.Time) []Alert {
	prevByID := map[string]Alert{}
	for _, a := range previous {
		if a.Status == AlertOpen {
			prevByID[a.IncidentID] = a
		}
	}
	curByID := map[string]Alert{}
	var out []Alert
	for _, a := range current {
		curByID[a.IncidentID] = a
		if _, open := prevByID[a.IncidentID]; !open {
			out = append(out, a)
		}
	}
	var resolved []Alert
	for id, a := range prevByID {
		if _, still := curByID[id]; still {
			continue
		}
		a.Status = AlertResolved
		a.Severity = SeverityInformational
		a.DetectedAt = now.UTC()
		a.Reason = "condition_cleared"
		a.Message = fmt.Sprintf("%s has cleared.", a.CheckID)
		resolved = append(resolved, a)
	}
	sort.Slice(resolved, func(i, j int) bool { return resolved[i].IncidentID < resolved[j].IncidentID })
	return append(out, resolved...)
}
