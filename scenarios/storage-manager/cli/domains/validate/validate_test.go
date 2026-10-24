package validate

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestGroupName(t *testing.T) {
	if GroupName != "validate" {
		t.Fatalf("group name = %q", GroupName)
	}
}

func TestRegisterKeepsAPIFreeIsolationProofLocal(t *testing.T) {
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:           "storage-manager-test",
		Version:        "test",
		Description:    "test",
		DefaultAPIBase: "http://127.0.0.1:1",
		AllowAnonymous: true,
	})
	if err != nil {
		t.Fatalf("NewStandardScenarioApp() error: %v", err)
	}
	manifest := []byte(`{"name":"storage-manager","groups":[{"name":"validate","description":"Validate storage","commands":[{"name":"scenario","description":"Validate a scenario","binding":{"kind":"connect-rpc","service":"ScenarioValidationService","method":"ValidateScenario"},"governance":{"effect":"read","run_eligible":true,"permissions":["network:internal"]}}]}]}`)
	group, err := Register(core, manifest)
	if err != nil {
		t.Fatalf("Register() error: %v", err)
	}
	if group.NeedsAPI {
		t.Fatal("validate group must not force API preflight for local commands")
	}
	needsAPI := map[string]bool{}
	for _, command := range group.Subcommands {
		needsAPI[command.Name] = command.NeedsAPI
	}
	if !needsAPI["scenario"] || !needsAPI["resource"] || !needsAPI["tool"] || !needsAPI["safeguard"] || !needsAPI["fleet"] {
		t.Fatalf("API-backed commands lost their API requirement: %#v", needsAPI)
	}
	if needsAPI["prove-isolation"] {
		t.Fatal("prove-isolation must not require the API")
	}
}
