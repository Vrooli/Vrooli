package baseline

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

// defaultProbeTimeout bounds one --probe-cmd run. Live has already restarted
// onto the candidate when the probe runs, so a hung probe must not hold it in
// that unverified state; a timeout counts as a failed probe and rolls back.
const defaultProbeTimeout = 15 * time.Minute

// probeDetailMax bounds how much probe output a step line carries.
const probeDetailMax = 600

// runLiveProbe runs the caller's --probe-cmd through `sh -c` against live and
// returns ("", true) when it exits 0, or a bounded failure detail. The scenario
// is removed from VROOLI_SHADOW_SCENARIOS for the probe: a caller working inside
// the shadow engagement routes nested CLI calls to the shadow, and the probe
// must exercise live. The call goes through the runCommand seam (via `env`) so
// tests record it in order with the other promote steps.
func runLiveProbe(ctx context.Context, scenario, cmd string, timeout time.Duration) (string, bool) {
	if timeout <= 0 {
		timeout = defaultProbeTimeout
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	args := []string{liveProbeShadowEnv(scenario), "sh", "-c", cmd}
	_, err := runCommand(probeCtx, "env", args...)
	if err == nil {
		return "", true
	}
	if errors.Is(probeCtx.Err(), context.DeadlineExceeded) {
		return fmt.Sprintf("live probe timed out after %s", timeout), false
	}
	detail := strings.TrimPrefix(err.Error(), "env "+strings.Join(args, " ")+": ")
	return "live probe failed: " + boundedTail(detail, probeDetailMax), false
}

// liveProbeShadowEnv returns the VROOLI_SHADOW_SCENARIOS assignment for the
// probe: the caller's ambient list without scenario.
func liveProbeShadowEnv(scenario string) string {
	bare := cliutil.BareScenarioName(scenario)
	var keep []string
	for name := range cliutil.ShadowedScenarios() {
		if name != bare {
			keep = append(keep, name)
		}
	}
	sort.Strings(keep)
	return cliutil.EnvShadowScenarios + "=" + strings.Join(keep, ",")
}

// boundedTail flattens multi-line output to its last lines and caps it at max
// runes, so a failing journey's log fits on one step line.
func boundedTail(s string, max int) string {
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) > 3 {
		lines = lines[len(lines)-3:]
	}
	out := []rune(strings.Join(lines, " | "))
	if len(out) > max {
		return "…" + string(out[len(out)-max:])
	}
	return string(out)
}
