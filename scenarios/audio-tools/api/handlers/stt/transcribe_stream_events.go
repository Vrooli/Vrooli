// Translation between sttchain.StreamEvent and the proto wire shape
// used by the TranscribeStream bidi-stream RPC. Pure functions only;
// no I/O and no handler state.
package stt

import (
	"errors"
	"strings"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/audioformat"
	"audio-tools/internal/capabilities"
	"audio-tools/internal/protomap"
	sttpipeline "audio-tools/internal/stt/pipeline"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// streamErrorCode classifies a stream error into a machine-readable code the
// clients key off (plan L2). A typed backend-down error becomes "backend_starting"
// (transient — show a "starting…" affordance) or "backend_unavailable" (operator
// action). Everything else keeps the legacy "provider_failure" code for
// Connect clients.
func streamErrorCode(err error) string {
	if err != nil && strings.Contains(err.Error(), "stt_busy") {
		return "stt_busy"
	}
	if errors.Is(err, audioformat.ErrStreamDecode) {
		return "input_decode_failed"
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
		return protoPartial(ev.Partial)
	case sttchain.StreamEventSegment:
		return protoSegment(ev.Segment)
	case sttchain.StreamEventWakeWord:
		return protoWakeWord(ev.WakeWord)
	case sttchain.StreamEventSpeakerRejection:
		return protoSpeakerRejection(ev.SpeakerRejection)
	case sttchain.StreamEventError:
		return protoStreamError(ev.Error)
	case sttchain.StreamEventVadState:
		return protoVadState(ev.VadState)
	case sttchain.StreamEventDone:
		return protoDone(ev.Done)
	case sttchain.StreamEventAcknowledgement:
		return protoAcknowledgement(ev.Acknowledgement)
	case sttchain.StreamEventSessionStatus:
		return protoSessionStatus(ev.SessionStatus)
	default:
		return nil
	}
}

func protoPartial(value *sttchain.PartialEvent) *sttv1.TranscribeStreamEvent {
	if value == nil {
		return nil
	}
	return &sttv1.TranscribeStreamEvent{Event: &sttv1.TranscribeStreamEvent_Partial{Partial: &sttv1.StreamPartial{Text: value.Text}}}
}

func protoSegment(value *sttchain.SegmentEvent) *sttv1.TranscribeStreamEvent {
	if value == nil {
		return nil
	}
	return &sttv1.TranscribeStreamEvent{Event: &sttv1.TranscribeStreamEvent_Segment{Segment: &sttv1.StreamSegment{
		Text: value.Text, StartMs: value.StartMs, EndMs: value.EndMs,
		DetectedLanguage: value.DetectedLanguage, ProviderTier: protomap.ProviderTierToProto(string(value.ProviderTier)),
		ModelId: value.ModelID, LatencyMs: value.LatencyMs, SegmentId: value.SegmentID, Generation: value.Generation,
		StartSample: value.StartSample, EndSample: value.EndSample, AlignmentQuality: value.AlignmentQuality,
		SpeakerPolicy: value.SpeakerPolicy,
	}}}
}

func protoWakeWord(value *sttchain.WakeWordEvent) *sttv1.TranscribeStreamEvent {
	if value == nil {
		return nil
	}
	return &sttv1.TranscribeStreamEvent{Event: &sttv1.TranscribeStreamEvent_WakeWord{WakeWord: &sttv1.StreamWakeWord{Score: value.Score, SampleId: value.SampleID}}}
}

func protoSpeakerRejection(value *sttchain.SpeakerRejectionEvent) *sttv1.TranscribeStreamEvent {
	if value == nil {
		return nil
	}
	return &sttv1.TranscribeStreamEvent{Event: &sttv1.TranscribeStreamEvent_SpeakerRejection{SpeakerRejection: &sttv1.StreamSpeakerRejection{
		Reason: value.Reason, FallbackUsed: value.FallbackUsed, Score: value.Score, Threshold: value.Threshold,
	}}}
}

func protoStreamError(err error) *sttv1.TranscribeStreamEvent {
	message := ""
	if err != nil {
		message = err.Error()
	}
	return &sttv1.TranscribeStreamEvent{Event: &sttv1.TranscribeStreamEvent_Error{Error: &sttv1.StreamError{Code: streamErrorCode(err), Message: message}}}
}

func protoVadState(value *sttchain.VadStateEvent) *sttv1.TranscribeStreamEvent {
	if value == nil {
		return nil
	}
	return &sttv1.TranscribeStreamEvent{Event: &sttv1.TranscribeStreamEvent_VadState{VadState: &sttv1.StreamVadState{
		Voiced: value.Voiced, SilenceElapsedMs: value.SilenceElapsedMs, SilenceTimeoutMs: value.SilenceTimeoutMs,
		TickSeq: value.TickSeq, SilenceTimedOut: value.SilenceTimedOut,
	}}}
}

func protoDone(value *sttchain.DoneEvent) *sttv1.TranscribeStreamEvent {
	if value == nil {
		value = &sttchain.DoneEvent{}
	}
	return &sttv1.TranscribeStreamEvent{Event: &sttv1.TranscribeStreamEvent_Done{Done: &sttv1.StreamDone{
		FinalText: value.FinalText, ProviderTier: protomap.ProviderTierToProto(string(value.LockedTier)),
		ProviderId: capabilities.CanonicalProviderID(value.ProviderID), ModelId: value.ModelID, LatencyMs: value.LatencyMs,
		FellBackToUnary: value.FellBackToUnary, ProcessedSequence: value.ProcessedSequence,
		ProcessedEndSample: value.ProcessedEndSample, TerminalReason: value.TerminalReason,
	}}}
}

func protoAcknowledgement(value *sttchain.AcknowledgementEvent) *sttv1.TranscribeStreamEvent {
	if value == nil {
		return nil
	}
	return &sttv1.TranscribeStreamEvent{Event: &sttv1.TranscribeStreamEvent_Acknowledgement{Acknowledgement: &sttv1.StreamAcknowledgement{
		ReceivedSequence: value.ReceivedSequence, ProcessedSequence: value.ProcessedSequence,
		ReceivedEndSample: value.ReceivedEndSample, ProcessedEndSample: value.ProcessedEndSample,
		DeliveryClass: sttv1.StreamDeliveryClass_STREAM_DELIVERY_CLASS_DURABLE,
	}}}
}

func protoSessionStatus(value *sttchain.SessionStatusEvent) *sttv1.TranscribeStreamEvent {
	if value == nil {
		return nil
	}
	return &sttv1.TranscribeStreamEvent{Event: &sttv1.TranscribeStreamEvent_SessionStatus{SessionStatus: &sttv1.StreamSessionStatus{
		SessionId: value.SessionID, Generation: value.Generation, State: value.State,
		QueuePosition: value.QueuePosition, CapabilityOutcome: value.CapabilityOutcome, RecoveryGuidance: value.RecoveryGuidance,
		ProviderId: capabilities.CanonicalProviderID(value.ProviderID), ModelId: value.ModelID, ProviderTier: protomap.ProviderTierToProto(string(value.ProviderTier)),
	}}}
}
