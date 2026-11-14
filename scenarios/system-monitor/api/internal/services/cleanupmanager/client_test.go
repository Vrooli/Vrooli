package cleanupmanager

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWaitRecoveryUsesTerminalReceiptAndChecksRunIdentity(t *testing.T) {
	for _, wrongRun := range []bool{false, true} {
		t.Run(map[bool]string{false: "completed", true: "wrong run"}[wrongRun], func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Path != waitRecoveryProcedure {
					t.Errorf("path = %s", r.URL.Path)
				}
				var body map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["runId"] != "run-1" {
					t.Errorf("request = %#v, %v", body, err)
				}
				id := "run-1"
				if wrongRun {
					id = "other"
				}
				_ = json.NewEncoder(w).Encode(map[string]string{"runId": id, "status": "complete", "action": "applied", "reclaimedBytes": "4294967296", "reason": "target_met"})
			}))
			defer server.Close()
			receipt, err := NewClient(Config{BaseURL: server.URL}).WaitRecovery(context.Background(), "run-1")
			if wrongRun {
				if err == nil {
					t.Fatal("accepted another run's receipt")
				}
			} else if err != nil || receipt.ReclaimedBytes != 4294967296 || receipt.Status != "complete" {
				t.Fatalf("receipt=%#v err=%v", receipt, err)
			}
			if calls != 1 {
				t.Fatalf("wait called %d times", calls)
			}
		})
	}
}

func TestWaitRecoveryHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewClient(Config{BaseURL: "http://127.0.0.1:1"}).WaitRecovery(ctx, "run-1")
	if err == nil {
		t.Fatal("canceled wait succeeded")
	}
}
