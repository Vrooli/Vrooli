package inference

import (
	"context"
	"fmt"
	"strings"
)

// ProviderRequest is the provider-neutral payload sent to a repository. The
// repository owns resource/provider execution; the inference service owns the
// schema gate and local validation.
type ProviderRequest struct {
	Source      string
	SchemaJSON  string
	Instruction string
	Role        string
}

type ProviderResult struct {
	ValueJSON    string
	Provider     string
	Model        string
	InputTokens  int64
	OutputTokens int64
	CostMicros   int64
}

type Repository interface {
	Run(context.Context, ProviderRequest) (ProviderResult, error)
}

// UnavailableRepository is the degraded stance used when the role catalog
// cannot be loaded. Inference reports a stated reason per call while the rest
// of the gateway — routing, measures, conformance — keeps serving.
type UnavailableRepository struct{ Reason string }

func (r UnavailableRepository) Run(context.Context, ProviderRequest) (ProviderResult, error) {
	reason := strings.TrimSpace(r.Reason)
	if reason == "" {
		reason = "the inference role catalog is not loaded"
	}
	return ProviderResult{}, fmt.Errorf("%w: %s", ErrUnavailable, reason)
}

// StaticRepository is the phase-one seam implementation. It provides a
// deterministic, schema-shaped value for contract tests until the resource
// command repository is wired in a later phase.
type StaticRepository struct{}

func (StaticRepository) Run(_ context.Context, request ProviderRequest) (ProviderResult, error) {
	return ProviderResult{
		ValueJSON:    `{}`,
		Provider:     "skeleton",
		Model:        "skeleton",
		InputTokens:  int64(len(request.Source)),
		OutputTokens: 1,
	}, nil
}
