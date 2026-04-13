package scenariocli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/process"
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
