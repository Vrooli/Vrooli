package control

import (
	"reflect"
)

func observedStateKey(deviceID, transport, attribute string) string {
	return deviceID + "\x00" + transport + "\x00" + attribute
}

func (s *Service) rememberObservedState(deviceID, transport, attribute string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.observedStates == nil {
		s.observedStates = map[string]any{}
	}
	s.observedStates[observedStateKey(deviceID, transport, attribute)] = value
}

func (s *Service) observeState(deviceID, transport, attribute string, value any) (old any, changed, hadPrevious bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.observedStates == nil {
		s.observedStates = map[string]any{}
	}
	key := observedStateKey(deviceID, transport, attribute)
	old, hadPrevious = s.observedStates[key]
	changed = hadPrevious && !reflect.DeepEqual(old, value)
	s.observedStates[key] = value
	return old, changed, hadPrevious
}
