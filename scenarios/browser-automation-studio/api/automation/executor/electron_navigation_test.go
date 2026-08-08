package executor

import (
	"testing"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

func TestRewriteElectronScenarioNavigationUsesAdmittedRendererOrigin(t *testing.T) {
	destinationType := basactions.NavigateDestinationType_NAVIGATE_DESTINATION_TYPE_SCENARIO
	instruction := contracts.CompiledInstruction{
		Action: &basactions.ActionDefinition{
			Type: basactions.ActionType_ACTION_TYPE_NAVIGATE,
			Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{
				Url:             "http://localhost:22829/?view=generator",
				DestinationType: &destinationType,
				Scenario:        stringPointer("scenario-to-desktop"),
				ScenarioPath:    stringPointer("/?view=generator&scenario=scenario-to-desktop"),
			}},
		},
	}

	rewritten, err := rewriteElectronScenarioNavigation(instruction, &driver.ElectronTarget{
		RendererURL:  "http://127.0.0.1:24100/",
		ScenarioName: "scenario-to-desktop",
	})
	if err != nil {
		t.Fatalf("rewrite scenario navigation: %v", err)
	}
	if got, want := rewritten.Action.GetNavigate().GetUrl(), "http://127.0.0.1:24100/?view=generator&scenario=scenario-to-desktop"; got != want {
		t.Fatalf("rewritten URL = %q, want %q", got, want)
	}
	if got := instruction.Action.GetNavigate().GetUrl(); got != "http://localhost:22829/?view=generator" {
		t.Fatalf("rewrite mutated original instruction URL: %q", got)
	}
}

func TestRewriteElectronScenarioNavigationLeavesOtherDestinationsUntouched(t *testing.T) {
	instruction := contracts.CompiledInstruction{
		Action: &basactions.ActionDefinition{
			Type: basactions.ActionType_ACTION_TYPE_NAVIGATE,
			Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{
				Url: "https://example.com/",
			}},
		},
	}

	rewritten, err := rewriteElectronScenarioNavigation(instruction, &driver.ElectronTarget{
		RendererURL:  "http://127.0.0.1:24100/",
		ScenarioName: "scenario-to-desktop",
	})
	if err != nil {
		t.Fatalf("direct URL navigation should not be rewritten: %v", err)
	}
	if got := rewritten.Action.GetNavigate().GetUrl(); got != "https://example.com/" {
		t.Fatalf("direct URL navigation changed to %q", got)
	}
}

func stringPointer(value string) *string { return &value }
