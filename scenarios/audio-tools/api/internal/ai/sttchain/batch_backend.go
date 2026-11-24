package sttchain

import (
	"context"
	"fmt"
	"strings"

	"github.com/vrooli/api-core/schedule"
)

// BatchBackend is the protocol boundary for a local, batch-only STT runtime.
// The chain owns tier routing and result provenance; a backend owns only its
// health check, wire request, and protocol result decoding. This keeps new
// batch protocols from reimplementing the strategy and provider lifecycle.
type BatchBackend interface {
	IsAvailable(context.Context) bool
	Transcribe(context.Context, Request) (BatchResult, error)
	Model() string
}

// BatchResult contains only signals batch providers can return. Empty words
// and confidence mean unknown; adapters must not synthesize backend evidence.
type BatchResult struct {
	Text             string
	DetectedLanguage string
	Words            []TimedWord
	Confidence       *Confidence
}

// BatchProvider adapts a protocol-specific BatchBackend to the common STT
// provider contract. It deliberately exposes no native streaming capability;
// existing VAD/overlap strategies can drive it by making bounded batch calls.
type BatchProvider struct {
	providerID string
	backend    BatchBackend
	clock      schedule.Clock
}

func NewBatchProvider(providerID string, backend BatchBackend, clock schedule.Clock) *BatchProvider {
	if clock == nil {
		clock = schedule.System()
	}
	return &BatchProvider{providerID: strings.TrimSpace(providerID), backend: backend, clock: clock}
}

func (p *BatchProvider) Type() ProviderTier { return TierLocal }

func (p *BatchProvider) IsAvailable(ctx context.Context) bool {
	return p != nil && p.backend != nil && p.backend.IsAvailable(ctx)
}

// IsServiceable is the read-only liveness seam. Backends that may recover or
// start a resource from IsAvailable can expose a separate observational probe;
// generic endpoints without that distinction fail closed for inventory use.
func (p *BatchProvider) IsServiceable(ctx context.Context) bool {
	if p == nil || p.backend == nil {
		return false
	}
	observer, ok := p.backend.(interface{ IsServiceable(context.Context) bool })
	return ok && observer.IsServiceable(ctx)
}

func (p *BatchProvider) Model() string {
	if p == nil || p.backend == nil {
		return ""
	}
	return p.backend.Model()
}

func (p *BatchProvider) Traits() ProviderTraits {
	return batchProviderTraits()
}

func batchProviderTraits() ProviderTraits {
	return ProviderTraits{Batch: true, Strategies: []StrategyKind{StrategyVADSegment, StrategyOverlapAgree, StrategyBuffered}}
}

func (p *BatchProvider) Transcribe(ctx context.Context, req Request) (*Result, error) {
	if p == nil || p.backend == nil || p.providerID == "" {
		return nil, fmt.Errorf("audio-tools/sttchain: batch provider is not configured")
	}
	started := p.clock.Now()
	got, err := p.backend.Transcribe(ctx, req)
	if err != nil {
		return nil, err
	}
	return &Result{
		Text:             got.Text,
		DetectedLanguage: firstNonEmpty(got.DetectedLanguage, req.Language),
		Tier:             TierLocal,
		ProviderID:       p.providerID,
		ModelID:          p.backend.Model(),
		Latency:          p.clock.Now().Sub(started),
		Words:            append([]TimedWord(nil), got.Words...),
		Confidence:       got.Confidence,
	}, nil
}

func (p *BatchProvider) TranscribeStreaming(context.Context, StreamStart, <-chan AudioChunk) (<-chan StreamEvent, error) {
	return nil, nil
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

var _ Provider = (*BatchProvider)(nil)
