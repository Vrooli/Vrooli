package projectapp

import (
	"testing"

	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/project"
	"github.com/vrooli/vrooli/internal/templatevalidation"
)

type fakeProjectOps struct {
	statusReq project.StatusOptions
	stopReq   project.StopOptions
}

func (f *fakeProjectOps) Status(opts project.StatusOptions) (project.StatusReport, error) {
	f.statusReq = opts
	return project.StatusReport{Summary: map[string]int{"ok": 1}}, nil
}

func (f *fakeProjectOps) Doctor() (project.DoctorReport, error) {
	return project.DoctorReport{Checks: []project.DoctorCheck{{Name: "doctor", Status: "ok"}}}, nil
}

func (f *fakeProjectOps) Stop(opts project.StopOptions) (control.StopReport, error) {
	f.stopReq = opts
	return control.StopReport{Message: "stopped"}, nil
}

type fakeMaintenanceOps struct{}

func (fakeMaintenanceOps) ListOrphans() ([]maintenance.SystemProcess, error) {
	return []maintenance.SystemProcess{{PID: 1, Command: "demo"}}, nil
}

func (fakeMaintenanceOps) KillOrphans() (control.StopReport, error) {
	return control.StopReport{Message: "killed"}, nil
}

func (fakeMaintenanceOps) ListRuntimeClaims() ([]maintenance.RuntimeClaimInfo, error) {
	return []maintenance.RuntimeClaimInfo{{Port: 8080, Scenario: "demo", ClaimStatus: "bound"}}, nil
}

func (fakeMaintenanceOps) CleanStaleLocks() (control.StopReport, error) {
	return control.StopReport{Message: "cleaned"}, nil
}

func (fakeMaintenanceOps) DiagnosePort(port int, scenarioName string) (maintenance.PortDiagnostic, error) {
	return maintenance.PortDiagnostic{Port: port, Scenario: scenarioName, InUse: true}, nil
}

func (fakeMaintenanceOps) ListTemplateValidationRuns(opts templatevalidation.CleanupOptions) (templatevalidation.CleanupResult, error) {
	return templatevalidation.CleanupResult{CleanupPlan: templatevalidation.CleanupPlan{DryRun: true}}, nil
}

func (fakeMaintenanceOps) CleanTemplateValidationRuns(opts templatevalidation.CleanupOptions) (templatevalidation.CleanupResult, error) {
	return templatevalidation.CleanupResult{CleanupPlan: templatevalidation.CleanupPlan{DryRun: opts.DryRun}}, nil
}

func TestServiceStatusUsesProjectOperations(t *testing.T) {
	projectOps := &fakeProjectOps{}
	svc := Service{Project: projectOps, Maintenance: fakeMaintenanceOps{}}

	report, err := svc.Status(StatusRequest{ResourcesOnly: true, Fast: true})
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if report.Summary["ok"] != 1 {
		t.Fatalf("report = %#v", report)
	}
	if !projectOps.statusReq.ResourcesOnly || !projectOps.statusReq.Fast {
		t.Fatalf("status req = %#v", projectOps.statusReq)
	}
}

func TestServiceRoutesMaintenanceUseCases(t *testing.T) {
	svc := Service{Project: &fakeProjectOps{}, Maintenance: fakeMaintenanceOps{}}

	orphans, err := svc.Orphans(OrphansRequest{})
	if err != nil {
		t.Fatalf("Orphans: %v", err)
	}
	if len(orphans.List) != 1 || orphans.List[0].PID != 1 {
		t.Fatalf("orphans = %#v", orphans)
	}

	locks, err := svc.Locks(LocksRequest{Clean: true})
	if err != nil {
		t.Fatalf("Locks: %v", err)
	}
	if locks.CleanReport == nil || locks.CleanReport.Message != "cleaned" {
		t.Fatalf("locks = %#v", locks)
	}
}
