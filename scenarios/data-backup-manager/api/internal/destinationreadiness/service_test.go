package destinationreadiness_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/sysmounts"
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
		LocationExists:      true,
		LocationIsDirectory: true,
		Identity:            identity(),
		FreeBytes:           20 << 30,
		Removable:           true,
		DriveClass:          "removable",
		NonEmptyRoot:        true,
		InstallerMedia:      true,
		Platform:            "linux",
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

func TestAnalyzeWarnsForNTFS3WhenCrossPlatformFidelityIsUnproven(t *testing.T) {
	inspected := inspection()
	inspected.Identity.Filesystem = "ntfs3"
	inspected.NonEmptyRoot = false
	inspected.InstallerMedia = false
	inspector := &fakeInspector{inspection: inspected}
	svc := destinationreadiness.NewService(inspector, &fakePreparer{})

	report, err := svc.Analyze(context.Background(), destinationreadiness.AnalyzeInput{
		Location:              "/media/user/Elements",
		SelectedTargetBytes:   1 << 30,
		RetentionCopies:       2,
		CrossPlatformRequired: true,
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := severity(report.Checks, "filesystem_suitability"); got != destinationreadiness.SeverityWarning {
		t.Fatalf("filesystem_suitability severity = %s, want %s (checks=%+v)", got, destinationreadiness.SeverityWarning, report.Checks)
	}
}

func TestAnalyzeFailsClosedForDirtyOrNeedsCheckFilesystem(t *testing.T) {
	for _, state := range []sysmounts.FilesystemState{
		sysmounts.FilesystemStateDirty,
		sysmounts.FilesystemStateNeedsCheck,
	} {
		t.Run(string(state), func(t *testing.T) {
			inspected := inspection()
			inspected.FilesystemState = state
			report, err := destinationreadiness.NewService(&fakeInspector{inspection: inspected}, nil).Analyze(context.Background(), destinationreadiness.AnalyzeInput{Location: "/media/user/USB"})
			if err != nil {
				t.Fatalf("Analyze returned error: %v", err)
			}
			if got := severity(report.Checks, "destination_dirty"); got != destinationreadiness.SeverityFail {
				t.Fatalf("destination_dirty severity = %s, want %s", got, destinationreadiness.SeverityFail)
			}
			if report.OverallSeverity != destinationreadiness.SeverityFail {
				t.Fatalf("overall severity = %s, want %s", report.OverallSeverity, destinationreadiness.SeverityFail)
			}
		})
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

func TestAnalyzeMissingDestinationFailsClosed(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "missing")
	inspector := destinationreadiness.NewReadOnlyInspector(fakeVolumeScanner{volumes: []sysmounts.Volume{
		{DevicePath: "/dev/root", Mountpoint: root, Filesystem: "ext4", Class: sysmounts.ClassFixed, FreeBytes: 100, TotalBytes: 200},
	}})
	report, err := destinationreadiness.NewService(inspector, nil).Analyze(context.Background(), destinationreadiness.AnalyzeInput{Location: missing})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := severity(report.Checks, "destination_missing"); got != destinationreadiness.SeverityFail {
		t.Fatalf("destination_missing severity = %q, want fail", got)
	}
	if report.OverallSeverity != destinationreadiness.SeverityFail {
		t.Fatalf("overall severity = %q, want fail", report.OverallSeverity)
	}
}

func refused(err error) bool {
	var refused destinationreadiness.ErrPreparationRefused
	return errors.As(err, &refused)
}

// checkByCode finds one readiness check, failing the test when the rule that
// is supposed to produce it did not run at all.
func checkByCode(t *testing.T, report destinationreadiness.Report, code string) destinationreadiness.CheckResult {
	t.Helper()
	for _, c := range report.Checks {
		if c.Code == code {
			return c
		}
	}
	t.Fatalf("no %q check in report; got %+v", code, report.Checks)
	return destinationreadiness.CheckResult{}
}

func containsSubstring(values []string, want string) bool {
	for _, v := range values {
		if strings.Contains(v, want) {
			return true
		}
	}
	return false
}

// readOnlyInspection reproduces the Elements incident: an NTFS volume the
// driver refused to mount read/write because the dirty flag is set.
func readOnlyInspection() destinationreadiness.Inspection {
	in := inspection()
	in.Identity.Filesystem = "ntfs3"
	in.ReadOnly = true
	in.Mounted = true
	in.MountOptions = []string{"ro", "nosuid", "relatime"}
	in.FilesystemState = sysmounts.FilesystemStateDirty
	in.ReadOnlyCause = sysmounts.CauseFilesystemDirty
	in.StateEvidence = "/proc/fs/ntfs3/sdz1/volinfo"
	return in
}

func TestAnalyzeAttributesReadOnlyToTheDirtyFilesystem(t *testing.T) {
	svc := destinationreadiness.NewService(&fakeInspector{inspection: readOnlyInspection()}, &fakePreparer{})

	report, err := svc.Analyze(context.Background(), destinationreadiness.AnalyzeInput{Location: "/media/user/USB"})
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	mount := checkByCode(t, report, "mounted_read_write")
	if mount.Severity != destinationreadiness.SeverityFail {
		t.Fatalf("mounted_read_write severity = %q, want fail", mount.Severity)
	}
	if !strings.Contains(mount.Message, "dirty") {
		t.Fatalf("read-only message must name the cause, got %q", mount.Message)
	}
	if mount.NextAction == "" {
		t.Fatal("an attributed read-only failure must carry a next action")
	}

	dirty := checkByCode(t, report, "destination_dirty")
	if dirty.Severity != destinationreadiness.SeverityFail {
		t.Fatalf("destination_dirty severity = %q, want fail", dirty.Severity)
	}
	if !strings.Contains(dirty.Message, "volinfo") {
		t.Fatalf("dirty check must cite its evidence source, got %q", dirty.Message)
	}

	if report.ReadOnlyCause != sysmounts.CauseFilesystemDirty {
		t.Fatalf("report cause = %q, want %q", report.ReadOnlyCause, sysmounts.CauseFilesystemDirty)
	}
	if report.FilesystemState != sysmounts.FilesystemStateDirty {
		t.Fatalf("report filesystem state = %q, want dirty", report.FilesystemState)
	}
	if !containsSubstring(report.RepairSteps, "repair it under explicit confirmation") {
		t.Fatalf("repair steps must propose a confirmed repair, got %v", report.RepairSteps)
	}
}

// Repairing the filesystem of a write-protected device changes nothing. The
// report must not propose it, even though the volume is also read-only.
func TestAnalyzeDoesNotProposeRepairForWriteProtectedDevice(t *testing.T) {
	in := readOnlyInspection()
	in.DeviceWriteProtected = true
	in.ReadOnlyCause = sysmounts.CauseDeviceWriteProtected
	svc := destinationreadiness.NewService(&fakeInspector{inspection: in}, &fakePreparer{})

	report, err := svc.Analyze(context.Background(), destinationreadiness.AnalyzeInput{Location: "/media/user/USB"})
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	mount := checkByCode(t, report, "mounted_read_write")
	if !strings.Contains(mount.Message, "write-protected") {
		t.Fatalf("message = %q, want the write-protect cause", mount.Message)
	}
	if !containsSubstring(report.RepairSteps, "write protection") {
		t.Fatalf("repair steps must target write protection, got %v", report.RepairSteps)
	}
	if containsSubstring(report.RepairSteps, "repair it under explicit confirmation") {
		t.Fatalf("must not propose filesystem repair for a write-protected device, got %v", report.RepairSteps)
	}
}

// An unexplained read-only mount is the one case where doing nothing is right.
func TestAnalyzeRefusesToGuessAnUnattributedReadOnlyCause(t *testing.T) {
	in := readOnlyInspection()
	in.FilesystemState = sysmounts.FilesystemStateUnknown
	in.ReadOnlyCause = sysmounts.CauseUnknown
	in.StateEvidence = "mount-options"
	svc := destinationreadiness.NewService(&fakeInspector{inspection: in}, &fakePreparer{})

	report, err := svc.Analyze(context.Background(), destinationreadiness.AnalyzeInput{Location: "/media/user/USB"})
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if containsSubstring(report.RepairSteps, "repair it under explicit confirmation") {
		t.Fatalf("must not propose repair without an attributed cause, got %v", report.RepairSteps)
	}
	if !containsSubstring(report.RepairSteps, "attribute the read-only cause") {
		t.Fatalf("repair steps must demand attribution first, got %v", report.RepairSteps)
	}
}

func TestMatchesDeviceIgnoresMountpointChange(t *testing.T) {
	planned := identity()
	observed := identity()
	observed.Mountpoint = "" // the volume was unmounted by the remediation itself

	if planned.Matches(observed) {
		t.Fatal("path-scoped Matches is expected to reject an unmounted volume")
	}
	if !planned.MatchesDevice(observed) {
		t.Fatal("MatchesDevice must survive the unmount it authorized")
	}
}

func TestMatchesDeviceRejectsADifferentVolume(t *testing.T) {
	planned := identity()
	cases := map[string]func(*destinationreadiness.DeviceIdentity){
		"different uuid":       func(i *destinationreadiness.DeviceIdentity) { i.UUID = "uuid-2" },
		"different serial":     func(i *destinationreadiness.DeviceIdentity) { i.Serial = "serial-2" },
		"different size":       func(i *destinationreadiness.DeviceIdentity) { i.TotalBytes = 64 << 30 },
		"different filesystem": func(i *destinationreadiness.DeviceIdentity) { i.Filesystem = "ext4" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			observed := identity()
			observed.Mountpoint = ""
			mutate(&observed)
			if planned.MatchesDevice(observed) {
				t.Fatalf("MatchesDevice accepted a %s", name)
			}
		})
	}
}

// Without a UUID or serial there is nothing that survives a replug, so the
// device path must match exactly rather than being waved through as optional.
func TestMatchesDeviceRequiresDevicePathWithoutStrongIdentity(t *testing.T) {
	weak := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdz1", Filesystem: "vfat", TotalBytes: 32 << 30}
	same := weak
	if !weak.MatchesDevice(same) {
		t.Fatal("identical weak identities must match")
	}
	moved := weak
	moved.DevicePath = "/dev/sdy1"
	if weak.MatchesDevice(moved) {
		t.Fatal("weak identity must not match across a device-path change")
	}
	empty := destinationreadiness.DeviceIdentity{Filesystem: "vfat"}
	if empty.MatchesDevice(empty) {
		t.Fatal("an identity with no evidence at all must never match")
	}
}

// fakeDeviceAwareInspector answers both lenses, so a remediation flow can be
// exercised across the unmount it performs.
type fakeDeviceAwareInspector struct {
	path   destinationreadiness.Inspection
	device destinationreadiness.Inspection
	err    error
}

func (f *fakeDeviceAwareInspector) Inspect(context.Context, string) (destinationreadiness.Inspection, error) {
	if f.err != nil {
		return destinationreadiness.Inspection{}, f.err
	}
	return f.path, nil
}

func (f *fakeDeviceAwareInspector) InspectDevice(context.Context, destinationreadiness.DeviceIdentity) (destinationreadiness.Inspection, error) {
	return f.device, nil
}

type fakeRemediator struct {
	outcome destinationreadiness.RemediationOutcome
	err     error
	plans   []destinationreadiness.Plan
	dryRuns []bool
}

func (f *fakeRemediator) Supported(destinationreadiness.PreparationAction) (bool, string) {
	return true, ""
}

func (f *fakeRemediator) Remediate(_ context.Context, plan destinationreadiness.Plan, dryRun bool) (destinationreadiness.RemediationOutcome, error) {
	f.plans = append(f.plans, plan)
	f.dryRuns = append(f.dryRuns, dryRun)
	return f.outcome, f.err
}

func mountedDirtyInspection() destinationreadiness.Inspection {
	in := readOnlyInspection()
	in.Identity.DevicePath = "/dev/sdz1"
	in.Identity.UUID = "uuid-1"
	in.Identity.Serial = "serial-1"
	in.Identity.Mountpoint = "/media/user/USB"
	return in
}

func TestPlanRepairIsDestructiveAndNamesTheDisk(t *testing.T) {
	inspector := &fakeDeviceAwareInspector{path: mountedDirtyInspection()}
	svc := destinationreadiness.NewService(inspector, &fakePreparer{}).WithRemediator(&fakeRemediator{})

	plan, err := svc.PlanPreparation(context.Background(), destinationreadiness.PlanInput{
		Location: "/media/user/USB",
		Action:   destinationreadiness.ActionRepairFilesystem,
	})
	if err != nil {
		t.Fatalf("PlanPreparation: %v", err)
	}
	if !plan.Destructive {
		t.Fatal("repair must be classified destructive; it can discard filesystem metadata")
	}
	if !plan.RequiresConfirm || plan.ConfirmationPhrase == "" {
		t.Fatalf("plan = %+v, want a confirmation requirement", plan)
	}
	for _, want := range []string{"REPAIR", "/dev/sdz1", "uuid-1"} {
		if !strings.Contains(plan.ConfirmationPhrase, want) {
			t.Fatalf("confirmation phrase %q must contain %q so the operator sees which disk", plan.ConfirmationPhrase, want)
		}
	}
}

// Non-repair remediation steps change attachment, not content, and must not
// demand a data-loss acknowledgement the operator would learn to click through.
func TestPlanNonRepairRemediationIsNotDestructive(t *testing.T) {
	inspector := &fakeDeviceAwareInspector{path: mountedDirtyInspection()}
	svc := destinationreadiness.NewService(inspector, &fakePreparer{}).WithRemediator(&fakeRemediator{})

	for _, action := range []destinationreadiness.PreparationAction{
		destinationreadiness.ActionUnmount,
		destinationreadiness.ActionCheckFilesystem,
		destinationreadiness.ActionMountReadWrite,
	} {
		plan, err := svc.PlanPreparation(context.Background(), destinationreadiness.PlanInput{Location: "/media/user/USB", Action: action})
		if err != nil {
			t.Fatalf("PlanPreparation(%s): %v", action, err)
		}
		if plan.Destructive {
			t.Fatalf("%s must not be classified destructive", action)
		}
		if !plan.RequiresConfirm {
			t.Fatalf("%s must still require confirmation", action)
		}
	}
}

func TestPlanRemediationRefusesADiskItCannotIdentify(t *testing.T) {
	in := mountedDirtyInspection()
	in.Identity.UUID, in.Identity.Serial = "", ""
	svc := destinationreadiness.NewService(&fakeDeviceAwareInspector{path: in}, &fakePreparer{}).WithRemediator(&fakeRemediator{})

	_, err := svc.PlanPreparation(context.Background(), destinationreadiness.PlanInput{
		Location: "/media/user/USB", Action: destinationreadiness.ActionRepairFilesystem,
	})

	var refused destinationreadiness.ErrPreparationRefused
	if !errors.As(err, &refused) {
		t.Fatalf("error = %v, want a refusal", err)
	}
	if !strings.Contains(refused.Reason, "UUID or serial") {
		t.Fatalf("reason = %q", refused.Reason)
	}
}

// The step that runs after the unmount must not be rejected for the unmount it
// was authorized to perform.
func TestExecuteRemediationSurvivesItsOwnUnmount(t *testing.T) {
	unmounted := mountedDirtyInspection()
	unmounted.Mounted = false
	unmounted.LocationExists = false
	unmounted.LocationIsDirectory = false
	unmounted.Identity.Mountpoint = ""

	inspector := &fakeDeviceAwareInspector{path: mountedDirtyInspection(), device: unmounted}
	remediator := &fakeRemediator{outcome: destinationreadiness.RemediationOutcome{Status: "changed", Changed: true, Backend: "udisks2"}}
	svc := destinationreadiness.NewService(inspector, &fakePreparer{}).WithRemediator(remediator)

	plan, err := svc.PlanPreparation(context.Background(), destinationreadiness.PlanInput{
		Location: "/media/user/USB", Action: destinationreadiness.ActionRepairFilesystem,
	})
	if err != nil {
		t.Fatalf("PlanPreparation: %v", err)
	}

	result, err := svc.ExecutePreparation(context.Background(), destinationreadiness.ExecuteInput{
		Plan:                plan,
		Confirmation:        plan.ConfirmationPhrase,
		AcknowledgeDataLoss: true,
	})
	if err != nil {
		t.Fatalf("ExecutePreparation: %v", err)
	}
	if !result.Changed || result.Status != "changed" || result.Backend != "udisks2" {
		t.Fatalf("result = %+v, want the control-plane outcome carried through", result)
	}
	if len(remediator.plans) != 1 {
		t.Fatalf("remediator calls = %d", len(remediator.plans))
	}
}

func TestExecuteRemediationRequiresConfirmationAndAcknowledgement(t *testing.T) {
	inspector := &fakeDeviceAwareInspector{path: mountedDirtyInspection(), device: mountedDirtyInspection()}
	remediator := &fakeRemediator{outcome: destinationreadiness.RemediationOutcome{Status: "changed", Changed: true}}
	svc := destinationreadiness.NewService(inspector, &fakePreparer{}).WithRemediator(remediator)

	plan, err := svc.PlanPreparation(context.Background(), destinationreadiness.PlanInput{
		Location: "/media/user/USB", Action: destinationreadiness.ActionRepairFilesystem,
	})
	if err != nil {
		t.Fatalf("PlanPreparation: %v", err)
	}

	if _, err := svc.ExecutePreparation(context.Background(), destinationreadiness.ExecuteInput{
		Plan: plan, Confirmation: "WRONG", AcknowledgeDataLoss: true,
	}); err == nil {
		t.Fatal("a mismatched confirmation phrase must be refused")
	}
	if _, err := svc.ExecutePreparation(context.Background(), destinationreadiness.ExecuteInput{
		Plan: plan, Confirmation: plan.ConfirmationPhrase,
	}); err == nil {
		t.Fatal("a repair without a data-loss acknowledgement must be refused")
	}
	if len(remediator.plans) != 0 {
		t.Fatalf("the control plane was reached despite a failed gate: %v", remediator.plans)
	}
}

// A remediation plan for a disk that is no longer the approved one must not run,
// even though the mountpoint check is intentionally relaxed for these actions.
func TestExecuteRemediationRefusesASwappedDisk(t *testing.T) {
	swapped := mountedDirtyInspection()
	swapped.Identity.UUID = "uuid-2"
	swapped.Identity.Serial = "serial-2"

	inspector := &fakeDeviceAwareInspector{path: mountedDirtyInspection(), device: swapped}
	remediator := &fakeRemediator{}
	svc := destinationreadiness.NewService(inspector, &fakePreparer{}).WithRemediator(remediator)

	plan, err := svc.PlanPreparation(context.Background(), destinationreadiness.PlanInput{
		Location: "/media/user/USB", Action: destinationreadiness.ActionRepairFilesystem,
	})
	if err != nil {
		t.Fatalf("PlanPreparation: %v", err)
	}

	_, err = svc.ExecutePreparation(context.Background(), destinationreadiness.ExecuteInput{
		Plan: plan, Confirmation: plan.ConfirmationPhrase, AcknowledgeDataLoss: true,
	})
	if err == nil {
		t.Fatal("a swapped disk must be refused")
	}
	if len(remediator.plans) != 0 {
		t.Fatal("the control plane must not be reached for a swapped disk")
	}
}

// A dry run must reach the control plane as a dry run, not be skipped locally:
// the control plane's own gates are part of what a rehearsal is meant to prove.
func TestExecuteRemediationDryRunReachesTheControlPlaneAsADryRun(t *testing.T) {
	inspector := &fakeDeviceAwareInspector{path: mountedDirtyInspection(), device: mountedDirtyInspection()}
	remediator := &fakeRemediator{outcome: destinationreadiness.RemediationOutcome{Status: "verified"}}
	svc := destinationreadiness.NewService(inspector, &fakePreparer{}).WithRemediator(remediator)

	plan, err := svc.PlanPreparation(context.Background(), destinationreadiness.PlanInput{
		Location: "/media/user/USB", Action: destinationreadiness.ActionCheckFilesystem,
	})
	if err != nil {
		t.Fatalf("PlanPreparation: %v", err)
	}
	if _, err := svc.ExecutePreparation(context.Background(), destinationreadiness.ExecuteInput{
		Plan: plan, Confirmation: plan.ConfirmationPhrase, DryRun: true,
	}); err != nil {
		t.Fatalf("ExecutePreparation: %v", err)
	}
	if len(remediator.dryRuns) != 1 || !remediator.dryRuns[0] {
		t.Fatalf("dry run was not propagated: %v", remediator.dryRuns)
	}
}

// Regression: with the destination volume unmounted, its path is an ordinary
// empty directory on the root filesystem. Resolving "the volume that owns this
// path" then lands on the *root disk*, and the confirmation phrase names it —
// so a distracted operator could authorise a filesystem action against their
// system disk. Remediation must refuse instead of silently retargeting.
func TestPlanRemediationRefusesToRetargetToTheHostVolume(t *testing.T) {
	rootVolume := destinationreadiness.Inspection{
		LocationExists:      true,
		LocationIsDirectory: true,
		Mounted:             true,
		Identity: destinationreadiness.DeviceIdentity{
			DevicePath: "/dev/nvme0n1p2",
			Mountpoint: "/",
			Filesystem: "ext4",
			UUID:       "root-uuid",
			TotalBytes: 1966736678912,
		},
	}
	svc := destinationreadiness.NewService(&fakeDeviceAwareInspector{path: rootVolume}, &fakePreparer{}).
		WithRemediator(&fakeRemediator{})

	for _, action := range []destinationreadiness.PreparationAction{
		destinationreadiness.ActionUnmount,
		destinationreadiness.ActionCheckFilesystem,
		destinationreadiness.ActionRepairFilesystem,
		destinationreadiness.ActionMountReadWrite,
	} {
		t.Run(string(action), func(t *testing.T) {
			plan, err := svc.PlanPreparation(context.Background(), destinationreadiness.PlanInput{
				Location: "/media/user/Elements",
				Action:   action,
			})
			if err == nil {
				t.Fatalf("planned %s against the host volume: %+v", action, plan)
			}
			var refused destinationreadiness.ErrPreparationRefused
			if !errors.As(err, &refused) {
				t.Fatalf("error = %v, want a refusal", err)
			}
			if !strings.Contains(refused.Reason, "not mounted") {
				t.Fatalf("reason = %q, want it to name the absent destination volume", refused.Reason)
			}
		})
	}
}

// A remediation action must never be planned against a system volume, however
// the path resolved to it.
func TestPlanRemediationRefusesASystemVolume(t *testing.T) {
	system := destinationreadiness.Inspection{
		LocationExists:      true,
		LocationIsDirectory: true,
		Mounted:             true,
		Identity: destinationreadiness.DeviceIdentity{
			DevicePath: "/dev/nvme0n1p1",
			Mountpoint: "/boot/efi",
			Filesystem: "vfat",
			UUID:       "boot-uuid",
		},
	}
	svc := destinationreadiness.NewService(&fakeDeviceAwareInspector{path: system}, &fakePreparer{}).
		WithRemediator(&fakeRemediator{})

	_, err := svc.PlanPreparation(context.Background(), destinationreadiness.PlanInput{
		Location: "/boot/efi", Action: destinationreadiness.ActionRepairFilesystem,
	})
	var refused destinationreadiness.ErrPreparationRefused
	if !errors.As(err, &refused) {
		t.Fatalf("error = %v, want a refusal", err)
	}
	if !strings.Contains(refused.Reason, "system volume") {
		t.Fatalf("reason = %q", refused.Reason)
	}
}

// Once the destination is unmounted, the only way to keep addressing it is by
// device. Supplying the expected device must switch planning to the device lens
// instead of resolving whatever now owns the path.
func TestPlanRemediationUsesTheDeviceLensWhenTheVolumeIsUnmounted(t *testing.T) {
	unmounted := mountedDirtyInspection()
	unmounted.Mounted = false
	unmounted.LocationExists = false
	unmounted.LocationIsDirectory = false
	unmounted.Identity.Mountpoint = ""

	rootVolume := destinationreadiness.Inspection{
		LocationExists: true, LocationIsDirectory: true, Mounted: true,
		Identity: destinationreadiness.DeviceIdentity{DevicePath: "/dev/nvme0n1p2", Mountpoint: "/", Filesystem: "ext4", UUID: "root-uuid"},
	}
	svc := destinationreadiness.NewService(&fakeDeviceAwareInspector{path: rootVolume, device: unmounted}, &fakePreparer{}).
		WithRemediator(&fakeRemediator{})

	plan, err := svc.PlanPreparation(context.Background(), destinationreadiness.PlanInput{
		Location:       "/media/user/USB",
		Action:         destinationreadiness.ActionCheckFilesystem,
		ExpectedDevice: destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdz1"},
	})
	if err != nil {
		t.Fatalf("PlanPreparation: %v", err)
	}
	if plan.Identity.DevicePath != "/dev/sdz1" {
		t.Fatalf("plan targeted %q, want the supplied device", plan.Identity.DevicePath)
	}
	if !strings.Contains(plan.ConfirmationPhrase, "/dev/sdz1") {
		t.Fatalf("confirmation phrase = %q", plan.ConfirmationPhrase)
	}
}
