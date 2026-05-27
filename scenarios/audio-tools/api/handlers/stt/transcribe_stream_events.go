// Translation between sttchain.StreamEvent and the proto wire shape
// used by the TranscribeStream bidi-stream RPC. Pure functions only;
// no I/O and no handler state.
package stt

import (
	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/protomap"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// protoForEvent translates a chain StreamEvent to its proto wire shape.
func protoForEvent(ev sttchain.StreamEvent) *sttv1.TranscribeStreamEvent {
	switch ev.Kind {
	case sttchain.StreamEventPartial:
		if ev.Partial == nil {
			return nil
		}
		return &sttv1.TranscribeStreamEvent{
			Event: &sttv1.TranscribeStreamEvent_Partial{
				Partial: &sttv1.StreamPartial{Text: ev.Partial.Text},
			},
		}
	case sttchain.StreamEventSegment:
		if ev.Segment == nil {
			return nil
		}
		return &sttv1.TranscribeStreamEvent{
			Event: &sttv1.TranscribeStreamEvent_Segment{
				Segment: &sttv1.StreamSegment{
					Text:             ev.Segment.Text,
					StartMs:          ev.Segment.StartMs,
					EndMs:            ev.Segment.EndMs,
					DetectedLanguage: ev.Segment.DetectedLanguage,
					ProviderTier:     protomap.ProviderTierToProto(string(ev.Segment.ProviderTier)),
					ModelId:          ev.Segment.ModelID,
					LatencyMs:        ev.Segment.LatencyMs,
				},
			},
		}
	case sttchain.StreamEventWakeWord:
		if ev.WakeWord == nil {
			return nil
		}
		return &sttv1.TranscribeStreamEvent{
			Event: &sttv1.TranscribeStreamEvent_WakeWord{
				WakeWord: &sttv1.StreamWakeWord{Score: ev.WakeWord.Score, SampleId: ev.WakeWord.SampleID},
			},
		}
	case sttchain.StreamEventSpeakerRejection:
		if ev.SpeakerRejection == nil {
			return nil
		}
		return &sttv1.TranscribeStreamEvent{
			Event: &sttv1.TranscribeStreamEvent_SpeakerRejection{
				SpeakerRejection: &sttv1.StreamSpeakerRejection{
					Reason:       ev.SpeakerRejection.Reason,
					FallbackUsed: ev.SpeakerRejection.FallbackUsed,
				},
			},
		}
	case sttchain.StreamEventError:
		msg := ""
		if ev.Error != nil {
			msg = ev.Error.Error()
		}
		return &sttv1.TranscribeStreamEvent{
			Event: &sttv1.TranscribeStreamEvent_Error{
				Error: &sttv1.StreamError{Code: "provider_failure", Message: msg},
			},
		}
	case sttchain.StreamEventVadState:
		if ev.VadState == nil {
			return nil
		}
		return &sttv1.TranscribeStreamEvent{
			Event: &sttv1.TranscribeStreamEvent_VadState{
				VadState: &sttv1.StreamVadState{
					Voiced:           ev.VadState.Voiced,
					SilenceElapsedMs: ev.VadState.SilenceElapsedMs,
					SilenceTimeoutMs: ev.VadState.SilenceTimeoutMs,
					TickSeq:          ev.VadState.TickSeq,
					SilenceTimedOut:  ev.VadState.SilenceTimedOut,
				},
			},
		}
	case sttchain.StreamEventDone:
		done := ev.Done
		if done == nil {
			done = &sttchain.DoneEvent{}
		}
		return &sttv1.TranscribeStreamEvent{
			Event: &sttv1.TranscribeStreamEvent_Done{
				Done: &sttv1.StreamDone{
					FinalText:       done.FinalText,
					ProviderTier:    protomap.ProviderTierToProto(string(done.LockedTier)),
					ProviderId:      done.ProviderID,
					ModelId:         done.ModelID,
					LatencyMs:       done.LatencyMs,
					FellBackToUnary: done.FellBackToUnary,
				},
			},
		}
	}
	return nil
}
