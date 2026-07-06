package validation

import (
	"testing"

	clitest "ai-gateway/cli/internal/testutil"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"

	"github.com/vrooli/cli-core/cliapp"
)

func TestManifestCoversScenarioValidationService(t *testing.T) { // [REQ:AIGW-CLI-OPERATIONS]
	cliapp.RequireProtoServiceCoverage(t, clitest.ManifestBytes(t), scenariovalidationv1.File_scenario_validation_v1_validation_proto, "ScenarioValidationService")
}
