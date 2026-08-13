package measureindex

import (
	"context"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	measures "github.com/vrooli/measures-go"
	"measures-health/internal/validation"
)

// Provider is the federated measures provider: the indexed corpus + the shared
// measures-go Engine that matches a question, resolves params, applies the
// auto-execution gate (keyed on effect), and (for a safe read-only measure at
// high confidence) proxies execution to the owning scenario. It is the brain
// behind the search-hub "measure" provider RPC.
type Provider struct {
	matcher   *LexicalMatcher
	engine    *measures.Engine
	threshold float64
	avail     func(ctx context.Context) bool
	usesLLM   bool
	indexedAt time.Time
}

// Config tunes a Provider. The zero value is production-ready (filesystem
// executor via discovery + ollama extractor + default threshold); tests override
// the seams to stay offline and deterministic.
type Config struct {
	// Threshold is the auto-execute confidence gate theta. <=0 uses
	// measures.DefaultConfidenceThreshold.
	Threshold float64
	// Now anchors relative time-window resolution (defaults to time.Now).
	Now func() time.Time
	// Executor is the execution-proxy. nil uses the production HTTPExecutor over
	// api-core discovery; a test injects a fake (or measures.Executor(nil) to
	// force resolve-only).
	Executor measures.Executor
	// Extractor is the constrained param extractor. A supplied extractor always
	// wins; otherwise the provider keeps the interactive search path
	// deterministic unless EnableLLMExtraction is explicitly enabled.
	Extractor measures.ParamExtractor
	// Completer overrides the extractor's transport when LLM extraction is
	// explicitly enabled; nil uses the gateway.
	Completer measures.Completer
	// EnableLLMExtraction opts this provider into the potentially slow
	// constrained extractor for non-canonical parameters. It is deliberately
	// false by default: search-hub needs an interactive routing leg, while a
	// caller that needs a fully parameterized computation can use the owning
	// measure's direct execute surface.
	EnableLLMExtraction bool
	// OllamaAvailable reports extractor-backend reachability for Status. nil uses
	// the real `resource-ollama status` probe; tests inject a deterministic stub
	// so Status stays hermetic.
	OllamaAvailable func(ctx context.Context) bool
}

// NewProvider builds a Provider over the harvested corpus. Production passes the
// zero Config; the search handler wires it from main.go.
func NewProvider(decls []measures.MeasureDeclaration, cfg Config) *Provider {
	threshold := cfg.Threshold
	if threshold <= 0 {
		threshold = measures.DefaultConfidenceThreshold
	}

	extractor := cfg.Extractor
	if extractor == nil {
		if cfg.EnableLLMExtraction {
			completer := cfg.Completer
			if completer == nil {
				completer = newOllamaCompleter()
			}
			extractor = measures.NewLLMExtractor(completer)
		} else {
			extractor = measures.NoopExtractor{}
		}
	}

	executor := cfg.Executor
	if executor == nil {
		executor = measures.NewHTTPExecutor(measures.BaseURLResolverFunc(resolveMeasuresBaseURL))
	}

	avail := cfg.OllamaAvailable
	if avail == nil {
		avail = ollamaAvailable
	}

	matcher := NewLexicalMatcher(decls)

	opts := []measures.EngineOption{
		measures.WithThreshold(threshold),
		measures.WithExtractor(extractor),
		measures.WithExecutor(executor),
	}
	if cfg.Now != nil {
		opts = append(opts, measures.WithEngineClock(cfg.Now))
	}

	return &Provider{
		matcher:   matcher,
		engine:    measures.NewEngine(matcher, opts...),
		threshold: threshold,
		avail:     avail,
		usesLLM:   cfg.EnableLLMExtraction || cfg.Extractor != nil,
		indexedAt: time.Now().UTC(),
	}
}

// Query matches the question to the best measure and returns its resolved /
// executed hit (at most one — a measures provider returns THE answer, not a
// browse list). It returns (nil, "none", nil) on no match (an honest empty
// result) and surfaces resolve/exec errors so the handler degrades gracefully.
// The `matcher` label reports which leg answered ("lexical" | "none").
func (p *Provider) Query(ctx context.Context, question string, limit int) ([]*measures.MeasureHit, string, error) {
	if strings.TrimSpace(question) == "" || p.matcher.Len() == 0 {
		return nil, "none", nil
	}
	hit, err := p.engine.Answer(ctx, question)
	if err != nil {
		return nil, "lexical", err
	}
	if hit == nil {
		return nil, "lexical", nil
	}
	_ = limit // best-1 semantics; reserved for a future multi-candidate matcher.
	return []*measures.MeasureHit{hit}, "lexical", nil
}

// Len reports the number of indexed measures.
func (p *Provider) Len() int { return p.matcher.Len() }

// IndexTimestamp is the materialization time of the declarations supplied to
// NewProvider. Measures are a live lexical corpus after construction, so no
// background vector reconciliation timestamp exists to report.
func (p *Provider) IndexTimestamp() time.Time { return p.indexedAt }

// Status reports provider availability for the Connect Status RPC and the
// search-hub status_endpoint: the index size, whether the ollama extractor is
// reachable, and the active matcher leg. qdrant is reported false until the
// aisearch hybrid index leg is wired (the lexical matcher needs no vector store).
func (p *Provider) Status(ctx context.Context) (available, ollama, qdrant bool, indexed int, matcher string) {
	indexed = p.matcher.Len()
	available = indexed > 0
	ollama = p.usesLLM && p.avail(ctx)
	qdrant = false
	matcher = "lexical"
	return available, ollama, qdrant, indexed, matcher
}

// resolveMeasuresBaseURL maps an owning scenario id to the base URL of its
// measures serve endpoint (scheme://host:port + the conventional mount prefix),
// resolved via api-core discovery — never client-computed. It is the same
// resolution the behavioral probe uses, so the index and the probe agree on
// where a scenario mounts its measures.
func resolveMeasuresBaseURL(ctx context.Context, scenario string) (string, error) {
	base, err := discovery.ResolveScenarioURLDefault(ctx, scenario)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(base, "/") + validation.DefaultMeasuresMountPath, nil
}
