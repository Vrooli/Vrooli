package config

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"tunnel-manager/internal/testutil/mocks"
)

func newTestDNSClient(doer *mocks.FakeDoer) *cfDNSClient {
	return &cfDNSClient{
		doer:      doer,
		apiToken:  "tok",
		tunnelID:  "tun123",
		baseURL:   "https://api.cloudflare.com/client/v4",
		zoneCache: map[string]string{},
	}
}

func TestApexOf(t *testing.T) {
	cases := map[string]string{
		"react-component-library.itsagitime.com": "itsagitime.com",
		"api.itsagitime.com":                     "itsagitime.com",
		"itsagitime.com":                         "itsagitime.com",
	}
	for in, want := range cases {
		if got := apexOf(in); got != want {
			t.Errorf("apexOf(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEnsureRecordCreatesWhenAbsent(t *testing.T) {
	doer := &mocks.FakeDoer{}
	doer.AddResponse(200, []byte(`{"success":true,"result":[{"id":"zone1"}]}`)) // zone lookup
	doer.AddResponse(200, []byte(`{"success":true,"result":[]}`))               // findRecord: none
	doer.AddResponse(200, []byte(`{"success":true,"result":{"id":"rec9"}}`))    // create

	c := newTestDNSClient(doer)
	res, err := c.EnsureRecord(context.Background(), "react-component-library.itsagitime.com")
	if err != nil {
		t.Fatalf("EnsureRecord: %v", err)
	}
	if !res.Created || res.RecordID != "rec9" {
		t.Fatalf("got %+v, want Created=true RecordID=rec9", res)
	}
	if doer.Calls.Load() != 3 {
		t.Fatalf("expected 3 calls, got %d", doer.Calls.Load())
	}
	// Assert the create request shape.
	create := doer.Requests[2]
	if create.Method != http.MethodPost {
		t.Errorf("create method = %s, want POST", create.Method)
	}
	if !strings.HasSuffix(create.URL.Path, "/zones/zone1/dns_records") {
		t.Errorf("create path = %s", create.URL.Path)
	}
	body, _ := io.ReadAll(create.Body)
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal create body: %v", err)
	}
	if payload["type"] != "CNAME" {
		t.Errorf("type = %v, want CNAME", payload["type"])
	}
	if payload["content"] != "tun123.cfargotunnel.com" {
		t.Errorf("content = %v, want tun123.cfargotunnel.com", payload["content"])
	}
	if payload["proxied"] != true {
		t.Errorf("proxied = %v, want true", payload["proxied"])
	}
	if payload["name"] != "react-component-library.itsagitime.com" {
		t.Errorf("name = %v", payload["name"])
	}
}

func TestEnsureRecordIdempotentWhenPresent(t *testing.T) {
	doer := &mocks.FakeDoer{}
	doer.AddResponse(200, []byte(`{"success":true,"result":[{"id":"zone1"}]}`))
	doer.AddResponse(200, []byte(`{"success":true,"result":[{"id":"recX","content":"tun123.cfargotunnel.com","type":"CNAME"}]}`))

	c := newTestDNSClient(doer)
	res, err := c.EnsureRecord(context.Background(), "api.itsagitime.com")
	if err != nil {
		t.Fatalf("EnsureRecord: %v", err)
	}
	if res.Created {
		t.Errorf("expected Created=false for pre-existing record")
	}
	if res.RecordID != "recX" {
		t.Errorf("RecordID = %q, want recX", res.RecordID)
	}
	if doer.Calls.Load() != 2 {
		t.Errorf("expected 2 calls (no POST), got %d", doer.Calls.Load())
	}
}

func TestRemoveRecordDeletesWhenFound(t *testing.T) {
	doer := &mocks.FakeDoer{}
	doer.AddResponse(200, []byte(`{"success":true,"result":[{"id":"zone1"}]}`))
	doer.AddResponse(200, []byte(`{"success":true,"result":[{"id":"recDel","content":"tun123.cfargotunnel.com"}]}`))
	doer.AddResponse(200, []byte(`{"success":true}`)) // delete

	c := newTestDNSClient(doer)
	removed, err := c.RemoveRecord(context.Background(), "api.itsagitime.com")
	if err != nil {
		t.Fatalf("RemoveRecord: %v", err)
	}
	if !removed {
		t.Errorf("expected removed=true")
	}
	del := doer.Requests[2]
	if del.Method != http.MethodDelete {
		t.Errorf("delete method = %s", del.Method)
	}
	if !strings.HasSuffix(del.URL.Path, "/zones/zone1/dns_records/recDel") {
		t.Errorf("delete path = %s", del.URL.Path)
	}
}

func TestRemoveRecordIdempotentWhenAbsent(t *testing.T) {
	doer := &mocks.FakeDoer{}
	doer.AddResponse(200, []byte(`{"success":true,"result":[{"id":"zone1"}]}`))
	doer.AddResponse(200, []byte(`{"success":true,"result":[]}`)) // no record

	c := newTestDNSClient(doer)
	removed, err := c.RemoveRecord(context.Background(), "gone.itsagitime.com")
	if err != nil {
		t.Fatalf("RemoveRecord: %v", err)
	}
	if removed {
		t.Errorf("expected removed=false")
	}
	if doer.Calls.Load() != 2 {
		t.Errorf("expected 2 calls (no DELETE), got %d", doer.Calls.Load())
	}
}

func TestNewCFDNSClientNilWithoutCreds(t *testing.T) {
	if NewCFDNSClient(&mocks.FakeDoer{}, CFConfig{TunnelID: "t"}) == nil {
		// token missing -> nil expected
	} else {
		t.Error("expected nil DNS client when token absent")
	}
	if NewCFDNSClient(&mocks.FakeDoer{}, CFConfig{APIToken: "x", TunnelID: "t"}) == nil {
		t.Error("expected a client when token+tunnel present")
	}
}
