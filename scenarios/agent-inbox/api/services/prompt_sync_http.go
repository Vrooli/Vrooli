// Package services provides business logic orchestration.
// This file implements HTTP communication with prompt-manager including
// retry logic, circuit breaking, and URL re-resolution.
package services

import (
	"agent-inbox/resilience"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// reResolveURL attempts to re-discover the prompt-manager URL via api-core discovery.
// Returns true if a new URL was found, false otherwise.
func (s *PromptSyncService) reResolveURL() bool {
	// Check env var override first
	if url := os.Getenv("PROMPT_MANAGER_URL"); url != "" {
		s.cfg.PromptManagerURL = url
		return true
	}

	resolver := discovery.NewResolver(discovery.ResolverConfig{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url, err := resolver.ResolveScenarioURLDefault(ctx, "prompt-manager")
	if err != nil {
		log.Printf("Prompt sync: re-resolution failed: %v", err)
		return false
	}

	if url != "" && url != s.cfg.PromptManagerURL {
		log.Printf("Prompt sync: re-resolved prompt-manager URL to %s", url)
		s.cfg.PromptManagerURL = url
		return true
	}
	return false
}

// doHTTPWithRetry performs an HTTP request with retry, circuit breaker, and URL re-resolution.
// On retry attempts > 1, it re-resolves the prompt-manager URL.
// 4xx responses are marked as permanent (non-retryable) errors.
func (s *PromptSyncService) doHTTPWithRetry(method, path string, body []byte) (*http.Response, error) {
	var resp *http.Response
	ctx := context.Background()

	err := resilience.Retry(ctx, s.retryCfg, func(ctx context.Context, attempt int) error {
		if attempt > 1 {
			s.reResolveURL()
		}

		if s.cfg.PromptManagerURL == "" {
			return fmt.Errorf("prompt-manager URL not available")
		}

		var reqBody io.Reader
		if body != nil {
			reqBody = bytes.NewReader(body)
		}

		req, err := http.NewRequest(method, s.cfg.PromptManagerURL+path, reqBody)
		if err != nil {
			return resilience.Permanent(err)
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		doReq := func(ctx context.Context) error {
			var doErr error
			resp, doErr = s.client.Do(req)
			return doErr
		}

		if s.cb != nil {
			err = s.cb.Execute(ctx, doReq)
		} else {
			err = doReq(ctx)
		}
		if err != nil {
			return err
		}

		// Mark 4xx as permanent
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return resilience.Permanent(fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody)))
		}

		return nil
	})

	return resp, err
}

// ensurePromptManagerURL checks that the prompt-manager URL is available,
// attempting re-resolution if needed. Returns an error if unavailable.
func (s *PromptSyncService) ensurePromptManagerURL() error {
	if s.cfg.PromptManagerURL == "" {
		s.reResolveURL()
		if s.cfg.PromptManagerURL == "" {
			return fmt.Errorf("prompt-manager URL not available")
		}
	}
	return nil
}

// ensureEnabledAndReachable checks that sync is enabled and the prompt-manager
// URL is available. Returns an error if either condition is not met.
func (s *PromptSyncService) ensureEnabledAndReachable() error {
	if !s.cfg.Enabled {
		return fmt.Errorf("prompt-manager sync is not enabled")
	}
	return s.ensurePromptManagerURL()
}
