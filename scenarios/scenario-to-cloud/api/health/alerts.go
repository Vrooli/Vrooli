package health

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/eventbus"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/perfbudget"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health"
)

// AlertSource is the eventbus.DomainEvent source for deployment alerts.
const AlertSource = "scenario-to-cloud"

// certTimeLayout is the layout tlsinfo uses for certificate times in the
// legacy report's tls_cert check details ("expires").
const certTimeLayout = "Jan 2 15:04:05 2006 MST"

// AlertPublisher is the delivery seam: eventbus.Client publishes to
// vrooli-events, which forwards to notification-hub's event webhook.
type AlertPublisher interface {
	PublishDomainEvent(ctx context.Context, event eventbus.DomainEvent) error
}

// DefaultDetection is the phase-23 detection budget used when
// certification/budgets.json cannot be read. Values mirror the frozen file.
func DefaultDetection() perfbudget.Detection {
	return perfbudget.Detection{
		AlertDetectionSecondsMax:       60,
		StaleObservationAfterSeconds:   int(DefaultMaxObservationAge / time.Second),
		CertificateExpiryWarningDays:   14,
		BackupRPOSecondsMax:            300,
		RecoveryNotificationSecondsMax: 60,
	}
}

// DefaultHostConcurrencyLimit mirrors phase-23's
// deployment_queue.effectful_operations_per_host_max for hosts where
// certification/budgets.json cannot be read.
const DefaultHostConcurrencyLimit = 4

// LoadHostConcurrencyLimit reads deployment_queue.effectful_operations_per_host_max
// from the same budget file LoadDetection uses, falling back to
// DefaultHostConcurrencyLimit. It reports which source was used.
func LoadHostConcurrencyLimit() (int, string) {
	for _, path := range budgetCandidates() {
		budgets, err := perfbudget.Load(path)
		if err != nil {
			continue
		}
		if n := budgets.Phase23.DeploymentQueue.EffectfulOperationsPerHostMax; n > 0 {
			return n, path
		}
	}
	return DefaultHostConcurrencyLimit, "default"
}

// budgetCandidates lists the budget file locations in precedence order.
func budgetCandidates() []string {
	candidates := []string{}
	if explicit := strings.TrimSpace(os.Getenv("SCENARIO_TO_CLOUD_BUDGETS")); explicit != "" {
		candidates = append(candidates, explicit)
	}
	return append(candidates, filepath.Join("certification", "budgets.json"), filepath.Join("..", "certification", "budgets.json"))
}

// LoadDetection reads the phase-23 detection budget from
// certification/budgets.json, looking relative to the working directory
// (scenario root or api/), or SCENARIO_TO_CLOUD_BUDGETS when set. It falls
// back to DefaultDetection and reports whether the file was used.
func LoadDetection() (perfbudget.Detection, string) {
	for _, path := range budgetCandidates() {
		budgets, err := perfbudget.Load(path)
		if err != nil {
			continue
		}
		d := budgets.Phase23.Detection
		if d.StaleObservationAfterSeconds <= 0 {
			d.StaleObservationAfterSeconds = DefaultDetection().StaleObservationAfterSeconds
		}
		if d.CertificateExpiryWarningDays <= 0 {
			d.CertificateExpiryWarningDays = DefaultDetection().CertificateExpiryWarningDays
		}
		return d, path
	}
	return DefaultDetection(), ""
}

// Alerter evaluates each produced observation against the detection budget
// and publishes scenario-to-cloud.deployment.alert.v1 transitions: every
// newly open condition and a resolved event for every condition that
// cleared. Steady state publishes nothing. It keeps the open set per
// deployment in memory; a restart re-publishes currently open conditions,
// which notification-hub de-duplicates on incident_id.
type Alerter struct {
	publisher AlertPublisher
	detection perfbudget.Detection
	log       func(string, map[string]interface{})
	now       func() time.Time

	mu   sync.Mutex
	open map[string][]perfbudget.Alert
}

// NewAlerter builds an alerter. A nil publisher disables delivery but keeps
// evaluation so callers can still read the open set.
func NewAlerter(publisher AlertPublisher, detection perfbudget.Detection, log func(string, map[string]interface{})) *Alerter {
	if log == nil {
		log = func(string, map[string]interface{}) {}
	}
	return &Alerter{publisher: publisher, detection: detection, log: log, now: func() time.Time { return time.Now().UTC() }, open: map[string][]perfbudget.Alert{}}
}

// Detection returns the budget in use.
func (a *Alerter) Detection() perfbudget.Detection { return a.detection }

// Evaluate returns the open alerts for one observation plus the edge facts
// carried in the legacy report. Pure: no publishing, no state change.
func (a *Alerter) Evaluate(obs *healthv1.HealthObservation, report domain.HealthResponse, now time.Time) []perfbudget.Alert {
	alerts := perfbudget.Evaluate(perfbudget.AlertInput{
		Observation:         obs,
		CertificateNotAfter: certificateNotAfter(report, now),
		Now:                 now,
	}, a.detection)
	alerts = carryObservationNextAction(alerts, obs)
	if renewal := certificateRenewalFailed(obs, report, now); renewal != nil {
		// A failed certificate check is a renewal failure, not merely an
		// approaching expiry: it replaces the date-derived alert.
		alerts = append(withoutCheck(alerts, perfbudget.CheckCertificateExpiring), *renewal)
	}
	return alerts
}

// Observe evaluates the observation, publishes the transitions since the
// previous evaluation for the same deployment, and returns what was
// published. Publish failures are logged and leave the previous open set in
// place so the transition is retried on the next observation; the detection
// budget therefore bounds delivery by the observation interval.
func (a *Alerter) Observe(ctx context.Context, obs *healthv1.HealthObservation, report domain.HealthResponse) []perfbudget.Alert {
	if a == nil || obs == nil {
		return nil
	}
	now := a.now()
	current := a.Evaluate(obs, report, now)
	key := obs.GetDeploymentId()

	a.mu.Lock()
	previous := a.open[key]
	a.mu.Unlock()

	transitions := perfbudget.Transitions(previous, current, now)
	if len(transitions) == 0 {
		a.mu.Lock()
		a.open[key] = current
		a.mu.Unlock()
		return nil
	}
	var published []perfbudget.Alert
	failed := false
	for _, alert := range transitions {
		if a.publisher == nil {
			published = append(published, alert)
			continue
		}
		if err := a.publisher.PublishDomainEvent(ctx, alertEvent(alert)); err != nil {
			failed = true
			a.log("deployment alert publish failed", map[string]interface{}{
				"deployment_id": alert.DeploymentID, "incident_id": alert.IncidentID, "status": alert.Status, "error": err.Error(),
			})
			continue
		}
		published = append(published, alert)
	}
	if !failed {
		a.mu.Lock()
		a.open[key] = current
		a.mu.Unlock()
	}
	return published
}

// Open returns the currently open alerts for a deployment.
func (a *Alerter) Open(deploymentID string) []perfbudget.Alert {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]perfbudget.Alert(nil), a.open[deploymentID]...)
}

// alertEvent is the wire event for one alert. Occurred is detected_at so
// the envelope time is the producer's detection time.
func alertEvent(alert perfbudget.Alert) eventbus.DomainEvent {
	return eventbus.DomainEvent{
		Source:    AlertSource,
		EventType: perfbudget.AlertEventType,
		Payload:   alert.Facts(),
		Occurred:  alert.DetectedAt,
	}
}

// carryObservationNextAction replaces the generic runbook/CLI next action
// of service and status alerts with the producer's own typed remedy when
// the observation carries one, so the operator sees the exact command.
func carryObservationNextAction(alerts []perfbudget.Alert, obs *healthv1.HealthObservation) []perfbudget.Alert {
	actions := obs.GetNextActions()
	if len(actions) == 0 {
		return alerts
	}
	first := actions[0]
	for i := range alerts {
		if alerts[i].CheckID != perfbudget.CheckServiceUnavailable && alerts[i].Reason != "status_unknown" {
			continue
		}
		alerts[i].NextAction = perfbudget.NextAction{Owner: first.GetOwner(), Kind: first.GetKind(), Reference: first.GetReference(), Label: first.GetLabel()}
	}
	return alerts
}

// certificateNotAfter reads the edge certificate expiry from the legacy
// report's tls_cert check ("expires" in tlsinfo's layout, or
// "days_remaining"). Zero when the certificate was not observed.
func certificateNotAfter(report domain.HealthResponse, now time.Time) time.Time {
	c := reportCheck(report, "tls", "tls_cert")
	if c == nil || c.Status == domain.HealthCheckSkip {
		return time.Time{}
	}
	if expires := strings.TrimSpace(c.Details["expires"]); expires != "" {
		if ts, err := time.Parse(certTimeLayout, expires); err == nil {
			return ts.UTC()
		}
		if ts, err := time.Parse(time.RFC3339, expires); err == nil {
			return ts.UTC()
		}
	}
	if days := strings.TrimSpace(c.Details["days_remaining"]); days != "" {
		if n, err := strconv.Atoi(days); err == nil {
			return now.Add(time.Duration(n) * 24 * time.Hour)
		}
	}
	return time.Time{}
}

// certificateRenewalFailed raises certificate_expiring with reason
// renewal_failed when the edge TLS check failed outright (invalid or
// expired certificate, failed probe), which Evaluate cannot see from an
// expiry date alone.
func certificateRenewalFailed(obs *healthv1.HealthObservation, report domain.HealthResponse, now time.Time) *perfbudget.Alert {
	c := reportCheck(report, "tls", "tls_cert")
	if c == nil || (c.Status != domain.HealthCheckFail && c.Status != domain.HealthCheckError) {
		return nil
	}
	deploymentID := obs.GetDeploymentId()
	observedAt := now
	if obs.GetObservedAt() != nil {
		observedAt = obs.GetObservedAt().AsTime().UTC()
	}
	return &perfbudget.Alert{
		CheckID: perfbudget.CheckCertificateExpiring, Severity: perfbudget.SeverityCritical, Status: perfbudget.AlertOpen,
		IncidentID:   fmt.Sprintf("%s:%s", deploymentID, perfbudget.CheckCertificateExpiring),
		DeploymentID: deploymentID, TargetID: obs.GetTargetId(), ReleaseDigest: obs.GetObservedReleaseDigest(),
		ObservedAt: observedAt, DetectedAt: now, Reason: "renewal_failed",
		Message:    "The edge certificate is invalid, expired or could not be probed: " + strings.TrimSpace(c.Message),
		NextAction: perfbudget.NextAction{Owner: "scenario-to-cloud", Kind: "command", Reference: "scenario-to-cloud edge tls-renew " + deploymentID, Label: "Renew the certificate"},
	}
}

func withoutCheck(alerts []perfbudget.Alert, check string) []perfbudget.Alert {
	out := alerts[:0:0]
	for _, a := range alerts {
		if a.CheckID != check {
			out = append(out, a)
		}
	}
	return out
}

func reportCheck(report domain.HealthResponse, category, id string) *domain.HealthCheck {
	for i := range report.Sections {
		if report.Sections[i].Category != category {
			continue
		}
		return findCheck(&report.Sections[i], id)
	}
	return nil
}
