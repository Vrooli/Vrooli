package conformance

import (
	"testing"

	clitest "ai-gateway/cli/internal/testutil"

	conformancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/conformance"

	"github.com/vrooli/cli-core/cliapp"
)

func TestManifestCoversConformanceService(t *testing.T) { // [REQ:AIGW-CLI-OPERATIONS]
	cliapp.RequireProtoServiceCoverage(t, clitest.ManifestBytes(t), conformancev1.File_ai_gateway_v1_conformance_conformance_proto, "ConformanceService")
}
