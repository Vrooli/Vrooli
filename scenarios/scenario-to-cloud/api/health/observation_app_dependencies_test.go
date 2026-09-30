package health

import (
	"testing"
	"time"

	"scenario-to-cloud/apphealth"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/vps"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health"
)

// buildWithApp composes the observation from the same evidence as build, plus
// the application's own health body.
func buildWithApp(t *testing.T, app *apphealth.Report, now time.Time) *healthv1.HealthObservation {
	t.Helper()
	dep := fixtureDeployment(domain.StatusDeployed)
	live := fixtureLiveState(now, "running")
	report := vps.ComputeHealth(dep, fixtureManifest(), fixtureIdentity(), live, fixtureDNS(), fixtureTLS(), nil, app)
	report.Freshness = &domain.FreshnessStatus{Status: domain.FreshnessCurrent, Summary: "bundle matches"}
	return Build(Input{Deployment: dep, Report: report, LiveState: live, Now: now}, DefaultPolicy())
}

// A failing application dependency must reach the typed observation, because
// statusFor downgrades DEGRADED to HEALTHY when every published check passes.
// Before this check existed, a deployment with broken customer sign-in
// published HEALTHY.
func TestObservationPublishesFailingAppDependencyAsWarned(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	app := apphealth.Parse("https://example.com/health", []byte(
		`{"status":"degraded","dependencies":{"sign_in_email":{"connected":false,"error":"smtp: 535 Authentication failed"}}}`), now)

	obs := buildWithApp(t, &app, now)

	check := checkByID(obs, CheckApplicationDependencies)
	if check == nil {
		t.Fatal("the observation must publish an application_dependencies check")
	}
	if check.GetStatus() != healthv1.CheckStatus_CHECK_STATUS_WARNED {
		t.Errorf("status = %s, want WARNED", check.GetStatus())
	}
	if check.GetReasonCode() != "app_dependency_failing" {
		t.Errorf("reason_code = %q, want app_dependency_failing", check.GetReasonCode())
	}
	if obs.GetStatus() != healthv1.HealthStatus_HEALTH_STATUS_DEGRADED {
		t.Errorf("status = %s, want DEGRADED: a warned check must not be downgraded to healthy", obs.GetStatus())
	}
	if len(obs.GetNextActions()) == 0 {
		t.Error("a warned dependency must carry a next action")
	}
}

func TestObservationPassesWhenEveryAppDependencyWorks(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	app := apphealth.Parse("https://example.com/health", []byte(
		`{"status":"healthy","dependencies":{"database":{"connected":true}}}`), now)

	obs := buildWithApp(t, &app, now)

	if check := checkByID(obs, CheckApplicationDependencies); check.GetStatus() != healthv1.CheckStatus_CHECK_STATUS_PASSED {
		t.Errorf("status = %s, want PASSED", check.GetStatus())
	}
	if obs.GetStatus() != healthv1.HealthStatus_HEALTH_STATUS_HEALTHY {
		t.Errorf("status = %s, want HEALTHY", obs.GetStatus())
	}
}

// An unread body is SKIPPED. It must not be PASSED, which would assert
// something never observed, and must not degrade an otherwise healthy
// deployment.
func TestObservationSkipsUnreadAppHealth(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

	obs := buildWithApp(t, nil, now)

	check := checkByID(obs, CheckApplicationDependencies)
	if check.GetStatus() != healthv1.CheckStatus_CHECK_STATUS_SKIPPED {
		t.Errorf("status = %s, want SKIPPED", check.GetStatus())
	}
	if obs.GetStatus() != healthv1.HealthStatus_HEALTH_STATUS_HEALTHY {
		t.Errorf("status = %s, want HEALTHY: an unread body is not a defect", obs.GetStatus())
	}
}

// A check that did not apply must not hold a deployment at DEGRADED. Adding
// the application-dependencies check surfaced this: on a deployment whose
// legacy aggregate says degraded, a SKIPPED body kept publishing a warning
// no operator could act on.
func TestSkippedChecksDoNotBlockTheDegradedDowngrade(t *testing.T) {
	checks := []*healthv1.HealthCheck{
		check(CheckHostPresence, healthv1.CheckStatus_CHECK_STATUS_PASSED, "", ""),
		check(CheckApplicationDependencies, healthv1.CheckStatus_CHECK_STATUS_SKIPPED, "app_dependencies_not_observed", ""),
	}

	if got := statusFor(domain.HealthDegraded, true, checks); got != healthv1.HealthStatus_HEALTH_STATUS_HEALTHY {
		t.Errorf("status = %s, want HEALTHY: a skipped check is not a complaint", got)
	}
}

// Evidence that could not be gathered is still not evidence of health.
func TestUnavailableChecksKeepTheDeploymentDegraded(t *testing.T) {
	checks := []*healthv1.HealthCheck{
		check(CheckHostPresence, healthv1.CheckStatus_CHECK_STATUS_PASSED, "", ""),
		check(CheckSystemResources, healthv1.CheckStatus_CHECK_STATUS_UNAVAILABLE, "system_metrics_unavailable", ""),
	}

	if got := statusFor(domain.HealthDegraded, true, checks); got != healthv1.HealthStatus_HEALTH_STATUS_DEGRADED {
		t.Errorf("status = %s, want DEGRADED", got)
	}
}
