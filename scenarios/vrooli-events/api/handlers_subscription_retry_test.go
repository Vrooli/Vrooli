package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/subscription"
)

func TestSubscriptionDeliveryRetryUsesPersistedPayloadAndPreservesHealthHistory(t *testing.T) {
	var fail atomic.Bool
	fail.Store(true)
	observed := make(chan []byte, 1)
	webhook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		body, _ := io.ReadAll(r.Body)
		observed <- body
		w.WriteHeader(http.StatusAccepted)
	}))
	defer webhook.Close()

	srv, ts := newTestServer(t)
	subscriptionID, err := srv.subStore.Create(context.Background(), subscription.Subscription{
		Name: "retry-webhook", OwnerScenario: "test", EventPattern: "example.**",
		DeliveryType: subscription.DeliveryWebhook, DeliveryTarget: webhook.URL, Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	queue := srv.subStore.(subscription.QueueStore)
	const eventID = "evt-exact-retry-001"
	const storedPayload = `{"event_id":"evt-exact-retry-001","event_type":"example.review.v1","payload":{"marker":"stored-facts"}}`
	if err := queue.Enqueue(context.Background(), subscriptionID, eventID, storedPayload); err != nil {
		t.Fatal(err)
	}
	markQueueDeadLetter(t, queue, subscriptionID, 5)

	readURL := ts.URL + "/api/v1/subscriptions/" + itoa(subscriptionID) + "/deliveries/" + eventID
	readResponse, err := http.Get(readURL)
	if err != nil {
		t.Fatal(err)
	}
	var metadata map[string]any
	if err := json.NewDecoder(readResponse.Body).Decode(&metadata); err != nil {
		readResponse.Body.Close()
		t.Fatal(err)
	}
	readResponse.Body.Close()
	if readResponse.StatusCode != http.StatusOK || metadata["status"] != "dead_letter" || metadata["attempts"] != float64(5) {
		t.Fatalf("metadata status=%d body=%#v", readResponse.StatusCode, metadata)
	}
	if _, ok := metadata["payload_json"]; ok {
		t.Fatal("delivery metadata exposed persisted payload")
	}
	if _, ok := metadata["delivery_target"]; ok {
		t.Fatal("delivery metadata exposed target")
	}

	retryURL := ts.URL + "/api/v1/subscriptions/" + itoa(subscriptionID) + "/deliveries/retry"
	retryBody := `{"event_id":"evt-exact-retry-001","expected_attempts":5,"payload":{"marker":"caller-replacement"}}`
	request, err := http.NewRequest(http.MethodPost, retryURL, strings.NewReader(retryBody))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	retryResponse, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	var retryResult map[string]any
	_ = json.NewDecoder(retryResponse.Body).Decode(&retryResult)
	retryResponse.Body.Close()
	if retryResponse.StatusCode != http.StatusAccepted || retryResult["status"] != "queued" || retryResult["attempts"] != float64(5) {
		t.Fatalf("retry status=%d body=%#v", retryResponse.StatusCode, retryResult)
	}

	duplicateRequest, err := http.NewRequest(http.MethodPost, retryURL, strings.NewReader(`{"event_id":"evt-exact-retry-001","expected_attempts":5}`))
	if err != nil {
		t.Fatal(err)
	}
	duplicateRequest.Header.Set("Content-Type", "application/json")
	duplicateResponse, err := http.DefaultClient.Do(duplicateRequest)
	if err != nil {
		t.Fatal(err)
	}
	duplicateResponse.Body.Close()
	if duplicateResponse.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate retry status=%d, want 409", duplicateResponse.StatusCode)
	}

	due, err := queue.Due(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 1 || due[0].EventID != eventID || due[0].PayloadJSON != storedPayload || due[0].Attempts != 5 {
		t.Fatalf("queued retry changed identity/payload/budget: %#v", due)
	}
	fail.Store(false)
	srv.drainSubscriptionQueue(context.Background(), queue)
	select {
	case delivered := <-observed:
		var webhookPayload map[string]any
		if err := json.Unmarshal(delivered, &webhookPayload); err != nil {
			t.Fatal(err)
		}
		inner, _ := webhookPayload["payload"].(map[string]any)
		if webhookPayload["event_id"] != eventID || inner["marker"] != "stored-facts" {
			t.Fatalf("dispatcher sent replaced or wrong event facts: %s", delivered)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("existing dispatcher did not deliver requeued event")
	}

	final, err := queue.GetDelivery(context.Background(), subscriptionID, eventID)
	if err != nil {
		t.Fatal(err)
	}
	if final.Status != "delivered" || final.Attempts != 5 {
		t.Fatalf("final delivery metadata = %#v, want delivered with five cumulative failed attempts", final)
	}
	health, err := srv.subStore.GetHealth(context.Background(), subscriptionID)
	if err != nil {
		t.Fatal(err)
	}
	if health.TotalFailed != 5 || health.TotalDelivered != 1 || health.ConsecutiveFailures != 0 || health.Status != "active" {
		t.Fatalf("health after recovery = %#v, want retained failures and one success", health)
	}
}

func markQueueDeadLetter(t *testing.T, queue subscription.QueueStore, subscriptionID int64, attempts int) {
	t.Helper()
	for i := 0; i < attempts; i++ {
		due, err := queue.Due(context.Background(), 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(due) != 1 {
			t.Fatalf("failure %d: due rows=%d, want 1", i+1, len(due))
		}
		if err := queue.MarkFailed(context.Background(), due[0].ID, subscriptionID, "webhook returned status 500", time.Now().Add(-time.Second), i == attempts-1); err != nil {
			t.Fatal(err)
		}
	}
}
