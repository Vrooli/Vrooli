package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	intsession "audio-tools/internal/session"
)

// TestToProto_CoversAllEventTypes drives toProto through every event
// branch so the mapper's switch is exercised end-to-end without
// requiring a live session.
func TestToProto_CoversAllEventTypes(t *testing.T) {
	ts := time.Now()
	cases := []struct {
		name  string
		ev    intsession.SessionEvent
		check func(t *testing.T, out interface{})
	}{
		{
			name: "transcript_delta",
			ev: intsession.SessionEvent{
				EventID: "e1", SessionID: "s1", EmittedAt: ts,
				Type:            intsession.EventTranscriptDelta,
				TranscriptDelta: &intsession.TranscriptDelta{Text: "t", FromSeconds: 0.1, ToSeconds: 0.5},
			},
		},
		{
			name: "transcript_final",
			ev: intsession.SessionEvent{
				EventID: "e2", SessionID: "s1", EmittedAt: ts,
				Type:            intsession.EventTranscriptFinal,
				TranscriptFinal: &intsession.TranscriptFinal{Text: "t", DurationSeconds: 1.5, SpeakerVerified: true},
			},
		},
		{
			name: "assistant_delta",
			ev: intsession.SessionEvent{
				EventID: "e3", SessionID: "s1", EmittedAt: ts,
				Type:           intsession.EventAssistantDelta,
				AssistantDelta: &intsession.AssistantDelta{Text: "hi"},
			},
		},
		{
			name: "assistant_final",
			ev: intsession.SessionEvent{
				EventID: "e4", SessionID: "s1", EmittedAt: ts,
				Type:           intsession.EventAssistantFinal,
				AssistantFinal: &intsession.AssistantFinal{Text: "done", HadAudio: true},
			},
		},
		{
			name: "vad",
			ev: intsession.SessionEvent{
				EventID: "e5", SessionID: "s1", EmittedAt: ts,
				Type: intsession.EventVAD,
				VAD:  &intsession.VADEvent{State: intsession.VADSpeechStart},
			},
		},
		{
			name: "tool",
			ev: intsession.SessionEvent{
				EventID: "e6", SessionID: "s1", EmittedAt: ts,
				Type: intsession.EventTool,
				Tool: &intsession.ToolEvent{Name: "search", PayloadJSON: `{"q":"x"}`},
			},
		},
		{
			name: "audio_output",
			ev: intsession.SessionEvent{
				EventID: "e9", SessionID: "s1", EmittedAt: ts,
				Type:        intsession.EventAudioOutput,
				AudioOutput: &intsession.AudioFrame{Bytes: []byte{1, 2}, Format: "pcm_s16le", SampleRate: 24000, Sequence: 7, IsFinal: true},
			},
		},
		{
			name: "provider_error",
			ev: intsession.SessionEvent{
				EventID: "e10", SessionID: "s1", EmittedAt: ts,
				Type: intsession.EventProviderError, ProviderError: &intsession.ProviderError{Message: "provider unavailable"},
			},
		},
		{
			name: "barge_in_cancel",
			ev: intsession.SessionEvent{
				EventID: "e7", SessionID: "s1", EmittedAt: ts,
				Type:          intsession.EventBargeInCancel,
				BargeInCancel: &intsession.BargeInCancel{Reason: intsession.BargeInExplicit, CanceledEventID: "x"},
			},
		},
		{
			name: "closed",
			ev: intsession.SessionEvent{
				EventID: "e8", SessionID: "s1", EmittedAt: ts,
				Type:   intsession.EventClosed,
				Closed: &intsession.SessionClosed{Reason: "client", ResumeFromEventID: "cursor"},
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			out := toProto(tc.ev)
			require.NotNil(t, out)
			require.Equal(t, tc.ev.EventID, out.GetEventId())
			require.Equal(t, tc.ev.SessionID, out.GetSessionId())
			require.NotEmpty(t, out.GetEmittedAt())
			require.NotNil(t, out.GetPayload(), "payload should be set for %s", tc.name)
		})
	}
}

// TestToProto_NilPayloadFieldsAreToleratedAsNilOutput exercises the
// branches where the typed payload pointer is nil — the mapper should
// still return a non-nil envelope.
func TestToProto_NilPayloadFieldsAreToleratedAsNilOutput(t *testing.T) {
	cases := []intsession.EventType{
		intsession.EventTranscriptDelta,
		intsession.EventTranscriptFinal,
		intsession.EventAssistantDelta,
		intsession.EventAssistantFinal,
		intsession.EventVAD,
		intsession.EventTool,
		intsession.EventAudioOutput,
		intsession.EventProviderError,
		intsession.EventBargeInCancel,
		intsession.EventClosed,
	}
	for _, eventType := range cases {
		t.Run(string(eventType), func(t *testing.T) {
			ev := intsession.SessionEvent{EventID: "x", SessionID: "s", EmittedAt: time.Now(), Type: eventType}
			out := toProto(ev)
			require.NotNil(t, out)
			require.Nil(t, out.GetPayload())
		})
	}
}

func TestToProto_MapsDuplexAudioAndProviderErrorPayloads(t *testing.T) {
	audio := toProto(intsession.SessionEvent{
		Type:        intsession.EventAudioOutput,
		AudioOutput: &intsession.AudioFrame{Bytes: []byte{1, 2, 3}, Format: "pcm_s16le", SampleRate: 24000, Sequence: 9, IsFinal: true},
	})
	gotAudio := audio.GetAudioOutput()
	require.NotNil(t, gotAudio)
	require.Equal(t, []byte{1, 2, 3}, gotAudio.GetData())
	require.Equal(t, "pcm_s16le", gotAudio.GetFormat())
	require.EqualValues(t, 24000, gotAudio.GetSampleRate())
	require.EqualValues(t, 9, gotAudio.GetSequence())
	require.True(t, gotAudio.GetIsFinal())
	closed := toProto(intsession.SessionEvent{
		Type:   intsession.EventClosed,
		Closed: &intsession.SessionClosed{Reason: "observer-overflow", ResumeFromEventID: "resume-here"},
	})
	require.Equal(t, "resume-here", closed.GetClosed().GetResumeFromEventId())

	providerErr := toProto(intsession.SessionEvent{
		Type:          intsession.EventProviderError,
		ProviderError: &intsession.ProviderError{Message: "provider unavailable"},
	})
	require.Equal(t, "provider unavailable", providerErr.GetProviderError().GetMessage())
}
