package workflow

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/storage"
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

// createExecution commits immutable recovery inputs before admitting a runner.
// Only the DB index changes with lifecycle; hydration joins it with this receipt.
func (s *WorkflowService) createExecution(ctx context.Context, execIndex *database.ExecutionIndex, pb *basexecution.Execution) error {
	if s == nil {
		return errors.New("workflow service is nil")
	}
	if execIndex == nil || execIndex.ID == uuid.Nil {
		return errors.New("execution index is nil")
	}
	if pb == nil {
		return errors.New("execution proto is nil")
	}
	pb.ExecutionId = execIndex.ID.String()
	pb.WorkflowId = execIndex.WorkflowID.String()
	pb.Status = enums.StringToExecutionStatus(execIndex.Status)
	pb.StartedAt = autocontracts.TimeToTimestamp(execIndex.StartedAt)
	pb.CreatedAt = autocontracts.TimeToTimestamp(execIndex.CreatedAt)
	pb.UpdatedAt = autocontracts.TimeToTimestamp(execIndex.UpdatedAt)

	path := s.executionSnapshotPath(execIndex.ID)
	raw, err := protojson.MarshalOptions{UseProtoNames: true, Indent: "  "}.Marshal(pb)
	if err != nil {
		return fmt.Errorf("marshal execution snapshot: %w", err)
	}
	if err := storage.WriteFileAtomic(path, raw, storage.SecretFilePerm); err != nil {
		return fmt.Errorf("commit execution %s admission metadata: %w", execIndex.ID, err)
	}
	if err := s.repo.CreateExecution(ctx, execIndex); err != nil {
		// A database error may have an uncertain commit outcome. Keep the sole
		// recovery receipt, report its ID, and never start the runner on error.
		return fmt.Errorf("index execution %s after metadata commit: %w", execIndex.ID, err)
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
	if execIndex.ResumedFromID != nil {
		id := execIndex.ResumedFromID.String()
		base.ResumedFrom = &id
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
	merged.ResumedFrom = base.ResumedFrom

	return merged, nil
}
