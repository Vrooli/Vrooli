package walletops

import "testing"

func TestNewLeasedTestNotificationHubRequiresLoopbackHTTP(t *testing.T) {
	for _, endpoint := range []string{
		"https://127.0.0.1:8443",
		"http://example.com:3000",
		"file:///tmp/notification-hub",
	} {
		if _, err := NewLeasedTestNotificationHub(endpoint, "cf5-local-agent"); err == nil {
			t.Fatalf("NewLeasedTestNotificationHub(%q) unexpectedly accepted a non-local endpoint", endpoint)
		}
	}
	for _, endpoint := range []string{"http://127.0.0.1:3000", "http://[::1]:3000", "http://localhost:3000"} {
		if _, err := NewLeasedTestNotificationHub(endpoint, "cf5-local-agent"); err != nil {
			t.Fatalf("NewLeasedTestNotificationHub(%q): %v", endpoint, err)
		}
	}
	if _, err := NewLeasedTestNotificationHub("http://127.0.0.1:3000", " "); err == nil {
		t.Fatal("leased test hub must require an isolated scoped producer identity")
	}
}
