package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestHostedDoorDroppedStreamRecoversSameKeyWithoutSecondPostOrPrivateContent(t *testing.T) {
	const deviceKey = "test-hosted-door-device-key"
	const attemptKey = "durable-child-attempt-key"
	var mu sync.Mutex
	posts := 0
	receiptGets := 0
	var seenKeys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+deviceKey {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/ai/receipt":
			mu.Lock()
			defer mu.Unlock()
			receiptGets++
			seenKeys = append(seenKeys, r.Header.Get("Idempotency-Key"))
			if r.Header.Get("Idempotency-Key") != attemptKey {
				t.Errorf("receipt key = %q", r.Header.Get("Idempotency-Key"))
			}
			if posts == 0 {
				w.WriteHeader(http.StatusNotFound)
				_, _ = fmt.Fprint(w, `{"status":"unknown"}`)
				return
			}
			_, _ = fmt.Fprint(w, `{"status":"settled","charged_credits":17,"private_provider_receipt":{"output":"PRIVATE-OUTPUT","model":"PRIVATE-MODEL"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/ai/models":
			_, _ = fmt.Fprint(w, `{"data":[{"id":"private-model"}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/ai/chat/completions":
			mu.Lock()
			posts++
			seenKeys = append(seenKeys, r.Header.Get("Idempotency-Key"))
			mu.Unlock()
			if r.Header.Get("Idempotency-Key") != attemptKey {
				t.Errorf("POST idempotency key = %q", r.Header.Get("Idempotency-Key"))
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"PRIVATE-PARTIAL-OUTPUT\"}}]}\n\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &HostedDoorClient{BaseURL: server.URL, ResolveDeviceKey: func(context.Context) (string, error) { return deviceKey, nil }}
	execution, err := client.Run(context.Background(), ExecutionRequest{
		IdempotencyKey: attemptKey, ExpectedKeyGeneration: deviceKeyGeneration(deviceKey),
		InputText: "test prompt",
	})
	if err != nil {
		t.Fatal(err)
	}
	if execution.Recovery == nil || execution.Recovery.GetState() != inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_SETTLED {
		t.Fatalf("recovery = %+v", execution.Recovery)
	}
	if execution.OutputText != "" {
		t.Fatalf("uncertain execution leaked output %q", execution.OutputText)
	}
	if execution.Recovery.GetChargedCredits() != 17 {
		t.Fatalf("charged credits = %v", execution.Recovery.GetChargedCredits())
	}
	if execution.Recovery.GetAccountingStatus() != "settled" {
		t.Fatalf("accounting status = %q", execution.Recovery.GetAccountingStatus())
	}
	encoded, err := protojson.Marshal(execution.Recovery)
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{"PRIVATE-OUTPUT", "PRIVATE-MODEL", "private_provider_receipt", "private-model"} {
		if strings.Contains(string(encoded), private) {
			t.Fatalf("public recovery contains %q: %s", private, encoded)
		}
	}

	// A reconstructed call is receipt-only and cannot issue a second provider POST.
	recovered, err := client.Run(context.Background(), ExecutionRequest{
		IdempotencyKey: attemptKey, ExpectedKeyGeneration: deviceKeyGeneration(deviceKey), RecoveryOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Recovery == nil || recovered.Recovery.GetState() != inferencev1.RecoveryReceiptState_RECOVERY_RECEIPT_STATE_SETTLED {
		t.Fatalf("same-key recovery = %+v", recovered.Recovery)
	}
	mu.Lock()
	defer mu.Unlock()
	if posts != 1 {
		t.Fatalf("provider POST count = %d, want one", posts)
	}
	if receiptGets != 3 {
		t.Fatalf("receipt GET count = %d, want initial, post-stream and retry reads", receiptGets)
	}
	for _, key := range seenKeys {
		if key != attemptKey {
			t.Fatalf("observed a different receipt/dispatch key %q", key)
		}
	}
}

func TestHostedDoorRecoveryStatusIsAllowlisted(t *testing.T) {
	for _, status := range []string{"pending", "settled", "released", "unknown"} {
		if got := safeAccountingStatus(status); got != status {
			t.Fatalf("safeAccountingStatus(%q) = %q", status, got)
		}
	}
	if got := safeAccountingStatus("PRIVATE-RECEIPT-CONTENT"); got != "unknown" {
		t.Fatalf("unexpected status leaked through projection: %q", got)
	}
}
