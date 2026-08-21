package scenarioruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	apihealth "github.com/vrooli/api-core/health"
	"github.com/vrooli/vrooli/internal/scenario"
)

type HealthProbe struct {
	Clock            Clock
	HTTPClient       *http.Client
	MaxResponseBytes int64
}

type HealthProbeInput struct {
	InstanceID   string
	Scenario     string
	HealthConfig *scenario.HealthConfig
	Ports        map[string]int
}

func (p HealthProbe) Probe(ctx context.Context, in HealthProbeInput) HealthSnapshot {
	now := p.now()
	snapshot := HealthSnapshot{
		InstanceID: in.InstanceID,
		Scenario:   in.Scenario,
		Status:     HealthStatusNotConfigured,
		CheckedAt:  &now,
	}
	if in.HealthConfig == nil || len(in.HealthConfig.Checks) == 0 {
		return snapshot
	}

	start := time.Now()
	client := p.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: healthTimeout(in.HealthConfig)}
	}

	var criticalFailure bool
	var nonCriticalFailure bool
	var readiness *bool
	var schemaValid *bool
	var responseJSON string
	var failures []string

	for _, check := range in.HealthConfig.Checks {
		result := p.performCheck(ctx, client, check, in.Ports)
		if result.responseJSON != "" && responseJSON == "" {
			responseJSON = result.responseJSON
		}
		if result.schemaValid != nil && schemaValid == nil {
			schemaValid = result.schemaValid
		}
		if result.readiness != nil && readiness == nil {
			readiness = result.readiness
		}
		if result.err == nil {
			if result.status == apihealth.StatusDegraded {
				nonCriticalFailure = true
			}
			if result.status == apihealth.StatusUnhealthy {
				if check.Critical {
					criticalFailure = true
				} else {
					nonCriticalFailure = true
				}
			}
			continue
		}
		failures = append(failures, checkFailureMessage(check, result.err))
		if check.Critical {
			criticalFailure = true
		} else {
			nonCriticalFailure = true
		}
	}

	switch {
	case criticalFailure:
		snapshot.Status = HealthStatusUnhealthy
	case nonCriticalFailure:
		snapshot.Status = HealthStatusDegraded
	default:
		snapshot.Status = HealthStatusHealthy
	}
	if readiness == nil {
		ready := snapshot.Status == HealthStatusHealthy || snapshot.Status == HealthStatusDegraded
		readiness = &ready
	}
	latency := time.Since(start).Milliseconds()
	snapshot.Readiness = readiness
	snapshot.LatencyMillis = &latency
	snapshot.SchemaValid = schemaValid
	snapshot.ResponseJSON = responseJSON
	snapshot.Error = boundString(strings.Join(failures, "; "), 4096)
	return snapshot
}

type checkProbeResult struct {
	status       string
	readiness    *bool
	schemaValid  *bool
	responseJSON string
	err          error
}

func (p HealthProbe) performCheck(ctx context.Context, client *http.Client, check scenario.HealthCheck, ports map[string]int) checkProbeResult {
	switch strings.TrimSpace(check.Type) {
	case "", "http":
		return p.performHTTPCheck(ctx, client, check, ports)
	default:
		if err := scenario.PerformHealthCheck(check, ports); err != nil {
			return checkProbeResult{err: err}
		}
		return checkProbeResult{status: HealthStatusHealthy}
	}
}

func (p HealthProbe) performHTTPCheck(ctx context.Context, client *http.Client, check scenario.HealthCheck, ports map[string]int) checkProbeResult {
	target, err := scenario.ExpandHealthTarget(check.Target, ports)
	if err != nil {
		return checkProbeResult{err: err}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return checkProbeResult{err: fmt.Errorf("invalid URL %q: %w", target, err)}
	}
	resp, err := client.Do(req)
	if err != nil {
		return checkProbeResult{err: err}
	}
	defer resp.Body.Close()

	body, truncated, err := readBounded(resp.Body, p.maxResponseBytes())
	if err != nil {
		return checkProbeResult{err: fmt.Errorf("read health response: %w", err)}
	}
	responseJSON := strings.TrimSpace(string(body))
	if truncated {
		responseJSON = boundString(responseJSON, int(p.maxResponseBytes()))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return checkProbeResult{
			responseJSON: responseJSON,
			err:          fmt.Errorf("HTTP %d", resp.StatusCode),
		}
	}
	if responseJSON == "" {
		return checkProbeResult{status: HealthStatusHealthy}
	}

	var decoded apihealth.Response
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&decoded); err != nil {
		valid := false
		return checkProbeResult{
			schemaValid:  &valid,
			responseJSON: responseJSON,
			err:          fmt.Errorf("invalid health response schema: %w", err),
		}
	}
	valid := isRecognizedHealthResponse(decoded)
	if !valid {
		return checkProbeResult{
			schemaValid:  &valid,
			responseJSON: responseJSON,
			err:          fmt.Errorf("invalid health response schema"),
		}
	}
	return checkProbeResult{
		status:       decoded.Status,
		readiness:    &decoded.Readiness,
		schemaValid:  &valid,
		responseJSON: responseJSON,
	}
}

func (p HealthProbe) now() time.Time {
	if p.Clock == nil {
		return time.Now().UTC()
	}
	return p.Clock.Now().UTC()
}

func (p HealthProbe) maxResponseBytes() int64 {
	if p.MaxResponseBytes <= 0 {
		return DefaultMaxHealthResponseBytes
	}
	return p.MaxResponseBytes
}

func healthTimeout(health *scenario.HealthConfig) time.Duration {
	if health != nil && health.Timeout > 0 {
		return time.Duration(health.Timeout) * time.Millisecond
	}
	return 5 * time.Second
}

func isRecognizedHealthResponse(resp apihealth.Response) bool {
	if strings.TrimSpace(resp.Service) == "" || strings.TrimSpace(resp.Timestamp) == "" {
		return false
	}
	switch resp.Status {
	case apihealth.StatusHealthy, apihealth.StatusDegraded, apihealth.StatusUnhealthy:
		return true
	default:
		return false
	}
}

func readBounded(r io.Reader, max int64) ([]byte, bool, error) {
	if max <= 0 {
		max = DefaultMaxHealthResponseBytes
	}
	limited := io.LimitReader(r, max+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, false, err
	}
	if int64(len(body)) <= max {
		return body, false, nil
	}
	return body[:max], true, nil
}

func checkFailureMessage(check scenario.HealthCheck, err error) string {
	name := strings.TrimSpace(check.Name)
	if name == "" {
		name = strings.TrimSpace(check.Target)
	}
	if name == "" {
		name = strings.TrimSpace(check.Type)
	}
	if name == "" {
		name = "health_check"
	}
	return fmt.Sprintf("%s: %v", name, err)
}

func boundString(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max]
}
