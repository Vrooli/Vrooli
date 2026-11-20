//go:build integration

// This check runs the real fleet scanner against the live workspace. It is an
// integration check, not a unit test: the scan reads every scenario UI and its
// verdict changes when another writer touches the tree, so running it in the
// parallel `go test ./...` sweep made an unrelated scenario's edit look like
// this package's failure. Run it alone with `make test-fleet-contract`.
package componenttests

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"react-component-library/handlers/components"
)

// Exercise the actual descriptor-owned phase command against the current
// workspace. The unit tests above inject a fake pnpm command to isolate error
// handling; this integration check verifies that the registered provider path
// can execute the real fleet scanner and consume its ratchet report.
func TestConsumerContractPhaseRunsActualFleetChecker(t *testing.T) {
	sourceRoot, err := components.DefaultSourceRoot()
	require.NoError(t, err)
	handler := &sharedHandler{sourceRoot: sourceRoot}
	response, err := handler.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario:         "react-component-library",
		Phase:            consumerContractPhase,
		IncludeExecution: true,
	}))
	require.NoError(t, err)
	require.Equal(t, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED, response.Msg.GetStatus())
	require.True(t, response.Msg.GetAssessment().GetLocal().GetClean())
	require.NotNil(t, response.Msg.GetNativeDetail())

	var detail structpb.Struct
	require.NoError(t, anypb.UnmarshalTo(response.Msg.GetNativeDetail(), &detail, proto.UnmarshalOptions{}))
	fields := detail.AsMap()
	fleet, ok := fields["fleet"].(map[string]any)
	require.True(t, ok, "native detail must expose the fleet scan metrics")
	require.Greater(t, fleet["scenarios"].(float64), float64(100))
	require.Greater(t, fleet["files"].(float64), float64(1000))
	require.Contains(t, fields, "ratchet")
}
