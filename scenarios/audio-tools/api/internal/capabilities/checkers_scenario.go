package capabilities

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// ScenarioChecker verifies that a sibling Vrooli scenario is installed and
// running by shelling out to `vrooli scenario status <slug> --json`. It is
// the runtime probe behind every DependencyScenario entry in the registry,
// and the seam audio-tools (and any future connected scenario) will plug
// into once it ships.
//
// CLIPath/Args/Slug are injectable so tests can substitute a fake.
type ScenarioChecker struct {
	// Slug is the scenario directory name under scenarios/, e.g. "audio-tools".
	Slug string

	// CLIPath defaults to "vrooli" (resolved via PATH). Override in tests.
	CLIPath string

	// Args defaults to ["scenario", "status", Slug, "--json"]. Override only
	// when the CLI subcommand surface changes.
	Args []string

	// Run is the command-runner seam. Defaults to exec.CommandContext.
	Run func(ctx context.Context, name string, args ...string) ([]byte, error)

	// Timeout caps the probe latency. Defaults to 5s.
	Timeout time.Duration
}

func (c *ScenarioChecker) Check(ctx context.Context) (Status, string) {
	slug := c.Slug
	if slug == "" {
		return StatusUnavailable, "scenario slug not configured"
	}
	cli := c.CLIPath
	if cli == "" {
		cli = "vrooli"
	}
	args := c.Args
	if len(args) == 0 {
		args = []string{"scenario", "status", slug, "--json"}
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	run := c.Run
	if run == nil {
		run = func(ctx context.Context, name string, a ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, a...).Output()
		}
	}

	out, err := run(probeCtx, cli, args...)
	if err != nil {
		// Either the CLI is missing, the scenario directory does not exist,
		// or the scenario is not running. All three resolve to "not yet
		// available" for audio-tools' capability registry.
		return StatusUnavailable, "scenario not available (run `vrooli scenario start " + slug + "` once installed)"
	}

	// The vrooli CLI emits a status payload per scenario. Health is keyed by
	// the presence of a "healthy" status field, but we accept either the
	// new {status:"healthy"} shape or the older {health_status:"healthy"} one
	// without coupling to either schema strictly: any successful invocation
	// that mentions "healthy" counts as available; otherwise the scenario is
	// installed but not running.
	body := strings.ToLower(string(out))
	switch {
	case strings.Contains(body, `"healthy"`), strings.Contains(body, `"running"`):
		return StatusAvailable, "scenario is healthy"
	case strings.Contains(body, `"stopped"`), strings.Contains(body, `"not_running"`):
		return StatusUnavailable, "scenario is installed but stopped"
	default:
		// CLI succeeded but emitted an unfamiliar status — treat as unknown so
		// the UI does not falsely advertise the integration as active.
		return StatusUnknown, "scenario status unrecognised"
	}
}
