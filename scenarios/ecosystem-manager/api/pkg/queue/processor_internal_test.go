package queue

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestReserveExecutionCreatesEntry(t *testing.T) {
	qp := &Processor{executions: make(map[string]*taskExecution)}

	startedAt := time.Unix(1_700_000_000, 0)
	qp.reserveExecution("task-1", "ecosystem-task-1", startedAt)

	qp.executionsMu.RLock()
	execState, ok := qp.executions["task-1"]
	qp.executionsMu.RUnlock()

	if !ok {
		t.Fatalf("expected reservation to exist")
	}
	if execState.agentTag != "ecosystem-task-1" {
		t.Fatalf("unexpected agent tag: %s", execState.agentTag)
	}
	if !execState.started.Equal(startedAt) {
		t.Fatalf("expected start time %s, got %s", startedAt, execState.started)
	}
	if execState.cmd != nil {
		t.Fatalf("expected cmd to be nil before registration")
	}
}

func TestRegisterExecutionUpgradesReservation(t *testing.T) {
	qp := &Processor{executions: make(map[string]*taskExecution)}

	startedAt := time.Unix(1_700_000_000, 0)
	qp.reserveExecution("task-2", "ecosystem-task-2", startedAt)

	cmd := &exec.Cmd{}
	cmd.Process = &os.Process{Pid: 4242}

	qp.registerExecution("task-2", "ecosystem-task-2", cmd, startedAt.Add(5*time.Minute))

	qp.executionsMu.RLock()
	execState, ok := qp.executions["task-2"]
	qp.executionsMu.RUnlock()

	if !ok {
		t.Fatalf("expected execution to exist")
	}
	if execState.cmd != cmd {
		t.Fatalf("expected cmd pointer to be stored")
	}
	if execState.pid() != 4242 {
		t.Fatalf("expected pid 4242, got %d", execState.pid())
	}
	if !execState.started.Equal(startedAt) {
		t.Fatalf("expected original start time to be preserved")
	}
}

func TestUnregisterExecutionRemovesEntry(t *testing.T) {
	qp := &Processor{executions: make(map[string]*taskExecution)}
	qp.reserveExecution("task-3", "ecosystem-task-3", time.Now())

	qp.unregisterExecution("task-3")

	qp.executionsMu.RLock()
	_, ok := qp.executions["task-3"]
	qp.executionsMu.RUnlock()

	if ok {
		t.Fatalf("expected execution to be removed")
	}
}
