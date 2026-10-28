package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"agent-manager/internal/goalhome"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
)

// goalhomeTestdata holds the shared goal-home fixtures; parser and rule tests
// live with the goalhome package.
const goalhomeTestdata = "../api/internal/goalhome/testdata"

func readGoalhomeFixture(t *testing.T, parts ...string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(append([]string{goalhomeTestdata}, parts...)...))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func writeFile(t *testing.T, path, text string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func exitCode(err error) int {
	var exit exitCodeError
	if errors.As(err, &exit) {
		return exit.code
	}
	if err != nil {
		return 1
	}
	return 0
}

// runEpochCheck runs the command and returns its stdout and exit status.
func runEpochCheck(t *testing.T, app *App, args ...string) (string, int) {
	t.Helper()
	code := 0
	output := captureStdout(t, func() error {
		code = exitCode(app.effortEpochCheck(args))
		return nil
	})
	return output, code
}

func TestParseTierWeightsWeighsSolAboveLuna(t *testing.T) {
	weights, err := parseTierWeights("")
	if err != nil || weights["sol"] != 10 || weights["luna"] != 1 || weights["terra"] != 5 {
		t.Fatalf("default tier weights wrong: %v %v", weights, err)
	}
	if weights, err = parseTierWeights("sol=12, luna=0.5"); err != nil || weights["sol"] != 12 || weights["luna"] != 0.5 {
		t.Fatalf("override weights = %v %v", weights, err)
	}
	if _, err := parseTierWeights("sol"); err == nil {
		t.Fatal("malformed weight entry must be refused")
	}
}

func TestEpochCheckReturnsStepBackExitCodeWithoutExiting(t *testing.T) {
	output, code := runEpochCheck(t, &App{}, filepath.Join(goalhomeTestdata, "epochs", "flatline.md"))
	if code != exitStepBack {
		t.Fatalf("want exit code %d, got %d", exitStepBack, code)
	}
	if !strings.Contains(output, "STEP_BACK flatline") {
		t.Fatalf("report not printed before the exit status: %s", output)
	}
}

func TestEpochEstimateOverrunsContinueWithoutWakingOrRefusingAcceptance(t *testing.T) {
	for _, fixture := range []string{"overrun.md", "growth-refactor.md", "growth-feature.md", "healthy.md"} {
		t.Run(fixture, func(t *testing.T) {
			services, recorder := newContractServices(t)
			app := &App{services: services}
			text := strings.Replace(readGoalhomeFixture(t, "epochs", fixture), "- Started: 2026-09-29T08:00:00Z", "- Started: 2000-01-01T00:00:00Z", 1)
			path := writeFile(t, filepath.Join(t.TempDir(), fixture), text)
			for _, acceptance := range []bool{false, true} {
				args := []string{path, "--overrun", "2", "--growth", "300", "--feature-factor", "1.5", "--wall-clock", "24h", "--wake-key", "orchestrator-run", "--effort", "effort:fixture"}
				if acceptance {
					args = append(args, "--acceptance")
				}
				output, code := runEpochCheck(t, app, args...)
				if code != 0 || strings.Contains(output, "STEP_BACK") || !strings.Contains(output, "OBSERVE (continue)") {
					t.Fatalf("estimates must report optimization and continue (acceptance=%v exit=%d):\n%s", acceptance, code, output)
				}
			}
			if requests := recorder.Requests(); len(requests) != 0 {
				t.Fatalf("estimate reviews must not interrupt the parent or issue lifecycle requests: %+v", requests)
			}
		})
	}
}

func TestEpochSpendIsAdvisoryAndUnavailableUsageStaysUnknown(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		known      bool
	}{
		{name: "over estimate", body: `{"tokensKnown":true,"nonCacheTokens":5000,"model":"gpt-6-luna"}`, status: http.StatusOK, known: true},
		{name: "unqualified", body: `{"tokensKnown":false,"nonCacheTokens":5000,"model":"gpt-6-luna"}`, status: http.StatusOK},
		{name: "unavailable", body: `{"message":"accounting unavailable"}`, status: http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/accounting") {
					w.WriteHeader(tc.status)
					_, _ = w.Write([]byte(tc.body))
					return
				}
				if strings.HasSuffix(r.URL.Path, "/11111111-1111-4111-8111-111111111111") {
					_, _ = w.Write([]byte(`{"run":{"id":"11111111-1111-4111-8111-111111111111","status":"RUN_STATUS_RUNNING"}}`))
					return
				}
				t.Errorf("an estimate must not issue a wake or lifecycle request: %s", r.URL.Path)
				http.NotFound(w, r)
			}))
			defer server.Close()
			api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
				return cliutil.APIBaseOptions{DefaultBase: server.URL}
			}, nil)
			app := &App{services: NewServices(api)}
			output, code := runEpochCheck(t, app, filepath.Join(goalhomeTestdata, "epochs", "healthy.md"),
				"--runs", "11111111-1111-4111-8111-111111111111", "--spend-threshold", "100", "--wake-key", "parent", "--effort", "effort:fixture", "--json")
			var report epochCheckReport
			if code != 0 || json.Unmarshal([]byte(output), &report) != nil || len(report.StepBack) != 0 {
				t.Fatalf("spend must not stop work or require parent disposition (exit %d): %s", code, output)
			}
			if tc.known {
				if report.WeightedTokens == nil || *report.WeightedTokens != 5000 || !strings.Contains(strings.Join(report.Review, " "), "spend:") {
					t.Fatalf("known overrun must remain visible as advisory: %s", output)
				}
			} else if report.WeightedTokens != nil || !strings.Contains(output, "spend-unavailable") {
				t.Fatalf("unknown usage must not acquire a spend verdict: %s", output)
			}
		})
	}
}

func TestLargeEpochNeedsNoBudgetAndIsNotPenalizedForItsSizeOrDuration(t *testing.T) {
	services, recorder := newContractServices(t)
	app := &App{services: services}
	text := strings.Replace(readGoalhomeFixture(t, "epochs", "overrun.md"), "- Estimate: 3 work units\n", "", 1)
	text = strings.ReplaceAll(text, "net runtime lines=-5", "net runtime lines=+5000")
	path := writeFile(t, filepath.Join(t.TempDir(), "E1.md"), text)
	output, code := runEpochCheck(t, app, path, "--wake-key", "parent", "--effort", "effort:fixture", "--json")
	var report epochCheckReport
	if code != 0 || json.Unmarshal([]byte(output), &report) != nil || len(report.StepBack) != 0 || len(report.Review) != 0 {
		t.Fatalf("large progressing work needs no budget or negative performance verdict (exit %d): %s", code, output)
	}
	if report.WorkUnits != 7 || report.NetRuntimeLines != 35000 || report.WeightedTokens != nil {
		t.Fatalf("retain progress/size facts without inventing budgets or token use: %s", output)
	}
	if requests := recorder.Requests(); len(requests) != 0 {
		t.Fatalf("optional metering must not be queried or interrupt work by default: %+v", requests)
	}
}

func TestEpochCheckAcceptedEpochNeverStepsBackOrWakes(t *testing.T) {
	services, recorder := newContractServices(t)
	app := &App{services: services}
	path := writeFile(t, filepath.Join(t.TempDir(), "E7.md"), readGoalhomeFixture(t, "epochs", "flatline.md")+"ACCEPTED 2026-09-29T09:40:00Z J02 passes\n")
	output, code := runEpochCheck(t, app, path, "--wake-key", "orchestrator-run")
	if code != 0 || strings.Contains(output, "STEP_BACK") || !strings.Contains(output, "Accepted epoch, reported only: flatline:") {
		t.Fatalf("an accepted epoch must report its triggers without a step-back (exit %d):\n%s", code, output)
	}
	if requests := recorder.Requests(); len(requests) != 0 {
		t.Fatalf("an accepted epoch must not wake the orchestrator: %+v", requests)
	}
}

func TestEpochCheckAcceptanceExitsFourOnUnmetGate(t *testing.T) {
	services, recorder := newContractServices(t)
	app := &App{services: services}
	typed := readGoalhomeFixture(t, "epochs", "typed-gates.md")
	met := writeFile(t, filepath.Join(t.TempDir(), "E26.md"), typed)
	output, code := runEpochCheck(t, app, met, "--acceptance", "--wake-key", "orchestrator-run")
	if code != 0 || !strings.Contains(output, "ACCEPTANCE ok") || !strings.Contains(output, "Yield: -801 runtime_lines (inventory gate G4)") {
		t.Fatalf("met gates must pass acceptance (exit %d):\n%s", code, output)
	}
	unmet := writeFile(t, filepath.Join(t.TempDir(), "E26.md"), strings.Replace(typed, "gate G4=pass 252651", "gate G4=pass 253900", 1))
	output, code = runEpochCheck(t, app, unmet, "--acceptance")
	if code != exitRefused || !strings.Contains(output, "ACCEPTANCE_BLOCKED G4 inventory: runtime_lines 253900 is not <= 253452") {
		t.Fatalf("an unmet gate must exit %d and name the gate (exit %d):\n%s", exitRefused, code, output)
	}
	legacy := filepath.Join(goalhomeTestdata, "bas", "epochs", "E26.md")
	output, code = runEpochCheck(t, app, legacy, "--acceptance")
	if code != 0 || !strings.Contains(output, "WARNING legacy-gates") || !strings.Contains(output, "ACCEPTANCE not checked: legacy exit gate") {
		t.Fatalf("a legacy exit gate warns and never fails (exit %d):\n%s", code, output)
	}
	if requests := recorder.Requests(); len(requests) != 0 {
		t.Fatalf("acceptance mode never wakes: %+v", requests)
	}
}

func TestEpochCheckDiminishingReturnsFailsOnlyTheOrchestrator(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, "GOAL.md"), "# Goal\n\n- Destination: runtime lines <= 205000\n- Current: 253452 @ 2026-10-07\n")
	writeFile(t, filepath.Join(home, "QUEUE.md"), "# Queue\n\n## Needs operator\n\n## Handoff\n\nE24 admitted.\n")
	writeFile(t, filepath.Join(home, "epochs", "E23.md"), readGoalhomeFixture(t, "bas", "epochs", "E23.md"))
	writeFile(t, filepath.Join(home, "epochs", "E24.md"), strings.Replace(readGoalhomeFixture(t, "bas", "epochs", "E24.md"), "ACCEPTED 2026-10-07T00:42:30Z —", "ACCEPTED 2026-10-07T00:42:30Z | yield=-66 runtime lines |", 1))
	worker := writeFile(t, filepath.Join(home, "epochs", "E25.md"), strings.Replace(readGoalhomeFixture(t, "bas", "epochs", "E25.md"), "ACCEPTED 2026-10-07T01:28:54Z —", "", 1))
	legacy, err := goalhome.LoadLegacy(os.DirFS(home))
	if err != nil {
		t.Fatal(err)
	}
	read := func(name string) []byte { b, _ := os.ReadFile(filepath.Join(home, name)); return b }
	converted := goalhome.ConvertHome(legacy, read(goalhome.GoalFile), read(goalhome.QueueFile), nil)
	if !converted.Equivalent {
		t.Fatal(converted.Divergence)
	}
	for name, body := range map[string][]byte{goalhome.CharterFile: converted.Charter, goalhome.StatusFile: converted.Status, goalhome.LogFile: converted.Log} {
		writeFile(t, filepath.Join(home, name), string(body))
	}
	output, code := runEpochCheck(t, &App{}, worker)
	if code != 0 || !strings.Contains(output, "FINDING diminishing-returns: gap-horizon:") {
		t.Fatalf("a worker sees the finding without failing (exit %d):\n%s", code, output)
	}
	output, code = runEpochCheck(t, &App{}, worker, "--acceptance")
	if code != exitRefused || !strings.Contains(output, "ACCEPTANCE_BLOCKED diminishing-returns") {
		t.Fatalf("the orchestrator's acceptance check must fail on it (exit %d):\n%s", code, output)
	}
}

func TestRunWakeByKeyAsksTheServerToMatchProducerAndKey(t *testing.T) {
	services, recorder := newContractServices(t)
	app := &App{services: services}
	if err := app.runWake([]string{"--key", "orchestrator-run", "--result", "STEP_BACK flatline"}); err != nil {
		t.Fatal(err)
	}
	requests := recorder.Requests()
	if len(requests) != 1 || requests[0].Method != "POST" || requests[0].Path != "/api/v1/runs/wake-by-key" {
		t.Fatalf("wake by key must be one server-side lookup, got %+v", requests)
	}
}

func TestEpochCheckRecordsBeforeWakingOnlyForNewStepBackSignal(t *testing.T) {
	var mu sync.Mutex
	order := []string{}
	keys := map[string]bool{}
	keyCalls := []string{}
	seq := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.URL.Path {
		case apiconnect.AgentManagerServiceRecordEffortSignalProcedure:
			var request struct {
				DedupeKey string `json:"dedupe_key"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode signal request: %v", err)
			}
			if request.DedupeKey == "" {
				t.Errorf("record request omitted its dedupe_key")
			}
			order = append(order, "record")
			keyCalls = append(keyCalls, request.DedupeKey)
			newlyRecorded := !keys[request.DedupeKey]
			if newlyRecorded {
				keys[request.DedupeKey] = true
				seq++
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"seq":"%d","effortRef":"effort:fixture","kind":"step_back","newlyRecorded":%t}`, seq, newlyRecorded)))
		case "/api/v1/runs/wake-by-key":
			order = append(order, "wake")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"wokenRunIds":["orchestrator-run"]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL}
	}, nil)
	app := &App{services: NewServices(api)}
	path := writeFile(t, filepath.Join(t.TempDir(), "E7.md"), readGoalhomeFixture(t, "epochs", "flatline.md"))

	for attempt := 0; attempt < 3; attempt++ {
		if attempt == 2 {
			writeFile(t, path, readGoalhomeFixture(t, "epochs", "flatline.md")+"2026-09-29T09:40:00Z | unit 10 | exit metric=3 | net runtime lines=-5 | net test lines=+2 | ack=-\n")
		}
		output, code := runEpochCheck(t, app, path, "--effort", "effort:fixture", "--wake-key", "orchestrator-run")
		if code != exitStepBack || !strings.Contains(output, "STEP_BACK flatline") {
			t.Fatalf("attempt %d must preserve the step-back result (exit %d):\n%s", attempt, code, output)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if strings.Join(order, ",") != "record,wake,record,record,wake" {
		t.Fatalf("signal must be recorded before first/new wake, and replay must not wake: %v", order)
	}
	if len(keyCalls) != 3 || keyCalls[0] != keyCalls[1] || keyCalls[1] == keyCalls[2] {
		t.Fatalf("unchanged attempt must reuse its dedupe key and changed log must create a new one: %v", keyCalls)
	}
}

func TestEpochCheckFailsOpenWhenRecordResultIsAbsentOrUnavailable(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		code int
	}{
		{name: "older response", body: `{"effortRef":"effort:fixture","seq":"1","kind":"step_back"}`, code: http.StatusOK},
		{name: "record error", body: `{"message":"unavailable"}`, code: http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			order := []string{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				switch r.URL.Path {
				case apiconnect.AgentManagerServiceRecordEffortSignalProcedure:
					order = append(order, "record")
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tc.code)
					_, _ = w.Write([]byte(tc.body))
				case "/api/v1/runs/wake-by-key":
					order = append(order, "wake")
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"wokenRunIds":[]}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
				return cliutil.APIBaseOptions{DefaultBase: server.URL}
			}, nil)
			app := &App{services: NewServices(api)}
			path := writeFile(t, filepath.Join(t.TempDir(), "E7.md"), readGoalhomeFixture(t, "epochs", "flatline.md"))
			_, code := runEpochCheck(t, app, path, "--effort", "effort:fixture", "--wake-key", "orchestrator-run")
			if code != exitStepBack {
				t.Fatalf("step-back must retain exit %d, got %d", exitStepBack, code)
			}
			mu.Lock()
			defer mu.Unlock()
			if strings.Join(order, ",") != "record,wake" {
				t.Fatalf("missing/failed result must preserve alert wake: %v", order)
			}
		})
	}
}

// TestGoalFindingsReadsV2Home proves epoch-check's diminishing-returns forecast
// reads a migrated v2 home (CHARTER/STATUS/LOG). Before the reader was routed
// through the format sniff, goalFindings used v1 Load, saw GOAL.md missing, and
// silently skipped the forecast for every v2 home.
func TestGoalFindingsReadsV2Home(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, "CHARTER.md"), "# T\n\n- Destination: runtime lines <= 100\n- Destination set: 2026-10-01T00:00:00Z\n")
	writeFile(t, filepath.Join(home, "STATUS.md"), "# S\n\n## Current: 1000 @ 2026-10-07\n\n## Handoff\n\ncontinue.\n")
	writeFile(t, filepath.Join(home, "LOG.md"), "# L\n\n## Accepted units\n\n"+
		"- E1 | ACCEPTED 2026-10-02T00:00:00Z | workers=11111111-1111-4111-8111-111111111111 | yield=-10 runtime lines | gates=G1 | s1\n"+
		"- E2 | ACCEPTED 2026-10-03T00:00:00Z | workers=22222222-2222-4222-8222-222222222222 | yield=-10 runtime lines | gates=G1 | s2\n"+
		"- E3 | ACCEPTED 2026-10-04T00:00:00Z | workers=33333333-3333-4333-8333-333333333333 | yield=-10 runtime lines | gates=G1 | s3\n")
	writeFile(t, filepath.Join(home, "FEEDBACK.md"), "# F\n\n## Resolved\n")
	epoch := writeFile(t, filepath.Join(home, "epochs", "E4.md"), "# E4\n\n- Status: in-progress\n\n## Slice log\n")

	findings := goalFindings(epoch)
	if len(findings) != 1 || findings[0].Code != "diminishing-returns" {
		var codes []string
		for _, f := range findings {
			codes = append(codes, f.Code)
		}
		t.Fatalf("v2 home gap-horizon forecast must surface a diminishing-returns finding, got %v", codes)
	}
	if findings[0].File != "STATUS.md" {
		t.Fatalf("v2 forecast finding must anchor STATUS.md, got %q", findings[0].File)
	}
}

func TestTokenReportLeavesUnmatchedModelWeightUnknown(t *testing.T) {
	report := &tokenReport{}
	report.add(tokenRow{RunID: "unknown-model", Model: "unrecognized", TokensKnown: true, NonCacheTokens: 100}, defaultTierWeights)
	if len(report.WeightUnknown) != 1 || report.Rows[0].Weight != nil || report.Rows[0].Weighted != nil {
		t.Fatalf("unmatched model silently compared as Luna: %+v", report)
	}
	report.add(tokenRow{RunID: "terra", Model: "gpt-6.1-terra", TokensKnown: true, NonCacheTokens: 100}, defaultTierWeights)
	if report.TotalWeighted != 500 {
		t.Fatalf("Terra heuristic missing: %+v", report)
	}
}
