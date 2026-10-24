package sttchain

import (
	"context"
	"fmt"
	"os"
	"strings"
)

const (
	virtualReplayEngineID = "virtual-replay"
	virtualReplayProvider = "virtual-replay"
	virtualReplayModel    = "virtual-corpus"
	virtualReplayText     = "the quick brown fox jumps."
)

// VirtualReplayProvider is the accelerated-lane recognizer. It is not a
// production engine and is intentionally not present in sttengine/manifest.json.
// It consumes the same browser-produced PCM chunks and emits the same
// passthrough events as a native streaming provider, but advances recognition
// without spending model time on repeated corpus audio. This makes a
// 60-simulated-minute capture measurable in CI while realtime model evidence
// remains attributable to Kyutai (or a future native streaming engine).
//
// Selection has independent guards: the stream handler requires the explicit
// virtual-capture query pair plus either a server-owned test-isolation lease or
// the operator's VROOLI_AUDIO_SOAK_REPLAY=1 gate, and IsAvailable requires the
// same environment gate. The provider is absent from the production manifest,
// so an untrusted request cannot select it in a normal process.
type VirtualReplayProvider struct{}

func NewVirtualReplayProvider() *VirtualReplayProvider { return &VirtualReplayProvider{} }

func (p *VirtualReplayProvider) Type() ProviderTier { return TierLocal }

func (p *VirtualReplayProvider) IsAvailable(context.Context) bool {
	return p != nil && strings.EqualFold(strings.TrimSpace(os.Getenv("VROOLI_AUDIO_SOAK_REPLAY")), "1")
}

func (p *VirtualReplayProvider) Model() string { return virtualReplayModel }

func (p *VirtualReplayProvider) Traits() ProviderTraits {
	return ProviderTraits{
		Batch:                   false,
		Stream:                  true,
		Strategies:              []StrategyKind{StrategyPassthrough},
		BackendAcknowledgements: true,
	}
}

func (p *VirtualReplayProvider) Transcribe(context.Context, Request) (*Result, error) {
	return nil, fmt.Errorf("audio-tools/sttchain: virtual replay is streaming-only and test-gated")
}

func (p *VirtualReplayProvider) TranscribeStreaming(ctx context.Context, start StreamStart, chunks <-chan AudioChunk) (<-chan StreamEvent, error) {
	if p == nil || start.EngineID != virtualReplayEngineID {
		return nil, fmt.Errorf("audio-tools/sttchain: virtual replay provider is test-gated")
	}
	events := make(chan StreamEvent, 16)
	go func() {
		defer close(events)
		var first AudioChunk
		haveChunk := false
		partialSent := false
		for {
			select {
			case <-ctx.Done():
				events <- StreamEvent{Kind: StreamEventError, Error: ctx.Err()}
				events <- virtualReplayDone(start)
				return
			case chunk, ok := <-chunks:
				if !ok {
					events <- virtualReplayDone(start)
					return
				}
				if len(chunk.Audio) == 0 {
					continue
				}
				if !haveChunk {
					first = chunk
					haveChunk = true
					// Commit immediately after the first acknowledged interval. A
					// stop can cancel the transport while the final input channel is
					// closing; waiting for that close would turn a successful replay
					// into retained-audio fallback. The segment is deliberately
					// timeline-stable (session + first sequence, no generation), so a
					// reconnect cannot create a duplicate committed segment.
					events <- StreamEvent{Kind: StreamEventSegment, Segment: &SegmentEvent{
						Text:             virtualReplayText,
						SegmentID:        fmt.Sprintf("%s:%d", start.SessionID, first.Sequence),
						Generation:       start.Generation,
						StartSample:      first.StartSample,
						EndSample:        first.EndSample,
						AlignmentQuality: "exact-test-replay",
						DetectedLanguage: start.Language,
						ProviderTier:     TierLocal,
						ProviderID:       virtualReplayProvider,
						ModelID:          virtualReplayModel,
					}}
				}
				if !partialSent {
					events <- StreamEvent{Kind: StreamEventPartial, Partial: &PartialEvent{Text: virtualReplayText}}
					partialSent = true
				}
				// Coverage is acknowledged per browser interval. The replay
				// provider does not acknowledge text commits, so continuous
				// speech and burst-shaped accelerated input exercise the same
				// retention contract as a native streaming backend.
				events <- StreamEvent{Kind: StreamEventAcknowledgement, Acknowledgement: &AcknowledgementEvent{
					ReceivedSequence: int64(chunk.Sequence), ProcessedSequence: int64(chunk.Sequence),
					ReceivedEndSample: chunk.EndSample, ProcessedEndSample: chunk.EndSample,
				}}
			}
		}
	}()
	return events, nil
}

func virtualReplayDone(start StreamStart) StreamEvent {
	return StreamEvent{Kind: StreamEventDone, Done: &DoneEvent{
		FinalText: virtualReplayText, LockedTier: TierLocal,
		ProviderID: virtualReplayProvider, ModelID: virtualReplayModel,
	}}
}

var _ Provider = (*VirtualReplayProvider)(nil)
