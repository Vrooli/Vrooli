package backends

import (
	"context"
	"fmt"
)

// BuiltinExec runs a deterministic provider operation in process.
type BuiltinExec func(context.Context, Request) error

// BuiltinProvider adapts an in-process operation to the shared backend contract.
type BuiltinProvider struct {
	name  string
	scope string
	ops   []string
	exec  BuiltinExec
}

// NewBuiltinProvider creates an always-available CPU provider with no setup cost.
func NewBuiltinProvider(name, scope string, ops []string, exec BuiltinExec) *BuiltinProvider {
	return &BuiltinProvider{name: name, scope: scope, ops: append([]string(nil), ops...), exec: exec}
}

func (p *BuiltinProvider) Name() string                   { return p.name }
func (p *BuiltinProvider) Operations() []string           { return append([]string(nil), p.ops...) }
func (p *BuiltinProvider) Standalone() bool               { return true }
func (p *BuiltinProvider) IsCloud() bool                  { return false }
func (p *BuiltinProvider) GPUCapable() bool               { return false }
func (p *BuiltinProvider) Available(context.Context) bool { return true }

func (p *BuiltinProvider) Availability(context.Context) Availability {
	return Availability{
		Available: true,
		Detail:    fmt.Sprintf("%s provider built into image-tools API binary", p.name),
		Provision: "no host provisioning required",
	}
}

func (p *BuiltinProvider) Execute(ctx context.Context, req Request) (Result, error) {
	if req.Output.LocalPath == "" {
		return Result{}, fmt.Errorf("%s: builtin backend %q requires a local output path", p.scope, p.name)
	}
	if err := p.exec(ctx, req); err != nil {
		return Result{}, fmt.Errorf("%s: builtin backend %q execution failed: %w", p.scope, p.name, err)
	}
	return Result{OutputRef: req.Output.LocalPath, Meta: map[string]string{"backend": p.name}}, nil
}
