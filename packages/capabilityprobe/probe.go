// Package capabilityprobe probes host capabilities without importing the
// control plane. It is deliberately standalone so the Bridge node agent can
// cross-compile it for every supported OS and architecture.
package capabilityprobe

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type State string

const (
	Ready         State = "ready"
	Missing       State = "missing"
	NotApplicable State = "not_applicable"
	Unknown       State = "unknown"
)

type Definition struct {
	Capability string
	ID         string
	Label      string
	Command    string
	VersionArg []string
}

type Observation struct {
	Capability string    `json:"capability"`
	ID         string    `json:"id"`
	Label      string    `json:"label"`
	State      State     `json:"state"`
	Path       string    `json:"path,omitempty"`
	Version    string    `json:"version,omitempty"`
	ProbedAt   time.Time `json:"probed_at"`
	Detail     string    `json:"detail,omitempty"`
}

// AITools is generated from internal/tools/*/tool.json where capability is
// ai-cli. Keep this table sorted by ID; the drift test protects the contract.
var AITools = []Definition{
	{Capability: "ai-cli", ID: "agy", Label: "Antigravity", Command: "agy", VersionArg: []string{"--version"}},
	{Capability: "ai-cli", ID: "claude", Label: "Claude Code", Command: "claude", VersionArg: []string{"--version"}},
	{Capability: "ai-cli", ID: "codex", Label: "Codex", Command: "codex", VersionArg: []string{"--version"}},
	{Capability: "ai-cli", ID: "grok", Label: "Grok", Command: "grok", VersionArg: []string{"--version"}},
	{Capability: "ai-cli", ID: "opencode", Label: "OpenCode", Command: "opencode", VersionArg: []string{"--version"}},
}

type LookPath func(string) (string, error)
type RunVersion func(context.Context, string, []string) (string, error)

func Probe(ctx context.Context, definitions []Definition) []Observation {
	return ProbeWith(ctx, definitions, ManagedLookPath, runVersion, time.Now)
}

// ManagedLookPath mirrors the runtime PATH contract for long-lived node
// services. Native service managers commonly omit the interactive user's
// PATH, while Vrooli-owned CLIs are deliberately installed in these two
// user-owned directories. Prefer the real PATH and then resolve those exact
// managed locations; never scan arbitrary directories or execute a shell.
func ManagedLookPath(binary string) (string, error) {
	if path, err := exec.LookPath(binary); err == nil {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	for _, dir := range []string{filepath.Join(home, ".local", "bin"), filepath.Join(home, ".vrooli", "bin")} {
		candidate := filepath.Join(dir, binary)
		if info, statErr := os.Stat(candidate); statErr == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0 {
			return candidate, nil
		}
	}
	return "", exec.ErrNotFound
}

func ProbeWith(ctx context.Context, definitions []Definition, lookPath LookPath, version RunVersion, now func() time.Time) []Observation {
	if now == nil {
		now = time.Now
	}
	observed := now().UTC()
	result := make([]Observation, 0, len(definitions))
	for _, definition := range definitions {
		item := Observation{Capability: definition.Capability, ID: definition.ID, Label: definition.Label, State: Unknown, ProbedAt: observed}
		path, err := lookPath(definition.Command)
		if err != nil || path == "" {
			item.State = Missing
			item.Detail = "command is not on PATH"
			result = append(result, item)
			continue
		}
		item.Path = path
		if version != nil {
			value, err := version(ctx, path, definition.VersionArg)
			if err != nil {
				item.State = Unknown
				item.Detail = "command was found but its version could not be read"
			} else {
				item.State = Ready
				item.Version = strings.TrimSpace(value)
			}
		} else {
			item.State = Ready
		}
		result = append(result, item)
	}
	return result
}

func runVersion(ctx context.Context, path string, args []string) (string, error) {
	return stringOutput(exec.CommandContext(ctx, path, args...).CombinedOutput())
}

// stringOutput exists to keep the version runner's return shape explicit.
func stringOutput(output []byte, err error) (string, error) { return string(output), err }
