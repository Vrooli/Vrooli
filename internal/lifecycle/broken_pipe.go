package lifecycle

import (
	"io"
	"sync"
)

// A lifecycle mutation must finish once begun, even when whoever reads the
// CLI's output goes away mid-operation: a `| head` that has its lines, a
// closed terminal, an agent harness that stops reading. Go ends a process that
// writes to a broken pipe on stdout or stderr unless it subscribes to SIGPIPE,
// which turns those writes into EPIPE errors instead. Every scenario lock holds
// that subscription for as long as it is held, so a restart that has begun
// stopping a scenario also starts it again. Once no lock is held the default
// returns, and `vrooli ... | head` still ends promptly.
var brokenPipeTolerance struct {
	mu    sync.Mutex
	holds int
}

// holdBrokenPipeTolerance keeps writes to a broken stdout or stderr survivable
// until the returned release runs. Holds nest (a dependency locks inside its
// consumer's lock); only the last release restores the default.
func holdBrokenPipeTolerance() (release func()) {
	brokenPipeTolerance.mu.Lock()
	brokenPipeTolerance.holds++
	if brokenPipeTolerance.holds == 1 {
		subscribeBrokenPipe()
	}
	brokenPipeTolerance.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			brokenPipeTolerance.mu.Lock()
			brokenPipeTolerance.holds--
			if brokenPipeTolerance.holds == 0 {
				unsubscribeBrokenPipe()
			}
			brokenPipeTolerance.mu.Unlock()
		})
	}
}

// runOutput is one run's shared output state: the lifecycle log, and whether
// the console has gone. Both of a run's writers reach the same console, so the
// first failed console write drops it for both.
type runOutput struct {
	mu          sync.Mutex
	log         io.Writer
	consoleGone bool
}

// logFirstTee writes the lifecycle log, then the console. The log is the
// record of the run and always receives everything. A console that fails a
// write (its reader has gone) is dropped and never fails the write, so a
// step's output keeps draining instead of stalling the tool that produced it.
type logFirstTee struct {
	out     *runOutput
	console io.Writer
}

// newLogFirstTees returns a run's orchestrator and child writers over one
// lifecycle log.
func newLogFirstTees(log, orchestratorConsole, childConsole io.Writer) (orchestrator, child io.Writer) {
	out := &runOutput{log: log}
	return logFirstTee{out: out, console: orchestratorConsole}, logFirstTee{out: out, console: childConsole}
}

func (t logFirstTee) Write(p []byte) (int, error) {
	t.out.mu.Lock()
	defer t.out.mu.Unlock()
	if n, err := t.out.log.Write(p); err != nil {
		return n, err
	}
	if !t.out.consoleGone {
		if _, err := t.console.Write(p); err != nil {
			t.out.consoleGone = true
		}
	}
	return len(p), nil
}
