package sessionstore

import (
	"encoding/json"
	"fmt"
	"strings"
)

// LaunchDescriptor is the explicit, persisted intent for a provider launch.
// It is intentionally separate from the observed ControlMode: intent alone
// never grants native control.
type LaunchDescriptor struct {
	Agent                string            `json:"agent"`
	LaunchMode           string            `json:"launchMode"`
	ControlMode          string            `json:"controlMode,omitempty"`
	ServerTransport      string            `json:"serverTransport,omitempty"`
	ProviderVersionRange string            `json:"providerVersionRange,omitempty"`
	Model                string            `json:"model,omitempty"`
	Profile              string            `json:"profile,omitempty"`
	ApprovalPolicy       string            `json:"approvalPolicy,omitempty"`
	Sandbox              string            `json:"sandbox,omitempty"`
	WorkspaceRoots       []string          `json:"workspaceRoots,omitempty"`
	Environment          map[string]string `json:"environment,omitempty"`
}

func ParseLaunchDescriptor(raw string) (LaunchDescriptor, error) {
	if strings.TrimSpace(raw) == "" {
		return LaunchDescriptor{LaunchMode: string(LaunchModeTerminalPTY), ControlMode: string(ControlModeTranscriptOnly)}, nil
	}
	var d LaunchDescriptor
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		return LaunchDescriptor{}, fmt.Errorf("invalid launch descriptor: %w", err)
	}
	switch LaunchMode(d.LaunchMode) {
	case LaunchModeTerminalPTY, LaunchModeCodexAppServer, LaunchModeNativeAPI, LaunchModeUnknown:
	default:
		return LaunchDescriptor{}, fmt.Errorf("unsupported launch mode %q", d.LaunchMode)
	}
	if d.LaunchMode == string(LaunchModeCodexAppServer) && !strings.EqualFold(d.Agent, string(AgentCodex)) {
		return LaunchDescriptor{}, fmt.Errorf("codex app-server mode requires agent codex")
	}
	if d.ApprovalPolicy != "" && d.ApprovalPolicy != "untrusted" && d.ApprovalPolicy != "on-request" && d.ApprovalPolicy != "never" {
		return LaunchDescriptor{}, fmt.Errorf("unsupported Codex approval policy %q", d.ApprovalPolicy)
	}
	if d.Sandbox != "" && d.Sandbox != "read-only" && d.Sandbox != "workspace-write" && d.Sandbox != "danger-full-access" {
		return LaunchDescriptor{}, fmt.Errorf("unsupported Codex sandbox mode %q", d.Sandbox)
	}
	return d, nil
}
