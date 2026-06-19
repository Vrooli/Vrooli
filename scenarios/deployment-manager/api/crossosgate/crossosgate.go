// Package crossosgate wires deployment-manager to vrooli-bridge's cross-OS
// deployment gate (bridge OT-P1-002). Bridge SUPPLIES the capability — validate
// a scenario natively on one node per target OS and aggregate per-OS verdicts;
// deployment-manager OWNS the resulting production-readiness verdict that gates
// promotion.
//
// This is the consumer side of bridge's gate seam: a narrow typed client over
// bridge's GateService Connect/JSON contract, so deployment-manager gains
// cross-OS readiness WITHOUT importing bridge's internals or proto module — it
// speaks the wire contract directly (Connect unary, application/json). The
// Bridge interface is the substitution seam: production wires the HTTP client;
// tests wire a fake.
package crossosgate

import "context"

// Request asks bridge to validate a scenario across the target OSes at a
// revision. deployment-manager fills it from the deployment profile under
// promotion.
type Request struct {
	Scenario       string   `json:"scenario"`
	Revision       string   `json:"revision"`
	TargetOSes     []string `json:"target_oses"`
	TimeoutSeconds int64    `json:"timeout_seconds,omitempty"`
}

// OSResult is one target OS's outcome, projected from bridge's per-OS ledger.
type OSResult struct {
	OS          string `json:"os"`
	NodeID      string `json:"node_id,omitempty"`
	RunID       string `json:"run_id,omitempty"`
	Disposition string `json:"disposition"`
	ExitCode    int32  `json:"exit_code,omitempty"`
	Detail      string `json:"detail,omitempty"`
}

// Verdict is deployment-manager's OWNED production-readiness result, derived from
// bridge's aggregate gate verdict. ProductionReady is the single boolean that
// gates promotion; the per-OS ledger + offending run ids are surfaced for the
// operator to drill into.
type Verdict struct {
	// ProductionReady is true only when every target OS validated green.
	ProductionReady bool `json:"production_ready"`
	// Verdict is bridge's aggregate label (pending / passed / failed).
	Verdict string `json:"verdict"`
	// GateID is the durable bridge gate this verdict came from (re-attachable).
	GateID string `json:"gate_id"`
	// TimedOut is true when the gate had not settled within the wait window.
	TimedOut bool       `json:"timed_out"`
	Results  []OSResult `json:"results"`
}

// Bridge is the substitution seam over bridge's GateService: start a gate, then
// block once for its terminal verdict. The HTTP client (http.go) satisfies it
// against a live bridge; unit tests satisfy it with a fake.
type Bridge interface {
	// RunGate starts a durable cross-OS gate and returns its id + the per-OS
	// dispatch ledger.
	RunGate(ctx context.Context, in Request) (gateID string, results []OSResult, err error)
	// WaitGate blocks once until the gate is terminal (or its timeout elapses)
	// and returns the aggregate verdict + per-OS ledger.
	WaitGate(ctx context.Context, gateID string, timeoutSeconds int64) (verdict string, timedOut bool, results []OSResult, err error)
}

// Gate orchestrates a cross-OS readiness evaluation over the Bridge seam and
// owns the mapping from bridge's aggregate verdict to deployment-manager's
// production-readiness decision.
type Gate struct {
	bridge Bridge
}

// New constructs a Gate over the given Bridge.
func New(bridge Bridge) *Gate { return &Gate{bridge: bridge} }

// passedVerdict is bridge's label for "every target OS validated green".
const passedVerdict = "passed"

// Evaluate runs the cross-OS gate and blocks for its terminal verdict, returning
// deployment-manager's owned production-readiness Verdict. A gate that has not
// settled within its wait window returns ProductionReady=false with TimedOut set
// — promotion is withheld until a real green is observed (re-evaluate by gate
// id), never on an unproven assumption.
func (g *Gate) Evaluate(ctx context.Context, in Request) (Verdict, error) {
	gateID, _, err := g.bridge.RunGate(ctx, in)
	if err != nil {
		return Verdict{}, err
	}
	verdict, timedOut, results, err := g.bridge.WaitGate(ctx, gateID, in.TimeoutSeconds)
	if err != nil {
		return Verdict{GateID: gateID}, err
	}
	return Verdict{
		ProductionReady: !timedOut && verdict == passedVerdict,
		Verdict:         verdict,
		GateID:          gateID,
		TimedOut:        timedOut,
		Results:         results,
	}, nil
}
