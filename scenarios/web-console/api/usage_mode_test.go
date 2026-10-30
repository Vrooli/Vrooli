package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/usagemode"
	capsH "web-console/handlers/capabilities"
	"web-console/internal/sessionstore"
)

// usageModeFixture points the mode file at a temp home and the resource
// policies at the repository's real ones, so the live system is never touched.
func usageModeFixture(t *testing.T) *Server {
	t.Helper()
	t.Setenv("VROOLI_HOME", t.TempDir())
	root := t.TempDir()
	for _, resource := range []string{"codex", "claude-code"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "..", "resources", resource, "model-policy.json"))
		if err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(root, "resources", resource)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "model-policy.json"), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("VROOLI_ROOT", root)
	s := &Server{router: mux.NewRouter()}
	s.mountUsageMode()
	return s
}

func usageModeCall(t *testing.T, s *Server, method, path, body string, kind identity.ActorKind) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.RemoteAddr = "127.0.0.1:4000"
	if kind != "" {
		req = req.WithContext(identity.WithPrincipal(req.Context(), identity.Principal{Kind: kind, Subject: "op@example.com", Verified: true, ExpiresAt: time.Now().Add(time.Hour)}))
	}
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)
	return rec
}

func decodeUsageMode(t *testing.T, rec *httptest.ResponseRecorder) usageModeResponse {
	t.Helper()
	var out usageModeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return out
}

func usageModeAgentFor(t *testing.T, r usageModeResponse, id string) usageModeAgent {
	t.Helper()
	for _, a := range r.Agents {
		if a.AgentID == id {
			return a
		}
	}
	t.Fatalf("agent %s missing in %+v", id, r.Agents)
	return usageModeAgent{}
}

func TestUsageModeNoModeChangesNothing(t *testing.T) {
	s := usageModeFixture(t)
	got := decodeUsageMode(t, usageModeCall(t, s, http.MethodGet, "/api/v1/usage-mode", "", ""))
	if len(got.Modes) != 0 || got.CanControl {
		t.Fatalf("no mode expected: %+v", got)
	}
	codex := usageModeAgentFor(t, got, "codex")
	if codex.Model != "" || codex.ModeID != "" || codex.FullPowerArgs != "" || codex.UsualModel == "" {
		t.Fatalf("no mode must show no model and no full-power offer: %+v", codex)
	}
	base := codexLaunchArgs(sessionstore.LaunchDescriptor{Agent: "codex"})
	if strings.Contains(strings.Join(base, " "), "model=") {
		t.Fatalf("no mode must leave managed codex args unchanged: %v", base)
	}
}

func TestUsageModeStartShowsModelEndsAndFullPowerIsOnePerLaunch(t *testing.T) {
	s := usageModeFixture(t)
	rec := usageModeCall(t, s, http.MethodPost, "/api/v1/usage-mode/start", `{"agentId":"codex","kind":"minutes","minutes":120}`, identity.ActorHuman)
	if rec.Code != http.StatusOK {
		t.Fatalf("start = %d %s", rec.Code, rec.Body.String())
	}
	got := decodeUsageMode(t, rec)
	if len(got.Modes) != 1 || got.Modes[0].Label != "Save usage" || got.Modes[0].ExpiresAt == "" || got.Modes[0].StartedBy != "op@example.com" {
		t.Fatalf("modes = %+v", got.Modes)
	}
	codex := usageModeAgentFor(t, got, "codex")
	if codex.Model != "gpt-6-luna" || codex.UsualModel != "gpt-6.1-terra" || codex.FullPowerArgs != "-m gpt-6.1-terra" {
		t.Fatalf("codex = %+v", codex)
	}
	// Managed Codex fills an empty model from the mode but never overrides one.
	if args := strings.Join(codexLaunchArgs(sessionstore.LaunchDescriptor{Agent: "codex"}), " "); !strings.Contains(args, `model="gpt-6-luna"`) {
		t.Fatalf("empty model not filled: %s", args)
	}
	if args := strings.Join(codexLaunchArgs(sessionstore.LaunchDescriptor{Agent: "codex", Model: "gpt-6.1-terra"}), " "); !strings.Contains(args, `model="gpt-6.1-terra"`) || strings.Contains(args, "luna") {
		t.Fatalf("explicit model overridden: %s", args)
	}
	// Full power is a per-launch argument: the mode is still there afterwards.
	again := decodeUsageMode(t, usageModeCall(t, s, http.MethodGet, "/api/v1/usage-mode", "", ""))
	if len(again.Modes) != 1 || usageModeAgentFor(t, again, "codex").Model != "gpt-6-luna" {
		t.Fatalf("mode must persist: %+v", again)
	}
	end := usageModeCall(t, s, http.MethodPost, "/api/v1/usage-mode/end", `{"id":"`+got.Modes[0].ID+`"}`, identity.ActorHuman)
	if end.Code != http.StatusOK || len(decodeUsageMode(t, end).Modes) != 0 {
		t.Fatalf("end = %d %s", end.Code, end.Body.String())
	}
	if rec := usageModeCall(t, s, http.MethodPost, "/api/v1/usage-mode/end", `{"id":"nope"}`, identity.ActorHuman); rec.Code != http.StatusNotFound {
		t.Fatalf("ending an unknown mode = %d", rec.Code)
	}
}

func TestUsageModeRefusesAgentsServicesAndAnonymous(t *testing.T) {
	s := usageModeFixture(t)
	for _, kind := range []identity.ActorKind{identity.ActorAgent, identity.ActorService, ""} {
		for _, path := range []string{"/api/v1/usage-mode/start", "/api/v1/usage-mode/end"} {
			rec := usageModeCall(t, s, http.MethodPost, path, `{"agentId":"codex","kind":"off","id":"x"}`, kind)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("%q %s = %d", kind, path, rec.Code)
			}
		}
	}
	if _, err := os.Stat(usagemode.Path()); !os.IsNotExist(err) {
		t.Fatalf("a refused request must not write the file: %v", err)
	}
	// Agents may read.
	if rec := usageModeCall(t, s, http.MethodGet, "/api/v1/usage-mode", "", identity.ActorAgent); rec.Code != http.StatusOK || decodeUsageMode(t, rec).CanControl {
		t.Fatalf("agent read = %d", rec.Code)
	}
}

func TestUsageModeUntilResetOnlyWithKnownReset(t *testing.T) {
	s := usageModeFixture(t)
	reset := time.Now().Add(3 * time.Hour).UTC().Truncate(time.Second)
	s.usageModeQuota = func(context.Context, string) (capsH.AgentUsageResult, error) {
		return capsH.AgentUsageResult{Match: "likely", Windows: []capsH.AgentUsageWindow{{WindowMinutes: 10080, ResetsAt: reset.Format(time.RFC3339)}}}, nil
	}
	body := `{"agentId":"codex","kind":"reset","windowMinutes":10080}`
	got := decodeUsageMode(t, usageModeCall(t, s, http.MethodPost, "/api/v1/usage-mode/start", body, identity.ActorHuman))
	if len(got.Modes) != 1 || got.Modes[0].UntilKind != usagemode.UntilWindowReset || got.Modes[0].ExpiresAt != reset.Format(time.RFC3339) {
		t.Fatalf("modes = %+v", got.Modes)
	}
	// An unknown window is refused, never guessed.
	if rec := usageModeCall(t, s, http.MethodPost, "/api/v1/usage-mode/start", `{"agentId":"codex","kind":"reset","windowMinutes":300}`, identity.ActorHuman); rec.Code != http.StatusConflict {
		t.Fatalf("unknown window = %d", rec.Code)
	}
	s.usageModeQuota = func(context.Context, string) (capsH.AgentUsageResult, error) {
		return capsH.AgentUsageResult{Match: "unavailable"}, nil
	}
	if rec := usageModeCall(t, s, http.MethodPost, "/api/v1/usage-mode/start", body, identity.ActorHuman); rec.Code != http.StatusConflict {
		t.Fatalf("unavailable quota = %d", rec.Code)
	}
}

func TestUsageModeCorruptFileFailsOpen(t *testing.T) {
	s := usageModeFixture(t)
	if err := os.MkdirAll(filepath.Dir(usagemode.Path()), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(usagemode.Path(), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	rec := usageModeCall(t, s, http.MethodGet, "/api/v1/usage-mode", "", "")
	got := decodeUsageMode(t, rec)
	if rec.Code != http.StatusOK || len(got.Modes) != 0 || usageModeAgentFor(t, got, "codex").Model != "" {
		t.Fatalf("corrupt file must mean no mode: %d %+v", rec.Code, got)
	}
	if args := strings.Join(codexLaunchArgs(sessionstore.LaunchDescriptor{Agent: "codex"}), " "); strings.Contains(args, "model=") {
		t.Fatalf("corrupt file changed args: %s", args)
	}
}

func TestUsageModeHigherModelIsTheTierAboveTheMode(t *testing.T) {
	tiers := usagemode.Tiers{SessionModel: "sonnet", ModelMap: map[string]string{"opus": "sonnet", "claude-opus-5": "claude-sonnet-5", "fable": "opus"}}
	if got := usageModeHigherModel(tiers, "sonnet"); got != "opus" {
		t.Fatalf("got %q", got)
	}
	if got := usageModeHigherModel(usagemode.Tiers{SessionModel: "x"}, "fallback"); got != "fallback" {
		t.Fatalf("got %q", got)
	}
}
