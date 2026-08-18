package inference

import (
	"context"
	"fmt"
	"strings"

	"ai-gateway/internal/providers"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

// ProviderRequest is the provider-neutral payload sent to a repository. The
// repository owns resource/provider execution; the inference service owns the
// schema gate and local validation.
type ProviderRequest struct {
	Source      string
	SchemaJSON  string
	Instruction string
	Role        string
	Profile     sharedv1.Profile
	Turns       []*inferencev1.Turn
	Attachments []*sharedv1.Attachment
	// Temperature is the caller's explicit sampling request, or nil for "use
	// the role's declared sampling". It is a plain pointer rather than the
	// proto message so this package does not depend on proto shape for one
	// scalar.
	Temperature *float64
	// MaxOutputTokens is the caller's explicit output cap, or 0 for "use the
	// resolved resource role's declared cap". It is request-level because the
	// budget depends on caller data a role cannot know.
	MaxOutputTokens int32
}

type ProviderResult struct {
	ValueJSON            string
	Provider             string
	Model                string
	CoordinateConvention string
	InputTokens          int64
	OutputTokens         int64
	CostMicros           int64
	// Applied records what the gateway actually sent and what the resolved role
	// declares, so a caller records provenance that is true rather than
	// provenance that echoes its own request.
	Applied       AppliedSettings
	ContextWindow int32
}

type EmbeddingResult struct {
	Vectors     [][]float64
	Provider    string
	Model       string
	Dimension   int
	InputTokens int64
	CostMicros  int64
}

// AppliedSettings is the gateway's honest account of one execution. It
// deliberately has no single "applied temperature": for a provider that accepts
// and silently discards the control, such a field would have to lie in exactly
// the case the caller most needs the truth. TemperatureSent and
// TemperatureSupport are reported separately so the caller can combine them.
type AppliedSettings struct {
	// TemperatureSent is the value passed to the provider, nil when the gateway
	// omitted the control. It is not proof the provider honoured it.
	TemperatureSent *float64
	// TemperatureSupport is the resolved role's policy declaration, or empty
	// when no candidate resolved.
	TemperatureSupport providers.SamplingSupport
	// MaxOutputTokens is the cap the gateway imposed and MaxOutputTokensSource
	// names where it came from. A zero cap with source OutputCapNoneImposed
	// means the gateway genuinely sent nothing, which is a stronger statement
	// than "unknown".
	MaxOutputTokens       int32
	MaxOutputTokensSource OutputCapSource
}

// OutputCapSource names where an imposed output cap came from. It reports only
// what the gateway knows: a provider's own internal default is not visible from
// here and is never guessed.
type OutputCapSource string

const (
	// OutputCapUnspecified means no candidate resolved, so there is nothing to
	// report.
	OutputCapUnspecified OutputCapSource = ""
	// OutputCapRequest means the caller set max_output_tokens.
	OutputCapRequest OutputCapSource = "request"
	// OutputCapRolePolicy means the resolved resource role declared the cap.
	OutputCapRolePolicy OutputCapSource = "role_policy"
	// OutputCapNoneImposed means the gateway sent no cap at all.
	OutputCapNoneImposed OutputCapSource = "none_imposed"
)

type Repository interface {
	Run(context.Context, ProviderRequest) (ProviderResult, error)
}

type EmbeddingRepository interface {
	Embed(context.Context, string, []string) (EmbeddingResult, error)
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
