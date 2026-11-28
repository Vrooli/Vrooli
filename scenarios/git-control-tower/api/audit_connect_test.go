package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/audit"
)

type auditQueryCapture struct {
	request AuditQueryRequest
	result  *AuditQueryResponse
	err     error
}

func (a *auditQueryCapture) Log(context.Context, AuditEntry) error { return nil }
func (a *auditQueryCapture) IsConfigured() bool                    { return true }
func (a *auditQueryCapture) Query(_ context.Context, request AuditQueryRequest) (*AuditQueryResponse, error) {
	a.request = request
	return a.result, a.err
}

func TestAuditConnectListEntriesPreservesFiltersPageAndOwnerFields(t *testing.T) {
	stamp := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	logger := &auditQueryCapture{result: &AuditQueryResponse{
		Entries: []AuditEntry{{
			ID: 8, Operation: AuditOpCommit, RepoDir: "/repo", Branch: "main", Paths: []string{"a.go"},
			CommitHash: "abc", CommitMessage: "save", Success: false, Error: "rejected", Timestamp: stamp,
			Metadata: map[string]any{"attempt": 2, "reason": "hook"},
		}},
		Total: 9, Timestamp: stamp,
	}}
	server := auditConnectServer{server: &Server{audit: logger}}
	response, err := server.ListEntries(context.Background(), connect.NewRequest(&auditv1.ListAuditEntriesRequest{
		Operation: "commit", Branch: "main", Limit: 5000, Offset: 3,
	}))
	if err != nil {
		t.Fatalf("ListEntries returned error: %v", err)
	}
	if logger.request.Operation != AuditOpCommit || logger.request.Branch != "main" || logger.request.Limit != 1000 || logger.request.Offset != 3 {
		t.Fatalf("query filters or bounds changed: %+v", logger.request)
	}
	if response.Msg.GetTotal() != 9 || len(response.Msg.GetEntries()) != 1 {
		t.Fatalf("unexpected response page: %+v", response.Msg)
	}
	entry := response.Msg.GetEntries()[0]
	if entry.GetId() != 8 || entry.GetRepoDir() != "/repo" || entry.GetError() != "rejected" || entry.GetPaths()[0] != "a.go" {
		t.Fatalf("audit entry fields were lost: %+v", entry)
	}
	if entry.GetMetadata().AsMap()["reason"] != "hook" || response.Msg.GetTimestamp() != stamp.Format(time.RFC3339Nano) {
		t.Fatalf("metadata or timestamp was lost: %+v", response.Msg)
	}
}

func TestAuditConnectListEntriesDefaultsLimitAndPreservesOwnerErrors(t *testing.T) {
	logger := &auditQueryCapture{result: &AuditQueryResponse{Entries: []AuditEntry{}, Timestamp: time.Now().UTC()}, err: errors.New("read failed")}
	server := auditConnectServer{server: &Server{audit: logger}}
	_, err := server.ListEntries(context.Background(), connect.NewRequest(&auditv1.ListAuditEntriesRequest{}))
	if connect.CodeOf(err) != connect.CodeInternal || logger.request.Limit != 50 {
		t.Fatalf("query defaults or owner failure changed: limit=%d err=%v", logger.request.Limit, err)
	}
}
