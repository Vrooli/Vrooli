package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
)

var errAgentIdentityInvalid = errors.New("agent identity token is invalid")

type agentIdentity struct {
	PrincipalID string
	RunID       string
	WorkspaceID string
}

type agentIdentityVerifier struct {
	baseURL string
	client  *http.Client
}

type agentIdentityVerifyResponse struct {
	Valid  bool `json:"valid"`
	Claims *struct {
		RunID       string            `json:"run_id"`
		WorkspaceID string            `json:"workspace_id"`
		Meta        map[string]string `json:"meta"`
	} `json:"claims"`
}

func newAgentIdentityVerifierFromEnv() *agentIdentityVerifier {
	return &agentIdentityVerifier{
		baseURL: strings.TrimRight(strings.TrimSpace(os.Getenv("SECRETS_MANAGER_AGENT_MANAGER_URL")), "/"),
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (v *agentIdentityVerifier) Verify(ctx context.Context, token string) (agentIdentity, error) {
	return v.verify(ctx, token, "")
}

func (v *agentIdentityVerifier) VerifyForWorkspace(ctx context.Context, token, workspace string) (agentIdentity, error) {
	return v.verify(ctx, token, strings.TrimSpace(workspace))
}

func (v *agentIdentityVerifier) verify(ctx context.Context, token, expectedWorkspace string) (agentIdentity, error) {
	if v == nil || strings.TrimSpace(token) == "" {
		return agentIdentity{}, errAgentIdentityInvalid
	}
	baseURL := v.baseURL
	if baseURL == "" {
		port := strings.TrimSpace(cliutil.DetectPortFromVrooli("agent-manager", "API_PORT")())
		if port == "" {
			return agentIdentity{}, fmt.Errorf("%w: agent-manager API port is unavailable", errAgentIdentityInvalid)
		}
		baseURL = "http://127.0.0.1:" + port
	}
	body, err := json.Marshal(map[string]string{"token": token})
	if err != nil {
		return agentIdentity{}, errAgentIdentityInvalid
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/identity/verify", bytes.NewReader(body))
	if err != nil {
		return agentIdentity{}, errAgentIdentityInvalid
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := v.client.Do(req)
	if err != nil {
		return agentIdentity{}, fmt.Errorf("%w: agent-manager verification unavailable", errAgentIdentityInvalid)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return agentIdentity{}, errAgentIdentityInvalid
	}
	var result agentIdentityVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || !result.Valid || result.Claims == nil {
		return agentIdentity{}, errAgentIdentityInvalid
	}
	runID, err := uuid.Parse(strings.TrimSpace(result.Claims.RunID))
	if err != nil || runID == uuid.Nil {
		return agentIdentity{}, errAgentIdentityInvalid
	}
	workspace := strings.TrimSpace(result.Claims.WorkspaceID)
	if workspace == "" {
		workspace = strings.TrimSpace(result.Claims.Meta["workspace_id"])
	}
	if expectedWorkspace != "" && workspace == "" {
		return agentIdentity{}, errAgentIdentityInvalid
	}
	if expectedWorkspace != "" && workspace != expectedWorkspace {
		return agentIdentity{}, errAgentIdentityInvalid
	}
	return agentIdentity{PrincipalID: "agent-run:" + runID.String(), RunID: runID.String(), WorkspaceID: workspace}, nil
}
