package soak

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
)

const passingRunResponse = `{"schemaVersion":1,"runId":"run-123","lane":"realtime","assertions":[{"name":"terminal","outcome":"passed","detail":"done"}]}`

func soakRunSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "lane"}, {Name: "profile"}, {Name: "turns"}, {Name: "feed-ms"},
		{Name: "fault"}, {Name: "reference-text"}, {Name: "reference-cycle-ms"},
		{Name: "engine-id"}, {Name: "model-id"}, {Name: "strategy"}, {Name: "policy"},
		{Name: "shape"}, {Name: "simulated-minutes"}, {Name: "mobile-recovery"},
		{Name: "evidence-path"},
	}}
}

func newSoakTestHandlers(t *testing.T, response string, service string, observe func(*http.Request, []byte)) (*handlers, *httptest.Server) {
	t.Helper()
	scenarioRoot := t.TempDir()
	if service == "" {
		service = "audio-tools-api"
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"service":%q}`, service)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request", http.StatusBadRequest)
			return
		}
		if observe != nil {
			observe(r, body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(srv.Close)
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name: "audio-tools", DefaultAPIBase: srv.URL, AllowAnonymous: true,
	})
	require.NoError(t, err)
	h := &handlers{
		core: core,
		getenv: func(key string) string {
			values := map[string]string{
				"VROOLI_AGENT_IDENTITY_TOKEN":        "identity-test",
				"VROOLI_BROWSER_GRANT_RECEIPT":       "grant-test",
				"TEST_GENIE_DEMAND_LEASE_RECEIPT_ID": "lease-test",
				"AUDIO_TOOLS_SCENARIO_ROOT":          scenarioRoot,
			}
			return values[key]
		},
		getwd: func() (string, error) { return "/unused", nil },
	}
	return h, srv
}

func capturedRun(t *testing.T, app *cliapp.ScenarioApp, options cliapp.TestRunContextOptions) (cliapp.RunContext, *bytes.Buffer) {
	t.Helper()
	ctx, output := cliapptest.NewCapturedRunContext(app, soakRunSchema(), options)
	return ctx, output
}

func TestRegisterDeclaresOwnerAuthorizedSoakCommand(t *testing.T) {
	group := Register(&cliapp.ScenarioApp{}, nil, nil)
	require.Equal(t, GroupName, group.Name)
	require.Len(t, group.Subcommands, 1)
	command := group.Subcommands[0]
	require.Equal(t, "run", command.Name)
	require.NotNil(t, command.RunCtx)
	flagNames := make(map[string]bool, len(command.Args.Flags))
	for _, flag := range command.Args.Flags {
		flagNames[flag.Name] = true
	}
	for _, name := range []string{"engine-id", "model-id", "lane", "evidence-path"} {
		require.True(t, flagNames[name], "missing --%s flag", name)
	}
}

func TestRunRejectsMissingAuthorityAndProviderCellBeforeAPI(t *testing.T) {
	h, _ := newSoakTestHandlers(t, passingRunResponse, "audio-tools-api", nil)
	app := h.core
	tests := []struct {
		name    string
		env     map[string]string
		flags   map[string]string
		message string
	}{
		{name: "missing identity", env: map[string]string{"VROOLI_BROWSER_GRANT_RECEIPT": "grant", "TEST_GENIE_DEMAND_LEASE_RECEIPT_ID": "lease"}, flags: map[string]string{"engine-id": "kyutai", "model-id": "stt-1b"}, message: "trusted Agent Manager identity"},
		{name: "missing browser grant", env: map[string]string{"VROOLI_AGENT_IDENTITY_TOKEN": "identity", "TEST_GENIE_DEMAND_LEASE_RECEIPT_ID": "lease"}, flags: map[string]string{"engine-id": "kyutai", "model-id": "stt-1b"}, message: "browser grant"},
		{name: "missing test lease", env: map[string]string{"VROOLI_AGENT_IDENTITY_TOKEN": "identity", "VROOLI_BROWSER_GRANT_RECEIPT": "grant"}, flags: map[string]string{"engine-id": "kyutai", "model-id": "stt-1b"}, message: "Test Genie demand lease"},
		{name: "missing engine", env: map[string]string{"VROOLI_AGENT_IDENTITY_TOKEN": "identity", "VROOLI_BROWSER_GRANT_RECEIPT": "grant", "TEST_GENIE_DEMAND_LEASE_RECEIPT_ID": "lease"}, flags: map[string]string{"model-id": "stt-1b"}, message: "--engine-id and --model-id"},
		{name: "missing model", env: map[string]string{"VROOLI_AGENT_IDENTITY_TOKEN": "identity", "VROOLI_BROWSER_GRANT_RECEIPT": "grant", "TEST_GENIE_DEMAND_LEASE_RECEIPT_ID": "lease"}, flags: map[string]string{"engine-id": "kyutai"}, message: "--engine-id and --model-id"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, _ := capturedRun(t, app, cliapp.TestRunContextOptions{Flags: test.flags})
			h.getenv = func(key string) string { return test.env[key] }
			err := h.run(ctx)
			require.ErrorContains(t, err, test.message)
		})
	}
}

func TestRunBuildsAuthorizedRequestAndWritesScenarioEvidence(t *testing.T) {
	type observedRequest struct {
		method string
		header http.Header
		body   []byte
	}
	observed := make(chan observedRequest, 1)
	h, _ := newSoakTestHandlers(t, passingRunResponse, "audio-tools-api", func(r *http.Request, body []byte) {
		observed <- observedRequest{method: r.Method, header: r.Header.Clone(), body: append([]byte(nil), body...)}
	})
	root := t.TempDir()
	h.getenv = func(key string) string {
		values := map[string]string{
			"VROOLI_AGENT_IDENTITY_TOKEN":        "identity-test",
			"VROOLI_BROWSER_GRANT_RECEIPT":       "grant-test",
			"TEST_GENIE_DEMAND_LEASE_RECEIPT_ID": "lease-test",
			"AUDIO_TOOLS_SCENARIO_ROOT":          root,
		}
		return values[key]
	}
	ctx, output := capturedRun(t, h.core, cliapp.TestRunContextOptions{
		Flags: map[string]string{
			"lane": "accelerated", "turns": "2", "simulated-minutes": "45",
			"engine-id": "kyutai", "model-id": "stt-1b-en_fr", "fault": "backend_restart",
			"evidence-path": "coverage/soak-run.json", "mobile-recovery": "true",
		},
	})
	require.NoError(t, h.run(ctx))
	req := <-observed
	var request map[string]any
	require.NoError(t, json.Unmarshal(req.body, &request))
	require.Equal(t, http.MethodPost, req.method)
	require.Equal(t, "identity-test", req.header.Get("X-Agent-Identity-Token"))
	require.Equal(t, "grant-test", req.header.Get("X-Vrooli-Browser-Grant-Receipt"))
	require.Equal(t, "lease-test", req.header.Get("X-Test-Genie-Lease-Receipt"))
	require.Equal(t, "accelerated", request["lane"])
	require.Equal(t, float64(2), request["turns"])
	require.Equal(t, float64(50), request["feed_ms"], "accelerated lane uses the low wall-clock feed")
	require.Equal(t, float64(45), request["simulated_minutes"])
	require.Equal(t, "kyutai", request["engine_id"])
	require.Equal(t, "stt-1b-en_fr", request["model_id"])
	require.Equal(t, "backend_restart", request["fault"])
	require.Equal(t, true, request["mobile_recovery"])
	require.Contains(t, output.String(), "audio-tools soak qualified: run-123")
	require.Contains(t, output.String(), filepath.Join(root, "coverage", "soak-run.json"))
	contents, err := os.ReadFile(filepath.Join(root, "coverage", "soak-run.json"))
	require.NoError(t, err)
	require.JSONEq(t, passingRunResponse, strings.TrimSpace(string(contents)))
	info, err := os.Stat(filepath.Join(root, "coverage", "soak-run.json"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestRunJSONModeReturnsEvidenceAndRejectsUnqualifiedOutcomes(t *testing.T) {
	failing := `{"schemaVersion":1,"runId":"run-failed","lane":"realtime","assertions":[{"name":"latency","outcome":"failed","detail":"slow"},{"name":"device","outcome":"unmeasured","detail":"missing"}]}`
	h, _ := newSoakTestHandlers(t, failing, "audio-tools-api", nil)
	ctx, output := capturedRun(t, h.core, cliapp.TestRunContextOptions{JSON: true, Flags: map[string]string{"engine-id": "kyutai", "model-id": "stt-1b"}})
	err := h.run(ctx)
	require.ErrorContains(t, err, "2 assertion(s) failed or were not measured")
	require.JSONEq(t, failing, strings.TrimSpace(output.String()))
}

func TestRunHumanModePrintsFailedAssertionsAndDoesNotClaimQualification(t *testing.T) {
	failing := `{"schemaVersion":1,"runId":"run-failed","lane":"realtime","assertions":[{"name":"latency","outcome":"failed","detail":"slow"}]}`
	h, _ := newSoakTestHandlers(t, failing, "audio-tools-api", nil)
	ctx, output := capturedRun(t, h.core, cliapp.TestRunContextOptions{Flags: map[string]string{"engine-id": "kyutai", "model-id": "stt-1b"}})
	err := h.run(ctx)
	require.ErrorContains(t, err, "1 assertion(s) failed or were not measured")
	require.Contains(t, output.String(), "latency: failed (slow)")
	require.NotContains(t, output.String(), "audio-tools soak qualified")
}

func TestRunRejectsWrongScenarioAPIBeforeSubmittingSoak(t *testing.T) {
	h, _ := newSoakTestHandlers(t, passingRunResponse, "another-api", nil)
	ctx, _ := capturedRun(t, h.core, cliapp.TestRunContextOptions{Flags: map[string]string{"engine-id": "kyutai", "model-id": "stt-1b"}})
	err := h.run(ctx)
	require.ErrorContains(t, err, `identifies as "another-api", want "audio-tools-api"`)
}

func TestRunRejectsInvalidPositiveIntegerFlags(t *testing.T) {
	h, _ := newSoakTestHandlers(t, passingRunResponse, "audio-tools-api", nil)
	for _, test := range []struct{ flag, value string }{{"turns", "0"}, {"feed-ms", "-1"}, {"simulated-minutes", "oops"}, {"reference-cycle-ms", "0"}} {
		t.Run(test.flag, func(t *testing.T) {
			ctx, _ := capturedRun(t, h.core, cliapp.TestRunContextOptions{Flags: map[string]string{
				"engine-id": "kyutai", "model-id": "stt-1b", test.flag: test.value,
			}})
			err := h.run(ctx)
			require.ErrorContains(t, err, "must be a positive integer")
		})
	}
}

func TestScenarioRootDiscoversAudioToolsFromNestedWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	scenarioRoot := filepath.Join(root, "scenarios", "audio-tools")
	nested := filepath.Join(scenarioRoot, "cli", "domains", "soak")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	h := &handlers{getenv: func(string) string { return "" }, getwd: func() (string, error) { return nested, nil }}
	got, err := h.scenarioRoot()
	require.NoError(t, err)
	require.Equal(t, scenarioRoot, got)
}

func TestScenarioRootReportsWhenAudioToolsIsNotInParentTree(t *testing.T) {
	root := t.TempDir()
	h := &handlers{getenv: func(string) string { return "" }, getwd: func() (string, error) { return root, nil }}
	_, err := h.scenarioRoot()
	require.ErrorContains(t, err, "audio-tools scenario root not found")
}

func TestPositiveIntUsesDefaultsAndRejectsInvalidValues(t *testing.T) {
	got, err := positiveInt("  ", 7)
	require.NoError(t, err)
	require.Equal(t, 7, got)
	got, err = positiveInt(" 12 ", 7)
	require.NoError(t, err)
	require.Equal(t, 12, got)
	for _, raw := range []string{"0", "-2", "not-a-number", "999999999999999999999999999999"} {
		_, err := positiveInt(raw, 7)
		require.ErrorContains(t, err, "must be a positive integer")
	}
}

func TestFirstNonEmptyTrimsAndSelectsFirstValue(t *testing.T) {
	require.Equal(t, "selected", firstNonEmpty("  ", " selected ", "later"))
	require.Empty(t, firstNonEmpty("", "  "))
}
