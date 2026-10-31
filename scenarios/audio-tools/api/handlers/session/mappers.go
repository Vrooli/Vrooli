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
		if ev.TranscriptDelta != nil {
			out.Payload = &sessv1.SessionEvent_TranscriptDelta{TranscriptDelta: &sessv1.TranscriptDelta{
				Text: ev.TranscriptDelta.Text, FromSeconds: ev.TranscriptDelta.FromSeconds, ToSeconds: ev.TranscriptDelta.ToSeconds,
			}}
		}
	case intsession.EventTranscriptFinal:
		if ev.TranscriptFinal != nil {
			out.Payload = &sessv1.SessionEvent_TranscriptFinal{TranscriptFinal: &sessv1.TranscriptFinal{
				Text: ev.TranscriptFinal.Text, DurationSeconds: ev.TranscriptFinal.DurationSeconds, SpeakerVerified: ev.TranscriptFinal.SpeakerVerified,
			}}
		}
	case intsession.EventAssistantDelta:
		if ev.AssistantDelta != nil {
			out.Payload = &sessv1.SessionEvent_AssistantDelta{AssistantDelta: &sessv1.AssistantDelta{Text: ev.AssistantDelta.Text}}
		}
	case intsession.EventAssistantFinal:
		if ev.AssistantFinal != nil {
			out.Payload = &sessv1.SessionEvent_AssistantFinal{AssistantFinal: &sessv1.AssistantFinal{Text: ev.AssistantFinal.Text, HadAudio: ev.AssistantFinal.HadAudio}}
		}
	case intsession.EventVAD:
		if ev.VAD != nil {
			out.Payload = &sessv1.SessionEvent_Vad{Vad: &sessv1.VadEvent{State: protomap.VadStateToProto(string(ev.VAD.State))}}
		}
	case intsession.EventTool:
		if ev.Tool != nil {
			out.Payload = &sessv1.SessionEvent_Tool{Tool: &sessv1.ToolEvent{Name: ev.Tool.Name, PayloadJson: ev.Tool.PayloadJSON}}
		}
	case intsession.EventBargeInCancel:
		if ev.BargeInCancel != nil {
			out.Payload = &sessv1.SessionEvent_BargeInCancel{BargeInCancel: &sessv1.BargeInCancel{Reason: protomap.BargeInReasonToProto(string(ev.BargeInCancel.Reason)), CanceledEventId: ev.BargeInCancel.CanceledEventID}}
		}
	case intsession.EventClosed:
		if ev.Closed != nil {
			out.Payload = &sessv1.SessionEvent_Closed{Closed: &sessv1.SessionClosed{Reason: ev.Closed.Reason}}
		}
	}
	return out
}
