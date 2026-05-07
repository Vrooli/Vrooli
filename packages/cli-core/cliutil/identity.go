package cliutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// Environment variable names for agent identity detection.
const (
	EnvIdentityToken = "VROOLI_AGENT_IDENTITY_TOKEN"
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
		PortEnvVars:  []string{"AGENT_MANAGER_API_PORT"},
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
