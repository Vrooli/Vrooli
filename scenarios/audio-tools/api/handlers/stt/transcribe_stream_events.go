// Translation between sttchain.StreamEvent and the proto wire shape
// used by the TranscribeStream bidi-stream RPC. Pure functions only;
// no I/O and no handler state.
package stt

import (
	"errors"
	"strings"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/protomap"
	sttpipeline "audio-tools/internal/stt/pipeline"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// streamErrorCode classifies a stream error into a machine-readable code the
// clients key off (plan L2). A typed backend-down error becomes "backend_starting"
// (transient — show a "starting…" affordance) or "backend_unavailable" (operator
// action). Everything else keeps the legacy "provider_failure" code. Shared by
// the Connect stream-event path and the WebSocket bridge so both surfaces stay
// in lockstep.
func streamErrorCode(err error) string {
	if err != nil && strings.Contains(err.Error(), "stt_busy") {
		return "stt_busy"
	}
	var backendErr *sttpipeline.STTBackendError
	if errors.As(err, &backendErr) {
		switch backendErr.EffectiveState() {
		case sttpipeline.STTBackendStateDegraded:
			return "backend_degraded"
		case sttpipeline.STTBackendStateStarting:
			return "backend_starting"
		case sttpipeline.STTBackendStateUnavailable:
			return "backend_unavailable"
		}
	}
	return "provider_failure"
}

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
					SegmentId:        ev.Segment.SegmentID,
					Generation:       ev.Segment.Generation,
					StartSample:      ev.Segment.StartSample,
					EndSample:        ev.Segment.EndSample,
					AlignmentQuality: ev.Segment.AlignmentQuality,
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
					Score:        ev.SpeakerRejection.Score,
					Threshold:    ev.SpeakerRejection.Threshold,
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
				Error: &sttv1.StreamError{Code: streamErrorCode(ev.Error), Message: msg},
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
					FinalText:          done.FinalText,
					ProviderTier:       protomap.ProviderTierToProto(string(done.LockedTier)),
					ProviderId:         done.ProviderID,
					ModelId:            done.ModelID,
					LatencyMs:          done.LatencyMs,
					FellBackToUnary:    done.FellBackToUnary,
					ProcessedSequence:  done.ProcessedSequence,
					ProcessedEndSample: done.ProcessedEndSample,
					TerminalReason:     done.TerminalReason,
				},
			},
		}
	case sttchain.StreamEventAcknowledgement:
		if ev.Acknowledgement == nil {
			return nil
		}
		return &sttv1.TranscribeStreamEvent{Event: &sttv1.TranscribeStreamEvent_Acknowledgement{Acknowledgement: &sttv1.StreamAcknowledgement{
			ReceivedSequence: ev.Acknowledgement.ReceivedSequence, ProcessedSequence: ev.Acknowledgement.ProcessedSequence,
			ReceivedEndSample: ev.Acknowledgement.ReceivedEndSample, ProcessedEndSample: ev.Acknowledgement.ProcessedEndSample,
			DeliveryClass: sttv1.StreamDeliveryClass_STREAM_DELIVERY_CLASS_DURABLE,
		}}}
	case sttchain.StreamEventSessionStatus:
		if ev.SessionStatus == nil {
			return nil
		}
		return &sttv1.TranscribeStreamEvent{Event: &sttv1.TranscribeStreamEvent_SessionStatus{SessionStatus: &sttv1.StreamSessionStatus{
			SessionId: ev.SessionStatus.SessionID, Generation: ev.SessionStatus.Generation, State: ev.SessionStatus.State,
			QueuePosition: ev.SessionStatus.QueuePosition, CapabilityOutcome: ev.SessionStatus.CapabilityOutcome, RecoveryGuidance: ev.SessionStatus.RecoveryGuidance,
		}}}
	}
	return nil
}
