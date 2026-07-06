package measures

import (
	"testing"

	clitest "ai-gateway/cli/internal/testutil"

	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/measures"

	"github.com/vrooli/cli-core/cliapp"
)

func TestManifestCoversMeasuresService(t *testing.T) { // [REQ:AIGW-CLI-OPERATIONS] [REQ:AIGW-ROUTE-MEASURES]
	cliapp.RequireProtoServiceCoverage(t, clitest.ManifestBytes(t), measuresv1.File_ai_gateway_v1_measures_measures_proto, "MeasuresService")
}
