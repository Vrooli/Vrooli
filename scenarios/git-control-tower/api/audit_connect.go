package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"connectrpc.com/connect"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/audit"
	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/audit/audit_v1connect"
	"google.golang.org/protobuf/types/known/structpb"
)

var _ auditconnect.AuditServiceHandler = auditConnectServer{}

type auditConnectServer struct{ server *Server }

func (s auditConnectServer) ListEntries(ctx context.Context, req *connect.Request[auditv1.ListAuditEntriesRequest]) (*connect.Response[auditv1.ListAuditEntriesResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("audit query request is required"))
	}
	limit := int(req.Msg.GetLimit())
	if limit == 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}
	result, err := s.server.audit.Query(ctx, AuditQueryRequest{
		Operation: AuditOperation(req.Msg.GetOperation()), Branch: req.Msg.GetBranch(),
		Limit: limit, Offset: int(req.Msg.GetOffset()),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &auditv1.ListAuditEntriesResponse{
		Entries: make([]*auditv1.AuditEntry, 0, len(result.Entries)),
		Total:   int32(result.Total), Timestamp: result.Timestamp.UTC().Format(time.RFC3339Nano),
	}
	for _, entry := range result.Entries {
		mapped, err := auditEntryToProto(entry)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		response.Entries = append(response.Entries, mapped)
	}
	return connect.NewResponse(response), nil
}

func auditEntryToProto(entry AuditEntry) (*auditv1.AuditEntry, error) {
	result := &auditv1.AuditEntry{
		Id: entry.ID, Operation: string(entry.Operation), RepoDir: entry.RepoDir,
		Branch: entry.Branch, Paths: append([]string(nil), entry.Paths...),
		CommitHash: entry.CommitHash, CommitMessage: entry.CommitMessage,
		Success: entry.Success, Error: entry.Error,
		Timestamp: entry.Timestamp.UTC().Format(time.RFC3339Nano),
	}
	if entry.Metadata != nil {
		encoded, err := json.Marshal(entry.Metadata)
		if err != nil {
			return nil, fmt.Errorf("encode audit metadata: %w", err)
		}
		var value map[string]any
		if err := json.Unmarshal(encoded, &value); err != nil {
			return nil, fmt.Errorf("decode audit metadata: %w", err)
		}
		result.Metadata, err = structpb.NewStruct(value)
		if err != nil {
			return nil, fmt.Errorf("convert audit metadata: %w", err)
		}
	}
	return result, nil
}
