package desktopoptimize

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	optimizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/optimization"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/optimization/optimizationconnect"
)

type connectJournal struct{ records map[string]Receipt }

func (j *connectJournal) Append(receipt Receipt) error {
	if j.records == nil {
		j.records = map[string]Receipt{}
	}
	j.records[receipt.RunID] = receipt
	return nil
}

func (j *connectJournal) Find(runID string) (Receipt, bool, error) {
	receipt, ok := j.records[runID]
	return receipt, ok, nil
}

type connectEvaluator struct{ operationID string }

func (e *connectEvaluator) MeasurePaired(_ context.Context, operationID string, request Request, _ MeasurementPlan) (Matrix, error) {
	e.operationID = operationID
	return goodMatrix(request), nil
}

func TestConnectServiceRequiresExactOwnerCallIDAndFailsClosedWithoutOwners(t *testing.T) {
	path, handler := optimizationconnect.NewDesktopOptimizationServiceHandler(ConnectService{})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := optimizationconnect.NewDesktopOptimizationServiceClient(server.Client(), server.URL)
	request := `{"schema":"desktop-startup.optimize-request.v1","run_id":"run-1","target":"desktop-startup"}`

	_, err := client.Measure(context.Background(), connect.NewRequest(&optimizationv1.MeasureRequest{OperationId: "run-1/measure-replay", RequestJson: request}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = client.Measure(context.Background(), connect.NewRequest(&optimizationv1.MeasureRequest{OperationId: "run-1/measure", RequestJson: request}))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
	require.ErrorContains(t, err, "owners are unavailable")
}

func TestConnectServiceRejectsUnknownTrailingAndOversizedRequests(t *testing.T) {
	path, handler := optimizationconnect.NewDesktopOptimizationServiceHandler(ConnectService{})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := optimizationconnect.NewDesktopOptimizationServiceClient(server.Client(), server.URL)

	for name, requestJSON := range map[string]string{
		"unknown field":  `{"schema":"desktop-startup.optimize-request.v1","unexpected":true}`,
		"trailing value": `{"schema":"desktop-startup.optimize-request.v1"} {}`,
		"oversized":      `{"padding":"` + strings.Repeat("x", maxConnectRequestBytes) + `"}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := client.Measure(context.Background(), connect.NewRequest(&optimizationv1.MeasureRequest{OperationId: "run-1/measure", RequestJson: requestJSON}))
			require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		})
	}
}

func TestConnectServiceTranslatesMeasureReceiptAndStableOwnerID(t *testing.T) {
	now := time.Now().UTC()
	request := testRequest(now)
	request.PreflightRights = request.Rights
	request.PreflightDisk = request.Disk
	evaluator := &connectEvaluator{}
	service := ConnectService{Dependencies: Dependencies{Evaluator: evaluator, Journal: &connectJournal{}, Now: func() time.Time { return now }}}
	path, handler := optimizationconnect.NewDesktopOptimizationServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := optimizationconnect.NewDesktopOptimizationServiceClient(server.Client(), server.URL)
	encoded, err := json.Marshal(request)
	require.NoError(t, err)
	response, err := client.Measure(context.Background(), connect.NewRequest(&optimizationv1.MeasureRequest{OperationId: request.RunID + "/measure", RequestJson: string(encoded)}))
	require.NoError(t, err)
	var receipt Receipt
	require.NoError(t, json.Unmarshal([]byte(response.Msg.GetReceiptJson()), &receipt))
	require.Equal(t, "measurement-complete", receipt.Stage)
	require.Equal(t, request.RunID, receipt.RunID)
	require.NotEmpty(t, receipt.RequestDigest)
	require.Len(t, receipt.Matrix, 6)
	require.Equal(t, "run-1/measure", evaluator.operationID)
}
