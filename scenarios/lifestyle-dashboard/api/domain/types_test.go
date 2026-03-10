// Package domain contains the core business entities for the lifestyle dashboard.
package domain

import (
	"encoding/json"
	"testing"
)

// TestEventSerialization validates Event JSON serialization
// [REQ:LD-EVENT-SCHEMA] Event type serialization
func TestEventSerialization(t *testing.T) {
	hypothesisID := "550e8400-e29b-41d4-a716-446655440000"
	event := Event{
		ID:             "123e4567-e89b-12d3-a456-426614174000",
		Timestamp:      "2026-03-10T12:00:00Z",
		Domain:         "nootropics",
		EventType:      "supplement.taken",
		Payload:        json.RawMessage(`{"name": "magnesium", "dose_mg": 400}`),
		IsIntervention: true,
		HypothesisID:   &hypothesisID,
		CreatedAt:      "2026-03-10T12:00:00Z",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal Event: %v", err)
	}

	var unmarshaled Event
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal Event: %v", err)
	}

	if unmarshaled.ID != event.ID {
		t.Errorf("ID mismatch: expected %s, got %s", event.ID, unmarshaled.ID)
	}
	if unmarshaled.Domain != event.Domain {
		t.Errorf("Domain mismatch: expected %s, got %s", event.Domain, unmarshaled.Domain)
	}
	if unmarshaled.IsIntervention != event.IsIntervention {
		t.Error("IsIntervention should be preserved")
	}
	if unmarshaled.HypothesisID == nil || *unmarshaled.HypothesisID != *event.HypothesisID {
		t.Error("HypothesisID should be preserved")
	}
}

// TestDomainSerialization validates Domain JSON serialization
// [REQ:LD-DOMAIN-REGISTER] Domain type serialization
func TestDomainSerialization(t *testing.T) {
	lastHealth := "2026-03-10T11:00:00Z"
	dom := Domain{
		Name:         "sleep-tracker",
		DisplayName:  "Sleep Tracker",
		Description:  "Track sleep patterns",
		Capabilities: []string{"track_sleep", "analyze_quality"},
		Status:       "active",
		HealthURL:    "http://localhost:8080/health",
		LastHealthAt: &lastHealth,
		RegisteredAt: "2026-03-10T10:00:00Z",
		UpdatedAt:    "2026-03-10T11:00:00Z",
	}

	data, err := json.Marshal(dom)
	if err != nil {
		t.Fatalf("Failed to marshal Domain: %v", err)
	}

	var unmarshaled Domain
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal Domain: %v", err)
	}

	if unmarshaled.Name != dom.Name {
		t.Errorf("Name mismatch: expected %s, got %s", dom.Name, unmarshaled.Name)
	}
	if len(unmarshaled.Capabilities) != len(dom.Capabilities) {
		t.Errorf("Capabilities length mismatch: expected %d, got %d", len(dom.Capabilities), len(unmarshaled.Capabilities))
	}
	if unmarshaled.Status != dom.Status {
		t.Errorf("Status mismatch: expected %s, got %s", dom.Status, unmarshaled.Status)
	}
}

// TestCreateEventRequestSerialization validates CreateEventRequest JSON handling
// [REQ:LD-EVENT-SCHEMA] Request serialization
func TestCreateEventRequestSerialization(t *testing.T) {
	timestamp := "2026-03-10T12:00:00Z"
	req := CreateEventRequest{
		Domain:         "exercise",
		EventType:      "workout.completed",
		Payload:        json.RawMessage(`{"duration_min": 45, "type": "strength"}`),
		Timestamp:      &timestamp,
		IsIntervention: false,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal CreateEventRequest: %v", err)
	}

	var unmarshaled CreateEventRequest
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal CreateEventRequest: %v", err)
	}

	if unmarshaled.Domain != req.Domain {
		t.Errorf("Domain mismatch: expected %s, got %s", req.Domain, unmarshaled.Domain)
	}
	if unmarshaled.EventType != req.EventType {
		t.Errorf("EventType mismatch: expected %s, got %s", req.EventType, unmarshaled.EventType)
	}
	if unmarshaled.Timestamp == nil || *unmarshaled.Timestamp != *req.Timestamp {
		t.Error("Timestamp should be preserved")
	}
}

// TestResponseTypes validates response type structures
// [REQ:LD-QUERY-AGGREGATE] Response types
func TestResponseTypes(t *testing.T) {
	// Test EventsResponse
	eventsResp := EventsResponse{
		Events: []Event{
			{ID: "1", Domain: "test", EventType: "test.event"},
		},
		Count: 1,
	}
	data, err := json.Marshal(eventsResp)
	if err != nil {
		t.Fatalf("Failed to marshal EventsResponse: %v", err)
	}
	if len(data) == 0 {
		t.Error("EventsResponse should serialize to non-empty JSON")
	}

	// Test DomainsResponse
	domainsResp := DomainsResponse{
		Domains: []Domain{
			{Name: "test", DisplayName: "Test"},
		},
		Count: 1,
	}
	data, err = json.Marshal(domainsResp)
	if err != nil {
		t.Fatalf("Failed to marshal DomainsResponse: %v", err)
	}
	if len(data) == 0 {
		t.Error("DomainsResponse should serialize to non-empty JSON")
	}

	// Test SummaryResponse
	summaryResp := SummaryResponse{
		TotalEvents:    100,
		ActiveDomains:  5,
		EventsByDomain: []DomainCount{{Domain: "sleep", Count: 50}},
		LastEventAt:    "2026-03-10T12:00:00Z",
	}
	data, err = json.Marshal(summaryResp)
	if err != nil {
		t.Fatalf("Failed to marshal SummaryResponse: %v", err)
	}
	if len(data) == 0 {
		t.Error("SummaryResponse should serialize to non-empty JSON")
	}

	// Test ErrorResponse
	errorResp := ErrorResponse{
		Error:   true,
		Message: "something went wrong",
	}
	data, err = json.Marshal(errorResp)
	if err != nil {
		t.Fatalf("Failed to marshal ErrorResponse: %v", err)
	}
	if len(data) == 0 {
		t.Error("ErrorResponse should serialize to non-empty JSON")
	}
}
