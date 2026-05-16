package audioports

import (
	"context"
	"io"

	inttts "web-console/internal/tts"
)

// LocalTextToSpeech is the production TextToSpeech adapter backed by the
// in-process internal/tts synthesizer, voice lister, and cache. The future
// audio-tools client will replace this with a remote-call adapter;
// orchestration code is unchanged when that swap happens.
type LocalTextToSpeech struct {
	Synthesizer inttts.Synthesizer
	VoiceLister inttts.VoiceLister
	Cache       *inttts.Cache
}

// Synthesize calls the underlying Kokoro synthesizer and concatenates the
// stream into bytes. Cache-on-write happens here when EventID is set, so
// callers do not need to know about cache keying.
func (l LocalTextToSpeech) Synthesize(ctx context.Context, req TTSRequest) (TTSResult, error) {
	if l.Synthesizer == nil {
		return TTSResult{}, nil
	}
	body, ct, err := l.Synthesizer.Synthesize(ctx, inttts.SynthesizeRequest{
		Input:          req.Input,
		Voice:          req.Voice,
		ResponseFormat: req.ResponseFormat,
		Speed:          req.Speed,
	})
	if err != nil {
		return TTSResult{}, err
	}
	data, err := io.ReadAll(body)
	body.Close()
	if err != nil {
		return TTSResult{}, err
	}
	if req.EventID != "" && l.Cache != nil {
		version := req.Version
		if version == "" {
			version = "active"
		}
		l.Cache.Put(inttts.CacheKey{
			EventID: req.EventID,
			Voice:   req.Voice,
			Speed:   req.Speed,
			Version: version,
		}, data, ct)
	}
	return TTSResult{Audio: data, ContentType: ct}, nil
}

// ListVoices delegates to the voice lister and maps to port-level Voice.
func (l LocalTextToSpeech) ListVoices(ctx context.Context) ([]Voice, error) {
	if l.VoiceLister == nil {
		return nil, nil
	}
	out, err := l.VoiceLister.ListVoices(ctx)
	if err != nil {
		return nil, err
	}
	voices := make([]Voice, 0, len(out))
	for _, v := range out {
		voices = append(voices, Voice{ID: v.ID, Name: v.Name})
	}
	return voices, nil
}

// GetCached looks up a pre-synthesized event in the cache. Returns ok=false
// when the cache is unset or the key is missing.
func (l LocalTextToSpeech) GetCached(_ context.Context, key CacheLookup) (TTSResult, bool) {
	if l.Cache == nil {
		return TTSResult{}, false
	}
	version := key.Version
	if version == "" {
		version = "active"
	}
	entry, ok := l.Cache.Get(inttts.CacheKey{
		EventID: key.EventID,
		Voice:   key.Voice,
		Speed:   key.Speed,
		Version: version,
	})
	if !ok {
		return TTSResult{}, false
	}
	return TTSResult{Audio: entry.Audio, ContentType: entry.ContentType}, true
}

// Compile-time assertion.
var _ TextToSpeech = LocalTextToSpeech{}
