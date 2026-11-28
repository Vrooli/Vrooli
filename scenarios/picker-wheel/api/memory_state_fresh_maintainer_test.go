package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFreshMaintainerHistoryEndpointRetainsConcurrentSpins(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()
	resetTestState()
	initTestWheels()

	const concurrency = 32
	type spinResponse struct {
		status int
	}
	start := make(chan struct{})
	results := make(chan spinResponse, concurrency)
	router := setupTestRouter()

	for i := 0; i < concurrency; i++ {
		sessionID := fmt.Sprintf("fresh-maintainer-session-%02d", i)
		payload, err := json.Marshal(map[string]string{
			"wheel_id":   "yes-or-no",
			"session_id": sessionID,
		})
		if err != nil {
			t.Fatalf("marshal spin request: %v", err)
		}

		go func(payload []byte) {
			<-start
			request := httptest.NewRequest(http.MethodPost, "/api/spin", bytes.NewReader(payload))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			results <- spinResponse{status: response.Code}
		}(payload)
	}
	close(start)

	for i := 0; i < concurrency; i++ {
		response := <-results
		if response.status != http.StatusOK {
			t.Errorf("spin status = %d, want %d", response.status, http.StatusOK)
		}
	}

	historyRequest := httptest.NewRequest(http.MethodGet, "/api/history", nil)
	historyResponse := httptest.NewRecorder()
	router.ServeHTTP(historyResponse, historyRequest)
	if historyResponse.Code != http.StatusOK {
		t.Fatalf("history status = %d, want %d", historyResponse.Code, http.StatusOK)
	}

	var historyResults []SpinResult
	if err := json.Unmarshal(historyResponse.Body.Bytes(), &historyResults); err != nil {
		t.Fatalf("decode history response: %v", err)
	}
	if len(historyResults) != concurrency {
		t.Fatalf("history contains %d entries, want exactly %d", len(historyResults), concurrency)
	}

	expectedSessions := make(map[string]bool, concurrency)
	for i := 0; i < concurrency; i++ {
		expectedSessions[fmt.Sprintf("fresh-maintainer-session-%02d", i)] = true
	}
	seenSessions := make(map[string]int, concurrency)
	for _, result := range historyResults {
		if _, expected := expectedSessions[result.SessionID]; !expected {
			t.Errorf("history contains unexpected session ID %q", result.SessionID)
		}
		seenSessions[result.SessionID]++
		if result.WheelID != "yes-or-no" {
			t.Errorf("history wheel ID = %q, want %q", result.WheelID, "yes-or-no")
		}
		if result.Result != "YES! ✅" && result.Result != "NO ❌" {
			t.Errorf("history result = %q, want one of the yes-or-no wheel labels", result.Result)
		}
		if result.Timestamp.IsZero() {
			t.Errorf("history result for session %q has zero timestamp", result.SessionID)
		}
	}
	for sessionID := range expectedSessions {
		if got := seenSessions[sessionID]; got != 1 {
			t.Errorf("history contains session %q %d times, want exactly once", sessionID, got)
		}
	}
}
