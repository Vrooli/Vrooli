// Package evaldeps composes experiment evaluation's concrete corpus, STT, and
// speaker-resource adapters into the transport-free eval runner ports.
package evaldeps

import (
	"context"
	"fmt"

	"audio-tools/internal/ai/sttchain"
	intcorpus "audio-tools/internal/corpus"
	inteval "audio-tools/internal/eval"
	"audio-tools/internal/logx"
	sttpkg "audio-tools/internal/stt"
	"audio-tools/internal/stt/egress"
	"audio-tools/internal/stt/ingress"
	sttpipeline "audio-tools/internal/stt/pipeline"
	sttspeaker "audio-tools/internal/stt/speaker"
)

// New creates the runner dependencies for experiment reports.
func New(logger logx.Logger, corpusSvc *intcorpus.Service, providerForEngine func(string) sttchain.Provider, defaults sttpkg.StreamConfig, speakerClient *sttpipeline.SpeakerClient) inteval.RunnerDeps {
	return inteval.RunnerDeps{
		LoadClips: func(ctx context.Context, ids []string) ([]inteval.Clip, error) {
			if corpusSvc == nil {
				return nil, fmt.Errorf("eval corpus service is not configured")
			}
			metas := make([]intcorpus.Clip, 0, len(ids))
			if len(ids) == 0 {
				all, err := corpusSvc.ListClips(ctx, intcorpus.ListFilter{})
				if err != nil {
					return nil, err
				}
				metas = all
			} else {
				for _, id := range ids {
					clip, err := corpusSvc.GetClip(ctx, id)
					if err != nil {
						return nil, err
					}
					metas = append(metas, clip)
				}
			}
			clips := make([]inteval.Clip, 0, len(metas))
			for _, meta := range metas {
				audio, _, err := corpusSvc.GetClipAudio(ctx, meta.ID)
				if err != nil {
					return nil, fmt.Errorf("load audio for clip %q: %w", meta.ID, err)
				}
				rate := meta.SampleRateHz
				if rate <= 0 {
					rate = 16000
				}
				clips = append(clips, inteval.Clip{ID: meta.ID, PCM: audio, SampleRate: rate, Reference: meta.ReferenceText, Format: meta.Format})
			}
			return clips, nil
		},
		NewProvider:          func() sttchain.Provider { return providerForEngine("whisper-local") },
		NewProviderForEngine: providerForEngine,
		Defaults:             defaults,
		SpeakerResource:      speakerClient,
		NewSpeakerIsolation:  func() egress.SpeakerIsolation { return nil },
		NewSpeakerExtraction: func() ingress.TargetExtractor { return nil },
		NewSpeakerIsolationForConfig: func(cfg sttpipeline.SpeakerConfig, client *sttpipeline.SpeakerClient) egress.SpeakerIsolation {
			return sttspeaker.NewIsolation(cfg, client, logger)
		},
		NewSpeakerExtractionForConfig: func(cfg sttpipeline.SpeakerConfig, client *sttpipeline.SpeakerClient) ingress.TargetExtractor {
			return sttspeaker.NewExtraction(cfg, client)
		},
	}
}
