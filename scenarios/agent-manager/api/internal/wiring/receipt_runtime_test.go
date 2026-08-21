package wiring

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/runreport"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/eventbus"
)

func TestDeclaredReceiptTargetsUsesAgentManagerDeclaration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipt-capture.json")
	if err := os.WriteFile(path, []byte(`{"policies":[{"targetScenario":"test-genie"},{"targetScenario":"test-genie"},{"targetScenario":"workspace-sandbox"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	targets, err := declaredReceiptTargets(path)
	if err != nil || len(targets) != 2 || targets[0] != "test-genie" || targets[1] != "workspace-sandbox" {
		t.Fatalf("targets=%v err=%v", targets, err)
	}
}

func TestReceiptSummaryReaderDistinguishesRuntimeEmptyStates(t *testing.T) {
	events := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(events.Close)
	for _, tc := range []struct {
		name       string
		runtime    eventbus.RuntimeState
		wantState  runreport.AvailabilityState
		wantDetail string
	}{
		{name: "never connected", runtime: eventbus.RuntimeState{State: "never_connected"}, wantState: runreport.AvailabilityUnavailable, wantDetail: "never connected"},
		{name: "connected empty", runtime: eventbus.RuntimeState{State: "connected_empty", Armed: true}, wantState: runreport.AvailabilityPolicyAbsent, wantDetail: "connected but has no capture policies"},
		{name: "armed", runtime: eventbus.RuntimeState{State: "armed", Armed: true, PolicyCount: 1}, wantState: runreport.AvailabilityUnobserved, wantDetail: "no verified receipts"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := newReceiptSummaryReader(eventbus.Client{BaseURL: events.URL, HTTPClient: events.Client()}, []string{"test-genie"}, func(context.Context, string) (eventbus.RuntimeState, error) {
				return tc.runtime, nil
			})
			summary, err := reader.ReadReceiptSummary(context.Background(), uuid.New())
			if err != nil || summary.State != tc.wantState || !strings.Contains(summary.Reason, tc.wantDetail) {
				t.Fatalf("summary=%+v err=%v", summary, err)
			}
		})
	}
}
