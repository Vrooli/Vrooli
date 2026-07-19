// Package privilegebroker implements the setup-managed, local-only host action
// boundary. It intentionally has no generic command execution API.
package privilegebroker

import "strings"

const (
	ProtocolVersion = "v1"
	BridgePort      = 18767
	BridgeScenario  = "vrooli-bridge"
	RuleComment     = "vrooli-bridge-admission-v1"

	ActionBridgeUFWInspect = "bridge.ufw.inspect"
	ActionBridgeUFWAllow   = "bridge.ufw.allow"
	ActionBridgeUFWVerify  = "bridge.ufw.verify"
	ActionBridgeUFWRevoke  = "bridge.ufw.revoke"
)

// Request is the complete v1 wire input. Deliberately, it has no command,
// argv, environment, working-directory, or path field.
type Request struct {
	Version   string  `json:"version"`
	RequestID string  `json:"request_id"`
	Action    string  `json:"action"`
	Subject   Subject `json:"subject"`
}

type Subject struct {
	Scenario    string `json:"scenario"`
	CandidateIP string `json:"candidate_ip"`
	Port        int    `json:"port"`
}

type Result struct {
	Version   string   `json:"version"`
	RequestID string   `json:"request_id,omitempty"`
	Action    string   `json:"action,omitempty"`
	Status    string   `json:"status"`
	Code      string   `json:"code,omitempty"`
	Changed   bool     `json:"changed,omitempty"`
	Evidence  Evidence `json:"evidence,omitempty"`
}

type Evidence struct {
	Available bool   `json:"available"`
	Active    bool   `json:"active"`
	RuleFound bool   `json:"rule_found"`
	Managed   bool   `json:"managed"`
	Detail    string `json:"detail,omitempty"`
}

func NewFailure(requestID, action, code string) Result {
	return Result{Version: ProtocolVersion, RequestID: strings.TrimSpace(requestID), Action: strings.TrimSpace(action), Status: "failed", Code: code}
}
