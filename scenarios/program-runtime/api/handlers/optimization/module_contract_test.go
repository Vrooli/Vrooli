package optimization

import (
	"context"
	"strconv"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	optimizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/optimization"
)

func TestConnectWaitRejectsTimeoutOutsideOneShotContract(t *testing.T) {
	h := &handler{}
	for _, waitMS := range []uint32{0, 60_001} {
		t.Run(strconv.FormatUint(uint64(waitMS), 10), func(t *testing.T) {
			_, err := h.Wait(context.Background(), connect.NewRequest(&optimizationv1.WaitOptimizationRequest{
				RunId:  "run-1",
				WaitMs: waitMS,
			}))
			require.Error(t, err)
			require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		})
	}
}
