package cliutil

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// Environment variable names for agent identity detection.
const (
	// HeaderCaller identifies the bounded caller attribution used by scenario
	// APIs. Keep this shared because producers and consumers span modules.
	HeaderCaller     = "X-Vrooli-Caller"
	EnvIdentityToken = "VROOLI_AGENT_IDENTITY_TOKEN"

	// These harness signals identify an execution channel only. They are
	// observations and must never be used as agent identity proof.
	EnvClaudeCodeSessionID = "CLAUDE_CODE_SESSION_ID"
	EnvCodexThreadID       = "CODEX_THREAD_ID"

	// HeaderAgentIdentityToken carries the opaque Agent Manager token. APIs must
	// verify it server-side before treating any request as agent-attributed.
	HeaderAgentIdentityToken = "X-Agent-Identity-Token"
	// HeaderInvocationScenario, HeaderInvocationCommand, and
	// HeaderInvocationID are channel observations. They identify what the CLI
	// says it invoked; unlike HeaderAgentIdentityToken, they are not proof that
	// a particular binary performed the mutation.
	HeaderInvocationScenario = "X-Vrooli-Invocation-Scenario"
	HeaderInvocationCommand  = "X-Vrooli-Invocation-Command"
	HeaderInvocationID       = "X-Vrooli-Invocation-Id"
	HeaderHarnessSessionID   = "X-Vrooli-Harness-Session-Id"
	HeaderHarnessKind        = "X-Vrooli-Harness-Kind"
)

var detectAgentManagerPort = DetectPortFromVrooli("agent-manager", "API_PORT")

// IdentityEnv holds the raw agent identity token extracted from the environment.
// A zero-value IdentityEnv (empty Token) means no identity token is present.
type IdentityEnv struct {
	Token string
}

// VerifiedClaims mirrors the identity.Claims struct from agent-manager.
// Fields are strings (not UUIDs) to avoid external dependencies in cli-core.
type VerifiedClaims struct {
	RunID      string            `json:"run_id"`
	TaskID     string            `json:"task_id"`
	Subject    string            `json:"subject"`
	Scopes     []string          `json:"scopes"`
	ProfileKey string            `json:"profile_key"`
	ScopePath  string            `json:"scope_path"`
	IssuedAt   int64             `json:"iat"`
	ExpiresAt  int64             `json:"exp"`
	Meta       map[string]string `json:"meta"`
}

// VerifyResult mirrors the IdentityVerifyResult struct from agent-manager.
type VerifyResult struct {
	Valid     bool            `json:"valid"`
	Claims    *VerifiedClaims `json:"claims,omitempty"`
	RunStatus string          `json:"run_status,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// DetectIdentity reads the agent identity token from the environment.
// Returns a zero-value IdentityEnv if the token is not set.
func DetectIdentity() IdentityEnv {
	return IdentityEnv{
		Token: os.Getenv(EnvIdentityToken),
	}
}

// IsIdentityPresent returns true if the identity token is set.
func (env IdentityEnv) IsIdentityPresent() bool {
	return env.Token != ""
}

// InvocationHeaders returns the common agent-provenance transport headers for
// one CLI command invocation. The identity token is read when a request is
// sent so a caller that establishes an agent environment after startup still
// gets correct forwarding. Invocation fields are deliberately bounded channel
// observations, not cryptographic attestations.
func InvocationHeaders(scenario, command string) func() map[string]string {
	scenario = strings.TrimSpace(scenario)
	command = strings.TrimSpace(command)
	invocationID := newInvocationID()
	return func() map[string]string {
		headers := map[string]string{
			HeaderInvocationScenario: scenario,
			HeaderInvocationCommand:  command,
			HeaderInvocationID:       invocationID,
		}
		if token := strings.TrimSpace(os.Getenv(EnvIdentityToken)); token != "" {
			headers[HeaderAgentIdentityToken] = token
		}
		if sessionID := strings.TrimSpace(os.Getenv(EnvClaudeCodeSessionID)); sessionID != "" {
			headers[HeaderHarnessSessionID] = sessionID
			headers[HeaderHarnessKind] = "claude-code"
		} else if threadID := strings.TrimSpace(os.Getenv(EnvCodexThreadID)); threadID != "" {
			headers[HeaderHarnessSessionID] = threadID
			headers[HeaderHarnessKind] = "codex"
		}
		return headers
	}
}

func newInvocationID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return fmt.Sprintf("cli-%x", raw[:])
	}
	// The random source should never fail on supported platforms. Keep request
	// transport available if it does, while retaining process-local uniqueness.
	return fmt.Sprintf("cli-%d", time.Now().UTC().UnixNano())
}

// VerifyIdentity calls the agent-manager's identity verification endpoint
// to validate the token and retrieve claims.
//
// Error semantics:
//   - Transport errors (network down, missing config) return a Go error.
//   - Auth errors (invalid/expired token) return a *VerifyResult with Valid=false.
func (env IdentityEnv) VerifyIdentity() (*VerifyResult, error) {
	if env.Token == "" {
		return nil, fmt.Errorf("no identity token present (env %s is empty)", EnvIdentityToken)
	}

	baseURL := DetermineAPIBase(APIBaseOptions{
		EnvVars: []string{
			"AGENT_MANAGER_API_BASE",
			"AGENT_MANAGER_API_URL",
		},
		PortDetector: detectAgentManagerPort,
	})
	if baseURL == "" {
		return nil, fmt.Errorf("agent-manager base URL not discoverable (run `vrooli scenario status agent-manager` or set --api-base/config for the calling CLI)")
	}

	client := NewHTTPClient(HTTPClientOptions{
		BaseOptions: APIBaseOptions{
			Override: baseURL,
		},
		Timeout: 10 * time.Second,
	})

	body := map[string]string{"token": env.Token}
	data, err := client.Do("POST", "/api/v1/identity/verify", nil, body)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 401 {
			var result VerifyResult
			if jsonErr := json.Unmarshal(apiErr.RawResponse, &result); jsonErr == nil {
				return &result, nil
			}
			return &VerifyResult{Valid: false, Error: apiErr.Message}, nil
		}
		return nil, fmt.Errorf("identity verification request failed: %w", err)
	}

	var result VerifyResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse verification response: %w", err)
	}
	return &result, nil
}
