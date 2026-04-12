package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	batchcontrol "github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/resources/catalog"
	compatpkg "github.com/vrooli/vrooli/internal/resources/compat"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/vroolierr"
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
	Resource   catalog.Resource `json:"resource"`
	Installed  bool             `json:"installed"`
	Running    bool             `json:"running"`
	Healthy    *bool            `json:"healthy,omitempty"`
	Health     string           `json:"health,omitempty"`
	StatusCode string           `json:"status_code,omitempty"`
	Message    string           `json:"message,omitempty"`
	ProbeError string           `json:"probe_error,omitempty"`
	Raw        json.RawMessage  `json:"raw,omitempty"`
}

type CommandResult struct {
	Output []byte
	Err    error
}

type Service struct {
	DiscoverFn           func() ([]catalog.Resource, error)
	DiscoverOneFn        func(name string) (*catalog.Resource, error)
	IsDeprecatedFn       func(name string) (bool, error)
	IsBlueprintArchFn    func(name string) (bool, error)
	LoadManifestFn       func(path string) (manifestpkg.ResourceManifest, error)
	DriverStatusFn       func(ctx context.Context, item catalog.Resource, manifest manifestpkg.ResourceManifest, fast bool) (Status, error)
	DriverRunFn          func(ctx context.Context, item catalog.Resource, manifest manifestpkg.ResourceManifest, operation string, args []string, stdout, stderr io.Writer) error
	RunLegacyFn          func(name, operation string, args []string, stdout, stderr io.Writer) error
	CommandForResourceFn func(name string, args ...string) (*exec.Cmd, error)
	RunCommandFn         func(ctx context.Context, cmd *exec.Cmd) CommandResult
}

func (s *Service) ListStatuses(fast bool, onlyEnabled bool) ([]Status, error) {
	items, err := s.DiscoverFn()
	if err != nil {
		return nil, err
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
				Installed: item.Exists || item.HasCLI || item.HasScript,
				Message:   statusErr.Error(),
			}
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func (s *Service) Status(name string, fast bool) (Status, error) {
	if deprecated, err := s.IsDeprecatedFn(name); err != nil {
		return Status{}, err
	} else if deprecated {
		return Status{}, &vroolierr.Error{
			Code:       "resource_deprecated",
			Resource:   name,
			Category:   "Usage",
			HTTPStatus: 409,
			Message:    fmt.Sprintf("resource %q is deprecated; use `vrooli resource list-deprecated` or `vrooli resource restore %s`", name, name),
		}
	}
	if archived, err := s.IsBlueprintArchFn(name); err != nil {
		return Status{}, err
	} else if archived {
		return Status{}, &vroolierr.Error{
			Code:       "resource_blueprint_archived",
			Resource:   name,
			Category:   "Usage",
			HTTPStatus: 409,
			Message:    fmt.Sprintf("resource %q is blueprint-archived; use `vrooli resource list-blueprint-archived` or `vrooli resource restore-blueprint %s`", name, name),
		}
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
			HTTPStatus: 404,
			Message:    fmt.Sprintf("resource %q not found", name),
		}
	}
	return s.StatusForResource(*item, fast)
}

func (s *Service) Run(name string, args []string, stdout, stderr io.Writer) error {
	if deprecated, err := s.IsDeprecatedFn(name); err != nil {
		return err
	} else if deprecated {
		return &vroolierr.Error{
			Code:       "resource_deprecated",
			Resource:   name,
			Category:   "Usage",
			HTTPStatus: 409,
			Message:    fmt.Sprintf("resource %q is deprecated and cannot be run from the active control surface", name),
		}
	}
	if archived, err := s.IsBlueprintArchFn(name); err != nil {
		return err
	} else if archived {
		return &vroolierr.Error{
			Code:       "resource_blueprint_archived",
			Resource:   name,
			Category:   "Usage",
			HTTPStatus: 409,
			Message:    fmt.Sprintf("resource %q is blueprint-archived and cannot be run from the active control surface", name),
		}
	}
	operation := "invoke"
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		operation = args[0]
	}
	item, err := s.DiscoverOneFn(name)
	if err != nil {
		return err
	}
	if item != nil && item.ManifestPath != "" && item.ControlMode == "manifest-native" {
		manifest, err := s.LoadManifestFn(item.ManifestPath)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := s.DriverRunFn(ctx, *item, manifest, operation, args[1:], stdout, stderr); err != nil {
			if shouldFallbackToLegacyResourceCommand(err) {
				return s.RunLegacyFn(name, operation, args[1:], stdout, stderr)
			}
			var resourceErr *vroolierr.Error
			if errors.As(err, &resourceErr) {
				return err
			}
			return &vroolierr.Error{
				Code:      ErrorCodeOperationFailed,
				Resource:  name,
				Operation: operation,
				Category:  "Runtime",
				Err:       err,
			}
		}
		return nil
	}
	return s.RunLegacyFn(name, operation, args[1:], stdout, stderr)
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
			if status.StatusCode == StatusCodeOK || (!status.Resource.HasCLI && !status.Resource.HasScript) {
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

func (s *Service) StatusForResource(item catalog.Resource, fast bool) (Status, error) {
	if item.ManifestPath != "" && item.ControlMode == "manifest-native" {
		manifest, err := s.LoadManifestFn(item.ManifestPath)
		if err != nil {
			return Status{}, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.DriverStatusFn(ctx, item, manifest, fast)
	}

	status := Status{
		Resource:   item,
		Installed:  item.Exists || item.HasCLI || item.HasScript,
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
	cmd.Dir = cmd.Dir
	cmd.Env = cmd.Env

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

	rawJSON, ok := compatpkg.ExtractJSONPayload(result.Output)
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

	status.Installed = compatpkg.BoolValue(payload["installed"], status.Installed)
	status.Running = compatpkg.BoolValue(payload["running"], false)
	status.Message = compatpkg.StringValue(payload["message"])
	if status.Message == "" {
		status.Message = compatpkg.StringValue(payload["status"])
	}
	status.Health = compatpkg.StringValue(payload["health"])
	if healthy, ok := payload["healthy"]; ok {
		value := compatpkg.BoolValue(healthy, false)
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

func shouldFallbackToLegacyResourceCommand(err error) bool {
	var resourceErr *vroolierr.Error
	if !errors.As(err, &resourceErr) {
		return false
	}
	return resourceErr.Code == ErrorCodeCommandUnavailable && resourceErr.Category == "Driver"
}
