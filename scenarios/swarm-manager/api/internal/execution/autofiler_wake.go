package execution

// SetAutoFilerWaker wires the governed backlog auto-filer wake channel. Tests
// can leave it nil; feature-pending targeting then relies on the periodic sweep.
func (s *Service) SetAutoFilerWaker(w AutoFilerWaker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.autoFilerWaker = w
}

func (s *Service) wakeAutoFiler() {
	s.mu.Lock()
	waker := s.autoFilerWaker
	s.mu.Unlock()
	if waker != nil {
		waker.WakeAutoFiler()
	}
}
