// Package main is the autoheal boot-recovery loop, built as the `loop`
// component of the vrooli-autoheal scenario by the lifecycle engine.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// loopState is where the state machine is. Every transition is logged and
// written to the status file, so the journal and the file agree.
type loopState int

const (
	// statePreflight: can this loop heal at all?
	statePreflight loopState = iota
	// stateDetect: is the autoheal API answering as itself somewhere?
	stateDetect
	// stateHeal: change the lifecycle, with backoff between attempts.
	stateHeal
	// stateVerify: one tick against a freshly adopted API.
	stateVerify
	// stateHealthy: tick on the interval, preflight every preflightEveryTicks.
	stateHealthy
	// stateDegraded: something is wrong that healing does not address; keep
	// ticking and re-run the preflight on every interval.
	stateDegraded
	// stateExit: the exit code is decided and the status file is written.
	stateExit
)

func (s loopState) String() string {
	switch s {
	case statePreflight:
		return "preflight"
	case stateDetect:
		return "detect"
	case stateHeal:
		return "heal"
	case stateVerify:
		return "verify"
	case stateHealthy:
		return "healthy"
	case stateDegraded:
		return "degraded"
	case stateExit:
		return "exit"
	}
	return "unknown"
}

const (
	// healBackoffMin and healBackoffMax bound the wait between healable heal
	// attempts. The first retry waits a minute; each further failure doubles
	// it; a return to Healthy resets it.
	healBackoffMin = time.Minute
	healBackoffMax = 15 * time.Minute
	// preflightEveryTicks is how often a Healthy loop re-checks that it can
	// still heal, so a CLI replaced under it is noticed within an hour.
	preflightEveryTicks = 60
)

// nextBackoff doubles the wait, from healBackoffMin up to healBackoffMax.
func nextBackoff(current time.Duration) time.Duration {
	if current < healBackoffMin {
		return healBackoffMin
	}
	return min(current*2, healBackoffMax)
}

// loop is the state machine and its memory.
type loop struct {
	config *Config
	status *statusWriter
	record loopStatus
	state  loopState
	// backoff is the wait before the next healable heal attempt.
	backoff time.Duration
	// healReason is why the next heal runs; logged, never acted on.
	healReason string
	// tickFailures counts consecutive failed ticks toward MaxFailures.
	tickFailures int
	// ticksSincePreflight counts ticks toward the periodic preflight.
	ticksSincePreflight int
	exitCode            int
	// sleep is the wait seam; tests substitute a recorder.
	sleep func(ctx context.Context, d time.Duration) bool
}

func newLoop(config *Config, status *statusWriter) *loop {
	return &loop{
		config: config,
		status: status,
		state:  statePreflight,
		record: loopStatus{
			StartedAt:      time.Now().UTC(),
			LastTickStatus: "pending",
			State:          statePreflight.String(),
			BinarySHA256:   executableSHA256(),
			PID:            os.Getpid(),
		},
		sleep: sleepCtx,
	}
}

// run drives the machine until Exit and returns the exit code.
func (l *loop) run(ctx context.Context) int {
	l.persist()
	for l.state != stateExit {
		if ctx.Err() != nil {
			l.exit(exitSignal, "shutdown signal received")
			break
		}
		l.step(ctx)
	}
	return l.exitCode
}

// step runs one state handler and applies its transition.
func (l *loop) step(ctx context.Context) {
	var next loopState
	switch l.state {
	case statePreflight:
		next = l.doPreflight(ctx)
	case stateDetect:
		next = l.doDetect(ctx)
	case stateHeal:
		next = l.doHeal(ctx)
	case stateVerify:
		next = l.doVerify(ctx)
	case stateHealthy:
		next = l.doHealthy(ctx)
	case stateDegraded:
		next = l.doDegraded(ctx)
	default:
		return
	}
	l.transition(next)
}

func (l *loop) transition(next loopState) {
	if next == l.state {
		return
	}
	log.Printf("state %s -> %s", l.state, next)
	l.state = next
	l.record.State = next.String()
	l.persist()
}

// exit records the code in the status file before the process ends, so the
// scheduler's escalation target can read why.
func (l *loop) exit(code int, reason string) loopState {
	log.Printf("exiting with code %d: %s", code, reason)
	l.exitCode = code
	l.record.ExitCode = &code
	if code != exitSignal {
		l.record.DegradedReason = reason
	}
	l.transition(stateExit)
	return stateExit
}

func (l *loop) persist() {
	if err := l.status.write(l.record); err != nil {
		log.Printf("status file write failed: %v", err)
	}
}

// nonHealable counts a failure that an identical retry cannot fix and
// reports whether the loop has now given up. The counter resets only when
// the API is verified healthy, so a passing preflight between two usage
// errors does not launder them.
func (l *loop) nonHealable(class, reason string) bool {
	l.record.ConsecutiveFailures++
	l.record.LastFailureClass = class
	l.record.DegradedReason = reason
	log.Printf("non-healable failure %d/%d (%s): %s", l.record.ConsecutiveFailures, nonHealableExitThreshold, class, reason)
	return l.record.ConsecutiveFailures >= nonHealableExitThreshold
}

func (l *loop) runPreflight(ctx context.Context) bool {
	result := Preflight(ctx, l.config)
	l.record.Preflight = &result
	l.ticksSincePreflight = 0
	for _, check := range result.Checks {
		log.Printf("preflight %-14s %-7s %s", check.Name, check.Status, check.Reason)
	}
	return result.OK
}

func (l *loop) doPreflight(ctx context.Context) loopState {
	if l.runPreflight(ctx) {
		l.record.DegradedReason = ""
		return stateDetect
	}
	if l.nonHealable(l.record.Preflight.FailureClass(), strings.Join(l.record.Preflight.Failed(), "; ")) {
		return l.exit(exitNonHealable, l.record.DegradedReason)
	}
	return stateDegraded
}

func (l *loop) doDetect(ctx context.Context) loopState {
	if l.config.FixedBaseURL != "" {
		l.config.setBaseURL(l.config.FixedBaseURL)
		return stateVerify
	}
	found := detectAPIPort(ctx, l.config)
	if found.Verified != "" && l.config.adoptPort(ctx, found.Verified) {
		return stateVerify
	}
	if found.Pending != "" {
		l.healReason = fmt.Sprintf("process registry names port %s but autoheal does not answer there", found.Pending)
	} else {
		l.healReason = "autoheal API not detected"
	}
	return stateHeal
}

func (l *loop) doHeal(ctx context.Context) loopState {
	if !l.config.ManageAPILifecycle {
		l.record.DegradedReason = "lifecycle management disabled (--no-manage-api); waiting for the API to appear"
		return stateDegraded
	}
	if l.backoff > 0 {
		log.Printf("heal backoff: waiting %v before the next attempt", l.backoff)
		if !l.sleep(ctx, l.backoff) {
			return l.exit(exitSignal, "shutdown signal received during heal backoff")
		}
	}
	err := ensureAPIRunning(ctx, l.config, l.healReason)
	if err == nil {
		l.backoff = 0
		l.record.DegradedReason = ""
		return stateVerify
	}
	if ctx.Err() != nil {
		return l.exit(exitSignal, "shutdown signal received during heal")
	}
	var failure *healError
	if errors.As(err, &failure) && !failure.Healable() {
		if l.nonHealable(failure.Class.String(), err.Error()) {
			return l.exit(exitNonHealable, err.Error())
		}
		// An identical retry needs the host to change, not time to pass;
		// the tick interval is a chance for that, not a backoff.
		l.persist()
		if !l.sleep(ctx, l.config.TickInterval) {
			return l.exit(exitSignal, "shutdown signal received between heal attempts")
		}
		return stateHeal
	}
	class := "lifecycle"
	if failure != nil {
		class = failure.Class.String()
	}
	l.record.LastFailureClass = class
	l.record.DegradedReason = err.Error()
	l.backoff = nextBackoff(l.backoff)
	log.Printf("heal failed (%s), next attempt in %v: %v", class, l.backoff, err)
	l.persist()
	return stateHeal
}

func (l *loop) doVerify(ctx context.Context) loopState {
	if l.config.TickEndpoint == "" {
		return stateDetect
	}
	return l.tick(ctx, stateDegraded)
}

func (l *loop) doHealthy(ctx context.Context) loopState {
	if !l.sleep(ctx, l.config.TickInterval) {
		return l.exit(exitSignal, "shutdown signal received")
	}
	if l.ticksSincePreflight >= preflightEveryTicks {
		if !l.runPreflight(ctx) {
			if l.nonHealable(l.record.Preflight.FailureClass(), strings.Join(l.record.Preflight.Failed(), "; ")) {
				return l.exit(exitNonHealable, l.record.DegradedReason)
			}
			return stateDegraded
		}
	}
	return l.tick(ctx, stateHealthy)
}

func (l *loop) doDegraded(ctx context.Context) loopState {
	if !l.sleep(ctx, l.config.TickInterval) {
		return l.exit(exitSignal, "shutdown signal received")
	}
	if !l.runPreflight(ctx) {
		if l.nonHealable(l.record.Preflight.FailureClass(), strings.Join(l.record.Preflight.Failed(), "; ")) {
			return l.exit(exitNonHealable, l.record.DegradedReason)
		}
		l.persist()
		return stateDegraded
	}
	if l.config.TickEndpoint == "" {
		return stateDetect
	}
	return l.tick(ctx, stateDegraded)
}

// tick runs one health tick. Success goes to Healthy and resets every
// counter. A failure stays in `stay` until MaxFailures, then asks whether
// autoheal is still answering: a live process whose ticks fail is busy, not
// dead, and a restart would only interrupt it (Degraded); a silent one is
// what healing is for (Heal).
func (l *loop) tick(ctx context.Context, stay loopState) loopState {
	l.ticksSincePreflight++
	result, err := runTick(ctx, l.config)
	now := time.Now().UTC()
	l.record.LastTickAt = &now
	defer l.persist()
	if err != nil {
		if ctx.Err() != nil {
			return l.exit(exitSignal, "shutdown signal received during tick")
		}
		l.tickFailures++
		l.record.LastTickStatus = "failed"
		l.record.LastFailureClass = "tick"
		log.Printf("tick failed (%d/%d): %v", l.tickFailures, l.config.MaxFailures, err)
		if l.tickFailures < l.config.MaxFailures {
			return stay
		}
		l.tickFailures = 0
		if autohealIsAlive(ctx, l.config.APIPort) {
			l.record.DegradedReason = fmt.Sprintf("%d consecutive tick failures but autoheal still answers on port %s; not restarting a live process", l.config.MaxFailures, l.config.APIPort)
			log.Print(l.record.DegradedReason)
			return stateDegraded
		}
		l.healReason = fmt.Sprintf("autoheal stopped answering on port %s after %d tick failures", l.config.APIPort, l.config.MaxFailures)
		l.config.dropPort()
		return stateHeal
	}
	l.tickFailures = 0
	l.backoff = 0
	l.record.LastTickStatus = "ok"
	l.record.ConsecutiveFailures = 0
	l.record.DegradedReason = ""
	log.Printf("tick: %s (ok %d, warn %d, crit %d)", result.Status, result.Summary.OK, result.Summary.Warning, result.Summary.Critical)
	return stateHealthy
}
