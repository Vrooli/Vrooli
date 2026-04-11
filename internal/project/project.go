package project

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/scenario"
)

type Controller struct {
	Root      string
	Home      string
	Stdout    io.Writer
	Stderr    io.Writer
	Resources *resources.Controller
	Scenarios *orchestrator.Service
}

type DoctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type DoctorReport struct {
	Checks []DoctorCheck `json:"checks"`
}

type StatusOptions struct {
	ResourcesOnly bool
	ScenariosOnly bool
	Fast          bool
}

type StatusReport struct {
	Resources []resources.Status          `json:"resources,omitempty"`
	Scenarios []orchestrator.ScenarioView `json:"scenarios,omitempty"`
	Summary   map[string]int              `json:"summary"`
}

type StopOptions struct {
	Target  string
	Args    []string
	DryRun  bool
	Verbose bool
}

func New(root, home string, stdout, stderr io.Writer) *Controller {
	return &Controller{
		Root:      filepath.Clean(root),
		Home:      filepath.Clean(home),
		Stdout:    stdout,
		Stderr:    stderr,
		Resources: resources.NewController(root, home),
		Scenarios: orchestrator.New(root, home, stdout, stderr),
	}
}

func (c *Controller) RunProjectPhase(phase string, args []string) error {
	project, err := LoadProject(c.Root)
	if err != nil {
		return err
	}
	if !phaseDefined(project.Manifest, phase) {
		return fmt.Errorf("project lifecycle phase %q is not defined in %s", phase, project.ServicePath)
	}

	runner, err := lifecycle.NewRunner(c.Root, c.Home, c.Stdout, c.Stderr)
	if err != nil {
		return err
	}
	return runner.RunPhase(project.Slug, phase, lifecycle.PhaseOptions{
		CustomPath:  c.Root,
		Args:        append([]string(nil), args...),
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
	return report, nil
}

func (c *Controller) Doctor() (DoctorReport, error) {
	checks := make([]DoctorCheck, 0, 8)
	for _, name := range []string{"jq", "curl", "git", "docker", "go", "lsof", "tput"} {
		status := "missing"
		if _, err := exec.LookPath(name); err == nil {
			status = "ok"
		}
		checks = append(checks, DoctorCheck{Name: name, Status: status})
	}

	apiPort := strings.TrimSpace(os.Getenv("VROOLI_API_PORT"))
	if apiPort == "" {
		apiPort = "8092"
	}
	checks = append(checks, DoctorCheck{
		Name:   "api_port_" + apiPort,
		Status: apiPortStatus(apiPort),
	})

	servicePath := filepath.Join(c.Root, ".vrooli", "service.json")
	if _, err := os.Stat(servicePath); err == nil {
		checks = append(checks, DoctorCheck{Name: "service_json", Status: "present"})
	} else {
		checks = append(checks, DoctorCheck{Name: "service_json", Status: "missing"})
	}

	return DoctorReport{Checks: checks}, nil
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
				failed = append(failed, control.Failed("scenarios", err))
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
		case target == "scenarios":
			scenarioResult, err := c.Scenarios.StopAll()
			if err != nil {
				failed = append(failed, control.Failed("scenarios", err))
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
	servicePath := filepath.Join(root, ".vrooli", "service.json")
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
