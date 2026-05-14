package adoptions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// SwarmManagerCLIReporter files drift backlog items by exec'ing
// `swarm-manager backlog create --data <json>`. Wraps an external tool
// rather than calling its HTTP API directly per the project's CLI-only
// discipline ([feedback_skills_use_cli_never_api.md]) — the library
// only invokes swarm-manager through its scenario CLI.
type SwarmManagerCLIReporter struct {
	// BinaryPath overrides the default "swarm-manager" lookup on PATH.
	// Empty string means PATH lookup. Tests can point this at a script
	// that captures the args; production leaves it unset.
	BinaryPath string

	// Runner exec's the CLI. Defaults to (*exec.Cmd).CombinedOutput on
	// the underlying command. Tests inject a fake that captures the
	// arguments and returns a canned response without spawning a
	// process.
	Runner CommandRunner

	// Timeout caps a single Report call. Zero means no timeout beyond
	// the caller's context.
	Timeout time.Duration
}

// CommandRunner is the seam SwarmManagerCLIReporter uses to invoke the
// CLI. Production wires defaultRunner (real exec); tests inject a fake.
type CommandRunner interface {
	Run(ctx context.Context, binary string, args []string) ([]byte, error)
}

// NewSwarmManagerCLIReporter constructs a reporter bound to the
// caller-supplied runner. Pass nil to get the real exec runner.
func NewSwarmManagerCLIReporter(runner CommandRunner) *SwarmManagerCLIReporter {
	if runner == nil {
		runner = defaultRunner{}
	}
	return &SwarmManagerCLIReporter{Runner: runner, Timeout: 30 * time.Second}
}

var _ DriftReporter = (*SwarmManagerCLIReporter)(nil)

func (r *SwarmManagerCLIReporter) Report(ctx context.Context, ev DriftEvent) (DriftReport, error) {
	if r.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.Timeout)
		defer cancel()
	}
	binary := r.BinaryPath
	if binary == "" {
		binary = "swarm-manager"
	}
	payload, err := buildDriftBacklogPayload(ev)
	if err != nil {
		return DriftReport{}, err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return DriftReport{}, fmt.Errorf("marshal swarm-manager payload: %w", err)
	}
	args := []string{"backlog", "create", "--json", "--data", string(data)}
	out, err := r.Runner.Run(ctx, binary, args)
	if err != nil {
		return DriftReport{}, fmt.Errorf("swarm-manager backlog create: %w (output: %s)", err, strings.TrimSpace(string(out)))
	}
	ref, err := parseCreatedRef(out)
	if err != nil {
		return DriftReport{}, fmt.Errorf("parse swarm-manager response: %w", err)
	}
	return DriftReport{Ref: ref}, nil
}

// buildDriftBacklogPayload composes the JSON `--data` swarm-manager
// expects (kind/name/title/description/tags). Kept pure so tests can
// pin the shape without exec'ing a binary.
func buildDriftBacklogPayload(ev DriftEvent) (map[string]any, error) {
	libID := strings.TrimSpace(ev.LibraryID)
	if libID == "" {
		libID = strings.TrimSpace(ev.ComponentID)
	}
	if libID == "" {
		return nil, fmt.Errorf("drift event missing component identity")
	}
	statusLabel := string(ev.Status)
	if statusLabel == "" {
		statusLabel = "drift"
	}
	scenario := strings.TrimSpace(ev.Scenario)
	if scenario == "" {
		scenario = "unknown-scenario"
	}
	name := slugifyDriftName(libID, scenario)
	title := fmt.Sprintf("Library component %s drift in %s", libID, scenario)
	var descBuilder strings.Builder
	fmt.Fprintf(&descBuilder, "The adopted copy of `%s` in scenario `%s` is %s.\n\n", libID, scenario, statusLabel)
	if ev.AdoptedPath != "" {
		fmt.Fprintf(&descBuilder, "- adopted path: `%s`\n", ev.AdoptedPath)
	}
	if ev.AdoptedVersion != "" {
		fmt.Fprintf(&descBuilder, "- adopted version: `%s`\n", ev.AdoptedVersion)
	}
	if ev.LibraryVersion != "" {
		fmt.Fprintf(&descBuilder, "- library version: `%s`\n", ev.LibraryVersion)
	}
	if ev.StatusDetail != "" {
		fmt.Fprintf(&descBuilder, "- detail: %s\n", ev.StatusDetail)
	}
	fmt.Fprintf(&descBuilder, "- adoption id: `%s`\n", ev.AdoptionID)
	descBuilder.WriteString("\nFiled by react-component-library refresh.")

	return map[string]any{
		"name":        name,
		"title":       title,
		"description": descBuilder.String(),
		"kind":        "fix",
		"tags":        []string{"react-component-library", "drift", statusLabel, scenario},
	}, nil
}

var driftNameSanitizer = regexp.MustCompile(`[^a-z0-9-]+`)

func slugifyDriftName(libID, scenario string) string {
	base := strings.ToLower(libID + "-drift-" + scenario)
	base = strings.ReplaceAll(base, ":", "-")
	base = strings.ReplaceAll(base, "/", "-")
	base = driftNameSanitizer.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "drift"
	}
	if len(base) > 60 {
		base = base[:60]
	}
	return base
}

// parseCreatedRef extracts `<kind>/<name>` from the swarm-manager
// `backlog create --json` response. The response shape is
// `{"item":{"kind":"...","name":"..."}}`.
func parseCreatedRef(raw []byte) (string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return "", fmt.Errorf("empty response")
	}
	var resp struct {
		Item struct {
			Kind string `json:"kind"`
			Name string `json:"name"`
		} `json:"item"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}
	kind := strings.TrimSpace(resp.Item.Kind)
	name := strings.TrimSpace(resp.Item.Name)
	if kind == "" || name == "" {
		return "", fmt.Errorf("response missing item.kind / item.name")
	}
	return kind + "/" + name, nil
}

// defaultRunner is the production CommandRunner — straight exec.Cmd.
type defaultRunner struct{}

func (defaultRunner) Run(ctx context.Context, binary string, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	return cmd.CombinedOutput()
}
