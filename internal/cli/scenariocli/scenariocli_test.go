package scenariocli

import (
	"bytes"
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
		Name:   "alpha",
		Status: "started",
		Ports:  map[string]int{"API_PORT": 18080},
	}})
	if err != nil {
		t.Fatalf("WriteLifecycleItems: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, `"success": true`) || !strings.Contains(output, `"scenarios":`) {
		t.Fatalf("output = %s", output)
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

func TestParseRequirementsRequestTreatsHelpAsCommandHelp(t *testing.T) {
	_, err := ParseRequirementsRequest([]string{"--help"})
	if err == nil {
		t.Fatal("expected help-only error")
	}
	if !strings.Contains(err.Error(), RequirementsHelpText()) {
		t.Fatalf("err = %v", err)
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

func TestCopyProcessRecordsReturnsIndependentSlice(t *testing.T) {
	values := []process.Record{{Step: "start-api", Port: 18080}}
	copied := CopyProcessRecords(values)
	values[0].Port = 19090

	if copied[0].Port != 18080 {
		t.Fatalf("copied = %#v", copied)
	}
}
