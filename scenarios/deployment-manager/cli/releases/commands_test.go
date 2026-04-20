package releases

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
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read: %v", err)
	}
	_ = r.Close()
	return buf.String()
}

func TestRunRequiresSubcommand(t *testing.T) {
	if err := New(nil).Run(nil); err == nil {
		t.Fatalf("expected error when subcommand missing")
	}
}

func TestUnknownSubcommand(t *testing.T) {
	if err := New(nil).Run([]string{"frobnicate"}); err == nil ||
		!strings.Contains(err.Error(), "unknown releases subcommand") {
		t.Fatalf("expected unknown subcommand error, got %v", err)
	}
}

func TestListRequiresProfileID(t *testing.T) {
	if err := New(nil).Run([]string{"list"}); err == nil ||
		!strings.Contains(err.Error(), "profile ID is required") {
		t.Fatalf("expected profile ID required error, got %v", err)
	}
}

func TestListEmptyTable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/profiles/p1/releases") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"releases": []}`)
	}))
	defer srv.Close()

	cmdutil.SetGlobalFormat("text")
	defer cmdutil.SetGlobalFormat("json")

	out := captureOutput(t, func() {
		if err := New(testAPIClient(srv.URL)).Run([]string{"list", "p1"}); err != nil {
			t.Fatalf("list: %v", err)
		}
	})
	if !strings.Contains(out, "No releases found") {
		t.Errorf("expected 'No releases found', got: %s", out)
	}
}

func TestListPopulated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"releases":[{"id":"abc1234567890","channel":"stable","release_version":"1.2.3","status":"published","git_commit_hash":"deadbeefcafe1234","created_at":"2026-04-19"}]}`)
	}))
	defer srv.Close()

	cmdutil.SetGlobalFormat("text")
	defer cmdutil.SetGlobalFormat("json")

	out := captureOutput(t, func() {
		if err := New(testAPIClient(srv.URL)).Run([]string{"list", "p1", "--limit", "10"}); err != nil {
			t.Fatalf("list: %v", err)
		}
	})
	if !strings.Contains(out, "stable") || !strings.Contains(out, "1.2.3") {
		t.Errorf("expected channel+version in table, got: %s", out)
	}
}

func TestStartRequiresFlags(t *testing.T) {
	cmd := New(nil)
	if err := cmd.Run([]string{"start"}); err == nil ||
		!strings.Contains(err.Error(), "profile ID is required") {
		t.Fatalf("expected profile ID required error, got %v", err)
	}
	if err := cmd.Run([]string{"start", "p1"}); err == nil ||
		!strings.Contains(err.Error(), "--commit is required") {
		t.Fatalf("expected commit required error, got %v", err)
	}
	if err := cmd.Run([]string{"start", "p1", "--commit", "abc"}); err == nil ||
		!strings.Contains(err.Error(), "--version is required") {
		t.Fatalf("expected version required error, got %v", err)
	}
}

func TestStartSendsPayload(t *testing.T) {
	var captured map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_, _ = io.WriteString(w, `{"release":{"id":"r1","status":"pending","channel":"beta","release_version":"2.0.0","git_commit_hash":"deadbeefcafe"},"steps":[]}`)
	}))
	defer srv.Close()

	cmdutil.SetGlobalFormat("text")
	defer cmdutil.SetGlobalFormat("json")

	out := captureOutput(t, func() {
		err := New(testAPIClient(srv.URL)).Run([]string{
			"start", "p1",
			"--commit", "deadbeefcafe",
			"--version", "2.0.0",
			"--channel", "beta",
			"--platforms", "linux-x64,darwin-arm64",
			"--notes", "hotfix",
		})
		if err != nil {
			t.Fatalf("start: %v", err)
		}
	})

	if captured["channel"] != "beta" {
		t.Errorf("expected channel=beta, got %v", captured["channel"])
	}
	if captured["release_notes"] != "hotfix" {
		t.Errorf("expected release_notes=hotfix, got %v", captured["release_notes"])
	}
	if plats, ok := captured["platforms"].([]interface{}); !ok || len(plats) != 2 {
		t.Errorf("expected 2 platforms, got %v", captured["platforms"])
	}
	if !strings.Contains(out, "Release r1 status: pending") {
		t.Errorf("expected status line, got: %s", out)
	}
}

func TestVerifySendsDeepFlag(t *testing.T) {
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		_, _ = io.WriteString(w, `{"id":"r1","status":"published","channel":"stable","platforms":[{"platform":"linux-x64","status":"published"}]}`)
	}))
	defer srv.Close()

	cmdutil.SetGlobalFormat("text")
	defer cmdutil.SetGlobalFormat("json")

	out := captureOutput(t, func() {
		if err := New(testAPIClient(srv.URL)).Run([]string{"verify", "r1", "--deep"}); err != nil {
			t.Fatalf("verify: %v", err)
		}
	})
	if !strings.Contains(query, "deep=true") {
		t.Errorf("expected deep=true query param, got %s", query)
	}
	if !strings.Contains(out, "linux-x64: published") {
		t.Errorf("expected per-platform line, got: %s", out)
	}
}

func TestGetReturnsDetail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"id":"r1","status":"published","channel":"stable","release_version":"1.0.0","platforms":[{"platform":"linux-x64","status":"published"}]}`)
	}))
	defer srv.Close()

	cmdutil.SetGlobalFormat("text")
	defer cmdutil.SetGlobalFormat("json")

	out := captureOutput(t, func() {
		if err := New(testAPIClient(srv.URL)).Run([]string{"get", "r1"}); err != nil {
			t.Fatalf("get: %v", err)
		}
	})
	if !strings.Contains(out, "Release: r1") {
		t.Errorf("expected release id in output, got: %s", out)
	}
}
