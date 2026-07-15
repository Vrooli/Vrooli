package opsrunner

import (
	"context"
	"encoding/json"
	"fmt"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/operatingmode"
)

// EnginePreparer is the production ModePreparer: it compiles and pins a real
// operating-mode definition and validates caller inputs through the SAME input
// SSOT the operating-mode engine uses for simulation and live runs
// (operatingmode.CompileInputContract + ValidateCallerInputSnapshot), so the
// effective inputs the runner pins are byte-identical to what a simulated or
// live round would see. It resolves definitions through an injected registry so
// it stays decoupled from the engine's Service and testable with synthetic
// modes.
type EnginePreparer struct {
	// Definitions resolves a registered mode id to its compiled definition.
	Definitions map[string]operatingmode.Definition
	// Delegated resolves a delegated sub-mode definition when a mode composes
	// another (executed_by). Optional; single-mode definitions need no entry.
	Delegated map[string]operatingmode.Definition
}

// NewEnginePreparer builds a preparer over a set of registered definitions.
func NewEnginePreparer(defs map[string]operatingmode.Definition) *EnginePreparer {
	return &EnginePreparer{Definitions: defs}
}

// An EnginePreparer is also a resolver ModeChecker (see the same note on
// LivePreparer): its RevisionExists/ModeCompatible back the binding checker, so
// the same preparer can be passed as the BindingResolver's checker. The
// assertion locks the method set so a drift is a compile error.
var _ agentops.ModeChecker = (*EnginePreparer)(nil)

func (p *EnginePreparer) def(mode string) (operatingmode.Definition, bool) {
	d, ok := p.Definitions[mode]
	return d, ok
}

// Prepare compiles the bound mode's input contract, validates the caller inputs
// against it, and returns the canonical bytes the runner hashes into provenance.
func (p *EnginePreparer) Prepare(_ context.Context, req PrepareRequest) (Prepared, error) {
	def, ok := p.def(req.Mode)
	if !ok {
		return Prepared{}, fmt.Errorf("mode %q is not registered", req.Mode)
	}
	defs := map[operatingmode.Mode]operatingmode.Definition{def.Mode: def}
	for id, d := range p.Delegated {
		defs[operatingmode.Mode(id)] = d
	}
	compiled, err := operatingmode.CompileInputContract(defs, def)
	if err != nil {
		return Prepared{}, fmt.Errorf("compile input contract for %q: %w", req.Mode, err)
	}
	snapshot, _, _, err := operatingmode.ValidateCallerInputSnapshot(compiled, req.CallerInputs)
	if err != nil {
		return Prepared{}, fmt.Errorf("validate caller inputs for %q: %w", req.Mode, err)
	}
	if len(snapshot) == 0 {
		snapshot = json.RawMessage(`{}`)
	}
	compiledMode, err := json.Marshal(def)
	if err != nil {
		return Prepared{}, err
	}
	promptCatalog, err := json.Marshal(compiled)
	if err != nil {
		return Prepared{}, err
	}
	return Prepared{
		Mode: req.Mode, ModeRevision: req.ModeRevision,
		CompiledMode: compiledMode, PromptCatalog: promptCatalog,
		PromptCatalogRevision: req.ModeRevision, EffectiveInputs: snapshot,
	}, nil
}

// RevisionExists reports whether the mode is registered. Revision pinning beyond
// registration is enforced by the binding's mode_revision, which the runner
// records in provenance and the execution store verifies on reproduce.
func (p *EnginePreparer) RevisionExists(mode, _ string) bool {
	_, ok := p.def(mode)
	return ok
}

// ModeCompatible reports whether the mode's declared target kind matches the
// operation's target, so a binding to a mode built for another unit of work
// fails closed with ErrIncompatibleMode.
func (p *EnginePreparer) ModeCompatible(mode string, _ agentops.OperationID, target agentops.TargetKind) bool {
	def, ok := p.def(mode)
	if !ok {
		return false
	}
	return string(def.Target.Kind) == string(target)
}
