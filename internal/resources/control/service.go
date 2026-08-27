package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/tuning"

	batchcontrol "github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/discovery"
	"github.com/vrooli/vrooli/internal/resources/catalog"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/vroolierr"
)

const (
	serviceParameterE = 404
	serviceParameterF = 409
)

const (
	StatusCodeOK                   = "ok"
	StatusCodeUnavailable          = "unavailable"
	StatusCodeTimeout              = "timeout"
	StatusCodeCommandError         = "command_error"
	StatusCodeInvalidStatusPayload = "invalid_status_payload"
)

const (
	ErrorCodeCommandUnavailable = "command_unavailable"
	ErrorCodeOperationFailed    = "operation_failed"
)

type Status struct {
	Resource  catalog.Resource `json:"resource"`
	Installed bool             `json:"installed"`
	Running   bool             `json:"running"`
	// Healthy is false whenever the resource is not fully meeting its contract,
	// which includes running below its declared accelerator backend. Read
	// Serving alongside it: a degraded resource answers requests, and treating
	// it as down starts a restart loop against something that is working.
	Healthy *bool `json:"healthy,omitempty"`
	// Serving is true whenever the resource can answer requests, whether or not
	// it is healthy. It is the field a consumer uses to tell degraded from down.
	Serving    *bool           `json:"serving,omitempty"`
	Health     string          `json:"health,omitempty"`
	StatusCode string          `json:"status_code,omitempty"`
	Message    string          `json:"message,omitempty"`
	ProbeError string          `json:"probe_error,omitempty"`
	Raw        json.RawMessage `json:"raw,omitempty"`
	// DeclaredMode is the accelerator backend the resource asked the platform
	// for. Empty when the resource declares no accelerator.
	DeclaredMode string `json:"declared_mode,omitempty"`
	// ObservedMode is the backend the host says the running resource is on.
	// Empty means the placement could not be read, which is never reported as
	// agreement.
	ObservedMode string `json:"observed_mode,omitempty"`
	// ModeDrift is true when the resource is serving below its declared
	// backend. It is the machine-readable form of "Degraded is a state, never
	// a secret".
	ModeDrift bool `json:"mode_drift,omitempty"`
	// ModeReason is the evidence behind ObservedMode, in the words of whatever
	// produced it.
	ModeReason string `json:"mode_reason,omitempty"`
}

// StatusCodeModeDrift marks a resource that is serving on a backend below the
// one it declared. running is true, serving is true, healthy is false.
const StatusCodeModeDrift = "mode_drift"

// StatusCodePlacementUndetermined means the resource is healthy and serving,
// but no workload is resident yet so placement cannot be read.
const StatusCodePlacementUndetermined = "placement_undetermined"

// StatusCodeNeedsReacquire marks an artifact that was staged under host facts
// the host no longer reports, so the resolver now selects a different artifact.
// It is distinct from an unavailable artifact on purpose: the bytes are intact,
// the host changed, and one named command repairs it.
const StatusCodeNeedsReacquire = "needs_reacquire"

type CommandResult struct {
	Output []byte
	Err    error
}

type StatusReport struct {
	Items    []Status            `json:"items"`
	Failures []discovery.Failure `json:"failures,omitempty"`
}

type Service struct {
	DiscoverFn           func() ([]catalog.Resource, error)
	DiscoverReportFn     func() (discovery.Report[catalog.Resource], error)
	DiscoverOneFn        func(name string) (*catalog.Resource, error)
	IsDeprecatedFn       func(name string) (bool, error)
	IsBlueprintArchFn    func(name string) (bool, error)
	LoadManifestFn       func(path string) (manifestpkg.ResourceManifest, error)
	DriverStatusFn       func(ctx context.Context, item catalog.Resource, manifest manifestpkg.ResourceManifest, fast bool) (Status, error)
	DriverRunFn          func(ctx context.Context, item catalog.Resource, manifest manifestpkg.ResourceManifest, operation string, args []string, stdout, stderr io.Writer) error
	RunResourceCommandFn func(name, operation string, args []string, stdout, stderr io.Writer) error
	CommandForResourceFn func(name string, args ...string) (*exec.Cmd, error)
	RunCommandFn         func(ctx context.Context, cmd *exec.Cmd) CommandResult
}

func (s *Service) ListStatuses(fast bool, onlyEnabled bool) ([]Status, error) {
	report, err := s.ListStatusesReport(fast, onlyEnabled)
	if err != nil {
		return nil, err
	}
	return report.Items, nil
}

func (s *Service) ListStatusesReport(fast bool, onlyEnabled bool) (StatusReport, error) {
	var (
		items    []catalog.Resource
		failures []discovery.Failure
		err      error
	)
	if s.DiscoverReportFn != nil {
		var report discovery.Report[catalog.Resource]
		report, err = s.DiscoverReportFn()
		if err != nil {
			return StatusReport{}, err
		}
		items = report.Items
		failures = append([]discovery.Failure(nil), report.Failures...)
	} else {
		items, err = s.DiscoverFn()
		if err != nil {
			return StatusReport{}, err
		}
	}

	statuses := make([]Status, 0, len(items))
	for _, item := range items {
		if onlyEnabled && !item.Enabled {
			continue
		}
		status, statusErr := s.StatusForResource(item, fast)
		if statusErr != nil {
			status = Status{
				Resource:  item,
				Installed: item.Exists || item.HasCLI,
				Message:   statusErr.Error(),
			}
		}
		statuses = append(statuses, status)
	}
	return StatusReport{Items: statuses, Failures: failures}, nil
}

func (s *Service) Status(name string, fast bool) (Status, error) {
	if err := s.validateActiveResource(name, true); err != nil {
		return Status{}, err
	}
	item, err := s.DiscoverOneFn(name)
	if err != nil {
		return Status{}, err
	}
	if item == nil {
		return Status{}, &vroolierr.Error{
			Code:       "resource_not_found",
			Resource:   name,
			Category:   "Usage",
			HTTPStatus: serviceParameterE,
			Message:    fmt.Sprintf("resource %q not found", name),
		}
	}
	return s.StatusForResource(*item, fast)
}

func (s *Service) Run(name string, args []string, stdout, stderr io.Writer) error {
	if err := s.validateActiveResource(name, false); err != nil {
		return err
	}
	operation := "invoke"
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		operation = args[0]
	}
	item, err := s.DiscoverOneFn(name)
	if err != nil {
		return err
	}
	if s.useManifestNativeControl(item) {
		return s.runNativeResourceCommand(*item, operation, args[1:], stdout, stderr)
	}
	return s.runResourceCommand(name, operation, args[1:], stdout, stderr)
}

func (s *Service) validateActiveResource(name string, forStatus bool) error {
	if deprecated, err := s.IsDeprecatedFn(name); err != nil {
		return err
	} else if deprecated {
		message := fmt.Sprintf("resource %q is deprecated and cannot be run from the active control surface", name)
		if forStatus {
			message = fmt.Sprintf("resource %q is deprecated; use `vrooli resource list-deprecated` or `vrooli resource restore %s`", name, name)
		}
		return &vroolierr.Error{
			Code:       "resource_deprecated",
			Resource:   name,
			Category:   "Usage",
			HTTPStatus: serviceParameterF,
			Message:    message,
		}
	}
	if archived, err := s.IsBlueprintArchFn(name); err != nil {
		return err
	} else if archived {
		message := fmt.Sprintf("resource %q is blueprint-archived and cannot be run from the active control surface", name)
		if forStatus {
			message = fmt.Sprintf("resource %q is blueprint-archived; use `vrooli resource list-blueprint-archived` or `vrooli resource restore-blueprint %s`", name, name)
		}
		return &vroolierr.Error{
			Code:       "resource_blueprint_archived",
			Resource:   name,
			Category:   "Usage",
			HTTPStatus: serviceParameterF,
			Message:    message,
		}
	}
	return nil
}

func (s *Service) useManifestNativeControl(item *catalog.Resource) bool {
	return item != nil && item.ManifestPath != "" && item.ControlMode == "manifest-native"
}

func (s *Service) runNativeResourceCommand(item catalog.Resource, operation string, args []string, stdout, stderr io.Writer) error {
	manifest, err := s.LoadManifestFn(item.ManifestPath)
	if err != nil {
		return err
	}
	// Managed-service installs may include large, checksum-verified model data
	// artifacts in addition to the launch binary. Lifecycle operations remain
	// short-bounded, while an explicit install gets enough time for a cold
	// model acquisition on a constrained connection.
	timeout := tuning.ResourceControlExtendedTimeout()
	if operation == "install" {
		timeout = tuning.RepairDeadline()
	}
	ctx, cancel := context.WithTimeout(context.Background(), tuning.ResourceOperationTimeout(timeout))
	defer cancel()
	if err := s.DriverRunFn(ctx, item, manifest, operation, args, stdout, stderr); err != nil {
		var resourceErr *vroolierr.Error
		if errors.As(err, &resourceErr) {
			return err
		}
		return &vroolierr.Error{
			Code:      ErrorCodeOperationFailed,
			Resource:  item.Name,
			Operation: operation,
			Category:  "Runtime",
			Err:       err,
		}
	}
	return nil
}

func (s *Service) runResourceCommand(name, operation string, args []string, stdout, stderr io.Writer) error {
	return s.RunResourceCommandFn(name, operation, args, stdout, stderr)
}

func (s *Service) StartAll(stdout, stderr io.Writer) (batchcontrol.StartReport, error) {
	statuses, err := s.ListStatuses(true, true)
	if err != nil {
		return batchcontrol.StartReport{}, err
	}
	started := make([]batchcontrol.ResultItem, 0)
	failed := make([]batchcontrol.ResultItem, 0)
	for _, status := range statuses {
		if !status.Resource.Enabled {
			continue
		}
		bestEffort := status.StatusCode != "" && status.StatusCode != StatusCodeOK
		if status.Running {
			started = append(started, batchcontrol.Started(status.Resource.Name, "Already running"))
			continue
		}
		if err := s.Run(status.Resource.Name, []string{"start"}, stdout, stderr); err != nil {
			failed = append(failed, batchcontrol.Failed(status.Resource.Name, err))
			continue
		}
		message := "Started successfully"
		if bestEffort {
			message = "Started successfully after degraded status probe"
		}
		started = append(started, batchcontrol.Started(status.Resource.Name, message))
	}
	return batchcontrol.StartReport{
		Started: started,
		Failed:  failed,
		Message: batchcontrol.StartSummary(len(started), len(failed)),
	}, nil
}

func (s *Service) StopAll(stdout, stderr io.Writer) (batchcontrol.StopReport, error) {
	statuses, err := s.ListStatuses(true, false)
	if err != nil {
		return batchcontrol.StopReport{}, err
	}
	stopped := make([]batchcontrol.ResultItem, 0)
	failed := make([]batchcontrol.ResultItem, 0)
	for _, status := range statuses {
		bestEffort := false
		if !status.Running {
			if status.StatusCode == StatusCodeOK || !status.Resource.HasCLI {
				continue
			}
			bestEffort = true
		}
		if err := s.Run(status.Resource.Name, []string{"stop"}, stdout, stderr); err != nil {
			failed = append(failed, batchcontrol.Failed(status.Resource.Name, err))
			continue
		}
		message := "Stopped successfully"
		if bestEffort {
			message = "Stopped successfully after degraded status probe"
		}
		stopped = append(stopped, batchcontrol.Stopped(status.Resource.Name, message))
	}
	return batchcontrol.StopReport{
		Stopped: stopped,
		Failed:  failed,
		Message: batchcontrol.StopSummary(len(stopped), len(failed)),
	}, nil
}

//nolint:gocyclo // resource status combines driver selection, health, lifecycle, and fast-path outcomes.
func (s *Service) StatusForResource(item catalog.Resource, fast bool) (Status, error) {
	if item.ManifestPath != "" && item.ControlMode == "manifest-native" {
		manifest, err := s.LoadManifestFn(item.ManifestPath)
		if err != nil {
			return Status{}, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), tuning.ResourceControlTimeout())
		defer cancel()
		return s.DriverStatusFn(ctx, item, manifest, fast)
	}

	status := Status{
		Resource:   item,
		Installed:  item.Exists || item.HasCLI,
		Running:    false,
		StatusCode: StatusCodeOK,
		Message:    "not running",
	}

	cmd, err := s.CommandForResourceFn(item.Name, append([]string{"status", "--format", "json"}, fastArgs(fast)...)...)
	if err != nil {
		status.StatusCode = StatusCodeUnavailable
		status.Message = "resource status command is unavailable"
		status.ProbeError = err.Error()
		return status, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), tuning.ResourceControlTimeout())
	defer cancel()
	origCmd := cmd
	cmd = shell.NewCommandContext(ctx, origCmd.Path, origCmd.Args[1:]...)
	cmd.Dir = origCmd.Dir
	cmd.Env = origCmd.Env

	result := s.RunCommandFn(ctx, cmd)
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(result.Err, context.DeadlineExceeded) {
		status.StatusCode = StatusCodeTimeout
		status.Message = "resource status command timed out"
		if ctx.Err() != nil {
			status.ProbeError = ctx.Err().Error()
		} else if result.Err != nil {
			status.ProbeError = result.Err.Error()
		}
		return status, nil
	}

	rawJSON, ok := extractJSONPayload(result.Output)
	if !ok {
		if result.Err != nil {
			status.StatusCode = StatusCodeCommandError
			status.Message = "resource status command failed"
			status.ProbeError = result.Err.Error()
			return status, nil
		}
		status.StatusCode = StatusCodeInvalidStatusPayload
		status.Message = "resource status command did not emit valid JSON"
		status.ProbeError = strings.TrimSpace(string(result.Output))
		return status, nil
	}

	status.Raw = rawJSON
	var payload map[string]any
	if err := json.Unmarshal(rawJSON, &payload); err != nil {
		status.StatusCode = StatusCodeInvalidStatusPayload
		status.Message = "resource status command emitted invalid JSON"
		status.ProbeError = err.Error()
		return status, nil
	}

	status.Installed = boolValue(payload["installed"], status.Installed)
	status.Running = boolValue(payload["running"], false)
	status.Message = stringValue(payload["message"])
	if status.Message == "" {
		status.Message = stringValue(payload["status"])
	}
	status.Health = stringValue(payload["health"])
	if healthy, ok := payload["healthy"]; ok {
		value := boolValue(healthy, false)
		status.Healthy = &value
	} else if status.Health != "" {
		value := strings.EqualFold(status.Health, "healthy")
		status.Healthy = &value
	}
	if status.Message == "" {
		switch {
		case status.Running && status.Healthy != nil && *status.Healthy:
			status.Message = "healthy"
		case status.Running:
			status.Message = "running"
		default:
			status.Message = "stopped"
		}
	}
	if result.Err != nil {
		status.ProbeError = result.Err.Error()
	}

	return status, nil
}

func fastArgs(fast bool) []string {
	if fast {
		return []string{"--fast"}
	}
	return nil
}
