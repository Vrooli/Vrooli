package privilegebroker

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// EdgeRuleComment tags the edge rules this policy manages so a later
// inspection can tell them from operator rules.
const EdgeRuleComment = "vrooli-edge-http-v1"

var edgeAllowedPorts = map[int]struct{}{80: {}, 443: {}}

func validateEdge(req Request) error {
	if req.Subject != (Subject{}) || req.Volume != nil || req.RuntimeHome != nil || req.Log != nil || req.Journal != nil || req.Docker != nil || req.Apt != nil || req.Process != nil || req.Caddy != nil {
		return fmt.Errorf("subject_not_allowed")
	}
	if req.Edge == nil {
		return fmt.Errorf("edge_subject_required")
	}
	if _, ok := edgeAllowedPorts[req.Edge.Port]; !ok {
		return fmt.Errorf("port_not_allowed")
	}
	return nil
}

// EdgeUFWArgs returns the fixed allow argv for an accepted edge request. The
// action is allow-only: this policy can never delete a rule, so a repair can
// never remove the management path.
func EdgeUFWArgs(req Request) ([]string, error) {
	if err := Validate(req); err != nil {
		return nil, err
	}
	if req.Action != ActionEdgeUFWAllow {
		return nil, fmt.Errorf("action_not_allowed")
	}
	return []string{"allow", strconv.Itoa(req.Edge.Port) + "/tcp", "comment", EdgeRuleComment}, nil
}

func executeEdgeUFW(ctx context.Context, executor Executor, req Request) Result {
	inspect := func() (Evidence, error) {
		out, err := executor.Run(ctx, "ufw", "status", "numbered")
		if err != nil {
			if errors.Is(err, exec.ErrNotFound) {
				return Evidence{}, errUFWUnavailable
			}
			return Evidence{}, err
		}
		return parseEdgeUFWStatus(string(out), req.Edge.Port), nil
	}
	evidence, err := inspect()
	if err != nil {
		return failureForUFW(req, err)
	}
	if !evidence.Active {
		// An inactive firewall already admits the port; enabling ufw is an
		// operator decision this action must not make on their behalf.
		return Result{Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action, Status: "verified", Evidence: evidence}
	}
	if evidence.RuleFound {
		return Result{Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action, Status: "already_present", Evidence: evidence}
	}
	args, _ := EdgeUFWArgs(req)
	if _, err := executor.Run(ctx, "ufw", args...); err != nil {
		return failureForUFW(req, err)
	}
	evidence, err = inspect()
	if err != nil {
		return failureForUFW(req, err)
	}
	if !evidence.RuleFound {
		return Result{Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action, Status: "failed", Code: "rule_not_verified", Evidence: evidence}
	}
	return Result{Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action, Status: "changed", Changed: true, Evidence: evidence}
}

func parseEdgeUFWStatus(output string, port int) Evidence {
	text := strings.ToLower(output)
	evidence := Evidence{Available: true, Active: strings.Contains(text, "status: active")}
	if !evidence.Active {
		return evidence
	}
	want := strconv.Itoa(port) + "/tcp"
	for _, line := range strings.Split(text, "\n") {
		if !strings.Contains(line, "allow") {
			continue
		}
		for _, field := range strings.Fields(line) {
			if field == want || field == strconv.Itoa(port) {
				evidence.RuleFound = true
				if strings.Contains(line, EdgeRuleComment) {
					evidence.Managed = true
				}
			}
		}
	}
	return evidence
}
