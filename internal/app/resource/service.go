package resourceapp

import (
	"io"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/discovery"
	"github.com/vrooli/vrooli/internal/resources"
)

type ResourceOperations interface {
	Discover() ([]resources.Resource, error)
	DiscoverReport() (resources.DiscoveryReport, error)
	ValidateResources(name string) (resources.ResourceValidationReport, error)
	ListStatuses(fast bool, includeDisabled bool) ([]resources.Status, error)
	ListStatusesReport(fast bool, includeDisabled bool) (resources.StatusReport, error)
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

type ListResponse struct {
	Items    []resources.Resource
	Failures []discovery.Failure
}

type StatusResponse struct {
	Items    []resources.Status
	Item     *resources.Status
	Failures []discovery.Failure
}

func (s Service) List() (ListResponse, error) {
	report, err := s.Resources.DiscoverReport()
	if err != nil {
		return ListResponse{}, err
	}
	return ListResponse{
		Items:    report.Items,
		Failures: append([]discovery.Failure(nil), report.Failures...),
	}, nil
}

func (s Service) Validate(name string) (resources.ResourceValidationReport, error) {
	return s.Resources.ValidateResources(name)
}

func (s Service) Status(name string, fast bool) (StatusResponse, error) {
	if name == "" {
		report, err := s.Resources.ListStatusesReport(fast, false)
		if err != nil {
			return StatusResponse{}, err
		}
		return StatusResponse{
			Items:    report.Items,
			Failures: append([]discovery.Failure(nil), report.Failures...),
		}, nil
	}
	item, err := s.Resources.Status(name, fast)
	if err != nil {
		return StatusResponse{}, err
	}
	return StatusResponse{Item: &item}, nil
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

func (s Service) SchemaValidate() (resources.ResourceSchemaValidationReport, error) {
	return s.Resources.ValidateSchemaArtifacts()
}

func (s Service) SchemaSync() (resources.ResourceSchemaSyncReport, error) {
	return s.Resources.SyncSchemaArtifacts()
}
