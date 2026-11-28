package session

import "fmt"

const (
	maxRetainedEvents = 128
	maxRetainedBytes  = 1 << 20
)

// appendHistoryLocked retains an ordered, byte-bounded copy for reconnects.
// When one event exceeds the budget, older history is invalidated so a cursor
// can never silently bridge a missing event.
func (s *Session) appendHistoryLocked(evt SessionEvent) {
	bytes := sessionEventBytes(evt)
	if bytes > maxRetainedBytes {
		s.history = nil
		s.historyBytes = 0
		s.historyFloorID = evt.EventID
		return
	}
	copy := cloneSessionEvent(evt)
	s.history = append(s.history, copy)
	s.historyBytes += bytes
	for len(s.history) > maxRetainedEvents || s.historyBytes > maxRetainedBytes {
		s.historyFloorID = s.history[0].EventID
		s.historyBytes -= sessionEventBytes(s.history[0])
		s.history = s.history[1:]
	}
}

// replayAfterLocked returns cloned events after cursor. An empty cursor means
// subscribe only to events emitted after registration.
func (s *Session) replayAfterLocked(cursor string) ([]SessionEvent, error) {
	if cursor == "" {
		return nil, nil
	}
	if cursor == s.historyFloorID {
		return cloneSessionEvents(s.history), nil
	}
	for i := range s.history {
		if s.history[i].EventID == cursor {
			return cloneSessionEvents(s.history[i+1:]), nil
		}
	}
	return nil, fmt.Errorf("%w: %q", ErrReplayCursorUnavailable, cursor)
}

func cloneSessionEvents(events []SessionEvent) []SessionEvent {
	cloned := make([]SessionEvent, len(events))
	for i := range events {
		cloned[i] = cloneSessionEvent(events[i])
	}
	return cloned
}

func cloneSessionEvent(evt SessionEvent) SessionEvent {
	if evt.TranscriptDelta != nil {
		v := *evt.TranscriptDelta
		evt.TranscriptDelta = &v
	}
	if evt.TranscriptFinal != nil {
		v := *evt.TranscriptFinal
		evt.TranscriptFinal = &v
	}
	if evt.AssistantDelta != nil {
		v := *evt.AssistantDelta
		evt.AssistantDelta = &v
	}
	if evt.AssistantFinal != nil {
		v := *evt.AssistantFinal
		evt.AssistantFinal = &v
	}
	if evt.VAD != nil {
		v := *evt.VAD
		evt.VAD = &v
	}
	if evt.Tool != nil {
		v := *evt.Tool
		evt.Tool = &v
	}
	if evt.AudioOutput != nil {
		v := *evt.AudioOutput
		v.Bytes = append([]byte(nil), v.Bytes...)
		evt.AudioOutput = &v
	}
	if evt.ProviderError != nil {
		v := *evt.ProviderError
		evt.ProviderError = &v
	}
	if evt.BargeInCancel != nil {
		v := *evt.BargeInCancel
		evt.BargeInCancel = &v
	}
	if evt.Closed != nil {
		v := *evt.Closed
		evt.Closed = &v
	}
	return evt
}

func sessionEventBytes(evt SessionEvent) int {
	n := 96 + len(evt.EventID) + len(evt.SessionID)
	if v := evt.TranscriptDelta; v != nil {
		n += len(v.Text) + 16
	}
	if v := evt.TranscriptFinal; v != nil {
		n += len(v.Text) + 16
	}
	if v := evt.AssistantDelta; v != nil {
		n += len(v.Text)
	}
	if v := evt.AssistantFinal; v != nil {
		n += len(v.Text)
	}
	if v := evt.Tool; v != nil {
		n += len(v.Name) + len(v.PayloadJSON)
	}
	if v := evt.AudioOutput; v != nil {
		n += len(v.Bytes) + len(v.Format) + 16
	}
	if v := evt.ProviderError; v != nil {
		n += len(v.Message)
	}
	if v := evt.BargeInCancel; v != nil {
		n += len(v.CanceledEventID) + len(v.Reason)
	}
	if v := evt.Closed; v != nil {
		n += len(v.Reason)
	}
	return n
}
