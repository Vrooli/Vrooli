//go:build linux

package credentials

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// keyringUnit is the systemd user unit that owns gnome-keyring-daemon on
// distributions that ship one. Its absence is a supported host shape, not a
// fault: some sessions start the daemon from an XDG autostart entry or from the
// display manager's PAM stack, and neither is restartable without a re-login.
const keyringUnit = "gnome-keyring-daemon.service"

// unitProbeTimeout bounds the two read-only systemctl queries. A user manager
// that is itself wedged must not convert a bounded repair into a hang — the
// whole reason this ladder exists is that an unbounded probe was costing eight
// seconds on every test run.
const unitProbeTimeout = tuning.HealthCheckTimeout

// reloadMargin is added to the unit's own stop timeout to get the restart
// budget, and reloadFallback is used when that timeout cannot be read.
//
// The budget is derived rather than fixed because a fixed one produced a false
// negative on the very first live run: a wedged gnome-keyring-daemon ignores
// SIGTERM, so systemd sits in stop-sigterm for the unit's TimeoutStopUSec (90 s
// by default) before escalating to SIGKILL and starting a replacement. A 15 s
// budget expired first and this rung reported "failed" while the restart went
// on to succeed in the background. Reporting a successful repair as a failure
// is the same class of defect as the false green this ladder was built to
// remove, so the budget follows the unit rather than a guess.
const (
	reloadMargin   = tuning.ReloadFallbackGracePeriod
	reloadFallback = tuning.ExtendedOperationTimeout
)

func platformReloadCredentialDaemon(ctx context.Context) ReloadOutcome {
	// Ask the shared host-fact authority rather than re-deriving init-system
	// facts here. hostinventory is the project's single answer to "what is this
	// machine", and a second opinion in this file would drift from it.
	facts := hostinventory.CollectPlatformFacts(ctx)
	if !facts.SupportsSystemd {
		return ReloadOutcome{
			Status: RungBlocked,
			Detail: fmt.Sprintf("this host's init system is %q, so there is no systemd user manager to restart the credential daemon through", initSystemLabel(facts.InitSystem)),
			Remedy: relogRemedy("the credential daemon cannot be restarted without a session manager"),
		}
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		return ReloadOutcome{
			Status: RungBlocked,
			Detail: "systemd is reported as supported but systemctl is not on PATH, so no unit can be addressed",
			Remedy: relogRemedy("systemctl is unavailable to this process"),
		}
	}

	present, detail := keyringUnitPresent(ctx)
	if !present {
		return ReloadOutcome{
			Status: RungBlocked,
			Detail: detail,
			Remedy: relogRemedy("no systemd user unit owns gnome-keyring-daemon on this host"),
		}
	}

	action := "systemctl --user restart " + keyringUnit
	budget := reloadBudget(ctx)
	runCtx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "systemctl", "--user", "restart", keyringUnit)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if runCtx.Err() != nil {
			return ReloadOutcome{
				Status: RungFailed,
				Action: action,
				Detail: fmt.Sprintf("the restart did not complete within %s (the unit's own stop timeout plus %s); the systemd user manager may itself be wedged", budget, reloadMargin),
			}
		}
		return ReloadOutcome{
			Status: RungFailed,
			Action: action,
			Detail: "restart failed: " + firstLine(string(output)),
		}
	}
	// Deliberately not RungRepaired. Whether anything was actually fixed is
	// decided by the live re-probe the caller runs next, never by exit status.
	return ReloadOutcome{Status: reloadApplied, Action: action, Detail: "restart accepted by the systemd user manager"}
}

// keyringUnitPresent reports whether the user manager knows the keyring unit.
// It queries rather than assuming, because a host without GNOME Keyring is a
// normal host and must not be told to restart a unit that does not exist.
func keyringUnitPresent(ctx context.Context) (bool, string) {
	probeCtx, cancel := context.WithTimeout(ctx, unitProbeTimeout)
	defer cancel()
	cmd := exec.CommandContext(probeCtx, "systemctl", "--user", "list-unit-files", "--no-legend", keyringUnit)
	output, err := cmd.Output()
	if err != nil {
		if probeCtx.Err() != nil {
			return false, fmt.Sprintf("the systemd user manager did not answer a unit query within %s", unitProbeTimeout)
		}
		return false, "the systemd user manager could not be queried for " + keyringUnit
	}
	if strings.TrimSpace(string(output)) == "" {
		return false, "no " + keyringUnit + " exists in the systemd user manager; this session starts the keyring another way"
	}
	return true, ""
}

// reloadBudget reads how long systemd will wait for the unit to stop before it
// escalates to SIGKILL, and allows that plus a margin for the subsequent start.
func reloadBudget(ctx context.Context) time.Duration {
	probeCtx, cancel := context.WithTimeout(ctx, unitProbeTimeout)
	defer cancel()
	output, err := exec.CommandContext(probeCtx, "systemctl", "--user", "show", keyringUnit, "-p", "TimeoutStopUSec", "--value").Output()
	if err != nil {
		return reloadFallback
	}
	stop, ok := parseSystemdDuration(strings.TrimSpace(string(output)))
	if !ok {
		return reloadFallback
	}
	return stop + reloadMargin
}

// parseSystemdDuration reads systemd's human-readable duration rendering
// ("1min 30s", "90s", "infinity"). An unbounded stop timeout is deliberately
// not honored: waiting forever is what this whole ladder exists to avoid.
func parseSystemdDuration(value string) (time.Duration, bool) {
	if value == "" || value == "infinity" {
		return 0, false
	}
	var total time.Duration
	for _, part := range strings.Fields(value) {
		// Go's time.ParseDuration understands "30s" and "1m" but not systemd's
		// "min" spelling or its microsecond suffix.
		part = strings.Replace(part, "min", "m", 1)
		part = strings.Replace(part, "us", "µs", 1)
		parsed, err := time.ParseDuration(part)
		if err != nil {
			return 0, false
		}
		total += parsed
	}
	return total, total > 0
}

func initSystemLabel(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}

func firstLine(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "no output"
	}
	if index := strings.IndexByte(value, '\n'); index >= 0 {
		return value[:index]
	}
	return value
}
