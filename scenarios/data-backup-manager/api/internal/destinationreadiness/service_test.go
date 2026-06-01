package destinationreadiness_test

import (
	"context"
	"errors"
	"testing"

	"data-backup-manager/internal/destinationreadiness"
)

type fakeInspector struct {
	inspection destinationreadiness.Inspection
	calls      int
}

func (f *fakeInspector) Inspect(_ context.Context, _ string) (destinationreadiness.Inspection, error) {
	f.calls++
	return f.inspection, nil
}

type fakePreparer struct {
	supported map[destinationreadiness.PreparationAction]bool
	reason    string
	calls     int
	plans     []destinationreadiness.Plan
}

func (f *fakePreparer) Supported(action destinationreadiness.PreparationAction) (bool, string) {
	if f.supported == nil {
		return false, f.reason
	}
	if f.supported[action] {
		return true, ""
	}
	if f.reason != "" {
		return false, f.reason
	}
	return false, "unsupported by fake"
}

func (f *fakePreparer) Execute(_ context.Context, plan destinationreadiness.Plan) error {
	f.calls++
	f.plans = append(f.plans, plan)
	return nil
}

func identity() destinationreadiness.DeviceIdentity {
	return destinationreadiness.DeviceIdentity{
		DevicePath: "/dev/sdz1",
		Mountpoint: "/media/user/USB",
		Label:      "USB",
		Filesystem: "vfat",
		TotalBytes: 32 << 30,
		Model:      "TestDrive",
		Serial:     "serial-1",
		UUID:       "uuid-1",
	}
}

func inspection() destinationreadiness.Inspection {
	return destinationreadiness.Inspection{
		Identity:       identity(),
		FreeBytes:      20 << 30,
		Removable:      true,
		DriveClass:     "removable",
		NonEmptyRoot:   true,
		InstallerMedia: true,
		Platform:       "linux",
	}
}

func TestAnalyzeIsReadOnlyAndReportsWarnings(t *testing.T) {
	inspector := &fakeInspector{inspection: inspection()}
	preparer := &fakePreparer{supported: map[destinationreadiness.PreparationAction]bool{
		destinationreadiness.ActionCreateSubdir: true,
	}}
	svc := destinationreadiness.NewService(inspector, preparer)

	report, err := svc.Analyze(context.Background(), destinationreadiness.AnalyzeInput{
		Location:            "/media/user/USB",
		SelectedTargetBytes: 1 << 30,
		RetentionCopies:     2,
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if preparer.calls != 0 {
		t.Fatalf("Analyze must not execute preparation, got %d calls", preparer.calls)
	}
	if inspector.calls != 1 {
		t.Fatalf("expected one inspection, got %d", inspector.calls)
	}
	if report.RecommendedDestinationLocation != "/media/user/USB/vrooli-backups" {
		t.Fatalf("recommended location = %q", report.RecommendedDestinationLocation)
	}
	wantCodes := map[string]destinationreadiness.CheckSeverity{
		"filesystem_suitability":   destinationreadiness.SeverityWarning,
		"root_non_empty":           destinationreadiness.SeverityWarning,
		"installer_media_detected": destinationreadiness.SeverityWarning,
	}
	for code, want := range wantCodes {
		if got := severity(report.Checks, code); got != want {
			t.Fatalf("check %s severity = %s, want %s (checks=%+v)", code, got, want, report.Checks)
		}
	}
}

func TestPlanGenerationDoesNotExecute(t *testing.T) {
	inspector := &fakeInspector{inspection: inspection()}
	preparer := &fakePreparer{supported: map[destinationreadiness.PreparationAction]bool{
		destinationreadiness.ActionCreateSubdir: true,
	}}
	svc := destinationreadiness.NewService(inspector, preparer)

	plan, err := svc.PlanPreparation(context.Background(), destinationreadiness.PlanInput{
		Location:       "/media/user/USB",
		Action:         destinationreadiness.ActionCreateSubdir,
		DesiredSubdir:  "vrooli-backups",
		ExpectedDevice: identity(),
	})
	if err != nil {
		t.Fatalf("PlanPreparation returned error: %v", err)
	}
	if plan.ID == "" || plan.ConfirmationPhrase == "" {
		t.Fatalf("expected stable plan id and confirmation phrase, got %+v", plan)
	}
	if preparer.calls != 0 {
		t.Fatalf("plan generation must not execute preparation, got %d calls", preparer.calls)
	}
}

func TestExecuteRefusesMissingConfirmation(t *testing.T) {
	svc, preparer, plan := plannedService(t, destinationreadiness.ActionFormat)
	_, err := svc.ExecutePreparation(context.Background(), destinationreadiness.ExecuteInput{
		Plan:                plan,
		DryRun:              false,
		AcknowledgeDataLoss: true,
	})
	if !refused(err) {
		t.Fatalf("expected refusal for missing confirmation, got %v", err)
	}
	if preparer.calls != 0 {
		t.Fatalf("preparer must not run after refused confirmation, got %d calls", preparer.calls)
	}
}

func TestExecuteRefusesDestructiveActionWithoutDataLossAcknowledgement(t *testing.T) {
	svc, preparer, plan := plannedService(t, destinationreadiness.ActionFormat)
	_, err := svc.ExecutePreparation(context.Background(), destinationreadiness.ExecuteInput{
		Plan:         plan,
		Confirmation: plan.ConfirmationPhrase,
		DryRun:       false,
	})
	if !refused(err) {
		t.Fatalf("expected refusal for missing acknowledgement, got %v", err)
	}
	if preparer.calls != 0 {
		t.Fatalf("preparer must not run without acknowledgement, got %d calls", preparer.calls)
	}
}

func TestExecuteRefusesStaleDeviceIdentity(t *testing.T) {
	ins := inspection()
	inspector := &fakeInspector{inspection: ins}
	preparer := &fakePreparer{supported: map[destinationreadiness.PreparationAction]bool{
		destinationreadiness.ActionCreateSubdir: true,
	}}
	svc := destinationreadiness.NewService(inspector, preparer)
	plan, err := svc.PlanPreparation(context.Background(), destinationreadiness.PlanInput{
		Location:       "/media/user/USB",
		Action:         destinationreadiness.ActionCreateSubdir,
		ExpectedDevice: identity(),
	})
	if err != nil {
		t.Fatalf("PlanPreparation returned error: %v", err)
	}
	changed := ins
	changed.Identity.Serial = "different"
	inspector.inspection = changed

	_, err = svc.ExecutePreparation(context.Background(), destinationreadiness.ExecuteInput{
		Plan:         plan,
		Confirmation: plan.ConfirmationPhrase,
		DryRun:       false,
	})
	if !refused(err) {
		t.Fatalf("expected stale identity refusal, got %v", err)
	}
	if preparer.calls != 0 {
		t.Fatalf("preparer must not run after identity drift, got %d calls", preparer.calls)
	}
}

func TestPlanRefusesProtectedOverlap(t *testing.T) {
	inspector := &fakeInspector{inspection: inspection()}
	preparer := &fakePreparer{supported: map[destinationreadiness.PreparationAction]bool{
		destinationreadiness.ActionCreateSubdir: true,
	}}
	svc := destinationreadiness.NewService(inspector, preparer)

	_, err := svc.PlanPreparation(context.Background(), destinationreadiness.PlanInput{
		Location:       "/media/user/USB",
		Action:         destinationreadiness.ActionCreateSubdir,
		DesiredSubdir:  "vrooli-backups",
		ExpectedDevice: identity(),
		ProtectedPaths: []string{"/media/user/USB/vrooli-backups/source"},
	})
	if !refused(err) {
		t.Fatalf("expected protected-overlap refusal, got %v", err)
	}
	if preparer.calls != 0 {
		t.Fatalf("preparer must not run after protected-overlap refusal, got %d calls", preparer.calls)
	}
}

func TestUnsupportedPlatformReturnsStructuredUnsupportedPlan(t *testing.T) {
	inspector := &fakeInspector{inspection: inspection()}
	preparer := &fakePreparer{reason: "format is unsupported on this platform"}
	svc := destinationreadiness.NewService(inspector, preparer)

	plan, err := svc.PlanPreparation(context.Background(), destinationreadiness.PlanInput{
		Location:       "/media/user/USB",
		Action:         destinationreadiness.ActionFormat,
		DesiredFS:      "ext4",
		DesiredLabel:   "VROOLI_BACKUP",
		ExpectedDevice: identity(),
	})
	if err != nil {
		t.Fatalf("PlanPreparation returned error: %v", err)
	}
	if plan.Supported {
		t.Fatalf("expected unsupported plan, got %+v", plan)
	}
	if plan.UnsupportedReason == "" {
		t.Fatalf("expected unsupported reason, got %+v", plan)
	}

	_, err = svc.ExecutePreparation(context.Background(), destinationreadiness.ExecuteInput{
		Plan:                plan,
		Confirmation:        plan.ConfirmationPhrase,
		DryRun:              false,
		AcknowledgeDataLoss: true,
	})
	if !refused(err) {
		t.Fatalf("expected unsupported execution refusal, got %v", err)
	}
}

func TestExecuteCallsFakePreparerOnlyAfterAllGuardsPass(t *testing.T) {
	svc, preparer, plan := plannedService(t, destinationreadiness.ActionCreateSubdir)

	result, err := svc.ExecutePreparation(context.Background(), destinationreadiness.ExecuteInput{
		Plan:         plan,
		Confirmation: plan.ConfirmationPhrase,
		DryRun:       false,
	})
	if err != nil {
		t.Fatalf("ExecutePreparation returned error: %v", err)
	}
	if result.DryRun {
		t.Fatal("expected non-dry-run result")
	}
	if preparer.calls != 1 {
		t.Fatalf("expected exactly one fake preparer call, got %d", preparer.calls)
	}
	if len(preparer.plans) != 1 || preparer.plans[0].Action != destinationreadiness.ActionCreateSubdir {
		t.Fatalf("unexpected fake preparer plans: %+v", preparer.plans)
	}
}

func TestExecuteDryRunDoesNotCallPreparer(t *testing.T) {
	svc, preparer, plan := plannedService(t, destinationreadiness.ActionCreateSubdir)

	result, err := svc.ExecutePreparation(context.Background(), destinationreadiness.ExecuteInput{
		Plan:         plan,
		Confirmation: plan.ConfirmationPhrase,
		DryRun:       true,
	})
	if err != nil {
		t.Fatalf("ExecutePreparation dry-run returned error: %v", err)
	}
	if !result.DryRun {
		t.Fatal("expected dry-run result")
	}
	if preparer.calls != 0 {
		t.Fatalf("dry-run must not call preparer, got %d calls", preparer.calls)
	}
}

func plannedService(t *testing.T, action destinationreadiness.PreparationAction) (*destinationreadiness.Service, *fakePreparer, destinationreadiness.Plan) {
	t.Helper()
	inspector := &fakeInspector{inspection: inspection()}
	preparer := &fakePreparer{supported: map[destinationreadiness.PreparationAction]bool{action: true}}
	svc := destinationreadiness.NewService(inspector, preparer)
	plan, err := svc.PlanPreparation(context.Background(), destinationreadiness.PlanInput{
		Location:       "/media/user/USB",
		Action:         action,
		DesiredSubdir:  "vrooli-backups",
		DesiredFS:      "ext4",
		DesiredLabel:   "VROOLI_BACKUP",
		ExpectedDevice: identity(),
	})
	if err != nil {
		t.Fatalf("PlanPreparation returned error: %v", err)
	}
	return svc, preparer, plan
}

func severity(checks []destinationreadiness.CheckResult, code string) destinationreadiness.CheckSeverity {
	for _, c := range checks {
		if c.Code == code {
			return c.Severity
		}
	}
	return ""
}

func refused(err error) bool {
	var refused destinationreadiness.ErrPreparationRefused
	return errors.As(err, &refused)
}
