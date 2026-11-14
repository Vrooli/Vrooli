package resources

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ManagedUpdateMetricProof struct {
	Path       string    `json:"path"`
	Name       string    `json:"name"`
	Value      float64   `json:"value"`
	ObservedAt time.Time `json:"observed_at"`
}

func managedUpdateMetricsQualified(p *ManagedUpdateProof, c ManagedUpdateContract) bool {
	if p == nil || c.Manifest.ManagedService == nil || c.Manifest.ManagedService.UpdatePolicy == nil {
		return false
	}
	metrics := c.Manifest.ManagedService.UpdatePolicy.ReadinessMetrics
	if len(p.Metrics) != len(metrics) {
		return false
	}
	for i, m := range metrics {
		proof := p.Metrics[i]
		if proof.Path != m.Path || proof.Name != m.Name || math.IsNaN(proof.Value) || math.IsInf(proof.Value, 0) || proof.Value < m.Minimum || proof.ObservedAt.IsZero() || proof.ObservedAt.After(p.ObservedAt) || p.ObservedAt.Sub(proof.ObservedAt) > managedUpdateProofMaxAge {
			return false
		}
	}
	return true
}

func managedUpdateGauge(body, name string) (float64, error) {
	count := 0
	value := float64(0)
	for _, line := range strings.Split(body, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		if strings.HasPrefix(fields[0], name+"{") {
			return 0, fmt.Errorf("readiness gauge must be one unlabelled current sample")
		}
		if fields[0] != name {
			continue
		}
		if len(fields) != 2 {
			return 0, fmt.Errorf("readiness gauge contains a timestamp or malformed sample")
		}
		parsed, err := strconv.ParseFloat(fields[1], 64)
		if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return 0, fmt.Errorf("readiness gauge is not finite")
		}
		count++
		value = parsed
	}
	if count != 1 {
		return 0, fmt.Errorf("readiness gauge has %d samples; exactly one is required", count)
	}
	return value, nil
}

func observeManagedUpdateMetrics(ctx context.Context, client *http.Client, port int, c ManagedUpdateContract) ([]ManagedUpdateMetricProof, error) {
	var proofs []ManagedUpdateMetricProof
	type observedBody struct {
		body string
		at   time.Time
	}
	bodies := map[string]observedBody{}
	for _, m := range c.Manifest.ManagedService.UpdatePolicy.ReadinessMetrics {
		observed, ok := bodies[m.Path]
		if !ok {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:"+strconv.Itoa(port)+m.Path, nil)
			if err != nil {
				return nil, err
			}
			req.Header.Set("Cache-Control", "no-cache")
			response, err := client.Do(req)
			if err != nil {
				return nil, err
			}
			data, readErr := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
			response.Body.Close()
			if readErr != nil || len(data) > 1<<20 || response.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("role metrics read is unavailable, oversized or not HTTP 200")
			}
			observed = observedBody{body: string(data), at: time.Now().UTC()}
			bodies[m.Path] = observed
		}
		value, err := managedUpdateGauge(observed.body, m.Name)
		if err != nil || value < m.Minimum {
			return nil, fmt.Errorf("role readiness metric %s does not meet minimum %g: %v", m.Name, m.Minimum, err)
		}
		proofs = append(proofs, ManagedUpdateMetricProof{Path: m.Path, Name: m.Name, Value: value, ObservedAt: observed.at})
	}
	return proofs, nil
}
