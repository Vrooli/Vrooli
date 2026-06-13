package projectapp

import (
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/project"
	"github.com/vrooli/vrooli/internal/templatevalidation"
)

type ProjectOperations interface {
	Status(project.StatusOptions) (project.StatusReport, error)
	Doctor() (project.DoctorReport, error)
	Stop(project.StopOptions) (control.StopReport, error)
}

type MaintenanceOperations interface {
	ListOrphans() ([]maintenance.SystemProcess, error)
	KillOrphans() (control.StopReport, error)
	ListRuntimeClaims() ([]maintenance.RuntimeClaimInfo, error)
	CleanStaleLocks() (control.StopReport, error)
	DiagnosePort(port int, scenarioName string) (maintenance.PortDiagnostic, error)
	ListTemplateValidationRuns(templatevalidation.CleanupOptions) (templatevalidation.CleanupResult, error)
	CleanTemplateValidationRuns(templatevalidation.CleanupOptions) (templatevalidation.CleanupResult, error)
}

type StatusRequest struct {
	ResourcesOnly bool
	ScenariosOnly bool
	Fast          bool
}

type StopRequest struct {
	Targets []string
}

type OrphansRequest struct {
	Kill   bool
	DryRun bool
}

type LocksRequest struct {
	Clean bool
}

type DiagnosePortRequest struct {
	Port         int
	ScenarioName string
}

type TemplateValidationCleanupRequest struct {
	DryRun          bool
	OlderThan       string
	IncludeRetained bool
	RunID           string
}

type OrphansResponse struct {
	List       []maintenance.SystemProcess
	KillReport *control.StopReport
	DryRun     bool
}

type LocksResponse struct {
	RuntimeClaims []maintenance.RuntimeClaimInfo
	CleanReport   *control.StopReport
}

type Service struct {
	Project     ProjectOperations
	Maintenance MaintenanceOperations
}

func (s Service) Status(req StatusRequest) (project.StatusReport, error) {
	return s.Project.Status(project.StatusOptions{
		ResourcesOnly: req.ResourcesOnly,
		ScenariosOnly: req.ScenariosOnly,
		Fast:          req.Fast,
	})
}

func (s Service) Doctor() (project.DoctorReport, error) {
	return s.Project.Doctor()
}

func (s Service) Stop(req StopRequest) (control.StopReport, error) {
	return s.Project.Stop(project.StopOptions{Args: append([]string(nil), req.Targets...)})
}

func (s Service) Orphans(req OrphansRequest) (OrphansResponse, error) {
	if req.Kill && req.DryRun {
		items, err := s.Maintenance.ListOrphans()
		if err != nil {
			return OrphansResponse{}, err
		}
		return OrphansResponse{List: items, DryRun: true}, nil
	}
	if req.Kill {
		report, err := s.Maintenance.KillOrphans()
		if err != nil {
			return OrphansResponse{}, err
		}
		return OrphansResponse{KillReport: &report}, nil
	}
	items, err := s.Maintenance.ListOrphans()
	if err != nil {
		return OrphansResponse{}, err
	}
	return OrphansResponse{List: items}, nil
}

func (s Service) Locks(req LocksRequest) (LocksResponse, error) {
	if req.Clean {
		report, err := s.Maintenance.CleanStaleLocks()
		if err != nil {
			return LocksResponse{}, err
		}
		return LocksResponse{CleanReport: &report}, nil
	}
	claims, err := s.Maintenance.ListRuntimeClaims()
	if err != nil {
		return LocksResponse{}, err
	}
	return LocksResponse{RuntimeClaims: claims}, nil
}

func (s Service) DiagnosePort(req DiagnosePortRequest) (maintenance.PortDiagnostic, error) {
	return s.Maintenance.DiagnosePort(req.Port, req.ScenarioName)
}

func (s Service) TemplateValidationCleanup(opts templatevalidation.CleanupOptions) (templatevalidation.CleanupResult, error) {
	if opts.DryRun {
		return s.Maintenance.ListTemplateValidationRuns(opts)
	}
	return s.Maintenance.CleanTemplateValidationRuns(opts)
}
