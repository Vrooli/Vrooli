package validation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/health"
)

const apiPortKey = "API_PORT"

type PortURLResolver interface {
	ResolveScenarioURL(ctx context.Context, scenarioSlug, portKey string) (string, error)
}

type HealthProbe interface {
	Probe(ctx context.Context, target Target, timeout time.Duration) HealthProbeResult
}

type HealthProbeResult struct {
	Requested        bool
	URL              string
	StatusCode       int
	ContentType      string
	ElapsedMillis    int64
	FailureClass     string
	Error            string
	Payload          *HealthPayload
	SchemaValid      bool
	SchemaViolations []string
}

type HealthPayload struct {
	Status          string
	Service         string
	Timestamp       string
	Readiness       bool
	Version         string
	DependencyCount int
}

type HTTPHealthProbeDeps struct {
	Client       *http.Client
	Resolver     PortURLResolver
	ProbeTimeout time.Duration
}

type httpHealthProbe struct {
	client       *http.Client
	resolver     PortURLResolver
	probeTimeout time.Duration
}

func NewHTTPHealthProbe(d HTTPHealthProbeDeps) HealthProbe {
	client := d.Client
	if client == nil {
		client = http.DefaultClient
	}
	resolver := d.Resolver
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	timeout := d.ProbeTimeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return httpHealthProbe{client: client, resolver: resolver, probeTimeout: timeout}
}

func (p httpHealthProbe) Probe(ctx context.Context, target Target, timeout time.Duration) HealthProbeResult {
	if timeout <= 0 {
		timeout = p.probeTimeout
	}
	result := HealthProbeResult{Requested: true}
	probeURL, err := p.healthURL(ctx, target)
	if err != nil {
		result.FailureClass = "url_unresolved"
		result.Error = err.Error()
		return result
	}
	result.URL = probeURL

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
	if err != nil {
		result.FailureClass = "url_invalid"
		result.Error = err.Error()
		return result
	}

	start := time.Now()
	resp, err := p.client.Do(req)
	result.ElapsedMillis = time.Since(start).Milliseconds()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			result.FailureClass = "timeout"
		} else {
			result.FailureClass = "unreachable"
		}
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.ContentType = resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(result.ContentType), "application/json") {
		result.FailureClass = "non_json"
		result.Error = "health endpoint did not return application/json"
		io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
		return result
	}

	var payload health.Response
	dec := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	if err := dec.Decode(&payload); err != nil {
		result.FailureClass = "malformed_json"
		result.Error = err.Error()
		return result
	}
	result.Payload = &HealthPayload{
		Status:          payload.Status,
		Service:         payload.Service,
		Timestamp:       payload.Timestamp,
		Readiness:       payload.Readiness,
		Version:         payload.Version,
		DependencyCount: len(payload.Dependencies),
	}
	result.SchemaViolations = validateHealthPayload(resp.StatusCode, payload)
	result.SchemaValid = len(result.SchemaViolations) == 0
	if !result.SchemaValid {
		result.FailureClass = "schema_invalid"
		result.Error = strings.Join(result.SchemaViolations, "; ")
	}
	return result
}

func (p httpHealthProbe) healthURL(ctx context.Context, target Target) (string, error) {
	raw := strings.TrimSpace(target.Service.HealthAPICheckURL)
	if raw != "" && !strings.Contains(raw, "${API_PORT}") {
		return raw, nil
	}
	base, err := p.resolver.ResolveScenarioURL(ctx, target.Scenario, apiPortKey)
	if err != nil {
		return "", err
	}
	healthPath := strings.TrimSpace(target.Service.HealthAPIPath)
	if healthPath == "" {
		healthPath = "/health"
	}
	return joinURL(base, healthPath)
}

func joinURL(base, path string) (string, error) {
	parsed, err := url.Parse(strings.TrimRight(base, "/"))
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("resolved API base URL %q is not absolute", base)
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + path
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func validateHealthPayload(statusCode int, payload health.Response) []string {
	var violations []string
	switch payload.Status {
	case health.StatusHealthy, health.StatusDegraded, health.StatusUnhealthy:
	default:
		violations = append(violations, "status must be healthy, degraded, or unhealthy")
	}
	if strings.TrimSpace(payload.Service) == "" {
		violations = append(violations, "service is required")
	}
	if strings.TrimSpace(payload.Timestamp) == "" {
		violations = append(violations, "timestamp is required")
	} else if _, err := time.Parse(time.RFC3339, payload.Timestamp); err != nil {
		violations = append(violations, "timestamp must be RFC3339")
	}
	switch payload.Status {
	case health.StatusHealthy, health.StatusDegraded:
		if !payload.Readiness {
			violations = append(violations, "readiness must be true for healthy or degraded status")
		}
		if statusCode != http.StatusOK {
			violations = append(violations, "healthy or degraded status must return HTTP 200")
		}
	case health.StatusUnhealthy:
		if payload.Readiness {
			violations = append(violations, "readiness must be false for unhealthy status")
		}
		if statusCode != http.StatusServiceUnavailable {
			violations = append(violations, "unhealthy status must return HTTP 503")
		}
	}
	for name, dep := range payload.Dependencies {
		if strings.TrimSpace(name) == "" {
			violations = append(violations, "dependency names must be non-empty")
		}
		if !dep.Connected && dep.Error == nil {
			violations = append(violations, fmt.Sprintf("dependency %q is disconnected without error detail", name))
		}
	}
	return violations
}

func (s *Service) validateLiveHealth(ctx context.Context, target *Target) []Finding {
	probe := s.healthProbe
	if probe == nil {
		probe = NewHTTPHealthProbe(HTTPHealthProbeDeps{ProbeTimeout: s.probeTimeout})
	}
	result := probe.Probe(ctx, *target, s.probeTimeout)
	target.Health = result
	switch {
	case result.FailureClass == "":
		return nil
	case result.FailureClass == "schema_invalid":
		return []Finding{healthFinding(
			CodeHealthSchemaInvalid,
			"Health response schema invalid",
			result.URL,
			fmt.Sprintf("live /health probe returned invalid api-core health payload: %s", result.Error),
			"return the api-core health response schema with consistent status, readiness, timestamp, service, and dependency details",
		)}
	default:
		location := result.URL
		if location == "" {
			location = nonEmpty(target.ServiceManifestPath, target.RootPath)
		}
		return []Finding{healthFinding(
			CodeHealthProbeFailed,
			"Health probe failed",
			location,
			fmt.Sprintf("live /health probe failed (%s): %s", result.FailureClass, result.Error),
			"start the target API through the scenario lifecycle and expose a reachable JSON /health endpoint",
		)}
	}
}

func healthFinding(code, title, location, message, remediation string) Finding {
	return Finding{
		Severity:    SeverityError,
		Code:        code,
		Title:       title,
		Location:    location,
		Message:     message,
		Remediation: remediation,
	}
}
