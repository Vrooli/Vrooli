package bindings

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	domainconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain/domainconnect"
)

type fakeAggregateService struct{}

func (fakeAggregateService) AggregateReceipts(context.Context, *connect.Request[domain.ReceiptAggregateRequest]) (*connect.Response[domain.ReceiptAggregateResponse], error) {
	return connect.NewResponse(&domain.ReceiptAggregateResponse{Aggregates: []*domain.ReceiptAggregate{
		{TargetScenario: "demo", Operation: "POST /api/v1/read/list", InvocationCount: 4, DistinctVerifiedCallers: 2, UnattributedRemainder: 1, LastInvokedAt: "2026-08-18T12:00:00Z"},
	}}), nil
}

func TestReceiptExerciseReaderUsesGeneratedAggregateContract(t *testing.T) {
	path, handler := domainconnect.NewReceiptAggregateServiceHandler(fakeAggregateService{})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	reader := NewReceiptExerciseReader(func(context.Context, string) (string, error) { return server.URL, nil }, server.Client())
	observations, err := reader.Aggregate(context.Background(), "demo", time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	require.Len(t, observations, 1)
	require.Equal(t, int64(4), observations[0].Invocations)
	require.Equal(t, int64(2), observations[0].DistinctVerifiedCallers)

	binding := &bindingsv1.Binding{Id: "demo/read/list", Group: "read", Command: "list", Service: "DemoService", Method: "List"}
	exercise := exerciseForBinding(binding, observations, true)
	require.Equal(t, int64(4), exercise.GetInvocations())
	require.Equal(t, int64(1), exercise.GetUnattributedRemainder())
	require.Equal(t, bindingsv1.ConditionStatus_CONDITION_STATUS_HEALTHY, exercise.GetFamily().GetStatus())
}

func TestExerciseForBindingFailsClosedWhenAggregateUnavailable(t *testing.T) {
	exercise := exerciseForBinding(&bindingsv1.Binding{Id: "demo/read/list"}, nil, false)
	require.Equal(t, bindingsv1.ConditionStatus_CONDITION_STATUS_UNINSTRUMENTED, exercise.GetFamily().GetStatus())
	require.Contains(t, exercise.GetFamily().GetReason(), "unavailable")
}
