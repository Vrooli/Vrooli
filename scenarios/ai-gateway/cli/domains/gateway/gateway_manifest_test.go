package gateway

import (
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"

	gatewayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/gateway"

	"github.com/vrooli/cli-core/cliapp"
)

func TestManifestCoversGatewayService(t *testing.T) { // [REQ:AIGW-CLI-OPERATIONS]
	cliapp.RequireProtoServiceCoverage(t, clitest.ManifestBytes(t), gatewayv1.File_ai_gateway_v1_gateway_gateway_proto, "GatewayService")
}
