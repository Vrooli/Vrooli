package channelmanager

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Channel Manager owns the consumer-specific adoption check; BAS remains
// unaware of this scenario and validates only the generic declaration shape.
func TestBASConsumerDeclarationIsScenarioOwnedAndNonSecret(t *testing.T) {
	path := filepath.Join("..", "..", "..", ".vrooli", "browser-automation-studio", "consumer-declaration.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var declaration struct {
		SchemaVersion string `json:"schemaVersion"`
		Profiles      []struct {
			Key         string `json:"key"`
			WorkflowRef string `json:"workflowRef"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(raw, &declaration); err != nil {
		t.Fatal(err)
	}
	if declaration.SchemaVersion != "browser-automation-studio.consumer-declaration/v1" || len(declaration.Profiles) != 1 {
		t.Fatalf("unexpected declaration: %+v", declaration)
	}
	profile := declaration.Profiles[0]
	if profile.Key == "" || profile.WorkflowRef == "" {
		t.Fatalf("profile must declare key and workflow reference: %+v", profile)
	}
	if _, err := os.Stat(filepath.Join("..", "..", "..", profile.WorkflowRef)); err != nil {
		t.Fatalf("declared workflow reference: %v", err)
	}
	lower := strings.ToLower(string(raw))
	for _, forbidden := range []string{"cookie", "password", "credential", "token", "runtimeprofile", "profileid", "proxy"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("declaration contains forbidden %q", forbidden)
		}
	}
}
