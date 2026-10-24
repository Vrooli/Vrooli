package services

import (
	"bytes"
	"context"
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

func TestFixCacheWithMockedTime(t *testing.T) {
	t.Run("FixCacheExpiresAfterTTL", func(t *testing.T) {
		mockRepo := &mockAppRepository{}
		mockTime := newMockTimeProvider(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
		service := NewAppServiceWithOptions(mockRepo, nil, mockTime.now)

		// Create cache entry
		cacheKey := "test-scenario"
		entry := &fixCacheEntry{
			active:    []AppFixSummary{{ID: "fix/test", Kind: "fix", Name: "test", Title: "Test"}},
			archived:  []AppFixSummary{},
			scenario:  "test-scenario",
			appID:     "test-app",
			fixesURL:  "/apps/swarm-manager/proxy/",
			fetchedAt: mockTime.now().UTC(),
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
			t.Error("Fix cache should be fresh immediately")
		}

		// Advance time past fix cache TTL (30s)
		mockTime.advance(35 * time.Second)

		// Verify cache is now stale
		service.issueCacheMu.RLock()
		age = mockTime.now().UTC().Sub(entry.fetchedAt)
		isStale := age >= service.issueCacheTTL
		service.issueCacheMu.RUnlock()

		if !isStale {
			t.Error("Fix cache should be stale after 35 seconds (TTL is 30s)")
		}
	})
}

// =============================================================================
// HTTP Error Scenario Tests
// =============================================================================

func TestHTTPErrorScenarios(t *testing.T) {
	t.Run("SwarmManagerHTTPTimeout", func(t *testing.T) {
		mockRepo := &mockAppRepository{}
		mockHTTP := &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("context deadline exceeded")
			},
		}
		service := NewAppServiceWithOptions(mockRepo, mockHTTP, nil)
		service.scenarioURL = func(context.Context, string) (string, error) {
			return "http://localhost:8080", nil
		}

		_, err := service.submitFixToSwarmManager(context.Background(), map[string]interface{}{
			"name": "test-fix", "title": "Test Fix", "kind": "fix",
		}, nil)

		if err == nil {
			t.Error("Expected error when HTTP client times out")
		}
		if !strings.Contains(err.Error(), "failed to call swarm-manager") {
			t.Errorf("Expected timeout error message, got: %v", err)
		}
	})

	t.Run("SwarmManager500Error", func(t *testing.T) {
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
		service.scenarioURL = func(context.Context, string) (string, error) {
			return "http://localhost:8080", nil
		}

		_, err := service.submitFixToSwarmManager(context.Background(), map[string]interface{}{
			"name": "test-fix", "title": "Test Fix", "kind": "fix",
		}, nil)

		if err == nil {
			t.Error("Expected error when Swarm Manager returns 500")
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("Expected Swarm Manager 500 status error, got: %v", err)
		}
	})

	t.Run("SwarmManagerInvalidJSON", func(t *testing.T) {
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
		service.scenarioURL = func(context.Context, string) (string, error) {
			return "http://localhost:8080", nil
		}

		_, err := service.submitFixToSwarmManager(context.Background(), map[string]interface{}{
			"name": "test-fix", "title": "Test Fix", "kind": "fix",
		}, nil)

		if err == nil {
			t.Error("Expected error when response is invalid JSON")
		}
		if !strings.Contains(err.Error(), "failed to decode") {
			t.Errorf("Expected JSON decode error, got: %v", err)
		}
	})

	t.Run("SwarmManagerSuccessButNoItemIdentity", func(t *testing.T) {
		mockRepo := &mockAppRepository{}
		mockHTTP := &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"item": {}}`)),
				}, nil
			},
		}
		service := NewAppServiceWithOptions(mockRepo, mockHTTP, nil)
		service.scenarioURL = func(context.Context, string) (string, error) {
			return "http://localhost:8080", nil
		}

		_, err := service.submitFixToSwarmManager(context.Background(), map[string]interface{}{
			"name": "test-fix", "title": "Test Fix", "kind": "fix",
		}, nil)
		if err == nil {
			t.Error("Expected error when item identity is missing")
		}
	})
}

// =============================================================================
// Swarm Manager Integration Tests
// =============================================================================

func TestSwarmManagerFixIntegration(t *testing.T) {
	t.Run("SubmitFixWithMultipartEvidence", func(t *testing.T) {
		mockRepo := &mockAppRepository{}
		requestReceived := false
		var receivedItem string
		var receivedManifest string

		mockHTTP := &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				requestReceived = true

				// Verify request details
				if req.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", req.Method)
				}
				if !strings.Contains(req.URL.String(), "/api/v1/backlog") {
					t.Errorf("Expected /api/v1/backlog endpoint, got %s", req.URL.String())
				}
				if !strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data") {
					t.Error("Expected multipart/form-data")
				}

				bodyBytes, _ := io.ReadAll(req.Body)
				bodyText := string(bodyBytes)
				receivedItem = bodyText
				receivedManifest = bodyText

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"item":{"name":"test-fix","title":"Test Fix","kind":"fix","status":"backlog"}}`)),
				}, nil
			},
		}

		service := NewAppServiceWithOptions(mockRepo, mockHTTP, nil)
		service.scenarioURL = func(context.Context, string) (string, error) {
			return "http://localhost:8080", nil
		}

		result, err := service.submitFixToSwarmManager(context.Background(), map[string]interface{}{
			"name": "test-fix", "title": "Test Fix", "kind": "fix",
		}, []swarmEvidenceFile{{Path: "evidence/report.json", Content: []byte(`{"ok":true}`), ContentType: "application/json"}})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !requestReceived {
			t.Error("HTTP request was not made")
		}

		if result == nil {
			t.Fatal("Expected non-nil result")
		}

		if result.Kind != "fix" || result.Name != "test-fix" {
			t.Errorf("Expected fix/test-fix, got %#v", result)
		}

		if !strings.Contains(receivedItem, `"title":"Test Fix"`) {
			t.Error("Multipart item did not contain fix title")
		}
		if !strings.Contains(receivedManifest, `"path":"evidence/report.json"`) {
			t.Error("Multipart manifest did not contain evidence path")
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
