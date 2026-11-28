package catalog

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
)

func TestAggregateGateMatrixPreservesDefinitionOrder(t *testing.T) {
	ordered := []gateMatrixResult{
		{response: testGateResponse("first", 2)},
		{}, // skipped definitions retain their matrix position without adding output
		{response: testGateResponse("third", 5)},
	}

	got, err := aggregateGateMatrix(ordered)
	require.NoError(t, err)
	require.Equal(t, int32(7), got.Msg.GetInspectedFiles())
	require.Equal(t, []string{"first", "third"}, []string{
		got.Msg.GetFindings()[0].GetCode(),
		got.Msg.GetFindings()[1].GetCode(),
	})
}

func testGateResponse(code string, inspected int32) *connect.Response[catalogv1.RunGateResponse] {
	return connect.NewResponse(&catalogv1.RunGateResponse{
		InspectedFiles: inspected,
		Findings:       []*catalogv1.GateFinding{{Code: code}},
	})
}
