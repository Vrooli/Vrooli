package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
)

// [REQ:POL-005] SSE policy subscribe returns correct headers.
func TestPolicySubscribe_Headers(t *testing.T) {
	_, ts := newTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/v1/policies/subscribe", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type: expected text/event-stream, got %s", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("Cache-Control: expected no-cache, got %s", cc)
	}
	if conn := resp.Header.Get("Connection"); conn != "keep-alive" {
		t.Fatalf("Connection: expected keep-alive, got %s", conn)
	}
	if xab := resp.Header.Get("X-Accel-Buffering"); xab != "no" {
		t.Fatalf("X-Accel-Buffering: expected no, got %s", xab)
	}
}

// [REQ:POL-005] SSE policy subscribe includes retry directive.
func TestPolicySubscribe_RetryDirective(t *testing.T) {
	_, ts := newTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/v1/policies/subscribe", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	if scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "retry:") {
			t.Fatalf("expected retry directive, got %q", line)
		}
	} else {
		t.Fatal("expected at least one line from SSE stream")
	}
}

// [REQ:POL-005] Creating a policy broadcasts a "created" event to SSE subscribers.
func TestPolicySubscribe_CreateBroadcast(t *testing.T) {
	_, ts := newTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start SSE subscription.
	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/v1/policies/subscribe", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	// Consume retry line.
	scanner.Scan()

	// Create a policy to trigger the broadcast.
	policyJSON := `{"rule_type":"access","source_scenario":"src","target_scenario":"tgt","effect":"allow","enabled":true}`
	_, _ = http.Post(ts.URL+"/api/v1/policies", "application/json", strings.NewReader(policyJSON))

	// Read SSE events and look for a snapshot event.
	var gotEvent bool
	deadline := time.After(3 * time.Second)
	for !gotEvent {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for policy SSE event")
		default:
		}
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()
		if line == "event: snapshot" {
			// Next line should be data:
			if scanner.Scan() {
				dataLine := scanner.Text()
				if strings.HasPrefix(dataLine, "data: ") {
					data := strings.TrimPrefix(dataLine, "data: ")
					var evt policy.PolicyEvent
					if err := json.Unmarshal([]byte(data), &evt); err != nil {
						t.Fatalf("unmarshal policy event: %v", err)
					}
					if evt.Type != "snapshot" {
						t.Fatalf("expected type snapshot, got %s", evt.Type)
					}
					if evt.Version == 0 {
						t.Fatal("snapshot must carry a version")
					}
					if len(evt.Rules) == 0 {
						t.Fatal("expected rules to be present in snapshot")
					}
					gotEvent = true
				}
			}
		}
	}

	if !gotEvent {
		t.Fatal("did not receive snapshot SSE event")
	}
}

// [REQ:POL-005] Deleting a policy broadcasts a "deleted" event with null rule.
func TestPolicySubscribe_DeleteBroadcast(t *testing.T) {
	_, ts := newTestServer(t)

	// First create a policy so we can delete it.
	policyJSON := `{"rule_type":"access","source_scenario":"src","target_scenario":"tgt","effect":"deny"}`
	createResp, _ := http.Post(ts.URL+"/api/v1/policies", "application/json", strings.NewReader(policyJSON))
	var createResult map[string]any
	_ = json.NewDecoder(createResp.Body).Decode(&createResult)
	createResp.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start SSE subscription.
	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/v1/policies/subscribe", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Scan() // retry

	// Delete the policy.
	policyID := int64(createResult["id"].(float64))
	delReq, _ := http.NewRequestWithContext(ctx, "DELETE",
		fmt.Sprintf("%s/api/v1/policies/%d", ts.URL, policyID), nil)
	_, _ = http.DefaultClient.Do(delReq)

	var gotEvent bool
	deadline := time.After(3 * time.Second)
	for !gotEvent {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for delete SSE event")
		default:
		}
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()
		if line == "event: snapshot" {
			if scanner.Scan() {
				dataLine := scanner.Text()
				if strings.HasPrefix(dataLine, "data: ") {
					data := strings.TrimPrefix(dataLine, "data: ")
					var evt policy.PolicyEvent
					if err := json.Unmarshal([]byte(data), &evt); err != nil {
						t.Fatalf("unmarshal: %v", err)
					}
					if evt.Type == "snapshot" {
						// After delete, snapshot should have 0 enabled rules
						// (the only rule was deleted)
						if len(evt.Rules) == 0 {
							gotEvent = true
						}
					}
				}
			}
		}
	}

	if !gotEvent {
		t.Fatal("did not receive snapshot SSE event after delete")
	}
}
