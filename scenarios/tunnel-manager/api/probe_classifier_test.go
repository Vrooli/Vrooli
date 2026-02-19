package main

import (
	"testing"
)

// [REQ:PROBE-003] Probe result classification - up
func TestClassifyRouteUp(t *testing.T) {
	results := []ProbeResult{
		{RouteID: 1, Subdomain: "app", ProbeType: "internal", Status: "up"},
		{RouteID: 1, Subdomain: "app", ProbeType: "external", Status: "up"},
	}
	classifications := ClassifyProbeResults(results)
	if len(classifications) != 1 {
		t.Fatalf("expected 1 classification, got %d", len(classifications))
	}
	if classifications[0].Status != "up" {
		t.Errorf("status = %q, want up", classifications[0].Status)
	}
	if classifications[0].Assessment == "" {
		t.Error("expected non-empty assessment")
	}
}

// [REQ:PROBE-003] Probe result classification - tunnel-issue
func TestClassifyTunnelIssue(t *testing.T) {
	results := []ProbeResult{
		{RouteID: 1, Subdomain: "app", ProbeType: "internal", Status: "up"},
		{RouteID: 1, Subdomain: "app", ProbeType: "external", Status: "down"},
	}
	classifications := ClassifyProbeResults(results)
	if len(classifications) != 1 {
		t.Fatalf("expected 1 classification, got %d", len(classifications))
	}
	if classifications[0].Status != "tunnel-issue" {
		t.Errorf("status = %q, want tunnel-issue", classifications[0].Status)
	}
}

// [REQ:PROBE-003] Probe result classification - scenario-down
func TestClassifyScenarioDown(t *testing.T) {
	results := []ProbeResult{
		{RouteID: 1, Subdomain: "app", ProbeType: "internal", Status: "down"},
		{RouteID: 1, Subdomain: "app", ProbeType: "external", Status: "up"},
	}
	classifications := ClassifyProbeResults(results)
	if len(classifications) != 1 {
		t.Fatalf("expected 1 classification, got %d", len(classifications))
	}
	if classifications[0].Status != "scenario-down" {
		t.Errorf("status = %q, want scenario-down", classifications[0].Status)
	}
}

// [REQ:PROBE-003] Probe result classification - unknown
func TestClassifyUnknown(t *testing.T) {
	results := []ProbeResult{
		{RouteID: 1, Subdomain: "app", ProbeType: "internal", Status: "down"},
		{RouteID: 1, Subdomain: "app", ProbeType: "external", Status: "down"},
	}
	classifications := ClassifyProbeResults(results)
	if len(classifications) != 1 {
		t.Fatalf("expected 1 classification, got %d", len(classifications))
	}
	if classifications[0].Status != "unknown" {
		t.Errorf("status = %q, want unknown", classifications[0].Status)
	}
}

// [REQ:PROBE-003] Classification handles multiple routes
func TestClassifyMultipleRoutes(t *testing.T) {
	results := []ProbeResult{
		{RouteID: 1, Subdomain: "app1", ProbeType: "internal", Status: "up"},
		{RouteID: 1, Subdomain: "app1", ProbeType: "external", Status: "up"},
		{RouteID: 2, Subdomain: "app2", ProbeType: "internal", Status: "up"},
		{RouteID: 2, Subdomain: "app2", ProbeType: "external", Status: "down"},
	}
	classifications := ClassifyProbeResults(results)
	if len(classifications) != 2 {
		t.Fatalf("expected 2 classifications, got %d", len(classifications))
	}

	statusByRoute := make(map[string]string)
	for _, c := range classifications {
		statusByRoute[c.Subdomain] = c.Status
	}
	if statusByRoute["app1"] != "up" {
		t.Errorf("app1 status = %q, want up", statusByRoute["app1"])
	}
	if statusByRoute["app2"] != "tunnel-issue" {
		t.Errorf("app2 status = %q, want tunnel-issue", statusByRoute["app2"])
	}
}
