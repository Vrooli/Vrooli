package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestRegisterLoadsProviderGroup(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	group, err := Register(&cliapp.ScenarioApp{}, manifest)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if group.Name != GroupName || len(group.Subcommands) != 1 {
		t.Fatalf("unexpected group: %+v", group)
	}
}

func TestPresentationReportPreservesProviderOrder(t *testing.T) {
	summary, lines := presentationReport(&commonv1.PhasePresentation{
		ContractVersion: "v1", Provider: "experience-manager", Phase: "experience", CurrentLevel: "L1",
		Capabilities: []*commonv1.PhaseCapabilityPresentation{
			{Label: "Second", CurrentLevel: "L1"},
			{Label: "First", CurrentLevel: "L0"},
		},
	})
	if summary != "experience-manager/experience: L1" {
		t.Fatalf("summary = %q", summary)
	}
	if len(lines) != 2 || lines[0] != "Second: L1" || lines[1] != "First: L0" {
		t.Fatalf("lines = %v", lines)
	}
}
