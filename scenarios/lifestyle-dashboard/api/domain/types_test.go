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

// TestHealthCheckResponse validates health check response serialization.
// [REQ:LD-DOMAIN-HEALTH] Health check response type.
func TestHealthCheckResponse(t *testing.T) {
	resp := HealthCheckResponse{
		Domain:    "sleep-tracker",
		Status:    "healthy",
		LastCheck: "2026-03-10T12:00:00Z",
		Message:   "",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal HealthCheckResponse: %v", err)
	}
	if len(data) == 0 {
		t.Error("HealthCheckResponse should serialize to non-empty JSON")
	}

	var unmarshaled HealthCheckResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal HealthCheckResponse: %v", err)
	}
	if unmarshaled.Domain != resp.Domain {
		t.Errorf("Domain mismatch: expected %s, got %s", resp.Domain, unmarshaled.Domain)
	}
	if unmarshaled.Status != resp.Status {
		t.Errorf("Status mismatch: expected %s, got %s", resp.Status, unmarshaled.Status)
	}
}

// TestTimelineResponse validates timeline response serialization.
// [REQ:LD-QUERY-AGGREGATE] Timeline response type.
func TestTimelineResponse(t *testing.T) {
	resp := TimelineResponse{
		Timeline: []TimelineEntry{
			{Day: "2026-03-10", Domain: "sleep", Count: 5},
			{Day: "2026-03-10", Domain: "exercise", Count: 2},
		},
		Days: "7",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal TimelineResponse: %v", err)
	}

	var unmarshaled TimelineResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal TimelineResponse: %v", err)
	}
	if len(unmarshaled.Timeline) != 2 {
		t.Errorf("Expected 2 timeline entries, got %d", len(unmarshaled.Timeline))
	}
	if unmarshaled.Days != "7" {
		t.Errorf("Expected days '7', got '%s'", unmarshaled.Days)
	}
}

// TestLifestyleScore validates lifestyle score serialization.
// [REQ:LD-UI-SCORE] Lifestyle score type.
func TestLifestyleScore(t *testing.T) {
	score := LifestyleScore{
		Score: 75,
		Date:  "2026-03-10",
		DomainScores: []DomainScore{
			{Domain: "sleep", DisplayName: "Sleep", Score: 80, Weight: 0.5, EventCount: 10},
			{Domain: "exercise", DisplayName: "Exercise", Score: 70, Weight: 0.5, EventCount: 5},
		},
		Trend:               "up",
		ChangeFromYesterday: 5,
		DataQuality:         "good",
		Message:             "Great progress!",
	}

	data, err := json.Marshal(score)
	if err != nil {
		t.Fatalf("Failed to marshal LifestyleScore: %v", err)
	}

	var unmarshaled LifestyleScore
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal LifestyleScore: %v", err)
	}
	if unmarshaled.Score != 75 {
		t.Errorf("Expected score 75, got %d", unmarshaled.Score)
	}
	if len(unmarshaled.DomainScores) != 2 {
		t.Errorf("Expected 2 domain scores, got %d", len(unmarshaled.DomainScores))
	}
	if unmarshaled.Trend != "up" {
		t.Errorf("Expected trend 'up', got '%s'", unmarshaled.Trend)
	}
}

// TestScoreResponse validates score response serialization.
// [REQ:LD-UI-SCORE] Score response type.
func TestScoreResponse(t *testing.T) {
	resp := ScoreResponse{
		Current: LifestyleScore{
			Score:       75,
			Date:        "2026-03-10",
			Trend:       "up",
			DataQuality: "good",
		},
		History: []ScoreHistoryEntry{
			{Date: "2026-03-09", Score: 70},
			{Date: "2026-03-08", Score: 65},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal ScoreResponse: %v", err)
	}

	var unmarshaled ScoreResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal ScoreResponse: %v", err)
	}
	if unmarshaled.Current.Score != 75 {
		t.Errorf("Expected current score 75, got %d", unmarshaled.Current.Score)
	}
	if len(unmarshaled.History) != 2 {
		t.Errorf("Expected 2 history entries, got %d", len(unmarshaled.History))
	}
}

// TestStorageTypes validates storage management type serialization.
// [REQ:LD-UI-STORAGE] Storage management types.
func TestStorageTypes(t *testing.T) {
	// StorageInfo
	info := StorageInfo{
		DatabaseSizeBytes: 1024000,
		TotalEvents:       500,
		TotalDomains:      5,
		EventsByDomain: []DomainStorageInfo{
			{Domain: "sleep", DisplayName: "Sleep", EventCount: 200},
		},
		OldestEvent: "2026-01-01T00:00:00Z",
		NewestEvent: "2026-03-10T12:00:00Z",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Failed to marshal StorageInfo: %v", err)
	}

	var unmarshaled StorageInfo
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal StorageInfo: %v", err)
	}
	if unmarshaled.TotalEvents != 500 {
		t.Errorf("Expected 500 events, got %d", unmarshaled.TotalEvents)
	}

	// CleanupResponse
	cleanup := CleanupResponse{
		DeletedEvents: 100,
		Message:       "Cleared all events",
	}

	data, err = json.Marshal(cleanup)
	if err != nil {
		t.Fatalf("Failed to marshal CleanupResponse: %v", err)
	}

	var unmarshaledCleanup CleanupResponse
	if err := json.Unmarshal(data, &unmarshaledCleanup); err != nil {
		t.Fatalf("Failed to unmarshal CleanupResponse: %v", err)
	}
	if unmarshaledCleanup.DeletedEvents != 100 {
		t.Errorf("Expected 100 deleted events, got %d", unmarshaledCleanup.DeletedEvents)
	}
}

// TestBriefTypes validates brief type serialization.
// [REQ:LD-BRIEF-MORNING] [REQ:LD-BRIEF-EVENING] Brief types.
func TestBriefTypes(t *testing.T) {
	score := 75
	brief := Brief{
		Type:        "morning",
		Date:        "2026-03-10",
		Summary:     "Good morning! Today looks productive.",
		Score:       &score,
		GeneratedAt: "2026-03-10T07:00:00Z",
		Sections: []BriefSection{
			{
				Domain:      "sleep",
				DisplayName: "Sleep",
				Priority:    1,
				Items:       []string{"Good sleep quality last night."},
				EventCount:  3,
			},
		},
	}

	data, err := json.Marshal(brief)
	if err != nil {
		t.Fatalf("Failed to marshal Brief: %v", err)
	}

	var unmarshaled Brief
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal Brief: %v", err)
	}
	if unmarshaled.Type != "morning" {
		t.Errorf("Expected type 'morning', got '%s'", unmarshaled.Type)
	}
	if len(unmarshaled.Sections) != 1 {
		t.Errorf("Expected 1 section, got %d", len(unmarshaled.Sections))
	}

	// BriefResponse
	resp := BriefResponse{
		Brief: brief,
		Config: BriefConfig{
			MorningHour: 7,
			EveningHour: 21,
		},
	}

	data, err = json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal BriefResponse: %v", err)
	}

	var unmarshaledResp BriefResponse
	if err := json.Unmarshal(data, &unmarshaledResp); err != nil {
		t.Fatalf("Failed to unmarshal BriefResponse: %v", err)
	}
	if unmarshaledResp.Config.MorningHour != 7 {
		t.Errorf("Expected morning hour 7, got %d", unmarshaledResp.Config.MorningHour)
	}
}
