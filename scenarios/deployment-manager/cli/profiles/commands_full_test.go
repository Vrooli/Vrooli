package profiles

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandsCoverProfileLifecycle(t *testing.T) {
	server := newProfileCommandServer()
	defer server.Close()
	cmd := New(testAPIClient(server.URL))

	if err := cmd.List([]string{"--format", "json"}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if err := cmd.List([]string{"--format", "table"}); err != nil {
		t.Fatalf("list table: %v", err)
	}
	if err := cmd.Create([]string{"demo", "example", "--tier", "prod"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := cmd.Show([]string{"demo", "--format", "json"}); err != nil {
		t.Fatalf("show json: %v", err)
	}
	if err := cmd.Show([]string{"demo"}); err != nil {
		t.Fatalf("show table: %v", err)
	}
	if err := cmd.Update([]string{"demo", "--tier", "3"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := cmd.Update([]string{"demo", "--format", "json"}); err != nil {
		t.Fatalf("update json: %v", err)
	}
	if err := cmd.Set([]string{"demo", "tier", "1"}); err != nil {
		t.Fatalf("set tier: %v", err)
	}
	if err := cmd.Set([]string{"demo", "env", "MODE", "production"}); err != nil {
		t.Fatalf("set env: %v", err)
	}
	if err := cmd.Set([]string{"demo", "region", "us-east"}); err != nil {
		t.Fatalf("set custom: %v", err)
	}
	if err := cmd.Swap([]string{"demo", "add", "postgres", "sqlite"}); err != nil {
		t.Fatalf("swap add: %v", err)
	}
	if err := cmd.Swap([]string{"demo", "set", "postgres", "postgres-ha"}); err != nil {
		t.Fatalf("swap set: %v", err)
	}
	if err := cmd.Swap([]string{"demo", "remove", "postgres"}); err != nil {
		t.Fatalf("swap remove: %v", err)
	}
	if err := cmd.Versions([]string{"demo", "--format", "json"}); err != nil {
		t.Fatalf("versions json: %v", err)
	}
	if err := cmd.Versions([]string{"demo"}); err != nil {
		t.Fatalf("versions table: %v", err)
	}
	if err := cmd.Analyze([]string{"demo", "--format", "json"}); err != nil {
		t.Fatalf("analyze json: %v", err)
	}
	if err := cmd.Analyze([]string{"demo"}); err != nil {
		t.Fatalf("analyze table: %v", err)
	}
	if err := cmd.Save([]string{"demo"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := cmd.Diff([]string{"demo", "--format", "table"}); err != nil {
		t.Fatalf("diff table: %v", err)
	}
	if err := cmd.Diff([]string{"demo", "--format", "json"}); err != nil {
		t.Fatalf("diff json: %v", err)
	}
	if err := cmd.Rollback([]string{"demo", "--to-version", "1"}); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	exportPath := filepath.Join(t.TempDir(), "profile.json")
	if err := cmd.Export([]string{"demo", "--output", exportPath}); err != nil {
		t.Fatalf("export: %v", err)
	}
	if data, err := os.ReadFile(exportPath); err != nil || !strings.Contains(string(data), "example") {
		t.Fatalf("exported profile missing expected data: %q, %v", string(data), err)
	}
	if err := cmd.Import([]string{exportPath, "--name", "imported"}); err != nil {
		t.Fatalf("import: %v", err)
	}

	if err := cmd.Secrets([]string{"identify", "demo", "--format", "json"}); err != nil {
		t.Fatalf("secrets identify json: %v", err)
	}
	if err := cmd.Secrets([]string{"identify", "demo"}); err != nil {
		t.Fatalf("secrets identify table: %v", err)
	}
	if err := cmd.Secrets([]string{"template", "demo", "--format", "json"}); err != nil {
		t.Fatalf("secrets template json: %v", err)
	}
	if err := cmd.Secrets([]string{"validate", "demo", "--format", "json"}); err != nil {
		t.Fatalf("secrets validate json: %v", err)
	}
	if err := cmd.Delete([]string{"demo"}); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestCommandsRejectMalformedOrMissingInput(t *testing.T) {
	cmd := New(testAPIClient("http://127.0.0.1:1"))
	cases := []struct {
		name string
		fn   func() error
	}{
		{"list flags", func() error { return cmd.List([]string{"--bad"}) }},
		{"create args", func() error { return cmd.Create(nil) }},
		{"show args", func() error { return cmd.Show(nil) }},
		{"delete args", func() error { return cmd.Delete(nil) }},
		{"export args", func() error { return cmd.Export(nil) }},
		{"import args", func() error { return cmd.Import(nil) }},
		{"update args", func() error { return cmd.Update(nil) }},
		{"set args", func() error { return cmd.Set(nil) }},
		{"swap args", func() error { return cmd.Swap(nil) }},
		{"versions args", func() error { return cmd.Versions(nil) }},
		{"analyze args", func() error { return cmd.Analyze(nil) }},
		{"save args", func() error { return cmd.Save(nil) }},
		{"diff args", func() error { return cmd.Diff(nil) }},
		{"rollback args", func() error { return cmd.Rollback([]string{"demo"}) }},
		{"secrets args", func() error { return cmd.Secrets(nil) }},
		{"unknown secrets", func() error { return cmd.Secrets([]string{"unknown"}) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func newProfileCommandServer() *httptest.Server {
	profile := `{"id":"demo","name":"example","scenario":"scenario-a","tiers":[2],"swaps":{"postgres":"sqlite"},"secrets":{"TOKEN":"secret"},"settings":{"env":{"MODE":"test"},"region":"us"},"version":2}`
	history := `{"profile_id":"demo","versions":[{"id":"demo","name":"example-old","scenario":"scenario-a","tiers":[1],"swaps":{},"secrets":{},"settings":{},"version":1},{"id":"demo","name":"example","scenario":"scenario-a","tiers":[2],"swaps":{"postgres":"sqlite"},"secrets":{"TOKEN":"secret"},"settings":{"env":{"MODE":"test"}},"version":2}]}`
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/profiles":
			_, _ = io.WriteString(w, "["+profile+"]")
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/versions"):
			_, _ = io.WriteString(w, history)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/secrets/template"):
			_, _ = io.WriteString(w, `{"template":"TOKEN=secret\n"}`)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/secrets"):
			_, _ = io.WriteString(w, `{"TOKEN":"secret"}`)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/secrets/validate"):
			_, _ = io.WriteString(w, `{"valid":true}`)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/profiles/"):
			_, _ = io.WriteString(w, profile)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/profiles":
			_, _ = io.WriteString(w, profile)
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/api/v1/profiles/"):
			body, _ := io.ReadAll(r.Body)
			if len(body) == 0 {
				body = []byte(profile)
			}
			_, _ = w.Write(body)
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/profiles/"):
			_, _ = io.WriteString(w, `{"deleted":true}`)
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestProfileTypesHandleAllSwapForms(t *testing.T) {
	var swaps Swaps
	if err := json.Unmarshal([]byte(`[{"from":"a","to":"b"}]`), &swaps); err != nil {
		t.Fatal(err)
	}
	swaps.set("a", "c")
	swaps.set("d", "e")
	swaps.remove("a")
	if len(swaps) != 1 || swaps[0].To != "e" {
		t.Fatalf("unexpected swaps: %+v", swaps)
	}
	if err := json.Unmarshal([]byte(`null`), &swaps); err != nil || len(swaps) != 0 {
		t.Fatalf("null swaps: %v %+v", err, swaps)
	}
	if err := json.Unmarshal([]byte(`"bad"`), &swaps); err == nil {
		t.Fatal("expected unsupported swaps format error")
	}
	if _, err := json.Marshal(Swaps{{From: "a", To: "b"}}); err != nil {
		t.Fatal(err)
	}
}
