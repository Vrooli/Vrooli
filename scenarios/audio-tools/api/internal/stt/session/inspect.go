package session

// Snapshot returns a metadata-only view of one live ledger. It is intended
// for qualification and diagnostics; audio bytes and transcript payloads are
// never exposed by this seam.
func (r *Registry) Snapshot(sessionID string) (Snapshot, bool) {
	if r == nil || sessionID == "" {
		return Snapshot{}, false
	}
	r.mu.Lock()
	ledger, ok := r.sessions[sessionID]
	r.mu.Unlock()
	if !ok || ledger == nil {
		return Snapshot{}, false
	}
	return ledger.Snapshot(), true
}
