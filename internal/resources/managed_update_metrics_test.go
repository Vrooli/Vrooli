package resources

import (
	"math"
	"testing"
	"time"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestManagedUpdateGaugeRejectsAmbiguousStaleAndNonfiniteSamples(t *testing.T) {
	for _, body := range []string{"", "active_connections 1\nactive_connections 2", "active_connections{edge=\"one\"} 4", "active_connections 4 12345", "active_connections NaN", "active_connections +Inf", "active_connections invalid", "active_connections 4 extra fields"} {
		if _, err := managedUpdateGauge(body, "active_connections"); err == nil {
			t.Fatalf("invalid gauge qualified: %q", body)
		}
	}
	value, err := managedUpdateGauge("# HELP active_connections Count\r\nother 3\nactive_connections 4\n", "active_connections")
	if err != nil || value != 4 {
		t.Fatalf("valid current sample: %g %v", value, err)
	}
}

func TestManagedUpdateMetricProofIsRequiredAndFresh(t *testing.T) {
	u := updateTestInitial(t)
	u.Target.Manifest.ManagedService.UpdatePolicy.ReadinessMetrics = []resourcedeployment.ServiceUpdateMetric{{Path: "/metrics", Name: "active_connections", Minimum: 1}}
	u.Target.Digest = u.Target.digest()
	now := time.Now().UTC()
	p := &ManagedUpdateProof{ObservedAt: now}
	if managedUpdateMetricsQualified(p, u.Target) {
		t.Fatal("missing edge evidence qualified")
	}
	p.Metrics = []ManagedUpdateMetricProof{{Path: "/metrics", Name: "active_connections", Value: 4, ObservedAt: now}}
	if !managedUpdateMetricsQualified(p, u.Target) {
		t.Fatal("current metric did not qualify")
	}
	for _, change := range []func(*ManagedUpdateMetricProof){func(p *ManagedUpdateMetricProof) { p.Value = 0 }, func(p *ManagedUpdateMetricProof) { p.Value = math.NaN() }, func(p *ManagedUpdateMetricProof) { p.Name = "foreign" }, func(p *ManagedUpdateMetricProof) { p.ObservedAt = now.Add(-managedUpdateProofMaxAge - time.Nanosecond) }, func(p *ManagedUpdateMetricProof) { p.ObservedAt = now.Add(time.Nanosecond) }} {
		proof := *p
		proof.Metrics = append([]ManagedUpdateMetricProof(nil), p.Metrics...)
		change(&proof.Metrics[0])
		if managedUpdateMetricsQualified(&proof, u.Target) {
			t.Fatal("stale, foreign or unhealthy metric qualified")
		}
	}
}
