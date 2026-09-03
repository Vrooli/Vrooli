// Package hetzner adapts the Hetzner Cloud API to Compute Manager's narrow
// provider boundary. The provider's service-specific terms must be reviewed
// separately from its general terms before enabling resale.
package hetzner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"compute-manager/internal/provider"
)

type TokenSource func(context.Context) (string, error)

type Provider struct {
	BaseURL    string
	HTTPClient *http.Client
	Token      TokenSource
}

func (p *Provider) Name() string { return "hetzner" }
func (*Provider) Facts() provider.BillingFacts {
	return provider.BillingFacts{RoundingUnit: time.Hour, MinimumBillable: time.Hour, StoppedStillBills: true, InboundCountsToward: false}
}

type serverResponse struct {
	Server server `json:"server"`
}
type listResponse struct {
	Servers []server `json:"servers"`
}
type server struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Created   time.Time `json:"created"`
	PublicNet struct {
		IPv4 struct {
			IP string `json:"ip"`
		} `json:"ipv4"`
	} `json:"public_net"`
	Datacenter struct {
		Location struct {
			Name string `json:"name"`
		} `json:"location"`
	} `json:"datacenter"`
	ServerType struct {
		Name string `json:"name"`
	} `json:"server_type"`
	Image struct {
		Name string `json:"name"`
	} `json:"image"`
}

func (p *Provider) Create(ctx context.Context, spec provider.Spec) (provider.Instance, error) {
	var body struct {
		Name       string            `json:"name"`
		ServerType string            `json:"server_type"`
		Image      string            `json:"image"`
		Location   string            `json:"location"`
		UserData   string            `json:"user_data,omitempty"`
		Labels     map[string]string `json:"labels,omitempty"`
	}
	body.Name = "vrooli-compute-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	body.ServerType, body.Image, body.Location, body.UserData, body.Labels = spec.Size, spec.Image, spec.Region, spec.UserData, spec.Tags
	var out serverResponse
	if err := p.do(ctx, http.MethodPost, "/servers", body, &out); err != nil {
		return provider.Instance{}, err
	}
	return p.mapServer(out.Server), nil
}

func (p *Provider) Describe(ctx context.Context, id string) (provider.Instance, error) {
	var out serverResponse
	if err := p.do(ctx, http.MethodGet, "/servers/"+id, nil, &out); err != nil {
		return provider.Instance{}, err
	}
	return p.mapServer(out.Server), nil
}

func (p *Provider) List(ctx context.Context) ([]provider.Instance, error) {
	var out listResponse
	if err := p.do(ctx, http.MethodGet, "/servers", nil, &out); err != nil {
		return nil, err
	}
	items := make([]provider.Instance, 0, len(out.Servers))
	for _, item := range out.Servers {
		items = append(items, p.mapServer(item))
	}
	return items, nil
}

func (p *Provider) Destroy(ctx context.Context, id string) error {
	return p.do(ctx, http.MethodDelete, "/servers/"+id, nil, nil)
}

func (p *Provider) mapServer(s server) provider.Instance {
	return provider.Instance{ID: strconv.Itoa(s.ID), Region: s.Datacenter.Location.Name, Size: s.ServerType.Name, Image: s.Image.Name, Address: s.PublicNet.IPv4.IP, CreatedAt: s.Created.UTC()}
}

func (p *Provider) do(ctx context.Context, method, path string, payload, out any) error {
	if p.Token == nil {
		return fmt.Errorf("%w: Hetzner credential source unavailable", provider.ErrProviderUnavailable)
	}
	token, err := p.Token(ctx)
	if err != nil {
		return fmt.Errorf("%w: resolve Hetzner credential: %v", provider.ErrProviderUnavailable, err)
	}
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	base := p.BaseURL
	if base == "" {
		base = "https://api.hetzner.cloud/v1"
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	client := p.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", provider.ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			return fmt.Errorf("%w: Hetzner returned %s: %s", provider.ErrProviderUnavailable, resp.Status, string(data))
		}
		return fmt.Errorf("Hetzner returned %s: %s", resp.Status, string(data))
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decode Hetzner response: %w", err)
		}
	}
	return nil
}
