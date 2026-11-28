package programs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

// AuthoringFixtureBridge intercepts binding calls only for registered authoring
// sessions. Ordinary runtime sessions continue through the governed bridge.
type AuthoringFixtureBridge struct {
	mu       sync.Mutex
	sessions map[string]*fixtureSession
}

type fixtureSession struct {
	replies    map[string][]FixtureReply
	calls      map[string]int32
	requests   []FixtureCallTrace
	violations []string
}

func NewAuthoringFixtureBridge() *AuthoringFixtureBridge {
	return &AuthoringFixtureBridge{sessions: map[string]*fixtureSession{}}
}

func (b *AuthoringFixtureBridge) Register(sessionID string, fixture *AuthoringFixture) {
	if fixture == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	replies := make(map[string][]FixtureReply, len(fixture.Bindings))
	for id, queue := range fixture.Bindings {
		replies[id] = append([]FixtureReply(nil), queue...)
	}
	b.sessions[sessionID] = &fixtureSession{replies: replies, calls: map[string]int32{}}
}

func (b *AuthoringFixtureBridge) Remove(sessionID string) {
	b.mu.Lock()
	delete(b.sessions, sessionID)
	b.mu.Unlock()
}

func (b *AuthoringFixtureBridge) Calls(sessionID string) map[string]int32 {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := map[string]int32{}
	if session := b.sessions[sessionID]; session != nil {
		for id, count := range session.calls {
			result[id] = count
		}
	}
	return result
}

func (b *AuthoringFixtureBridge) Requests(sessionID string) []FixtureCallTrace {
	b.mu.Lock()
	defer b.mu.Unlock()
	if session := b.sessions[sessionID]; session != nil {
		return append([]FixtureCallTrace(nil), session.requests...)
	}
	return nil
}

func (b *AuthoringFixtureBridge) Violations(sessionID string) []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if session := b.sessions[sessionID]; session != nil {
		return append([]string(nil), session.violations...)
	}
	return nil
}

func (b *AuthoringFixtureBridge) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "invalid fixture bridge request", http.StatusBadRequest)
			return
		}
		var request struct {
			SessionID string         `json:"session_id"`
			BindingID string         `json:"binding_id"`
			Args      map[string]any `json:"args"`
			Rows      string         `json:"rows"`
		}
		if json.Unmarshal(raw, &request) != nil {
			r.Body = io.NopCloser(bytes.NewReader(raw))
			next.ServeHTTP(w, r)
			return
		}
		b.mu.Lock()
		session := b.sessions[request.SessionID]
		if session == nil {
			b.mu.Unlock()
			r.Body = io.NopCloser(bytes.NewReader(raw))
			next.ServeHTTP(w, r)
			return
		}
		session.calls[request.BindingID]++
		session.requests = append(session.requests, FixtureCallTrace{BindingID: request.BindingID, Args: request.Args, Rows: request.Rows})
		queue, listed := session.replies[request.BindingID]
		if !listed || len(queue) == 0 {
			session.violations = append(session.violations, fmt.Sprintf("%s: no fixture response", request.BindingID))
			b.mu.Unlock()
			writeFixtureError(w, http.StatusBadRequest, request.BindingID, "authoring fixture has no response for this binding call")
			return
		}
		reply := queue[0]
		session.replies[request.BindingID] = queue[1:]
		b.mu.Unlock()
		if reply.Status < 200 || reply.Status > 599 || reply.Body == nil {
			b.mu.Lock()
			if session := b.sessions[request.SessionID]; session != nil {
				session.violations = append(session.violations, fmt.Sprintf("%s: malformed fixture response", request.BindingID))
			}
			b.mu.Unlock()
			writeFixtureError(w, http.StatusBadRequest, request.BindingID, "authoring fixture response is malformed")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(reply.Status)
		_ = json.NewEncoder(w).Encode(reply.Body)
	})
}

func writeFixtureError(w http.ResponseWriter, status int, bindingID, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": fmt.Sprintf("%s: %s", bindingID, detail), "class": "binding_error", "status": "failed", "binding_id": bindingID, "http_status": status})
}
