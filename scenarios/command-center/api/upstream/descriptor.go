package upstream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Transport is the small runtime vocabulary understood by connector packs.
type Transport string

const (
	TransportHTTPJSON Transport = "http-json"
	TransportConnect  Transport = "connect"
	TransportGraphQL  Transport = "graphql"
)

// Descriptor contains data-only source instructions. Secrets are deliberately
// represented by an auth resolver, never serialized into this structure.
type Descriptor struct {
	ID          string
	Transport   Transport
	ResolveBase func() string
	Auth        func(*http.Request)
	TTL         time.Duration
	Paths       []string
	Features    map[string]string
}

// NewDescriptorClient executes a connector descriptor without importing a
// producer's generated client. Connect and GraphQL descriptors use POST; the
// response remains raw JSON so the domain registry owns interpretation.
func NewDescriptorClient(d Descriptor) Client {
	return &descriptorClient{descriptor: d, http: &http.Client{Timeout: 5 * time.Second}}
}

type descriptorClient struct {
	descriptor Descriptor
	http       *http.Client
}

func (c *descriptorClient) Name() string { return c.descriptor.ID }
func (c *descriptorClient) Fetch(ctx context.Context, path string) (json.RawMessage, error) {
	if c.descriptor.ResolveBase == nil || strings.TrimSpace(c.descriptor.ResolveBase()) == "" {
		return nil, ErrNotAvailable
	}
	base := strings.TrimRight(c.descriptor.ResolveBase(), "/")
	u, err := url.Parse(base + path)
	if err != nil {
		return nil, err
	}
	method := http.MethodGet
	var body io.Reader
	// Connector catalogs can expose a mixed producer surface: REST projections
	// and Connect procedures share one origin. Procedure paths carry the
	// generated service namespace, so route those as Connect POSTs even when
	// the catalog's health/probe path is REST.
	isProcedure := strings.Contains(u.Path, ".v1.")
	if path != "/health" && (c.descriptor.Transport == TransportConnect || c.descriptor.Transport == TransportGraphQL || isProcedure) {
		method = http.MethodPost
		payload := map[string]any{}
		for key, values := range u.Query() {
			if len(values) == 0 || values[0] == "" {
				continue
			}
			value := values[0]
			if integer, parseErr := strconv.Atoi(value); parseErr == nil {
				payload[key] = integer
				continue
			}
			payload[key] = value
		}
		encoded, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return nil, marshalErr
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.descriptor.Auth != nil {
		c.descriptor.Auth(req)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w: %v", c.descriptor.ID, ErrNotAvailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotAvailable
	}
	result, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, readErr
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%s http %d: %s", c.descriptor.ID, resp.StatusCode, truncate(string(result), 200))
	}
	return json.RawMessage(result), nil
}
func (c *descriptorClient) ProbeFeatures(ctx context.Context) (map[string]string, map[string]string) {
	if len(c.descriptor.Paths) == 0 {
		return nil, nil
	}
	if _, err := c.Fetch(ctx, c.descriptor.Paths[0]); err != nil {
		return nil, nil
	}
	status, reasons := map[string]string{}, map[string]string{}
	for feature := range c.descriptor.Features {
		status[feature] = "compatible"
		reasons[feature] = "descriptor projection returned successfully"
	}
	return status, reasons
}
