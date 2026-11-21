package protoconv

import (
	"testing"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

func TestTaskCurrentRunAndCompletedStatusRoundTrip(t *testing.T) {
	runID := uuid.New()
	task := &domain.Task{ID: uuid.New(), Status: domain.TaskStatusCompleted, CurrentRunID: &runID}
	protoTask := TaskToProto(task)
	if protoTask.CurrentRunId != runID.String() || protoTask.Status != TaskStatusToProto(domain.TaskStatusCompleted) {
		t.Fatalf("task proto lost lifecycle projection: %+v", protoTask)
	}
	back := TaskFromProto(protoTask)
	if back.CurrentRunID == nil || *back.CurrentRunID != runID || back.Status != domain.TaskStatusCompleted {
		t.Fatalf("task round trip = %+v", back)
	}
}
