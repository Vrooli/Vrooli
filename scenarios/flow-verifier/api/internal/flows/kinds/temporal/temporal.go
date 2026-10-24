// Package temporal registers the "temporal" flow kind with the flows
// chassis. It implements kind.Kind by delegating to the existing
// temporal-flow packages (contract, compile, model, layout, scaffold,
// codegen).
package temporal

import (
	"context"
	"errors"

	"flow-verifier/internal/flows/kind"
	"flow-verifier/internal/flows/kinds/temporal/compile"
	"flow-verifier/internal/flows/kinds/temporal/contract"
	tlayout "flow-verifier/internal/flows/kinds/temporal/layout"
	"flow-verifier/internal/flows/kinds/temporal/model"
	"flow-verifier/internal/flows/kinds/temporal/scaffold"
	"flow-verifier/internal/flows/schemas"
)

// Name is the wire identifier for the temporal kind.
const Name = "temporal"

func init() {
	kind.Register(&Kind{})
}

// Kind implements kind.Kind for temporal state-machine flows.
type Kind struct{}

func (*Kind) Name() string { return Name }

func (*Kind) SchemaJSON() []byte { return schemas.Temporal }

// FilenameGlobs returns the on-disk filename pattern temporal contracts
// use. Discovery glob matches any *.json inside a flow/ directory and
// the kind field disambiguates; temporal sticks with the conventional
// "flow.json" name so an agent grepping a feature directory recognizes
// it instantly.
func (*Kind) FilenameGlobs() []string { return []string{"flow.json"} }

// Load parses and validates raw contract bytes, compiles them into a
// model.Flow, and wraps the result in a Spec. The contractPath is
// repo-relative and used in error messages plus stamped into the
// returned Spec.
func (*Kind) Load(raw []byte, contractPath string) (kind.Spec, error) {
	c, err := contract.LoadBytes(raw, contractPath)
	if err != nil {
		return nil, err
	}
	flow, err := compile.Compile(c)
	if err != nil {
		return nil, err
	}
	return &Spec{flow: flow}, nil
}

// Verify is not yet routed through the Kind interface. The temporal
// verification pipeline (pipeline + verification/{quint,artifact,lint})
// is invoked directly by handlers and CLI today; later phases move
// that orchestration here.
func (*Kind) Verify(_ context.Context, _ kind.Spec) (kind.VerifyResult, error) {
	return kind.VerifyResult{}, errors.New("temporal: Verify not yet routed through Kind interface")
}

// Scaffold writes a fresh temporal flow directory and returns its
// repo-relative path.
func (*Kind) Scaffold(opts kind.ScaffoldOptions) (string, error) {
	return scaffold.Write(scaffold.Options{
		Root:      opts.Root,
		ParentDir: opts.ParentDir,
		FlowID:    opts.FlowID,
		Language:  tlayout.Language(opts.Language),
	})
}

// Codegen is driven by the verification pipeline today (which needs the
// built formal-artifact) rather than the Kind interface; routing it
// here is a later-phase concern.
func (*Kind) Codegen(_ kind.Spec, _ kind.Language) (kind.Artifacts, error) {
	return kind.Artifacts{}, errors.New("temporal: Codegen not yet routed through Kind interface")
}

// StudioDescriptor returns a minimal descriptor. Flow Studio currently
// renders temporal flows via dedicated UI; this stub will be filled in
// when Studio consumes the registry.
func (*Kind) StudioDescriptor(_ kind.Spec) kind.StudioDescriptor {
	return kind.StudioDescriptor{Renderer: "temporal-graph"}
}

// FlowsFromSpecs projects a list of kind.Spec values down to the
// temporal-flow shape, dropping any specs of other kinds. Callers that
// know they want temporal flows (current handlers, CLI, pipeline) use
// this to bridge between the generic discovery surface and the rich
// temporal-flow shape.
func FlowsFromSpecs(specs []kind.Spec) []model.Flow {
	flows := make([]model.Flow, 0, len(specs))
	for _, s := range specs {
		if t, ok := s.(*Spec); ok {
			flows = append(flows, t.flow)
		}
	}
	return flows
}

// Spec wraps a compiled model.Flow and implements kind.Spec.
type Spec struct {
	flow model.Flow
}

// Flow returns the underlying compiled temporal flow. Callers that need
// the rich shape type-assert to *Spec and call this.
func (s *Spec) Flow() model.Flow { return s.flow }

func (s *Spec) FlowID() string       { return s.flow.FlowID }
func (s *Spec) Domain() string       { return s.flow.Domain }
func (s *Spec) Description() string  { return s.flow.Description }
func (s *Spec) ContractPath() string { return s.flow.ContractPath }
func (s *Spec) SchemaVersion() int   { return s.flow.SchemaVersion }
func (*Spec) Kind() string           { return Name }
