package bootstrap

import (
	"context"
	"io"

	"audio-tools/internal/capabilities"
	"audio-tools/internal/httpc"
	"audio-tools/internal/logx"
	inttts "audio-tools/internal/tts"
)

func newLocalTTSService(baseURL, model, speedMode, cacheBackend, capabilityID string, doer httpc.Doer, logger logx.Logger, capsRegistry *capabilities.Registry, ttsCache *inttts.Cache) *inttts.Service {
	synth := &inttts.SpeechSynthesizer{BaseURL: baseURL, Model: model, SpeedMode: speedMode, Doer: doer}
	voiceLister := &inttts.KokoroVoiceLister{BaseURL: baseURL, Doer: doer}
	ttsCfg := inttts.DefaultConfig()
	return inttts.NewService(inttts.Deps{
		Logger: logger, GetConfig: func() inttts.Config { return ttsCfg }, CacheBackend: cacheBackend,
		SetConfig: func(c inttts.Config) { ttsCfg = c }, PersistConfig: func(inttts.Config) error { return nil },
		KokoroCapability: func(ctx context.Context) (string, string) {
			if capsRegistry.IsProviderLive(ctx, capabilityID) {
				return "available", model + " (Local)"
			}
			return "unavailable", model + " (Local)"
		},
		InvalidateBackendLiveness: capsRegistry.Invalidate,
		SynthesizeAudio: func(ctx context.Context, in inttts.SynthesizeInput) (io.ReadCloser, string, error) {
			return synth.Synthesize(ctx, inttts.SynthesizeRequest{Input: in.Input, Voice: in.Voice, ResponseFormat: in.ResponseFormat, Speed: in.Speed})
		},
		GetCache: func(key inttts.CacheKey) (inttts.SynthesizeResult, bool) {
			out, ok := ttsCache.Get(key)
			if !ok {
				return inttts.SynthesizeResult{}, false
			}
			return inttts.SynthesizeResult{Audio: out.Audio, ContentType: out.ContentType}, true
		},
		PutCache:         func(key inttts.CacheKey, audio []byte, ct string) { ttsCache.Put(key, audio, ct) },
		ListVoiceCatalog: voiceLister.ListVoices,
	})
}
