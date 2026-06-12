package shareddriftcli

import (
	"bytes"
	"encoding/json"
	"testing"

	shareddrift "github.com/vrooli/vrooli/internal/app/shareddrift"
	"github.com/vrooli/vrooli/internal/cliout"
)

// TestRenderJSONContract pins the `check-shared-drift --json` wire shape,
// including that elapsed_ms stays a JSON number (the int32 policy — int64 would
// have serialized as a string under protojson).
func TestRenderJSONContract(t *testing.T) {
	report := shareddrift.Report{
		Clean:           false,
		Root:            "/repo",
		TouchedPackages: []string{"packages/api-core"},
		OnlyTouchedUsed: true,
		BuildChecked:    true,
		ElapsedMs:       1234,
		Scenarios: []shareddrift.ScenarioReport{
			{Path: "/s/foo", APIDir: "/s/foo/api", Status: shareddrift.StatusStaleModules, DiffPaths: []string{"go.mod"}},
		},
	}

	var buf bytes.Buffer
	if err := Render(&buf, cliout.FormatJSON, report); err != nil {
		t.Fatalf("Render: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if got["clean"] != false || got["root"] != "/repo" {
		t.Errorf("top-level mismatch: clean=%v root=%v", got["clean"], got["root"])
	}
	// int32 policy: elapsed_ms must be a JSON number, not a string.
	if v, ok := got["elapsed_ms"].(float64); !ok || v != 1234 {
		t.Errorf("elapsed_ms must be JSON number 1234, got %v (%T)", got["elapsed_ms"], got["elapsed_ms"])
	}
	if got["only_touched"] != true {
		t.Errorf("only_touched (snake_case?): %v", got["only_touched"])
	}
	scenarios, ok := got["scenarios"].([]any)
	if !ok || len(scenarios) != 1 {
		t.Fatalf("scenarios: want 1, got %v", got["scenarios"])
	}
	first := scenarios[0].(map[string]any)
	if first["api_dir"] != "/s/foo/api" || first["status"] != "stale-modules" {
		t.Errorf("scenario mapping (snake_case?): %v", first)
	}
}
