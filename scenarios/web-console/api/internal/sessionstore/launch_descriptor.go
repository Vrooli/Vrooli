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
	ControlsVersion       int               `json:"controlsVersion,omitempty"`
	Agent                 string            `json:"agent"`
	LaunchMode            string            `json:"launchMode"`
	ControlMode           string            `json:"controlMode,omitempty"`
	ServerTransport       string            `json:"serverTransport,omitempty"`
	ProviderVersionRange  string            `json:"providerVersionRange,omitempty"`
	ReasoningEffort       string            `json:"reasoningEffort,omitempty"`
	ContextLimitTokens    int               `json:"contextLimitTokens,omitempty"`
	NetworkAccessDisabled bool              `json:"networkAccessDisabled,omitempty"`
	Model                 string            `json:"model,omitempty"`
	Profile               string            `json:"profile,omitempty"`
	ApprovalPolicy        string            `json:"approvalPolicy,omitempty"`
	Sandbox               string            `json:"sandbox,omitempty"`
	WorkspaceRoots        []string          `json:"workspaceRoots,omitempty"`
	Environment           map[string]string `json:"environment,omitempty"`
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
	if d.ControlsVersion == 1 {
		decoder := json.NewDecoder(strings.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&d); err != nil {
			return LaunchDescriptor{}, fmt.Errorf("invalid managed Codex controls: %w", err)
		}
	}
	if d.ControlsVersion != 0 && d.ControlsVersion != 1 {
		return LaunchDescriptor{}, fmt.Errorf("unsupported Codex controls version %d", d.ControlsVersion)
	}
	if d.ControlsVersion == 1 && (d.ApprovalPolicy != "never" || d.Profile != "" || (d.Sandbox != "danger-full-access" && !(d.Sandbox == "workspace-write" && d.NetworkAccessDisabled))) {
		return LaunchDescriptor{}, fmt.Errorf("invalid managed Codex permission controls")
	}
	if d.ReasoningEffort != "" && d.ReasoningEffort != "low" && d.ReasoningEffort != "medium" && d.ReasoningEffort != "high" && d.ReasoningEffort != "xhigh" {
		return LaunchDescriptor{}, fmt.Errorf("unsupported Codex reasoning effort %q", d.ReasoningEffort)
	}
	if d.ContextLimitTokens < 0 {
		return LaunchDescriptor{}, fmt.Errorf("invalid Codex context limit")
	}
	if d.NetworkAccessDisabled && (d.Sandbox != "workspace-write" || d.ApprovalPolicy != "never" || d.Profile != "") {
		return LaunchDescriptor{}, fmt.Errorf("network denial requires workspace-write, never approval, and no profile")
	}
	return d, nil
}
