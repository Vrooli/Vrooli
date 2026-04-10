package approvals

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"deployment-manager/cli/cmdutil"

	"github.com/vrooli/cli-core/cliutil"
)

func testAPIClient(base string) *cliutil.APIClient {
	return cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{BaseOptions: cliutil.APIBaseOptions{DefaultBase: base}}),
		func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: base} },
		func() string { return "" },
	)
}

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	_ = r.Close()
	return buf.String()
}

// --- list ---

func TestListRequiresProfileID(t *testing.T) {
	cmd := New(nil)
	if err := cmd.Run([]string{"list"}); err == nil || !strings.Contains(err.Error(), "profile ID is required") {
		t.Fatalf("expected profile ID required error, got %v", err)
	}
}

func TestListSendsCommitQueryParam(t *testing.T) {
	var queryParams string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queryParams = r.URL.RawQuery
		_, _ = io.WriteString(w, `[]`)
	}))
	defer server.Close()

	cmdutil.SetGlobalFormat("text")
	defer cmdutil.SetGlobalFormat("json")

	cmd := New(testAPIClient(server.URL))
	output := captureOutput(t, func() {
		if err := cmd.Run([]string{"list", "prof-1", "--commit", "abc123"}); err != nil {
			t.Fatalf("list failed: %v", err)
		}
	})

	if !strings.Contains(queryParams, "commit=abc123") {
		t.Fatalf("expected commit query param, got %s", queryParams)
	}
	if !strings.Contains(output, "No approvals found") {
		t.Fatalf("expected empty message, got %s", output)
	}
}

func TestListTableOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": "approval-1", "platform": "linux", "status": "pending", "git_commit_hash": "abc123def456", "approved_by": "", "updated_at": "2026-01-01"},
		})
	}))
	defer server.Close()

	cmdutil.SetGlobalFormat("text")
	defer cmdutil.SetGlobalFormat("json")

	cmd := New(testAPIClient(server.URL))
	output := captureOutput(t, func() {
		if err := cmd.Run([]string{"list", "prof-1"}); err != nil {
			t.Fatalf("list failed: %v", err)
		}
	})

	if !strings.Contains(output, "approval-1") || !strings.Contains(output, "linux") || !strings.Contains(output, "pending") {
		t.Fatalf("expected table output with approval data, got %s", output)
	}
}

func TestListJSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `[{"id":"approval-1"}]`)
	}))
	defer server.Close()

	cmd := New(testAPIClient(server.URL))
	output := captureOutput(t, func() {
		if err := cmd.Run([]string{"list", "prof-1", "--format", "json"}); err != nil {
			t.Fatalf("list failed: %v", err)
		}
	})

	if !strings.Contains(output, "approval-1") {
		t.Fatalf("expected JSON output, got %s", output)
	}
}

// --- get ---

func TestGetRequiresApprovalID(t *testing.T) {
	cmd := New(nil)
	if err := cmd.Run([]string{"get"}); err == nil || !strings.Contains(err.Error(), "approval ID is required") {
		t.Fatalf("expected approval ID required error, got %v", err)
	}
}

func TestGetHumanOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/api/v1/approvals/approval-1") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "approval-1", "profile_id": "prof-1", "platform": "linux",
			"status": "approved", "git_commit_hash": "abc123", "approved_by": "alice",
			"notes": "looks good", "created_at": "2026-01-01", "updated_at": "2026-01-02",
		})
	}))
	defer server.Close()

	cmdutil.SetGlobalFormat("text")
	defer cmdutil.SetGlobalFormat("json")

	cmd := New(testAPIClient(server.URL))
	output := captureOutput(t, func() {
		if err := cmd.Run([]string{"get", "approval-1"}); err != nil {
			t.Fatalf("get failed: %v", err)
		}
	})

	for _, want := range []string{"approval-1", "prof-1", "linux", "approved", "alice", "looks good"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected %q in output, got %s", want, output)
		}
	}
}

// --- create ---

func TestCreateRequiresFlags(t *testing.T) {
	cmd := New(nil)
	tests := []struct {
		args   []string
		errMsg string
	}{
		{[]string{"create"}, "profile ID is required"},
		{[]string{"create", "prof-1"}, "--commit is required"},
		{[]string{"create", "prof-1", "--commit", "abc"}, "--platform is required"},
	}
	for _, tt := range tests {
		if err := cmd.Run(tt.args); err == nil || !strings.Contains(err.Error(), tt.errMsg) {
			t.Fatalf("args %v: expected %q, got %v", tt.args, tt.errMsg, err)
		}
	}
}

func TestCreateSendsCorrectPayload(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "approval-new", "status": "pending"})
	}))
	defer server.Close()

	cmdutil.SetGlobalFormat("text")
	defer cmdutil.SetGlobalFormat("json")

	cmd := New(testAPIClient(server.URL))
	output := captureOutput(t, func() {
		if err := cmd.Run([]string{"create", "prof-1", "--commit", "abc123", "--platform", "linux"}); err != nil {
			t.Fatalf("create failed: %v", err)
		}
	})

	if receivedBody["git_commit_hash"] != "abc123" || receivedBody["platform"] != "linux" {
		t.Fatalf("unexpected payload: %v", receivedBody)
	}
	if !strings.Contains(output, "approval-new") {
		t.Fatalf("expected confirmation, got %s", output)
	}
}

// --- decide ---

func TestDecideRequiresFlags(t *testing.T) {
	cmd := New(nil)
	tests := []struct {
		args   []string
		errMsg string
	}{
		{[]string{"decide"}, "approval ID is required"},
		{[]string{"decide", "a-1"}, "--decision must be"},
		{[]string{"decide", "a-1", "--decision", "approved"}, "--reviewer is required"},
	}
	for _, tt := range tests {
		if err := cmd.Run(tt.args); err == nil || !strings.Contains(err.Error(), tt.errMsg) {
			t.Fatalf("args %v: expected %q, got %v", tt.args, tt.errMsg, err)
		}
	}
}

func TestDecideSendsCorrectPayload(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "a-1", "status": "approved", "approved_by": "alice"})
	}))
	defer server.Close()

	cmdutil.SetGlobalFormat("text")
	defer cmdutil.SetGlobalFormat("json")

	cmd := New(testAPIClient(server.URL))
	output := captureOutput(t, func() {
		if err := cmd.Run([]string{"decide", "a-1", "--decision", "approved", "--reviewer", "alice", "--notes", "lgtm"}); err != nil {
			t.Fatalf("decide failed: %v", err)
		}
	})

	if receivedBody["decision"] != "approved" || receivedBody["reviewer"] != "alice" || receivedBody["notes"] != "lgtm" {
		t.Fatalf("unexpected payload: %v", receivedBody)
	}
	if !strings.Contains(output, "approved") && !strings.Contains(output, "alice") {
		t.Fatalf("expected confirmation, got %s", output)
	}
}

// --- gate ---

func TestGateRequiresFlags(t *testing.T) {
	cmd := New(nil)
	tests := []struct {
		args   []string
		errMsg string
	}{
		{[]string{"gate"}, "profile ID is required"},
		{[]string{"gate", "prof-1"}, "--commit is required"},
	}
	for _, tt := range tests {
		if err := cmd.Run(tt.args); err == nil || !strings.Contains(err.Error(), tt.errMsg) {
			t.Fatalf("args %v: expected %q, got %v", tt.args, tt.errMsg, err)
		}
	}
}

func TestGateReadyOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "commit=abc123") {
			t.Fatalf("expected commit query param, got %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"profile_id": "prof-1", "git_commit_hash": "abc123", "ready": true,
			"platforms": []map[string]interface{}{
				{"platform": "linux", "required": true, "status": "approved"},
				{"platform": "windows", "required": true, "status": "approved"},
			},
		})
	}))
	defer server.Close()

	cmdutil.SetGlobalFormat("text")
	defer cmdutil.SetGlobalFormat("json")

	cmd := New(testAPIClient(server.URL))
	output := captureOutput(t, func() {
		if err := cmd.Run([]string{"gate", "prof-1", "--commit", "abc123"}); err != nil {
			t.Fatalf("gate failed: %v", err)
		}
	})

	if !strings.Contains(output, "READY") {
		t.Fatalf("expected READY status, got %s", output)
	}
	if strings.Contains(output, "Next Steps") {
		t.Fatalf("unexpected next steps for ready gate, got %s", output)
	}
}

func TestGateBlockedOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"profile_id": "prof-1", "git_commit_hash": "abc123", "ready": false,
			"platforms": []map[string]interface{}{
				{"platform": "linux", "required": true, "status": "pending"},
			},
		})
	}))
	defer server.Close()

	cmdutil.SetGlobalFormat("text")
	defer cmdutil.SetGlobalFormat("json")

	cmd := New(testAPIClient(server.URL))
	output := captureOutput(t, func() {
		if err := cmd.Run([]string{"gate", "prof-1", "--commit", "abc123"}); err != nil {
			t.Fatalf("gate failed: %v", err)
		}
	})

	if !strings.Contains(output, "BLOCKED") {
		t.Fatalf("expected BLOCKED status, got %s", output)
	}
	if !strings.Contains(output, "Next Steps") {
		t.Fatalf("expected next steps for blocked gate, got %s", output)
	}
}

// --- platforms set ---

func TestPlatformsSetRequiresFlags(t *testing.T) {
	cmd := New(nil)
	tests := []struct {
		args   []string
		errMsg string
	}{
		{[]string{"platforms", "set"}, "profile ID is required"},
		{[]string{"platforms", "set", "prof-1"}, "--platforms is required"},
	}
	for _, tt := range tests {
		if err := cmd.Run(tt.args); err == nil || !strings.Contains(err.Error(), tt.errMsg) {
			t.Fatalf("args %v: expected %q, got %v", tt.args, tt.errMsg, err)
		}
	}
}

func TestPlatformsSetParsesCSV(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"profile_id": "prof-1", "platforms": []string{"win", "mac", "linux"}})
	}))
	defer server.Close()

	cmdutil.SetGlobalFormat("text")
	defer cmdutil.SetGlobalFormat("json")

	cmd := New(testAPIClient(server.URL))
	output := captureOutput(t, func() {
		if err := cmd.Run([]string{"platforms", "set", "prof-1", "--platforms", "win,mac,linux"}); err != nil {
			t.Fatalf("platforms set failed: %v", err)
		}
	})

	platforms, ok := receivedBody["platforms"].([]interface{})
	if !ok || len(platforms) != 3 {
		t.Fatalf("expected 3 platforms, got %v", receivedBody)
	}
	if !strings.Contains(output, "win") || !strings.Contains(output, "mac") || !strings.Contains(output, "linux") {
		t.Fatalf("expected platform names in output, got %s", output)
	}
}

// --- platforms get ---

func TestPlatformsGetRequiresProfileID(t *testing.T) {
	cmd := New(nil)
	if err := cmd.Run([]string{"platforms", "get"}); err == nil || !strings.Contains(err.Error(), "profile ID is required") {
		t.Fatalf("expected profile ID required error, got %v", err)
	}
}

func TestPlatformsGetOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"profile_id": "prof-1", "platforms": []string{"linux", "windows"}})
	}))
	defer server.Close()

	cmdutil.SetGlobalFormat("text")
	defer cmdutil.SetGlobalFormat("json")

	cmd := New(testAPIClient(server.URL))
	output := captureOutput(t, func() {
		if err := cmd.Run([]string{"platforms", "get", "prof-1"}); err != nil {
			t.Fatalf("platforms get failed: %v", err)
		}
	})

	if !strings.Contains(output, "linux") || !strings.Contains(output, "windows") {
		t.Fatalf("expected platform names, got %s", output)
	}
}

// --- dispatch errors ---

func TestRunRequiresSubcommand(t *testing.T) {
	cmd := New(nil)
	if err := cmd.Run(nil); err == nil || !strings.Contains(err.Error(), "subcommand required") {
		t.Fatalf("expected subcommand required error, got %v", err)
	}
}

func TestRunUnknownSubcommand(t *testing.T) {
	cmd := New(nil)
	if err := cmd.Run([]string{"bogus"}); err == nil || !strings.Contains(err.Error(), "unknown approvals subcommand") {
		t.Fatalf("expected unknown subcommand error, got %v", err)
	}
}

func TestPlatformsUnknownSubcommand(t *testing.T) {
	cmd := New(nil)
	if err := cmd.Run([]string{"platforms", "bogus"}); err == nil || !strings.Contains(err.Error(), "unknown platforms subcommand") {
		t.Fatalf("expected unknown platforms subcommand error, got %v", err)
	}
}
