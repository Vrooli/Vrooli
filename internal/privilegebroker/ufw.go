package privilegebroker

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

type OSExecutor struct{}

func (OSExecutor) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

func executeUFW(ctx context.Context, executor Executor, req Request) Result {
	inspect := func() (Evidence, error) {
		out, err := executor.Run(ctx, "ufw", "status", "numbered")
		if err != nil {
			if errors.Is(err, exec.ErrNotFound) {
				return Evidence{}, errUFWUnavailable
			}
			return Evidence{}, err
		}
		return parseUFWStatus(string(out), req.Subject.CandidateIP), nil
	}
	evidence, err := inspect()
	if err != nil {
		return failureForUFW(req, err)
	}
	if req.Action == ActionBridgeUFWInspect || req.Action == ActionBridgeUFWVerify {
		return Result{Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action, Status: "verified", Evidence: evidence}
	}
	if req.Action == ActionBridgeUFWAllow {
		if !evidence.Active {
			return Result{Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action, Status: "verified", Evidence: evidence}
		}
		if evidence.Managed {
			return Result{Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action, Status: "already_present", Evidence: evidence}
		}
		args, _ := UFWArgs(req)
		if _, err := executor.Run(ctx, "ufw", args...); err != nil {
			return failureForUFW(req, err)
		}
		evidence, err = inspect()
		if err != nil {
			return failureForUFW(req, err)
		}
		if !evidence.Managed {
			return Result{Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action, Status: "failed", Code: "rule_not_verified", Evidence: evidence}
		}
		return Result{Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action, Status: "changed", Changed: true, Evidence: evidence}
	}
	if !evidence.Managed {
		return Result{Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action, Status: "verified", Evidence: evidence}
	}
	args, _ := UFWArgs(req)
	if _, err := executor.Run(ctx, "ufw", args...); err != nil {
		return failureForUFW(req, err)
	}
	evidence, err = inspect()
	if err != nil {
		return failureForUFW(req, err)
	}
	return Result{Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action, Status: "changed", Changed: true, Evidence: evidence}
}

var errUFWUnavailable = errors.New("ufw unavailable")

func failureForUFW(req Request, err error) Result {
	if errors.Is(err, errUFWUnavailable) {
		return Result{Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action, Status: "unavailable", Code: "ufw_unavailable"}
	}
	return NewFailure(req.RequestID, req.Action, "ufw_execution_failed")
}

func parseUFWStatus(output, candidate string) Evidence {
	text := strings.ToLower(output)
	evidence := Evidence{Available: true, Active: strings.Contains(text, "status: active")}
	if !evidence.Active {
		return evidence
	}
	port := strconv.Itoa(BridgePort)
	ip := strings.ToLower(candidate)
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, ip) && strings.Contains(line, port) && strings.Contains(line, "allow") {
			evidence.RuleFound = true
			if strings.Contains(line, RuleComment) {
				evidence.Managed = true
			}
		}
	}
	return evidence
}
