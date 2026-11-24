package session

// Snapshot returns a metadata-only view of one live ledger. It is intended
// for qualification and diagnostics; audio bytes and transcript payloads are
// never exposed by this seam.
func (r *Registry) Snapshot(sessionID string) (MetadataSnapshot, bool) {
	if r == nil || sessionID == "" {
		return MetadataSnapshot{}, false
	}
	r.mu.Lock()
	ledger, ok := r.sessions[sessionID]
	r.mu.Unlock()
	if !ok || ledger == nil {
		return MetadataSnapshot{}, false
	}
	return ledger.MetadataSnapshot(), true
}

// ConformanceSnapshot returns full transcript state to the trusted internal
// qualification driver. Diagnostics and product surfaces must use Snapshot.
func (r *Registry) ConformanceSnapshot(sessionID string) (Snapshot, bool) {
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
