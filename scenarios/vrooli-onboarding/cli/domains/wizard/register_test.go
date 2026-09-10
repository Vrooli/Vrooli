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
	if group.Name != "wizard" || len(group.Subcommands) != 7 {
		t.Fatalf("unexpected wizard group: %+v", group)
	}
	if err := group.Subcommands[1].Run(nil); err == nil {
		t.Fatal("wizard commit without a selection should fail")
	}
	if err := group.Subcommands[2].Run(nil); err == nil {
		t.Fatal("wizard export without an output should fail")
	}
}

func TestReportReadinessBlockersRetainsSafeMetadataInError(t *testing.T) {
	err := reportReadinessBlockers([]byte(`{"status":"missing","blockers":[{"kind":"host","name":"workspace_sandbox_userns","reason":"safeguard is missing on this host","remediation":"Apply the selection."}],"degraded":[]}`))
	if err == nil {
		t.Fatal("expected readiness blocker error")
	}
	for _, want := range []string{"workspace_sandbox_userns", "safeguard is missing on this host", "Apply the selection."} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want %q", err.Error(), want)
		}
	}
}

func TestDeclarativeWizardApplyExportAndStatus(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/vrooli.vrooli_onboarding.v1.apply.ApplyService/GetApplyPlan" {
			_, _ = w.Write([]byte(`{"planId":"plan-1","planDigest":"digest-1","revision":"rev-1","items":[]}`))
			return
		}
		if r.URL.Path == "/vrooli.vrooli_onboarding.v1.apply.ApplyService/ReviewApply" {
			_, _ = w.Write([]byte(`{"planId":"plan-1","planDigest":"digest-1","revision":"rev-1","consentReceiptId":"receipt-1"}`))
			return
		}
		if r.URL.Path == "/vrooli.vrooli_onboarding.v1.apply.ApplyService/StartApply" {
			_, _ = w.Write([]byte(`{"run":{"runId":"run-1","status":"APPLY_RUN_STATE_APPLIED","legacyStatus":"applied","steps":[]}}`))
			return
		}
		if r.URL.Path == "/vrooli.vrooli_onboarding.v1.session.SessionService/GetSession" {
			_, _ = w.Write([]byte(`{"step":0,"stepId":"welcome","firstUnsatisfiedStep":0,"completion":false}`))
			return
		}
		if r.URL.Path == "/vrooli.vrooli_onboarding.v1.selection.SelectionService/AcceptRecommendation" {
			_, _ = w.Write([]byte(`{"selection":{"schemaVersion":"v1","apply":true},"firstUnsatisfiedStep":0}`))
			return
		}
		if r.URL.Path == "/vrooli.vrooli_onboarding.v1.selection.SelectionService/ListScenarios" {
			_, _ = w.Write([]byte(`{"scenarios":[{"name":"demo","enabled":true}]}`))
			return
		}
		if r.URL.Path == "/vrooli.vrooli_onboarding.v1.operatorstate.OperatorStateService/GetOperatorState" {
			_, _ = w.Write([]byte(`{"state":{"activeProfile":"engineering","core":{"seed":["demo"]},"scenarios":{"demo":{"enabled":true,"autoRestart":true},"disabled":{"enabled":false,"autoRestart":false}},"resources":{"ollama":{"enabled":false}},"hostTools":{"git":{"optedIn":true}},"hostSafeguards":{"sandbox":{"optedIn":false}}}}`))
			return
		}
		if r.URL.Path == "/vrooli.vrooli_onboarding.v1.operatorstate.OperatorStateService/PatchOperatorState" {
			_, _ = w.Write([]byte(`{"state":{}}`))
			return
		}
		if r.URL.Path == "/vrooli.vrooli_onboarding.v1.readiness.ReadinessService/GetReadiness" {
			_, _ = w.Write([]byte(`{"configurationRevision":"test","status":"READINESS_STATE_READY","credentials":[],"hosts":[],"blockers":[],"degraded":[]}`))
			return
		}
		if r.Method == http.MethodPatch || r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"ok":true,"status":"applied"}`))
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
	contents, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var exported Selection
	if err := json.Unmarshal(contents, &exported); err != nil {
		t.Fatal(err)
	}
	if exported.ActiveProfile != "engineering" || len(exported.CoreSeed) != 1 || exported.CoreSeed[0] != "demo" {
		t.Fatalf("export lost profile or core seed: %+v", exported)
	}
	if exported.ScenarioState["disabled"] || exported.ScenarioState["demo"] != true {
		t.Fatalf("export lost explicit scenario choices: %+v", exported.ScenarioState)
	}
	if exported.OperatingMode["demo"].AutoRestart != true || exported.OperatingMode["disabled"].AutoRestart != false {
		t.Fatalf("export lost operating mode: %+v", exported.OperatingMode)
	}
	if exported.Resources["ollama"] || exported.HostTools["git"] != true || exported.HostSafeguards["sandbox"] {
		t.Fatalf("export lost explicit false-valued choices: resources=%+v tools=%+v safeguards=%+v", exported.Resources, exported.HostTools, exported.HostSafeguards)
	}
	if err := group.Subcommands[4].Run([]string{}); err != nil {
		t.Fatal(err)
	}
	supportPath := filepath.Join(t.TempDir(), "support.json")
	if err := os.WriteFile(supportPath, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := group.Subcommands[3].Run([]string{"--output", supportPath, "--include", "selection,readiness,session"}); err != nil {
		t.Fatal(err)
	}
	supportContents, err := os.ReadFile(supportPath)
	if err != nil {
		t.Fatal(err)
	}
	fileInfo, err := os.Stat(supportPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("support export permissions = %o, want 600", got)
	}
	if strings.Contains(string(supportContents), "SECRET-CANARY") || strings.Contains(string(supportContents), "credential_diagnosis") {
		t.Fatalf("support export leaked excluded diagnostic data: %s", supportContents)
	}
	var supportDocument map[string]json.RawMessage
	if err := json.Unmarshal(supportContents, &supportDocument); err != nil {
		t.Fatal(err)
	}
	for _, section := range []string{"selection", "readiness", "session"} {
		if _, ok := supportDocument[section]; !ok {
			t.Fatalf("support export omitted requested section %q: %s", section, supportContents)
		}
	}
}

func TestSupportExportRequiresKnownExplicitSections(t *testing.T) {
	if _, err := supportExportSections(""); err == nil {
		t.Fatal("support export must require explicit inclusion choices")
	}
	if _, err := supportExportSections("selection,unknown"); err == nil {
		t.Fatal("unsupported support-export section was accepted")
	}
	sections, err := supportExportSections("selection,readiness,selection")
	if err != nil || strings.Join(sections, ",") != "selection,readiness" {
		t.Fatalf("sections=%v err=%v, want deduplicated explicit sections", sections, err)
	}
}

func TestWritePrivateExportRefusesSymlinkTargets(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.json")
	link := filepath.Join(root, "export.json")
	if err := os.WriteFile(target, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable on this platform: %v", err)
	}

	if err := writePrivateExport(link, []byte("replacement")); err == nil {
		t.Fatal("private export followed a symlink target")
	}
	contents, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "original" {
		t.Fatalf("symlink target changed to %q", contents)
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("refused symlink was replaced")
	}

	external := filepath.Join(root, "external")
	if err := os.Mkdir(external, 0o700); err != nil {
		t.Fatal(err)
	}
	linkedDirectory := filepath.Join(root, "linked-directory")
	if err := os.Symlink(external, linkedDirectory); err != nil {
		t.Skipf("symlinked directories unavailable on this platform: %v", err)
	}
	if err := writePrivateExport(filepath.Join(linkedDirectory, "export.json"), []byte("replacement")); err == nil {
		t.Fatal("private export accepted a symlinked parent directory")
	}
}

func TestInteractiveWizardWalksAllTenSteps(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/vrooli.vrooli_onboarding.v1.apply.ApplyService/GetApplyPlan":
			_, _ = w.Write([]byte(`{"planId":"plan-1","planDigest":"digest-1","revision":"rev-1","items":[]}`))
		case "/vrooli.vrooli_onboarding.v1.apply.ApplyService/ReviewApply":
			_, _ = w.Write([]byte(`{"planId":"plan-1","planDigest":"digest-1","revision":"rev-1","consentReceiptId":"receipt-1"}`))
		case "/vrooli.vrooli_onboarding.v1.apply.ApplyService/StartApply":
			_, _ = w.Write([]byte(`{"run":{"runId":"run-1","status":"APPLY_RUN_STATE_APPLIED","legacyStatus":"applied"}}`))
		case "/vrooli.vrooli_onboarding.v1.session.SessionService/GetStepModel":
			_, _ = w.Write([]byte(testStepModelJSON))
		case "/vrooli.vrooli_onboarding.v1.session.SessionService/GetSession":
			_, _ = w.Write([]byte(`{"step":0,"stepId":"welcome","firstUnsatisfiedStep":0,"completion":false}`))
		case "/vrooli.vrooli_onboarding.v1.session.SessionService/AdvanceSessionStep":
			_, _ = w.Write([]byte(`{"step":0,"stepId":"welcome","firstUnsatisfiedStep":0,"completion":false}`))
		case "/vrooli.vrooli_onboarding.v1.selection.SelectionService/ListScenarios":
			_, _ = w.Write([]byte(`{"scenarios":[{"name":"demo"}]}`))
		case "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetRecommendation":
			_, _ = w.Write([]byte(`{"profile":"starter","scenarios":[],"resources":[],"explanation":"starter"}`))
		case "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetCoreSet":
			_, _ = w.Write([]byte(`{"available":true,"seed":["demo"],"trustedBase":["demo"],"memberCounts":{"scenario":1,"resource":0}}`))
		case "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetUnion":
			_, _ = w.Write([]byte(`{"optionalResources":[{"name":"ollama"}],"standaloneResources":[]}`))
		case "/vrooli.vrooli_onboarding.v1.credentials.CredentialsService/ListCredentials":
			_, _ = w.Write([]byte(`{"credentials":[]}`))
		case "/vrooli.vrooli_onboarding.v1.operatorinputs.OperatorInputsService/ListOperatorInputs":
			_, _ = w.Write([]byte(`{"requests":[]}`))
		case "/vrooli.vrooli_onboarding.v1.readiness.ReadinessService/GetReadiness":
			_, _ = w.Write([]byte(`{"configurationRevision":"test"}`))
		case "/vrooli.vrooli_onboarding.v1.operatorstate.OperatorStateService/PatchOperatorState":
			_, _ = w.Write([]byte(`{"state":{}}`))
		case "/api/v1/v2/host-requirements":
			_, _ = w.Write([]byte(`{"tools":[{"name":"git","required":true}],"safeguards":[{"name":"safe","required":false}]}`))
		default:
			_, _ = w.Write([]byte(`{"status":"ready","credentials":[],"hosts":[]}`))
		}
	}))
	input := "\ndemo\n\nollama\n\n\n\n\n demo\nyes\n\n"
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
		case "/vrooli.vrooli_onboarding.v1.session.SessionService/GetStepModel":
			_, _ = w.Write([]byte(testStepModelJSON))
		case "/vrooli.vrooli_onboarding.v1.session.SessionService/GetSession":
			_, _ = w.Write([]byte(`{"step":0,"stepId":"welcome","firstUnsatisfiedStep":0,"completion":false}`))
		case "/vrooli.vrooli_onboarding.v1.session.SessionService/AdvanceSessionStep":
			_, _ = w.Write([]byte(`{"step":0,"stepId":"welcome","firstUnsatisfiedStep":0,"completion":false}`))
		case "/vrooli.vrooli_onboarding.v1.selection.SelectionService/ListScenarios":
			_, _ = w.Write([]byte(`{"scenarios":[{"name":"demo"}]}`))
		case "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetRecommendation":
			_, _ = w.Write([]byte(`{"profile":"starter","scenarios":[],"resources":[],"explanation":"starter"}`))
		case "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetCoreSet":
			_, _ = w.Write([]byte(`{"available":true,"seed":["demo"],"trustedBase":["demo"],"memberCounts":{"scenario":1,"resource":0}}`))
		case "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetUnion":
			_, _ = w.Write([]byte(`{"optionalResources":[],"standaloneResources":[]}`))
		case "/vrooli.vrooli_onboarding.v1.credentials.CredentialsService/ListCredentials":
			_, _ = w.Write([]byte(`{"credentials":[{"logical_id":"demo","field":"api_key","label":"Demo key","required":true,"status":"missing"}]}`))
		case "/vrooli.vrooli_onboarding.v1.operatorinputs.OperatorInputsService/ListOperatorInputs":
			_, _ = w.Write([]byte(`{"requests":[]}`))
		case "/vrooli.vrooli_onboarding.v1.readiness.ReadinessService/GetReadiness":
			_, _ = w.Write([]byte(`{"configurationRevision":"test"}`))
		case "/vrooli.vrooli_onboarding.v1.operatorstate.OperatorStateService/PatchOperatorState":
			_, _ = w.Write([]byte(`{"state":{}}`))
		case "/api/v1/v2/host-requirements":
			_, _ = w.Write([]byte(`{"tools":[{"name":"git","required":true,"risk":"low","privilege":"user"}],"safeguards":[{"name":"safe","required":false,"risk":"medium","privilege":"elevated"}]}`))
		case "/vrooli.vrooli_onboarding.v1.credentials.CredentialsService/ProvisionCredential":
			provisioned = true
			_, _ = w.Write([]byte(`{"status":"provisioned"}`))
		case "/vrooli.vrooli_onboarding.v1.apply.ApplyService/GetApplyPlan":
			reviewedPlan = true
			_, _ = w.Write([]byte(`{"planId":"plan-1","planDigest":"digest-1","revision":"rev-1","items":[{"kind":"tool","name":"git","required":true,"observedState":"pending"}]}`))
		case "/vrooli.vrooli_onboarding.v1.apply.ApplyService/ReviewApply":
			_, _ = w.Write([]byte(`{"planId":"plan-1","planDigest":"digest-1","revision":"rev-1","consentReceiptId":"receipt-1"}`))
		case "/vrooli.vrooli_onboarding.v1.apply.ApplyService/StartApply":
			_, _ = w.Write([]byte(`{"run":{"runId":"run-1","status":"APPLY_RUN_STATE_APPLIED","legacyStatus":"applied","steps":[{"name":"git","state":"APPLY_STEP_STATE_APPLIED","legacyOutcome":"applied"}]}}`))
		default:
			_, _ = w.Write([]byte(`{"status":"ready","credentials":[],"hosts":[]}`))
		}
	}))
	input := "\ndemo\n\n\n\nsecret\n\n\n\n\nyes\n\n"
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

const testStepModelJSON = `{"steps":[{"id":"welcome","ordinal":0},{"id":"scenarios","ordinal":1},{"id":"core-set","ordinal":2},{"id":"resources","ordinal":3},{"id":"credentials","ordinal":4},{"id":"integrations","ordinal":5},{"id":"host","ordinal":6},{"id":"operating-mode","ordinal":7},{"id":"apply","ordinal":8},{"id":"validation","ordinal":9}]}`

func TestWizardPureFormattingHelpers(t *testing.T) {
	if got := ensureModeMap(nil); got == nil {
		t.Fatal("ensureModeMap must allocate a map")
	}
}

func TestSelectionPatchCoversAllAutomationChoices(t *testing.T) {
	selection := Selection{Scenarios: []string{"alpha"}, OptionalResources: []string{"ollama"}}
	selection.CoreSeed = []string{"alpha"}
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
	if decoded["core"].(map[string]any)["seed"].([]any)[0] != "alpha" {
		t.Fatal("core seed must be represented in the shared patch")
	}
}

func TestCoreSetCommandPreviewsBeforeFieldScopedPatch(t *testing.T) {
	order := []string{}
	var patched []string
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetCoreSet" && len(order) == 0:
			order = append(order, "current")
			_, _ = w.Write([]byte(`{"available":true,"seed":["alpha"],"trustedBase":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetCoreSet":
			order = append(order, "preview")
			_, _ = w.Write([]byte(`{"available":true,"seed":["beta"],"trustedBase":[],"memberCounts":{"scenario":1,"resource":0}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/vrooli.vrooli_onboarding.v1.operatorstate.OperatorStateService/PatchOperatorState":
			order = append(order, "patch")
			var body struct {
				State struct {
					Core struct {
						Seed []string `json:"seed"`
					} `json:"core"`
				} `json:"state"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			patched = body.State.Core.Seed
			_, _ = w.Write([]byte(`{"state":{"version":"1.0.0"}}`))
		default:
			http.NotFound(w, r)
		}
	}))

	if err := runCoreSet(core, []string{"--add", "beta", "--remove", "alpha", "--json"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(order, ","); got != "current,preview,patch" {
		t.Fatalf("request order = %s", got)
	}
	if got := strings.Join(patched, ","); got != "beta" {
		t.Fatalf("patched seed = %s", got)
	}
}

func TestCoreSetCommandDoesNotWriteWithoutClosure(t *testing.T) {
	patched := false
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/vrooli.vrooli_onboarding.v1.operatorstate.OperatorStateService/PatchOperatorState" {
			patched = true
		}
		if r.URL.Path == "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetCoreSet" {
			_, _ = w.Write([]byte(`{"available":false,"seed":["alpha","beta"],"error":"catalog unavailable"}`))
			return
		}
		_, _ = w.Write([]byte(`{"available":true,"seed":["alpha"]}`))
	}))

	err := runCoreSet(core, []string{"--add", "beta"})
	if err == nil || !strings.Contains(err.Error(), "closure unavailable") {
		t.Fatalf("error = %v", err)
	}
	if patched {
		t.Fatal("core-set command wrote without a computed closure")
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
		case "/vrooli.vrooli_onboarding.v1.session.SessionService/GetStepModel":
			_, _ = w.Write([]byte(testStepModelJSON))
		case "/vrooli.vrooli_onboarding.v1.session.SessionService/GetSession":
			_, _ = w.Write([]byte(`{"step":0,"stepId":"welcome","firstUnsatisfiedStep":0,"completion":false}`))
		case "/vrooli.vrooli_onboarding.v1.session.SessionService/AdvanceSessionStep":
			_, _ = w.Write([]byte(`{"step":0,"stepId":"welcome","firstUnsatisfiedStep":0,"completion":false}`))
		case "/vrooli.vrooli_onboarding.v1.selection.SelectionService/ListScenarios":
			_, _ = w.Write([]byte(`{"scenarios":[{"name":"demo"}]}`))
		case "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetRecommendation":
			_, _ = w.Write([]byte(`{"profile":"starter","scenarios":[],"resources":[],"explanation":"starter"}`))
		case "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetCoreSet":
			_, _ = w.Write([]byte(`{"available":true,"seed":["demo"],"trustedBase":["demo"],"memberCounts":{"scenario":1,"resource":0}}`))
		case "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetUnion":
			_, _ = w.Write([]byte(`{"optionalResources":[],"standaloneResources":[]}`))
		case "/vrooli.vrooli_onboarding.v1.credentials.CredentialsService/ListCredentials":
			_, _ = w.Write([]byte(`{"credentials":[]}`))
		case "/vrooli.vrooli_onboarding.v1.operatorinputs.OperatorInputsService/ListOperatorInputs":
			_, _ = w.Write([]byte(`{"requests":[]}`))
		case "/vrooli.vrooli_onboarding.v1.readiness.ReadinessService/GetReadiness":
			_, _ = w.Write([]byte(`{"configurationRevision":"test"}`))
		case "/vrooli.vrooli_onboarding.v1.operatorstate.OperatorStateService/PatchOperatorState":
			_, _ = w.Write([]byte(`{"state":{}}`))
		case "/api/v1/v2/host-requirements":
			_, _ = w.Write([]byte(`{"tools":[],"safeguards":[]}`))
		case "/vrooli.vrooli_onboarding.v1.apply.ApplyService/GetApplyPlan":
			planRequested = true
			_, _ = w.Write([]byte(`{"planId":"plan-1","planDigest":"digest-1","revision":"rev-1","items":[{"kind":"safeguard","name":"host_hardening","required":true,"privileged":true,"observedState":"pending"}]}`))
		case "/vrooli.vrooli_onboarding.v1.apply.ApplyService/ReviewApply":
			_, _ = w.Write([]byte(`{"planId":"plan-1","planDigest":"digest-1","revision":"rev-1","consentReceiptId":"receipt-1"}`))
		case "/vrooli.vrooli_onboarding.v1.apply.ApplyService/StartApply":
			applyRequested = true
			_, _ = w.Write([]byte(`{"run":{"runId":"run-1","status":"APPLY_RUN_STATE_APPLIED","legacyStatus":"applied"}}`))
		default:
			_, _ = w.Write([]byte(`{"status":"ready","credentials":[],"hosts":[]}`))
		}
	}))

	// One answer per prompt, in order: welcome, scenarios, core set, resources,
	// credentials, integrations, host tools, host safeguards, operating-mode,
	// and finally the apply confirmation, which is the "no".
	input := "\ndemo\n\n\n\n\n\n\n\nno\n"
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
		t.Fatal("a declined run must not start an apply procedure")
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
		{"kind":"tool","name":"git","required":true,"observedState":"satisfied"},
		{"kind":"tool","name":"jq","required":false,"observedState":"satisfied"},
		{"kind":"safeguard","name":"host_hardening","required":true,"privileged":true,"observedState":"pending"},
		{"kind":"safeguard","name":"clock","required":true,"observedState":"pending"},
		{"kind":"resource","name":"postgres","required":true,"observedState":"unknown"}
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
		"1 not sampled",
		"NOT YET IN PLACE",
		"ALREADY IN PLACE",
		"NOT SAMPLED",
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
	if !strings.Contains(output, "NOT SAMPLED") {
		t.Fatalf("an item with no state must be reported as unsampled:\n%s", output)
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

// TestApplyReportShowsWhyNothingHappened covers the refused-run path. A run can
// end without applying anything, and printing only the status would name the
// failure without saying what to do about it.
func TestApplyReportShowsWhyNothingHappened(t *testing.T) {
	body := []byte(`{"status":"APPLY_RUN_STATE_FAILED","legacyStatus":"blocked","steps":[],"blockers":[{"name":"vrooli-onboarding","reason":"artifacts are stale and a dependency would restart it","remediation":"Run ` + "`vrooli scenario start vrooli-onboarding`" + `, then apply again."}]}`)
	output := captureStdout(t, func() {
		if err := printApplyReport(body); err != nil {
			t.Fatalf("printApplyReport: %v", err)
		}
	})
	for _, want := range []string{
		"blocked",
		"vrooli-onboarding",
		"artifacts are stale",
		"fix:",
		"vrooli scenario start",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("apply report missing %q:\n%s", want, output)
		}
	}
}

// TestApplyReportDoesNotCallASkippedItemCompleted keeps a skipped item from
// rendering as if it had run. The detail fell straight through to "completed"
// whenever an item carried a reason but no error.
func TestApplyReportDoesNotCallASkippedItemCompleted(t *testing.T) {
	body := []byte(`{"status":"APPLY_RUN_STATE_APPLIED","legacyStatus":"applied","steps":[{"name":"vrooli-onboarding","state":"APPLY_STEP_STATE_SKIPPED_SELF","legacyOutcome":"skipped_self","remediation":"already running; onboarding does not restart itself mid-apply"}]}`)
	output := captureStdout(t, func() {
		if err := printApplyReport(body); err != nil {
			t.Fatalf("printApplyReport: %v", err)
		}
	})
	if strings.Contains(output, "completed") {
		t.Fatalf("a skipped item was reported as completed:\n%s", output)
	}
	if !strings.Contains(output, "already running") {
		t.Fatalf("a skipped item must carry its reason:\n%s", output)
	}
}
