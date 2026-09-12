package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// WorkItem is the small disposition surface prompt-manager needs from the
// unified swarm-manager stream. The stream owns review state and rationale;
// prompt-manager only stores the stable reference.
type WorkItem struct {
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	ReviewStatus string `json:"review_status"`
	Rationale    string `json:"rationale,omitempty"`
}

type workPublisher interface {
	CreateWork(context.Context, string, string, string) (string, error)
	GetWork(context.Context, string) (WorkItem, error)
}

type httpWorkPublisher struct {
	base   string
	client *http.Client
}

// NewHTTPWorkPublisherFromEnv creates the production publisher. The control
// plane supplies SWARM_MANAGER_API_BASE when the dependency is available.
func NewHTTPWorkPublisherFromEnv() interface {
	CreateWork(context.Context, string, string, string) (string, error)
	GetWork(context.Context, string) (WorkItem, error)
} {
	return newHTTPWorkPublisher(os.Getenv("SWARM_MANAGER_API_BASE"))
}

func newHTTPWorkPublisher(base string) *httpWorkPublisher {
	return &httpWorkPublisher{base: strings.TrimRight(strings.TrimSpace(base), "/"), client: http.DefaultClient}
}

func (p *httpWorkPublisher) request(ctx context.Context, method, path string, payload any, out any) error {
	var body *strings.Reader
	if payload == nil {
		body = strings.NewReader("")
	} else {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = strings.NewReader(string(data))
	}
	req, err := http.NewRequestWithContext(ctx, method, p.base+path, body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("swarm-manager work request returned HTTP %d", resp.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (p *httpWorkPublisher) CreateWork(ctx context.Context, name, title, description string) (string, error) {
	if p == nil || p.base == "" {
		return "", fmt.Errorf("swarm-manager work publisher is not configured")
	}
	var resp struct {
		Item WorkItem `json:"item"`
	}
	err := p.request(ctx, http.MethodPost, "/api/v1/backlog", map[string]any{
		"kind":        "execute",
		"name":        name,
		"title":       title,
		"description": description,
	}, &resp)
	if err != nil {
		return "", err
	}
	if resp.Item.Name == "" {
		return "", fmt.Errorf("swarm-manager returned no work item reference")
	}
	return resp.Item.Kind + "/" + resp.Item.Name, nil
}

func (p *httpWorkPublisher) GetWork(ctx context.Context, ref string) (WorkItem, error) {
	parts := strings.SplitN(strings.Trim(ref, "/"), "/", 2)
	if p == nil || p.base == "" || len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return WorkItem{}, fmt.Errorf("invalid swarm-manager work reference %q", ref)
	}
	var resp struct {
		Item WorkItem `json:"item"`
	}
	if err := p.request(ctx, http.MethodGet, "/api/v1/backlog/"+parts[0]+"/"+parts[1], nil, &resp); err != nil {
		return WorkItem{}, err
	}
	return resp.Item, nil
}
