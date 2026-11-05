package skills

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/discovery"
)

const agentIdentityHeader = "X-Agent-Identity-Token"

// VerifiedWorkflowIdentity intentionally exposes only the claims needed for
// experiment provenance, not an opaque caller-provided source label.
type VerifiedWorkflowIdentity struct {
	RunID string
	Meta  map[string]string
}

type IdentityVerifier interface {
	VerifyWorkflowIdentity(context.Context, string) (*VerifiedWorkflowIdentity, error)
}

type agentManagerIdentityVerifier struct{ client *http.Client }

func NewAgentManagerIdentityVerifier(client *http.Client) IdentityVerifier {
	if client == nil {
		client = http.DefaultClient
	}
	return &agentManagerIdentityVerifier{client: client}
}

func (v *agentManagerIdentityVerifier) VerifyWorkflowIdentity(ctx context.Context, token string) (*VerifiedWorkflowIdentity, error) {
	if strings.TrimSpace(token) == "" {
		return nil, nil
	}
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
	if err != nil {
		return nil, fmt.Errorf("resolve agent-manager identity verifier: %w", err)
	}
	body, _ := json.Marshal(map[string]string{"token": token})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/identity/verify", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("identity verifier returned %d", resp.StatusCode)
	}
	var decoded struct {
		Valid  bool `json:"valid"`
		Claims struct {
			RunID string            `json:"run_id"`
			Meta  map[string]string `json:"meta"`
		} `json:"claims"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	if !decoded.Valid || decoded.Claims.RunID == "" {
		return nil, fmt.Errorf("identity verifier returned no valid claims")
	}
	return &VerifiedWorkflowIdentity{RunID: decoded.Claims.RunID, Meta: decoded.Claims.Meta}, nil
}
