package executions

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

const testExecutionID = "8f14e45f-ea0b-4b6b-a501-2c2d7f45a102"

func purgeTestApp(t *testing.T, handler http.HandlerFunc) *cliapp.ScenarioApp {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	t.Setenv("BAS_PURGE_CLI_TEST_CONFIG", t.TempDir())
	app, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name: "bas-purge-cli-test", DefaultAPIBase: server.URL, APIPrefix: "/api/v1",
		AllowAnonymous: true, ConfigDirEnvVars: []string{"BAS_PURGE_CLI_TEST_CONFIG"},
	})
	if err != nil {
		t.Fatalf("new CLI app: %v", err)
	}
	return app
}

func commandByName(t *testing.T, commands []cliapp.Command, name string) cliapp.Command {
	t.Helper()
	for _, command := range commands {
		if command.Name == name {
			return command
		}
	}
	t.Fatalf("missing command %q", name)
	return cliapp.Command{}
}

func TestPurgePreviewUsesExactRouteAndPrintsMetadataOnly(t *testing.T) {
	app := purgeTestApp(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/executions/"+testExecutionID+"/purge-preview" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"execution_id":"` + testExecutionID + `","artifact_files":2,"artifact_bytes":99,"artifact_sha256":"abc","status":"completed","ready":true,"private_path":"must-not-print"}`))
	})
	command := commandByName(t, purgeCommands(app), "purge-preview")
	var stdout, stderr bytes.Buffer
	rc, err := cliapp.NewTestRunContextFromArgs(command.Args, []string{testExecutionID}, app, &stdout, &stderr)
	if err != nil {
		t.Fatalf("parse args: %v", err)
	}
	if err := command.RunCtx(rc); err != nil {
		t.Fatalf("run: %v", err)
	}
	if strings.Contains(stdout.String(), "private_path") {
		t.Fatalf("printed non-metadata response: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"artifact_files": 2`) {
		t.Fatalf("missing approved metadata: %s", stdout.String())
	}
}

func TestPurgeRequiresMatchingConfirmationAndPreviewMetadata(t *testing.T) {
	calls := 0
	app := purgeTestApp(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path == "/api/v1/executions/"+testExecutionID+"/purge-preview" {
			_, _ = w.Write([]byte(`{"execution_id":"` + testExecutionID + `","artifact_files":1,"artifact_bytes":5,"artifact_sha256":"def","status":"completed","ready":true}`))
			return
		}
		if r.URL.Path != "/api/v1/executions/"+testExecutionID+"/purge" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body struct {
			Confirmation string       `json:"confirmation"`
			Preview      purgePreview `json:"preview"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if body.Confirmation != testExecutionID || body.Preview.ArtifactBytes != 5 {
			t.Errorf("purge body = %+v", body)
		}
		_, _ = w.Write([]byte(`{"execution_id":"` + testExecutionID + `","artifact_files":1,"artifact_bytes":5,"artifact_sha256":"def","status":"completed","ready":true,"deleted":true}`))
	})
	command := commandByName(t, purgeCommands(app), "purge")
	var stdout bytes.Buffer
	bad, err := cliapp.NewTestRunContextFromArgs(command.Args, []string{testExecutionID, "--confirm", "00000000-0000-0000-0000-000000000000"}, app, &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("parse mismatched confirmation: %v", err)
	}
	if err := command.RunCtx(bad); err == nil {
		t.Fatal("expected mismatch to refuse purge")
	}
	if calls != 0 {
		t.Fatalf("mismatched confirmation made %d API calls", calls)
	}
	rc, err := cliapp.NewTestRunContextFromArgs(command.Args, []string{testExecutionID, "--confirm", testExecutionID}, app, &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("parse args: %v", err)
	}
	if err := command.RunCtx(rc); err != nil {
		t.Fatalf("run: %v", err)
	}
	if calls != 2 {
		t.Fatalf("API calls = %d, want preview plus purge", calls)
	}
	if !strings.Contains(stdout.String(), `"deleted": true`) {
		t.Fatalf("missing deletion metadata: %s", stdout.String())
	}
}

func TestParseExecutionUUIDRejectsNonUUID(t *testing.T) {
	for _, value := range []string{"", "latest", "../all", testExecutionID + "x"} {
		if _, err := parseExecutionUUID(value); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
}
