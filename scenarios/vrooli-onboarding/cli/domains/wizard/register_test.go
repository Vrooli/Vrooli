package wizard

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	clitest "github.com/vrooli/cli-core/cliapptest"
	localtest "vrooli-onboarding/cli/internal/testutil"
)

func TestRegisterAndSelectionErrors(t *testing.T) {
	group := Register(&cliapp.ScenarioApp{})
	if group.Name != "wizard" || len(group.Subcommands) != 4 {
		t.Fatalf("unexpected wizard group: %+v", group)
	}
	if err := group.Subcommands[1].Run(nil); err == nil {
		t.Fatal("wizard apply without a selection should fail")
	}
	if err := group.Subcommands[2].Run(nil); err == nil {
		t.Fatal("wizard export without an output should fail")
	}
}

func TestDeclarativeWizardApplyExportAndStatus(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch || r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"ok":true,"status":"applied"}`))
			return
		}
		if r.URL.Path == "/api/v1/v2/scenarios" {
			_, _ = w.Write([]byte(`{"scenarios":[{"name":"demo","enabled":true}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"ready","credentials":[],"hosts":[]}`))
	}))
	selectionPath := filepath.Join(t.TempDir(), "selection.json")
	if err := localtest.WriteJSON(selectionPath, Selection{Scenarios: []string{"demo"}, Apply: true}); err != nil {
		t.Fatal(err)
	}
	group := Register(core)
	if err := group.Subcommands[0].Run([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	if err := group.Subcommands[1].Run([]string{"--selection", selectionPath, "--json"}); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "export.json")
	if err := group.Subcommands[2].Run([]string{"--output", output}); err != nil {
		t.Fatal(err)
	}
	if err := group.Subcommands[3].Run([]string{}); err != nil {
		t.Fatal(err)
	}
}

func TestInteractiveWizardWalksAllNineSteps(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/v2/scenarios":
			_, _ = w.Write([]byte(`{"scenarios":[{"name":"demo"}]}`))
		case "/api/v1/v2/resources":
			_, _ = w.Write([]byte(`{"optional":[{"name":"ollama"}],"standalone":[]}`))
		case "/api/v1/v2/host-requirements":
			_, _ = w.Write([]byte(`{"tools":[{"name":"git","required":true}],"safeguards":[{"name":"safe","required":false}]}`))
		default:
			_, _ = w.Write([]byte(`{"status":"ready","credentials":[],"hosts":[]}`))
		}
	}))
	input := "\ndemo\nollama\n\n\n\n\n demo\nyes\n\n"
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.WriteString(writer, input)
	_ = writer.Close()
	oldStdin := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() { os.Stdin = oldStdin; _ = reader.Close() })
	if err := runWizard(core, []string{"--interactive"}); err != nil {
		t.Fatal(err)
	}
}

func TestInteractiveWizardProvisionsCredentialsAndReviewsPlan(t *testing.T) {
	var provisioned bool
	var reviewedPlan bool
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/v2/scenarios":
			_, _ = w.Write([]byte(`{"scenarios":[{"name":"demo"}]}`))
		case "/api/v1/v2/resources":
			_, _ = w.Write([]byte(`{"optional":[],"standalone":[]}`))
		case "/api/v1/v2/credentials":
			_, _ = w.Write([]byte(`{"credentials":[{"logical_id":"demo","field":"api_key","label":"Demo key","required":true,"status":"missing"}]}`))
		case "/api/v1/v2/host-requirements":
			_, _ = w.Write([]byte(`{"tools":[{"name":"git","required":true,"risk":"low","privilege":"user"}],"safeguards":[{"name":"safe","required":false,"risk":"medium","privilege":"elevated"}]}`))
		case "/api/v1/v2/credentials/provision":
			provisioned = true
			_, _ = w.Write([]byte(`{"status":"provisioned"}`))
		case "/api/v1/v2/apply/plan":
			reviewedPlan = true
			_, _ = w.Write([]byte(`{"items":[{"kind":"tool","name":"git","required":true}]}`))
		case "/api/v1/v2/apply":
			_, _ = w.Write([]byte(`{"run_id":"run-1","status":"applied","items":[{"name":"git","outcome":"applied"}]}`))
		default:
			_, _ = w.Write([]byte(`{"status":"ready","credentials":[],"hosts":[]}`))
		}
	}))
	input := "\ndemo\n\n\nsecret\n\n\n\n\nyes\n\n"
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.WriteString(writer, input)
	_ = writer.Close()
	oldStdin := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() { os.Stdin = oldStdin; _ = reader.Close() })
	if err := runWizard(core, []string{"--interactive"}); err != nil {
		t.Fatal(err)
	}
	if !provisioned {
		t.Fatal("interactive wizard must provision missing credentials through the authority")
	}
	if !reviewedPlan {
		t.Fatal("interactive wizard must review the API-produced apply plan")
	}
}

func TestWizardPureFormattingHelpers(t *testing.T) {
	if got := hostNames([]string{"git", "ssh"}); got != "git, ssh" {
		t.Fatalf("hostNames = %q", got)
	}
	if got := ensureModeMap(nil); got == nil {
		t.Fatal("ensureModeMap must allocate a map")
	}
}

func TestSelectionPatchCoversAllAutomationChoices(t *testing.T) {
	selection := Selection{Scenarios: []string{"alpha"}, OptionalResources: []string{"ollama"}}
	selection.Resources = map[string]bool{"postgres": false}
	selection.Host.Tools = []string{"demo-tool"}
	selection.Host.Safeguards = []string{"kernel_config"}
	selection.OperatingMode = map[string]struct {
		AutoRestart bool `json:"auto_restart"`
	}{"alpha": {AutoRestart: true}}

	patch := selectionPatch(selection)
	data, err := json.Marshal(patch)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["scenarios"].(map[string]any)["alpha"].(map[string]any)["auto_restart"] != true {
		t.Fatal("operating mode must be represented in the shared patch")
	}
	if decoded["resources"].(map[string]any)["ollama"].(map[string]any)["enabled"] != true {
		t.Fatal("optional resources must be represented in the shared patch")
	}
	if decoded["host_tools"].(map[string]any)["demo-tool"].(map[string]any)["opted_in"] != true {
		t.Fatal("host tools must be represented in the shared patch")
	}
}
