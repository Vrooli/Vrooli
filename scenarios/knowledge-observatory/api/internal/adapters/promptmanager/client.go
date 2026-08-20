// Package promptmanager provides an HTTP client for the prompt-manager service.
package promptmanager

// DOC: docs/concepts/ARCHITECTURE.md#integrations

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	skillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills"
	skillsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills/skills_v1connect"
)

// Client is an HTTP client for prompt-manager.
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new prompt-manager client.
func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
	}
}

// GetSkill loads a skill by ID and returns its content.
func (c *Client) GetSkill(ctx context.Context, id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("skill id is required")
	}
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return "", err
	}
	client := skillsconnect.NewSkillsServiceClient(c.httpClient, baseURL)
	resp, err := client.GetSkill(ctx, connect.NewRequest(&skillsv1.GetSkillRequest{Id: id}))
	if err != nil {
		return "", fmt.Errorf("get prompt-manager skill: %w", err)
	}
	if resp.Msg.GetSkill() == nil {
		return "", fmt.Errorf("prompt-manager returned no skill")
	}
	return resp.Msg.GetSkill().GetContent(), nil
}

func (c *Client) resolveBaseURL(ctx context.Context) (string, error) {
	url, err := discovery.ResolveScenarioURLDefault(ctx, "prompt-manager")
	if err != nil {
		return "", fmt.Errorf("resolve prompt-manager url: %w", err)
	}
	return strings.TrimRight(url, "/"), nil
}
