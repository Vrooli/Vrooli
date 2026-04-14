package resourceapp

import (
	"io"
	"testing"

	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/resources"
)

type fakeResourceOps struct {
	toggledName    string
	toggledEnabled bool
	syncCalls      int
}

func (f *fakeResourceOps) Discover() ([]resources.Resource, error) {
	return []resources.Resource{{Name: "redis"}}, nil
}

func (f *fakeResourceOps) ValidateResources(name string) (resources.ResourceValidationReport, error) {
	return resources.ResourceValidationReport{Passed: true}, nil
}

func (f *fakeResourceOps) ListStatuses(fast bool, includeDisabled bool) ([]resources.Status, error) {
	return []resources.Status{{Resource: resources.Resource{Name: "redis"}}}, nil
}

func (f *fakeResourceOps) Status(name string, fast bool) (resources.Status, error) {
	return resources.Status{Resource: resources.Resource{Name: name}}, nil
}
func (f *fakeResourceOps) Run(name string, args []string, stdout, stderr io.Writer) error { return nil }
func (f *fakeResourceOps) SetEnabled(name string, enabled bool) error {
	f.toggledName = name
	f.toggledEnabled = enabled
	return nil
}

func (f *fakeResourceOps) StartAll(stdout, stderr io.Writer) (control.StartReport, error) {
	return control.StartReport{Started: []control.ResultItem{{Name: "redis"}}}, nil
}

func (f *fakeResourceOps) StopAll(stdout, stderr io.Writer) (control.StopReport, error) {
	return control.StopReport{}, nil
}

func (f *fakeResourceOps) DeprecateResource(name string) (resources.DeprecationReport, error) {
	return resources.DeprecationReport{}, nil
}

func (f *fakeResourceOps) ListDeprecatedResources() ([]resources.DeprecatedResource, error) {
	return nil, nil
}

func (f *fakeResourceOps) RestoreDeprecatedResource(name string) (resources.RestoreReport, error) {
	return resources.RestoreReport{}, nil
}

func (f *fakeResourceOps) ArchiveResourceToBlueprint(name string) (resources.BlueprintArchiveReport, error) {
	return resources.BlueprintArchiveReport{}, nil
}

func (f *fakeResourceOps) ListBlueprintArchivedResources() ([]resources.BlueprintArchivedResource, error) {
	return nil, nil
}

func (f *fakeResourceOps) RestoreBlueprintArchivedResource(name string) (resources.BlueprintRestoreReport, error) {
	return resources.BlueprintRestoreReport{}, nil
}
func (f *fakeResourceOps) ListBlueprints() ([]resources.Blueprint, error) { return nil, nil }
func (f *fakeResourceOps) Blueprint(name string) (resources.Blueprint, error) {
	return resources.Blueprint{Name: name}, nil
}

func (f *fakeResourceOps) SearchBlueprints(query string) ([]resources.Blueprint, error) {
	return nil, nil
}

func (f *fakeResourceOps) ValidateBlueprints() (resources.BlueprintValidationReport, error) {
	return resources.BlueprintValidationReport{}, nil
}

func (f *fakeResourceOps) ListResourceTemplates() ([]resources.ResourceTemplateInfo, error) {
	return nil, nil
}

func (f *fakeResourceOps) ResourceTemplate(name string) (resources.ResourceTemplateInfo, error) {
	return resources.ResourceTemplateInfo{Name: name}, nil
}

func (f *fakeResourceOps) ValidateResourceTemplates() (resources.ResourceTemplateValidationReport, error) {
	return resources.ResourceTemplateValidationReport{}, nil
}

func (f *fakeResourceOps) GenerateResourceTemplate(req resources.ResourceTemplateGenerateRequest) (resources.ResourceTemplateGenerateReport, error) {
	return resources.ResourceTemplateGenerateReport{}, nil
}

func (f *fakeResourceOps) ValidateSchemaArtifacts() (resources.ResourceSchemaValidationReport, error) {
	return resources.ResourceSchemaValidationReport{Passed: true}, nil
}

func (f *fakeResourceOps) SyncSchemaArtifacts() (resources.ResourceSchemaSyncReport, error) {
	f.syncCalls++
	return resources.ResourceSchemaSyncReport{Passed: true}, nil
}

func TestServiceUsesInterfaceBasedResourceOperations(t *testing.T) {
	ops := &fakeResourceOps{}
	svc := Service{Resources: ops}

	if err := svc.Toggle(ToggleRequest{Name: "redis", Enabled: true}); err != nil {
		t.Fatalf("Toggle: %v", err)
	}
	if ops.toggledName != "redis" || !ops.toggledEnabled {
		t.Fatalf("toggle = %q %t", ops.toggledName, ops.toggledEnabled)
	}

	report, err := svc.StartAll()
	if err != nil {
		t.Fatalf("StartAll: %v", err)
	}
	if report.Start == nil || len(report.Start.Started) != 1 {
		t.Fatalf("report = %#v", report)
	}
}

func TestServiceSyncsSchemaAfterCatalogMutations(t *testing.T) {
	ops := &fakeResourceOps{}
	svc := Service{Resources: ops}

	if _, err := svc.ArchiveToBlueprint("redis"); err != nil {
		t.Fatalf("ArchiveToBlueprint: %v", err)
	}
	if _, err := svc.RestoreBlueprint("redis"); err != nil {
		t.Fatalf("RestoreBlueprint: %v", err)
	}
	if _, err := svc.TemplateGenerate(resources.ResourceTemplateGenerateRequest{}); err != nil {
		t.Fatalf("TemplateGenerate: %v", err)
	}
	if ops.syncCalls != 3 {
		t.Fatalf("sync calls = %d, want 3", ops.syncCalls)
	}
}
