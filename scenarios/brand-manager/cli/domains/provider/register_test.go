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

func TestPresentationReportDoesNotSynthesizeHistoricalContract(t *testing.T) {
	summary, lines := presentationReport(&commonv1.PhasePresentation{ContractVersion: "legacy"})
	if summary == "" || len(lines) != 1 {
		t.Fatalf("historical presentation must be explicit: summary=%q lines=%v", summary, lines)
	}
}
