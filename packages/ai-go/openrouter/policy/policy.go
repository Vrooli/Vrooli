// Package policy resolves OpenRouter role/model metadata through
// resource-openrouter. It is the shared consumer seam: scenarios call
// ResolveRole to turn a logical role (e.g. "image.generate.logo") into a
// concrete model slug plus endpoint family and request defaults, without ever
// reading the policy file directly. resource-openrouter remains the single
// source of truth.
package policy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

const DefaultBin = "resource-openrouter"

type Provenance struct {
	SourceKind  string `json:"source_kind"`
	Confidence  string `json:"confidence"`
	Source      string `json:"source"`
	ObservedAt  string `json:"observed_at"`
	SampleCount int    `json:"sample_count"`
}

type Modalities struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

// RequestDefaults is the bounded, role-owned request lever. Pointer/empty fields
// are absent — the consumer should only set request parameters the role
// actually declared and otherwise let the model defaults apply.
type RequestDefaults struct {
	Temperature    *float64 `json:"temperature,omitempty"`
	MaxTokens      *int     `json:"max_tokens,omitempty"`
	ResponseFormat string   `json:"response_format,omitempty"`
	Resolution     string   `json:"resolution,omitempty"`
	AspectRatio    string   `json:"aspect_ratio,omitempty"`
	Size           string   `json:"size,omitempty"`
	OutputFormat   string   `json:"output_format,omitempty"`
	Background     string   `json:"background,omitempty"`
	Quality        string   `json:"quality,omitempty"`
	Stream         *bool    `json:"stream,omitempty"`
}

type Pricing struct {
	Unit               string  `json:"unit"`
	InputPerMTok       float64 `json:"input_per_mtok,omitempty"`
	OutputPerMTok      float64 `json:"output_per_mtok,omitempty"`
	PerImage           float64 `json:"per_image,omitempty"`
	ImageTokensPerMTok float64 `json:"image_tokens_per_mtok,omitempty"`
	InputMP            float64 `json:"input_mp,omitempty"`
	FirstOutputMP      float64 `json:"first_output_mp,omitempty"`
	AdditionalMP       float64 `json:"additional_mp,omitempty"`
}

// ResolvedRole is the decoded `resource-openrouter policy resolve --json` shape.
type ResolvedRole struct {
	SchemaVersion         string                `json:"schema_version"`
	PolicyPath            string                `json:"policy_path"`
	Role                  string                `json:"role,omitempty"`
	Source                string                `json:"source"`
	Model                 string                `json:"model"`
	Endpoint              string                `json:"endpoint"`
	Fallbacks             []string              `json:"fallbacks,omitempty"`
	RequiredCapabilities  []string              `json:"required_capabilities,omitempty"`
	PreferredCapabilities []string              `json:"preferred_capabilities,omitempty"`
	Capabilities          []string              `json:"capabilities"`
	Modalities            Modalities            `json:"modalities"`
	Endpoints             []string              `json:"endpoints,omitempty"`
	ContextWindowTokens   int                   `json:"context_window_tokens,omitempty"`
	DefaultEligible       bool                  `json:"default_eligible"`
	Pricing               *Pricing              `json:"pricing,omitempty"`
	RequestDefaults       *RequestDefaults      `json:"request_defaults,omitempty"`
	Provider              string                `json:"provider,omitempty"`
	Family                string                `json:"family,omitempty"`
	RoleProvenance        *Provenance           `json:"role_provenance,omitempty"`
	Provenance            map[string]Provenance `json:"provenance,omitempty"`
}

// ModelCandidates returns the resolved default model followed by its declared
// fallbacks — the ordered list a consumer can pass to OpenRouter's `models`
// routing array, or iterate for its own failover.
func (r ResolvedRole) ModelCandidates() []string {
	out := make([]string, 0, 1+len(r.Fallbacks))
	if strings.TrimSpace(r.Model) != "" {
		out = append(out, r.Model)
	}
	out = append(out, r.Fallbacks...)
	return out
}

// CommandRunner is the process-exec seam for resource-openrouter.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type Resolver struct {
	Bin string
	Run CommandRunner
}

func (r Resolver) ResolveRole(ctx context.Context, role string) (ResolvedRole, error) {
	role = strings.TrimSpace(role)
	if role == "" {
		return ResolvedRole{}, errors.New("openrouter policy role is required")
	}
	return r.resolve(ctx, "--role", role)
}

func (r Resolver) ResolveModel(ctx context.Context, model string) (ResolvedRole, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return ResolvedRole{}, errors.New("openrouter policy model is required")
	}
	return r.resolve(ctx, "--model", model)
}

func (r Resolver) resolve(ctx context.Context, selectorFlag, selectorValue string) (ResolvedRole, error) {
	bin := strings.TrimSpace(r.Bin)
	if bin == "" {
		bin = DefaultBin
	}
	runner := r.Run
	if runner == nil {
		runner = execRunner{}
	}
	args := []string{"policy", "resolve", selectorFlag, selectorValue, "--json"}
	out, err := runner.Run(ctx, bin, args...)
	if err != nil {
		return ResolvedRole{}, fmt.Errorf("%s %s: %w: %s", bin, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	var resolved ResolvedRole
	if err := json.Unmarshal(out, &resolved); err != nil {
		return ResolvedRole{}, fmt.Errorf("decode %s policy resolve output: %w", bin, err)
	}
	if strings.TrimSpace(resolved.Model) == "" {
		return ResolvedRole{}, fmt.Errorf("%s policy resolve returned no model", bin)
	}
	return resolved, nil
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}
