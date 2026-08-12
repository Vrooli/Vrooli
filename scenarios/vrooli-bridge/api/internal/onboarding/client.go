// Package onboarding is the transport-neutral contract between vrooli-bridge
// and vrooli-onboarding. Bridge owns reaching a node; this package owns the
// stable selection document and the machine-readable result of applying it.
package onboarding

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// Selection is intentionally capability-shaped rather than an operator-state
// document. It can therefore cross a deployment boundary without coupling
// bridge to onboarding's persistence schema.
type Selection struct {
	Scenarios         []string        `json:"scenarios,omitempty"`
	OptionalResources []string        `json:"optional_resources,omitempty"`
	Host              HostSelection   `json:"host,omitempty"`
	OperatingMode     map[string]Mode `json:"operating_mode,omitempty"`
	Apply             bool            `json:"apply,omitempty"`
}

// HandoffRequest is the only identity Bridge sends across the scenario
// boundary. It deliberately contains no credentials or Bridge persistence
// details; onboarding remains the authority for the returned selection.
type HandoffRequest struct {
	MachineID string `json:"machine_id"`
	NodeID    string `json:"node_id"`
	NodeKind  string `json:"node_kind"`
}

// HandoffClient is the optional cross-scenario seam. A nil client means the
// legacy Bridge-owned setup profile remains in force.
type HandoffClient interface {
	Resolve(ctx context.Context, request HandoffRequest) (Selection, error)
}

// HTTPHandoffClient calls the onboarding scenario's stable JSON surface. The
// endpoint is configured by the operator; Bridge does not assume onboarding is
// installed or reachable.
type HTTPHandoffClient struct {
	Endpoint string
	Client   *http.Client
}

func (c HTTPHandoffClient) Resolve(ctx context.Context, request HandoffRequest) (Selection, error) {
	if strings.TrimSpace(c.Endpoint) == "" {
		return Selection{}, nil
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return Selection{}, fmt.Errorf("encode onboarding handoff: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return Selection{}, fmt.Errorf("create onboarding handoff request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return Selection{}, fmt.Errorf("onboarding handoff: %w", err)
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if readErr != nil {
		return Selection{}, fmt.Errorf("read onboarding handoff: %w", readErr)
	}
	if resp.StatusCode/100 != 2 {
		return Selection{}, fmt.Errorf("onboarding handoff returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var selection Selection
	if err := json.Unmarshal(body, &selection); err != nil {
		return Selection{}, fmt.Errorf("decode onboarding selection: %w", err)
	}
	return selection, nil
}

type HostSelection struct {
	Tools      []string `json:"tools,omitempty"`
	Safeguards []string `json:"safeguards,omitempty"`
}

type Mode struct {
	AutoRestart bool `json:"auto_restart"`
}

// FromSetupProfile converts the existing bridge profile flags into the stable
// onboarding document. Empty profiles intentionally return nil: old callers
// that request only pairing/bootstrap retain their existing behavior.
func FromSetupProfile(scenarios, resources string, includeOptional bool) (Selection, bool) {
	selection := Selection{Apply: true}
	selection.Scenarios = append(selection.Scenarios, splitNames(scenarios)...)
	selection.OptionalResources = append(selection.OptionalResources, splitNames(resources)...)
	return selection, len(selection.Scenarios) > 0 || len(selection.OptionalResources) > 0
}

func splitNames(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if name := strings.TrimSpace(part); name != "" && name != "all" && name != "none" && name != "enabled" {
			result = append(result, name)
		}
	}
	return result
}

// Target is the minimum connection information needed by a bridge transport.
type Target struct {
	Host string
	Port int
	User string
	Key  string
}

type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// Runner is implemented by bridge's SSH transport. Keeping it here avoids
// importing SSH details into the selection contract and makes fake remotes
// straightforward to test.
type Runner interface {
	Run(ctx context.Context, target Target, command string) (Result, error)
}

// Apply sends the selection through the remote CLI without putting JSON in
// argv. The temporary file is private, removed on every exit path, and the
// selection contains no credential values.
func Apply(ctx context.Context, runner Runner, target Target, selection Selection) (Result, error) {
	data, err := json.Marshal(selection)
	if err != nil {
		return Result{}, fmt.Errorf("encode onboarding selection: %w", err)
	}
	payload := base64.StdEncoding.EncodeToString(data)
	command := "tmp=$(mktemp); trap 'rm -f \"$tmp\"' EXIT; printf '%s' " + shellQuote(payload) +
		" | base64 --decode > \"$tmp\"; vrooli-onboarding wizard apply --selection \"$tmp\" --json"
	return runner.Run(ctx, target, command)
}

// Readiness reads the remote onboarding report after an apply. The report is
// intentionally produced by onboarding itself so Bridge never reimplements
// credential, host-tool, or safeguard readiness rules.
func Readiness(ctx context.Context, runner Runner, target Target) (Result, error) {
	return runner.Run(ctx, target, "vrooli-onboarding readiness --json")
}

// ApplyAndReadiness is the complete remote contract exposed to Bridge: apply
// the capability-shaped selection, then return the authoritative readiness
// report and its exit code.
func ApplyAndReadiness(ctx context.Context, runner Runner, target Target, selection Selection) (Result, error) {
	result, err := Apply(ctx, runner, target, selection)
	if err != nil || result.ExitCode != 0 {
		return result, err
	}
	return Readiness(ctx, runner, target)
}

// ReadinessExitCode is the bridge-facing policy: onboarding's exit code is
// authoritative, while an unavailable remote process is a distinct failure.
func ReadinessExitCode(result Result, transportErr error) (int, error) {
	if transportErr != nil {
		return 75, transportErr
	}
	if result.ExitCode < 0 {
		return 70, fmt.Errorf("remote onboarding returned an invalid exit code %s", strconv.Itoa(result.ExitCode))
	}
	return result.ExitCode, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
