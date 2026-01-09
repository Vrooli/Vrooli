package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const executionSnapshotFileName = "execution.proto.json"

func (s *WorkflowService) executionDataRootOrDefault() string {
	if s == nil {
		return "/tmp/bas-executions"
	}
	root := strings.TrimSpace(s.executionDataRoot)
	if root == "" {
		return "/tmp/bas-executions"
	}
	return root
}

func (s *WorkflowService) executionSnapshotPath(executionID uuid.UUID) string {
	return filepath.Join(s.executionDataRootOrDefault(), executionID.String(), executionSnapshotFileName)
}

func (s *WorkflowService) readExecutionSnapshot(ctx context.Context, executionID uuid.UUID) (*basexecution.Execution, error) {
	_ = ctx
	path := s.executionSnapshotPath(executionID)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, database.ErrNotFound
		}
		return nil, fmt.Errorf("read execution snapshot: %w", err)
	}
	var pb basexecution.Execution
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, &pb); err != nil {
		return nil, fmt.Errorf("parse execution snapshot: %w", err)
	}
	return &pb, nil
}

func (s *WorkflowService) writeExecutionSnapshot(ctx context.Context, execIndex *database.ExecutionIndex, pb *basexecution.Execution) error {
	if s == nil {
		return errors.New("workflow service is nil")
	}
	if execIndex == nil || execIndex.ID == uuid.Nil {
		return errors.New("execution index is nil")
	}
	if pb == nil {
		return errors.New("execution proto is nil")
	}
	_ = ctx

	now := time.Now().UTC()
	if strings.TrimSpace(pb.ExecutionId) == "" {
		pb.ExecutionId = execIndex.ID.String()
	}
	if strings.TrimSpace(pb.WorkflowId) == "" && execIndex.WorkflowID != uuid.Nil {
		pb.WorkflowId = execIndex.WorkflowID.String()
	}
	if pb.StartedAt == nil && !execIndex.StartedAt.IsZero() {
		pb.StartedAt = autocontracts.TimeToTimestamp(execIndex.StartedAt)
	}
	if pb.CreatedAt == nil && !execIndex.CreatedAt.IsZero() {
		pb.CreatedAt = autocontracts.TimeToTimestamp(execIndex.CreatedAt)
	}
	pb.UpdatedAt = autocontracts.TimeToTimestamp(now)

	path := s.executionSnapshotPath(execIndex.ID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("ensure execution snapshot dir: %w", err)
	}

	raw, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: false}.Marshal(pb)
	if err != nil {
		return fmt.Errorf("marshal execution snapshot: %w", err)
	}
	indented := raw
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err == nil {
		indented = buf.Bytes()
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, indented, 0o644); err != nil {
		return fmt.Errorf("write execution snapshot tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("finalize execution snapshot: %w", err)
	}
	return nil
}

// HydrateExecutionProto assembles the canonical Execution proto from the DB index and optional on-disk proto snapshot.
// The DB index remains authoritative for queryable lifecycle fields (status/timestamps/error/result_path).
func (s *WorkflowService) HydrateExecutionProto(ctx context.Context, execIndex *database.ExecutionIndex) (*basexecution.Execution, error) {
	if execIndex == nil {
		return nil, errors.New("execution index is nil")
	}

	base := &basexecution.Execution{
		ExecutionId: execIndex.ID.String(),
		WorkflowId:  execIndex.WorkflowID.String(),
		Status:      enums.StringToExecutionStatus(execIndex.Status),
		StartedAt:   autocontracts.TimeToTimestamp(execIndex.StartedAt),
		CreatedAt:   autocontracts.TimeToTimestamp(execIndex.CreatedAt),
		UpdatedAt:   autocontracts.TimeToTimestamp(execIndex.UpdatedAt),
	}
	if execIndex.CompletedAt != nil {
		base.CompletedAt = autocontracts.TimePtrToTimestamp(execIndex.CompletedAt)
	}
	if strings.TrimSpace(execIndex.ErrorMessage) != "" {
		msg := execIndex.ErrorMessage
		base.Error = &msg
	}

	snapshot, err := s.readExecutionSnapshot(ctx, execIndex.ID)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return nil, err
	}
	if snapshot == nil {
		return base, nil
	}

	merged := proto.Clone(snapshot).(*basexecution.Execution)
	merged.ExecutionId = base.ExecutionId
	merged.WorkflowId = base.WorkflowId
	merged.Status = base.Status
	merged.StartedAt = base.StartedAt
	merged.CompletedAt = base.CompletedAt
	merged.CreatedAt = base.CreatedAt
	merged.UpdatedAt = base.UpdatedAt
	merged.Error = base.Error

	return merged, nil
}

