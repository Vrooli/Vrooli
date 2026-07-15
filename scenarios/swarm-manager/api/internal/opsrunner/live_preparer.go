package opsrunner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/operatingmode"
	"swarm-manager/internal/opscatalog"
)

// ErrInvalidCallerInput is returned when caller-supplied operation inputs violate
// the operation contract: an unknown input, a missing required input, or a value
// whose sensitivity/retention forbids replay. It is fail-closed — no execution is
// pinned and no run is started — so a malformed invocation never dispatches.
var ErrInvalidCallerInput = errors.New("caller inputs violate the operation contract")

// LivePreparer is the production ModePreparer for the LIVE operating-mode path. It
// differs from EnginePreparer in one load-bearing way: operation caller-inputs
// (OPERATOR_NOTE, USER_QUESTION, ...) are validated against the OPERATION
// contract, not the mode input contract, because the authored backlog modes
// declare no caller-source inputs — every mode input is a generic-provider or
// target-adapter value the engine fills itself. Forwarding an operation input
// through the mode's ValidateCallerInputSnapshot would trip its unknown-key guard.
//
// So the live preparer:
//   - validates and normalizes caller inputs against the operation contract,
//     pinning that normalized snapshot as EffectiveInputs (the reproducible caller
//     value the provenance caller_input_digest summarizes);
//   - compiles the bound mode with NO mode caller-inputs for the compiled-mode and
//     prompt-catalog digests — byte-identical to what the engine compiles for the
//     run — and confirms the mode genuinely accepts an empty caller-input set; and
//   - leaves delivery of the operation inputs to the engine to the EngineRunStarter,
//     which routes the operator note onto StartTargetPhase.Note.
type LivePreparer struct {
	catalog     *opscatalog.Catalog
	definitions map[string]operatingmode.Definition
	delegated   map[string]operatingmode.Definition
}

// NewLivePreparer builds a live preparer over a loaded catalog and the registered
// mode definitions (keyed by mode id). Optional delegated definitions resolve a
// mode that composes another via executed_by.
func NewLivePreparer(catalog *opscatalog.Catalog, defs map[string]operatingmode.Definition) *LivePreparer {
	return &LivePreparer{catalog: catalog, definitions: defs}
}

// A LivePreparer is also the resolver's ModeChecker: its RevisionExists and
// ModeCompatible answer the runtime binding questions (mode revision still
// registered, mode implements the operation for the target). Production wiring
// passes the same *LivePreparer as both the Runner's ModePreparer and the
// BindingResolver's ModeChecker, so a binding to a deleted revision or an
// incompatible mode fails closed against the SAME registry the runner prepares
// from. No separate adapter is needed — the method set already matches — and
// this assertion locks that so a signature drift is a compile error, not a
// silent nil-checker fallback.
var _ agentops.ModeChecker = (*LivePreparer)(nil)

// WithDelegated registers delegated sub-mode definitions for composed modes.
func (p *LivePreparer) WithDelegated(delegated map[string]operatingmode.Definition) *LivePreparer {
	p.delegated = delegated
	return p
}

func (p *LivePreparer) def(mode string) (operatingmode.Definition, bool) {
	d, ok := p.definitions[mode]
	return d, ok
}

// Prepare validates the caller inputs against the operation contract and compiles
// the bound mode's contract for the provenance digests.
func (p *LivePreparer) Prepare(_ context.Context, req PrepareRequest) (Prepared, error) {
	lc, ok := p.catalog.Contract(req.Operation, "")
	if !ok {
		return Prepared{}, fmt.Errorf("live preparer: operation %q is not declared in the catalog", req.Operation)
	}
	effective, err := validateOperationInputs(lc.Contract, req.CallerInputs)
	if err != nil {
		return Prepared{}, err
	}

	def, ok := p.def(req.Mode)
	if !ok {
		return Prepared{}, fmt.Errorf("live preparer: mode %q is not registered", req.Mode)
	}
	defs := map[operatingmode.Mode]operatingmode.Definition{def.Mode: def}
	for id, d := range p.delegated {
		defs[operatingmode.Mode(id)] = d
	}
	compiled, err := operatingmode.CompileInputContract(defs, def)
	if err != nil {
		return Prepared{}, fmt.Errorf("live preparer: compile input contract for %q: %w", req.Mode, err)
	}
	// The backlog modes carry no caller-source inputs, so an empty caller set must
	// validate. If a mode ever declares a required caller input, this fails closed
	// here rather than dispatching a run the mode cannot serve.
	if _, _, _, err := operatingmode.ValidateCallerInputSnapshot(compiled, nil); err != nil {
		return Prepared{}, fmt.Errorf("live preparer: mode %q rejects an empty caller-input set (declare its inputs on the operation contract, not the mode): %w", req.Mode, err)
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
		PromptCatalogRevision: req.ModeRevision, EffectiveInputs: effective,
	}, nil
}

// RevisionExists reports whether the mode is registered. Revision pinning beyond
// registration is enforced by the binding's mode_revision recorded in provenance.
func (p *LivePreparer) RevisionExists(mode, _ string) bool {
	_, ok := p.def(mode)
	return ok
}

// ModeCompatible reports whether the mode's declared target kind matches the
// operation's target, so a binding to a mode built for another unit of work fails
// closed with ErrIncompatibleMode.
func (p *LivePreparer) ModeCompatible(mode string, _ agentops.OperationID, target agentops.TargetKind) bool {
	def, ok := p.def(mode)
	if !ok {
		return false
	}
	return string(def.Target.Kind) == string(target)
}

// validateOperationInputs validates caller-supplied values against the operation
// contract's declared inputs and returns the normalized, canonical JSON snapshot
// the runner pins as caller_input_digest. It is fail-closed and mirrors the
// operating-mode caller-input rules (unknown key, missing required, sensitive
// retention, non-value retention) so the operation and mode paths agree on what a
// replayable caller value is.
func validateOperationInputs(contract agentops.OperationContract, supplied map[string]any) (json.RawMessage, error) {
	specs := make(map[string]agentops.CallerInput, len(contract.Inputs))
	for _, in := range contract.Inputs {
		specs[in.Name] = in
	}
	var unknown []string
	for name := range supplied {
		if _, ok := specs[name]; !ok {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, fmt.Errorf("%w: unknown operation inputs: %s", ErrInvalidCallerInput, strings.Join(unknown, ", "))
	}
	names := make([]string, 0, len(specs))
	for name := range specs {
		names = append(names, name)
	}
	sort.Strings(names)
	normalized := make(map[string]any, len(supplied))
	for _, name := range names {
		spec := specs[name]
		val, present := supplied[name]
		if !present {
			if spec.Required {
				return nil, fmt.Errorf("%w: required operation input %q is missing", ErrInvalidCallerInput, name)
			}
			continue
		}
		if strings.EqualFold(spec.Sensitivity, "sensitive") {
			return nil, fmt.Errorf("%w: operation input %q is sensitive and cannot be retained for replay", ErrInvalidCallerInput, name)
		}
		if spec.Retention != "" && !strings.EqualFold(spec.Retention, "value") {
			return nil, fmt.Errorf("%w: operation input %q uses %q retention; replayable inputs require value retention", ErrInvalidCallerInput, name, spec.Retention)
		}
		normalized[name] = val
	}
	snapshot, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("marshal operation input snapshot: %w", err)
	}
	return snapshot, nil
}
