package scenariocli

import (
	"bytes"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/process"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
)

func TestRenderStatusResponseHumanSingleIncludesScenarioHeader(t *testing.T) {
	startedAt := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	resp := StatusResponse{
		Single: &StatusSingleOutput{
			Scenario: StatusItemOutput{
				Name:      "alpha",
				Status:    "running",
				Health:    "healthy",
				Processes: 1,
				Runtime:   "2m",
				Ports:     map[string]int{"API_PORT": 18080},
			},
			Info: InfoScenarioData{
				Name:        "alpha",
				DisplayName: "Alpha",
				Description: "Alpha scenario",
				Path:        "/repo/scenarios/alpha",
			},
			Runtime: InfoRuntimeData{
				Status:    "running",
				Runtime:   "2m",
				StartedAt: &startedAt,
				Ports:     map[string]int{"API_PORT": 18080},
				ProcessInfo: []process.Record{
					{Step: "start-api", PID: 1234, Port: 18080, StartedAt: startedAt},
				},
			},
		},
	}

	var stdout bytes.Buffer
	if err := RenderStatusResponse(&stdout, cliout.FormatHuman, resp); err != nil {
		t.Fatalf("RenderStatusResponse: %v", err)
	}
	output := stdout.String()
	for _, want := range []string{"Scenario: alpha", "Status: running", "Health: healthy", "Processes:", "start-api pid=1234"} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in output:\n%s", want, output)
		}
	}
}

func TestWriteLifecycleItemsJSONIncludesSuccessEnvelope(t *testing.T) {
	var stdout bytes.Buffer
	err := WriteLifecycleItems(&stdout, cliout.FormatJSON, []LifecycleItemOutput{{
		Name:      "alpha",
		Status:    "started",
		Ports:     map[string]int{"API_PORT": 18080},
		Endpoints: []EndpointOutput{{Key: "API_PORT", URL: "http://localhost:18080", Port: 18080}},
	}})
	if err != nil {
		t.Fatalf("WriteLifecycleItems: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, `"success": true`) || !strings.Contains(output, `"scenarios":`) {
		t.Fatalf("output = %s", output)
	}
}

func TestRenderTemplateValidateResponseJSONReflectsIssues(t *testing.T) {
	var stdout bytes.Buffer
	err := RenderTemplateValidateResponse(&stdout, cliout.FormatJSON, TemplateValidationReport{
		Count: 1,
		Issues: []TemplateValidationIssue{{
			Template: "react-vite",
			Message:  "test-genie deep validation failed",
		}},
	})
	if err != nil {
		t.Fatalf("RenderTemplateValidateResponse: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, `"success": false`) || !strings.Contains(output, `"issues":`) {
		t.Fatalf("output = %s", output)
	}
}

func TestParseTemplateCleanupRequest(t *testing.T) {
	req, err := ParseTemplateCleanupRequest([]string{"--dry-run", "--older-than", "24h", "--include-retained", "--run", "run-1"})
	if err != nil {
		t.Fatalf("ParseTemplateCleanupRequest: %v", err)
	}
	if !req.DryRun || req.OlderThan != "24h" || !req.IncludeRetained || req.RunID != "run-1" {
		t.Fatalf("req = %#v", req)
	}
}

func TestWriteLifecycleItemsHumanIncludesURLs(t *testing.T) {
	var stdout bytes.Buffer
	err := WriteLifecycleItems(&stdout, cliout.FormatHuman, []LifecycleItemOutput{{
		Name:            "alpha",
		Status:          "started",
		Health:          "healthy",
		Ports:           map[string]int{"API_PORT": 18080},
		Endpoints:       []EndpointOutput{{Key: "API_PORT", URL: "http://localhost:18080", Port: 18080}},
		FailedResources: []string{"qdrant"},
	}})
	if err != nil {
		t.Fatalf("WriteLifecycleItems: %v", err)
	}
	output := stdout.String()
	for _, want := range []string{"Started scenario 'alpha' (healthy)", "Ports: API_PORT=18080", "URLs:", "API_PORT: http://localhost:18080", "Failed resources: qdrant"} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in output:\n%s", want, output)
		}
	}
}

func TestWriteLifecycleItemsHumanStartsWithBlankLine(t *testing.T) {
	// Leading blank line separates the summary from the preceding
	// progress pings / slog stream. Compact/JSON modes have their own
	// coverage; this test guards the normal-mode visual spacing.
	var stdout bytes.Buffer
	err := WriteLifecycleItems(&stdout, cliout.FormatHuman, []LifecycleItemOutput{{
		Name:   "alpha",
		Status: "restarted",
	}})
	if err != nil {
		t.Fatalf("WriteLifecycleItems: %v", err)
	}
	if !strings.HasPrefix(stdout.String(), "\n") {
		t.Fatalf("expected leading blank line, got %q", stdout.String())
	}
}

func TestWriteLifecycleItemsHumanIncludesLogsFooter(t *testing.T) {
	// The summary must always point users at the log tail command so
	// they know how to drill deeper on any status — started, restarted,
	// stopped, or degraded.
	cases := []string{"started", "restarted", "stopped", "already_running"}
	for _, status := range cases {
		status := status
		t.Run(status, func(t *testing.T) {
			var stdout bytes.Buffer
			err := WriteLifecycleItems(&stdout, cliout.FormatHuman, []LifecycleItemOutput{{
				Name:   "alpha",
				Status: status,
			}})
			if err != nil {
				t.Fatalf("WriteLifecycleItems: %v", err)
			}
			if !strings.Contains(stdout.String(), "Logs: vrooli scenario logs alpha") {
				t.Fatalf("%s: missing Logs footer in %q", status, stdout.String())
			}
		})
	}
}

func TestWriteLifecycleItemsCompactOmitsLogsFooter(t *testing.T) {
	// Compact single-line mode is for scripts that want the minimum
	// possible signal per scenario; the Logs footer would break the
	// one-line-per-scenario contract documented on writeLifecycleItemsCompact.
	t.Setenv("VROOLI_OUTPUT", "quiet")
	var stdout bytes.Buffer
	err := WriteLifecycleItems(&stdout, cliout.FormatHuman, []LifecycleItemOutput{{
		Name:   "alpha",
		Status: "started",
	}})
	if err != nil {
		t.Fatalf("WriteLifecycleItems: %v", err)
	}
	if strings.Contains(stdout.String(), "Logs:") {
		t.Fatalf("compact mode must not include Logs footer, got %q", stdout.String())
	}
}

func TestWriteLifecycleItemsHumanUsesRestartVerb(t *testing.T) {
	var stdout bytes.Buffer
	err := WriteLifecycleItems(&stdout, cliout.FormatHuman, []LifecycleItemOutput{{
		Name:   "alpha",
		Status: "restarted",
	}})
	if err != nil {
		t.Fatalf("WriteLifecycleItems: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "Restarted scenario 'alpha'") {
		t.Fatalf("stdout = %q", got)
	}
}

func TestWriteLifecycleItemsCompactAtQuiet(t *testing.T) {
	t.Setenv("VROOLI_OUTPUT", "quiet")
	var stdout bytes.Buffer
	err := WriteLifecycleItems(&stdout, cliout.FormatHuman, []LifecycleItemOutput{{
		Name:   "alpha",
		Status: "restarted",
		Health: "healthy",
		Ports: map[string]int{
			"API_PORT": 18800,
			"UI_PORT":  36238,
		},
	}})
	if err != nil {
		t.Fatalf("WriteLifecycleItems: %v", err)
	}
	got := stdout.String()
	lines := strings.Count(strings.TrimRight(got, "\n"), "\n") + 1
	if lines != 1 {
		t.Fatalf("quiet output should be single line, got %d lines: %q", lines, got)
	}
	if !strings.Contains(got, "alpha") || !strings.Contains(got, "restarted") {
		t.Errorf("missing expected fields: %q", got)
	}
	if !strings.Contains(got, "API_PORT=18800") {
		t.Errorf("ports not inlined: %q", got)
	}
	if !strings.Contains(got, "OK") {
		t.Errorf("missing status glyph: %q", got)
	}
}

func TestWriteLifecycleItemsCompactFlagsFailures(t *testing.T) {
	t.Setenv("VROOLI_OUTPUT", "quiet")
	var stdout bytes.Buffer
	err := WriteLifecycleItems(&stdout, cliout.FormatHuman, []LifecycleItemOutput{{
		Name:               "alpha",
		Status:             "started",
		Health:             "degraded",
		FailedDependencies: []string{"workspace-sandbox"},
	}})
	if err != nil {
		t.Fatalf("WriteLifecycleItems: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "!!") {
		t.Errorf("missing failure glyph: %q", got)
	}
	if !strings.Contains(got, "failed deps") {
		t.Errorf("missing failed deps: %q", got)
	}
}

func TestParseStartRequestRequiresSingleNameWhenPathProvided(t *testing.T) {
	_, err := ParseStartRequest(false, []string{"alpha", "beta", "--path", "/tmp/custom"})
	if err == nil {
		t.Fatal("expected path + multiple names to fail")
	}
	if !strings.Contains(err.Error(), "accepts exactly one scenario name") {
		t.Fatalf("err = %v", err)
	}
}

func TestRenderListResponseHumanIncludesPortsWhenPresent(t *testing.T) {
	var stdout bytes.Buffer
	err := RenderListResponse(&stdout, cliout.FormatHuman, ListResponse{
		Items: []ListItemOutput{{
			Name:        "alpha",
			Description: "demo",
			Ports: []ListPortOutput{{
				Key:  "API_PORT",
				Port: 18080,
			}},
		}},
		RunningCount: 1,
	})
	if err != nil {
		t.Fatalf("RenderListResponse: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "API_PORT=18080") {
		t.Fatalf("stdout = %q", got)
	}
}

func TestParseOpenRequestAcceptsJSONAndPrintURL(t *testing.T) {
	req, err := ParseOpenRequest(false, []string{"alpha", "--print-url", "--json"})
	if err != nil {
		t.Fatalf("ParseOpenRequest: %v", err)
	}
	if req.ScenarioName != "alpha" || !req.PrintURL || !req.JSON {
		t.Fatalf("req = %+v", req)
	}
}

func TestRenderPortResponseHumanSingleMatchesContract(t *testing.T) {
	var stdout bytes.Buffer
	err := RenderPortResponse(&stdout, cliout.FormatHuman, PortResponse{
		Single: &PortSingleOutput{
			Success:  true,
			Scenario: "alpha",
			PortName: "UI_PORT",
			Port:     38080,
		},
	})
	if err != nil {
		t.Fatalf("RenderPortResponse: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "38080" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestParseValidateEnvRequestAcceptsJSON(t *testing.T) {
	req, err := ParseValidateEnvRequest(false, []string{"alpha", "--json"})
	if err != nil {
		t.Fatalf("ParseValidateEnvRequest: %v", err)
	}
	if req.Name != "alpha" || !req.JSON {
		t.Fatalf("req = %+v", req)
	}
}

func TestParseGenerateRequestRequiresTemplateName(t *testing.T) {
	_, err := ParseGenerateRequest(nil, &bytes.Buffer{}, func(name string) (TemplateInfo, error) {
		return TemplateInfo{}, nil
	}, ParseGenerateArgs)
	if err == nil {
		t.Fatal("expected missing template name error")
	}
}

func TestRenderGenerateResponseDryRun(t *testing.T) {
	var stdout bytes.Buffer
	err := RenderGenerateResponse(&stdout, cliout.FormatHuman, GenerateResult{
		TemplateName: "demo",
		Destination:  "/tmp/alpha",
		Values:       map[string]string{"SCENARIO_ID": "alpha"},
		DryRun:       true,
	})
	if err != nil {
		t.Fatalf("RenderGenerateResponse: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "[DRY-RUN] Would generate template demo at /tmp/alpha") {
		t.Fatalf("stdout = %q", output)
	}
}

func TestRenderGenerateResponseIncludesStartDocument(t *testing.T) {
	var stdout bytes.Buffer
	err := RenderGenerateResponse(&stdout, cliout.FormatHuman, GenerateResult{
		TemplateName: "demo",
		DisplayName:  "Alpha",
		Destination:  "/tmp/alpha",
		Values:       map[string]string{"SCENARIO_ID": "alpha"},
		Manifest:     TemplateManifest{StartDocument: "docs/START-HERE.md"},
	})
	if err != nil {
		t.Fatalf("RenderGenerateResponse: %v", err)
	}
	output := stdout.String()
	for _, want := range []string{
		"Start here:",
		"docs/START-HERE.md",
		"1. Read the start document",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in output:\n%s", want, output)
		}
	}
}

func TestParseRequirementsRequestTreatsHelpAsCommandHelp(t *testing.T) {
	_, err := ParseRequirementsRequest([]string{"--help"})
	if err == nil {
		t.Fatal("expected help-only error")
	}
	if !strings.Contains(err.Error(), RequirementsHelpText()) {
		t.Fatalf("err = %v", err)
	}
}

func TestRequirementsHelpTextIsGeneratedFromSubcommandSpecs(t *testing.T) {
	text := RequirementsHelpText()
	for _, want := range []string{
		"Scenario Requirements Commands",
		"report",
		"snapshot",
		"manual-log",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in help:\n%s", want, text)
		}
	}
}

func TestParseLogsArgsParsesFlagsAndStep(t *testing.T) {
	name, opts, err := ParseLogsArgs([]string{"alpha", "--follow", "--step", "build", "--runtime", "--previous", "--tail", "25"})
	if err != nil {
		t.Fatalf("ParseLogsArgs: %v", err)
	}
	if name != "alpha" {
		t.Fatalf("name = %q", name)
	}
	if !opts.Follow || opts.StepName != "build" || !opts.Runtime || !opts.Previous || opts.Tail != 25 {
		t.Fatalf("opts = %+v", opts)
	}
}

func TestParseLogsArgsRejectsInvalidTail(t *testing.T) {
	cases := [][]string{
		{"alpha", "--tail", "nope"},
		{"alpha", "--tail", "0"},
		{"alpha", "--tail", "-1"},
	}
	for _, args := range cases {
		args := args
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			if _, _, err := ParseLogsArgs(args); err == nil {
				t.Fatalf("ParseLogsArgs(%v) expected error", args)
			}
		})
	}
}

func TestParseHealFromSandboxRequestUsesDefaultMergedPath(t *testing.T) {
	req, err := ParseHealFromSandboxRequest("/merged", nil)
	if err != nil {
		t.Fatalf("ParseHealFromSandboxRequest: %v", err)
	}
	if req.MergedPath != "/merged" || req.DryRun {
		t.Fatalf("req = %+v", req)
	}
}

func TestBuildListPortsSortsAndMapsRecords(t *testing.T) {
	manifest := scenariomodel.ServiceManifest{
		Ports: map[string]scenariomodel.Port{
			"api": {EnvVar: "API_PORT"},
			"ui":  {EnvVar: "UI_PORT"},
		},
	}

	listPorts, ports := BuildListPorts(manifest, []process.Record{
		{Step: "start-ui", Port: 38080},
		{Step: "start-api", Port: 18080},
	})

	if len(listPorts) != 2 {
		t.Fatalf("len(listPorts) = %d, want 2", len(listPorts))
	}
	if listPorts[0].Key != "API_PORT" || listPorts[1].Key != "UI_PORT" {
		t.Fatalf("listPorts = %#v", listPorts)
	}
	if ports["API_PORT"] != 18080 || ports["UI_PORT"] != 38080 {
		t.Fatalf("ports = %#v", ports)
	}
}

func TestBuildListPortsKeepsFirstExplicitRecordPerPort(t *testing.T) {
	manifest := scenariomodel.ServiceManifest{
		Ports: map[string]scenariomodel.Port{
			"api": {EnvVar: "API_PORT"},
		},
	}

	listPorts, ports := BuildListPorts(manifest, []process.Record{
		{Step: "start-api", Port: 18080},
		{Step: "run-api", Port: 19090},
	})

	if len(listPorts) != 1 {
		t.Fatalf("len(listPorts) = %d, want 1", len(listPorts))
	}
	if listPorts[0].Port != 18080 {
		t.Fatalf("listPorts[0].Port = %d, want 18080", listPorts[0].Port)
	}
	if ports["API_PORT"] != 18080 {
		t.Fatalf("ports = %#v", ports)
	}
}

func TestParseOptionalScenarioNameAndJSONValidation(t *testing.T) {
	name, jsonFlag, err := ParseOptionalScenarioNameAndJSON("status", false, []string{"alpha", "--json"})
	if err != nil {
		t.Fatalf("ParseOptionalScenarioNameAndJSON() error = %v", err)
	}
	if name != "alpha" || !jsonFlag {
		t.Fatalf("name/json = %q/%v", name, jsonFlag)
	}

	if _, _, err := ParseOptionalScenarioNameAndJSON("status", false, []string{"alpha", "beta"}); err == nil {
		t.Fatal("expected duplicate scenario names to fail")
	}
	if _, _, err := ParseOptionalScenarioNameAndJSON("status", false, []string{"--bogus"}); err == nil {
		t.Fatal("expected unknown option to fail")
	}
	if _, _, err := ParseScenarioNameAndJSON("info", false, nil); err == nil {
		t.Fatal("expected missing scenario name to fail")
	}
}

func TestParseScenarioStartArgsAndSingleStartValidation(t *testing.T) {
	names, opts, jsonFlag, openAfter, err := ParseScenarioStartArgs(false, []string{
		"alpha", "beta", "--json", "--open", "--best-effort", "--clean-stale", "--path", "/tmp/custom",
	})
	if err != nil {
		t.Fatalf("ParseScenarioStartArgs() error = %v", err)
	}
	if got := strings.Join(names, ","); got != "alpha,beta" {
		t.Fatalf("names = %q", got)
	}
	if !jsonFlag || !openAfter || !opts.BestEffort || !opts.CleanStale || opts.CustomPath != "/tmp/custom" {
		t.Fatalf("opts/json/open = %+v/%v/%v", opts, jsonFlag, openAfter)
	}

	if _, _, _, _, err := ParseScenarioStartArgs(false, []string{"--path"}); err == nil {
		t.Fatal("expected missing --path value to fail")
	}
	if _, _, _, _, err := ParseScenarioStartArgs(false, []string{"--bogus"}); err == nil {
		t.Fatal("expected unknown option to fail")
	}
	if _, _, _, _, err := ParseScenarioSingleStartArgs("restart", false, nil); err == nil {
		t.Fatal("expected missing restart target to fail")
	}
	if _, _, _, _, err := ParseScenarioSingleStartArgs("restart", false, []string{"alpha", "beta"}); err == nil {
		t.Fatal("expected duplicate restart targets to fail")
	}
}

func TestTemplateHelpAndManualHooksOutput(t *testing.T) {
	var templateHelp bytes.Buffer
	RenderTemplateHelp(&templateHelp)
	helpText := templateHelp.String()
	if !strings.Contains(helpText, "show               Show scenario template details") {
		t.Fatalf("template help = %q", templateHelp.String())
	}
	if !strings.Contains(helpText, "validate           Validate scenario templates") {
		t.Fatalf("template help = %q", helpText)
	}

	var generateHelp bytes.Buffer
	RenderGenerateHelp(&generateHelp)
	if !strings.Contains(generateHelp.String(), "--run-hooks") {
		t.Fatalf("generate help = %q", generateHelp.String())
	}

	var hooks bytes.Buffer
	WriteTemplateHooks(&hooks, TemplateManifest{
		PostHooks: []TemplateHook{{Description: "Install deps", Cmd: "pnpm install"}},
	})
	if !strings.Contains(hooks.String(), "Install deps") {
		t.Fatalf("hook output = %q", hooks.String())
	}
}

func TestTemplateParsersCaptureFlagsAndValues(t *testing.T) {
	manifest := TemplateManifest{
		RequiredVars: map[string]TemplateVar{
			"SCENARIO_ID":           {Flag: "id"},
			"SCENARIO_DISPLAY_NAME": {Flag: "display-name"},
			"SCENARIO_DESCRIPTION":  {Flag: "description"},
		},
		OptionalVars: map[string]TemplateVar{
			"AUTHOR": {Flag: "author"},
		},
	}

	var stderr bytes.Buffer
	opts, err := ParseGenerateArgs([]string{
		"--id", "alpha",
		"--display-name=Alpha App",
		"--description", "Generated alpha",
		"--design", "vrooli-default",
		"--var", "CUSTOM=1",
		"--unknown", "mystery",
	}, manifest, &stderr)
	if err != nil {
		t.Fatalf("ParseGenerateArgs() error = %v", err)
	}
	if opts.Values["SCENARIO_ID"] != "alpha" ||
		opts.Values["SCENARIO_DISPLAY_NAME"] != "Alpha App" ||
		opts.Values["SCENARIO_DESCRIPTION"] != "Generated alpha" ||
		opts.Values["CUSTOM"] != "1" ||
		opts.Design != "vrooli-default" {
		t.Fatalf("opts = %#v", opts)
	}
	if !strings.Contains(stderr.String(), "unknown flag --unknown") {
		t.Fatalf("stderr = %q", stderr.String())
	}

	if _, _, _, err := ParseTemplateFlag("--display-name", []string{"--display-name"}, 0); err == nil {
		t.Fatal("expected ParseTemplateFlag() to reject missing value")
	}
	if _, _, err := ParseTemplateKeyValue("broken"); err == nil {
		t.Fatal("expected ParseTemplateKeyValue() to reject invalid pair")
	}
}

func TestParseTemplateValidateRequest(t *testing.T) {
	req, err := ParseTemplateValidateRequest(nil)
	if err != nil {
		t.Fatalf("ParseTemplateValidateRequest() error = %v", err)
	}
	if req.Mode != TemplateValidationModeShallow || req.TemplateName != "" || req.RetainTemp || req.TestPreset != DefaultTemplateValidationTestPreset {
		t.Fatalf("default req = %#v", req)
	}

	req, err = ParseTemplateValidateRequest([]string{"--mode", "deep", "--template=react-vite", "--retain-temp", "--test-preset", "quick"})
	if err != nil {
		t.Fatalf("ParseTemplateValidateRequest(deep) error = %v", err)
	}
	if req.Mode != TemplateValidationModeDeep || req.TemplateName != "react-vite" || !req.RetainTemp || req.TestPreset != "quick" {
		t.Fatalf("deep req = %#v", req)
	}

	req, err = ParseTemplateValidateRequest([]string{"--mode=shallow", "--template", "react-vite"})
	if err != nil {
		t.Fatalf("ParseTemplateValidateRequest(shallow) error = %v", err)
	}
	if req.Mode != TemplateValidationModeShallow || req.TemplateName != "react-vite" {
		t.Fatalf("shallow req = %#v", req)
	}

	if _, err := ParseTemplateValidateRequest([]string{"extra"}); err == nil {
		t.Fatal("expected ParseTemplateValidateRequest() to reject extra args")
	}
	if _, err := ParseTemplateValidateRequest([]string{"--mode", "full"}); err == nil {
		t.Fatal("expected ParseTemplateValidateRequest() to reject invalid mode")
	}
	if _, err := ParseTemplateValidateRequest([]string{"--test-preset", "quick"}); err == nil {
		t.Fatal("expected ParseTemplateValidateRequest() to reject shallow test preset")
	}
	if _, err := ParseTemplateValidateRequest([]string{"--retain-temp"}); err == nil {
		t.Fatal("expected ParseTemplateValidateRequest() to reject shallow retain temp")
	}
}

func TestDesignParsers(t *testing.T) {
	if _, err := ParseDesignListRequest(nil); err != nil {
		t.Fatalf("ParseDesignListRequest() error = %v", err)
	}
	show, err := ParseDesignShowRequest([]string{"vrooli-default"})
	if err != nil {
		t.Fatalf("ParseDesignShowRequest() error = %v", err)
	}
	if show.ID != "vrooli-default" {
		t.Fatalf("show.ID = %q", show.ID)
	}
	validate, err := ParseDesignValidateRequest([]string{"--all"})
	if err != nil {
		t.Fatalf("ParseDesignValidateRequest(--all) error = %v", err)
	}
	if !validate.All || validate.ID != "" {
		t.Fatalf("validate = %#v", validate)
	}
	validate, err = ParseDesignValidateRequest([]string{"vrooli-default"})
	if err != nil {
		t.Fatalf("ParseDesignValidateRequest(id) error = %v", err)
	}
	if validate.All || validate.ID != "vrooli-default" {
		t.Fatalf("validate = %#v", validate)
	}
	if _, err := ParseDesignValidateRequest([]string{"vrooli-default", "--all"}); err == nil {
		t.Fatal("expected id plus --all to fail")
	}
}

func TestBuildScenarioStatusItemAndHumanWriters(t *testing.T) {
	startedAt := time.Date(2026, time.April, 10, 12, 0, 0, 0, time.UTC)
	fixedPort := 5432
	item := scenariomodel.Scenario{
		Slug:       "alpha",
		Path:       "/repo/scenarios/alpha",
		Redirected: true,
		Manifest: scenariomodel.ServiceManifest{
			Service: scenariomodel.ServiceMetadata{
				Name:        "alpha",
				DisplayName: "Alpha",
				Description: "Alpha scenario",
				Version:     "0.1.0",
				Type:        "tool",
				Category:    "ops",
				Tags:        []string{"internal", "go"},
			},
			Ports: map[string]scenariomodel.Port{
				"api": {EnvVar: "API_PORT", Range: "15000-19999"},
				"db":  {Port: &fixedPort},
			},
			Lifecycle: scenariomodel.Lifecycle{Version: "2.0.0"},
		},
	}
	runtimeState := process.ScenarioRuntime{
		ProcessCount: 1,
		Runtime:      "2m",
		StartedAt:    &startedAt,
		Records: []process.Record{
			{Step: "start-api", PID: 1234, Port: 18080, StartedAt: startedAt},
		},
	}

	status := BuildStatusItem(item, runtimeState)
	if status.Status != "running" || status.Health != "running" {
		t.Fatalf("status item = %+v", status)
	}

	var infoOut bytes.Buffer
	WriteInfoHuman(&infoOut, BuildInfoData(item), BuildRuntimeData(item.Manifest, runtimeState))
	for _, want := range []string{
		"Configured ports:",
		"API_PORT (api)",
		"DB_PORT (db) fixed=5432",
		"Version: 0.1.0",
		"Type: tool",
		"Category: ops",
		"Tags: internal, go",
		"Lifecycle version: 2.0.0",
		"Sandbox: using redirected scenario path",
	} {
		if !strings.Contains(infoOut.String(), want) {
			t.Fatalf("missing %q in info output:\n%s", want, infoOut.String())
		}
	}

	var tableOut bytes.Buffer
	WriteStatusTable(&tableOut, []StatusItemOutput{status})
	if !strings.Contains(tableOut.String(), "Name") || !strings.Contains(tableOut.String(), "alpha") {
		t.Fatalf("scenario table output = %s", tableOut.String())
	}

	var statusOut bytes.Buffer
	WriteStatusHuman(&statusOut, StatusSingleOutput{
		Scenario: status,
		Info:     BuildInfoData(item),
		Runtime:  BuildRuntimeData(item.Manifest, runtimeState),
	})
	if !strings.Contains(statusOut.String(), "Health: running") || !strings.Contains(statusOut.String(), "Processes:") {
		t.Fatalf("scenario status output = %s", statusOut.String())
	}
}

func TestBuildListPortsFallsBackToEnvironment(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("process environment inspection uses /proc on linux")
	}

	cmd := exec.Command("sleep", "30")
	cmd.Env = append(os.Environ(), "API_PORT=18080", "WS_PORT=28080")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})

	manifest := scenariomodel.ServiceManifest{
		Ports: map[string]scenariomodel.Port{
			"api":       {EnvVar: "API_PORT"},
			"websocket": {EnvVar: "WS_PORT"},
		},
	}

	var listPorts []ListPortOutput
	var ports map[string]int
	for attempt := 0; attempt < 20; attempt++ {
		listPorts, ports = BuildListPorts(manifest, []process.Record{{
			PID:  cmd.Process.Pid,
			Step: "start-api",
			Port: 18080,
		}})
		if ports["WS_PORT"] == 28080 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if len(listPorts) != 1 || listPorts[0].Key != "API_PORT" {
		t.Fatalf("list ports = %#v", listPorts)
	}
	if ports["API_PORT"] != 18080 || ports["WS_PORT"] != 28080 {
		t.Fatalf("ports = %#v", ports)
	}
}

func TestCopyHelpersReturnIndependentSlices(t *testing.T) {
	originalStrings := []string{"alpha"}
	originalRecords := []process.Record{{PID: 1234}}
	copiedStrings := CopyStrings(originalStrings)
	copiedRecords := CopyProcessRecords(originalRecords)
	if len(CopyStrings(nil)) != 0 || len(CopyProcessRecords(nil)) != 0 {
		t.Fatal("expected nil inputs to return empty slices")
	}

	copiedStrings[0] = "beta"
	copiedRecords[0].PID = 99
	if originalStrings[0] != "alpha" || originalRecords[0].PID != 1234 {
		t.Fatal("expected copies to avoid mutating originals")
	}
}

func TestCopyProcessRecordsReturnsIndependentSlice(t *testing.T) {
	values := []process.Record{{Step: "start-api", Port: 18080}}
	copied := CopyProcessRecords(values)
	values[0].Port = 19090

	if copied[0].Port != 18080 {
		t.Fatalf("copied = %#v", copied)
	}
}
