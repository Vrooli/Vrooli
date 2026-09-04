// Package digitalocean adapts the DigitalOcean Droplet API to Compute
// Manager's deliberately narrow provider boundary.
package digitalocean

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

func (*Provider) Name() string { return "digitalocean" }

func (*Provider) Facts() provider.BillingFacts {
	return provider.BillingFacts{
		RoundingUnit:        time.Second,
		MinimumBillable:     time.Minute,
		StoppedStillBills:   true,
		InboundCountsToward: false,
	}
}

type dropletEnvelope struct {
	Droplet droplet `json:"droplet"`
}

type listEnvelope struct {
	Droplets []droplet `json:"droplets"`
}

type invoiceListEnvelope struct {
	Invoices []invoiceSummary `json:"invoices"`
	Meta     pageMeta         `json:"meta"`
}

type invoiceEnvelope struct {
	Items []invoiceItem `json:"invoice_items"`
	Meta  pageMeta      `json:"meta"`
}

type pageMeta struct {
	Total int `json:"total"`
}

type invoiceSummary struct {
	UUID string `json:"invoice_uuid"`
}

type invoiceItem struct {
	Amount       string `json:"amount"`
	Duration     string `json:"duration"`
	DurationUnit string `json:"duration_unit"`
	Product      string `json:"product"`
	ResourceID   string `json:"resource_id"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
}

type droplet struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Region    struct {
		Slug string `json:"slug"`
	} `json:"region"`
	Size struct {
		Slug string `json:"slug"`
	} `json:"size"`
	Image struct {
		Slug string `json:"slug"`
		Name string `json:"name"`
	} `json:"image"`
	Tags     []string `json:"tags"`
	Networks struct {
		V4 []struct {
			Type string `json:"type"`
			IP   string `json:"ip_address"`
		} `json:"v4"`
	} `json:"networks"`
}

func (p *Provider) Create(ctx context.Context, spec provider.Spec) (provider.Instance, error) {
	now := time.Now
	if p.Now != nil {
		now = p.Now
	}
	body := struct {
		Name     string   `json:"name"`
		Region   string   `json:"region"`
		Size     string   `json:"size"`
		Image    string   `json:"image"`
		UserData string   `json:"user_data,omitempty"`
		Tags     []string `json:"tags,omitempty"`
	}{
		Name:     "vrooli-compute-" + strconv.FormatInt(now().UnixNano(), 10),
		Region:   spec.Region,
		Size:     spec.Size,
		Image:    spec.Image,
		UserData: spec.UserData,
		Tags:     tagList(spec.Tags),
	}
	var out dropletEnvelope
	if err := p.do(ctx, http.MethodPost, "/droplets", body, &out); err != nil {
		return provider.Instance{}, err
	}
	return mapDroplet(out.Droplet), nil
}

func (p *Provider) Describe(ctx context.Context, id string) (provider.Instance, error) {
	var out dropletEnvelope
	if err := p.do(ctx, http.MethodGet, "/droplets/"+url.PathEscape(id), nil, &out); err != nil {
		return provider.Instance{}, err
	}
	return mapDroplet(out.Droplet), nil
}

func (p *Provider) List(ctx context.Context) ([]provider.Instance, error) {
	var out listEnvelope
	if err := p.do(ctx, http.MethodGet, "/droplets", nil, &out); err != nil {
		return nil, err
	}
	items := make([]provider.Instance, 0, len(out.Droplets))
	for _, item := range out.Droplets {
		items = append(items, mapDroplet(item))
	}
	return items, nil
}

func (p *Provider) Destroy(ctx context.Context, id string) error {
	return p.do(ctx, http.MethodDelete, "/droplets/"+url.PathEscape(id), nil, nil)
}

// BillingStatements reads invoice items for the requested period and maps
// resource durations to the provider-neutral minute statement used by the
// reconciliation service. DigitalOcean exposes resource IDs and billed
// durations on invoice items, unlike its aggregate billing-insights endpoint.
func (p *Provider) BillingStatements(ctx context.Context, from, to time.Time) ([]provider.BillingStatement, error) {
	if !from.Before(to) {
		return nil, fmt.Errorf("billing statement range must have an end after its start")
	}
	const pageSize = 200
	var statements []provider.BillingStatement
	for page := 1; ; page++ {
		var invoices invoiceListEnvelope
		path := fmt.Sprintf("/customers/my/invoices?per_page=%d&page=%d", pageSize, page)
		if err := p.do(ctx, http.MethodGet, path, nil, &invoices); err != nil {
			return nil, err
		}
		for _, summary := range invoices.Invoices {
			if summary.UUID == "" {
				continue
			}
			for itemPage := 1; ; itemPage++ {
				var invoice invoiceEnvelope
				itemPath := fmt.Sprintf("/customers/my/invoices/%s?per_page=%d&page=%d", url.PathEscape(summary.UUID), pageSize, itemPage)
				if err := p.do(ctx, http.MethodGet, itemPath, nil, &invoice); err != nil {
					return nil, err
				}
				for _, item := range invoice.Items {
					statement, ok := mapInvoiceItem(item, from, to)
					if ok {
						statements = append(statements, statement)
					}
				}
				if len(invoice.Items) < pageSize || itemPage*pageSize >= invoice.Meta.Total {
					break
				}
			}
		}
		if len(invoices.Invoices) < pageSize || page*pageSize >= invoices.Meta.Total {
			break
		}
	}
	return statements, nil
}

func mapInvoiceItem(item invoiceItem, from, to time.Time) (provider.BillingStatement, bool) {
	if item.ResourceID == "" || (item.Product != "" && !strings.Contains(strings.ToLower(item.Product), "droplet")) {
		return provider.BillingStatement{}, false
	}
	start, err := parseOptionalTime(item.StartTime)
	if err != nil {
		return provider.BillingStatement{}, false
	}
	end, err := parseOptionalTime(item.EndTime)
	if err != nil {
		return provider.BillingStatement{}, false
	}
	if start.IsZero() {
		start = from
	}
	if end.IsZero() {
		end = to
	}
	if !end.After(from) || !start.Before(to) {
		return provider.BillingStatement{}, false
	}
	duration, err := strconv.ParseFloat(item.Duration, 64)
	if err != nil || duration < 0 {
		return provider.BillingStatement{}, false
	}
	minutes := duration
	switch strings.ToLower(strings.TrimSpace(item.DurationUnit)) {
	case "hour", "hours":
		minutes *= 60
	case "day", "days":
		minutes *= 24 * 60
	case "second", "seconds":
		minutes /= 60
	}
	return provider.BillingStatement{Provider: "digitalocean", ProviderInstanceID: item.ResourceID, Minutes: int64(math.Ceil(minutes)), From: start, To: end}, true
}

func parseOptionalTime(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, value)
}

func mapDroplet(d droplet) provider.Instance {
	address := ""
	for _, network := range d.Networks.V4 {
		if network.Type == "public" {
			address = network.IP
			break
		}
	}
	image := d.Image.Slug
	if image == "" {
		image = d.Image.Name
	}
	return provider.Instance{ID: strconv.Itoa(d.ID), Region: d.Region.Slug, Size: d.Size.Slug, Image: image, Address: address, CreatedAt: d.CreatedAt.UTC(), Tags: parseTags(d.Tags)}
}

func parseTags(tags []string) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	parsed := make(map[string]string, len(tags))
	for _, tag := range tags {
		key, value, ok := strings.Cut(tag, ":")
		if ok && key != "" {
			parsed[key] = value
		}
	}
	return parsed
}

func tagList(tags map[string]string) []string {
	result := make([]string, 0, len(tags))
	for key, value := range tags {
		if value == "" {
			result = append(result, key)
		} else {
			// DigitalOcean tags are names rather than a label map. Use the
			// documented key:value convention so the metadata remains
			// filterable without relying on an unsupported '=' delimiter.
			result = append(result, key+":"+value)
		}
	}
	return result
}

func (p *Provider) do(ctx context.Context, method, path string, payload, out any) error {
	if p.Token == nil {
		return fmt.Errorf("%w: DigitalOcean credential source unavailable", provider.ErrProviderUnavailable)
	}
	token, err := p.Token(ctx)
	if err != nil {
		return fmt.Errorf("%w: resolve DigitalOcean credential: %v", provider.ErrProviderUnavailable, err)
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
		base = "https://api.digitalocean.com/v2"
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
			return fmt.Errorf("%w: DigitalOcean returned %s: %s", provider.ErrProviderUnavailable, resp.Status, string(data))
		}
		return fmt.Errorf("DigitalOcean returned %s: %s", resp.Status, string(data))
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decode DigitalOcean response: %w", err)
		}
	}
	return nil
}
