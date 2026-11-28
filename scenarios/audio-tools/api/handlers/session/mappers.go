package session

import (
	"audio-tools/internal/protomap"
	intsession "audio-tools/internal/session"

	sessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/session"
)

func toProto(ev intsession.SessionEvent) *sessv1.SessionEvent {
	out := &sessv1.SessionEvent{
		EventId:   ev.EventID,
		SessionId: ev.SessionID,
		EmittedAt: protomap.TimeToProto(ev.EmittedAt),
	}
	switch ev.Type {
	case intsession.EventTranscriptDelta:
		out.Payload = transcriptDeltaToProto(ev.TranscriptDelta)
	case intsession.EventTranscriptFinal:
		out.Payload = transcriptFinalToProto(ev.TranscriptFinal)
	case intsession.EventAssistantDelta:
		out.Payload = assistantDeltaToProto(ev.AssistantDelta)
	case intsession.EventAssistantFinal:
		out.Payload = assistantFinalToProto(ev.AssistantFinal)
	case intsession.EventVAD:
		out.Payload = vadToProto(ev.VAD)
	case intsession.EventTool:
		out.Payload = toolToProto(ev.Tool)
	case intsession.EventAudioOutput:
		out.Payload = audioOutputToProto(ev.AudioOutput)
	case intsession.EventProviderError:
		out.Payload = providerErrorToProto(ev.ProviderError)
	case intsession.EventBargeInCancel:
		out.Payload = bargeInCancelToProto(ev.BargeInCancel)
	case intsession.EventClosed:
		out.Payload = closedToProto(ev.Closed)
	}
	return out
}

func transcriptDeltaToProto(delta *intsession.TranscriptDelta) *sessv1.SessionEvent_TranscriptDelta {
	if delta == nil {
		return nil
	}
	return &sessv1.SessionEvent_TranscriptDelta{TranscriptDelta: &sessv1.TranscriptDelta{
		Text: delta.Text, FromSeconds: delta.FromSeconds, ToSeconds: delta.ToSeconds,
	}}
}

func transcriptFinalToProto(final *intsession.TranscriptFinal) *sessv1.SessionEvent_TranscriptFinal {
	if final == nil {
		return nil
	}
	return &sessv1.SessionEvent_TranscriptFinal{TranscriptFinal: &sessv1.TranscriptFinal{
		Text: final.Text, DurationSeconds: final.DurationSeconds, SpeakerVerified: final.SpeakerVerified,
	}}
}

func assistantDeltaToProto(delta *intsession.AssistantDelta) *sessv1.SessionEvent_AssistantDelta {
	if delta == nil {
		return nil
	}
	return &sessv1.SessionEvent_AssistantDelta{AssistantDelta: &sessv1.AssistantDelta{Text: delta.Text}}
}

func assistantFinalToProto(final *intsession.AssistantFinal) *sessv1.SessionEvent_AssistantFinal {
	if final == nil {
		return nil
	}
	return &sessv1.SessionEvent_AssistantFinal{AssistantFinal: &sessv1.AssistantFinal{Text: final.Text, HadAudio: final.HadAudio}}
}

func vadToProto(vad *intsession.VADEvent) *sessv1.SessionEvent_Vad {
	if vad == nil {
		return nil
	}
	return &sessv1.SessionEvent_Vad{Vad: &sessv1.VadEvent{State: protomap.VadStateToProto(string(vad.State))}}
}

func toolToProto(tool *intsession.ToolEvent) *sessv1.SessionEvent_Tool {
	if tool == nil {
		return nil
	}
	return &sessv1.SessionEvent_Tool{Tool: &sessv1.ToolEvent{Name: tool.Name, PayloadJson: tool.PayloadJSON}}
}

func audioOutputToProto(frame *intsession.AudioFrame) *sessv1.SessionEvent_AudioOutput {
	if frame == nil {
		return nil
	}
	return &sessv1.SessionEvent_AudioOutput{AudioOutput: &sessv1.SessionAudioFrame{
		Data: frame.Bytes, Format: frame.Format, SampleRate: frame.SampleRate,
		Sequence: frame.Sequence, IsFinal: frame.IsFinal,
	}}
}

func providerErrorToProto(event *intsession.ProviderError) *sessv1.SessionEvent_ProviderError {
	if event == nil {
		return nil
	}
	return &sessv1.SessionEvent_ProviderError{ProviderError: &sessv1.ProviderError{Message: event.Message}}
}

func bargeInCancelToProto(cancel *intsession.BargeInCancel) *sessv1.SessionEvent_BargeInCancel {
	if cancel == nil {
		return nil
	}
	return &sessv1.SessionEvent_BargeInCancel{BargeInCancel: &sessv1.BargeInCancel{
		Reason: protomap.BargeInReasonToProto(string(cancel.Reason)), CanceledEventId: cancel.CanceledEventID,
	}}
}

func closedToProto(closed *intsession.SessionClosed) *sessv1.SessionEvent_Closed {
	if closed == nil {
		return nil
	}
	return &sessv1.SessionEvent_Closed{Closed: &sessv1.SessionClosed{
		Reason: closed.Reason, ResumeFromEventId: closed.ResumeFromEventID,
	}}
}
