package handlers_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/testutil/mocks/procmocks"
)

func TestHostVrooliCLIRefreshRunsOnlyTypedRootCommand(t *testing.T) {
	starter := procmocks.NewFakeStarter()
	starter.SetDefault(procmocks.CommandBehavior{Exit: process.ProcessExit{ExitCode: 0}})
	live := newLive(t, nil, withStarter(starter))
	t.Setenv("VROOLI_ROOT", "/host/repo")
	t.Setenv("VROOLI_SANDBOX_ID", "sandbox-id")
	t.Setenv("VROOLI_SANDBOX_MERGED", "/sandbox/merged")
	t.Setenv("VROOLI_SANDBOX_SCOPE", "scenarios/scenario-qa")
	t.Setenv("VROOLI_AGENT_IDENTITY_TOKEN", "keep-agent-attribution")

	body := `{"kind":"scenario","name":"scenario-qa","running_executable":"/home/agent/.vrooli/bin/scenario-qa"}`
	resp, response := live.DoJSON(t, http.MethodPost, "/api/v1/host/vrooli/cli-refresh", body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, response)
	}
	var result struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(response, &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !result.Success {
		t.Fatalf("response = %s, want success", response)
	}
	calls := starter.MatchedCalls("vrooli cli-refresh ")
	if len(calls) != 1 {
		t.Fatalf("root cli-refresh calls=%d, want exactly one", len(calls))
	}
	call := calls[0]
	wantArgs := []string{"cli-refresh", "scenario", "scenario-qa", "/home/agent/.vrooli/bin/scenario-qa"}
	if !reflect.DeepEqual(call.Args, wantArgs) {
		t.Fatalf("argv=%q, want %q", call.Args, wantArgs)
	}
	if call.Dir != "/host/repo" {
		t.Fatalf("working directory=%q, want host root", call.Dir)
	}
	for _, item := range call.Env {
		key, _, _ := strings.Cut(item, "=")
		if key == "VROOLI_SANDBOX_ID" || key == "VROOLI_SANDBOX_MERGED" || key == "VROOLI_SANDBOX_SCOPE" {
			t.Fatalf("sandbox env marker leaked into host command: %s", key)
		}
	}
	if !strings.Contains(strings.Join(call.Env, "\n"), "VROOLI_AGENT_IDENTITY_TOKEN=keep-agent-attribution") {
		t.Fatal("verified agent identity attribution was stripped from host environment")
	}
}

func TestHostVrooliCLIRefreshRejectsInvalidInputBeforeStartingProcess(t *testing.T) {
	for name, body := range map[string]string{
		"unknown field":       `{"kind":"scenario","name":"scenario-qa","running_executable":"/bin/scenario-qa","args":["sh"]}`,
		"extra JSON object":   `{"kind":"scenario","name":"scenario-qa","running_executable":"/bin/scenario-qa"}{}`,
		"unknown kind":        `{"kind":"shell","name":"scenario-qa","running_executable":"/bin/scenario-qa"}`,
		"relative executable": `{"kind":"scenario","name":"scenario-qa","running_executable":"scenario-qa"}`,
	} {
		t.Run(name, func(t *testing.T) {
			starter := procmocks.NewFakeStarter()
			live := newLive(t, nil, withStarter(starter))
			resp, response := live.DoJSON(t, http.MethodPost, "/api/v1/host/vrooli/cli-refresh", body)
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s, want bad request", resp.StatusCode, response)
			}
			if len(starter.Calls) != 0 {
				t.Fatalf("invalid request started host process: %+v", starter.Calls)
			}
		})
	}
}
