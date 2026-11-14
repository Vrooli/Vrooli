package authcli

import (
	"bytes"
	"encoding/json"
	"testing"

	authapp "github.com/vrooli/vrooli/internal/app/auth"
	"github.com/vrooli/vrooli/internal/cliout"
)

// TestRenderStatusJSONContract pins the auth-status `--json` wire shape.
func TestRenderStatusJSONContract(t *testing.T) {
	report := authapp.Report{
		Statuses: []authapp.Status{
			{
				Name: "claude",
				Result: authapp.ProbeResult{
					State:         authapp.State("signed_in"),
					Detail:        "ok",
					SignInCommand: []string{"claude", "login"},
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := RenderStatus(&buf, cliout.FormatJSON, report); err != nil {
		t.Fatalf("RenderStatus: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if got["success"] != true {
		t.Errorf("success: want true, got %v", got["success"])
	}
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("data missing/wrong type: %v", got["data"])
	}
	statuses, ok := data["statuses"].([]any)
	if !ok || len(statuses) != 1 {
		t.Fatalf("statuses: want 1, got %v", data["statuses"])
	}
	first := statuses[0].(map[string]any)
	if first["name"] != "claude" {
		t.Errorf("name: %v", first["name"])
	}
	result := first["result"].(map[string]any)
	if result["state"] != "signed_in" || result["detail"] != "ok" {
		t.Errorf("result mismatch: %v", result)
	}
	if _, ok := result["sign_in_command"].([]any); !ok {
		t.Errorf("sign_in_command missing/wrong (snake_case?): %v", result)
	}
}
