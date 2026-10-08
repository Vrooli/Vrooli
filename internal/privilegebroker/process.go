package privilegebroker

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	processScenarioPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
	processWorkdirPattern  = regexp.MustCompile(`^/[A-Za-z0-9._/-]{0,255}$`)
)

func validateProcess(req Request) error {
	if req.Subject != (Subject{}) || req.Volume != nil || req.RuntimeHome != nil || req.Log != nil || req.Journal != nil || req.Docker != nil || req.Apt != nil || req.Edge != nil || req.Caddy != nil {
		return fmt.Errorf("subject_not_allowed")
	}
	if req.Process == nil {
		return fmt.Errorf("process_subject_required")
	}
	if !processScenarioPattern.MatchString(req.Process.Scenario) {
		return fmt.Errorf("invalid_scenario_id")
	}
	workdir := req.Process.Workdir
	if workdir == "" || workdir != filepath.Clean(workdir) || strings.Contains(workdir, "..") || !processWorkdirPattern.MatchString(workdir) {
		return fmt.Errorf("invalid_workdir")
	}
	return nil
}

// ProcessStopArgs returns the fixed argv for an accepted scoped stop. The stop
// is delegated to the lifecycle owner by scenario id; the workdir binds the
// subject to one deployment for audit and refusal, never to a process pattern.
func ProcessStopArgs(req Request) (string, []string, error) {
	if err := Validate(req); err != nil {
		return "", nil, err
	}
	if req.Action != ActionProcessStopScoped {
		return "", nil, fmt.Errorf("action_not_allowed")
	}
	return "vrooli", []string{"scenario", "stop", req.Process.Scenario, "--json"}, nil
}

func executeProcessStop(ctx context.Context, executor Executor, req Request) Result {
	name, args, err := ProcessStopArgs(req)
	if err != nil {
		return NewFailure(req.RequestID, req.Action, "action_not_allowed")
	}
	if _, err := executor.Run(ctx, name, args...); err != nil {
		return NewFailure(req.RequestID, req.Action, "scenario_stop_failed")
	}
	return Result{Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action, Status: "completed", Changed: true}
}
