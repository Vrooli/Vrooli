package digitalocean

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"compute-manager/internal/provider"
)

func TestTagListUsesDigitalOceanTagNames(t *testing.T) {
	tags := tagList(map[string]string{"vrooli-tenant": "owner", "vrooli-managed": "compute-manager"})
	seen := map[string]bool{}
	for _, tag := range tags {
		seen[tag] = true
	}
	if !seen["vrooli-tenant:owner"] || !seen["vrooli-managed:compute-manager"] {
		t.Fatalf("tags = %v", tags)
	}
}

func TestProviderMapsDropletLifecycleAndResolvesTokenAtCallTime(t *testing.T) { // [REQ:COMPUTEM-P1-004]
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method+" "+r.URL.Path)
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost || r.URL.Path != "/droplets":
			_, _ = w.Write([]byte(`{"droplet":{"id":12,"created_at":"2026-09-04T00:00:00Z","region":{"slug":"nyc3"},"size":{"slug":"s-1vcpu-1gb"},"image":{"slug":"ubuntu-24-04"},"networks":{"v4":[{"type":"public","ip_address":"203.0.113.12"}]}}}`))
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"droplets":[{"id":12,"created_at":"2026-09-04T00:00:00Z","region":{"slug":"nyc3"},"size":{"slug":"s-1vcpu-1gb"},"image":{"slug":"ubuntu-24-04"},"networks":{"v4":[{"type":"public","ip_address":"203.0.113.12"}]}}]}`))
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()
	p := &Provider{BaseURL: server.URL, Token: func(context.Context) (string, error) { return "secret", nil }}
	created, err := p.Create(context.Background(), provider.Spec{Region: "nyc3", Size: "s-1vcpu-1gb", Image: "ubuntu-24-04"})
	if err != nil || created.ID != "12" || created.Address != "203.0.113.12" || created.Region != "nyc3" {
		t.Fatalf("created = %+v, err = %v", created, err)
	}
	described, err := p.Describe(context.Background(), "12")
	if err != nil || described.ID != "12" {
		t.Fatalf("described = %+v, err = %v", described, err)
	}
	listed, err := p.List(context.Background())
	if err != nil || len(listed) != 1 {
		t.Fatalf("listed = %+v, err = %v", listed, err)
	}
	if err := p.Destroy(context.Background(), "12"); err != nil {
		t.Fatal(err)
	}
	if len(methods) != 4 || methods[0] != "POST /droplets" || methods[1] != "GET /droplets/12" || methods[2] != "GET /droplets" || methods[3] != "DELETE /droplets/12" {
		t.Fatalf("methods = %v", methods)
	}
}

func TestUnavailableCredentialIsTyped(t *testing.T) { // [REQ:COMPUTEM-P1-004]
	p := &Provider{Token: func(context.Context) (string, error) { return "", errMissingCredential{} }}
	_, err := p.Describe(context.Background(), "12")
	if !errors.Is(err, provider.ErrProviderUnavailable) {
		t.Fatalf("err = %v", err)
	}
}

func TestBillingStatementsMapsDropletInvoiceItemsForRequestedRange(t *testing.T) { // [REQ:COMPUTEM-P1-003]
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/customers/my/invoices":
			_, _ = w.Write([]byte(`{"invoices":[{"invoice_uuid":"invoice-1"}],"meta":{"total":1}}`))
		case "/customers/my/invoices/invoice-1":
			_, _ = w.Write([]byte(`{"invoice_items":[
{"product":"Droplets","resource_id":"12","duration":"1.5","duration_unit":"Hours","start_time":"2026-09-03T00:00:00Z","end_time":"2026-09-03T02:00:00Z"},
{"product":"Spaces","resource_id":"space-1","duration":"10","duration_unit":"Hours","start_time":"2026-09-03T00:00:00Z","end_time":"2026-09-03T10:00:00Z"},
{"product":"Droplets","resource_id":"13","duration":"1","duration_unit":"Hours","start_time":"2026-09-02T00:00:00Z","end_time":"2026-09-02T01:00:00Z"}
],"meta":{"total":3}}`))
		default:
			t.Fatalf("unexpected billing path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	p := &Provider{BaseURL: server.URL, Token: func(context.Context) (string, error) { return "secret", nil }}
	from := time.Date(2026, 9, 3, 0, 30, 0, 0, time.UTC)
	to := time.Date(2026, 9, 3, 1, 0, 0, 0, time.UTC)
	statements, err := p.BillingStatements(context.Background(), from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(statements) != 1 {
		t.Fatalf("statements = %#v, want one droplet statement", statements)
	}
	if statements[0].ProviderInstanceID != "12" || statements[0].Minutes != 90 || statements[0].Provider != "digitalocean" {
		t.Fatalf("statement = %#v", statements[0])
	}
}

type errMissingCredential struct{}

func (errMissingCredential) Error() string { return "missing" }
