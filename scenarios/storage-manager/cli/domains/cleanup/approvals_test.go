package cleanup

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

type flagContext struct {
	cliapp.OperationContext
	flags map[string]string
}

func (c flagContext) Flag(name string) string { return c.flags[name] }

func TestApprovalsGrantBindsApprovalToThisHost(t *testing.T) {
	host, err := os.Hostname()
	if err != nil {
		t.Fatalf("hostname: %v", err)
	}
	var got standingApproval
	var gotPath, gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(got)
	}))
	defer server.Close()

	h := &handlers{httpClient: server.Client(), baseURL: server.URL}
	before := time.Now().UTC()
	saved, err := h.approvalsGrantCall(flagContext{flags: map[string]string{"provider": "go-build-cache", "approved-by": "operator"}})
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/v1/cleanup/approvals/go-build-cache" {
		t.Fatalf("request = %s %s", gotMethod, gotPath)
	}
	if got.HostID != host || got.ApprovedBy != "operator" || got.ApprovedAt.Before(before) {
		t.Fatalf("approval body = %+v, want host %q, approved_by operator, approved_at >= %s", got, host, before)
	}
	if len(saved) != 1 || saved[0].Provider != "go-build-cache" {
		t.Fatalf("saved = %+v", saved)
	}
}

func TestApprovalsGrantRequiresProvider(t *testing.T) {
	h := &handlers{httpClient: http.DefaultClient, baseURL: "http://127.0.0.1:0"}
	if _, err := h.approvalsGrantCall(flagContext{flags: map[string]string{}}); err == nil {
		t.Fatal("grant without --provider succeeded")
	}
}

func TestApprovalsRevokeSurfacesAPIRefusal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "standing approval host_id does not match current host", http.StatusBadRequest)
	}))
	defer server.Close()
	h := &handlers{httpClient: server.Client(), baseURL: server.URL}
	if _, err := h.approvalsRevokeCall(flagContext{flags: map[string]string{"provider": "go-build-cache"}}); err == nil {
		t.Fatal("revoke hid the API refusal")
	}
}

func TestApprovalsListOrdersByProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]standingApproval{
			"journald":       {ApprovedBy: "a", HostID: "h"},
			"go-build-cache": {ApprovedBy: "b", HostID: "h"},
		})
	}))
	defer server.Close()
	h := &handlers{httpClient: server.Client(), baseURL: server.URL}
	list, err := h.approvalsListCall(nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 || list[0].Provider != "go-build-cache" || list[1].Provider != "journald" {
		t.Fatalf("list = %+v", list)
	}
}
