package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/vrooli/browser-automation-studio/internal/toolexecution"
	toolsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/tools"
)

// service implements toolsconnect.ToolsServiceHandler.
type service struct {
	deps Deps
}

func (s *service) List(
	ctx context.Context,
	_ *connect.Request[toolsv1.ListToolsRequest],
) (*connect.Response[toolsv1.ListToolsResponse], error) {
	manifest := s.deps.Registry.GetManifest(ctx)
	return connect.NewResponse(&toolsv1.ListToolsResponse{Manifest: manifest}), nil
}

func (s *service) Get(
	ctx context.Context,
	req *connect.Request[toolsv1.GetToolRequest],
) (*connect.Response[toolsv1.GetToolResponse], error) {
	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errMissingToolName)
	}
	tool := s.deps.Registry.GetTool(ctx, name)
	if tool == nil {
		return nil, connect.NewError(connect.CodeNotFound, errToolNotFound)
	}
	return connect.NewResponse(&toolsv1.GetToolResponse{Tool: tool}), nil
}

func (s *service) Execute(
	ctx context.Context,
	req *connect.Request[toolsv1.ExecuteToolRequest],
) (*connect.Response[toolsv1.ExecuteToolResponse], error) {
	toolName := strings.TrimSpace(req.Msg.GetToolName())
	if toolName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errMissingToolName)
	}

	var args map[string]interface{}
	if a := req.Msg.GetArguments(); a != nil {
		args = a.AsMap()
	}

	result, err := s.deps.Executor.Execute(ctx, toolName, args)
	if err != nil {
		s.deps.Logger.WithError(err).WithField("tool", toolName).Error("tools.Execute failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp, encodeErr := encodeExecutionResult(result)
	if encodeErr != nil {
		s.deps.Logger.WithError(encodeErr).WithField("tool", toolName).Error("tools.Execute encode failed")
		return nil, connect.NewError(connect.CodeInternal, encodeErr)
	}
	return connect.NewResponse(resp), nil
}

// encodeExecutionResult converts the legacy ExecutionResult envelope into
// the proto response. The envelope intentionally carries success/code
// rather than mapping onto connect.Error codes, because tool-level error
// codes (unknown_tool, validation_error, ...) need to round-trip to
// callers untouched.
func encodeExecutionResult(r *toolexecution.ExecutionResult) (*toolsv1.ExecuteToolResponse, error) {
	if r == nil {
		return &toolsv1.ExecuteToolResponse{}, nil
	}
	out := &toolsv1.ExecuteToolResponse{
		Success: r.Success,
		Error:   r.Error,
		Code:    r.Code,
		IsAsync: r.IsAsync,
		RunId:   r.RunID,
		Status:  r.Status,
	}
	if r.Result != nil {
		s, err := toStruct(r.Result)
		if err != nil {
			return nil, err
		}
		out.Result = s
	}
	return out, nil
}

// toStruct normalizes the open-shape result payload into a Struct.
// Tool implementations return map[string]interface{} that may contain
// time.Time, uuid.UUID, or other JSON-marshalable values that structpb's
// direct constructor rejects. We JSON-round-trip through protojson to
// match the previous REST handler's wire format exactly (time.Time →
// RFC3339 string, etc).
func toStruct(v interface{}) (*structpb.Struct, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal tool result: %w", err)
	}
	// Non-object roots (string, number, array, bool, null) cannot fit a
	// Struct, so wrap them under a "value" key to preserve the result.
	if len(raw) > 0 && raw[0] != '{' {
		raw, err = json.Marshal(map[string]json.RawMessage{"value": raw})
		if err != nil {
			return nil, fmt.Errorf("marshal tool result envelope: %w", err)
		}
	}
	out := &structpb.Struct{}
	if err := protojson.Unmarshal(raw, out); err != nil {
		return nil, fmt.Errorf("decode tool result into struct: %w", err)
	}
	return out, nil
}
