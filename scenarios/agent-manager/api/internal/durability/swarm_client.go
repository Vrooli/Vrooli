package durability

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"agent-manager/internal/domain"

	"github.com/vrooli/api-core/discovery"
)

type swarmEvidenceResponse struct {
	Evidence []struct {
		Kind      string `json:"kind"`
		Reference string `json:"reference"`
		At        string `json:"at"`
		Lane      string `json:"lane"`
	} `json:"evidence"`
}

// SwarmEvidenceClient reads only the swarm evidence projection. It does not
// import swarm-manager's internal packages, preserving scenario boundaries.
type SwarmEvidenceClient struct {
	HTTP       *http.Client
	ResolveURL func(context.Context, string) (string, error)
}

func NewSwarmEvidenceClient() *SwarmEvidenceClient {
	return &SwarmEvidenceClient{HTTP: &http.Client{Timeout: 3 * time.Second}, ResolveURL: discovery.ResolveScenarioURLDefault}
}

func (c *SwarmEvidenceClient) ReadDurabilityEvidence(ctx context.Context, run *domain.Run) ([]Evidence, error) {
	if run == nil {
		return nil, fmt.Errorf("run is required")
	}
	base, err := c.ResolveURL(ctx, "swarm-manager")
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("run_id", run.ID.String())
	if run.StartedAt != nil {
		query.Set("started_at", run.StartedAt.UTC().Format(time.RFC3339Nano))
	} else if run.ImportedAt != nil {
		query.Set("started_at", run.ImportedAt.UTC().Format(time.RFC3339Nano))
	}
	if len(run.Subject) > 0 {
		encoded, marshalErr := json.Marshal(run.Subject)
		if marshalErr != nil {
			return nil, marshalErr
		}
		query.Set("subjects", string(encoded))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/api/v1/durability/evidence?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("swarm durability evidence returned %s", resp.Status)
	}
	var payload swarmEvidenceResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	out := make([]Evidence, 0, len(payload.Evidence))
	for _, item := range payload.Evidence {
		at, parseErr := time.Parse(time.RFC3339Nano, item.At)
		if parseErr != nil {
			return nil, fmt.Errorf("parse swarm evidence timestamp: %w", parseErr)
		}
		out = append(out, Evidence{Kind: item.Kind, Reference: item.Reference, At: at, Lane: item.Lane})
	}
	return out, nil
}
