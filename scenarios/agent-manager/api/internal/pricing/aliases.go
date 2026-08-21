package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/cli-core/agentcatalog"
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

// A model's canonical name and provider do not change while a resource is
// installed, so resolution is memoized. Failures are memoized too, for a
// shorter window: an unresolvable model is retried by every pricing lookup,
// and without a negative cache that becomes an unbounded exec loop.
var (
	modelResolutionMu sync.Mutex
	modelResolutions  = map[string]modelResolution{}

	modelResolutionTTL         = 10 * time.Minute
	modelResolutionNegativeTTL = 1 * time.Minute

	modelResolutionNow = time.Now
)

type modelResolution struct {
	canonical string
	provider  string
	err       error
	at        time.Time
}

// UnknownModel is the sentinel the runner codecs record when they cannot
// detect the model for a run. It is never a real model name, so resolving it
// can only fail; callers short-circuit instead of asking a resource.
const UnknownModel = "unknown"

// resetModelResolutionCache drops every memoized resolution. Tests use it so a
// replaced command runner is actually consulted.
func resetModelResolutionCache() {
	modelResolutionMu.Lock()
	defer modelResolutionMu.Unlock()
	modelResolutions = map[string]modelResolution{}
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
	if model == UnknownModel {
		return "", "", fmt.Errorf("model is unresolved (%q sentinel); no resource lookup attempted", UnknownModel)
	}

	key := runner + "\x00" + model
	modelResolutionMu.Lock()
	if cached, ok := modelResolutions[key]; ok {
		ttl := modelResolutionTTL
		if cached.err != nil {
			ttl = modelResolutionNegativeTTL
		}
		if modelResolutionNow().Sub(cached.at) < ttl {
			modelResolutionMu.Unlock()
			return cached.canonical, cached.provider, cached.err
		}
	}
	modelResolutionMu.Unlock()

	canonical, provider, err := r.resolveUncached(ctx, runner, model)

	modelResolutionMu.Lock()
	modelResolutions[key] = modelResolution{canonical: canonical, provider: provider, err: err, at: modelResolutionNow()}
	modelResolutionMu.Unlock()

	return canonical, provider, err
}

func (r CLIModelResolver) resolveUncached(ctx context.Context, runner, model string) (string, string, error) {
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
	var response agentcatalog.ModelResolution
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
