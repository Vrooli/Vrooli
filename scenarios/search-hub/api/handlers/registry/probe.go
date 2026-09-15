package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

type ScenarioURLResolver interface {
	ResolveScenarioURL(ctx context.Context, scenarioID string) (string, error)
}

type EndpointProber interface {
	Probe(ctx context.Context, descriptor *registryv1.ProviderDescriptor) error
}

type HTTPProbe struct {
	Resolver ScenarioURLResolver
	Client   *http.Client
}

func (p HTTPProbe) Probe(ctx context.Context, descriptor *registryv1.ProviderDescriptor) error {
	if descriptor == nil || descriptor.GetEndpoint() == nil || descriptor.GetEndpoint().GetHttpJson() == nil {
		return nil
	}
	hj := descriptor.GetEndpoint().GetHttpJson()
	if p.Resolver == nil {
		return fmt.Errorf("endpoint probe: scenario resolver is not configured")
	}
	base, err := p.Resolver.ResolveScenarioURL(ctx, hj.GetScenarioId())
	if err != nil {
		return fmt.Errorf("endpoint probe: resolve %q: %w", hj.GetScenarioId(), err)
	}
	body := probeBody(hj.GetBodyTemplate())
	if !json.Valid(body) {
		return fmt.Errorf("endpoint probe: body_template is not valid JSON after probe substitution")
	}
	method := strings.TrimSpace(hj.GetMethod().String())
	if method == "" || method == "HTTP_METHOD_UNSPECIFIED" {
		method = http.MethodPost
	} else {
		method = strings.TrimPrefix(method, "HTTP_METHOD_")
	}
	request, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(base, "/")+"/"+strings.TrimLeft(hj.GetPath(), "/"), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("endpoint probe: build request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	client := p.Client
	if client == nil {
		client = &http.Client{}
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("endpoint probe: request: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("endpoint probe: provider returned HTTP %d", response.StatusCode)
	}
	return nil
}

func probeBody(template string) []byte {
	if strings.TrimSpace(template) == "" {
		return []byte(`{"query":"__search_hub_registration_probe__","limit":1}`)
	}
	for from, to := range map[string]string{
		"{{query}}": "__search_hub_registration_probe__", "{{limit}}": "1", "{{type}}": "probe",
		"{{scope}}": "", "{{scenario}}": "search-hub", "{{control_token}}": "",
	} {
		template = strings.ReplaceAll(template, from, to)
	}
	return []byte(template)
}
