package monetization

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBillingNilGateFailsClosedWithUpgradePath(t *testing.T) {
	decision := NewBilling(nil).Feature(context.Background(), "buyer@example.com", "voice_synthesis", 0)
	if decision.Allowed || decision.UpgradePath != "/settings/subscription" || decision.Reason != ReasonLeaseUnavailable {
		t.Fatalf("decision = %+v", decision)
	}
}

func TestJourneyProbeReturnsProviderNeutralObservation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("operation") != string(JourneyTamperedClassA) {
			t.Fatalf("operation = %q", r.URL.Query().Get("operation"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"observed":"class_a=refused","route":"lpbs-authority"}`))
	}))
	defer server.Close()
	got, err := (JourneyProbe{BaseURL: server.URL}).Run(context.Background(), JourneyTamperedClassA)
	if err != nil {
		t.Fatal(err)
	}
	if got.Observed != "class_a=refused" || got.Route != "lpbs-authority" {
		t.Fatalf("observation = %+v", got)
	}
}
