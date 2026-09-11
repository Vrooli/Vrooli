package main

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"web-console/backends"
	"web-console/internal/backend"
	"web-console/internal/sessionstore"
	"web-console/session"
)

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#session-activity-contract

// sessionActivityPayload is the payload of a HubKindSessionActivity envelope.
// snake_case, like every hub payload.
type sessionActivityPayload struct {
	State          string                `json:"state"`
	Source         string                `json:"source"`
	Confidence     float32               `json:"confidence"`
	Since          string                `json:"since,omitempty"`
	LastOutputAt   string                `json:"last_output_at,omitempty"`
	Prompt         *pendingPromptPayload `json:"prompt,omitempty"`
	Harness        string                `json:"harness,omitempty"`
	HarnessVersion string                `json:"harness_version,omitempty"`
}

type pendingPromptPayload struct {
	Kind         string                `json:"kind"`
	Text         string                `json:"text"`
	Options      []promptOptionPayload `json:"options"`
	Answerable   bool                  `json:"answerable"`
	FreeTextHint string                `json:"free_text_hint,omitempty"`
	Hash         string                `json:"hash,omitempty"`
	Cancellable  bool                  `json:"cancellable,omitempty"`
}

type promptOptionPayload struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Selected bool   `json:"selected"`
}

func newSessionActivityPayload(a session.Activity) sessionActivityPayload {
	payload := sessionActivityPayload{
		State:          string(a.State),
		Source:         string(a.Source),
		Confidence:     a.Confidence,
		Since:          formatActivityTime(a.Since),
		LastOutputAt:   formatActivityTime(a.LastOutputAt),
		Harness:        a.Harness,
		HarnessVersion: a.HarnessVersion,
	}
	if a.Prompt != nil {
		prompt := &pendingPromptPayload{
			Kind: a.Prompt.Kind, Text: a.Prompt.Text, FreeTextHint: a.Prompt.FreeTextHint, Options: []promptOptionPayload{},
			Answerable: a.Answerable, Hash: backend.PromptHash(a.Prompt), Cancellable: a.Prompt.Cancellable,
		}
		for _, option := range a.Prompt.Options {
			prompt.Options = append(prompt.Options, promptOptionPayload{Key: option.Key, Label: option.Label, Selected: option.Selected})
		}
		payload.Prompt = prompt
	}
	return payload
}

func formatActivityTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// newActivityDetector builds a session's activity detector: it reads the
// session's screen with the detector for its agent harness and publishes
// changes on the conversation hub.
func (s *Server) newActivityDetector(sess *session.Session) *session.ActivityDetector {
	id := sess.ID
	return session.NewActivityDetector(session.ActivityConfig{
		SessionID: id,
		Screen:    sess.ActivityScreen,
		Harness: func() (string, backend.PromptDetector) {
			agent := s.agentTypes.lookup(id)
			return agent, backends.PromptDetectorFor(agent)
		},
		Publish: func(a session.Activity) {
			s.hub.Publish(HubEnvelope{SessionID: id, Kind: HubKindSessionActivity, Payload: newSessionActivityPayload(a)})
		},
		Answering: s.promptAnswering.decide,
	})
}

// activityDetector returns a live session's activity detector, or nil.
func (s *Server) activityDetector(sessionID string) *session.ActivityDetector {
	if sessionID == "" || s.sessions == nil {
		return nil
	}
	sess, ok := s.sessions.Get(sessionID)
	if !ok || sess == nil {
		return nil
	}
	return sess.ActivityDetector()
}

// launchHarnesses are the agent executables a launch command can name.
var launchHarnesses = map[string]sessionstore.Agent{
	"claude":   sessionstore.AgentClaude,
	"codex":    sessionstore.AgentCodex,
	"opencode": sessionstore.AgentOpenCode,
	"grok":     sessionstore.AgentGrok,
}

// agentFromLaunchCommand names the harness a launch command runs: the first
// word whose executable name is a known agent ("/…/shims/claude --model x" is
// claude). Empty when none is named.
func agentFromLaunchCommand(command string) string {
	for _, word := range strings.Fields(command) {
		word = strings.Trim(word, `'"`)
		if agent, ok := launchHarnesses[filepath.Base(word)]; ok {
			return string(agent)
		}
	}
	return ""
}

// agentTypeCacheTTL bounds how stale a session's cached agent type can be; a
// session's agent becomes known at launch or on its first hook.
const agentTypeCacheTTL = 10 * time.Second

// agentTypeCache answers "which harness runs in this session" for detector
// selection without a store read per screen evaluation.
type agentTypeCache struct {
	server *Server
	mu     sync.Mutex
	byID   map[string]agentTypeEntry
}

type agentTypeEntry struct {
	agent string
	at    time.Time
}

func (c *agentTypeCache) lookup(sessionID string) string {
	c.mu.Lock()
	if entry, ok := c.byID[sessionID]; ok && time.Since(entry.at) < agentTypeCacheTTL {
		c.mu.Unlock()
		return entry.agent
	}
	c.mu.Unlock()
	agent := ""
	if c.server != nil && c.server.sessionStore != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		meta, err := c.server.sessionStore.Get(ctx, sessionID)
		cancel()
		if err != nil {
			// Not cached: during the startup storm a read can time out, and
			// caching "no harness" left static screens (a waiting prompt)
			// unread until the next frame after the TTL. Retry next time.
			return ""
		}
		agent = string(meta.AgentType)
		if agent == "" || agent == string(sessionstore.AgentNone) {
			// Panes launched by other scenarios (agent-manager) record no
			// agent type; their launch command still names the harness.
			agent = agentFromLaunchCommand(meta.LaunchCommand)
		}
	}
	c.mu.Lock()
	if c.byID == nil {
		c.byID = make(map[string]agentTypeEntry)
	}
	c.byID[sessionID] = agentTypeEntry{agent: agent, at: time.Now()}
	c.mu.Unlock()
	return agent
}
