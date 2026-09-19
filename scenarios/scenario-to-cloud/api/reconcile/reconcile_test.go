package reconcile

import (
	"testing"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
)

func desired(state domain.DesiredState) Desired {
	return Desired{DeploymentID: "dep-1", Revision: 4, State: state, ReleaseDigest: "sha256:aaaa", ConfigurationDigest: "sha256:cccc", RecordedAt: time.Unix(1, 0)}
}

func boolPtr(v bool) *bool { return &v }

// [REQ:STC-P0-028] P15-A01: an intentional stop is never drift that starts
// the workload; observation proposes at most a stop when the workload is
// still running, and a reboot leaves it stopped.
func TestIntentionalStopIsNeverRestarted(t *testing.T) {
	now := time.Unix(100, 0)
	report := Detect(desired(domain.DesiredStopped), Observed{ReleaseDigest: "sha256:aaaa", ConfigurationDigest: "sha256:cccc", Status: "unhealthy", Freshness: "current", WorkloadRunning: boolPtr(false)}, &domain.ClosureSupervision{AutoRestart: true}, now)
	if report.Outcome != Unchanged || report.Correction != nil || report.Reasons[0] != ReasonDesiredStopped {
		t.Fatalf("stopped workload must be unchanged with no correction: %+v", report)
	}
	if report.Reboot.Decision != RebootLeaveStopped {
		t.Fatalf("a reboot must leave a stopped deployment stopped even when auto_restart is declared: %+v", report.Reboot)
	}
	running := Detect(desired(domain.DesiredStopped), Observed{ReleaseDigest: "sha256:aaaa", Status: "healthy", Freshness: "current", WorkloadRunning: boolPtr(true)}, nil, now)
	if running.Outcome != Changed || running.Correction == nil || running.Correction.Kind != CorrectionStopWorkload || running.Correction.Scope != execplan.ScopeStop {
		t.Fatalf("a running workload under a stopped intent is corrected by a stop, never a start: %+v", running)
	}
	retired := Detect(desired(domain.DesiredRetired), Observed{}, nil, now)
	if retired.Outcome != Unchanged || retired.Correction != nil {
		t.Fatalf("retired = %+v", retired)
	}
}

// [REQ:STC-P0-028] P15-A06: drift is corrected only through the current
// authorized plan; the report names the scope and never applies anything.
func TestDriftProposesPlanScopedCorrections(t *testing.T) {
	now := time.Unix(100, 0)
	cases := []struct {
		name       string
		observed   Observed
		outcome    Outcome
		correction string
		scope      string
	}{
		{"release differs", Observed{ReleaseDigest: "sha256:bbbb", ConfigurationDigest: "sha256:cccc", Status: "healthy", Freshness: "current"}, Changed, CorrectionApplyRelease, execplan.ScopeRuntime},
		{"configuration differs", Observed{ReleaseDigest: "aaaa", ConfigurationDigest: "sha256:dddd", Status: "healthy", Freshness: "current"}, Changed, CorrectionApplyRelease, execplan.ScopeRuntime},
		{"unhealthy at desired release", Observed{ReleaseDigest: "sha256:aaaa", ConfigurationDigest: "sha256:cccc", Status: "unhealthy", Freshness: "current"}, Changed, CorrectionRestartWorkload, execplan.ScopeStart},
		{"satisfied", Observed{ReleaseDigest: "sha256:aaaa", ConfigurationDigest: "sha256:cccc", Status: "healthy", Freshness: "current"}, Unchanged, "", ""},
		{"stale observation", Observed{ReleaseDigest: "sha256:bbbb", Status: "healthy", Freshness: "stale"}, Unknown, CorrectionObserveFirst, ""},
		{"unknown freshness", Observed{ReleaseDigest: "sha256:bbbb", Status: "healthy"}, Unknown, CorrectionObserveFirst, ""},
		{"release unobserved", Observed{Status: "healthy", Freshness: "current"}, Unknown, CorrectionObserveFirst, ""},
		{"unknown status", Observed{ReleaseDigest: "sha256:aaaa", ConfigurationDigest: "sha256:cccc", Status: "mystery", Freshness: "current"}, Unknown, CorrectionObserveFirst, ""},
		{"interrupted activation", Observed{ReleaseDigest: "sha256:aaaa", Status: "healthy", Freshness: "current", ActivationInterrupted: true}, Blocked, CorrectionResolveActivate, execplan.ScopeRuntime},
	}
	for _, c := range cases {
		report := Detect(desired(domain.DesiredRunning), c.observed, nil, now)
		if report.Outcome != c.outcome {
			t.Fatalf("%s: outcome %s, want %s (%v)", c.name, report.Outcome, c.outcome, report.Reasons)
		}
		if c.correction == "" && report.Correction != nil {
			t.Fatalf("%s: unexpected correction %+v", c.name, report.Correction)
		}
		if c.correction != "" && (report.Correction == nil || report.Correction.Kind != c.correction || report.Correction.Scope != c.scope) {
			t.Fatalf("%s: correction %+v, want %s/%s", c.name, report.Correction, c.correction, c.scope)
		}
		if report.Desired.Revision != 4 || report.ComputedAt != now.UTC() {
			t.Fatalf("%s: desired revision and computed time must be carried: %+v", c.name, report)
		}
	}
}

// [REQ:STC-P0-028] P15-A05: reboot recovery follows the declared
// supervision, never an implicit restart.
func TestRebootPolicyFollowsDeclaredSupervision(t *testing.T) {
	if p := Reboot(domain.DesiredRunning, &domain.ClosureSupervision{AutoRestart: true, StartupPolicy: "always", AutoRestartSource: "service.json"}); p.Decision != RebootSupervisorRestarts || p.StartupPolicy != "always" {
		t.Fatalf("policy = %+v", p)
	}
	if p := Reboot(domain.DesiredRunning, &domain.ClosureSupervision{AutoRestart: false}); p.Decision != RebootOperatorStartRequired {
		t.Fatalf("policy = %+v", p)
	}
	if p := Reboot(domain.DesiredRunning, nil); p.Decision != RebootOperatorStartRequired {
		t.Fatalf("undeclared supervision must never restart: %+v", p)
	}
	if p := Reboot(domain.DesiredStopped, &domain.ClosureSupervision{AutoRestart: true}); p.Decision != RebootLeaveStopped {
		t.Fatalf("policy = %+v", p)
	}
}

// [REQ:STC-P0-028] P15-A02/A04: the retirement plan lists retained and
// deleted objects in owner order, requires an explicit data policy, and
// keeps active, previous, recovery-referenced artifacts and every recovery
// point.
func TestRetirementPlanListsRetainedAndDeletedInOwnerOrder(t *testing.T) {
	in := RetirementInputs{
		DeploymentID: "dep-1", Domain: "app.example.test", Scenarios: []string{"app"}, CredentialRefs: []string{"db-password"},
		DataBindings: []string{"records-db", "uploads"}, LegacyData: []string{"app/cache"},
		ActiveRelease: "r2", PreviousRelease: "r1", Releases: []string{"r0", "r1", "r2", "r3"},
		RecoveryPoints: []string{"rp-1"}, RecoveryPointReleases: []string{"r0"},
	}
	blocked := Retirement(in)
	if blocked.Outcome != Blocked || len(blocked.Blocked) != 1 {
		t.Fatalf("missing retention policy must block: %+v", blocked)
	}
	for _, o := range blocked.Deleted {
		if o.Kind == KindData {
			t.Fatalf("data must never be deleted without a policy: %+v", o)
		}
	}
	in.RetentionPolicy = execplan.RetentionRetain
	plan := Retirement(in)
	if plan.Outcome != Changed || len(plan.Blocked) != 0 {
		t.Fatalf("retain plan = %+v", plan)
	}
	if got := []string{plan.Order[0], plan.Order[1], plan.Order[2], plan.Order[3], plan.Order[4]}; got[0] != KindRoute || got[1] != KindRuntime || got[2] != KindGrant || got[3] != KindData || got[4] != KindArtifact {
		t.Fatalf("order = %v", got)
	}
	deleted := map[string]bool{}
	for _, o := range plan.Deleted {
		deleted[o.Kind+":"+o.ID] = true
	}
	retained := map[string]bool{}
	for _, o := range plan.Retained {
		retained[o.Kind+":"+o.ID] = true
	}
	for _, want := range []string{"route:app.example.test", "runtime:app", "grant:db-password", "artifact:r3"} {
		if !deleted[want] {
			t.Fatalf("%s must be deleted: %+v", want, plan.Deleted)
		}
	}
	for _, want := range []string{"data:records-db", "data:uploads", "data:app/cache", "artifact:r0", "artifact:r1", "artifact:r2", "recovery_point:rp-1"} {
		if !retained[want] {
			t.Fatalf("%s must be retained: %+v", want, plan.Retained)
		}
	}
	in.RetentionPolicy = execplan.RetentionDelete
	deleting := Retirement(in)
	if deleting.Outcome != Blocked || len(deleting.Blocked) != 1 {
		t.Fatalf("an irreversible disposition without a target owner must be blocked, not silently retained: %+v", deleting)
	}
}

// [REQ:STC-P0-028] P15-A04: cleanup protection comes from references, not
// from lease age: a crashed owner's expired lease does not expose the active
// release, the predecessor, a release a recovery point references, or a
// recovery point an in-flight operation holds.
func TestCleanupProtectionSurvivesStaleLeases(t *testing.T) {
	points := []domain.RecoveryPoint{
		{ID: "rp-active", ReleaseDigest: "sha256:r2"},
		{ID: "rp-old", ReleaseDigest: "sha256:r0", OperationID: "op-restore"},
		{ID: "rp-loose", ReleaseDigest: "sha256:r9"},
		{ID: "rp-pinned", ReleaseDigest: "sha256:r9", Protected: true},
	}
	protection := Protect("sha256:r2", "r1", points, []string{"op-restore"})
	wantReleases := []string{"r0", "r1", "r2", "r9"}
	if len(protection.ReleaseDigests) != len(wantReleases) {
		t.Fatalf("releases = %v", protection.ReleaseDigests)
	}
	for i, want := range wantReleases {
		if protection.ReleaseDigests[i] != want {
			t.Fatalf("releases = %v", protection.ReleaseDigests)
		}
	}
	if len(protection.RecoveryPoints) != 3 || protection.RecoveryPoints[0] != "rp-active" || protection.RecoveryPoints[1] != "rp-old" || protection.RecoveryPoints[2] != "rp-pinned" {
		t.Fatalf("recovery points = %v", protection.RecoveryPoints)
	}
}
