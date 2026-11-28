package ttschain

import (
	"context"
	"errors"
	"fmt"

	"audio-tools/internal/modelcatalog"
	"audio-tools/internal/tts"

	"github.com/vrooli/api-core/schedule"
)

// LocalProvider wraps tts.Service.Synthesize (Kokoro backend).
type LocalProvider struct {
	svc          *tts.Service
	clk          schedule.Clock
	providerID   string
	modelID      string
	adapterID    string
	voiceMap     tts.AdapterVoiceMap
	cacheBackend string
	fallback     *LocalProvider
	ensurer      BackendEnsurer
	resource     string
}

// SetBackendEnsurer configures bounded on-demand startup for this local provider.
func (p *LocalProvider) SetBackendEnsurer(ensurer BackendEnsurer, resource string) {
	if p != nil {
		p.ensurer, p.resource = ensurer, resource
	}
}

func NewLocalProvider(svc *tts.Service) *LocalProvider {
	return NewConfiguredLocalProvider(svc, schedule.System(), "kokoro-local", defaultKokoroModelID(), tts.AdapterLocalKokoro, tts.LocalKokoroMapping)
}

func defaultKokoroModelID() string {
	for _, model := range modelcatalog.Default().ByProvider("kokoro-tts") {
		if model.Capability == modelcatalog.CapabilityTTS && model.HasRole("default") && model.DefaultEligible() {
			return model.ID
		}
	}
	panic("audio-tools/ttschain: provider catalog has no eligible Kokoro default model")
}

// NewConfiguredLocalProvider binds runtime identity and voice mapping for a
// selected local speech engine while preserving the same synthesis contract.
func NewConfiguredLocalProvider(svc *tts.Service, clk schedule.Clock, providerID, modelID, adapterID string, voiceMap tts.AdapterVoiceMap) *LocalProvider {
	if clk == nil {
		clk = schedule.System()
	}
	return &LocalProvider{svc: svc, clk: clk, providerID: providerID, modelID: modelID, adapterID: adapterID, voiceMap: voiceMap, cacheBackend: modelID}
}

// NewFailoverLocalProvider keeps Kale preferred while retaining the Kokoro
// local route as an availability fallback. Fallback is only attempted for
// ErrUnavailable; invalid input and synthesis faults remain visible.
func NewFailoverLocalProvider(preferred, fallback *LocalProvider) *LocalProvider {
	if preferred == nil {
		return fallback
	}
	preferred.fallback = fallback
	return preferred
}

// NewFailoverLocalProviders links declared providers in preference order. The
// last provider is the terminal route; each provider only falls back when its
// own backend is unavailable.
func NewFailoverLocalProviders(providers ...*LocalProvider) *LocalProvider {
	var head *LocalProvider
	for i := len(providers) - 1; i >= 0; i-- {
		if providers[i] == nil {
			continue
		}
		providers[i].fallback = head
		head = providers[i]
	}
	return head
}

func (p *LocalProvider) SetCacheBackend(backend string) {
	if p != nil && backend != "" {
		p.cacheBackend = backend
	}
}

// CacheBackends returns this provider and its local failover backends in
// preference order. Cache replay follows the composed chain instead of
// maintaining a second list of model/backend identifiers in the handler.
func (p *LocalProvider) CacheBackends() []string {
	seen := make(map[string]struct{})
	var out []string
	for current := p; current != nil; current = current.fallback {
		backend := current.cacheBackend
		if backend == "" {
			backend = current.modelID
		}
		if backend == "" {
			continue
		}
		if _, ok := seen[backend]; ok {
			continue
		}
		seen[backend] = struct{}{}
		out = append(out, backend)
	}
	return out
}

// NewLocalProviderWith constructs a LocalProvider with a custom schedule.
func NewLocalProviderWith(svc *tts.Service, clk schedule.Clock) *LocalProvider {
	if clk == nil {
		clk = schedule.System()
	}
	return NewConfiguredLocalProvider(svc, clk, "kokoro-local", defaultKokoroModelID(), tts.AdapterLocalKokoro, tts.LocalKokoroMapping)
}

func (p *LocalProvider) Type() ProviderTier { return TierLocal }

// ProviderID reports the preferred local provider identity for status and
// other surfaces that describe the configured route before a request runs.
func (p *LocalProvider) ProviderID() string {
	if p == nil {
		return ""
	}
	return p.providerID
}

// IsAvailable reports whether the local TTS backend can actually serve a
// request. It consults the same readiness signal Synthesize gates on, so the
// availability a caller is shown and the availability it will get are the same
// fact. Returning an unconditional true here made every operator surface --
// `settings providers`, the TTS status capability label -- report the local
// tier as up while Kokoro was not running at all.
func (p *LocalProvider) IsAvailable(ctx context.Context) bool {
	for current := p; current != nil; current = current.fallback {
		if current.svc != nil && localBackendReady(ctx, current) {
			return true
		}
	}
	return false
}

func (p *LocalProvider) Synthesize(ctx context.Context, req Request) (*Result, error) {
	if p == nil {
		return nil, fmt.Errorf("audio-tools/ttschain: local provider not configured")
	}
	clk := p.clk
	if clk == nil {
		clk = schedule.System()
	}
	start := clk.Now()
	var lastErr error
	for provider := p; provider != nil; provider = provider.fallback {
		if provider.svc == nil {
			lastErr = fmt.Errorf("audio-tools/ttschain: local provider %q not configured", provider.providerID)
			continue
		}
		if !localBackendReady(ctx, provider) {
			if err := ensureBackend(ctx, provider); err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
				return nil, err
			}
		}
		if !localBackendReady(ctx, provider) {
			lastErr = tts.ErrUnavailable
			continue
		}
		out, voice, err := provider.synthesize(ctx, req)
		if errors.Is(err, tts.ErrUnavailable) {
			lastErr = err
			continue
		}
		if err != nil {
			return nil, err
		}
		return &Result{
			Audio: out.Audio, ContentType: out.ContentType, Tier: TierLocal,
			ProviderID: provider.providerID, ModelID: provider.modelID,
			VoiceUsed: voice, CacheBackend: provider.cacheBackend,
			Latency: clk.Now().Sub(start),
		}, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("audio-tools/ttschain: local provider not configured")
	}
	return nil, lastErr
}

func ensureBackend(ctx context.Context, p *LocalProvider) error {
	if p == nil || p.svc == nil || p.ensurer == nil || p.resource == "" || localBackendReady(ctx, p) {
		return nil
	}
	if err := p.ensurer.EnsureRunning(ctx, p.resource); err != nil {
		return err
	}
	p.svc.InvalidateBackendReadiness()
	return nil
}

func localBackendReady(ctx context.Context, provider *LocalProvider) bool {
	return provider != nil && provider.svc != nil && provider.svc.LocalBackendReady(ctx)
}

func (p *LocalProvider) synthesize(ctx context.Context, req Request) (tts.SynthesizeResult, string, error) {
	in := tts.SynthesizeInput{
		Input:          req.Text,
		Voice:          resolveLocalVoiceWith(req.Voice, req.VoiceOverrides, p.adapterID, p.voiceMap),
		ResponseFormat: req.ResponseFormat,
		Speed:          req.Speed,
		EventID:        req.EventID,
		Version:        req.Version,
	}
	out, err := p.svc.Synthesize(ctx, in)
	return out, in.Voice, err
}

func (p *LocalProvider) Model() string {
	if p == nil || p.modelID == "" {
		return "kokoro"
	}
	return p.modelID
}

// StreamingCapability for Local Kokoro is false today; Phase D upgrades
// this to true once incremental synthesis is wired.
func (p *LocalProvider) StreamingCapability() bool { return false }

func (p *LocalProvider) SynthesizeStreaming(_ context.Context, _ Request) (<-chan AudioFrame, error) {
	return nil, nil
}

// resolveLocalVoice maps canonical voice IDs to Kokoro voice names using the
// shared canonical voice catalog.
func resolveLocalVoice(canonical string, overrides map[string]string) string {
	return resolveLocalVoiceWith(canonical, overrides, tts.AdapterLocalKokoro, tts.LocalKokoroMapping)
}

func resolveLocalVoiceWith(canonical string, overrides map[string]string, adapterID string, mapping tts.AdapterVoiceMap) string {
	// Explicit overrides win before the canonical mapping; the current adapter id
	// takes precedence over the pre-catalog key when both are present.
	if v, ok := overrides[adapterID]; ok && v != "" {
		return v
	}
	if adapterID == tts.AdapterLocalKokoro || adapterID == tts.LegacyAdapterLocalKokoro {
		if v, ok := overrides[tts.LegacyAdapterLocalKokoro]; ok && v != "" {
			return v
		}
	}
	if v, ok := mapping[canonical]; ok && v != "" {
		return v
	}
	// Unknown canonical ids pass through to Kokoro unchanged.
	return canonical
}
