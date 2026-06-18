package ai

import (
	"context"
	"fmt"
	"os"

	"image-tools/internal/backends"
	"image-tools/internal/models"
)

// builtinExec runs an op entirely in-process (no external program, no model
// weights). It reads the request's inputs, computes the result, and writes it to
// req.Output.LocalPath by the same contract execProvider follows.
type builtinExec func(ctx context.Context, req backends.Request) error

// builtinProvider is a backends.Provider whose work is deterministic Go code
// shipped in the binary — the always-runnable Local-CPU tier. Unlike
// execProvider it has no PATH dependency: Available() is always true, so an op
// backed by a builtin provider runs on any host with zero provisioning. It is
// CPU-only and never claims the GPU.
type builtinProvider struct {
	name string
	ops  []string
	exec builtinExec
}

func (p *builtinProvider) Name() string                   { return p.name }
func (p *builtinProvider) Operations() []string           { return append([]string(nil), p.ops...) }
func (p *builtinProvider) Standalone() bool               { return true }
func (p *builtinProvider) IsCloud() bool                  { return false }
func (p *builtinProvider) GPUCapable() bool               { return false }
func (p *builtinProvider) Available(context.Context) bool { return true }

func (p *builtinProvider) Execute(ctx context.Context, req backends.Request) (backends.Result, error) {
	if req.Output.LocalPath == "" {
		return backends.Result{}, fmt.Errorf("ai: builtin backend %q requires a local output path", p.name)
	}
	if err := p.exec(ctx, req); err != nil {
		return backends.Result{}, fmt.Errorf("ai: builtin backend %q execution failed: %w", p.name, err)
	}
	return backends.Result{OutputRef: req.Output.LocalPath, Meta: map[string]string{"backend": p.name}}, nil
}

// runNaturalize is the builtin executor for the naturalize op: decode the input
// image, run the deterministic realism compositor, and write the PNG result.
func runNaturalize(_ context.Context, req backends.Request) error {
	in, err := in0(req)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(in)
	if err != nil {
		return fmt.Errorf("ai: read naturalize input: %w", err)
	}
	img, err := decodeNaturalizeInput(data)
	if err != nil {
		return err
	}
	out := Naturalize(img, parseNaturalizeParams(req.Params))
	enc, err := encodePNG(out)
	if err != nil {
		return err
	}
	if err := os.WriteFile(req.Output.LocalPath, enc, 0o644); err != nil {
		return fmt.Errorf("ai: write naturalize output: %w", err)
	}
	return nil
}

// builtinProviderSpecs declares the in-process providers, keyed by the registry
// `backend` name (models.BackendBuiltin) so the selector's match-by-backend path
// lines up with the weightless seed models that name this backend.
func builtinProviderSpecs() []*builtinProvider {
	return []*builtinProvider{
		{name: models.BackendBuiltin, ops: []string{"naturalize"}, exec: runNaturalize},
	}
}
