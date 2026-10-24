package scenariocli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/cli-core/cliutil"

	scenarioapp "github.com/vrooli/vrooli/internal/app/scenario"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// `vrooli scenario wait` — the single-blocking-call primitive of the scenario
// start wait contract. Attach to an in-flight start (or evaluate current
// health when none), return ONCE with the verdict in the exit code. Agents
// must never loop `scenario status`; they block here.

// WaitResponse mirrors scenarioapp.WaitResponse at the CLI layer.
type WaitResponse = scenarioapp.WaitResponse

// WaitVerdictDetached is the CLI-layer verdict for a Ctrl-C detach: the wait
// stopped observing but the awaited start (if any) continues. Exit 0 —
// detaching is not a failure (GCT precedent).
const WaitVerdictDetached = "detached"

// ParkScenarioWait asks agent-manager to park the current run on the
// scenario's start operation instead of blocking inline. Mirrors test-genie
// `runs wait`: outside an AM run it is a no-op (parked=false); park failures
// inside an AM run degrade to the inline wait with a stderr warning.
func ParkScenarioWait(stderr io.Writer, slug string) (message string, parked bool) {
	key, err := scenarioruntime.ParseInstanceKey(slug, "")
	if err != nil {
		return "", false
	}
	park, handled, perr := cliutil.ParkForAwait(cliutil.ParkRequest{
		Producer: cliutil.ParkProducerLifecycle,
		Key:      key.Scenario + "/" + key.Variant,
	})
	if !handled {
		return "", false
	}
	if perr != nil {
		fmt.Fprintf(stderr, "agent-manager park unavailable (%v) — waiting inline instead\n", perr)
		return "", false
	}
	return park.Message, true
}

// Eager-wait ledger — mirrors test-genie's wait-attempts guardrail (bounded
// re-wait timestamp file, 30s window, stderr warning). Mirrored rather than
// extracted because test-genie changes are out of scope for the wait-contract
// plan; unify into cli-core/cliutil as a follow-up.
var eagerScenarioWaitWindow = tuning.EagerScenarioWaitWindow()

func scenarioWaitStatePath() string {
	base, err := os.UserCacheDir()
	if err != nil || strings.TrimSpace(base) == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "vrooli", "scenario", "wait-attempts.json")
}

func readScenarioWaitAttempts(path string) map[string]int64 {
	attempts := map[string]int64{}
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, &attempts)
	}
	return attempts
}

func writeScenarioWaitAttempts(path string, attempts map[string]int64) {
	if err := os.MkdirAll(filepath.Dir(path), tuning.PermDir); err != nil {
		return
	}
	data, err := json.MarshalIndent(attempts, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, tuning.PermFile)
}

// WarnIfEagerScenarioWait warns (stderr) when a wait for the same scenario
// ran within the eager window — the poll-loop smell this contract exists to
// end — and records this attempt.
func WarnIfEagerScenarioWait(stderr io.Writer, slug string, now time.Time) {
	path := scenarioWaitStatePath()
	attempts := readScenarioWaitAttempts(path)
	if prevUnix, ok := attempts[slug]; ok {
		if elapsed := now.Sub(time.Unix(prevUnix, 0)); elapsed >= 0 && elapsed <= eagerScenarioWaitWindow {
			fmt.Fprintf(stderr, "recent wait detected for %s (%s ago). Do not poll with short repeated waits — block once with `vrooli scenario wait %s --json` and trust the exit code.\n",
				slug, elapsed.Round(time.Second), slug)
		}
	}
	attempts[slug] = now.Unix()
	writeScenarioWaitAttempts(path, attempts)
}

// ClearScenarioWaitAttempt drops the ledger entry once a wait resolved to a
// terminal verdict (re-waiting after resolution is legitimate).
func ClearScenarioWaitAttempt(slug string) {
	path := scenarioWaitStatePath()
	attempts := readScenarioWaitAttempts(path)
	delete(attempts, slug)
	writeScenarioWaitAttempts(path, attempts)
}

func waitHelpText() string {
	spec := commandSpec(CommandWait)
	spec.Help.Description = "Block ONCE until the scenario's in-flight start/restart reaches a terminal state, then exit with the verdict:\n" +
		"  0    healthy (or running for scenarios with no health checks)\n" +
		"  1    start failed / not running / abandoned\n" +
		"  2    degraded-after-timeout success (usable, but non-critical checks failing)\n" +
		"  124  --timeout ceiling elapsed — this wait detached; the start itself keeps running\n\n" +
		"With no start in flight the current runtime health is evaluated and returned immediately.\n" +
		"Anti-polling: do NOT loop `vrooli scenario status`; call wait once (optionally with --timeout)\n" +
		"and trust the exit code. --json emits one vrooli.cli.v1.ScenarioWaitResponse document on stdout;\n" +
		"all hints go to stderr."
	return commandtree.SpecHelpText("", "vrooli scenario wait", spec)
}

// ScenarioWaitResponseMessage maps a WaitResponse onto its wire message.
func ScenarioWaitResponseMessage(resp WaitResponse) *cliv1.ScenarioWaitResponse {
	return &cliv1.ScenarioWaitResponse{
		Success:       resp.Success,
		Scenario:      resp.Scenario,
		Verdict:       resp.Verdict,
		ExitCode:      int32(resp.ExitCode),
		Source:        resp.Source,
		WaitedSeconds: int32(resp.WaitedSeconds),
		Operation:     ScenarioStartOperationMessage(resp.Operation),
	}
}

// RenderWaitResponse writes the single wait document (JSON) or a compact
// human verdict line, then converts the verdict into the process exit code
// via a silent ExitCodeError (the document/line IS the report; no duplicate
// error text).
func RenderWaitResponse(w io.Writer, format cliout.Format, resp WaitResponse) error {
	if resp.ParkedMessage != "" {
		_, err := fmt.Fprintln(w, resp.ParkedMessage)
		return err
	}
	if err := cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ScenarioWaitResponseMessage(resp)) }, func(w io.Writer) error {
		suffix := ""
		if resp.Error != "" {
			suffix = ": " + resp.Error
		}
		if _, err := fmt.Fprintf(w, "%s: %s (source=%s, waited %ds)%s\n", resp.Scenario, resp.Verdict, resp.Source, resp.WaitedSeconds, suffix); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	if resp.ExitCode != 0 {
		return VerdictExitError{Code: resp.ExitCode}
	}
	return nil
}

// VerdictExitError carries a verdict exit code up through the dispatcher
// without printing anything: the rendered document/line IS the report.
// (Local rather than rootcli.ExitCodeError because rootcli imports this
// package.) It satisfies the dispatcher's ExitCode()/Silent() interfaces.
type VerdictExitError struct{ Code int }

func (e VerdictExitError) Error() string { return fmt.Sprintf("exit code %d", e.Code) }

func (e VerdictExitError) ExitCode() int { return e.Code }

// Silent suppresses the dispatcher's error printer.
func (e VerdictExitError) Silent() bool { return true }
