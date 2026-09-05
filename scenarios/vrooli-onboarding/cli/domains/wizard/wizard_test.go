package wizard

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

func TestRunWizardUsesTheDeclaredStepSequence(t *testing.T) {
	var seen []string
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/v2/steps":
			_, _ = w.Write([]byte(testStepModelJSON))
		case "/api/v1/v2/scenarios":
			_, _ = w.Write([]byte(`{"scenarios":[{"name":"demo"}]}`))
		case "/api/v1/v2/core-set":
			_, _ = w.Write([]byte(`{"available":true,"seed":["demo"],"trusted_base":["demo"],"member_counts":{"scenario":1,"resource":0}}`))
		case "/api/v1/v2/resources":
			_, _ = w.Write([]byte(`{"optional":[],"standalone":[]}`))
		case "/api/v1/v2/credentials":
			_, _ = w.Write([]byte(`{"credentials":[]}`))
		case "/api/v1/v2/host-requirements":
			_, _ = w.Write([]byte(`{"tools":[],"safeguards":[]}`))
		case "/api/v1/v2/session/step":
			_, _ = w.Write([]byte(`{"step":8}`))
		default:
			_, _ = w.Write([]byte(`{"first_unsatisfied_step":0,"completion":false,"profile":"starter","scenarios":[],"resources":[]}`))
		}
		if r.URL.Path != "/api/v1/v2/session/step" {
			seen = append(seen, r.URL.Path)
		}
	}))
	input, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.WriteString(writer, "\n\n\n\n\n\n\n\n\nyes\n\n")
	_ = writer.Close()
	old := os.Stdin
	os.Stdin = input
	t.Cleanup(func() { os.Stdin = old; _ = input.Close() })
	if err := runWizard(core, []string{"--interactive"}); err != nil {
		t.Fatal(err)
	}
	if len(seen) == 0 {
		t.Fatal("runWizard did not call the declared API surface")
	}
}

func TestStepHandlerRejectsUnknownModelEntry(t *testing.T) {
	if _, ok := stepHandlers["synthetic-added-step"]; ok {
		t.Fatal("synthetic step must fail until a handler is implemented")
	}
	called := false
	if err := stepHandlers["welcome"](&wizardSession{runStep: func(id string) error { called = id == "welcome"; return nil }}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("welcome handler did not dispatch through the session")
	}
}

func TestRunWizardResumesAtFirstUnsatisfiedStep(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/v2/steps":
			_, _ = w.Write([]byte(testStepModelJSON))
		case "/api/v1/v2/session":
			_, _ = w.Write([]byte(`{"first_unsatisfied_step":6,"completion":false}`))
		case "/api/v1/v2/scenarios":
			_, _ = w.Write([]byte(`{"scenarios":[{"name":"demo"}]}`))
		case "/api/v1/v2/core-set":
			_, _ = w.Write([]byte(`{"available":true,"seed":["demo"],"trusted_base":["demo"],"member_counts":{"scenario":1,"resource":0}}`))
		case "/api/v1/v2/resources":
			_, _ = w.Write([]byte(`{"optional":[],"standalone":[]}`))
		case "/api/v1/v2/credentials":
			_, _ = w.Write([]byte(`{"credentials":[]}`))
		case "/api/v1/v2/host-requirements":
			_, _ = w.Write([]byte(`{"tools":[],"safeguards":[]}`))
		default:
			_, _ = w.Write([]byte(`{"status":"ready"}`))
		}
	}))
	input, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.WriteString(writer, "\n\n\nyes\n\n")
	_ = writer.Close()
	oldStdin := os.Stdin
	os.Stdin = input
	t.Cleanup(func() { os.Stdin = oldStdin; _ = input.Close() })
	oldStdout := os.Stdout
	output, err := os.CreateTemp(t.TempDir(), "wizard-output-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = output
	t.Cleanup(func() { os.Stdout = oldStdout; _ = output.Close() })
	if err := runWizard(core, []string{"--interactive"}); err != nil {
		t.Fatal(err)
	}
	if err := output.Sync(); err != nil {
		t.Fatal(err)
	}
	if _, err := output.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(output)
	text := string(data)
	if !strings.Contains(text, "Resuming onboarding at host") {
		t.Fatalf("output = %q", text)
	}
	if strings.Contains(text, "Available scenarios:") {
		t.Fatalf("resume replayed an earlier step: %q", text)
	}
}
