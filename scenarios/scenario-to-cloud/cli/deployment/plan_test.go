package deployment

import (
	"bytes"
	"testing"

	identityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/identity"
	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/plans"
)

func TestWritePreviewGolden(t *testing.T) {
	ref := &identityv1.DeploymentRef{
		Id: "dep-1", ScenarioId: "demo", Environment: "production",
		Target: &identityv1.TargetRef{MachineId: "machine-1", Transport: "bridge"},
	}
	compiled := &plansv1.CompilePlanResponse{
		PlanDigest: "sha256:plan", ClosureStatus: "unavailable: test",
		Plan: &plansv1.ExecutablePlan{
			Scope: "full", Outcome: OutcomeApply, DesiredRevision: 7,
			ReleaseDigest: "sha256:release", ConfigurationDigest: "sha256:config", ClosureDigest: "sha256:closure",
			Presentation: &plansv1.Presentation{Summary: "Deploy demo", DowntimeNote: "Brief outage", RecoveryNote: "Restore snapshot"},
			Handoff:      &plansv1.Handoff{Owner: "deployments", Kind: "resume", Reference: "op-1", Missing: []string{"token", "domain"}},
		},
		Preview: &plansv1.Preview{
			Target: "machine:machine-1", RecoveryStrategy: "restore last snapshot",
			Changes:      []*plansv1.Change{{ActionId: "install", Effect: "host_write", Summary: "Install bundle", Capability: "runtime", Verification: "health check", Recovery: "restore", Retry: "safe_replay", CancelPoint: true}},
			DataEffects:  []*plansv1.DataEffect{{Subject: "database", Effect: "preserve", ActionId: "backup"}},
			Downtime:     &plansv1.Downtime{ExpectedSeconds: 12, Reason: "restart"},
			Advisories:   []*plansv1.Advisory{{Summary: "Capability may be unavailable", Capability: "mailer", Detail: "No credential found", Hint: "Configure mailer"}},
			ShellPreview: []*plansv1.ShellPreviewLine{{ActionId: "install", Command: "echo derived"}},
		},
	}

	var got bytes.Buffer
	WritePreview(&got, ref, compiled, true)
	want := "" +
		"deployment: dep-1  scenario: demo  environment: production  target: machine:machine-1 (bridge)\n" +
		"plan digest: sha256:plan\n" +
		"scope: full  outcome: apply  desired revision: 7\n" +
		"release digest: sha256:release\n" +
		"configuration digest: sha256:config\n" +
		"closure digest: sha256:closure\n" +
		"closure: unavailable: test\n" +
		"target: machine:machine-1\n" +
		"summary: Deploy demo\n" +
		"changes:\n" +
		"   1. install  [host_write] Install bundle\n" +
		"      capability=runtime verify=health check recovery=restore retry=safe_replay cancel-point\n" +
		"data effects:\n" +
		"  database: preserve (backup)\n" +
		"downtime: ~12s (restart)\n" +
		"recovery strategy: restore last snapshot\n" +
		"downtime note: Brief outage\n" +
		"recovery note: Restore snapshot\n" +
		"warnings (1, non-blocking):\n" +
		"  ! Capability may be unavailable\n" +
		"      capability: mailer\n" +
		"      detail: No credential found\n" +
		"      fix: Configure mailer\n" +
		"handoff: deployments (resume) op-1\n" +
		"  missing: token, domain\n" +
		"shell preview (derived; execution uses typed verbs, not these strings):\n" +
		"  [install] echo derived\n"
	if got.String() != want {
		t.Fatalf("WritePreview output changed:\n--- got ---\n%s--- want ---\n%s", got.String(), want)
	}
}
