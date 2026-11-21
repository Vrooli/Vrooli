package vps

import (
	"testing"
	"time"

	"scenario-to-cloud/apphealth"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/sshidentity"
)

// The live failure this exists to prevent: the application answers 200, so
// every reachability check passes, while its own body reports that customer
// sign-in email is broken. The report must not read healthy.
func TestComputeHealth_FailingAppDependencyIsDegradedNotHealthy(t *testing.T) {
	app := apphealth.Parse("https://example.com/health", []byte(
		`{"status":"degraded","dependencies":{"database":{"connected":true},`+
			`"sign_in_email":{"connected":false,"error":"smtp: 535 Authentication failed"}}}`), time.Now())

	resp := ComputeHealth(newTestDeployment(domain.StatusDeployed), newTestManifest(),
		newExplicitIdentity(sshidentity.VerificationAuthorized), newHealthyLiveState(),
		newHealthyDNSEval(), newHealthyTLSSnapshot(), nil, &app)

	if resp.Health != domain.HealthDegraded {
		t.Errorf("health = %s, want degraded", resp.Health)
	}
	check := findCheck(resp, appDependenciesCategory, "app_dependency_sign_in_email")
	if check == nil {
		t.Fatal("the failing dependency must appear as its own check")
	}
	if check.Status != domain.HealthCheckWarn {
		t.Errorf("status = %s, want warn (a lost capability is not a broken deployment)", check.Status)
	}
	if check.Message == "" || check.Details["app_status"] != "degraded" {
		t.Errorf("check must carry the application's reason and status: %+v", check)
	}
	if passing := findCheck(resp, appDependenciesCategory, "app_dependency_database"); passing == nil ||
		passing.Status != domain.HealthCheckPass {
		t.Errorf("a working dependency must still be reported as passing: %+v", passing)
	}
}

func TestComputeHealth_FailingAppDependencyProducesRecommendation(t *testing.T) {
	app := apphealth.Parse("https://example.com/health", []byte(
		`{"status":"degraded","dependencies":{"sign_in_email":{"connected":false,"error":"smtp: 535"}}}`), time.Now())

	resp := ComputeHealth(newTestDeployment(domain.StatusDeployed), newTestManifest(),
		newExplicitIdentity(sshidentity.VerificationAuthorized), newHealthyLiveState(),
		newHealthyDNSEval(), newHealthyTLSSnapshot(), nil, &app)

	for _, rec := range resp.Recommendations {
		if rec.Category == appDependenciesCategory {
			if rec.Command == "" {
				t.Error("an app-dependency recommendation must carry a command")
			}
			return
		}
	}
	t.Errorf("a failing dependency must produce a recommendation, got %+v", resp.Recommendations)
}

func TestComputeHealth_HealthyAppDependenciesStayHealthy(t *testing.T) {
	app := apphealth.Parse("https://example.com/health", []byte(
		`{"status":"healthy","dependencies":{"database":{"connected":true}}}`), time.Now())

	resp := ComputeHealth(newTestDeployment(domain.StatusDeployed), newTestManifest(),
		newExplicitIdentity(sshidentity.VerificationAuthorized), newHealthyLiveState(),
		newHealthyDNSEval(), newHealthyTLSSnapshot(), nil, &app)

	if resp.Health != domain.HealthHealthy {
		t.Errorf("health = %s, want healthy", resp.Health)
	}
}

// An unobserved application body must be a skip. Skips do not count as
// warnings, so an unreadable body neither degrades a deployment nor lets one
// claim its dependencies pass.
func TestComputeHealth_UnobservedAppHealthIsSkippedNotPassed(t *testing.T) {
	for name, app := range map[string]*apphealth.Report{
		"never fetched": nil,
		"unreadable": func() *apphealth.Report {
			r := apphealth.Parse("https://example.com/health", []byte("<html>"), time.Now())
			return &r
		}(),
	} {
		t.Run(name, func(t *testing.T) {
			resp := ComputeHealth(newTestDeployment(domain.StatusDeployed), newTestManifest(),
				newExplicitIdentity(sshidentity.VerificationAuthorized), newHealthyLiveState(),
				newHealthyDNSEval(), newHealthyTLSSnapshot(), nil, app)

			if resp.Health != domain.HealthHealthy {
				t.Errorf("health = %s: an unobserved body must not degrade the deployment", resp.Health)
			}
			check := findCheck(resp, appDependenciesCategory, "app_dependencies_unavailable")
			if check == nil {
				t.Fatal("an unobserved body must be reported as unavailable")
			}
			if check.Status != domain.HealthCheckSkip {
				t.Errorf("status = %s, want skip", check.Status)
			}
		})
	}
}
