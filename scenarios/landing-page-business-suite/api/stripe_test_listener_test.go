package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStripeCLIEnvironmentUsesOnlyNarrowChildInputs(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://must-not-inherit")
	t.Setenv("STRIPE_LIVE_SECRET_KEY", "sk_live_must-not-inherit")
	key := "rk_test_test-only-value"
	env := stripeCLIEnvironment(key, t.TempDir())
	joined := strings.Join(env, "\n")
	if !strings.Contains(joined, "STRIPE_API_KEY="+key) {
		t.Fatal("Stripe CLI did not receive the authority-scoped test key")
	}
	if strings.Contains(joined, "DATABASE_URL=") || strings.Contains(joined, "STRIPE_LIVE_SECRET_KEY=") || strings.Contains(joined, "sk_live_must-not-inherit") {
		t.Fatal("Stripe CLI inherited unrelated or live credentials")
	}
}

func TestStripeCLIRedactsAPIAndWebhookSecrets(t *testing.T) {
	key := "rk_test_authority-key-value"
	webhook := "whsec_listener-secret-value"
	got := redactStripeCLIOutput("key="+key+" webhook="+webhook+" event=evt_abc123", key)
	if strings.Contains(got, key) || strings.Contains(got, webhook) {
		t.Fatalf("CLI output leaked a secret: %q", got)
	}
	if !strings.Contains(got, "evt_abc123") {
		t.Fatalf("redaction removed nonsecret event evidence: %q", got)
	}
}

func TestStripeWebhookForwardProxyRetainsReceiptAndForwardsSignature(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"id":"evt_test123","type":"customer.updated"}` {
			t.Errorf("forwarded body = %s", body)
		}
		if got := r.Header.Get("Stripe-Signature"); got != "t=1,v1=test" {
			t.Errorf("Stripe-Signature = %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	deliveries := make(chan stripeDeliveryReceipt, 1)
	proxy := stripeWebhookForwardProxy(context.Background(), upstream.URL+"/api/v1/webhooks/stripe", deliveries)
	defer proxy.Close()

	req, err := http.NewRequest(http.MethodPost, proxy.URL+"/api/v1/webhooks/stripe", strings.NewReader(`{"id":"evt_test123","type":"customer.updated"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "t=1,v1=test")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("proxy status = %d", response.StatusCode)
	}
	select {
	case receipt := <-deliveries:
		if receipt.eventID != "evt_test123" || receipt.eventType != "customer.updated" || receipt.status != http.StatusOK {
			t.Fatalf("receipt = %+v", receipt)
		}
	default:
		t.Fatal("proxy did not retain the delivered event receipt")
	}
}
