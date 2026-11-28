// Responsibility: expose nested-run reads (run wait, children wait, run tree)
// on the Connect surface.
package handlers

import (
	"context"
	"errors"
	"time"

	"agent-manager/internal/orchestration"
	"agent-manager/internal/protoconv"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// WaitRun blocks until one run settles or the bounded timeout elapses. The
// route clears the server write deadline (LongWaitResponse); the request's own
// timeout bounds it.
func (h *AgentManagerConnectHandler) WaitRun(ctx context.Context, req *connect.Request[apipb.WaitRunRequest]) (*connect.Response[apipb.WaitRunResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.Msg.GetRunId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	result, err := h.h.svc.WaitRun(ctx, id, time.Duration(req.Msg.GetTimeoutSeconds())*time.Second)
	if err != nil {
		return nil, workflowConnectError(err)
	}
	return connect.NewResponse(&apipb.WaitRunResponse{Run: protoconv.RunToProto(result.Run), Outcome: result.Outcome}), nil
}

// WaitRunChildren blocks until a direct child of the parent settles, no child
// is live, or the bounded timeout elapses.
func (h *AgentManagerConnectHandler) WaitRunChildren(ctx context.Context, req *connect.Request[apipb.WaitRunChildrenRequest]) (*connect.Response[apipb.WaitRunChildrenResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	parentID, err := uuid.Parse(req.Msg.GetParentRunId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	var since *time.Time
	if req.Msg.Since != nil {
		at := req.Msg.GetSince().AsTime()
		since = &at
	}
	result, err := h.h.svc.WaitRunChildren(ctx, parentID, since, time.Duration(req.Msg.GetTimeoutSeconds())*time.Second)
	if err != nil {
		return nil, workflowConnectError(err)
	}
	return connect.NewResponse(childrenWaitResultToProto(result)), nil
}

// GetRunTree reads one run and its descendants.
func (h *AgentManagerConnectHandler) GetRunTree(ctx context.Context, req *connect.Request[apipb.GetRunTreeRequest]) (*connect.Response[apipb.RunTree], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.Msg.GetRunId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	tree, err := h.h.svc.GetRunTree(ctx, id)
	if err != nil {
		return nil, workflowConnectError(err)
	}
	return connect.NewResponse(&apipb.RunTree{
		Root:          runTreeNodeToProto(tree.Root),
		NodeCount:     int32(tree.NodeCount),
		Truncated:     tree.Truncated,
		MaxChildDepth: int32(tree.MaxChildDepth),
	}), nil
}

func childrenWaitResultToProto(result *orchestration.ChildrenWaitResult) *apipb.WaitRunChildrenResponse {
	rows := func(children []orchestration.ChildRunSummary) []*apipb.RunChildSummary {
		out := make([]*apipb.RunChildSummary, 0, len(children))
		for _, child := range children {
			row := &apipb.RunChildSummary{RunId: child.RunID.String(), Status: string(child.Status)}
			if child.EndedAt != nil {
				row.EndedAt = timestamppb.New(*child.EndedAt)
			}
			out = append(out, row)
		}
		return out
	}
	return &apipb.WaitRunChildrenResponse{
		WakeReason: result.WakeReason,
		Ended:      rows(result.Ended),
		Active:     rows(result.Active),
		ObservedAt: timestamppb.New(result.ObservedAt),
	}
}

func runTreeNodeToProto(node *orchestration.RunTreeNode) *apipb.RunTreeNode {
	if node == nil || node.Run == nil {
		return nil
	}
	run := node.Run
	out := &apipb.RunTreeNode{
		RunId:               run.ID.String(),
		Depth:               int32(run.Depth),
		Status:              protoconv.RunStatusToProto(run.Status),
		Label:               run.Label,
		Model:               run.ActualModel,
		GoalDelivery:        run.GoalDelivery,
		ElapsedSeconds:      node.ElapsedSeconds,
		WeightedTokens:      node.WeightedTokens,
		WeightedTokensKnown: node.WeightedTokensKnown,
		Orphaned:            node.Orphaned,
		CharterPath:         run.CharterPath,
		StepBack:            protoconv.StepBackTriggersToProto(node.StepBack),
	}
	if out.Model == "" {
		out.Model = run.RequestedModel
	}
	if run.ResolvedConfig != nil {
		out.Runner = string(run.ResolvedConfig.RunnerType)
	}
	if run.ParentRunID != nil {
		parent := run.ParentRunID.String()
		out.ParentRunId = &parent
	}
	if run.StartedAt != nil {
		out.StartedAt = timestamppb.New(*run.StartedAt)
	}
	if run.EndedAt != nil {
		out.EndedAt = timestamppb.New(*run.EndedAt)
	}
	if run.OwnerExpiresAt != nil {
		out.OwnerExpiresAt = timestamppb.New(*run.OwnerExpiresAt)
	}
	for _, child := range node.Children {
		if converted := runTreeNodeToProto(child); converted != nil {
			out.Children = append(out.Children, converted)
		}
	}
	return out
}
