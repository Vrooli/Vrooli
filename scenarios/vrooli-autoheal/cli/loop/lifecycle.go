package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/vrooli/envkit-go"
	"github.com/vrooli/repo-contract-go/cliinvoke"
)

// healError is a heal attempt's failure carrying the class the state
// machine routes on. Usage and BinaryMissing are non-healable: nothing about
// the host changes between identical retries. Everything else is retried
// with backoff, and Lifecycle failures first pass through the recovery floor.
type healError struct {
	Class cliinvoke.Class
	Err   error
}

func (e *healError) Error() string { return fmt.Sprintf("%s: %v", e.Class, e.Err) }
func (e *healError) Unwrap() error { return e.Err }

// Healable reports whether waiting and retrying could change the outcome.
func (e *healError) Healable() bool {
	return e.Class != cliinvoke.Usage && e.Class != cliinvoke.BinaryMissing
}

// invokeVrooli is the ONLY way this loop runs the CLI. cliinvoke owns the
// timeout, the WaitDelay tolerance for inherited pipes (the 2026-08-01
// outage), and the failure classification; the loop owns only the argv, the
// environment boundary, and the context that cancels the child on shutdown.
func invokeVrooli(ctx context.Context, config *Config, stdout, stderr io.Writer, subArgs ...string) cliinvoke.Result {
	return cliinvoke.Run(ctx, cliinvoke.Invocation{
		Binary:    config.VrooliCmdPath,
		Args:      subArgs,
		Env:       envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env{fmt.Sprintf("VROOLI_ROOT=%s", config.VrooliRoot)}),
		Timeout:   cliinvoke.DefaultTimeout,
		WaitDelay: cliinvoke.DefaultWaitDelay,
		Stdout:    stdout,
		Stderr:    stderr,
	})
}

// ensureAPIRunning verifies the API is answering as itself and heals only
// when it is not. The Heal state calls it after its backoff, because by then
// the runtime supervisor may have brought the API up on its own, and a
// restart would only interrupt it.
func ensureAPIRunning(ctx context.Context, config *Config, reason string) error {
	found := detectAPIPort(ctx, config)
	if found.Verified != "" && config.adoptPort(ctx, found.Verified) {
		return nil
	}
	if found.Pending != "" {
		if err := waitForAPIHealthy(ctx, config, found.Pending); err == nil {
			return nil
		} else if ctx.Err() != nil {
			return err
		}
	}
	return heal(ctx, config, reason)
}

// heal is the one path that changes the API's lifecycle. It picks the verb
// from what is observably running, runs it through the recovery floor, and
// then waits for the API to identify itself. The reason is logged, not
// acted on: whatever the trigger, the remedy is the same.
func heal(ctx context.Context, config *Config, reason string) error {
	log.Printf("heal: %s", reason)
	if config.VrooliCmdPath == "" {
		return &healError{Class: cliinvoke.BinaryMissing, Err: config.VrooliResolveErr}
	}
	verb := "start"
	if isScenarioRunning(ctx, config) {
		verb = "restart"
	}
	if err := runLifecycleWithRecovery(ctx, config, verb); err != nil {
		return err
	}
	return waitForAPIHealthy(ctx, config, "")
}

// runLifecycleWithRecovery runs `vrooli scenario <verb> <scenario>` and, when
// it fails with a healable dependency-drift signature, applies the recovery
// floor and retries exactly once.
//
// The single retry is deliberate. Recovery either changes tracked files (in
// which case one more attempt is the whole point) or it does not (in which
// case further attempts fail identically). Repetition beyond that is the
// state machine's backoff, not this function's.
//
// A non-healable class returns before the floor is consulted: a usage error
// or a missing binary carries no drift signature and must not spend a
// breaker slot.
func runLifecycleWithRecovery(ctx context.Context, config *Config, verb string) error {
	argv := cliinvoke.ScenarioLifecycle(verb, config.ScenarioName, true)
	res := invokeVrooli(ctx, config, nil, nil, argv...)
	combined := string(res.Combined())
	if res.Class == cliinvoke.OK {
		log.Printf("%s output: %s", titleVerb(verb), strings.TrimSpace(combined))
		// A clean start means any earlier drift is resolved; release the
		// breaker so a future, unrelated failure gets a full attempt budget.
		resetSelfHealState(selfHealStatePath())
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	failure := &healError{Class: res.Class, Err: fmt.Errorf("%s command failed: %w\nOutput: %s", verb, res.Err, combined)}
	if !failure.Healable() || res.Class == cliinvoke.Timeout {
		return failure
	}

	// The output we already hold is the only place the healable signature
	// appears. Before this existed, the loop discarded it.
	outcome := attemptSelfHeal(ctx, config, combined)
	if !outcome.Attempted {
		log.Printf("Recovery floor did not engage: %s", outcome.Detail)
		return failure
	}
	log.Printf("Recovery floor engaged: %s", outcome.Detail)
	if !outcome.Healed {
		return failure
	}

	log.Printf("Recovery floor healed dependency drift; retrying scenario %s once", verb)
	retryRes := invokeVrooli(ctx, config, nil, nil, argv...)
	if retryRes.Class != cliinvoke.OK {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return &healError{Class: retryRes.Class, Err: fmt.Errorf("%s failed again after recovery: %w\nOutput: %s", verb, retryRes.Err, retryRes.Combined())}
	}
	log.Printf("%s succeeded after recovery floor repair", titleVerb(verb))
	return nil
}

// waitForAPIHealthy polls until the autoheal API identifies itself, looking
// first at pending (a port the registry named before anything answered) and
// then at full detection. A port is adopted only through adoptPort.
func waitForAPIHealthy(ctx context.Context, config *Config, pending string) error {
	deadline := time.Now().Add(config.StartupTimeout)
	lastState := "no port detected yet"
	for {
		if pending != "" && config.adoptPort(ctx, pending) {
			return nil
		}
		found := detectAPIPort(ctx, config)
		if found.Verified != "" && config.adoptPort(ctx, found.Verified) {
			return nil
		}
		if found.Pending != "" {
			pending = found.Pending
		}
		if pending != "" {
			lastState = fmt.Sprintf("port %s named but autoheal does not answer there yet", pending)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return &healError{Class: cliinvoke.Lifecycle, Err: fmt.Errorf("API failed to become healthy within %v: %s", config.StartupTimeout, lastState)}
		}
		log.Printf("Waiting for API... (%v left) - %s", remaining.Round(time.Second), lastState)
		if !sleepCtx(ctx, min(config.HealthCheckInterval, remaining)) {
			return ctx.Err()
		}
	}
}

// isScenarioRunning reports whether the lifecycle engine has a live process
// for the scenario, or the API answers as itself somewhere.
func isScenarioRunning(ctx context.Context, config *Config) bool {
	for _, pidFile := range processRegistryPaths(config, "start-api.pid") {
		data, err := os.ReadFile(pidFile)
		if err != nil {
			continue
		}
		if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && isProcessRunning(pid) {
			return true
		}
	}
	return detectAPIPort(ctx, config).Verified != ""
}

// isProcessRunning checks if a process with the given PID exists.
func isProcessRunning(pid int) bool {
	if runtime.GOOS == "windows" {
		output, err := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH").Output()
		return err == nil && strings.Contains(string(output), strconv.Itoa(pid))
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}

// TickResponse represents the API response from /tick.
type TickResponse struct {
	Success bool   `json:"success"`
	Status  string `json:"status"`
	Summary struct {
		Total    int `json:"total"`
		OK       int `json:"ok"`
		Warning  int `json:"warning"`
		Critical int `json:"critical"`
	} `json:"summary"`
}

// tickTimeout bounds one /tick call. Ticks run every check; a busy database
// makes them slow, not hung.
const tickTimeout = 5 * time.Minute

// runTick calls the /tick endpoint.
func runTick(ctx context.Context, config *Config) (*TickResponse, error) {
	if config.TickEndpoint == "" {
		return nil, fmt.Errorf("tick endpoint not configured")
	}
	requestURL, err := url.Parse(config.TickEndpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid tick endpoint URL: %w", err)
	}
	if err := validateLocalEndpoint(requestURL.String()); err != nil {
		return nil, err
	}
	query := requestURL.Query()
	query.Set("compact", "true")
	requestURL.RawQuery = query.Encode()

	reqCtx, cancel := context.WithTimeout(ctx, tickTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, requestURL.String(), bytes.NewReader(nil))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	var result TickResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}
