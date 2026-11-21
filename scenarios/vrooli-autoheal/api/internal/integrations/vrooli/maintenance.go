package vrooli

import (
	"context"
	"fmt"
	"strconv"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

type OrphanProcess struct {
	PID     int
	PPID    int
	Command string
}

type PortListener struct {
	PID     int
	Command string
	Zombie  bool
}

type ListenerInspection struct {
	Available bool
	Tool      string
	Reason    string
}

type PortDiagnostic struct {
	Port               int
	Scenario           string
	InUse              bool
	Listeners          []PortListener
	ListenerInspection ListenerInspection
	HostOrphanCount    int
	Recommendations    []string
}

type StopReportItem struct {
	Name    string
	Message string
}

type StopReportFailure struct {
	Name  string
	Error string
}

type CleanupReport struct {
	Stopped []StopReportItem
	Failed  []StopReportFailure
	Message string
}

type RuntimeSupervisorStatus struct {
	SupervisorID        string
	Status              string
	StatusReason        string
	HeartbeatDeadlineAt string
	PID                 *int
}

// ListOrphans runs `vrooli orphans --json` (read-only) and returns the host's
// orphaned Vrooli processes.
func (c *Client) ListOrphans(ctx context.Context) ([]OrphanProcess, []byte, error) {
	resp, err := c.cli.Orphans(ctx)
	if err != nil {
		return nil, nil, err
	}
	orphans := make([]OrphanProcess, 0, len(resp.GetOrphans()))
	for _, p := range resp.GetOrphans() {
		orphans = append(orphans, OrphanProcess{
			PID:     int(p.GetPid()),
			PPID:    int(p.GetPpid()),
			Command: p.GetCommand(),
		})
	}
	return orphans, diagnosticJSON(resp), nil
}

// CleanupOrphans runs `vrooli cleanup orphans --json` (MUTATION) and returns the
// stop report.
func (c *Client) CleanupOrphans(ctx context.Context) (CleanupReport, []byte, error) {
	resp, err := c.cli.CleanupOrphans(ctx)
	if err != nil {
		return CleanupReport{}, nil, err
	}
	return cleanupReportFromProto(resp.GetData()), diagnosticJSON(resp), nil
}

// CleanupLocks runs `vrooli cleanup locks --json` (MUTATION) and returns the
// stop report.
func (c *Client) CleanupLocks(ctx context.Context) (CleanupReport, []byte, error) {
	resp, err := c.cli.CleanupLocks(ctx)
	if err != nil {
		return CleanupReport{}, nil, err
	}
	return cleanupReportFromProto(resp.GetData()), diagnosticJSON(resp), nil
}

// DiagnosePort runs `vrooli diagnose-port <port> [scenario] --json` and returns
// the port diagnostic. port is accepted as a string for caller convenience.
func (c *Client) DiagnosePort(ctx context.Context, port string, scenario string) (PortDiagnostic, []byte, error) {
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return PortDiagnostic{}, nil, fmt.Errorf("diagnose-port: invalid port %q: %w", port, err)
	}
	resp, err := c.cli.DiagnosePort(ctx, portNum, scenario)
	if err != nil {
		return PortDiagnostic{}, nil, err
	}
	return portDiagnosticFromProto(resp.GetDiagnostic()), diagnosticJSON(resp), nil
}

// RuntimeSupervisorStatus runs `vrooli runtime supervisor status --json`.
func (c *Client) RuntimeSupervisorStatus(ctx context.Context) (RuntimeSupervisorStatus, []byte, error) {
	resp, err := c.cli.RuntimeSupervisorStatus(ctx)
	if err != nil {
		return RuntimeSupervisorStatus{}, nil, err
	}
	status := RuntimeSupervisorStatus{
		SupervisorID:        resp.GetSupervisorId(),
		Status:              resp.GetStatus(),
		StatusReason:        resp.GetStatusReason(),
		HeartbeatDeadlineAt: resp.GetHeartbeatDeadlineAt(),
	}
	if pid := resp.GetPid(); pid != 0 {
		p := int(pid)
		status.PID = &p
	}
	return status, diagnosticJSON(resp), nil
}

// InstallRuntimeSupervisor runs `vrooli runtime supervisor install --user`
// (MUTATION, plain text).
func (c *Client) InstallRuntimeSupervisor(ctx context.Context) ([]byte, error) {
	return c.CombinedOutput(ctx, "runtime", "supervisor", "install", "--user")
}

func portDiagnosticFromProto(d *cliv1.ProjectPortDiagnostic) PortDiagnostic {
	diag := PortDiagnostic{
		Port:            int(d.GetPort()),
		Scenario:        d.GetScenario(),
		InUse:           d.GetInUse(),
		HostOrphanCount: int(d.GetHostOrphanCount()),
		Recommendations: d.GetRecommendations(),
	}
	if li := d.GetListenerInspection(); li != nil {
		diag.ListenerInspection = ListenerInspection{
			Available: li.GetAvailable(),
			Tool:      li.GetTool(),
			Reason:    li.GetReason(),
		}
	}
	for _, l := range d.GetListeners() {
		diag.Listeners = append(diag.Listeners, PortListener{
			PID:     int(l.GetPid()),
			Command: l.GetCommand(),
			Zombie:  l.GetZombie(),
		})
	}
	return diag
}

func cleanupReportFromProto(r *cliv1.ProjectStopReport) CleanupReport {
	report := CleanupReport{Message: r.GetMessage()}
	for _, item := range r.GetStopped() {
		report.Stopped = append(report.Stopped, StopReportItem{
			Name:    item.GetName(),
			Message: item.GetMessage(),
		})
	}
	for _, item := range r.GetFailed() {
		report.Failed = append(report.Failed, StopReportFailure{
			Name:  item.GetName(),
			Error: item.GetError(),
		})
	}
	return report
}
