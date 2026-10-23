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
	if err != nil || tierWeight("gpt-6-sol", weights) != 10 || tierWeight("gpt-6-luna", weights) != 1 || tierWeight("unknown-model", weights) != 1 {
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
