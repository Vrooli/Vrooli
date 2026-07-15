package opsbridge

import (
	"context"
	"fmt"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/operatingmode"
	"swarm-manager/internal/opscatalog"
	"swarm-manager/internal/opsrunner"
)

// SimulationEngine walks a registered operating mode against a deterministic
// preset and returns the resulting trace. It is exactly
// operatingmode.Service.SimulateMode: the transport-agnostic simulation core
// that drives the mode's REAL transition guards over a preset (never a bypass of
// the phase graph). It is kept as an interface so the driver stays testable with
// a fake engine and free of a hard dependency on the concrete Service.
type SimulationEngine interface {
	SimulateMode(ctx context.Context, mode operatingmode.Mode, presetID string) (operatingmode.SimulationResponse, error)
}

// ContractSource resolves an operation contract so the driver can map a round's
// outcome name onto the contract-declared disposition, mirroring exactly what
// Runner.CommitResult does on the live path. Satisfied by *opscatalog.Catalog.
type ContractSource interface {
	Contract(id agentops.OperationID, version string) (opscatalog.LoadedContract, bool)
}

// SimulationDriver is the production opsrunner.ExecutionDriver for the
// synchronous simulation path (Invoke with Simulate=true). It drives ONE
// operation execution — a single round — by walking the bound mode's real
// transition guards through the operating-mode simulation seam and mapping the
// first round's resolved handoff onto the operation outcome, reusing the SAME
// round->delivery mapping (HandoffRoundDelivery) the live completion bridge
// uses. So a simulated Invoke records the identical outcome+result a live round
// would deliver to CommitResult — with no agent spawn and no stub, which is what
// lets a production Runner be constructed (opsrunner.New requires a real Driver)
// without a shim.
//
// It lives in opsbridge, not opsrunner, because it is the same one-way bridge
// concern the package already owns: it is the only place that both walks the
// operating-mode engine AND speaks the runner's outcome vocabulary, keeping the
// dependency edge one-way (opsrunner never imports the engine).
type SimulationDriver struct {
	engine    SimulationEngine
	contracts ContractSource
}

// NewSimulationDriver builds the driver over a simulation engine and the catalog
// that declares the operation contracts (for disposition resolution).
func NewSimulationDriver(engine SimulationEngine, contracts ContractSource) *SimulationDriver {
	return &SimulationDriver{engine: engine, contracts: contracts}
}

// Compile-time proof the driver is a runner execution driver.
var _ opsrunner.ExecutionDriver = (*SimulationDriver)(nil)

// Drive simulates the bound mode and returns the first round's outcome.
func (d *SimulationDriver) Drive(ctx context.Context, prep opsrunner.Prepared, run opsrunner.RunHandle) (opsrunner.ExecutionOutcome, error) {
	if d.engine == nil {
		return opsrunner.ExecutionOutcome{}, fmt.Errorf("opsbridge: no simulation engine configured")
	}
	sim, err := d.engine.SimulateMode(ctx, operatingmode.Mode(prep.Mode), run.Preset)
	if err != nil {
		return opsrunner.ExecutionOutcome{}, fmt.Errorf("opsbridge: simulate mode %q: %w", prep.Mode, err)
	}
	if len(sim.Trace) == 0 {
		return opsrunner.ExecutionOutcome{}, fmt.Errorf("opsbridge: simulation of mode %q produced no round", prep.Mode)
	}

	// A single operation execution is a single round: the live path starts ONE
	// round per Invoke and finalizes it via CommitResult (there is no
	// in-operation loop — target-round continuation is the runner/policy's job).
	// So the simulation driver reports the FIRST round's outcome, the exact twin
	// of that one live round; later steps in the walk model the rounds a
	// subsequent Invoke would drive.
	round := sim.Trace[0].Round
	delivery, err := roundDeliveryFor(string(run.Operation), round)
	if err != nil {
		return opsrunner.ExecutionOutcome{}, fmt.Errorf("opsbridge: map simulated round for %q: %w", prep.Mode, err)
	}
	if !delivery.Deliver {
		return opsrunner.ExecutionOutcome{}, fmt.Errorf("opsbridge: simulated first round for %q is not a terminal outcome (round status %q)", prep.Mode, round.Status)
	}

	disposition, err := d.disposition(run.Operation, delivery.Outcome)
	if err != nil {
		return opsrunner.ExecutionOutcome{}, err
	}
	return opsrunner.ExecutionOutcome{
		Outcome:     delivery.Outcome,
		Disposition: disposition,
		Result:      delivery.Result,
		RunID:       round.RunID,
	}, nil
}

// disposition resolves the workflow disposition for a delivered outcome from the
// operation contract — the same SSOT Runner.CommitResult reads — so a simulated
// run records the identical operation-record state a live commit would. It is
// fail-closed: an operation the catalog does not declare, or an outcome the
// contract does not declare, is an error rather than a guessed disposition.
func (d *SimulationDriver) disposition(op agentops.OperationID, outcome string) (opsrunner.Disposition, error) {
	if d.contracts == nil {
		return "", fmt.Errorf("opsbridge: no contract source to resolve disposition for %q", op)
	}
	lc, ok := d.contracts.Contract(op, "")
	if !ok {
		return "", fmt.Errorf("opsbridge: operation %q is not declared in the catalog", op)
	}
	for _, o := range lc.Contract.Outcomes {
		if o.Name == outcome {
			return opsrunner.Disposition(o.Disposition), nil
		}
	}
	return "", fmt.Errorf("opsbridge: outcome %q is not declared by operation %q", outcome, op)
}
