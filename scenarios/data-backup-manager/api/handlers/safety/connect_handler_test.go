package safety

import (
	"context"
	"testing"

	internalsafety "data-backup-manager/internal/safety"

	"connectrpc.com/connect"

	safetyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/safety"
)

type fakeService struct {
	dest        internalsafety.DestinationRef
	created     bool
	ensureErr   error
	backup      internalsafety.BackupResult
	backupErr   error
	register    internalsafety.RegisterTargetsResult
	registerErr error
	populate    internalsafety.PopulateShadowResult
	populateErr error
}

func (f *fakeService) EnsureSafetyDestination(context.Context, int64) (internalsafety.DestinationRef, bool, error) {
	return f.dest, f.created, f.ensureErr
}

func (f *fakeService) BackupScenarioNow(context.Context, string, int32) (internalsafety.BackupResult, error) {
	return f.backup, f.backupErr
}

func (f *fakeService) RegisterScenarioTargets(context.Context, string) (internalsafety.RegisterTargetsResult, error) {
	return f.register, f.registerErr
}

func (f *fakeService) PopulateShadow(context.Context, string, string, []internalsafety.ShadowMapping) (internalsafety.PopulateShadowResult, error) {
	return f.populate, f.populateErr
}

func TestEnsureSafetyDestination_MapsFieldsAndCreatedFlag(t *testing.T) {
	svc := &fakeService{
		dest: internalsafety.DestinationRef{
			ID:                 "dst-1",
			Name:               "baseline-safety",
			Location:           "/r/baseline-safety",
			RepositoryLocation: "/r/baseline-safety/repositories/baseline-safety.kopia",
		},
		created: true,
	}
	h := NewConnectHandler(Deps{Service: svc})

	resp, err := h.EnsureSafetyDestination(context.Background(), connect.NewRequest(&safetyv1.EnsureSafetyDestinationRequest{}))
	if err != nil {
		t.Fatalf("EnsureSafetyDestination: %v", err)
	}
	if resp.Msg.Destination == nil {
		t.Fatalf("no destination in response")
	}
	if got := resp.Msg.Destination.Id; got != "dst-1" {
		t.Fatalf("id = %q, want dst-1", got)
	}
	if got := resp.Msg.Destination.RepositoryLocation; got != svc.dest.RepositoryLocation {
		t.Fatalf("repository_location = %q, want %q", got, svc.dest.RepositoryLocation)
	}
	if !resp.Msg.Created {
		t.Fatalf("created = false, want true")
	}
}

func TestBackupScenarioNow_MapsResult(t *testing.T) {
	svc := &fakeService{
		backup: internalsafety.BackupResult{
			Run:           internalsafety.RunRef{ID: "run-1", PlanID: "plan-1", Status: "pending"},
			DestinationID: "dst-1",
			TargetCount:   3,
		},
	}
	h := NewConnectHandler(Deps{Service: svc})

	resp, err := h.BackupScenarioNow(context.Background(), connect.NewRequest(&safetyv1.BackupScenarioNowRequest{Scenario: "foo"}))
	if err != nil {
		t.Fatalf("BackupScenarioNow: %v", err)
	}
	m := resp.Msg
	if m.RunId != "run-1" || m.PlanId != "plan-1" || m.DestinationId != "dst-1" {
		t.Fatalf("ids = (%q,%q,%q), want (run-1,plan-1,dst-1)", m.RunId, m.PlanId, m.DestinationId)
	}
	if m.TargetCount != 3 {
		t.Fatalf("target_count = %d, want 3", m.TargetCount)
	}
	if m.Status != "pending" {
		t.Fatalf("status = %q, want pending", m.Status)
	}
}

func TestRegisterScenarioTargets_MapsRegisteredAndSkipped(t *testing.T) {
	svc := &fakeService{
		register: internalsafety.RegisterTargetsResult{
			Scenario: "alpha",
			Registered: []internalsafety.ScenarioTargetSpec{
				{Name: "postgres", Kind: "postgres", Locator: "vrooli_alpha"},
				{Name: "data", Kind: "filesystem", Locator: "/d/alpha"},
			},
			Skipped: []internalsafety.SkippedNote{{Kind: "redis, qdrant, sqlite", Reason: "not derivable"}},
		},
	}
	h := NewConnectHandler(Deps{Service: svc})

	resp, err := h.RegisterScenarioTargets(context.Background(), connect.NewRequest(&safetyv1.RegisterScenarioTargetsRequest{Scenario: "alpha"}))
	if err != nil {
		t.Fatalf("RegisterScenarioTargets: %v", err)
	}
	m := resp.Msg
	if m.Scenario != "alpha" {
		t.Fatalf("scenario = %q, want alpha", m.Scenario)
	}
	if len(m.Registered) != 2 {
		t.Fatalf("registered = %d, want 2", len(m.Registered))
	}
	if m.Registered[0].SourceKind != "postgres" || m.Registered[0].Locator != "vrooli_alpha" {
		t.Fatalf("registered[0] = %+v, want postgres/vrooli_alpha", m.Registered[0])
	}
	if len(m.Skipped) != 1 || m.Skipped[0].SourceKind != "redis, qdrant, sqlite" {
		t.Fatalf("skipped = %+v, want one non-derivable note", m.Skipped)
	}
}

func TestBackupScenarioNow_NoTargetsIsFailedPrecondition(t *testing.T) {
	svc := &fakeService{backupErr: internalsafety.ErrNoTargets}
	h := NewConnectHandler(Deps{Service: svc})

	_, err := h.BackupScenarioNow(context.Background(), connect.NewRequest(&safetyv1.BackupScenarioNowRequest{Scenario: "foo"}))
	if err == nil {
		t.Fatalf("want error")
	}
	if got := connect.CodeOf(err); got != connect.CodeFailedPrecondition {
		t.Fatalf("code = %v, want FailedPrecondition", got)
	}
}

func TestPopulateShadow_MapsRestoresAndSkipped(t *testing.T) {
	svc := &fakeService{
		populate: internalsafety.PopulateShadowResult{
			Scenario: "alpha",
			RunID:    "run-7",
			Restores: []internalsafety.ShadowRestoreRef{
				{TargetName: "postgres", TargetID: "t-pg", SnapshotID: "snap-pg", RestoreID: "restore-1", Location: "vrooli_alpha_shadow", Status: "requested"},
			},
			Skipped: []internalsafety.ShadowSkip{{TargetName: "redis", Reason: "no target named \"redis\""}},
		},
	}
	h := NewConnectHandler(Deps{Service: svc})

	resp, err := h.PopulateShadow(context.Background(), connect.NewRequest(&safetyv1.PopulateShadowRequest{
		Scenario: "alpha",
		Mappings: []*safetyv1.ShadowTargetMapping{{TargetName: "postgres", Location: "vrooli_alpha_shadow"}},
	}))
	if err != nil {
		t.Fatalf("PopulateShadow: %v", err)
	}
	m := resp.Msg
	if m.Scenario != "alpha" || m.RunId != "run-7" {
		t.Fatalf("scenario/run = %q/%q, want alpha/run-7", m.Scenario, m.RunId)
	}
	if len(m.Restores) != 1 || m.Restores[0].RestoreId != "restore-1" || m.Restores[0].Location != "vrooli_alpha_shadow" {
		t.Fatalf("restores wrong: %+v", m.Restores)
	}
	if len(m.Skipped) != 1 || m.Skipped[0].TargetName != "redis" {
		t.Fatalf("skipped wrong: %+v", m.Skipped)
	}
}

func TestPopulateShadow_NoSafetyBackupIsFailedPrecondition(t *testing.T) {
	svc := &fakeService{populateErr: internalsafety.ErrNoSafetyBackup}
	h := NewConnectHandler(Deps{Service: svc})

	_, err := h.PopulateShadow(context.Background(), connect.NewRequest(&safetyv1.PopulateShadowRequest{
		Scenario: "alpha",
		Mappings: []*safetyv1.ShadowTargetMapping{{TargetName: "postgres", Location: "x"}},
	}))
	if got := connect.CodeOf(err); got != connect.CodeFailedPrecondition {
		t.Fatalf("code = %v, want FailedPrecondition", got)
	}
}
