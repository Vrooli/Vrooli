// Package hetzner adapts the Hetzner Cloud API to Compute Manager's narrow
// provider boundary. The provider's service-specific terms must be reviewed
// separately from its general terms before enabling resale.
package hetzner

import (
	"context"
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
	Now        func() time.Time
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
	Labels map[string]string `json:"labels"`
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
	now := time.Now
	if p.Now != nil {
		now = p.Now
	}
	body.Name = "vrooli-compute-" + strconv.FormatInt(now().UnixNano(), 10)
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
	return provider.Instance{ID: strconv.Itoa(s.ID), Region: s.Datacenter.Location.Name, Size: s.ServerType.Name, Image: s.Image.Name, Address: s.PublicNet.IPv4.IP, CreatedAt: s.Created.UTC(), Tags: s.Labels}
}

func (p *Provider) do(ctx context.Context, method, path string, payload, out any) error {
	return provider.DoJSONRequest(ctx, p.Token, p.HTTPClient, p.BaseURL, "https://api.hetzner.cloud/v1", "Hetzner", method, path, payload, out)
}
