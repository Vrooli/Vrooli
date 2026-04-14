package resourceapp

import (
	"io"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/resources"
)

type ResourceOperations interface {
	Discover() ([]resources.Resource, error)
	ValidateResources(name string) (resources.ResourceValidationReport, error)
	ListStatuses(fast bool, includeDisabled bool) ([]resources.Status, error)
	Status(name string, fast bool) (resources.Status, error)
	Run(name string, args []string, stdout, stderr io.Writer) error
	SetEnabled(name string, enabled bool) error
	StartAll(stdout, stderr io.Writer) (control.StartReport, error)
	StopAll(stdout, stderr io.Writer) (control.StopReport, error)
	DeprecateResource(name string) (resources.DeprecationReport, error)
	ListDeprecatedResources() ([]resources.DeprecatedResource, error)
	RestoreDeprecatedResource(name string) (resources.RestoreReport, error)
	ArchiveResourceToBlueprint(name string) (resources.BlueprintArchiveReport, error)
	ListBlueprintArchivedResources() ([]resources.BlueprintArchivedResource, error)
	RestoreBlueprintArchivedResource(name string) (resources.BlueprintRestoreReport, error)
	ListBlueprints() ([]resources.Blueprint, error)
	Blueprint(name string) (resources.Blueprint, error)
	SearchBlueprints(query string) ([]resources.Blueprint, error)
	ValidateBlueprints() (resources.BlueprintValidationReport, error)
	ListResourceTemplates() ([]resources.ResourceTemplateInfo, error)
	ResourceTemplate(name string) (resources.ResourceTemplateInfo, error)
	ValidateResourceTemplates() (resources.ResourceTemplateValidationReport, error)
	GenerateResourceTemplate(req resources.ResourceTemplateGenerateRequest) (resources.ResourceTemplateGenerateReport, error)
	ValidateSchemaArtifacts() (resources.ResourceSchemaValidationReport, error)
	SyncSchemaArtifacts() (resources.ResourceSchemaSyncReport, error)
}

type Service struct {
	Resources ResourceOperations
	Format    cliout.Format
	Stdout    io.Writer
	Stderr    io.Writer
}

type ControlModeRequest struct {
	Name      string
	Operation string
}

type ToggleRequest struct {
	Name    string
	Enabled bool
}

type ControlReportResponse struct {
	Start *control.StartReport
	Stop  *control.StopReport
}

func (s Service) List() ([]resources.Resource, error) {
	return s.Resources.Discover()
}

func (s Service) Validate(name string) (resources.ResourceValidationReport, error) {
	return s.Resources.ValidateResources(name)
}

func (s Service) Status(name string, fast bool) ([]resources.Status, *resources.Status, error) {
	if name == "" {
		items, err := s.Resources.ListStatuses(fast, false)
		return items, nil, err
	}
	item, err := s.Resources.Status(name, fast)
	if err != nil {
		return nil, nil, err
	}
	return nil, &item, nil
}

func (s Service) Info(name string) (resources.Status, error) {
	return s.Resources.Status(name, true)
}

func (s Service) Control(req ControlModeRequest) error {
	return s.Resources.Run(req.Name, []string{req.Operation}, s.Stdout, s.Stderr)
}

func (s Service) Toggle(req ToggleRequest) error {
	return s.Resources.SetEnabled(req.Name, req.Enabled)
}

func (s Service) StartAll() (ControlReportResponse, error) {
	report, err := s.Resources.StartAll(s.Stdout, s.Stderr)
	if err != nil {
		return ControlReportResponse{}, err
	}
	return ControlReportResponse{Start: &report}, nil
}

func (s Service) StopAll() (ControlReportResponse, error) {
	report, err := s.Resources.StopAll(s.Stdout, s.Stderr)
	if err != nil {
		return ControlReportResponse{}, err
	}
	return ControlReportResponse{Stop: &report}, nil
}

func (s Service) Deprecate(name string) (resources.DeprecationReport, error) {
	return s.Resources.DeprecateResource(name)
}

func (s Service) ListDeprecated() ([]resources.DeprecatedResource, error) {
	return s.Resources.ListDeprecatedResources()
}

func (s Service) Restore(name string) (resources.RestoreReport, error) {
	return s.Resources.RestoreDeprecatedResource(name)
}

func (s Service) ArchiveToBlueprint(name string) (resources.BlueprintArchiveReport, error) {
	report, err := s.Resources.ArchiveResourceToBlueprint(name)
	if err != nil {
		return resources.BlueprintArchiveReport{}, err
	}
	if _, err := s.Resources.SyncSchemaArtifacts(); err != nil {
		return resources.BlueprintArchiveReport{}, err
	}
	return report, nil
}

func (s Service) ListBlueprintArchived() ([]resources.BlueprintArchivedResource, error) {
	return s.Resources.ListBlueprintArchivedResources()
}

func (s Service) RestoreBlueprint(name string) (resources.BlueprintRestoreReport, error) {
	report, err := s.Resources.RestoreBlueprintArchivedResource(name)
	if err != nil {
		return resources.BlueprintRestoreReport{}, err
	}
	if _, err := s.Resources.SyncSchemaArtifacts(); err != nil {
		return resources.BlueprintRestoreReport{}, err
	}
	return report, nil
}

func (s Service) BlueprintList() ([]resources.Blueprint, error) {
	return s.Resources.ListBlueprints()
}

func (s Service) BlueprintInfo(name string) (resources.Blueprint, error) {
	return s.Resources.Blueprint(name)
}

func (s Service) BlueprintSearch(query string) ([]resources.Blueprint, error) {
	return s.Resources.SearchBlueprints(query)
}

func (s Service) BlueprintValidate() (resources.BlueprintValidationReport, error) {
	return s.Resources.ValidateBlueprints()
}

func (s Service) TemplateList() ([]resources.ResourceTemplateInfo, error) {
	return s.Resources.ListResourceTemplates()
}

func (s Service) TemplateShow(name string) (resources.ResourceTemplateInfo, error) {
	return s.Resources.ResourceTemplate(name)
}

func (s Service) TemplateValidate() (resources.ResourceTemplateValidationReport, error) {
	return s.Resources.ValidateResourceTemplates()
}

func (s Service) TemplateGenerate(req resources.ResourceTemplateGenerateRequest) (resources.ResourceTemplateGenerateReport, error) {
	report, err := s.Resources.GenerateResourceTemplate(req)
	if err != nil {
		return resources.ResourceTemplateGenerateReport{}, err
	}
	if !report.DryRun {
		if _, err := s.Resources.SyncSchemaArtifacts(); err != nil {
			return resources.ResourceTemplateGenerateReport{}, err
		}
	}
	return report, nil
}

func (s Service) SchemaValidate() (resources.ResourceSchemaValidationReport, error) {
	return s.Resources.ValidateSchemaArtifacts()
}

func (s Service) SchemaSync() (resources.ResourceSchemaSyncReport, error) {
	return s.Resources.SyncSchemaArtifacts()
}
