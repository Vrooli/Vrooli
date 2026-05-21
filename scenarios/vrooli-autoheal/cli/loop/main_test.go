package main

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
)

func TestBuildVrooliCmd_InjectsNoStaleCheck(t *testing.T) {
	cases := [][]string{
		{"scenario", "port", "vrooli-autoheal", "API_PORT"},
		{"scenario", "status", "vrooli-autoheal", "--json"},
		{"scenario", "start", "vrooli-autoheal", "--best-effort"},
		{"scenario", "restart", "vrooli-autoheal", "--best-effort"},
	}
	for _, sub := range cases {
		cmd := buildVrooliCmd("/tmp/vrooli", sub...)
		joined := strings.Join(cmd.Args, " ")
		if !strings.Contains(joined, "--no-stale-check") {
			t.Errorf("argv missing --no-stale-check for %v: %v", sub, cmd.Args)
			continue
		}
		idxFlag := indexOf(cmd.Args, "--no-stale-check")
		idxSub := indexOf(cmd.Args, sub[0])
		if idxFlag < 0 || idxSub < 0 || idxFlag > idxSub {
			t.Errorf("--no-stale-check must precede %q in %v", sub[0], cmd.Args)
		}
	}
	_ = runtime.GOOS
}

func indexOf(s []string, want string) int {
	for i, v := range s {
		if v == want {
			return i
		}
	}
	return -1
}

func TestRunTick_RequestsCompactResponse(t *testing.T) {
	t.Helper()

	var gotCompact string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		gotCompact = r.URL.Query().Get("compact")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"status":"ok","summary":{"total":1,"ok":1,"warning":0,"critical":0}}`))
	}))
	defer ts.Close()

	cfg := &Config{TickEndpoint: ts.URL + "/api/v1/tick"}
	result, err := runTick(cfg)
	if err != nil {
		t.Fatalf("runTick() error = %v", err)
	}
	if gotCompact != "true" {
		t.Fatalf("compact query param = %q, want true", gotCompact)
	}
	if result.Status != "ok" {
		t.Fatalf("status = %q, want ok", result.Status)
	}
}
