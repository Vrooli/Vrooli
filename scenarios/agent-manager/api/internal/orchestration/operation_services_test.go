package orchestration

import "testing"

func TestHandlerServicesExposeOnlyTheConcreteCapabilityImplementations(t *testing.T) {
	o := New(nil, nil, nil)
	services := NewHandlerServices(o)
	for name, got := range map[string]any{
		"profiles": services.ProfileService, "tasks": services.TaskService, "workflows": services.WorkflowService,
		"runs": services.RunService, "approval": services.ApprovalService, "events": services.EventService,
		"policy": services.PolicyService, "status": services.StatusService, "maintenance": services.MaintenanceService,
		"investigation-settings": services.InvestigationSettingsService, "orchestration-settings": services.OrchestrationSettingsService,
		"path-validation": services.PathValidationService, "identity": services.IdentityService, "project-root": services.ProjectRootService,
	} {
		if got != o {
			t.Fatalf("%s capability = %T, want orchestrator", name, got)
		}
	}

	empty := EmptyHandlerServices()
	if empty.ProfileService != nil || empty.RunService != nil || empty.StatusService != nil {
		t.Fatalf("empty handler services must not grant capabilities: %#v", empty)
	}
}
