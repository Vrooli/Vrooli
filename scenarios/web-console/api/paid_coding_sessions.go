package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

var errPaidCodingSessionUnavailable = errors.New("paid coding session is unavailable")

type paidCodingBinding struct {
	SessionID         string
	BusinessAccountID string
}

// paidCodingSessionRegistry keeps the OpenCode-to-Web-Console bearer mapping
// in memory. The bearer is generated before PTY environment creation, is only
// exposed to that PTY through OPENCODE_CONFIG_CONTENT, and is never persisted
// or returned to the browser.
type paidCodingSessionRegistry struct {
	mu             sync.RWMutex
	byTokenHash    map[string]paidCodingBinding
	bySession      map[string]string
	tokenBySession map[string]string
}

func newPaidCodingSessionRegistry() *paidCodingSessionRegistry {
	return &paidCodingSessionRegistry{byTokenHash: map[string]paidCodingBinding{}, bySession: map[string]string{}, tokenBySession: map[string]string{}}
}

func (r *paidCodingSessionRegistry) Bind(sessionID, accountID string) (string, error) {
	token, err := newPaidCodingToken()
	if err != nil {
		return "", err
	}
	r.Register(sessionID, accountID, token)
	return token, nil
}

func newPaidCodingToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("create paid coding session credential: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func (r *paidCodingSessionRegistry) Register(sessionID, accountID, token string) {
	sessionID = strings.TrimSpace(sessionID)
	accountID = strings.TrimSpace(accountID)
	token = strings.TrimSpace(token)
	if sessionID == "" || accountID == "" || token == "" {
		return
	}
	hash := paidCodingTokenHash(token)
	r.mu.Lock()
	defer r.mu.Unlock()
	if previous := r.bySession[sessionID]; previous != "" {
		delete(r.byTokenHash, previous)
	}
	r.byTokenHash[hash] = paidCodingBinding{SessionID: sessionID, BusinessAccountID: accountID}
	r.bySession[sessionID] = hash
	r.tokenBySession[sessionID] = token
}

func (r *paidCodingSessionRegistry) Lookup(token string) (paidCodingBinding, bool) {
	if strings.TrimSpace(token) == "" {
		return paidCodingBinding{}, false
	}
	r.mu.RLock()
	binding, ok := r.byTokenHash[paidCodingTokenHash(token)]
	r.mu.RUnlock()
	return binding, ok
}

func (r *paidCodingSessionRegistry) RemoveSession(sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if hash := r.bySession[sessionID]; hash != "" {
		delete(r.byTokenHash, hash)
		delete(r.bySession, sessionID)
		delete(r.tokenBySession, sessionID)
	}
}

func (r *paidCodingSessionRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byTokenHash = map[string]paidCodingBinding{}
	r.bySession = map[string]string{}
	r.tokenBySession = map[string]string{}
}

func paidCodingTokenHash(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func (r *paidCodingSessionRegistry) OpenCodeEnvironment(sessionID string) (map[string]string, error) {
	r.mu.RLock()
	hash := r.bySession[sessionID]
	binding := r.byTokenHash[hash]
	token := r.tokenBySession[sessionID]
	r.mu.RUnlock()
	if hash == "" || binding.SessionID == "" || token == "" {
		return nil, nil
	}
	baseURL, err := paidCodingBaseURL(os.Getenv("API_PORT"))
	if err != nil {
		return nil, err
	}
	config, err := openCodeConfig(baseURL, token)
	if err != nil {
		return nil, err
	}
	return map[string]string{"OPENCODE_CONFIG_CONTENT": config}, nil
}

var activePaidCodingSessions = newPaidCodingSessionRegistry()

// paidCodingBaseURL uses a numeric loopback port so no untrusted host can be
// injected into the provider configuration passed to a PTY.
func paidCodingBaseURL(port string) (string, error) {
	value, err := strconv.Atoi(strings.TrimSpace(port))
	if err != nil || value < 1 || value > 65535 {
		return "", errors.New("Web Console API port is unavailable")
	}
	return fmt.Sprintf("http://%s/api/v1/paid-coding/v1", net.JoinHostPort("127.0.0.1", strconv.Itoa(value))), nil
}

func openCodeConfig(baseURL, token string) (string, error) {
	if strings.TrimSpace(baseURL) == "" || strings.TrimSpace(token) == "" {
		return "", errPaidCodingSessionUnavailable
	}
	config := map[string]any{
		"model": "vrooli/coding",
		"provider": map[string]any{
			"vrooli": map[string]any{
				"npm":     "@ai-sdk/openai-compatible",
				"name":    "Vrooli Coding",
				"options": map[string]any{"baseURL": baseURL, "apiKey": token},
				"models": map[string]any{"coding": map[string]any{
					"name":  "Vrooli Coding",
					"limit": map[string]int{"context": 128000, "output": 16384},
				}},
			},
		},
	}
	encoded, err := json.Marshal(config)
	return string(encoded), err
}
