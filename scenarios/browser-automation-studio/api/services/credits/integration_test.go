package credits

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
)

func TestUsageOutboxSurvivesServiceRestart(t *testing.T) {
	received := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/usage/report" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer restart-token" {
			t.Errorf("authorization = %q", got)
		}
		var report map[string]any
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			t.Errorf("decode usage event: %v", err)
		}
		received <- report
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	db := createTestDB(t)
	defer db.Close()
	access := func(context.Context) (string, error) { return "restart-token", nil }
	first := NewService(ServiceOptions{
		DB: db, Logger: logrus.New(), LPBSURL: server.URL,
		LPBSAccessToken: access, AppBundleKey: "browser-automation-studio",
	})
	usage := monetization.Usage{
		OperationID: "restart-operation", UserIdentity: "restart@example.com",
		BundleKey: "business_suite", AppKey: "browser-automation-studio",
		MeterKey: "workflow_executions", Units: 1, OccurredAt: time.Now().UTC(),
	}
	if err := first.monetizationOutbox.Enqueue(context.Background(), usage); err != nil {
		t.Fatalf("enqueue before restart: %v", err)
	}

	// A fresh service instance represents the process after a crash/restart.
	second := NewService(ServiceOptions{
		DB: db, Logger: logrus.New(), LPBSURL: server.URL,
		LPBSAccessToken: access, AppBundleKey: "browser-automation-studio",
	})
	delivered, err := second.DrainOutbox(context.Background(), 10)
	if err != nil || delivered != 1 {
		t.Fatalf("restart drain delivered=%d err=%v", delivered, err)
	}
	var status string
	if err := db.QueryRow("SELECT status FROM monetization_usage_outbox WHERE operation_id = ?", usage.OperationID).Scan(&status); err != nil {
		t.Fatalf("read durable outbox status: %v", err)
	}
	if status != "delivered" {
		t.Fatalf("outbox status = %q, want delivered", status)
	}
	if got := (<-received)["operation_id"]; got != usage.OperationID {
		t.Fatalf("reported operation_id = %v", got)
	}
}
