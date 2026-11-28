package project

import (
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"slices"
	"strings"

	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/shell"
)

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

func summarizeDiscoveryFailures(name string, failures []scenario.Failure) DoctorCheck {
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

func (c *Controller) agentSessions() ([]scenarioruntime.EditorLease, error) {
	return maintenance.NewController(c.Root, c.Home).ListAgentSessions()
}

// countSessionsInTree counts sessions whose working directory is the root or
// inside it.
func countSessionsInTree(sessions []scenarioruntime.EditorLease, root string) int {
	root = filepath.Clean(root)
	count := 0
	for _, session := range sessions {
		dir := filepath.Clean(session.WorkingDir)
		if dir == root || strings.HasPrefix(dir, root+string(filepath.Separator)) {
			count++
		}
	}
	return count
}
