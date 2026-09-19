// Package session implements the voice-session abstraction: a duplex audio
// interaction with multi-observer pub/sub, barge-in coordination, and a
// transport-pluggable boundary.
//
// Audio in -> transcript events. Text in -> TTS audio out. Multiple observers
// subscribe to the same session and receive transcript deltas, assistant
// deltas, VAD signals, tool events, and barge-in cancels.
//
// Concrete transports plug in below the session boundary; P0 ships
// handlers/stt/stream_ws.go (WebSocket).
package session

import "time"

// EventType is the typed discriminator for SessionEvent.
type EventType string

const (
	EventTranscriptDelta EventType = "transcript_delta"
	EventTranscriptFinal EventType = "transcript_final"
	EventAssistantDelta  EventType = "assistant_delta"
	EventAssistantFinal  EventType = "assistant_final"
	EventVAD             EventType = "vad"
	EventTool            EventType = "tool"
	EventBargeInCancel   EventType = "barge_in_cancel"
	EventClosed          EventType = "closed"
)

// SessionEvent is what observers receive over Session.Subscribe channels.
type SessionEvent struct {
	EventID   string
	SessionID string
	Type      EventType
	EmittedAt time.Time

	// Typed payload — exactly one is set per Type.
	TranscriptDelta *TranscriptDelta
	TranscriptFinal *TranscriptFinal
	AssistantDelta  *AssistantDelta
	AssistantFinal  *AssistantFinal
	VAD             *VADEvent
	Tool            *ToolEvent
	BargeInCancel   *BargeInCancel
	Closed          *SessionClosed
}

type TranscriptDelta struct {
	Text        string
	FromSeconds float64
	ToSeconds   float64
}

type TranscriptFinal struct {
	Text            string
	DurationSeconds float64
	SpeakerVerified bool
}

type AssistantDelta struct {
	Text string
}

type AssistantFinal struct {
	Text     string
	HadAudio bool
}

// VADState is the speech-activity state reported by the transport's VAD.
type VADState string

const (
	VADSpeechStart VADState = "speech_start"
	VADSpeechEnd   VADState = "speech_end"
)

type VADEvent struct {
	State VADState
}

type ToolEvent struct {
	Name        string
	PayloadJSON string
}

// BargeInReason is why an in-flight assistant action was canceled.
type BargeInReason string

const (
	BargeInVAD      BargeInReason = "vad"
	BargeInExplicit BargeInReason = "explicit"
)

type BargeInCancel struct {
	Reason          BargeInReason
	CanceledEventID string
}

type SessionClosed struct {
	Reason string
}
