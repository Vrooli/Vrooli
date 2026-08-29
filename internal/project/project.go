package project

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/discovery"
	"github.com/vrooli/vrooli/internal/dockerhost"
	"github.com/vrooli/vrooli/internal/hostreqcheck"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/privilegebroker"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/vroolierr"
)

const (
	projectParameterA      = 3
	projectParameterB      = 400
	projectParameterC      = 8
	projectSeverityWarning = "warning"
)

type Controller struct {
	Root                  string
	Home                  string
	Stdout                io.Writer
	Stderr                io.Writer
	Resources             ResourceController
	Scenarios             ScenarioController
	Maintenance           MaintenanceController
	HostReqValidateFn     func(string, string) (hostreqcheck.Report, error)
	MaintenanceSnapshotFn func() (maintenance.ProcessSnapshot, error)
	RepairRuntimeHomeFn   func() DoctorCheck
	LookPathFn            func(string) (string, error)
	NewPhaseRunner        func(root, home string, stdout, stderr io.Writer) (PhaseRunner, error)
}

type Dependencies struct {
	Resources   ResourceController
	Scenarios   ScenarioController
	Maintenance MaintenanceController
}

type ResourceController interface {
	ListStatuses(fast bool, includeDisabled bool) ([]resources.Status, error)
	StopAll(stdout, stderr io.Writer) (control.StopReport, error)
	Run(name string, args []string, stdout, stderr io.Writer) error
}

type ScenarioController interface {
	List() ([]orchestrator.ScenarioView, error)
	Status(name string) (orchestrator.ScenarioView, bool, error)
	StopAll() (control.StopReport, error)
	Stop(name string, opts lifecycle.StopOptions) error
}

type MaintenanceController interface {
	Snapshot() (maintenance.ProcessSnapshot, error)
}

type PhaseRunner interface {
	RunPhase(name, phase string, opts lifecycle.PhaseOptions) error
}

type DoctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type DoctorReport struct {
	Checks []DoctorCheck `json:"checks"`
}

type DoctorOptions struct {
	RepairFilePermissions bool
}

type StatusOptions struct {
	ResourcesOnly bool
	ScenariosOnly bool
	Fast          bool
}

type StatusReport struct {
	Resources   []resources.Status           `json:"resources,omitempty"`
	Scenarios   []orchestrator.ScenarioView  `json:"scenarios,omitempty"`
	Maintenance *maintenance.ProcessSnapshot `json:"maintenance,omitempty"`
	Summary     map[string]int               `json:"summary"`
}

type StopOptions struct {
	Target  string
	Args    []string
	DryRun  bool
	Verbose bool
}

func New(root, home string, stdout, stderr io.Writer) *Controller {
	return NewWithDependencies(root, home, stdout, stderr, Dependencies{})
}

func NewWithDependencies(root, home string, stdout, stderr io.Writer, deps Dependencies) *Controller {
	cleanRoot := filepath.Clean(root)
	cleanHome := filepath.Clean(home)
	if deps.Resources == nil {
		deps.Resources = resources.NewController(cleanRoot, cleanHome)
	}
	if deps.Scenarios == nil {
		deps.Scenarios = orchestrator.New(cleanRoot, cleanHome, stdout, stderr)
	}
	if deps.Maintenance == nil {
		deps.Maintenance = maintenance.NewController(cleanRoot, cleanHome)
	}
	return &Controller{
		Root:              cleanRoot,
		Home:              cleanHome,
		Stdout:            stdout,
		Stderr:            stderr,
		Resources:         deps.Resources,
		Scenarios:         deps.Scenarios,
		Maintenance:       deps.Maintenance,
		HostReqValidateFn: hostreqcheck.Validate,
		LookPathFn:        shell.LookPath,
		NewPhaseRunner: func(root, home string, stdout, stderr io.Writer) (PhaseRunner, error) {
			return lifecycle.NewRunner(root, home, stdout, stderr)
		},
	}
}

func (c *Controller) RunProjectPhase(phase string, args []string) error {
	switch strings.TrimSpace(phase) {
	case "setup", "develop", "build":
		return &vroolierr.Error{
			Code:       "project_phase_native_only",
			Category:   "Usage",
			HTTPStatus: projectParameterB,
			Message:    fmt.Sprintf("project lifecycle phase %q is native-only and cannot run via the generic phase runner", phase),
			Hint:       fmt.Sprintf("Use `vrooli %s` instead of the generic project lifecycle endpoint.", phase),
		}
	}

	project, err := LoadProject(c.Root)
	if err != nil {
		return err
	}
	if !phaseDefined(project.Manifest, phase) {
		return &vroolierr.Error{
			Code:       "project_phase_not_defined",
			Category:   "Usage",
			HTTPStatus: projectParameterB,
			Message:    fmt.Sprintf("project lifecycle phase %q is not defined in %s", phase, project.ServicePath),
			Hint:       "Define the phase in .vrooli/service.json or run a supported lifecycle command.",
		}
	}

	newRunner := c.NewPhaseRunner
	if newRunner == nil {
		newRunner = func(root, home string, stdout, stderr io.Writer) (PhaseRunner, error) {
			return lifecycle.NewRunner(root, home, stdout, stderr)
		}
	}
	runner, err := newRunner(c.Root, c.Home, c.Stdout, c.Stderr)
	if err != nil {
		return err
	}
	return runner.RunPhase(project.Slug, phase, lifecycle.PhaseOptions{
		CustomPath:  c.Root,
		ProjectMode: true,
	})
}

func (c *Controller) Status(opts StatusOptions) (StatusReport, error) {
	report := StatusReport{Summary: map[string]int{}}
	if !opts.ScenariosOnly {
		resourceStatuses, err := c.Resources.ListStatuses(opts.Fast, false)
		if err != nil {
			return StatusReport{}, err
		}
		report.Resources = resourceStatuses
		report.Summary["resources_total"] = len(resourceStatuses)
		for _, item := range resourceStatuses {
			if item.Resource.Enabled {
				report.Summary["resources_enabled"]++
			}
			if item.Running {
				report.Summary["resources_running"]++
			}
			if item.Healthy != nil && *item.Healthy {
				report.Summary["resources_healthy"]++
			}
		}
	}
	if !opts.ResourcesOnly {
		scenarios, err := c.Scenarios.List()
		if err != nil {
			return StatusReport{}, err
		}
		report.Scenarios = scenarios
		report.Summary["scenarios_total"] = len(scenarios)
		for _, item := range scenarios {
			if item.Processes > 0 {
				report.Summary["scenarios_running"]++
			} else {
				report.Summary["scenarios_stopped"]++
			}
		}
	}
	maintenanceSnapshot, err := c.maintenanceSnapshot()
	if err != nil {
		return StatusReport{}, err
	}
	report.Maintenance = &maintenanceSnapshot
	report.Summary["maintenance_tracked_processes"] = maintenanceSnapshot.TrackedProcesses
	report.Summary["maintenance_zombie_processes"] = maintenanceSnapshot.ZombieProcesses
	report.Summary["maintenance_orphan_processes"] = maintenanceSnapshot.OrphanProcesses
	return report, nil
}

func (c *Controller) Doctor() (DoctorReport, error) {
	return c.DoctorWithOptions(DoctorOptions{})
}

func (c *Controller) DoctorWithOptions(options DoctorOptions) (DoctorReport, error) {
	checks := make([]DoctorCheck, 0, projectParameterC)
	for _, name := range []string{"jq", "curl", "git", "docker", "go", "lsof", "tput"} {
		status := "missing"
		message := ""
		if _, err := c.lookPath(name); err == nil {
			status = "ok"
		} else {
			message = err.Error()
		}
		checks = append(checks, DoctorCheck{Name: name, Status: status, Message: message})
	}
	checks = append(checks, dockerDaemonDoctorCheck())

	apiPort := strings.TrimSpace(os.Getenv("VROOLI_API_PORT"))
	if apiPort == "" {
		apiPort = "8092"
	}
	checks = append(checks, DoctorCheck{
		Name:   "api_port_" + apiPort,
		Status: apiPortStatus(apiPort),
	})

	servicePath := filepath.Join(c.Root, repocontractmeta.ProjectConfigDir, "service.json")
	if _, err := os.Stat(servicePath); err == nil {
		checks = append(checks, DoctorCheck{Name: "service_json", Status: "present"})
	} else {
		checks = append(checks, DoctorCheck{Name: "service_json", Status: "missing", Message: err.Error()})
	}

	maintenanceSnapshot, err := c.maintenanceSnapshot()
	if err != nil {
		return DoctorReport{}, err
	}
	checks = append(checks, DoctorCheck{
		Name:    "orphan_processes",
		Status:  countStatus(maintenanceSnapshot.OrphanProcesses),
		Message: fmt.Sprintf("%d orphaned Vrooli processes detected", maintenanceSnapshot.OrphanProcesses),
	})
	checks = append(checks, DoctorCheck{
		Name:    "zombie_processes",
		Status:  countStatus(maintenanceSnapshot.ZombieProcesses),
		Message: fmt.Sprintf("%d zombie processes detected", maintenanceSnapshot.ZombieProcesses),
	})
	if options.RepairFilePermissions {
		checks = append(checks, c.repairRuntimeHomeOwnershipCheck())
	}

	listenerInspection := network.ListenerInspectionStatus()
	listenerStatus := "ok"
	if !listenerInspection.Available {
		listenerStatus = "degraded"
	}
	checks = append(checks, DoctorCheck{
		Name:    "listener_inspection",
		Status:  listenerStatus,
		Message: listenerInspectionMessage(listenerInspection),
	})

	if c.HostReqValidateFn != nil {
		report, err := c.HostReqValidateFn(c.Root, c.Home)
		if err != nil {
			checks = append(checks, DoctorCheck{
				Name:    "host_requirements_validation",
				Status:  "error",
				Message: err.Error(),
			})
		} else {
			checks = append(checks,
				summarizeHostReqFindings("hostreq_undeclared_references", report.Findings, hostreqcheck.FindingUndeclaredReference),
				summarizeHostReqFindings("hostreq_missing_handlers", report.Findings, hostreqcheck.FindingMissingHandler),
				summarizeHostReqFindings("hostreq_root_overreach", report.Findings, hostreqcheck.FindingRootOverreach),
			)
		}
	}

	installChecks, err := c.cliInstallLocationChecks()
	if err != nil {
		return DoctorReport{}, err
	}
	checks = append(checks, installChecks...)

	return DoctorReport{Checks: checks}, nil
}

func (c *Controller) repairRuntimeHomeOwnershipCheck() DoctorCheck {
	if c.RepairRuntimeHomeFn != nil {
		return c.RepairRuntimeHomeFn()
	}
	uid, gid := config.RepairIdentity()
	classes := []string{"bin", "cache", "logs", "metrics", "processes", "build", "test_runs", "backups", "artifacts"}
	broker := privilegebroker.NewClient()
	if !broker.Available() {
		return DoctorCheck{Name: "runtime_home_ownership_repair", Status: "degraded", Message: "privilege broker is unavailable; no repair was attempted"}
	}
	var repaired, failed uint64
	for index, class := range classes {
		result, err := broker.Do(context.Background(), privilegebroker.Request{
			Version: privilegebroker.ProtocolVersion, RequestID: fmt.Sprintf("doctor-runtime-home-%d", index+1),
			Action:      privilegebroker.ActionRuntimeHomeOwnershipRepair,
			RuntimeHome: &privilegebroker.RuntimeHomeSubject{Class: class, ExpectedUID: uid, ExpectedGID: gid},
		})
		if err != nil {
			return DoctorCheck{Name: "runtime_home_ownership_repair", Status: "degraded", Message: fmt.Sprintf("%s: %v", class, err)}
		}
		repaired += result.Evidence.Repaired
		failed += result.Evidence.Failed
		if result.Status == "failed" {
			return DoctorCheck{Name: "runtime_home_ownership_repair", Status: "degraded", Message: fmt.Sprintf("%s: broker repair failed (%s)", class, result.Code)}
		}
	}
	if failed > 0 {
		return DoctorCheck{Name: "runtime_home_ownership_repair", Status: "degraded", Message: fmt.Sprintf("repaired=%d failed=%d", repaired, failed)}
	}
	return DoctorCheck{Name: "runtime_home_ownership_repair", Status: "ok", Message: fmt.Sprintf("repaired=%d entries", repaired)}
}

func dockerDaemonDoctorCheck() DoctorCheck {
	health := dockerhost.InspectHealth()
	if !health.ClientInstalled {
		return DoctorCheck{Name: "docker_daemon", Status: "missing", Message: "Docker CLI is not installed"}
	}
	if health.InfoOK {
		return DoctorCheck{Name: "docker_daemon", Status: "ok", Message: "Docker daemon is reachable"}
	}
	message := dockerhost.DiagnosticLine(health.Detail)
	if health.PermissionDenied {
		message = "Current user cannot access the Docker socket: " + message
	} else if health.ServiceFailed {
		message = "Docker systemd service is failed: " + message
	}
	if !health.ConfigValid && health.ValidationDetail != "" {
		if message != "" {
			message += "; "
		}
		message += "daemon config validation failed: " + health.ValidationDetail
	}
	return DoctorCheck{Name: "docker_daemon", Status: "error", Message: message}
}

func (c *Controller) cliInstallLocationChecks() ([]DoctorCheck, error) {
	manager, err := cliinstall.NewManager(c.Root, c.Home)
	if err != nil {
		return nil, err
	}

	scenarioReport, err := manager.DiscoverScenarioCLIReport()
	if err != nil {
		return nil, err
	}
	resourceReport, err := manager.DiscoverEnabledResourceCLIReport()
	if err != nil {
		return nil, err
	}

	scenarioStatuses := make([]cliinstall.InstallLocationStatus, 0, len(scenarioReport.Items))
	for _, item := range scenarioReport.Items {
		status, err := manager.InspectScenarioCLIInstallLocation(item.Name, c.lookPath)
		if err != nil {
			return nil, err
		}
		scenarioStatuses = append(scenarioStatuses, status)
	}

	resourceStatuses := make([]cliinstall.InstallLocationStatus, 0, len(resourceReport.Items))
	for _, item := range resourceReport.Items {
		status, err := manager.InspectResourceCLIInstallLocation(item.Name, c.lookPath)
		if err != nil {
			return nil, err
		}
		resourceStatuses = append(resourceStatuses, status)
	}

	return []DoctorCheck{
		summarizeDiscoveryFailures("scenario_cli_discovery", scenarioReport.Failures),
		summarizeCLIInstallStatuses("scenario_cli_install_locations", scenarioStatuses),
		summarizeDiscoveryFailures("resource_cli_discovery", resourceReport.Failures),
		summarizeCLIInstallStatuses("resource_cli_install_locations", resourceStatuses),
	}, nil
}

func summarizeHostReqFindings(name string, findings []hostreqcheck.Finding, code hostreqcheck.FindingCode) DoctorCheck {
	count := 0
	samples := make([]string, 0, projectParameterA)
	for _, finding := range findings {
		if finding.Code != code {
			continue
		}
		count++
		if len(samples) < projectParameterA {
			samples = append(samples, fmt.Sprintf("%s/%s:%s", finding.OwnerKind, finding.OwnerName, finding.Requirement))
		}
	}
	if count == 0 {
		return DoctorCheck{Name: name, Status: "ok"}
	}
	message := fmt.Sprintf("%d findings", count)
	if len(samples) > 0 {
		message += ": " + strings.Join(samples, ", ")
	}
	return DoctorCheck{Name: name, Status: "warning", Message: message}
}

func summarizeCLIInstallStatuses(name string, statuses []cliinstall.InstallLocationStatus) DoctorCheck {
	if len(statuses) == 0 {
		return DoctorCheck{Name: name, Status: "ok", Message: "no managed CLIs discovered"}
	}

	samples := make([]string, 0, projectParameterA)
	nonCanonical := 0
	notInstalled := 0
	notOnPath := 0
	for _, status := range statuses {
		switch {
		case status.PathMismatch():
			nonCanonical++
			if len(samples) < projectParameterA {
				samples = append(samples, fmt.Sprintf("%s resolved to non-canonical path %s (canonical: %s)", status.Command, status.ResolvedPath, status.CanonicalPath))
			}
		case !status.CanonicalExists:
			notInstalled++
			if len(samples) < projectParameterA {
				samples = append(samples, fmt.Sprintf("%s is not installed in the canonical path %s", status.Command, status.CanonicalPath))
			}
		case !status.Resolved:
			notOnPath++
			if len(samples) < projectParameterA {
				samples = append(samples, fmt.Sprintf("%s is installed canonically at %s but not currently resolvable on PATH", status.Command, status.CanonicalPath))
			}
		}
	}

	if nonCanonical == 0 && notInstalled == 0 && notOnPath == 0 {
		return DoctorCheck{
			Name:    name,
			Status:  "ok",
			Message: fmt.Sprintf("%d managed CLIs resolve canonically", len(statuses)),
		}
	}

	parts := make([]string, 0, projectParameterA)
	if nonCanonical > 0 {
		parts = append(parts, fmt.Sprintf("%d resolve to non-canonical paths", nonCanonical))
	}
	if notInstalled > 0 {
		parts = append(parts, fmt.Sprintf("%d are not installed in the canonical path", notInstalled))
	}
	if notOnPath > 0 {
		parts = append(parts, fmt.Sprintf("%d canonical installs are not on PATH", notOnPath))
	}
	slices.Sort(samples)

	message := strings.Join(parts, "; ")
	if len(samples) > 0 {
		message += ": " + strings.Join(samples, ", ")
	}
	return DoctorCheck{Name: name, Status: "warning", Message: message}
}

func summarizeDiscoveryFailures(name string, failures []discovery.Failure) DoctorCheck {
	if len(failures) == 0 {
		return DoctorCheck{Name: name, Status: "ok"}
	}
	samples := make([]string, 0, projectParameterA)
	for _, failure := range failures {
		if len(samples) >= projectParameterA {
			break
		}
		samples = append(samples, fmt.Sprintf("%s: %s", failure.Name, failure.Error))
	}
	message := fmt.Sprintf("%d discovery failures", len(failures))
	if len(samples) > 0 {
		message += ": " + strings.Join(samples, ", ")
	}
	return DoctorCheck{Name: name, Status: "warning", Message: message}
}

func countStatus(count int) string {
	if count > 0 {
		return projectSeverityWarning
	}
	return "ok"
}

func listenerInspectionMessage(status network.ListenerInspection) string {
	if status.Available {
		if strings.TrimSpace(status.Tool) != "" {
			return fmt.Sprintf("listener inspection available via %s", status.Tool)
		}
		return "listener inspection available"
	}
	if strings.TrimSpace(status.Reason) != "" {
		return status.Reason
	}
	return "listener inspection unavailable"
}

func (c *Controller) lookPath(name string) (string, error) {
	if c.LookPathFn != nil {
		return c.LookPathFn(name)
	}
	return shell.LookPath(name)
}

func (c *Controller) maintenanceSnapshot() (maintenance.ProcessSnapshot, error) {
	if c.MaintenanceSnapshotFn != nil {
		return c.MaintenanceSnapshotFn()
	}
	return c.Maintenance.Snapshot()
}

func (c *Controller) Stop(opts StopOptions) (control.StopReport, error) {
	targets := opts.Args
	if len(targets) == 0 {
		targets = []string{"all"}
	}

	stopped := make([]control.ResultItem, 0)
	failed := make([]control.ResultItem, 0)
	for _, target := range targets {
		switch {
		case target == "all":
			scenarioResult, err := c.Scenarios.StopAll()
			if err != nil {
				failed = append(failed, control.Failed(repocontractmeta.ScenarioDir, err))
			} else {
				stopped = append(stopped, scenarioResult.Stopped...)
				failed = append(failed, scenarioResult.Failed...)
			}
			resourceResult, err := c.Resources.StopAll(c.Stdout, c.Stderr)
			if err != nil {
				failed = append(failed, control.Failed("resources", err))
			} else {
				stopped = append(stopped, resourceResult.Stopped...)
				failed = append(failed, resourceResult.Failed...)
			}
		case target == repocontractmeta.ScenarioDir:
			scenarioResult, err := c.Scenarios.StopAll()
			if err != nil {
				failed = append(failed, control.Failed(repocontractmeta.ScenarioDir, err))
			} else {
				stopped = append(stopped, scenarioResult.Stopped...)
				failed = append(failed, scenarioResult.Failed...)
			}
		case target == "resources":
			resourceResult, err := c.Resources.StopAll(c.Stdout, c.Stderr)
			if err != nil {
				failed = append(failed, control.Failed("resources", err))
			} else {
				stopped = append(stopped, resourceResult.Stopped...)
				failed = append(failed, resourceResult.Failed...)
			}
		case strings.HasPrefix(target, "scenario:"):
			name := strings.TrimPrefix(target, "scenario:")
			if err := c.Scenarios.Stop(name, lifecycle.StopOptions{}); err != nil {
				failed = append(failed, control.Failed(name, err))
			} else {
				stopped = append(stopped, control.Stopped(name, "Stopped successfully"))
			}
		case strings.HasPrefix(target, "resource:"):
			name := strings.TrimPrefix(target, "resource:")
			if err := c.Resources.Run(name, []string{"stop"}, c.Stdout, c.Stderr); err != nil {
				failed = append(failed, control.Failed(name, err))
			} else {
				stopped = append(stopped, control.Stopped(name, "Stopped successfully"))
			}
		default:
			if _, exists, err := c.Scenarios.Status(target); err == nil && exists {
				if err := c.Scenarios.Stop(target, lifecycle.StopOptions{}); err != nil {
					failed = append(failed, control.Failed(target, err))
				} else {
					stopped = append(stopped, control.Stopped(target, "Stopped successfully"))
				}
				continue
			}
			if err := c.Resources.Run(target, []string{"stop"}, c.Stdout, c.Stderr); err != nil {
				failed = append(failed, control.Failed(target, err))
			} else {
				stopped = append(stopped, control.Stopped(target, "Stopped successfully"))
			}
		}
	}

	return control.StopReport{
		Stopped: stopped,
		Failed:  failed,
		Message: control.StopSummary(len(stopped), len(failed)),
	}, nil
}

func LoadProject(root string) (scenario.Scenario, error) {
	servicePath := filepath.Join(root, repocontractmeta.ProjectConfigDir, "service.json")
	manifest, err := scenario.ReadService(servicePath)
	if err != nil {
		return scenario.Scenario{}, fmt.Errorf("read project service manifest: %w", err)
	}
	slug := strings.TrimSpace(manifest.Service.Name)
	if slug == "" {
		slug = filepath.Base(root)
	}
	if slug == "" || slug == "." {
		slug = "vrooli-dev"
	}
	return scenario.Scenario{
		Slug:        slug,
		Path:        root,
		ServicePath: servicePath,
		Manifest:    manifest,
	}, nil
}

func phaseDefined(manifest scenario.ServiceManifest, phase string) bool {
	for _, summary := range manifest.PhaseSummaries() {
		if summary.Name == phase {
			return summary.Defined
		}
	}
	return false
}

func apiPortStatus(port string) string {
	conn, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", port))
	if err != nil {
		return "in_use"
	}
	_ = conn.Close()
	return "free"
}

func (r DoctorReport) JSON() ([]byte, error) {
	return json.Marshal(r)
}
