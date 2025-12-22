package testgenie

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/browser-automation-studio/database"
)

type stubExecutionReader struct {
	status string
	err    error
}

func (s *stubExecutionReader) GetExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &database.ExecutionIndex{
		ID:     id,
		Status: s.status,
	}, nil
}

func TestSeedCleanupManager_CleansUpCompletedExecution(t *testing.T) {
	execID := uuid.New()
	seedScenario := "browser-automation-studio"
	cleanupToken := "cleanup-token"

	errCh := make(chan error, 1)
	cleanupCalled := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			errCh <- fmt.Errorf("expected POST, got %s", r.Method)
			http.Error(w, "invalid method", http.StatusMethodNotAllowed)
			return
		}
		if !strings.Contains(r.URL.Path, "/api/v1/scenarios/"+seedScenario+"/playbooks/seed/cleanup") {
			errCh <- fmt.Errorf("unexpected cleanup path: %s", r.URL.Path)
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		var payload struct {
			CleanupToken string `json:"cleanup_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			errCh <- fmt.Errorf("decode cleanup payload: %w", err)
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if payload.CleanupToken != cleanupToken {
			errCh <- fmt.Errorf("expected cleanup_token %q, got %q", cleanupToken, payload.CleanupToken)
			http.Error(w, "invalid token", http.StatusBadRequest)
			return
		}
		cleanupCalled <- struct{}{}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":   "cleaned",
			"scenario": seedScenario,
			"run_id":   "run-1",
		})
	}))
	defer server.Close()

	resolver := resolverForTestServer(t, server.URL)
	client := NewClient(resolver, server.Client())

	manager := &SeedCleanupManager{
		exec:          &stubExecutionReader{status: database.ExecutionStatusCompleted},
		client:        client,
		log:           logrus.New(),
		pollInterval:  time.Second,
		maxWait:       time.Minute,
		retryInterval: 0,
		jobs:          map[uuid.UUID]*seedCleanupJob{},
	}

	if err := manager.Schedule(execID.String(), seedScenario, cleanupToken); err != nil {
		t.Fatalf("schedule cleanup: %v", err)
	}

	manager.processJobs()

	select {
	case err := <-errCh:
		t.Fatalf("cleanup handler error: %v", err)
	default:
	}
	select {
	case <-cleanupCalled:
	default:
		t.Fatalf("expected cleanup to be called")
	}
	if len(manager.jobs) != 0 {
		t.Fatalf("expected cleanup job to be removed")
	}
}

func resolverForTestServer(t *testing.T, rawURL string) *discovery.Resolver {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	host, port, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}

	return discovery.NewResolver(discovery.ResolverConfig{
		Host:   host,
		Scheme: parsed.Scheme,
		CommandRunner: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte(port), nil
		},
	})
}
