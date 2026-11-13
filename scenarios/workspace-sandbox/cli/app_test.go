package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapptest"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/workspace-sandbox/v1/workspace"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestMain(m *testing.M) { os.Exit(cliapptest.RunWithoutAgentSession(m)) }

func TestApprovalCommandsForwardReviewedPatch(t *testing.T) {
	for _, command := range []string{"approve", "promote"} {
		t.Run(command, func(t *testing.T) {
			digest := strings.Repeat("a", 64)
			calls := 0
			_ = cliapptest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/health" {
					_, _ = w.Write([]byte(`{"status":"healthy"}`))
					return
				}
				calls++
				var request workspacev1.PromoteSandboxRequest
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				if command == "promote" {
					err = proto.Unmarshal(body, &request)
				} else {
					err = protojson.Unmarshal(body, &request)
				}
				if err != nil || request.GetExpectedPatchSha256() != digest || request.GetMode() != "all" || !request.GetForce() {
					t.Errorf("CLI lost reviewed patch precondition: %v, %v", &request, err)
				}
				if command == "promote" {
					w.Header().Set("Content-Type", "application/proto")
					body, err := proto.Marshal(&workspacev1.PromoteSandboxResponse{Success: true, Applied: 1, AppliedPatchSha256: digest})
					if err != nil {
						t.Error(err)
					}
					_, _ = w.Write(body)
					return
				}
				_, _ = w.Write([]byte(`{"success":true,"applied":1,"appliedPatchSha256":"` + digest + `"}`))
			}))
			t.Setenv("CLI_CONFIG_DIR_OVERRIDE", t.TempDir())
			app, err := NewApp()
			if err != nil {
				t.Fatal(err)
			}
			args := []string{"change", command, "bd34c03c-e2bf-4f24-97fb-0f078570fb8a", "--force", "--expected-patch-sha256", digest, "--json"}
			if command == "promote" {
				args = append(args, "--confirm")
			}
			if err := app.Run(args); err != nil || calls != 1 {
				t.Fatalf("CLI approval failed: %v, calls %d", err, calls)
			}
		})
	}
}

func TestProcessRunUsesBackgroundProcessEndpoint(t *testing.T) {
	const sandboxID = "bd34c03c-e2bf-4f24-97fb-0f078570fb8a"
	calls := 0
	_ = cliapptest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/health" {
			_, _ = w.Write([]byte(`{"status":"healthy"}`))
			return
		}
		calls++
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/sandboxes/"+sandboxID+"/processes" {
			t.Errorf("process run request = %s %s, want POST /api/v1/sandboxes/%s/processes", r.Method, r.URL.Path, sandboxID)
			http.NotFound(w, r)
			return
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode process request: %v", err)
		}
		if request["command"] != "/bin/sleep" || request["name"] != "f3-lifecycle" {
			t.Errorf("process request lost command/name: %#v", request)
		}
		_, _ = w.Write([]byte(`{"pid":4321,"sandboxId":"` + sandboxID + `","command":"/bin/sleep","name":"f3-lifecycle"}`))
	}))
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", t.TempDir())
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"process", "run", sandboxID, "--name=f3-lifecycle", "--json", "--", "/bin/sleep", "60"}); err != nil {
		t.Fatalf("process run failed: %v", err)
	}
	if calls != 1 {
		t.Fatalf("request count = %d, want 1", calls)
	}
}

func TestProcessHelpShowsRunnableSyntaxAndOptions(t *testing.T) {
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", t.TempDir())
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{"exec", []string{"process", "exec", "--help"}, []string{"process exec <sandbox-id> [options] -- <command> [args...]", "--timeout=SECONDS", "Place -- before the command", "npm run build"}},
		{"run", []string{"process", "run", "--help"}, []string{"process run <sandbox-id> [options] -- <command> [args...]", "--name=NAME", "Place -- before the command"}},
		{"list", []string{"process", "list", "--help"}, []string{"process list <sandbox-id> [--running] [--json]", "--running"}},
		{"kill", []string{"process", "kill", "--help"}, []string{"process kill <sandbox-id> (--pid=<pid> | --all) [--json]", "Choose exactly one target"}},
		{"logs", []string{"process", "logs", "--help"}, []string{"process logs <sandbox-id>", "--stream=stdout|stderr", "--follow"}},
		{"shell", []string{"process", "shell", "--help"}, []string{"process shell <sandbox-id>", "--vrooli-aware", "--network"}},
		{"attach", []string{"process", "attach", "--help"}, []string{"process attach <sandbox-id>", "-- <command> [args...]", "bash"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			originalStdout := os.Stdout
			os.Stdout = writer
			runErr := app.Run(tt.args)
			os.Stdout = originalStdout
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			output, err := io.ReadAll(reader)
			if err != nil {
				t.Fatal(err)
			}
			if err := reader.Close(); err != nil {
				t.Fatal(err)
			}
			if runErr != nil {
				t.Fatalf("help failed: %v", runErr)
			}
			for _, want := range tt.want {
				if !strings.Contains(string(output), want) {
					t.Errorf("help missing %q:\n%s", want, output)
				}
			}
		})
	}
}

func TestProcessLogsForwardsRequiredStreamParameter(t *testing.T) {
	const sandboxID = "bd34c03c-e2bf-4f24-97fb-0f078570fb8a"
	const logPath = "/api/v1/sandboxes/" + sandboxID + "/processes/4321/logs"
	wantStreams := []string{"stdout", "stderr", "stderr"}
	calls := 0
	_ = cliapptest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/health" {
			_, _ = w.Write([]byte(`{"status":"healthy"}`))
			return
		}
		calls++
		if r.Method != http.MethodGet || r.URL.Path != logPath {
			t.Errorf("process logs request = %s %s, want GET %s", r.Method, r.URL.Path, logPath)
		}
		if calls > len(wantStreams) {
			t.Errorf("unexpected extra log request %d", calls)
			return
		}
		if got := r.URL.Query().Get("stream"); got != wantStreams[calls-1] {
			t.Errorf("stream query = %q, want %q", got, wantStreams[calls-1])
		}
		_, _ = w.Write([]byte(`{"pid":4321,"sandboxId":"` + sandboxID + `","content":"route-log-ok"}`))
	}))
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", t.TempDir())
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	commands := [][]string{
		{"process", "logs", sandboxID, "--pid=4321", "--json"},
		{"process", "logs", sandboxID, "--pid=4321", "--stream=stderr", "--json"},
		{"process", "logs", sandboxID, "--pid=4321", "--stream=stderr", "--follow"},
	}
	for _, command := range commands {
		if err := app.Run(command); err != nil {
			t.Fatalf("process logs %v failed: %v", command, err)
		}
	}
	if calls != len(wantStreams) {
		t.Fatalf("request count = %d, want %d", calls, len(wantStreams))
	}
}

func TestAppConstants(t *testing.T) {
	if appName != "workspace-sandbox" {
		t.Fatalf("appName = %q, want workspace-sandbox", appName)
	}
	if appVersion != "0.1.0" {
		t.Fatalf("appVersion = %q, want 0.1.0", appVersion)
	}
	if defaultAPIBase != "" {
		t.Fatalf("defaultAPIBase = %q, want empty", defaultAPIBase)
	}
}

func TestNewApp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Skipf("NewApp requires environment setup: %v", err)
	}
	if app == nil || app.core == nil {
		t.Fatalf("NewApp returned nil wiring")
	}
}

func TestWorkspaceSandboxCLIUsesDomainArchitecture(t *testing.T) {
	app := &App{}
	if groups := app.commandGroups(); len(groups) != 0 {
		t.Fatalf("commandGroups len = %d, want 0 (status served by cli-core)", len(groups))
	}

	subcommands := map[string]bool{}
	for _, group := range app.subcommandGroups() {
		subcommands[group.Name] = true
	}

	for _, expected := range []string{"sandbox", "process", "change", "maintenance", "provenance"} {
		if !subcommands[expected] {
			t.Fatalf("missing subcommand group %q", expected)
		}
	}
}
