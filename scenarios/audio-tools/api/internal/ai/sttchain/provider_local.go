package sttchain

import (
	"context"
	"fmt"

	"audio-tools/internal/clock"
	voice "audio-tools/internal/stt/pipeline"
	"audio-tools/internal/stt/whisperinfo"
)

// LocalProvider wraps voice.Service.Transcribe (Whisper backend).
//
// ModelID/Model() reflect what the local Whisper sidecar is actually
// running — resolved at construction via the injected whisperinfo.Client
// seam, not a hard-coded string. See scenarios/audio-tools/docs/internal/
// SEAMS.md row "whisperinfo.Client".
type LocalProvider struct {
	svc  *voice.Service
	clk  clock.Clock
	info whisperinfo.Client
}

// NewLocalProvider constructs a Local STT provider with the system clock
// and the env-backed whisperinfo.EnvClient.
func NewLocalProvider(svc *voice.Service) *LocalProvider {
	return &LocalProvider{svc: svc, clk: clock.System{}, info: whisperinfo.New()}
}

// NewLocalProviderWith constructs a LocalProvider with a custom clock
// and (optional) custom info client. Either may be nil to use defaults.
func NewLocalProviderWith(svc *voice.Service, clk clock.Clock, info whisperinfo.Client) *LocalProvider {
	if clk == nil {
		clk = clock.System{}
	}
	if info == nil {
		info = whisperinfo.New()
	}
	return &LocalProvider{svc: svc, clk: clk, info: info}
}

func (p *LocalProvider) Type() ProviderTier { return TierLocal }

func (p *LocalProvider) IsAvailable(ctx context.Context) bool {
	if p == nil || p.svc == nil {
		return false
	}
	return p.svc.WhisperAvailable(ctx)
}

func (p *LocalProvider) Transcribe(ctx context.Context, req Request) (*Result, error) {
	if p == nil || p.svc == nil {
		return nil, fmt.Errorf("audio-tools/sttchain: local provider not configured")
	}
	clk := p.clk
	if clk == nil {
		clk = clock.System{}
	}
	start := clk.Now()
	tr, err := p.svc.Transcribe(ctx, req.Audio, req.Format, req.Language, req.InitialPrompt, req.VADFilter)
	if err != nil {
		return nil, err
	}
	res := &Result{
		Text:             tr.Text,
		DetectedLanguage: req.Language,
		Tier:             TierLocal,
		ProviderID:       "whisper-local",
		ModelID:          p.modelID(),
		Latency:          clk.Now().Sub(start),
	}
	if tr.HasConfidence {
		res.Confidence = &Confidence{NoSpeechProb: tr.NoSpeechProb, AvgLogProb: tr.AvgLogProb}
	}
	if len(tr.Words) > 0 {
		res.Words = make([]TimedWord, len(tr.Words))
		for i, w := range tr.Words {
			res.Words[i] = TimedWord{Word: w.Word, Start: w.Start, End: w.End, Prob: w.Prob}
		}
	}
	return res, nil
}

// Model reports the loaded sidecar model. Returns whisperinfo.ModelUnknown
// when the resource hasn't propagated AUDIO_WHISPER_MODEL — never fabricates
// a name.
func (p *LocalProvider) Model() string { return p.modelID() }

func (p *LocalProvider) modelID() string {
	if p == nil || p.info == nil {
		return whisperinfo.ModelUnknown
	}
	return p.info.CurrentModel().ModelID
}

// Traits reports the LocalProvider as a batch-only provider. The
// streaming surface is provided externally by VADSegmentStrategy or
// OverlapAgreeStrategy calling Transcribe per segment/window.
func (p *LocalProvider) Traits() ProviderTraits {
	return ProviderTraits{
		Batch:      true,
		Stream:     false,
		Strategies: []StrategyKind{StrategyVADSegment, StrategyOverlapAgree, StrategyBuffered},
	}
}

// TranscribeStreaming on the LocalProvider always declines native
// streaming. The chain's selector pairs this provider with a batch
// strategy (VAD-segment or overlap-and-agree) that drives Transcribe
// per segment.
func (p *LocalProvider) TranscribeStreaming(_ context.Context, _ StreamStart, _ <-chan AudioChunk) (<-chan StreamEvent, error) {
	return nil, nil
}
