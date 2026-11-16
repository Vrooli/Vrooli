package desktopoptimize

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"connectrpc.com/connect"
	optimizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/optimization"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/optimization/optimizationconnect"
)

const maxConnectRequestBytes = 64 * 1024

// ConnectService exposes only the fixed staged DP1 operations. The production
// API currently constructs it without evaluator/artifact owners, so calls fail
// closed until those owner implementations are supplied.
type ConnectService struct {
	optimizationconnect.UnimplementedDesktopOptimizationServiceHandler
	Dependencies Dependencies
}

func (s ConnectService) Measure(ctx context.Context, request *connect.Request[optimizationv1.MeasureRequest]) (*connect.Response[optimizationv1.StageReceipt], error) {
	var req Request
	if err := decodeRequest(request.Msg.GetRequestJson(), &req); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := validateOperation(request.Msg.GetOperationId(), req.RunID, "measure"); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if s.Dependencies.Evaluator == nil || s.Dependencies.Journal == nil {
		return nil, unavailable("DP1 measurement and receipt journal owners are unavailable")
	}
	return encodeReceipt(Measure(ctx, s.Dependencies, req))
}

func (s ConnectService) Promote(ctx context.Context, request *connect.Request[optimizationv1.PromoteRequest]) (*connect.Response[optimizationv1.StageReceipt], error) {
	var req Request
	if err := decodeRequest(request.Msg.GetRequestJson(), &req); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := validateOperation(request.Msg.GetOperationId(), req.RunID, "promote"); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if s.Dependencies.Promotion == nil || s.Dependencies.Journal == nil {
		return nil, unavailable("DP1 promotion and receipt journal owners are unavailable")
	}
	return encodeReceipt(Promote(ctx, s.Dependencies, req))
}

func (s ConnectService) Cleanup(ctx context.Context, request *connect.Request[optimizationv1.CleanupRequest]) (*connect.Response[optimizationv1.StageReceipt], error) {
	var req Request
	if err := decodeRequest(request.Msg.GetRequestJson(), &req); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := validateOperation(request.Msg.GetOperationId(), req.RunID, "remove-predecessor"); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if s.Dependencies.Promotion == nil || s.Dependencies.Journal == nil {
		return nil, unavailable("DP1 predecessor removal and receipt journal owners are unavailable")
	}
	return encodeReceipt(Cleanup(ctx, s.Dependencies, req))
}

func decodeRequest(encoded string, request *Request) error {
	if strings.TrimSpace(encoded) == "" {
		return errors.New("request_json is required")
	}
	if len(encoded) > maxConnectRequestBytes {
		return fmt.Errorf("request_json exceeds %d bytes", maxConnectRequestBytes)
	}
	decoder := json.NewDecoder(bytes.NewBufferString(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(request); err != nil {
		return fmt.Errorf("decode desktop optimization request: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errors.New("desktop optimization request must contain exactly one JSON value")
	}
	return nil
}

func validateOperation(operationID, runID, stage string) error {
	want := strings.TrimSpace(runID) + "/" + stage
	if strings.TrimSpace(runID) == "" || operationID != want {
		return fmt.Errorf("operation_id must be the stable %s owner call ID %q", stage, want)
	}
	return nil
}

func encodeReceipt(receipt Receipt) (*connect.Response[optimizationv1.StageReceipt], error) {
	encoded, err := json.Marshal(receipt)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode desktop optimization receipt: %w", err))
	}
	return connect.NewResponse(&optimizationv1.StageReceipt{ReceiptJson: string(encoded)}), nil
}

func unavailable(message string) error {
	return connect.NewError(connect.CodeUnavailable, errors.New(message))
}

var _ optimizationconnect.DesktopOptimizationServiceHandler = ConnectService{}
