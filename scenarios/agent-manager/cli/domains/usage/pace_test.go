package usage

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

const paceFixture = `{"generated_at":"2026-11-16T12:00:00Z","selected_tier":"use_by_thursday","selected_target_run_out_at":"2026-11-19T23:59:59Z","baseline_max_concurrent_runs":10,"effective_max_concurrent_runs":5,"tiers":[{"id":"at_reset","label":"Last until quota resets","status":"available","available_slots":8,"effective_max_concurrent_runs":8,"target_run_out_at":"2026-11-20T07:00:00Z","unavailable_scopes":[{"provider":"anthropic","pool":"seven_day","window":"10080m","reason":"anthropic seven_day: quota readings in the burn interval carry no source run IDs, so the burn cannot be tied to runs"}]},{"id":"use_by_thursday","label":"Use by Thursday","status":"available","available_slots":5,"effective_max_concurrent_runs":5,"target_run_out_at":"2026-11-19T23:59:59Z"},{"id":"use_by_wednesday","label":"Use by Wednesday","status":"unavailable","reason":"no fresh provider quota forecast is available to estimate slots"},{"id":"extra_allowance","label":"Use extra allowance","status":"unavailable","reason":"provider quota readings do not identify a separate extra-allowance budget"}]}`

func runPace(t *testing.T, base string, args ...string) (string, error) {
	t.Helper()
	cmd := paceCommand(func(*cliapp.ScenarioApp) (paceHTTPDoer, string) {
		return http.DefaultClient, base
	})
	var stdout, stderr bytes.Buffer
	ctx, err := cliapp.NewTestRunContextFromArgs(cmd.Args, args, nil, &stdout, &stderr)
	if err != nil {
		return "", err
	}
	if err := cmd.RunCtx(ctx); err != nil {
		return stdout.String(), err
	}
	return stdout.String(), nil
}

func TestPaceCommandReadsForecastAndPreservesUnknownSlots(t *testing.T) {
	var method, path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, paceFixture)
	}))
	defer server.Close()

	out, err := runPace(t, server.URL, "--json")
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodGet || path != usagePaceAPIPath {
		t.Fatalf("request = %s %s", method, path)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("--json is not the pace API response: %v\n%s", err, out)
	}
	if decoded["effective_max_concurrent_runs"] != float64(5) || len(decoded["tiers"].([]any)) != 4 {
		t.Fatalf("unexpected pace response: %+v", decoded)
	}

	method = ""
	out, err = runPace(t, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodGet || !strings.Contains(out, "Selected pace: use_by_thursday") || !strings.Contains(out, "unknown (no fresh provider quota forecast") || !strings.Contains(out, "unknown (provider quota readings do not identify") {
		t.Fatalf("human output hides selection or unknown tiers:\n%s", out)
	}
	if !strings.Contains(out, "not measured: anthropic seven_day (anthropic seven_day: quota readings in the burn interval carry no source run IDs") {
		t.Fatalf("human output hides a window the ceiling does not measure:\n%s", out)
	}
}

func TestPaceCommandSetAppliesTheSelectedTierOverAPI(t *testing.T) {
	var method, path, selected string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		if r.Method == http.MethodPut {
			var input struct {
				SelectedTier string `json:"selected_tier"`
			}
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Errorf("decode request: %v", err)
			}
			selected = input.SelectedTier
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, paceFixture)
	}))
	defer server.Close()

	out, err := runPace(t, server.URL, "set", "--tier", "use_by_wednesday", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPut || path != usagePaceAPIPath || selected != "use_by_wednesday" {
		t.Fatalf("request = %s %s selected=%q", method, path, selected)
	}
	if !strings.Contains(out, `"tiers"`) {
		t.Fatalf("set did not return the updated API object:\n%s", out)
	}
}
