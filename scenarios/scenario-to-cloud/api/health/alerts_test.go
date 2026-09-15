package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/eventbus"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/perfbudget"
	"scenario-to-cloud/tlsinfo"
	"scenario-to-cloud/vps"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
)

// eventSink is a deterministic vrooli-events ingest: it records every
// envelope posted to /api/v1/events and answers 202.
type eventSink struct {
	mu        sync.Mutex
	envelopes []*domainv1.EventEnvelope
	status    int
}

func newEventSink(t *testing.T) (*eventSink, eventbus.Client) {
	t.Helper()
	sink := &eventSink{status: http.StatusAccepted}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/events" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		var envelope domainv1.EventEnvelope
		body := make([]byte, r.ContentLength)
		if _, err := r.Body.Read(body); err != nil && err.Error() != "EOF" {
			t.Errorf("read body: %v", err)
		}
		if err := protojson.Unmarshal(body, &envelope); err != nil {
			t.Errorf("decode envelope: %v", err)
		}
		sink.mu.Lock()
		sink.envelopes = append(sink.envelopes, &envelope)
		status := sink.status
		sink.mu.Unlock()
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return sink, eventbus.Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
}

func (s *eventSink) facts(t *testing.T) []map[string]any {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []map[string]any
	for _, env := range s.envelopes {
		if env.GetEventType() != perfbudget.AlertEventType || env.GetSource().GetScenario() != AlertSource {
			t.Fatalf("unexpected event %s from %s", env.GetEventType(), env.GetSource().GetScenario())
		}
		var payload structpb.Struct
		if err := env.GetData().UnmarshalTo(&payload); err != nil {
			t.Fatalf("unpack payload: %v", err)
		}
		out = append(out, payload.AsMap())
	}
	return out
}

func fixedAlerter(publisher AlertPublisher, now time.Time) *Alerter {
	a := NewAlerter(publisher, DefaultDetection(), nil)
	a.now = func() time.Time { return now }
	return a
}

func observationFor(t *testing.T, live *domain.LiveStateResult, tlsSnap *tlsinfo.Snapshot, now time.Time) (*healthv1.HealthObservation, domain.HealthResponse) {
	t.Helper()
	dep := fixtureDeployment(domain.StatusDeployed)
	if tlsSnap == nil {
		tlsSnap = fixtureTLS()
	}
	report := vps.ComputeHealth(dep, fixtureManifest(), fixtureIdentity(), live, fixtureDNS(), tlsSnap, nil)
	report.Freshness = &domain.FreshnessStatus{Status: domain.FreshnessCurrent}
	obs := Build(Input{Deployment: dep, Report: report, LiveState: live, Now: now}, DefaultPolicy())
	return obs, report
}

// TestAlerterHealthyCurrentEmitsNothing: steady healthy state publishes no
// event, on the first and on repeated observations.
func TestAlerterHealthyCurrentEmitsNothing(t *testing.T) {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	sink, client := newEventSink(t)
	a := fixedAlerter(client, now.Add(5*time.Second))
	obs, report := observationFor(t, fixtureLiveState(now, "running"), nil, now.Add(5*time.Second))
	if obs.GetStatus() != healthv1.HealthStatus_HEALTH_STATUS_HEALTHY || obs.GetFreshness() != healthv1.Freshness_FRESHNESS_CURRENT {
		t.Fatalf("fixture is not healthy+current: %s/%s", obs.GetStatus(), obs.GetFreshness())
	}
	for i := 0; i < 3; i++ {
		if published := a.Observe(context.Background(), obs, report); len(published) != 0 {
			t.Fatalf("healthy observation published %+v", published)
		}
	}
	if got := sink.facts(t); len(got) != 0 {
		t.Fatalf("sink received %d events for a healthy deployment", len(got))
	}
	if !perfbudget.ServiceHealthy(a.Open(obs.GetDeploymentId())) {
		t.Fatal("healthy deployment has open service alerts")
	}
}

// TestAlerterOutageOpensThenResolves: an unhealthy current observation opens
// service_unavailable once (carrying the observation's typed next action),
// repeats publish nothing, and recovery publishes the resolved pair.
func TestAlerterOutageOpensThenResolves(t *testing.T) {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	sink, client := newEventSink(t)
	a := fixedAlerter(client, now.Add(10*time.Second))

	down, downReport := observationFor(t, fixtureLiveState(now, "stopped"), nil, now.Add(10*time.Second))
	published := a.Observe(context.Background(), down, downReport)
	if len(published) != 1 || published[0].CheckID != perfbudget.CheckServiceUnavailable || published[0].Status != perfbudget.AlertOpen {
		t.Fatalf("first outage observation published %+v", published)
	}
	if len(a.Observe(context.Background(), down, downReport)) != 0 {
		t.Fatal("repeated outage observation re-published the open alert")
	}
	facts := sink.facts(t)
	if len(facts) != 1 {
		t.Fatalf("sink events = %d", len(facts))
	}
	f := facts[0]
	if f["check_id"] != perfbudget.CheckServiceUnavailable || f["severity"] != perfbudget.SeverityCritical || f["status"] != perfbudget.AlertOpen || f["reason"] != "status_unhealthy" {
		t.Fatalf("facts = %v", f)
	}
	if f["deployment_id"] != down.GetDeploymentId() || f["target_id"] != "host:1.2.3.4" || f["release_digest"] != "sha256:"+bundleSHA {
		t.Fatalf("identity facts = %v", f)
	}
	if f["incident_id"] != down.GetDeploymentId()+":"+perfbudget.CheckServiceUnavailable {
		t.Fatalf("incident_id = %v", f["incident_id"])
	}
	if f["observed_at"] != now.Format(time.RFC3339) || f["detected_at"] != now.Add(10*time.Second).Format(time.RFC3339) {
		t.Fatalf("times = %v / %v", f["observed_at"], f["detected_at"])
	}
	next, _ := f["next_action"].(map[string]any)
	if next["owner"] != "scenario-to-cloud" || next["kind"] != "command" || next["reference"] != "scenario-to-cloud process control "+down.GetDeploymentId()+" restart" {
		t.Fatalf("next_action was not carried over from the observation: %v", next)
	}
	if schema, ok := f["schema_version"].(float64); !ok || schema != 1 {
		t.Fatalf("schema_version = %v", f["schema_version"])
	}

	a.now = func() time.Time { return now.Add(70 * time.Second) }
	up, upReport := observationFor(t, fixtureLiveState(now.Add(60*time.Second), "running"), nil, now.Add(70*time.Second))
	published = a.Observe(context.Background(), up, upReport)
	if len(published) != 1 || published[0].Status != perfbudget.AlertResolved || published[0].Severity != perfbudget.SeverityInformational {
		t.Fatalf("recovery published %+v", published)
	}
	facts = sink.facts(t)
	if len(facts) != 2 || facts[1]["status"] != perfbudget.AlertResolved || facts[1]["incident_id"] != facts[0]["incident_id"] {
		t.Fatalf("resolved event = %v", facts)
	}
	if len(a.Open(up.GetDeploymentId())) != 0 {
		t.Fatalf("open set after recovery = %+v", a.Open(up.GetDeploymentId()))
	}
}

// TestAlerterStaleOrUnknownObservationRaisesStaleNeverHealthy: an
// unreachable target (UNKNOWN/UNKNOWN) and a stale-by-budget observation
// both raise health_observation_stale and never count as healthy.
func TestAlerterStaleOrUnknownObservationRaisesStaleNeverHealthy(t *testing.T) {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	sink, client := newEventSink(t)
	a := fixedAlerter(client, now)

	unknown, unknownReport := observationFor(t, &domain.LiveStateResult{OK: false, Error: "connection refused"}, nil, now)
	published := a.Observe(context.Background(), unknown, unknownReport)
	if len(published) != 1 || published[0].CheckID != perfbudget.CheckHealthObservationStale {
		t.Fatalf("unknown observation published %+v", published)
	}
	if perfbudget.ServiceHealthy(a.Open(unknown.GetDeploymentId())) {
		t.Fatal("unknown observation counted as healthy")
	}

	late := now.Add(10 * time.Minute)
	a.now = func() time.Time { return late }
	stale, staleReport := observationFor(t, fixtureLiveState(now, "running"), nil, late)
	if stale.GetFreshness() != healthv1.Freshness_FRESHNESS_STALE {
		t.Fatalf("fixture freshness = %s", stale.GetFreshness())
	}
	if published := a.Observe(context.Background(), stale, staleReport); len(published) != 0 {
		// Same incident id as the unknown case: still open, nothing new.
		t.Fatalf("stale observation re-published %+v", published)
	}
	facts := sink.facts(t)
	if len(facts) != 1 || facts[0]["check_id"] != perfbudget.CheckHealthObservationStale || facts[0]["severity"] != perfbudget.SeverityCritical {
		t.Fatalf("facts = %v", facts)
	}
}

// TestAlerterCertificateExpiringAndRenewalFailed covers the edge
// observation: a certificate inside the 14-day window warns, an expired or
// unprobeable certificate is critical with reason renewal_failed.
func TestAlerterCertificateExpiringAndRenewalFailed(t *testing.T) {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	sink, client := newEventSink(t)
	a := fixedAlerter(client, now)

	expiring := fixtureTLS()
	expiring.Probe.DaysRemaining = 10
	expiring.Probe.NotAfter = now.Add(10 * 24 * time.Hour).Format(certTimeLayout)
	obs, report := observationFor(t, fixtureLiveState(now, "running"), expiring, now)
	published := a.Observe(context.Background(), obs, report)
	// The legacy report also warns on tls_cert, so the deployment is
	// degraded: a warning service alert accompanies the certificate alert.
	var certAlert *perfbudget.Alert
	for i := range published {
		if published[i].CheckID == perfbudget.CheckCertificateExpiring {
			certAlert = &published[i]
		}
	}
	if certAlert == nil || certAlert.Severity != perfbudget.SeverityWarning || certAlert.Reason != "certificate_expiring" {
		t.Fatalf("expiring certificate published %+v", published)
	}

	b := fixedAlerter(client, now)
	failed := fixtureTLS()
	failed.Probe.Valid = false
	failed.Probe.DaysRemaining = -2
	failed.Probe.NotAfter = now.Add(-2 * 24 * time.Hour).Format(certTimeLayout)
	obs, report = observationFor(t, fixtureLiveState(now, "running"), failed, now)
	published = b.Observe(context.Background(), obs, report)
	byCheck := map[string]perfbudget.Alert{}
	for _, p := range published {
		byCheck[p.CheckID] = p
	}
	cert, ok := byCheck[perfbudget.CheckCertificateExpiring]
	if !ok || cert.Severity != perfbudget.SeverityCritical {
		t.Fatalf("expired certificate published %+v", published)
	}
	if _, ok := byCheck[perfbudget.CheckServiceUnavailable]; !ok {
		t.Fatalf("an invalid certificate makes the report unhealthy; expected service_unavailable too, got %+v", published)
	}

	c := fixedAlerter(client, now)
	unprobeable := fixtureTLS()
	unprobeable.Probe.Valid = false
	unprobeable.Probe.NotAfter = ""
	unprobeable.Probe.DaysRemaining = 0
	obs, report = observationFor(t, fixtureLiveState(now, "running"), unprobeable, now)
	published = c.Observe(context.Background(), obs, report)
	found := false
	for _, p := range published {
		if p.CheckID == perfbudget.CheckCertificateExpiring && p.Reason == "renewal_failed" && p.NextAction.Reference == "scenario-to-cloud edge tls-renew "+obs.GetDeploymentId() {
			found = true
		}
	}
	if !found {
		t.Fatalf("invalid certificate without expiry did not raise renewal_failed: %+v", published)
	}
	if len(sink.facts(t)) == 0 {
		t.Fatal("no certificate events reached the sink")
	}
}

// TestAlerterPublishFailureRetriesNextObservation: a failed delivery keeps
// the transition pending so the next observation publishes it again.
func TestAlerterPublishFailureRetriesNextObservation(t *testing.T) {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	sink, client := newEventSink(t)
	sink.status = http.StatusServiceUnavailable
	var logged []string
	a := NewAlerter(client, DefaultDetection(), func(msg string, _ map[string]interface{}) { logged = append(logged, msg) })
	a.now = func() time.Time { return now }

	down, report := observationFor(t, fixtureLiveState(now, "stopped"), nil, now)
	if published := a.Observe(context.Background(), down, report); len(published) != 0 {
		t.Fatalf("failed publish reported as published: %+v", published)
	}
	if len(logged) != 1 || len(a.Open(down.GetDeploymentId())) != 0 {
		t.Fatalf("failure not logged or open set advanced: %v / %+v", logged, a.Open(down.GetDeploymentId()))
	}
	sink.mu.Lock()
	sink.status = http.StatusAccepted
	sink.mu.Unlock()
	if published := a.Observe(context.Background(), down, report); len(published) != 1 {
		t.Fatalf("retry did not publish: %+v", published)
	}
}

// TestAlerterNilPublisherStillEvaluates: without a configured event bus the
// open set is still maintained (and nothing panics).
func TestAlerterNilPublisherStillEvaluates(t *testing.T) {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	a := fixedAlerter(nil, now)
	down, report := observationFor(t, fixtureLiveState(now, "stopped"), nil, now)
	if published := a.Observe(context.Background(), down, report); len(published) != 1 {
		t.Fatalf("published = %+v", published)
	}
	if perfbudget.ServiceHealthy(a.Open(down.GetDeploymentId())) {
		t.Fatal("open outage read as healthy")
	}
}

func TestLoadDetectionFallsBackToDefaults(t *testing.T) {
	t.Setenv("SCENARIO_TO_CLOUD_BUDGETS", "/nonexistent/budgets.json")
	dir := t.TempDir()
	wd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	d, path := LoadDetection()
	if path != "" || d != DefaultDetection() {
		t.Fatalf("detection = %+v from %q", d, path)
	}
	if d.StaleObservationAfterSeconds != 120 || d.CertificateExpiryWarningDays != 14 || d.AlertDetectionSecondsMax != 60 {
		t.Fatalf("defaults = %+v", d)
	}
}

func TestAlertEventShape(t *testing.T) {
	alert := perfbudget.Alert{CheckID: "x", IncidentID: "d:x", DetectedAt: time.Date(2026, 9, 9, 1, 2, 3, 0, time.UTC)}
	ev := alertEvent(alert)
	if ev.Source != AlertSource || ev.EventType != perfbudget.AlertEventType || !ev.Occurred.Equal(alert.DetectedAt) {
		t.Fatalf("event = %+v", ev)
	}
	if _, err := json.Marshal(ev.Payload); err != nil {
		t.Fatal(err)
	}
}
