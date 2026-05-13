// Package navigation registers the "navigation" flow kind. A navigation
// contract declares the routes, containers, affordances, and policy
// invariants of a UI's navigation graph; flow-verifier validates,
// reconciles, and (in later phases) renders it.
package navigation

import (
	"context"
	"fmt"

	"flow-verifier/internal/flows/kind"
	"flow-verifier/internal/flows/kinds/navigation/compile"
	"flow-verifier/internal/flows/kinds/navigation/contract"
	"flow-verifier/internal/flows/kinds/navigation/scaffold"
	"flow-verifier/internal/flows/kinds/navigation/verify"
	"flow-verifier/internal/flows/schemas"
)

// Name is the wire identifier for the navigation kind.
const Name = "navigation"

func init() {
	kind.Register(&Kind{})
}

// Kind implements kind.Kind for navigation graphs.
type Kind struct{}

func (*Kind) Name() string { return Name }

func (*Kind) SchemaJSON() []byte { return schemas.Navigation }

// FilenameGlobs returns the on-disk filename pattern navigation
// contracts use. Single-file-per-scenario is the default convention;
// multi-file split is allowed by the schema and unblocked in Phase 5.
func (*Kind) FilenameGlobs() []string { return []string{"navigation.json"} }

// Load parses, schema-validates, then structurally validates a raw
// navigation contract. The contractPath is repo-relative and used in
// error messages plus stamped into the returned Spec.
func (*Kind) Load(raw []byte, contractPath string) (kind.Spec, error) {
	c, err := contract.LoadBytes(raw, contractPath)
	if err != nil {
		return nil, err
	}
	g, err := compile.Compile(c)
	if err != nil {
		return nil, err
	}
	return &Spec{graph: g}, nil
}

// Verify runs the reachability invariant checker and the deep-link
// policy verifier. Findings are aggregated into VerifyResult; Passed is
// the AND of every finding's Passed flag.
func (*Kind) Verify(_ context.Context, s kind.Spec) (kind.VerifyResult, error) {
	spec, ok := s.(*Spec)
	if !ok {
		return kind.VerifyResult{}, fmt.Errorf("navigation.Verify: spec is %T, want *navigation.Spec", s)
	}
	reach, err := verify.CheckInvariants(spec.graph)
	if err != nil {
		return kind.VerifyResult{}, err
	}
	deep, err := verify.CheckDeepLinkPolicy(spec.graph)
	if err != nil {
		return kind.VerifyResult{}, err
	}
	all := append(reach, deep...)
	passed := true
	for _, f := range all {
		if !f.Passed {
			passed = false
			break
		}
	}
	return kind.VerifyResult{Passed: passed, Findings: all}, nil
}

// Scaffold writes a fresh navigation contract directory and returns its
// repo-relative path.
func (*Kind) Scaffold(opts kind.ScaffoldOptions) (string, error) {
	return scaffold.Write(scaffold.Options{
		Root:      opts.Root,
		ParentDir: opts.ParentDir,
		FlowID:    opts.FlowID,
	})
}

// Codegen returns no artifacts until Phase 4 (routes.generated.ts).
func (*Kind) Codegen(_ kind.Spec, _ kind.Language) (kind.Artifacts, error) {
	return kind.Artifacts{}, nil
}

// StudioDescriptor returns a minimal descriptor; Flow Studio rendering
// lands in Phase 6.
func (*Kind) StudioDescriptor(_ kind.Spec) kind.StudioDescriptor {
	return kind.StudioDescriptor{Renderer: "navigation-graph"}
}

// Spec wraps a compiled navigation Graph and implements kind.Spec.
type Spec struct {
	graph compile.Graph
}

// Graph returns the underlying compiled navigation graph. Callers that
// need the rich shape type-assert to *Spec and call this.
func (s *Spec) Graph() compile.Graph { return s.graph }

func (s *Spec) FlowID() string       { return s.graph.Contract.FlowID }
func (s *Spec) Domain() string       { return s.graph.Contract.Domain }
func (s *Spec) Description() string  { return s.graph.Contract.Description }
func (s *Spec) ContractPath() string { return s.graph.Contract.ContractPath }
func (s *Spec) SchemaVersion() int   { return s.graph.Contract.SchemaVersion }
func (*Spec) Kind() string           { return Name }
