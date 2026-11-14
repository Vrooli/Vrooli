package signing

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCommandsCoverConfiguredSigningLifecycle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/signing"):
			_, _ = io.WriteString(w, `{"enabled":true,"windows":{"certificate_source":"file","certificate_file":"cert.pfx","certificate_password_env":"CERT_PASS","timestamp_server":"timestamp","sign_algorithm":"sha256","dual_sign":true},"macos":{"identity":"Developer ID","team_id":"TEAM","hardened_runtime":true,"notarize":true},"linux":{"gpg_key_id":"ABC","gpg_passphrase_env":"GPG_PASS"}}`)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/signing/prerequisites"):
			_, _ = io.WriteString(w, `{"tools":[{"platform":"windows","tool":"signtool","installed":true,"path":"signtool","version":"1"},{"platform":"linux","tool":"gpg","installed":false,"error":"not installed"}]}`)
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/signing/discover/"):
			_, _ = io.WriteString(w, `{"platform":"windows","certificates":[{"id":"c1","name":"Expired","days_to_expiry":-1,"is_expired":true,"is_code_sign":true,"type":"pfx","expires_at":"2020"},{"id":"c2","name":"Soon","days_to_expiry":10,"is_code_sign":true,"expires_at":"soon"},{"id":"c3","name":"Never","expires_at":"never"}],"errors":["one warning"]}`)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/signing/validate"):
			_, _ = io.WriteString(w, `{"valid":false,"message":"missing key","errors":[{"platform":"linux","code":"missing","message":"gpg","remediation":"install gpg"}],"warnings":[{"platform":"windows","code":"soon","message":"expires"}]}`)
		default:
			_, _ = io.WriteString(w, `{"ok":true}`)
		}
	}))
	defer server.Close()
	cmd := New(testAPIClient(server.URL))

	if err := cmd.Show([]string{"demo", "--format", "json"}); err != nil {
		t.Fatalf("show json: %v", err)
	}
	for _, platform := range []string{"windows", "macos", "linux"} {
		if err := cmd.Show([]string{"demo", "--platform", platform, "--format", "table"}); err != nil {
			t.Fatalf("show %s: %v", platform, err)
		}
	}
	if err := cmd.Set([]string{"demo", "--platform", "windows", "--cert", "cert.pfx", "--password-env", "PASS", "--dual-sign"}); err != nil {
		t.Fatalf("set windows: %v", err)
	}
	if err := cmd.Set([]string{"demo", "--platform", "windows", "--thumbprint", "ABC"}); err != nil {
		t.Fatalf("set windows store: %v", err)
	}
	if err := cmd.Set([]string{"demo", "--platform", "macos", "--identity", "Developer ID", "--team-id", "TEAM", "--api-key-id", "KEY", "--api-key-file", "key.p8", "--api-issuer", "ISSUER"}); err != nil {
		t.Fatalf("set macos: %v", err)
	}
	if err := cmd.Set([]string{"demo", "--platform", "linux", "--gpg-key", "ABC", "--gpg-passphrase-env", "PASS"}); err != nil {
		t.Fatalf("set linux: %v", err)
	}
	if err := cmd.Remove([]string{"demo"}); err != nil {
		t.Fatalf("remove all: %v", err)
	}
	if err := cmd.Remove([]string{"demo", "--platform", "linux"}); err != nil {
		t.Fatalf("remove linux: %v", err)
	}
	if err := cmd.Validate([]string{"demo", "--format", "table"}); err != nil {
		t.Fatalf("validate table: %v", err)
	}
	if err := cmd.Prerequisites([]string{"--format", "table"}); err != nil {
		t.Fatalf("prerequisites table: %v", err)
	}
	if err := cmd.Discover([]string{"--platform", "windows", "--format", "table"}); err != nil {
		t.Fatalf("discover table: %v", err)
	}
	if err := cmd.Discover([]string{"--platform", "linux", "--format", "json"}); err != nil {
		t.Fatalf("discover json: %v", err)
	}
}

func TestCommandsRejectSigningInput(t *testing.T) {
	cmd := New(testAPIClient("http://127.0.0.1:1"))
	for _, args := range [][]string{
		{"show"}, {"show", "demo", "--platform", "freebsd"},
		{"set"}, {"set", "demo"}, {"set", "demo", "--platform", "freebsd"},
		{"set", "demo", "--platform", "macos"}, {"remove"}, {"validate"},
		{"prerequisites"}, {"discover"}, {"discover", "--platform", "freebsd"},
	} {
		if err := cmd.Run(args); err == nil {
			t.Errorf("expected error for %v", args)
		}
	}
	if err := cmd.Run([]string{"validate"}); err == nil {
		t.Fatal("expected owned-by-scenario-to-desktop refusal")
	}
}
