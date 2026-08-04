package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/vrooli/cli-core/agentpolicy"
)

// ModelResolver is the narrow seam through which pricing asks a runner's
// resource to translate its model vocabulary. The resource owns all concrete
// model and provider knowledge.
type ModelResolver interface {
	Resolve(context.Context, string, string) (canonical, provider string, err error)
}

// CLIModelResolver uses the resource-owned model resolution contract. It is
// intentionally small so tests can replace it without invoking a resource.
type modelCommandRunner func(context.Context, string, ...string) ([]byte, error)

type CLIModelResolver struct {
	run modelCommandRunner
}

func NewCLIModelResolver() ModelResolver {
	return CLIModelResolver{run: func(ctx context.Context, command string, args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, command, args...).CombinedOutput()
	}}
}

func (r CLIModelResolver) Resolve(ctx context.Context, runner, model string) (string, string, error) {
	runner = strings.TrimSpace(runner)
	model = strings.TrimSpace(model)
	if runner == "" || strings.ContainsAny(runner, "/\\ \t\n") {
		return "", "", fmt.Errorf("invalid runner %q", runner)
	}
	if model == "" {
		return "", "", fmt.Errorf("empty model name")
	}
	commandRunner := r.run
	if commandRunner == nil {
		commandRunner = func(ctx context.Context, command string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, command, args...).CombinedOutput()
		}
	}
	out, err := commandRunner(ctx, "resource-"+runner, "models", "resolve", "--model", model, "--json")
	if err != nil {
		return "", "", fmt.Errorf("resource-%s model resolution: %s: %w", runner, strings.TrimSpace(string(out)), err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(out)))
	decoder.DisallowUnknownFields()
	var response agentpolicy.ModelResolution
	if err := decoder.Decode(&response); err != nil {
		return "", "", fmt.Errorf("decode resource-%s model resolution: %w", runner, err)
	}
	if response.Runner != runner || response.Model != model || strings.TrimSpace(response.CanonicalModel) == "" {
		return "", "", fmt.Errorf("resource-%s returned an invalid model resolution", runner)
	}
	return response.CanonicalModel, response.Provider, nil
}

// ResolveModelAlias is retained for callers that need to distinguish an
// unresolved bare name. Canonicalization itself is exclusively resource- or
// database-owned; Agent Manager does not infer a provider from a model ID.
func ResolveModelAlias(model string) (canonical, provider string, found bool) {
	model = strings.TrimSpace(model)
	if model == "" {
		return "", "", false
	}
	return model, "", false
}
