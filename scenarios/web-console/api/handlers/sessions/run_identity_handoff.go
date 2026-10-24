package sessions

import "sync"

// sessionRunIdentityHandoff is a process-local, one-use handoff from the
// verified reserved Create request to that exact PTY's environment builder.
// It is intentionally separate from the verified-human delegation holder.
type sessionRunIdentityHandoff struct {
	mu    sync.Mutex
	items map[string]string
}

var activeSessionRunIdentities = &sessionRunIdentityHandoff{items: map[string]string{}}

func putSessionRunIdentity(sessionID, token string) bool {
	if sessionID == "" || token == "" {
		return false
	}
	activeSessionRunIdentities.mu.Lock()
	defer activeSessionRunIdentities.mu.Unlock()
	if _, exists := activeSessionRunIdentities.items[sessionID]; exists {
		return false
	}
	activeSessionRunIdentities.items[sessionID] = token
	return true
}

func deleteSessionRunIdentity(sessionID string) {
	if sessionID == "" {
		return
	}
	activeSessionRunIdentities.mu.Lock()
	defer activeSessionRunIdentities.mu.Unlock()
	delete(activeSessionRunIdentities.items, sessionID)
}

// TakeSessionRunIdentity transfers a reserved child identity into only the
// matching PTY environment. It is exported solely for package-main's existing
// session environment builder; it does not expose list/read access to callers.
func TakeSessionRunIdentity(sessionID string) (string, bool) {
	if sessionID == "" {
		return "", false
	}
	activeSessionRunIdentities.mu.Lock()
	defer activeSessionRunIdentities.mu.Unlock()
	token, ok := activeSessionRunIdentities.items[sessionID]
	delete(activeSessionRunIdentities.items, sessionID)
	return token, ok
}
