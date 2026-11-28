package sttchain

import (
	"context"
	"fmt"

	voice "audio-tools/internal/stt/pipeline"
	"audio-tools/internal/stt/whisperinfo"

	"github.com/vrooli/api-core/schedule"
)

// LocalProvider is the Whisper chain provider. The shared BatchProvider owns
// tier identity, strategy traits, result provenance, timing, and language
// fallback; this wrapper keeps the legacy constructor and read-only health
// method while Whisper-specific request/result conversion lives in the
// whisperBatchBackend.
type LocalProvider struct {
	BatchProvider
	whisper *whisperBatchBackend
}

// NewLocalProvider constructs a Local STT provider with the system clock and
// the env-backed whisperinfo.EnvClient.
func NewLocalProvider(svc *voice.WhisperTranscriber) *LocalProvider {
	return NewLocalProviderWith(svc, nil, nil)
}

// NewLocalProviderWith constructs a LocalProvider with a custom clock and
// (optional) custom info client. Either may be nil to use defaults.
func NewLocalProviderWith(svc *voice.WhisperTranscriber, clk schedule.Clock, info whisperinfo.Client) *LocalProvider {
	if clk == nil {
		clk = schedule.System()
	}
	if info == nil {
		info = whisperinfo.New()
	}
	backend := &whisperBatchBackend{svc: svc, info: info}
	return &LocalProvider{
		BatchProvider: *NewBatchProvider("whisper-local", backend, clk),
		whisper:       backend,
	}
}

// Model reports the loaded sidecar model. Returns whisperinfo.ModelUnknown
// when the resource hasn't propagated AUDIO_WHISPER_MODEL; never fabricates a
// model name.
func (p *LocalProvider) Model() string {
	if p == nil || p.whisper == nil {
		return whisperinfo.ModelUnknown
	}
	return p.whisper.Model()
}

// whisperBatchBackend adapts the Whisper service's richer pipeline response
// into the shared batch result contract. It deliberately retains Whisper's
// recovery-capable availability probe and its distinct observational probe.
type whisperBatchBackend struct {
	svc  *voice.WhisperTranscriber
	info whisperinfo.Client
}

func (b *whisperBatchBackend) IsAvailable(ctx context.Context) bool {
	return b != nil && b.svc != nil && b.svc.EnsureWhisperAvailable(ctx)
}

func (b *whisperBatchBackend) IsServiceable(ctx context.Context) bool {
	return b != nil && b.svc != nil && b.svc.WhisperAvailable(ctx)
}

func (b *whisperBatchBackend) Model() string {
	if b == nil || b.info == nil {
		return whisperinfo.ModelUnknown
	}
	return b.info.CurrentModel().ModelID
}

func (b *whisperBatchBackend) Transcribe(ctx context.Context, req Request) (BatchResult, error) {
	if b == nil || b.svc == nil {
		return BatchResult{}, fmt.Errorf("audio-tools/sttchain: local provider not configured")
	}
	transcription, err := b.svc.Transcribe(ctx, req.Audio, req.Format, req.Language, req.InitialPrompt, req.VADFilter)
	if err != nil {
		return BatchResult{}, err
	}
	result := BatchResult{Text: transcription.Text, DetectedLanguage: req.Language}
	if transcription.HasConfidence {
		result.Confidence = &Confidence{NoSpeechProb: transcription.NoSpeechProb, AvgLogProb: transcription.AvgLogProb}
	}
	if len(transcription.Words) > 0 {
		result.Words = make([]TimedWord, len(transcription.Words))
		for i, word := range transcription.Words {
			result.Words[i] = TimedWord{Word: word.Word, Start: word.Start, End: word.End, Prob: word.Prob}
		}
	}
	return result, nil
}

var (
	_ BatchBackend = (*whisperBatchBackend)(nil)
	_ Provider     = (*LocalProvider)(nil)
)
