package wizard

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	localtest "vrooli-onboarding/cli/internal/testutil"

	"github.com/vrooli/cli-core/cliapp"
	clitest "github.com/vrooli/cli-core/cliapptest"
)

func TestRegisterAndSelectionErrors(t *testing.T) {
	group := Register(&cliapp.ScenarioApp{})
	if group.Name != "wizard" || len(group.Subcommands) != 4 {
		t.Fatalf("unexpected wizard group: %+v", group)
	}
	if err := group.Subcommands[1].Run(nil); err == nil {
		t.Fatal("wizard commit without a selection should fail")
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
		case "/api/v1/v2/steps":
			_, _ = w.Write([]byte(testStepModelJSON))
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
		case "/api/v1/v2/steps":
			_, _ = w.Write([]byte(testStepModelJSON))
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

const testStepModelJSON = `{"steps":[{"id":"welcome","ordinal":0},{"id":"scenarios","ordinal":1},{"id":"resources","ordinal":2},{"id":"credentials","ordinal":3},{"id":"integrations","ordinal":4},{"id":"host","ordinal":5},{"id":"operating-mode","ordinal":6},{"id":"apply","ordinal":7},{"id":"validation","ordinal":8}]}`

func TestWizardPureFormattingHelpers(t *testing.T) {
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

// captureStdout redirects os.Stdout to a temp file for the duration of fn and
// returns what was written. A file is used rather than a pipe so a large apply
// plan cannot deadlock on a full pipe buffer.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	path := t.TempDir() + "/stdout"
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create capture file: %v", err)
	}
	old := os.Stdout
	os.Stdout = file
	defer func() {
		os.Stdout = old
		_ = file.Close()
	}()
	fn()
	_ = file.Sync()
	captured, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read capture file: %v", err)
	}
	return string(captured)
}

// TestApplyPlanIsDisclosedBeforeConsent pins the ordering of disclosure and
// consent. The wizard used to ask "apply this selection now?" first and print
// the plan only afterwards, so the operator answered with nothing disclosed.
// Answering no is the proof: under the old order the plan was never fetched at
// all on a declined run, so a declined run that still shows the plan can only
// mean the disclosure now precedes the question.
func TestApplyPlanIsDisclosedBeforeConsent(t *testing.T) {
	var planRequested, applyRequested bool
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/v2/steps":
			_, _ = w.Write([]byte(testStepModelJSON))
		case "/api/v1/v2/scenarios":
			_, _ = w.Write([]byte(`{"scenarios":[{"name":"demo"}]}`))
		case "/api/v1/v2/resources":
			_, _ = w.Write([]byte(`{"optional":[],"standalone":[]}`))
		case "/api/v1/v2/credentials":
			_, _ = w.Write([]byte(`{"credentials":[]}`))
		case "/api/v1/v2/host-requirements":
			_, _ = w.Write([]byte(`{"tools":[],"safeguards":[]}`))
		case "/api/v1/v2/apply/plan":
			planRequested = true
			_, _ = w.Write([]byte(`{"items":[{"kind":"safeguard","name":"host_hardening","required":true,"privileged":true}]}`))
		case "/api/v1/v2/apply":
			applyRequested = true
			_, _ = w.Write([]byte(`{"run_id":"run-1","status":"applied"}`))
		default:
			_, _ = w.Write([]byte(`{"status":"ready","credentials":[],"hosts":[]}`))
		}
	}))

	// One answer per prompt, in order: welcome, scenarios, resources,
	// credentials, integrations, host tools, host safeguards, operating-mode,
	// and finally the apply confirmation, which is the "no".
	input := "\ndemo\n\n\n\n\n\n\nno\n"
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.WriteString(writer, input)
	_ = writer.Close()
	oldStdin := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() { os.Stdin = oldStdin; _ = reader.Close() })

	var runErr error
	output := captureStdout(t, func() { runErr = runWizard(core, []string{"--interactive"}) })

	if runErr == nil {
		t.Fatal("answering no must not apply the selection")
	}
	// Proves the "no" was consumed by the apply confirmation rather than by an
	// earlier prompt, which would make the rest of this test vacuous.
	if !strings.Contains(runErr.Error(), "selection not applied") {
		t.Fatalf("the decline must come from the apply step, got %v", runErr)
	}
	if !planRequested {
		t.Fatal("the apply plan must be fetched before consent, so a declined run still discloses it")
	}
	if applyRequested {
		t.Fatal("a declined run must not POST /v2/apply")
	}
	if !strings.Contains(output, "host_hardening") {
		t.Fatalf("the declined run did not disclose the plan contents: %q", output)
	}
	if !strings.Contains(output, "Nothing has been applied yet") {
		t.Fatalf("the disclosure must state that nothing has happened yet: %q", output)
	}
}

// TestApplyPlanSeparatesPendingFromAlreadyApplied pins the state split. The
// plan is a desired-state list, so before this an operator was shown 53 items
// that read as 53 pending changes when most were already in place. It also
// guards the elevation callout: Privileged was on the wire and the renderer
// discarded it, so elevated host mutations could be approved without ever
// being shown as elevated.
func TestApplyPlanSeparatesPendingFromAlreadyApplied(t *testing.T) {
	body := []byte(`{"items":[
		{"kind":"tool","name":"git","required":true,"state":"satisfied"},
		{"kind":"tool","name":"jq","required":false,"state":"satisfied"},
		{"kind":"safeguard","name":"host_hardening","required":true,"privileged":true,"state":"pending"},
		{"kind":"safeguard","name":"clock","required":true,"state":"pending"},
		{"kind":"resource","name":"postgres","required":true,"state":"unknown"}
	]}`)
	output := captureStdout(t, func() {
		if err := printApplyPlan(body); err != nil {
			t.Fatalf("printApplyPlan: %v", err)
		}
	})

	for _, want := range []string{
		"5 selected item(s)",
		"2 not yet in place",
		"2 already in place",
		"1 not checkable",
		"NOT YET IN PLACE",
		"ALREADY IN PLACE",
		"NOT CHECKED",
		"! host_hardening (required, elevated)",
		"- clock (required)",
		"Nothing is removed, disabled, or uninstalled by apply",
		"vrooli host safeguard <name>",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("apply plan disclosure missing %q:\n%s", want, output)
		}
	}

	// The already-satisfied items must not appear in the section that claims
	// the host will change, which is the whole point of the split.
	pendingSection := output[strings.Index(output, "NOT YET IN PLACE"):strings.Index(output, "ALREADY IN PLACE")]
	for _, satisfied := range []string{"git", "jq"} {
		if strings.Contains(pendingSection, satisfied) {
			t.Fatalf("already-satisfied item %q listed as a pending change:\n%s", satisfied, pendingSection)
		}
	}
}

// TestApplyPlanTreatsMissingStateAsUnknown keeps the renderer honest when the
// API is older than the CLI: an absent state must never be presented as a
// verified "already in place".
func TestApplyPlanTreatsMissingStateAsUnknown(t *testing.T) {
	output := captureStdout(t, func() {
		if err := printApplyPlan([]byte(`{"items":[{"kind":"tool","name":"git","required":true}]}`)); err != nil {
			t.Fatalf("printApplyPlan: %v", err)
		}
	})
	if !strings.Contains(output, "NOT CHECKED") {
		t.Fatalf("an item with no state must be reported as unchecked:\n%s", output)
	}
	if strings.Contains(output, "ALREADY IN PLACE") {
		t.Fatalf("an item with no state must never be claimed as already in place:\n%s", output)
	}
}

// TestApplyPlanWithNoChangesSaysSo keeps the empty case honest rather than
// printing an empty heading over a bare prompt.
func TestApplyPlanWithNoChangesSaysSo(t *testing.T) {
	output := captureStdout(t, func() {
		if err := printApplyPlan([]byte(`{"items":[]}`)); err != nil {
			t.Fatalf("printApplyPlan: %v", err)
		}
	})
	if !strings.Contains(output, "no changes") {
		t.Fatalf("empty plan must say so plainly, got %q", output)
	}
}
