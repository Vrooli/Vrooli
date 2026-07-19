package privilegebroker

import (
	"fmt"
	"net"
	"strings"
)

// Validate rejects every request shape outside the immutable v1 registry.
func Validate(req Request) error {
	if req.Version != ProtocolVersion {
		return fmt.Errorf("unsupported_version")
	}
	if strings.TrimSpace(req.RequestID) == "" || len(req.RequestID) > 128 {
		return fmt.Errorf("invalid_request_id")
	}
	switch req.Action {
	case ActionBridgeUFWInspect, ActionBridgeUFWAllow, ActionBridgeUFWVerify, ActionBridgeUFWRevoke:
	default:
		return fmt.Errorf("action_not_allowed")
	}
	if req.Subject.Scenario != BridgeScenario {
		return fmt.Errorf("scenario_not_allowed")
	}
	if req.Subject.Port != BridgePort {
		return fmt.Errorf("port_not_allowed")
	}
	ip := net.ParseIP(strings.TrimSpace(req.Subject.CandidateIP))
	if ip == nil || ip.IsUnspecified() || ip.IsLoopback() || ip.IsMulticast() {
		return fmt.Errorf("invalid_candidate_ip")
	}
	return nil
}

// UFWArgs returns a fixed argv only after Validate has accepted the request.
// The executor always invokes ufw directly; callers cannot influence a shell.
func UFWArgs(req Request) ([]string, error) {
	if err := Validate(req); err != nil {
		return nil, err
	}
	ip := net.ParseIP(strings.TrimSpace(req.Subject.CandidateIP)).String()
	base := []string{"from", ip, "to", "any", "port", "18767", "proto", "tcp", "comment", RuleComment}
	switch req.Action {
	case ActionBridgeUFWInspect, ActionBridgeUFWVerify:
		return []string{"status", "numbered"}, nil
	case ActionBridgeUFWAllow:
		return append([]string{"allow"}, base...), nil
	case ActionBridgeUFWRevoke:
		return append([]string{"delete", "allow"}, base...), nil
	default:
		return nil, fmt.Errorf("action_not_allowed")
	}
}
