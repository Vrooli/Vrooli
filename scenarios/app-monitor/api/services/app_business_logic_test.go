package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"app-monitor-api/repository"
)

// =============================================================================
// Mock HTTP Client
// =============================================================================

type mockHTTPClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.doFunc != nil {
		return m.doFunc(req)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(`{"success": true}`)),
	}, nil
}

// =============================================================================
// Mock Time Provider
// =============================================================================

type mockTimeProvider struct {
	currentTime time.Time
	calls       int
}

func (m *mockTimeProvider) now() time.Time {
	m.calls++
	return m.currentTime
}

func (m *mockTimeProvider) advance(d time.Duration) {
	m.currentTime = m.currentTime.Add(d)
}

func newMockTimeProvider(t time.Time) *mockTimeProvider {
	return &mockTimeProvider{
		currentTime: t,
	}
}

// =============================================================================
// Cache Expiration Tests
// =============================================================================

func TestCacheExpirationWithMockedTime(t *testing.T) {
	t.Run("PartialCacheExpiresAfterTTL", func(t *testing.T) {
		mockRepo := &mockAppRepository{}
		mockTime := newMockTimeProvider(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
		service := NewAppServiceWithOptions(mockRepo, nil, mockTime.now)

		// Populate cache with partial data
		service.cache.mu.Lock()
		service.cache.data = []repository.App{{ID: "test", Name: "Test App"}}
		service.cache.timestamp = mockTime.now()
		service.cache.isPartial = true
		service.cache.mu.Unlock()

		// Verify cache is fresh
		service.cache.mu.RLock()
		age := mockTime.now().Sub(service.cache.timestamp)
		isFresh := age < partialCacheTTL
		service.cache.mu.RUnlock()

		if !isFresh {
			t.Error("Cache should be fresh immediately after being set")
		}

		// Advance time past partial cache TTL (45s) but not full TTL (90s)
		mockTime.advance(50 * time.Second)

		// Verify cache is now stale for partial
		service.cache.mu.RLock()
		age = mockTime.now().Sub(service.cache.timestamp)
		isStalePartial := age >= partialCacheTTL
		isStaleFull := age >= orchestratorCacheTTL
		service.cache.mu.RUnlock()

		if !isStalePartial {
			t.Error("Partial cache should be stale after 50 seconds")
		}
		if isStaleFull {
			t.Error("Full cache should not be stale yet (only 50s elapsed, TTL is 90s)")
		}
	})

	t.Run("FullCacheExpiresAfterTTL", func(t *testing.T) {
		mockRepo := &mockAppRepository{}
		mockTime := newMockTimeProvider(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
		service := NewAppServiceWithOptions(mockRepo, nil, mockTime.now)

		// Populate cache with full data
		service.cache.mu.Lock()
		service.cache.data = []repository.App{{ID: "test", Name: "Test App"}}
		service.cache.timestamp = mockTime.now()
		service.cache.isPartial = false
		service.cache.mu.Unlock()

		// Advance time past full cache TTL
		mockTime.advance(100 * time.Second)

		// Verify cache is now stale
		service.cache.mu.RLock()
		age := mockTime.now().Sub(service.cache.timestamp)
		isStale := age >= orchestratorCacheTTL
		service.cache.mu.RUnlock()

		if !isStale {
			t.Error("Full cache should be stale after 100 seconds (TTL is 90s)")
		}
	})

	t.Run("InvalidateCacheClearsTimestamp", func(t *testing.T) {
		mockRepo := &mockAppRepository{}
		mockTime := newMockTimeProvider(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
		service := NewAppServiceWithOptions(mockRepo, nil, mockTime.now)

		// Populate cache
		service.cache.mu.Lock()
		service.cache.data = []repository.App{{ID: "test", Name: "Test App"}}
		service.cache.timestamp = mockTime.now()
		service.cache.mu.Unlock()

		// Invalidate cache
		service.invalidateCache()

		// Verify timestamp was cleared
		service.cache.mu.RLock()
		isEmpty := service.cache.timestamp.IsZero()
		service.cache.mu.RUnlock()

		if !isEmpty {
			t.Error("Cache timestamp should be zero after invalidation")
		}
	})
}

func TestIssueCacheWithMockedTime(t *testing.T) {
	t.Run("IssueCacheExpiresAfterTTL", func(t *testing.T) {
		mockRepo := &mockAppRepository{}
		mockTime := newMockTimeProvider(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
		service := NewAppServiceWithOptions(mockRepo, nil, mockTime.now)

		// Create cache entry
		cacheKey := "test-scenario"
		entry := &issueCacheEntry{
			issues:      []AppIssueSummary{{ID: "issue-1", Title: "Test"}},
			scenario:    "test-scenario",
			appID:       "test-app",
			trackerURL:  "http://localhost:8080",
			fetchedAt:   mockTime.now().UTC(),
			openCount:   1,
			activeCount: 1,
			totalCount:  1,
		}

		service.issueCacheMu.Lock()
		service.issueCache[cacheKey] = entry
		service.issueCacheMu.Unlock()

		// Verify cache is fresh
		service.issueCacheMu.RLock()
		age := mockTime.now().UTC().Sub(entry.fetchedAt)
		isFresh := age < service.issueCacheTTL
		service.issueCacheMu.RUnlock()

		if !isFresh {
			t.Error("Issue cache should be fresh immediately")
		}

		// Advance time past issue cache TTL (30s)
		mockTime.advance(35 * time.Second)

		// Verify cache is now stale
		service.issueCacheMu.RLock()
		age = mockTime.now().UTC().Sub(entry.fetchedAt)
		isStale := age >= service.issueCacheTTL
		service.issueCacheMu.RUnlock()

		if !isStale {
			t.Error("Issue cache should be stale after 35 seconds (TTL is 30s)")
		}
	})
}

// =============================================================================
// HTTP Error Scenario Tests
// =============================================================================

func TestHTTPErrorScenarios(t *testing.T) {
	t.Run("IssueTrackerHTTPTimeout", func(t *testing.T) {
		mockRepo := &mockAppRepository{}
		mockHTTP := &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("context deadline exceeded")
			},
		}
		service := NewAppServiceWithOptions(mockRepo, mockHTTP, nil)

		// Try to submit issue (this would fail during HTTP call)
		_, err := service.submitIssueToTracker(context.Background(), 8080, map[string]interface{}{
			"title": "Test Issue",
		})

		if err == nil {
			t.Error("Expected error when HTTP client times out")
		}
		if !strings.Contains(err.Error(), "failed to call app-issue-tracker") {
			t.Errorf("Expected timeout error message, got: %v", err)
		}
	})

	t.Run("IssueTracker500Error", func(t *testing.T) {
		mockRepo := &mockAppRepository{}
		mockHTTP := &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBufferString(`{"error": "Internal server error"}`)),
				}, nil
			},
		}
		service := NewAppServiceWithOptions(mockRepo, mockHTTP, nil)

		_, err := service.submitIssueToTracker(context.Background(), 8080, map[string]interface{}{
			"title": "Test Issue",
		})

		if err == nil {
			t.Error("Expected error when issue tracker returns 500")
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("Expected 500 status error, got: %v", err)
		}
	})

	t.Run("IssueTrackerInvalidJSON", func(t *testing.T) {
		mockRepo := &mockAppRepository{}
		mockHTTP := &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`invalid json`)),
				}, nil
			},
		}
		service := NewAppServiceWithOptions(mockRepo, mockHTTP, nil)

		_, err := service.submitIssueToTracker(context.Background(), 8080, map[string]interface{}{
			"title": "Test Issue",
		})

		if err == nil {
			t.Error("Expected error when response is invalid JSON")
		}
		if !strings.Contains(err.Error(), "failed to decode") {
			t.Errorf("Expected JSON decode error, got: %v", err)
		}
	})

	t.Run("IssueTrackerSuccessButNoIssueID", func(t *testing.T) {
		mockRepo := &mockAppRepository{}
		mockHTTP := &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"success": true, "message": "Created", "data": {}}`)),
				}, nil
			},
		}
		service := NewAppServiceWithOptions(mockRepo, mockHTTP, nil)

		result, err := service.submitIssueToTracker(context.Background(), 8080, map[string]interface{}{
			"title": "Test Issue",
		})

		if err != nil {
			t.Errorf("Should not error when success=true, got: %v", err)
		}
		if result == nil {
			t.Fatal("Expected non-nil result")
		}
		if result.IssueID != "" {
			t.Errorf("Expected empty issue ID when not provided, got: %q", result.IssueID)
		}
		if result.Message == "" {
			t.Error("Expected result message to be populated")
		}
	})
}

// =============================================================================
// Issue Tracker Integration Tests
// =============================================================================

func TestIssueTrackerIntegration(t *testing.T) {
	t.Run("SubmitIssueWithAllFields", func(t *testing.T) {
		mockRepo := &mockAppRepository{}
		requestReceived := false
		var receivedPayload map[string]interface{}

		mockHTTP := &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				requestReceived = true

				// Verify request details
				if req.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", req.Method)
				}
				if !strings.Contains(req.URL.String(), "/api/v1/issues") {
					t.Errorf("Expected /api/v1/issues endpoint, got %s", req.URL.String())
				}
				if req.Header.Get("Content-Type") != "application/json" {
					t.Error("Expected Content-Type: application/json")
				}

				// Capture payload
				bodyBytes, _ := io.ReadAll(req.Body)
				_ = json.Unmarshal(bodyBytes, &receivedPayload)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(bytes.NewBufferString(`{
						"success": true,
						"message": "Issue created successfully",
						"data": {"issue_id": "ISSUE-123"}
					}`)),
				}, nil
			},
		}

		service := NewAppServiceWithOptions(mockRepo, mockHTTP, nil)

		result, err := service.submitIssueToTracker(context.Background(), 8080, map[string]interface{}{
			"title":       "Test Issue",
			"description": "This is a test",
			"app_id":      "test-app",
		})

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !requestReceived {
			t.Error("HTTP request was not made")
		}

		if result == nil {
			t.Fatal("Expected non-nil result")
		}

		if result.IssueID != "ISSUE-123" {
			t.Errorf("Expected issue ID 'ISSUE-123', got %q", result.IssueID)
		}

		if result.Message != "Issue created successfully" {
			t.Errorf("Expected message 'Issue created successfully', got %q", result.Message)
		}

		// Verify payload was sent
		if receivedPayload == nil {
			t.Error("Payload was not captured")
		} else {
			if title, ok := receivedPayload["title"].(string); !ok || title != "Test Issue" {
				t.Error("Payload did not contain correct title")
			}
		}
	})

	t.Run("ParseNestedIssueIDFormats", func(t *testing.T) {
		testCases := []struct {
			name            string
			responseBody    string
			expectedIssueID string
		}{
			{
				name:            "issue_id in data",
				responseBody:    `{"success": true, "data": {"issue_id": "DIRECT-123"}}`,
				expectedIssueID: "DIRECT-123",
			},
			{
				name:            "issueId in data",
				responseBody:    `{"success": true, "data": {"issueId": "CAMEL-456"}}`,
				expectedIssueID: "CAMEL-456",
			},
			{
				name:            "nested issue.id",
				responseBody:    `{"success": true, "data": {"issue": {"id": "NESTED-789"}}}`,
				expectedIssueID: "NESTED-789",
			},
			{
				name:            "no issue ID",
				responseBody:    `{"success": true, "data": {}}`,
				expectedIssueID: "",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mockRepo := &mockAppRepository{}
				mockHTTP := &mockHTTPClient{
					doFunc: func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(bytes.NewBufferString(tc.responseBody)),
						}, nil
					},
				}

				service := NewAppServiceWithOptions(mockRepo, mockHTTP, nil)
				result, err := service.submitIssueToTracker(context.Background(), 8080, map[string]interface{}{
					"title": "Test",
				})

				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				if result.IssueID != tc.expectedIssueID {
					t.Errorf("Expected issue ID %q, got %q", tc.expectedIssueID, result.IssueID)
				}
			})
		}
	})
}

// =============================================================================
// View Stats Tracking Tests
// =============================================================================

func TestViewStatsWithMockedTime(t *testing.T) {
	t.Run("RecordFirstView", func(t *testing.T) {
		mockRepo := &mockAppRepository{}
		mockTime := newMockTimeProvider(time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC))
		service := NewAppServiceWithOptions(mockRepo, nil, mockTime.now)

		// Record first view
		stats := service.recordAppViewInMemory("test-app", "test-scenario")

		if stats == nil {
			t.Fatal("Expected non-nil stats")
		}

		if stats.ViewCount != 1 {
			t.Errorf("Expected ViewCount=1, got %d", stats.ViewCount)
		}

		if stats.FirstViewed == nil {
			t.Error("Expected FirstViewed to be set")
		} else if !stats.FirstViewed.Equal(mockTime.now().UTC()) {
			t.Errorf("Expected FirstViewed to be %v, got %v", mockTime.now().UTC(), *stats.FirstViewed)
		}
	})

	t.Run("RecordMultipleViews", func(t *testing.T) {
		mockRepo := &mockAppRepository{}
		mockTime := newMockTimeProvider(time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC))
		service := NewAppServiceWithOptions(mockRepo, nil, mockTime.now)

		// Record first view
		service.recordAppViewInMemory("test-app", "test-scenario")

		// Advance time
		mockTime.advance(5 * time.Minute)

		// Record second view
		stats := service.recordAppViewInMemory("test-app", "test-scenario")

		if stats == nil {
			t.Fatal("Expected non-nil stats")
		}

		if stats.ViewCount != 2 {
			t.Errorf("Expected ViewCount=2, got %d", stats.ViewCount)
		}

		if stats.LastViewed == nil {
			t.Error("Expected LastViewed to be set")
		} else if !stats.LastViewed.Equal(mockTime.now().UTC()) {
			t.Errorf("Expected LastViewed to be updated to %v, got %v", mockTime.now().UTC(), *stats.LastViewed)
		}
	})
}
