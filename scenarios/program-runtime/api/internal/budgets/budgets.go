// Package budgets is the single authority for every time budget on the
// program-execution path.
//
// Before this package existed the budgets were independent literals scattered
// across four files and two languages: the kernel gave `discover` 10s, the
// judge role declared 120s, the judge's HTTP client allowed 3 minutes, and the
// API server inherited api-core's 30s write deadline because `main.go` never
// set one. Nothing was derived from anything, so the effective ceiling on every
// synchronous call was 30 seconds — set by a default nobody chose — while
// sessions advertised a four-hour wall budget. Calls that exceeded it were
// killed mid-write and surfaced as `unexpected EOF` or `RemoteDisconnected`,
// neither of which is in the closed failure-cause vocabulary.
//
// The fix is not a bigger number. It is a *ladder*: each layer is strictly
// shorter than the layer that contains it, so a slow dependency always
// surfaces as a typed error from the innermost layer that owns it rather than
// as a severed connection from the outermost. Validate enforces that ordering
// at startup, so the ladder cannot silently rot the way the literals did.
package budgets

import (
	"fmt"
	"time"
)

// The ladder, innermost first. Every value is chosen relative to its
// neighbours, and the rationale for each is stated because these numbers are
// judgment calls that a future reader must be able to revisit.
const (
	// KernelTelemetry bounds the kernel's fire-and-forget unresolved-name
	// report. Telemetry must never change a program's deterministic outcome,
	// so this is deliberately the shortest budget in the ladder.
	KernelTelemetry = 2 * time.Second

	// KernelDescribe bounds `describe()` and `reachable()`. Both are local
	// registry reads with no outbound model or scenario call, so they are held
	// to a tight budget: a slow answer here means the runtime itself is
	// unhealthy, not that a dependency is thinking.
	KernelDescribe = 20 * time.Second

	// BridgeCall bounds one outbound governed call from this API to the owning
	// scenario. It is strictly shorter than KernelInvoke so that a slow
	// dependency returns a typed `bridge_transport` failure *through* the
	// bridge rather than tripping the kernel's own client timeout, which would
	// surface as an untyped transport error.
	BridgeCall = 90 * time.Second

	// KernelInvoke bounds the kernel's wait on the bridge for a binding call,
	// a projection verb, or intent discovery. It exceeds BridgeCall by the
	// margin the bridge needs to serialise and write its typed error.
	KernelInvoke = 100 * time.Second

	// SyncSubmit bounds a synchronous SubmitProgram. Work that legitimately
	// takes longer is not an error — it is what `--async` plus WaitForProgram
	// exists for — so exceeding this yields `deadline_exceeded` naming the
	// async path, never a severed connection.
	SyncSubmit = 2 * time.Minute

	// MaxWait is the ceiling WaitForProgram will honour for a caller-supplied
	// deadline. It is bounded by ServerWrite because the wait is served as one
	// HTTP response; a caller needing longer polls the wait again, which is a
	// bounded resumption rather than a busy loop.
	MaxWait = 5 * time.Minute

	// ServerWrite is the outermost layer: the API's HTTP write deadline. It
	// must exceed MaxWait, because the wait is written as a single response.
	//
	// Fifteen minutes matches search-hub, which sets the same ceiling for the
	// same reason: one legitimate operation on this API — the authoring eval —
	// authors and executes a program per corpus case, and measured authoring
	// latency is ~28s with a ~100s tail. This is a ceiling, not a target; no
	// ordinary response comes close to it, and every inner budget still bounds
	// the work that actually runs.
	ServerWrite = 15 * time.Minute

	// ServerRead bounds reading a request body. Program sources are small; a
	// slow read is a client defect, not long work.
	ServerRead = time.Minute

	// AuthoringEval bounds a whole authoring-eval run.
	//
	// It sits deliberately inside ServerWrite rather than on the ladder,
	// because it bounds a *handler*, not a nested call: the eval authors and
	// executes one program per corpus case, so its natural duration is
	// unbounded in the number of cases. Go's http.Server write timeout does not
	// create a context deadline, so without an explicit one the eval's own
	// per-case reserve had nothing to measure against and the run was severed
	// mid-write after doing all the work — which is how a 1-of-12 measurement
	// came back as `unexpected EOF` and was discarded.
	//
	// A run that cannot finish inside this returns `partial` with the count it
	// did not attempt, which is honest and comparable, rather than a whole-
	// corpus number it did not earn.
	AuthoringEval = ServerWrite - time.Minute
)

// Ladder returns the ordered budgets with their names, innermost first. It is
// the input to Validate and the payload handed to the kernel, so the Go
// constants above remain the only definition of these numbers.
func Ladder() []Rung {
	return []Rung{
		{Name: "kernel_telemetry", Budget: KernelTelemetry},
		{Name: "kernel_describe", Budget: KernelDescribe},
		{Name: "bridge_call", Budget: BridgeCall},
		{Name: "kernel_invoke", Budget: KernelInvoke},
		{Name: "sync_submit", Budget: SyncSubmit},
		{Name: "max_wait", Budget: MaxWait},
		{Name: "server_write", Budget: ServerWrite},
	}
}

// Rung is one layer of the ladder.
type Rung struct {
	Name   string
	Budget time.Duration
}

// Validate asserts the nesting invariant: every rung is strictly shorter than
// the one that contains it. It is called at startup and fails the process,
// because a violated ladder means some layer will be killed mid-write and the
// resulting failure will be untyped — exactly the class of defect this package
// exists to prevent.
func Validate() error {
	ladder := Ladder()
	for index := 1; index < len(ladder); index++ {
		inner, outer := ladder[index-1], ladder[index]
		if inner.Budget >= outer.Budget {
			return fmt.Errorf(
				"timeout ladder is not strictly nested: %s (%s) must be shorter than %s (%s); "+
					"a violated ladder severs connections instead of returning typed failures",
				inner.Name, inner.Budget, outer.Name, outer.Budget)
		}
	}
	return nil
}

// KernelEnvelope is the budget subset the Python kernel needs. It is
// marshalled into one environment variable at spawn so the kernel never
// carries its own copy of these numbers; Go stays the single authority and the
// two languages cannot drift.
type KernelEnvelope struct {
	Telemetry float64 `json:"telemetry_seconds"`
	Describe  float64 `json:"describe_seconds"`
	Invoke    float64 `json:"invoke_seconds"`
}

// Kernel returns the envelope handed to a kernel process at spawn.
func Kernel() KernelEnvelope {
	return KernelEnvelope{
		Telemetry: KernelTelemetry.Seconds(),
		Describe:  KernelDescribe.Seconds(),
		Invoke:    KernelInvoke.Seconds(),
	}
}

// BoundWait clamps a caller-supplied wait deadline into the ladder. A caller
// asking for longer than MaxWait is not refused — the wait returns the
// program's current, non-terminal state at the ceiling, which the caller can
// resume — because refusing would push callers back to polling.
func BoundWait(requested time.Duration) time.Duration {
	if requested <= 0 {
		return MaxWait
	}
	if requested > MaxWait {
		return MaxWait
	}
	return requested
}
