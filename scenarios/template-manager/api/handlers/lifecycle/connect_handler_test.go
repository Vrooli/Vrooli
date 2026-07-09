package lifecycle

import (
	"testing"

	lifecyclev1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/lifecycle"
)

func TestGenerateScenarioValuesMapsDisplayNamePlaceholder(t *testing.T) {
	values := generateScenarioValues(&lifecyclev1.GenerateScenarioRequest{
		Id:          "alpha",
		DisplayName: "Alpha App",
		Description: "Alpha description",
		Values: map[string]string{
			"AUTHOR": "Test Agent",
		},
	})

	if values["SCENARIO_ID"] != "alpha" {
		t.Fatalf("SCENARIO_ID = %q, want alpha", values["SCENARIO_ID"])
	}
	if values["SCENARIO_DISPLAY_NAME"] != "Alpha App" {
		t.Fatalf("SCENARIO_DISPLAY_NAME = %q, want Alpha App", values["SCENARIO_DISPLAY_NAME"])
	}
	if values["SCENARIO_DESCRIPTION"] != "Alpha description" {
		t.Fatalf("SCENARIO_DESCRIPTION = %q, want Alpha description", values["SCENARIO_DESCRIPTION"])
	}
	if values["AUTHOR"] != "Test Agent" {
		t.Fatalf("AUTHOR = %q, want Test Agent", values["AUTHOR"])
	}
	if _, ok := values["SCENARIO_NAME"]; ok {
		t.Fatalf("SCENARIO_NAME should not be emitted; templates consume SCENARIO_DISPLAY_NAME")
	}
}
