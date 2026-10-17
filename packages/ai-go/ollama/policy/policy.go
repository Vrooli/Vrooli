// Package policy resolves Ollama role/model metadata through resource-ollama.
package policy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

const DefaultBin = "resource-ollama"

type Provenance struct {
	SourceKind      string `json:"source_kind"`
	Confidence      string `json:"confidence"`
	Source          string `json:"source"`
	ObservedAt      string `json:"observed_at"`
	HostFingerprint string `json:"host_fingerprint,omitempty"`
	OllamaVersion   string `json:"ollama_version,omitempty"`
	SampleCount     int    `json:"sample_count"`
}

type ResolvedRole struct {
	SchemaVersion        string                `json:"schema_version"`
	PolicyPath           string                `json:"policy_path"`
	Role                 string                `json:"role,omitempty"`
	Source               string                `json:"source"`
	Model                string                `json:"model"`
	Fallbacks            []string              `json:"fallbacks,omitempty"`
	RequiredCapabilities []string              `json:"required_capabilities,omitempty"`
	Capabilities         []string              `json:"capabilities"`
	ContextWindowTokens  int                   `json:"context_window_tokens,omitempty"`
	EmbeddingDimensions  int                   `json:"embedding_dimensions,omitempty"`
	DiskSizeGBEstimate   float64               `json:"disk_size_gb_estimate"`
	RAMGBEstimate        float64               `json:"ram_gb_estimate"`
	VRAMGBEstimate       float64               `json:"vram_gb_estimate"`
	DefaultEligible      bool                  `json:"default_eligible"`
	RoleProvenance       *Provenance           `json:"role_provenance,omitempty"`
	Provenance           map[string]Provenance `json:"provenance,omitempty"`
}

// CommandRunner is the process-exec seam for resource-ollama.
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
		return ResolvedRole{}, errors.New("ollama policy role is required")
	}
	return r.resolve(ctx, "--role", role)
}

func (r Resolver) ResolveModel(ctx context.Context, model string) (ResolvedRole, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return ResolvedRole{}, errors.New("ollama policy model is required")
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
